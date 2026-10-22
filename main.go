//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"
)

const maxRefreshPasses = 3

func parseArgs(args []string) int {
	if len(args) == 0 {
		return 0
	}
	exitCode, _ := strconv.Atoi(args[len(args)-1])
	return exitCode
}

func refresh(cfg Config, root string) bool {
	return WriteAsyncCache(cfg, root, GetStatus(cfg, root))
}

func handleAsync(cfg Config, args []string) bool {
	if len(args) < 2 || args[0] != "--async" {
		return false
	}

	targetPID, err := strconv.Atoi(args[1])
	if err != nil || targetPID <= 0 {
		return true
	}

	pwd, err := os.Getwd()
	if err != nil {
		return true
	}
	root := findGitRoot(pwd)
	if root == "" {
		return true
	}

	dir, err := stateDir()
	if err != nil {
		return true
	}
	lock := acquireLock(dir, targetPID, root)
	if lock == nil {
		return true
	}
	defer lock.Close()

	updated := false
	for pass := 0; pass < maxRefreshPasses; pass++ {
		if refresh(cfg, root) {
			updated = true
		}
		next := takePending(dir, targetPID)
		if next == "" {
			break
		}
		root = next
	}
	maybeCleanup(dir, time.Now())

	if updated && os.Getppid() == targetPID {
		_ = syscall.Kill(targetPID, syscall.SIGUSR1)
	}

	return true
}

func main() {
	args := os.Args[1:]

	if handleInit(args) {
		return
	}

	cfg := DefaultConfig

	if handleDiagnose(cfg, args) {
		return
	}

	if handleAsync(cfg, args) {
		return
	}

	fmt.Print(Render(cfg, parseArgs(args)))
}
