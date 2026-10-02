package main

import (
	"os"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	// Backup original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name         string
		args         []string
		expectedPID  int
		expectedExit int
	}{
		{
			name:         "No args",
			args:         []string{"weakline"},
			expectedPID:  0,
			expectedExit: 0,
		},
		{
			name:         "PID only",
			args:         []string{"weakline", "1234"},
			expectedPID:  1234,
			expectedExit: 0,
		},
		{
			name:         "PID and Exit Code",
			args:         []string{"weakline", "1234", "1"},
			expectedPID:  1234,
			expectedExit: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			pid, exitCode := parseArgs()

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
