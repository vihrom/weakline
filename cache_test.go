//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheFilePath(t *testing.T) {
	useStateDir(t)

	a1, err := cacheFilePath("/repo/a")
	if err != nil {
		t.Fatal(err)
	}
	a2, _ := cacheFilePath("/repo/a")
	b, _ := cacheFilePath("/repo/b")

	if a1 != a2 {
		t.Error("cache path is not stable for the same root")
	}
	if a1 == b {
		t.Error("different roots share a cache path")
	}
	if !strings.HasPrefix(filepath.Base(a1), "cache_") {
		t.Errorf("unexpected cache file name %q", filepath.Base(a1))
	}
}

func TestWriteAsyncCacheLifecycle(t *testing.T) {
	useStateDir(t)
	cfg := DefaultConfig
	root := "/repo"
	path, err := cacheFilePath(root)
	if err != nil {
		t.Fatal(err)
	}

	st := Status{IsGit: true, Branch: "main", Staged: 1}
	if !WriteAsyncCache(cfg, root, st) {
		t.Fatal("first write must report a change")
	}
	if WriteAsyncCache(cfg, root, st) {
		t.Fatal("identical content must not report a change (anti-flicker)")
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "main") {
		t.Fatalf("cache content = %q, err = %v", data, err)
	}

	// A timeout must not keep stale counters: it degrades to branch + marker.
	slow := Status{IsGit: true, IsTimeout: true, Branch: "main"}
	if !WriteAsyncCache(cfg, root, slow) {
		t.Fatal("degrading to the timeout marker must report a change")
	}
	data, _ = os.ReadFile(path)
	if !strings.Contains(string(data), cfg.IconTimeout) || strings.Contains(string(data), cfg.IconStaged+"1") {
		t.Errorf("timeout content = %q", data)
	}

	// A failed status (not a repository anymore) removes the entry.
	failed := Status{Err: errors.New("fatal: not a git repository")}
	if !WriteAsyncCache(cfg, root, failed) {
		t.Fatal("removing the cache must report a change")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("cache file still exists: %v", err)
	}
	if WriteAsyncCache(cfg, root, failed) {
		t.Error("removing a missing cache must not report a change")
	}
}

func TestReadCache(t *testing.T) {
	useStateDir(t)
	unsetEnv(t, "GIT_DIR")

	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repo, "pkg", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	if got := readCache(sub); got != nil {
		t.Errorf("readCache before any write = %q, want nil", got)
	}

	WriteAsyncCache(DefaultConfig, repo, Status{IsGit: true, Branch: "feature"})
	for _, dir := range []string{repo, sub} {
		got := readCache(dir)
		if !strings.Contains(string(got), "feature") {
			t.Errorf("readCache(%q) = %q, want the repository's cache", dir, got)
		}
	}

	if got := readCache(t.TempDir()); got != nil {
		t.Errorf("readCache outside a repository = %q, want nil", got)
	}
	if got := readCache(""); got != nil {
		t.Errorf("readCache(\"\") = %q, want nil", got)
	}
}

func TestReadCacheIgnoresOversizedFile(t *testing.T) {
	useStateDir(t)
	unsetEnv(t, "GIT_DIR")

	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := cacheFilePath(repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x", maxCacheBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readCache(repo); got != nil {
		t.Errorf("oversized cache was read (%d bytes)", len(got))
	}
}
