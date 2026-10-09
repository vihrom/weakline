package main

import (
	"hash/fnv"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

var (
	mkdirOnce  sync.Once
	userTmpDir string
)

// getCacheFilePath returns a secure, unique temporary file path derived from the current directory hash.
func getCacheFilePath(pwd string) string {
	mkdirOnce.Do(func() {
		uid := os.Getuid()
		userTmpDir = filepath.Join(os.TempDir(), "weakline-"+strconv.Itoa(uid))
		_ = os.MkdirAll(userTmpDir, 0700)
	})

	h := fnv.New64a()
	_, _ = h.Write([]byte(pwd))
	hashStr := strconv.FormatUint(h.Sum64(), 36)

	return filepath.Join(userTmpDir, "cache_"+hashStr)
}


// WriteAsyncCache evaluates and saves formatted Git status to a temp file.
func WriteAsyncCache(cfg Config, st Status) bool {
	if st.IsTimeout || st.Err != nil || !st.IsGit {
		return false
	}

	pwd, _ := os.Getwd()
	cachePath := getCacheFilePath(pwd)
	newResult := buildGitStatusString(cfg, st)

	oldData, _ := os.ReadFile(cachePath)

	if string(oldData) == newResult {
		return false
	}

	_ = os.WriteFile(cachePath, []byte(newResult), 0o600)
	return true
}
