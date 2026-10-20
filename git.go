//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	// waitDelay bounds how long we wait for git's pipes after a timeout kill,
	// so a grandchild (hook, filter) holding stdout cannot wedge the worker.
	waitDelay = 100 * time.Millisecond

	shortOIDLength = 7

	headerOID    = "# branch.oid "
	headerHead   = "# branch.head "
	headerAhead  = "# branch.ab "
	detachedHead = "(detached)"
)

type Status struct {
	Branch    string
	Detached  bool
	Staged    int
	Unstaged  int
	Untracked int
	Ahead     int
	Behind    int
	IsGit     bool
	IsTimeout bool
	Err       error
}

func ceilingDirs() []string {
	var dirs []string
	for _, entry := range strings.Split(os.Getenv("GIT_CEILING_DIRECTORIES"), ":") {
		if filepath.IsAbs(entry) {
			dirs = append(dirs, filepath.Clean(entry))
		}
	}
	return dirs
}

func findGitRoot(dir string) string {
	if os.Getenv("GIT_DIR") != "" {
		return dir
	}

	ceilings := ceilingDirs()
	for {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		if slices.Contains(ceilings, parent) {
			return ""
		}
		dir = parent
	}
}

func parseAheadBehind(value []byte, st *Status) {
	for _, field := range bytes.Fields(value) {
		switch field[0] {
		case '+':
			st.Ahead, _ = strconv.Atoi(string(field[1:]))
		case '-':
			st.Behind, _ = strconv.Atoi(string(field[1:]))
		}
	}
}

func parseHeader(line []byte, st *Status, oid *string) {
	switch {
	case bytes.HasPrefix(line, []byte(headerOID)):
		*oid = string(line[len(headerOID):])
	case bytes.HasPrefix(line, []byte(headerHead)):
		head := string(line[len(headerHead):])
		if head == detachedHead {
			st.Detached = true
		} else {
			st.Branch = head
		}
	case bytes.HasPrefix(line, []byte(headerAhead)):
		parseAheadBehind(line[len(headerAhead):], st)
	}
}

func parseChangedEntry(line []byte, st *Status) {
	if len(line) < 4 {
		return
	}
	if line[2] != '.' {
		st.Staged++
	}
	if line[3] != '.' {
		st.Unstaged++
	}
}

func parseStatusV2(out []byte) Status {
	st := Status{IsGit: true}
	oid := ""

	rem := out
	for len(rem) > 0 {
		var line []byte
		if idx := bytes.IndexByte(rem, '\n'); idx >= 0 {
			line, rem = rem[:idx], rem[idx+1:]
		} else {
			line, rem = rem, nil
		}
		if len(line) == 0 {
			continue
		}

		switch line[0] {
		case '#':
			parseHeader(line, &st, &oid)
		case '1', '2':
			parseChangedEntry(line, &st)
		case 'u': // unmerged (conflict)
			st.Unstaged++
		case '?':
			st.Untracked++
		}
	}

	if st.Detached {
		st.Branch = oid
		if len(st.Branch) > shortOIDLength {
			st.Branch = st.Branch[:shortOIDLength]
		}
	}
	if st.Branch == "" {
		st.Branch = "HEAD"
	}
	return st
}

func gitCommand(ctx context.Context, cfg Config, dir string, args ...string) *exec.Cmd {
	full := []string{"--no-optional-locks"}
	if !cfg.AllowFSMonitor {
		full = append(full, "-c", "core.fsmonitor=false")
	}
	full = append(full, args...)

	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Dir = dir
	cmd.WaitDelay = waitDelay
	return cmd
}

func quickBranch(cfg Config, dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	out, err := gitCommand(ctx, cfg, dir, "symbolic-ref", "--short", "-q", "HEAD").Output()
	if err != nil {
		return ""
	}
	return string(bytes.TrimSpace(out))
}

func GetStatus(cfg Config, dir string) Status {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	out, err := gitCommand(ctx, cfg, dir, "status", "--porcelain=v2", "--branch", "-unormal").Output()
	if err == nil {
		return parseStatusV2(out)
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		st := Status{IsGit: true, IsTimeout: true, Branch: quickBranch(cfg, dir)}
		if st.Branch == "" {
			st.Branch = "HEAD"
		}
		return st
	}

	// Not a repository, git missing, "dubious ownership", corrupted repo, ...
	return Status{Err: err}
}
