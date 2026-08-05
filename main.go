package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"weakline/config"
	"weakline/git"
	"weakline/prompt"
)

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

// handleAsync runs in the background to calculate Git status and signal Zsh on update.
func handleAsync(cfg config.Config) bool {
	if len(os.Args) <= 2 || os.Args[1] != "--async" {
		return false
	}

	targetPID, err := strconv.Atoi(os.Args[2])
	if err != nil {
		return true
	}

	gitStatus := git.GetStatus()
	updated := prompt.WriteAsyncCache(cfg, gitStatus)

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
func spawnAsyncProcess(pid int) {
	exe, err := os.Executable()
	if err != nil {
		return
	}

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

	fmt.Print(prompt.Render(cfg, exitCode))
	spawnAsyncProcess(pid)
}
