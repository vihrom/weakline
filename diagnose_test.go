//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"strings"
	"testing"
)

func TestDescribeStatus(t *testing.T) {
	tests := []struct {
		name string
		st   Status
		want string
	}{
		{"timeout", Status{IsGit: true, IsTimeout: true}, "TIMED OUT"},
		{"failure", Status{Err: errTest}, "git failed: test error"},
		{
			"ok",
			Status{IsGit: true, Branch: "main", Staged: 1, Unstaged: 2, Untracked: 3, Ahead: 4, Behind: 5},
			"ok  branch=main staged=1 unstaged=2 untracked=3 ahead=4 behind=5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := describeStatus(tt.st); !strings.Contains(got, tt.want) {
				t.Errorf("describeStatus() = %q, want it to contain %q", got, tt.want)
			}
		})
	}
}

func TestHandleDiagnoseIgnoresOtherArgs(t *testing.T) {
	for _, args := range [][]string{nil, {"1"}, {"--async", "1"}, {"--diagnose", "extra"}} {
		if handleDiagnose(DefaultConfig, args) {
			t.Errorf("handleDiagnose(%v) = true, want false", args)
		}
	}
}
