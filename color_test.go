package main

import "testing"

func TestColorMap(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Named basic color", "green", "32"},
		{"Named bold color", "bold_red", "31;1"},
		{"HEX color with #", "#babdbf", "38;2;186;189;191"},
		{"HEX color without #", "FF0000", "38;2;255;0;0"},
		{"256 color index", "196", "38;5;196"},
		{"Trim whitespace", "  cyan  ", "36"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Map(tt.input)
			if got != tt.expected {
				t.Errorf("Map(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}
