// Package git provides utilities for fetching current repository status and commit counters.
package git

import (
	"os/exec"
	"strconv"
	"strings"
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

// getBranch returns the current branch name or HEAD short commit hash if detached.
func getBranch() (string, bool) {
	outBranch, err := exec.Command("git", "symbolic-ref", "--short", "HEAD").Output()
	if err == nil {
		return strings.TrimSpace(string(outBranch)), true
	}

	outHead, errHead := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if errHead == nil {
		return strings.TrimSpace(string(outHead)), true
	}

	return "", false
}

// parseStatusLines parses git status --porcelain output into staged, unstaged, and untracked counts.
func parseStatusLines(st *Status) {
	out, err := exec.Command("git", "status", "--porcelain").Output()
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

// parseAheadBehind parses HEAD...@{u} counts into ahead and behind values.
func parseAheadBehind(st *Status) {
	out, err := exec.Command("git", "rev-list", "--left-right", "--count", "HEAD...@{u}").Output()
	if err != nil {
		return
	}

	parts := strings.Fields(string(out))
	if len(parts) == 2 {
		st.Ahead, _ = strconv.Atoi(parts[0])
		st.Behind, _ = strconv.Atoi(parts[1])
	}
}

// GetStatus executes git commands synchronously to collect status data.
func GetStatus() Status {
	branch, ok := getBranch()
	if !ok {
		return Status{IsGit: false}
	}

	st := Status{
		Branch: branch,
		IsGit:  true,
	}

	parseStatusLines(&st)
	parseAheadBehind(&st)

	return st
}
