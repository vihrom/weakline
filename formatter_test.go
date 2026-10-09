package main

import (
	"strings"
	"testing"
)

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

	// Basic sanity checks to ensure icons and branch are present in the output
	if !strings.Contains(res, "main") {
		t.Errorf("expected output to contain branch 'main', got %q", res)
	}
	if !strings.Contains(res, cfg.IconAhead) {
		t.Errorf("expected output to contain ahead icon, got %q", res)
	}
}

// BenchmarkRender measures the performance and allocations of the complete prompt rendering.
func BenchmarkRender(b *testing.B) {
	cfg := DefaultConfig
	exitCode := 0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// We call Render to ensure our strings.Builder and writeColored optimizations work end-to-end
		_ = Render(cfg, exitCode)
	}
}
