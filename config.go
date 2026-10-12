//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import "time"

// Colors encapsulates the ANSI code color mapping for every semantic element of the prompt.
// Supported color representation formats via Map():
// 1. Plain text names: "red", "cyan", "gray", etc.
// 2. Bold/Styled text names: "bold_red", "bold_cyan", "underline", "dim", etc.
// 3. HEX / TrueColor (24-bit): "#babdbf", "#FF0055", "RRGGBB", "#f05"
// 4. ANSI 256-color palette index: "196", "208", "39"
// 5. Raw SGR parameters: "1;31"
// Anything else maps to an empty (neutral) sequence instead of garbage.
type Colors struct {
	PathParent string
	PathActive string
	Branch     string
	Staged     string
	Unstaged   string
	Untracked  string
	Ahead      string
	Behind     string
	Timeout    string
	Python     string
	PromptOK   string
	PromptErr  string
}

// Config centralizes the functional parameters, glyph representations, and theme colors.
type Config struct {
	// Timeout bounds one git invocation in the background worker.
	Timeout time.Duration

	// AllowFSMonitor lets git run the repository's core.fsmonitor setting.
	// It is off by default: the prompt runs git automatically in every
	// directory you enter, and core.fsmonitor can be an arbitrary command
	// from the repository's own config. Enable it for huge repositories
	// that rely on the built-in fsmonitor daemon.
	AllowFSMonitor bool

	IconPrompt    string
	IconFolder    string
	IconGitBranch string
	IconDetached  string
	IconStaged    string
	IconUnstaged  string
	IconUntracked string
	IconAhead     string
	IconBehind    string
	IconTimeout   string
	IconPython    string
	Colors        Colors
}

var DefaultConfig = Config{
	Timeout:        500 * time.Millisecond,
	AllowFSMonitor: false,

	IconPrompt:    "$",
	IconFolder:    "",
	IconGitBranch: "\ue702 ",
	IconDetached:  "@",
	IconStaged:    "+",
	IconUnstaged:  "+",
	IconUntracked: "?",
	IconAhead:     "⇡",
	IconBehind:    "⇣",
	IconTimeout:   "…",
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
		Timeout:    Map("gray"),
		Python:     Map("green"),
		PromptOK:   Map("red"),
		PromptErr:  Map("bold_red"),
	},
}
