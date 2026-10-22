//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	stateDirPerm    = 0o700
	staleAfter      = 7 * 24 * time.Hour
	cleanupInterval = 24 * time.Hour
	cleanupMarker   = "cleanup"
)

func stateDir() (string, error) {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if !filepath.IsAbs(base) {
		base = os.TempDir()
	}

	dir := filepath.Join(base, "weakline-"+strconv.Itoa(os.Getuid()))
	if err := ensurePrivateDir(dir); err != nil {
		return "", err
	}
	return dir, nil
}

func ensurePrivateDir(dir string) error {
	info, err := os.Lstat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		if err = os.Mkdir(dir, stateDirPerm); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
		info, err = os.Lstat(dir)
	}
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return errors.New("state path is not a real directory")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Getuid() {
		return errors.New("state directory is not owned by the current user")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return os.Chmod(dir, stateDirPerm)
	}
	return nil
}

func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "tmp_")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err = tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err = os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

func lockPath(dir string, pid int) string {
	return filepath.Join(dir, "lock_"+strconv.Itoa(pid))
}

func pendingPath(dir string, pid int) string {
	return filepath.Join(dir, "pending_"+strconv.Itoa(pid))
}

func tryFlock(file *os.File) bool {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) == nil
}

func acquireLock(dir string, pid int, root string) *os.File {
	file, err := os.OpenFile(lockPath(dir, pid), os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil
	}
	if tryFlock(file) {
		return file
	}

	markPending(dir, pid, root)
	if tryFlock(file) { // the holder may have just finished
		_ = os.Remove(pendingPath(dir, pid))
		return file
	}
	_ = file.Close()
	return nil
}

func markPending(dir string, pid int, root string) {
	_ = os.WriteFile(pendingPath(dir, pid), []byte(root), 0o600)
}

func takePending(dir string, pid int) string {
	path := pendingPath(dir, pid)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	_ = os.Remove(path)
	return strings.TrimSpace(string(data))
}

func maybeCleanup(dir string, now time.Time) {
	marker := filepath.Join(dir, cleanupMarker)
	if info, err := os.Stat(marker); err == nil && now.Sub(info.ModTime()) < cleanupInterval {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "cache_") && !strings.HasPrefix(name, "lock_") &&
			!strings.HasPrefix(name, "pending_") && !strings.HasPrefix(name, "tmp_") {
			continue
		}
		info, err := entry.Info()
		if err == nil && now.Sub(info.ModTime()) > staleAfter {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
	_ = os.WriteFile(marker, []byte(strconv.FormatInt(now.Unix(), 10)), 0o600)
}
