//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"errors"
	"testing"
)

var errTest = errors.New("test error")

func TestMap(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"red", "31"},
		{"bold_cyan", "36;1"},
		{"bg_bright_blue", "104"},
		{"  green ", "32"},
		{"#babdbf", "38;2;186;189;191"},
		{"BABDBF", "38;2;186;189;191"},
		{"123456", "38;2;18;52;86"},
		{"#fff", "38;2;255;255;255"},
		{"#0af", "38;2;0;170;255"},
		{"196", "38;5;196"},
		{"0", "38;5;0"},
		{"255", "38;5;255"},
		{"1;31", "1;31"},
		{"nonsense", ""},
		{"#12345", ""},
		{"#gggggg", ""},
		{"0af", ""},
		{"", ""},
	}

	for _, tt := range tests {
		if got := Map(tt.in); got != tt.want {
			t.Errorf("Map(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// A typo in the default theme would silently produce an empty color, so pin the whole theme.
func TestDefaultColorsAreValid(t *testing.T) {
	colors := DefaultConfig.Colors
	theme := map[string]string{
		"PathParent": colors.PathParent,
		"PathActive": colors.PathActive,
		"Branch":     colors.Branch,
		"Staged":     colors.Staged,
		"Unstaged":   colors.Unstaged,
		"Untracked":  colors.Untracked,
		"Ahead":      colors.Ahead,
		"Behind":     colors.Behind,
		"Timeout":    colors.Timeout,
		"Python":     colors.Python,
		"PromptOK":   colors.PromptOK,
		"PromptErr":  colors.PromptErr,
	}

	for name, value := range theme {
		if !isSGR(value) {
			t.Errorf("DefaultConfig.Colors.%s = %q is not a valid SGR sequence", name, value)
		}
	}
}

func TestIsSGR(t *testing.T) {
	for _, valid := range []string{"0", "31", "38;5;196", "1;31"} {
		if !isSGR(valid) {
			t.Errorf("isSGR(%q) = false, want true", valid)
		}
	}
	for _, invalid := range []string{"", "red", "3 1", "31m", "\x1b[31"} {
		if isSGR(invalid) {
			t.Errorf("isSGR(%q) = true, want false", invalid)
		}
	}
}
