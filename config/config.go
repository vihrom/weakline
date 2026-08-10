// Package config defines default icons, colors, and styling settings for the prompt.
package config

import "time"

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

var Default = Config{
	Timeout:       250 * time.Millisecond,
	IconPrompt:    "❯",
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
		PromptOK:   Map("bold_magenta"),
		PromptErr:  Map("bold_red"),
	},
}
