// Package prompt handles the formatting and rendering of the Zsh prompt string.
package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"weakline/config"
	"weakline/git"
)

// color wraps text with ANSI escape codes and Zsh escaping sequences.
func color(ansiCode string, text string) string {
	return fmt.Sprintf("%%{\033[%sm%%}%s%%{\033[0m%%}", ansiCode, text)
}

// shortenHome replaces the leading home directory path with a tilde (~).
func shortenHome(dir string) string {
	home := os.Getenv("HOME")
	if trimmed, found := strings.CutPrefix(dir, home); found {
		return "~" + trimmed
	}
	return dir
}

// formatPath colors the parent directories and active directory separately.
func formatPath(cfg config.Config, displayPath string) string {
	if displayPath == "~" || displayPath == "/" {
		return color(cfg.Colors.PathActive, displayPath)
	}

	parts := strings.Split(displayPath, "/")
	var parentParts []string

	for i, part := range parts {
		if part == "" {
			continue
		}

		if i == len(parts)-1 {
			parentSlash := color(cfg.Colors.PathParent, "/")
			parentsFormatted := strings.Join(parentParts, parentSlash)

			if strings.HasPrefix(displayPath, "/") {
				parentsFormatted = parentSlash + parentsFormatted
			}

			if len(parentParts) > 0 {
				activeSegment := color(cfg.Colors.PathActive, "/"+part)
				return parentsFormatted + activeSegment
			}

			return color(cfg.Colors.PathActive, part)
		}

		parentParts = append(parentParts, color(cfg.Colors.PathParent, part))
	}

	return color(cfg.Colors.PathActive, displayPath)
}

// getCacheFilePath returns a unique temporary file path for the current working directory.
func getCacheFilePath() string {
	pwd, _ := os.Getwd()
	safeName := strings.ReplaceAll(pwd, "/", "_")
	return filepath.Join(os.TempDir(), fmt.Sprintf("weakline_async%s", safeName))
}

// buildGitStatusString formats individual status indicators into a single styled string.
func buildGitStatusString(cfg config.Config, st git.Status) string {
	if !st.IsGit {
		return ""
	}

	var parts []string
	if st.Ahead > 0 {
		parts = append(parts, color(cfg.Colors.Ahead, fmt.Sprintf("%s%d", cfg.IconAhead, st.Ahead)))
	}
	if st.Behind > 0 {
		parts = append(parts, color(cfg.Colors.Behind, fmt.Sprintf("%s%d", cfg.IconBehind, st.Behind)))
	}
	if st.Staged > 0 {
		parts = append(parts, color(cfg.Colors.Staged, fmt.Sprintf("%s%d", cfg.IconStaged, st.Staged)))
	}
	if st.Unstaged > 0 {
		parts = append(parts, color(cfg.Colors.Unstaged, fmt.Sprintf("%s%d", cfg.IconUnstaged, st.Unstaged)))
	}
	if st.Untracked > 0 {
		parts = append(parts, color(cfg.Colors.Untracked, fmt.Sprintf("%s%d", cfg.IconUntracked, st.Untracked)))
	}

	statusStr := ""
	if len(parts) > 0 {
		statusStr = " " + strings.Join(parts, " ")
	}

	branchStr := color(cfg.Colors.Branch, cfg.IconGitBranch+st.Branch)
	return fmt.Sprintf(" %s%s", branchStr, statusStr)
}

// WriteAsyncCache evaluates and saves formatted Git status to a temp file.
func WriteAsyncCache(cfg config.Config, st git.Status) bool {
	cachePath := getCacheFilePath()
	newResult := buildGitStatusString(cfg, st)

	oldData, _ := os.ReadFile(cachePath)
	if string(oldData) == newResult {
		return false
	}

	_ = os.WriteFile(cachePath, []byte(newResult), 0o644)
	return true
}

// renderVenv returns a formatted Python virtual environment segment if active.
func renderVenv(cfg config.Config) string {
	venv := os.Getenv("VIRTUAL_ENV")
	if venv == "" {
		return ""
	}
	envName := filepath.Base(venv)
	return " " + color(cfg.Colors.Python, fmt.Sprintf("(%s%s)", cfg.IconPython, envName))
}

// renderGitCache reads and returns the cached Git status string.
func renderGitCache() string {
	if cacheData, err := os.ReadFile(getCacheFilePath()); err == nil {
		return string(cacheData)
	}
	return ""
}

// Render builds and returns the complete two-line terminal prompt.
func Render(cfg config.Config, exitCode int) string {
	dir, _ := os.Getwd()
	displayPath := shortenHome(dir)

	folderIcon := color(cfg.Colors.PathParent, cfg.IconFolder)
	folderSegment := folderIcon + formatPath(cfg, displayPath)

	venvSegment := renderVenv(cfg)
	gitSegment := renderGitCache()

	arrowColor := cfg.Colors.PromptOK
	if exitCode != 0 {
		arrowColor = cfg.Colors.PromptErr
	}
	line2 := color(arrowColor, "❯")

	line1 := fmt.Sprintf("%s%s%s", folderSegment, venvSegment, gitSegment)

	return fmt.Sprintf("\n%s\n%s ", line1, line2)
}
