package main

import (
	"strings"
	"testing"
)

func TestShortenHome(t *testing.T) {
	t.Setenv("HOME", "/home/developer")

	tests := []struct {
		input    string
		expected string
	}{
		{"/home/developer/code/weakline", "~/code/weakline"},
		{"/home/developer", "~"},
		{"/usr/local/bin", "/usr/local/bin"},
	}

	for _, tt := range tests {
		got := shortenHome(tt.input)
		if got != tt.expected {
			t.Errorf("shortenHome(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestWriteFormattedPath(t *testing.T) {
	cfg := DefaultConfig
	var sb strings.Builder

	writeFormattedPath(&sb, cfg, "~/projects/weakline")
	res := sb.String()

	if !strings.Contains(res, "~/projects") || !strings.Contains(res, "/weakline") {
		t.Errorf("expected output to contain path segments, got %q", res)
	}
}

func TestBuildGitStatusString(t *testing.T) {
	cfg := DefaultConfig
	st := Status{
		IsGit:     true,
		Branch:    "main",
		Ahead:     1,
		Behind:    2,
		Staged:    3,
		Unstaged:  4,
		Untracked: 5,
	}

	res := buildGitStatusString(cfg, st)

	if !strings.Contains(res, "main") {
		t.Errorf("expected output to contain branch 'main', got %q", res)
	}
	if !strings.Contains(res, cfg.IconAhead) {
		t.Errorf("expected output to contain ahead icon, got %q", res)
	}
}

func TestBuildGitStatusStringNonGit(t *testing.T) {
	cfg := DefaultConfig
	st := Status{IsGit: false}

	res := buildGitStatusString(cfg, st)
	if res != "" {
		t.Errorf("expected empty string for non-git status, got %q", res)
	}
}

func BenchmarkRender(b *testing.B) {
	cfg := DefaultConfig
	exitCode := 0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Render(cfg, exitCode)
	}
}
