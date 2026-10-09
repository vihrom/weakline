package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetCacheFilePath(t *testing.T) {
	pwd1 := "/users/dev/project1"
	pwd2 := "/users/dev/project2"

	path1 := getCacheFilePath(pwd1)
	path2 := getCacheFilePath(pwd2)

	if path1 == path2 {
		t.Errorf("expected different cache paths for different working directories, got identical: %q", path1)
	}

	if !strings.Contains(filepath.Base(path1), "cache_") {
		t.Errorf("expected cache filename to start with 'cache_', got %q", filepath.Base(path1))
	}
}

func TestWriteAsyncCacheAndAntiFlicker(t *testing.T) {
	cfg := DefaultConfig
	pwd, _ := os.Getwd()
	cachePath := getCacheFilePath(pwd)

	// Clean up temp test cache file after run
	defer os.Remove(cachePath)

	stValid := Status{IsGit: true, Branch: "main", Staged: 1}
	updated := WriteAsyncCache(cfg, stValid)
	if !updated {
		t.Errorf("expected WriteAsyncCache to return true on initial write")
	}

	// Verify idempotency (no disk rewrite if output identical)
	updatedSecond := WriteAsyncCache(cfg, stValid)
	if updatedSecond {
		t.Errorf("expected WriteAsyncCache to return false when content hasn't changed")
	}

	// Verify Anti-Flicker Protection: Timeout error must not overwrite valid existing cache
	stTimeout := Status{
		IsGit:     true,
		IsTimeout: true,
		Branch:    "Timeout expired! Run git status manually",
	}
	updatedTimeout := WriteAsyncCache(cfg, stTimeout)
	if updatedTimeout {
		t.Errorf("expected WriteAsyncCache to block timeout error from overwriting existing valid cache")
	}
}
