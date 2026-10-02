package main

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"time"
)

// Status holds details about the current git repository status.
type Status struct {
	Branch    string
	Staged    int
	Unstaged  int
	Untracked int
	Ahead     int
	Behind    int
	IsGit     bool
}

// parseAheadBehind parses the ahead/behind token, e.g., "ahead 1, behind 2]" or "ahead 1]".
func parseAheadBehind(token []byte, st *Status) {
	// Trim the trailing bracket if present
	if len(token) > 0 && token[len(token)-1] == ']' {
		token = token[:len(token)-1]
	}

	// Split multiple metrics by comma (e.g., "ahead 1" and "behind 2")
	for _, part := range bytes.Split(token, []byte(", ")) {
		part = bytes.TrimSpace(part)
		if bytes.HasPrefix(part, []byte("ahead ")) {
			val, _ := strconv.Atoi(string(part[6:]))
			st.Ahead = val
		} else if bytes.HasPrefix(part, []byte("behind ")) {
			val, _ := strconv.Atoi(string(part[7:]))
			st.Behind = val
		}
	}
}

// parseBranchLine extracts the branch name and ahead/behind counters from the header line.
// Example formats:
// "## main...origin/main [ahead 1, behind 2]"
// "## initial...origin/initial [behind 4]"
// "## master"
func parseBranchLine(line []byte, st *Status) {
	if len(line) < 3 {
		return
	}
	// Skip the leading "## "
	line = line[3:]

	// Check if there are ahead/behind tracking metrics
	idxBracket := bytes.IndexByte(line, '[')
	if idxBracket != -1 {
		trackingInfo := line[idxBracket+1:]
		parseAheadBehind(trackingInfo, st)
		line = line[:idxBracket]
	}

	// Clean up trailing spaces or ellipsis if remote exists
	line = bytes.TrimSpace(line)
	if idxEllipsis := bytes.Index(line, []byte("...")); idxEllipsis != -1 {
		line = line[:idxEllipsis]
	}

	st.Branch = string(line)
}

// GetStatus orchestrates repository metadata parsing bound by a global configuration timeout.
// CRITICAL OPTIMIZATION: Executes a single 'git status' process to retrieve both file status
// and tracking distance simultaneously, cutting execution overhead strictly in half.
func GetStatus(timeout time.Duration) Status {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Use --branch to fetch tracking distance and current branch in a single process fork.
	// Use --no-optional-locks to prevent lock file collisions in the background.
	cmd := exec.CommandContext(ctx, "git", "--no-optional-locks", "status", "--porcelain=v1", "--branch", "-unormal")
	out, err := cmd.Output()
	if err != nil {
		return Status{IsGit: false}
	}

	st := Status{IsGit: true}

	// Process the raw byte output line by line using bytes.IndexByte.
	// This approach is completely zero-alloc for loop steps, unlike strings.Split sequences.
	rem := out
	isFirstLine := true

	for len(rem) > 0 {
		var line []byte
		idx := bytes.IndexByte(rem, '\n')
		if idx >= 0 {
			line = rem[:idx]
			rem = rem[idx+1:]
		} else {
			line = rem
			rem = nil
		}

		if len(line) == 0 {
			continue
		}

		// The very first line containing "##" holds the branch name and sync statistics
		if isFirstLine {
			isFirstLine = false
			if bytes.HasPrefix(line, []byte("##")) {
				parseBranchLine(line, &st)
				continue
			}
		}

		if len(line) < 2 {
			continue
		}

		x, y := line[0], line[1]
		if x == '?' && y == '?' {
			st.Untracked++
			continue
		}
		if x != ' ' && x != '?' {
			st.Staged++
		}
		if y != ' ' && y != '?' {
			st.Unstaged++
		}
	}

	// Handle edge case where git status was completely empty (e.g. detached HEAD without explicit branch line)
	if st.Branch == "" {
		st.Branch = "HEAD"
	}

	return st
}
