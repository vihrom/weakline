//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"hash/fnv"
	"os"
	"path/filepath"
	"strconv"
)

func cacheFilePath(root string) (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}

	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(root))
	return filepath.Join(dir, "cache_"+strconv.FormatUint(hasher.Sum64(), 36)), nil
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
