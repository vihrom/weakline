package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// writeColored writes text wrapped in ANSI escape codes and Zsh escape sequences
// directly to the strings.Builder. This bypasses costly fmt.Sprintf allocations.
func writeColored(sb *strings.Builder, ansiCode string, text string) {
	if text == "" {
		return
	}
	sb.WriteString("%{\x1b[")
	sb.WriteString(ansiCode)
	sb.WriteString("m%}")
	sb.WriteString(text)
	sb.WriteString("%{\x1b[0m%}")
}

// shortenHome replaces the leading home directory path with a tilde (~).
func shortenHome(dir string) string {
	home := os.Getenv("HOME")
	if trimmed, found := strings.CutPrefix(dir, home); found {
		return "~" + trimmed
	}
	return dir
}

// writeFormattedPath colors parent and active directories directly into the strings.Builder.
func writeFormattedPath(sb *strings.Builder, cfg Config, displayPath string) {
	if displayPath == "~" || displayPath == "/" {
		writeColored(sb, cfg.Colors.PathActive, displayPath)
		return
	}

	idx := strings.LastIndex(displayPath, "/")
	if idx == -1 {
		writeColored(sb, cfg.Colors.PathActive, displayPath)
		return
	}

	parents := displayPath[:idx]
	active := displayPath[idx:]

	if parents == "" {
		parents = "/"
		active = displayPath[1:]
	}

	writeColored(sb, cfg.Colors.PathParent, parents)
	writeColored(sb, cfg.Colors.PathActive, active)
}


// buildGitStatusString formats individual status indicators into a single styled string.
func buildGitStatusString(cfg Config, st Status) string {
	if !st.IsGit {
		return ""
	}

	var sb strings.Builder
	sb.Grow(128)

	sb.WriteString(" ")

	sb.WriteString("%{\x1b[")
	sb.WriteString(cfg.Colors.Branch)
	sb.WriteString("m%}")
	sb.WriteString(cfg.IconGitBranch)
	sb.WriteString(st.Branch)
	sb.WriteString("%{\x1b[0m%}")

	statuses := [...]struct {
		count int
		color string
		icon  string
	}{
		{st.Ahead, cfg.Colors.Ahead, cfg.IconAhead},
		{st.Behind, cfg.Colors.Behind, cfg.IconBehind},
		{st.Staged, cfg.Colors.Staged, cfg.IconStaged},
		{st.Unstaged, cfg.Colors.Unstaged, cfg.IconUnstaged},
		{st.Untracked, cfg.Colors.Untracked, cfg.IconUntracked},
	}

	var buf [32]byte

	for _, s := range statuses {
		if s.count <= 0 {
			continue
		}

		sb.WriteString(" ")
		sb.WriteString("%{\x1b[")
		sb.WriteString(s.color)
		sb.WriteString("m%}")
		sb.WriteString(s.icon)

		res := strconv.AppendInt(buf[:0], int64(s.count), 10)
		sb.Write(res)

		sb.WriteString("%{\x1b[0m%}")
	}

	return sb.String()
}

// writeVenv writes a formatted Python virtual environment segment directly to the builder.
func writeVenv(sb *strings.Builder, cfg Config) {
	venv := os.Getenv("VIRTUAL_ENV")
	if venv == "" {
		return
	}
	envName := filepath.Base(venv)

	sb.WriteString(" ")
	sb.WriteString("%{\x1b[")
	sb.WriteString(cfg.Colors.Python)
	sb.WriteString("m%}")
	sb.WriteString("(")
	sb.WriteString(cfg.IconPython)
	sb.WriteString(envName)
	sb.WriteString(")")
	sb.WriteString("%{\x1b[0m%}")
}

// Render builds and returns the complete two-line terminal prompt using a single strings.Builder.
func Render(cfg Config, exitCode int) string {
	dir, _ := os.Getwd()
	displayPath := shortenHome(dir)

	var sb strings.Builder
	sb.Grow(512)

	sb.WriteString("\n")

	writeColored(&sb, cfg.Colors.PathParent, cfg.IconFolder)
	writeFormattedPath(&sb, cfg, displayPath)
	writeVenv(&sb, cfg)

	if cacheData, err := os.ReadFile(getCacheFilePath()); err == nil {
		sb.Write(cacheData)
	}

	sb.WriteString("\n")

	arrowColor := cfg.Colors.PromptOK
	if exitCode != 0 {
		arrowColor = cfg.Colors.PromptErr
	}
	writeColored(&sb, arrowColor, cfg.IconPrompt)
	sb.WriteString(" ")

	return sb.String()
}
