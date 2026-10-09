package main

import "time"

// Colors encapsulates the ANSI code color mapping for every semantic element of the prompt.
// Supported color representation formats via Map():
// 1. Plain text names: "red", "cyan", "gray", etc.
// 2. Bold/Styled text names: "bold_red", "bold_cyan", "underline", "dim", etc.
// 3. HEX / TrueColor (24-bit): "#babdbf", "#FF0055", "RRGGBB"
// 4. ANSI 256-color palette index: "196", "208", "39"
type Colors struct {
	PathParent string
	PathActive string
	Branch     string
	Staged     string
	Unstaged   string
	Untracked  string
	Ahead      string
	Behind     string
	Python     string
	PromptOK   string
	PromptErr  string
}

// Config centralizes the functional parameters, glyph representations, and theme colors.
type Config struct {
	Timeout       time.Duration
	IconPrompt    string
	IconFolder    string
	IconGitBranch string
	IconStaged    string
	IconUnstaged  string
	IconUntracked string
	IconAhead     string
	IconBehind    string
	IconPython    string
	Colors        Colors
}

	Timeout:       250 * time.Millisecond,
var DefaultConfig = Config{
	IconPrompt:    "$",
	IconFolder:    "",
	IconGitBranch: "\ue702 ",
	IconStaged:    "+",
	IconUnstaged:  "+",
	IconUntracked: "?",
	IconAhead:     "⇡",
	IconBehind:    "⇣",
	IconPython:    "\ue73c ",
	Colors: Colors{
		PathParent: Map("cyan"),
		PathActive: Map("bold_cyan"),
		Branch:     Map("#babdbf"),
		Staged:     Map("green"),
		Unstaged:   Map("red"),
		Untracked:  Map("yellow"),
		Ahead:      Map("blue"),
		Behind:     Map("yellow"),
		Python:     Map("green"),
		PromptOK:   Map("red"),
		PromptErr:  Map("bold_red"),
	},
}
