package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"path/filepath"

	"weakline/config"
	"weakline/git"
	"weakline/prompt"
)

// zshInitScript injects configuration hooks into the active shell environment.
// PROMPT_SUBST enables dynamic function evaluation inside the prompt string.
// TRAPUSR1 intercepts signals from background tasks to immediately refresh the visual grid.
const zshInitScript = `
setopt PROMPT_SUBST

TRAPUSR1() {
    zle && zle reset-prompt
}

PROMPT='$(weakline $$ $?)'
`

// handleInit prints shell integration code if requested.
func handleInit() bool {
	if len(os.Args) > 2 && os.Args[1] == "init" && os.Args[2] == "zsh" {
		fmt.Print(zshInitScript)
		return true
	}
	return false
}

// getLockPath returns a secure lockfile path isolated inside the user's private directory.
func getLockPath(pid int) string {
	uid := os.Getuid()
	userTmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("weakline-%d", uid))
	// Ensure the private subdirectory exists with strict 0700 permissions
	_ = os.MkdirAll(userTmpDir, 0700)

	return filepath.Join(userTmpDir, fmt.Sprintf("lock_%d", pid))
}

// handleAsync executes as a detached low-priority thread to compute expensive Git status.
// It acquires a kernel-level lock to ensure only one worker scans the directory graph per session.
func handleAsync(cfg config.Config) bool {
	if len(os.Args) <= 2 || os.Args[1] != "--async" {
		return false
	}

	targetPID, err := strconv.Atoi(os.Args[2])
	if err != nil {
		return true
	}

	lockPath := getLockPath(targetPID)
	file, err := os.OpenFile(lockPath, os.O_RDWR, 0600)
	if err == nil {
		defer file.Close()
		// Try to hold the lock during execution. Abort if a previous worker is still processing.
		if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			return true
		}
	}

	gitStatus := git.GetStatus(cfg.Timeout)
	updated := prompt.WriteAsyncCache(cfg, gitStatus)

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
	defer file.Close()

	// Perform a non-blocking test lock to see if a background thread is already active
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return
	}

	exe, err := os.Executable()
	if err != nil {
		return
	}

	// Fork into background detached session via Setsid attribute
	cmd := exec.Command(exe, "--async", strconv.Itoa(pid))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	_ = cmd.Start()
}

func main() {
	if handleInit() {
		return
	}

	cfg := config.Default

	if handleAsync(cfg) {
		return
	}

	pid, exitCode := parseArgs()

	// Render prompt instantly from fast fallback memory/cache
	fmt.Print(prompt.Render(cfg, exitCode))

	// Offload Git lookup computations to a detached child process
	spawnAsyncProcess(pid)
}
