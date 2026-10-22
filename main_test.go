//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"os"
	"testing"
	"time"
)

// unsetEnv removes an environment variable for the duration of the test.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	old, had := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, old)
		}
	})
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int
	}{
		{"no args", nil, 0},
		{"exit code only", []string{"1"}, 1},
		{"legacy pid and exit code", []string{"1234", "127"}, 127},
		{"legacy pid and zero", []string{"1234", "0"}, 0},
		{"garbage", []string{"abc"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseArgs(tt.args); got != tt.want {
				t.Errorf("parseArgs(%v) = %d, want %d", tt.args, got, tt.want)
			}
		})
	}
}

func TestHandleAsyncIgnoresOtherArgs(t *testing.T) {
	for _, args := range [][]string{nil, {"1"}, {"init", "zsh"}, {"--async"}} {
		if handleAsync(DefaultConfig, args) {
			t.Errorf("handleAsync(%v) = true, want false", args)
		}
	}
}

func TestHandleAsyncRejectsBadPID(t *testing.T) {
	// Handled (true) but must do nothing: no lock, no signal, no state.
	for _, pid := range []string{"abc", "0", "-1"} {
		if !handleAsync(DefaultConfig, []string{"--async", pid}) {
			t.Errorf("handleAsync(--async %s) = false, want true", pid)
		}
	}
}

// The worker is asynchronous, so a tight timeout only produces false "slow" markers on
// ordinary machines (cold disk cache, load). Guard against going back to a UI-sized budget.
func TestDefaultTimeoutIsGenerousForABackgroundWorker(t *testing.T) {
	if DefaultConfig.Timeout < 2*time.Second {
		t.Errorf("DefaultConfig.Timeout = %s, want at least 2s", DefaultConfig.Timeout)
	}
}
