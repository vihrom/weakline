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
	IsTimeout bool
	Err       error
}

// parseAheadBehind parses the ahead/behind token, e.g., "ahead 1, behind 2] (gone)" or "ahead 1]".
func parseAheadBehind(token []byte, st *Status) {
	// Cut trailing metadata at the closing bracket
	if idxClose := bytes.IndexByte(token, ']'); idxClose != -1 {
		token = token[:idxClose]
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
// "## No commits yet on main"
// "## master"
func parseBranchLine(line []byte, st *Status) {
	if !bytes.HasPrefix(line, []byte("## ")) {
		return
	}
	// Skip leading "## "
	line = line[3:]

	// Handle initial repository states before first commit
	if bytes.HasPrefix(line, []byte("No commits yet on ")) {
		st.Branch = string(line[18:])
		return
	}
	if bytes.HasPrefix(line, []byte("Initial commit on ")) {
		st.Branch = string(line[18:])
		return
	}

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
func GetStatus(timeout time.Duration) Status {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "--no-optional-locks", "status", "--porcelain=v1", "--branch", "-unormal")

	out, err := cmd.Output()

	if err != nil {
		// Anti-flicker guard: if Git execution exceeds the timeout deadline,
		// mark IsTimeout = true so WriteAsyncCache can safely abort without
		// wiping the existing cache or triggering false prompt redraws.
		if ctx.Err() == context.DeadlineExceeded {
			return Status{
				IsGit:     true,
				IsTimeout: true,
				Err:       ctx.Err(),
			}
		}
		// Non-git directory or execution failure (e.g., git not installed, bad flags)
		return Status{
			IsGit: false,
			Err:   err,
		}
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

		// Handle merge conflict states explicitly to prevent duplicate count
		if (x == 'D' && y == 'D') || (x == 'A' && y == 'U') || (x == 'U' && y == 'D') ||
			(x == 'U' && y == 'A') || (x == 'D' && y == 'U') || (x == 'A' && y == 'A') || (x == 'U' && y == 'U') {
			st.Unstaged++
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
