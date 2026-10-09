//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// getLockPath returns a lockfile path isolated inside the user's private directory.
func getLockPath(pid int) string {
	uid := os.Getuid()
	userTmpDir := filepath.Join(os.TempDir(), "weakline-"+strconv.Itoa(uid))
	_ = os.MkdirAll(userTmpDir, 0700)
	return filepath.Join(userTmpDir, "lock_"+strconv.Itoa(pid))
}

// handleAsync executes as a detached low-priority thread to compute expensive Git status.
// It acquires a kernel-level lock to ensure only one worker scans the directory graph per session.
func handleAsync(cfg Config, args []string) bool {
	if len(args) < 2 || args[0] != "--async" {
		return false
	}

	targetPID, err := strconv.Atoi(args[1])
	if err != nil {
		return true
	}

	lockPath := getLockPath(targetPID)

	file, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return true
	}
	defer file.Close()

	// Try to hold the lock during execution. Abort if a previous worker is still processing.
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return true
	}

	gitStatus := GetStatus(cfg.Timeout)
	updated := WriteAsyncCache(cfg, gitStatus)

	// If metadata changed, notify Zsh shell using cross-process signaling (SIGUSR1)
	if updated && targetPID > 0 {
		_ = syscall.Kill(targetPID, syscall.SIGUSR1)
	}

	return true
}

// parseArgs extracts target shell PID and last exit code from command line arguments.
func parseArgs(args []string) (int, int) {
	pid := 0
	exitCode := 0

	if len(args) > 0 {
		pid, _ = strconv.Atoi(args[0])
	}
	if len(args) > 1 {
		exitCode, _ = strconv.Atoi(args[1])
	}

	return pid, exitCode
}

// spawnAsyncProcess launches weakline in detached background mode to refresh Git status.
func spawnAsyncProcess(pid int) {
	exe, err := os.Executable()
	if err != nil {
		return
	}

	cmd := exec.Command(exe, "--async", strconv.Itoa(pid))
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	_ = cmd.Start()
}

func main() {
	args := os.Args[1:]

	if handleInit(args) {
		return
	}

	cfg := DefaultConfig

	if handleAsync(cfg, args) {
		return
	}

	pid, exitCode := parseArgs(args)

	// Render prompt instantly from fast fallback memory/cache
	fmt.Print(Render(cfg, exitCode))

	// Offload Git lookup computations to a detached child process
	if pid > 0 {
		spawnAsyncProcess(pid)
	}
}
