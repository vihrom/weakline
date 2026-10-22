//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func useStateDir(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", base)
	return filepath.Join(base, "weakline-"+strconv.Itoa(os.Getuid()))
}

func TestStateDirCreatesPrivateDir(t *testing.T) {
	want := useStateDir(t)

	dir, err := stateDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != want {
		t.Errorf("stateDir() = %q, want %q", dir, want)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("permissions = %o, want 700", perm)
	}
}

func TestStateDirTightensLoosePermissions(t *testing.T) {
	path := useStateDir(t)
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}

	dir, err := stateDir()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("permissions = %o, want 700", perm)
	}
}

func TestStateDirRejectsSymlink(t *testing.T) {
	path := useStateDir(t)
	if err := os.Symlink(t.TempDir(), path); err != nil {
		t.Fatal(err)
	}
	if dir, err := stateDir(); err == nil {
		t.Errorf("stateDir() accepted a symlink, returned %q", dir)
	}
}

func TestStateDirRejectsRegularFile(t *testing.T) {
	path := useStateDir(t)
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if dir, err := stateDir(); err == nil {
		t.Errorf("stateDir() accepted a regular file, returned %q", dir)
	}
}

func TestStateDirIgnoresRelativeRuntimeDir(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "relative/path")
	t.Setenv("TMPDIR", t.TempDir())

	dir, err := stateDir()
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("stateDir() = %q, want an absolute path", dir)
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache_x")

	for _, content := range []string{"first", "second, longer than the first"} {
		if err := writeFileAtomic(path, []byte(content)); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != content {
			t.Errorf("read %q, want %q", got, content)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("found %d files after writing, want 1 (temp file left behind?)", len(entries))
	}
}

func TestLockIsExclusiveAndLeavesPendingNote(t *testing.T) {
	useStateDir(t)
	dir, err := stateDir()
	if err != nil {
		t.Fatal(err)
	}

	first := acquireLock(dir, 4242, "/repo/a")
	if first == nil {
		t.Fatal("first acquireLock returned nil")
	}

	if second := acquireLock(dir, 4242, "/repo/b"); second != nil {
		_ = second.Close()
		t.Fatal("second acquireLock succeeded while the first still holds the lock")
	}
	if got := takePending(dir, 4242); got != "/repo/b" {
		t.Errorf("takePending = %q, want /repo/b", got)
	}
	if got := takePending(dir, 4242); got != "" {
		t.Errorf("pending note was not cleared, got %q", got)
	}

	_ = first.Close()
	third := acquireLock(dir, 4242, "/repo/c")
	if third == nil {
		t.Fatal("acquireLock failed after the lock was released")
	}
	_ = third.Close()
}

func TestLockIsPerShell(t *testing.T) {
	useStateDir(t)
	dir, err := stateDir()
	if err != nil {
		t.Fatal(err)
	}

	one := acquireLock(dir, 1001, "/repo")
	two := acquireLock(dir, 1002, "/repo")
	if one == nil || two == nil {
		t.Fatal("workers of different shells must not block each other")
	}
	_ = one.Close()
	_ = two.Close()
}

func TestMaybeCleanup(t *testing.T) {
	useStateDir(t)
	dir, err := stateDir()
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	old := now.Add(-8 * 24 * time.Hour)
	write := func(name string, mtime time.Time) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, mtime, mtime); err != nil {
			t.Fatal(err)
		}
		return path
	}
	exists := func(path string) bool {
		_, err := os.Stat(path)
		return err == nil
	}

	oldCache := write("cache_old", old)
	oldLock := write("lock_1", old)
	freshCache := write("cache_fresh", now)
	foreign := write("keep_me", old)

	maybeCleanup(dir, now)

	if exists(oldCache) || exists(oldLock) {
		t.Error("stale cache/lock files were not removed")
	}
	if !exists(freshCache) {
		t.Error("fresh cache file was removed")
	}
	if !exists(foreign) {
		t.Error("a file weakline does not own was removed")
	}

	// The marker throttles cleanup to once per interval.
	again := write("cache_old_again", old)
	maybeCleanup(dir, now)
	if !exists(again) {
		t.Error("cleanup ran twice within its interval")
	}
}
