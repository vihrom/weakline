package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// writeColored writes text wrapped in ANSI escape codes and Zsh escape sequences directly to the strings.Builder.
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

func promptSafe(text string) string {
	clean := strings.Map(func(char rune) rune {
		if char < 0x20 || (char >= 0x7f && char <= 0x9f) {
			return '?'
		}
		return char
	}, text)
	return strings.ReplaceAll(clean, "%", "%%")
}

func shortenHome(dir string) string {
	home := strings.TrimRight(os.Getenv("HOME"), "/")
	if home != "" && (dir == home || strings.HasPrefix(dir, home+"/")) {
		return "~" + dir[len(home):]
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
		writeColored(sb, cfg.Colors.PathActive, promptSafe(displayPath))
		return
	}

	parents := displayPath[:idx]
	active := displayPath[idx:]

	if parents == "" {
		parents = "/"
		active = displayPath[1:] // Anti-double slash guard for root-level folders (e.g. /usr)
	}

	writeColored(sb, cfg.Colors.PathParent, promptSafe(parents))
	writeColored(sb, cfg.Colors.PathActive, promptSafe(active))
}

func buildGitStatusString(cfg Config, st Status) string {
	if !st.IsGit || st.Err != nil {
		return ""
	}

	var sb strings.Builder
	sb.Grow(128)

	label := promptSafe(st.Branch)
	if st.Detached {
		label = cfg.IconDetached + label
	}
	sb.WriteString(" ")
	writeColored(&sb, cfg.Colors.Branch, cfg.IconGitBranch+label)

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

	for _, s := range statuses {
		if s.count <= 0 {
			continue
		}
		sb.WriteString(" ")
		writeColored(&sb, s.color, s.icon+strconv.Itoa(s.count))
	}

	if st.IsTimeout && cfg.IconTimeout != "" {
		sb.WriteString(" ")
		writeColored(&sb, cfg.Colors.Timeout, cfg.IconTimeout)
	}

	return sb.String()
}

func venvName(venv string) string {
	name := filepath.Base(venv)
	if name == ".venv" || name == "venv" || name == "env" {
		name = filepath.Base(filepath.Dir(venv))
	}
	return name
}

func writeVenv(sb *strings.Builder, cfg Config) {
	venv := os.Getenv("VIRTUAL_ENV")
	if venv == "" {
		return
	}

	sb.WriteString(" ")
	writeColored(sb, cfg.Colors.Python, "("+cfg.IconPython+promptSafe(venvName(venv))+")")
}

func renderPrompt(cfg Config, exitCode int, pwd string, gitStatus []byte) string {
	var sb strings.Builder
	sb.Grow(512)

	sb.WriteString("\n")

	writeColored(&sb, cfg.Colors.PathParent, cfg.IconFolder)
	writeFormattedPath(&sb, cfg, shortenHome(pwd))
	writeVenv(&sb, cfg)
	sb.Write(gitStatus)

	sb.WriteString("\n")

	arrowColor := cfg.Colors.PromptOK
	if exitCode != 0 {
		arrowColor = cfg.Colors.PromptErr
	}
	writeColored(&sb, arrowColor, cfg.IconPrompt)
	sb.WriteString(" ")

	return sb.String()
}

func Render(cfg Config, exitCode int) string {
	pwd, _ := os.Getwd()
	return renderPrompt(cfg, exitCode, pwd, readCache(pwd))
}
