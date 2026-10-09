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

// getLockPath returns a secure lockfile path isolated inside the user's private directory.
// Optimized to utilize strconv.Itoa, eliminating costly fmt.Sprintf allocations.
func getLockPath(pid int) string {
	uid := os.Getuid()

	// Avoid fmt.Sprintf by concatenating pre-converted strings
	userTmpDir := filepath.Join(os.TempDir(), "weakline-"+strconv.Itoa(uid))

	// Ensure the private subdirectory exists with strict 0700 permissions
	_ = os.MkdirAll(userTmpDir, 0700)

	return filepath.Join(userTmpDir, "lock_"+strconv.Itoa(pid))
}

// handleAsync executes as a detached low-priority thread to compute expensive Git status.
// It acquires a kernel-level lock to ensure only one worker scans the directory graph per session.
func handleAsync(cfg Config) bool {
	if len(os.Args) <= 2 || os.Args[1] != "--async" {
		return false
	}

	targetPID, err := strconv.Atoi(os.Args[2])
	if err != nil {
		return true
	}

	lockPath := getLockPath(targetPID)

	// FIX: Added os.O_CREATE flag to prevent failure if the lockfile was deleted from tmp
	file, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0600)
	if err == nil {
		defer file.Close()
		// Try to hold the lock during execution. Abort if a previous worker is still processing.
		if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			return true
		}
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
func parseArgs() (int, int) {
	pid := 0
	exitCode := 0

	if len(os.Args) > 1 {
		pid, _ = strconv.Atoi(os.Args[1])
	}
	if len(os.Args) > 2 {
		exitCode, _ = strconv.Atoi(os.Args[2])
	}

	return pid, exitCode
}

// spawnAsyncProcess launches weakline in detached background mode to refresh Git status.
// Uses advisory kernel locks (flock) to prevent duplicate runs and process spikes.
func spawnAsyncProcess(pid int) {
	lockPath := getLockPath(pid)
	// Open or create the lockfile securely with exclusive owner permissions (0600)
	file, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return
	}

	// Perform a non-blocking test lock to see if a background thread is already active
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		file.Close()
		return
	}

	exe, err := os.Executable()
	if err != nil {
		file.Close()
		return
	}

	// Fork into background detached session via Setsid attribute
	cmd := exec.Command(exe, "--async", strconv.Itoa(pid))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	_ = cmd.Start()

	// FIX: Explicitly close the file handle immediately after process start.
	// This prevents descriptor leakage and ensures the child doesn't lock its own parent context.
	file.Close()
}

func main() {
	if handleInit() {
		return
	}

	cfg := DefaultConfig

	if handleAsync(cfg) {
		return
	}

	pid, exitCode := parseArgs()

	// Render prompt instantly from fast fallback memory/cache
	fmt.Print(Render(cfg, exitCode))

	// Offload Git lookup computations to a detached child process
	spawnAsyncProcess(pid)
}
