//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const diagnoseRuns = 3

func describeStatus(st Status) string {
	switch {
	case st.IsTimeout:
		return "TIMED OUT: git did not finish within the timeout"
	case st.Err != nil:
		msg := st.Err.Error()
		var exitErr *exec.ExitError
		if errors.As(st.Err, &exitErr) && len(exitErr.Stderr) > 0 {
			msg = strings.TrimSpace(string(exitErr.Stderr))
		}
		return "git failed: " + msg
	default:
		return fmt.Sprintf("ok  branch=%s staged=%d unstaged=%d untracked=%d ahead=%d behind=%d",
			st.Branch, st.Staged, st.Unstaged, st.Untracked, st.Ahead, st.Behind)
	}
}

func handleDiagnose(cfg Config, args []string) bool {
	if len(args) != 1 || args[0] != "--diagnose" {
		return false
	}

	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println("cannot determine the current directory:", err)
		return true
	}
	root := findGitRoot(pwd)
	if root == "" {
		fmt.Println("not inside a git repository (no .git found in or above " + pwd + ")")
		return true
	}

	if out, err := exec.Command("git", "--version").Output(); err == nil {
		fmt.Printf("git:        %s\n", strings.TrimSpace(string(out)))
	} else {
		fmt.Println("git:        cannot run git:", err)
	}
	fmt.Printf("repository: %s\n", root)
	fmt.Printf("timeout:    %s (AllowFSMonitor=%t)\n", cfg.Timeout, cfg.AllowFSMonitor)

	for run := 1; run <= diagnoseRuns; run++ {
		start := time.Now()
		st := GetStatus(cfg, root)
		fmt.Printf("run %d: %6d ms  %s\n", run, time.Since(start).Milliseconds(), describeStatus(st))
	}
	return true
}
