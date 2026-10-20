//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"hash/fnv"
	"os"
	"path/filepath"
	"strconv"
)

const maxCacheBytes = 4096

func cacheFilePath(root string) (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}

	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(root))
	return filepath.Join(dir, "cache_"+strconv.FormatUint(hasher.Sum64(), 36)), nil
}

func readCache(pwd string) []byte {
	if pwd == "" {
		return nil
	}
	root := findGitRoot(pwd)
	if root == "" {
		return nil
	}
	path, err := cacheFilePath(root)
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > maxCacheBytes {
		return nil
	}
	return data
}

func WriteAsyncCache(cfg Config, root string, st Status) bool {
	path, err := cacheFilePath(root)
	if err != nil {
		return false
	}

	content := buildGitStatusString(cfg, st)
	if content == "" {
		return os.Remove(path) == nil // true only if there was something to remove
	}

	oldData, _ := os.ReadFile(path)
	if string(oldData) == content {
		return false
	}
	return writeFileAtomic(path, []byte(content)) == nil
}
