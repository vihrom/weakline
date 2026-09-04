// Package git provides utilities for fetching current repository status and commit counters.
package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

// findGitDir recursively traverses upward from the current working directory
// to locate the root .git directory.
func findGitDir() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}

	for {
		gitDir := filepath.Join(dir, ".git")
		_, err := os.Stat(gitDir)
		if err == nil {
			return gitDir, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

// getBranch reads .git/HEAD directly from disk to determine the active branch.
func getBranch(gitDir string) (string, bool) {
	headPath := filepath.Join(gitDir, "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", false
	}

	content := strings.TrimSpace(string(data))
	if strings.HasPrefix(content, "ref: ") {
		ref := content[5:]
		parts := strings.Split(ref, "/")
		return parts[len(parts)-1], true
	}

	if len(content) >= 7 {
		return content[:7], true
	}

	return "", false
}

// parseStatusLines scans files using the '-unormal' flag linked with a timeout context.
// CRITICAL OPTIMIZATION: Bails out safely if the operations exceed the configured timeout.
func parseStatusLines(ctx context.Context, st *Status) {
	out, err := exec.CommandContext(ctx, "git", "--no-optional-locks", "status", "--porcelain", "-unormal").Output()
	if err != nil {
		return
	}

	for line := range strings.SplitSeq(string(out), "\n") {
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
}

// readHashFromPackedRefs searches for packed branch commit hashes inside .git/packed-refs.
func readHashFromPackedRefs(gitDir, refPath string) string {
	packedPath := filepath.Join(gitDir, "packed-refs")
	data, err := os.ReadFile(packedPath)
	if err != nil {
		return ""
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		if len(line) == 0 || line[0] == '#' || line[0] == '^' {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == refPath {
			return fields[0]
		}
	}
	return ""
}

// getLocalAndRemoteHashes retrieves active branch hashes from loose references or packed-refs.
func getLocalAndRemoteHashes(gitDir, branch string) (string, string) {
	localRefPath := "refs/heads/" + branch
	remoteRefPath := "refs/remotes/origin/" + branch

	var localHash string
	localData, err := os.ReadFile(filepath.Join(gitDir, localRefPath))
	if err == nil {
		localHash = strings.TrimSpace(string(localData))
	} else {
		localHash = readHashFromPackedRefs(gitDir, localRefPath)
	}

	var remoteHash string
	remoteData, err := os.ReadFile(filepath.Join(gitDir, remoteRefPath))
	if err == nil {
		remoteHash = strings.TrimSpace(string(remoteData))
	} else {
		remoteHash = readHashFromPackedRefs(gitDir, remoteRefPath)
	}

	if localHash == "" {
		return "1", "2"
	}

	return localHash, remoteHash
}

// parseAheadBehind calculates the branch commit distance against upstream using a timeout context.
func parseAheadBehind(ctx context.Context, gitDir, branch string, st *Status) {
	localHash, remoteHash := getLocalAndRemoteHashes(gitDir, branch)

	if localHash == remoteHash && localHash != "" {
		st.Ahead = 0
		st.Behind = 0
		return
	}

	out, err := exec.CommandContext(ctx, "git", "rev-list", "--left-right", "--count", "HEAD...@{u}").Output()
	if err != nil {
		return
	}

	parts := strings.Fields(string(out))
	if len(parts) == 2 {
		st.Ahead, _ = strconv.Atoi(parts[0])
		st.Behind, _ = strconv.Atoi(parts[1])
	}
}

// GetStatus orchestrates repository metadata parsing bound by a global configuration timeout.
func GetStatus(timeout time.Duration) Status {
	gitDir, ok := findGitDir()
	if !ok {
		return Status{IsGit: false}
	}

	branch, ok := getBranch(gitDir)
	if !ok {
		return Status{IsGit: false}
	}

	st := Status{
		Branch: branch,
		IsGit:  true,
	}

	// Enforce strict global timeout constraints using the configuration value
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	parseStatusLines(ctx, &st)
	parseAheadBehind(ctx, gitDir, branch, &st)

	return st
}
