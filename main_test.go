package main

import (
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedPID  int
		expectedExit int
	}{
		{
			name:         "No args",
			args:         []string{},
			expectedPID:  0,
			expectedExit: 0,
		},
		{
			name:         "PID only",
			args:         []string{"1234"},
			expectedPID:  1234,
			expectedExit: 0,
		},
		{
			name:         "PID and Exit Code",
			args:         []string{"1234", "1"},
			expectedPID:  1234,
			expectedExit: 1,
		},
		{
			name:         "Non-numeric args fallback",
			args:         []string{"invalid", "code"},
			expectedPID:  0,
			expectedExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pid, exitCode := parseArgs(tt.args)

			if pid != tt.expectedPID {
				t.Errorf("expected PID %d, got %d", tt.expectedPID, pid)
			}
			if exitCode != tt.expectedExit {
				t.Errorf("expected exit code %d, got %d", tt.expectedExit, exitCode)
			}
		})
	}
}

func TestGetLockPath(t *testing.T) {
	pid := 9999
	path := getLockPath(pid)

	if !strings.Contains(path, "lock_9999") {
		t.Errorf("expected lock path to contain 'lock_9999', got %q", path)
	}
}
