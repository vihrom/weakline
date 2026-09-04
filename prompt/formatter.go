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
// Optimized version to minimize memory allocations and slicing overhead.
func formatPath(cfg config.Config, displayPath string) string {
	if displayPath == "~" || displayPath == "/" {
		return color(cfg.Colors.PathActive, displayPath)
	}

	idx := strings.LastIndex(displayPath, "/")
	if idx == -1 {
		return color(cfg.Colors.PathActive, displayPath)
	}

	// Safely split into parents and the final active directory segment
	parents := displayPath[:idx]
	active := displayPath[idx:] // Includes the leading slash, e.g., "/kubernetes"

	// Exceptional case for root paths like "/etc"
	if parents == "" {
		parents = "/"
		active = displayPath[1:]
	}

	// Colorize paths using Zsh color delimiters
	formattedParents := color(cfg.Colors.PathParent, parents)
	formattedActive := color(cfg.Colors.PathActive, active)

	return formattedParents + formattedActive
}

// getCacheFilePath returns a secure, unique temporary file path for the current directory.
// Fixes security flaw by isolating files inside a private user-owned directory.
func getCacheFilePath() string {
	pwd, _ := os.Getwd()

	// Create a user-specific subdirectory inside tmp (e.g., /tmp/weakline-1000)
	uid := os.Getuid()
	userTmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("weakline-%d", uid))

	// Ensure directory exists with strict permissions: owner can rwx, others nothing (0700)
	_ = os.MkdirAll(userTmpDir, 0700)

	// Hash-friendly string escaping for the current working directory path
	safeName := strings.ReplaceAll(pwd, "/", "_")
	return filepath.Join(userTmpDir, fmt.Sprintf("async%s", safeName))
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

	// Writing file securely inside the owner-only directory
	_ = os.WriteFile(cachePath, []byte(newResult), 0o600)
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
	line2 := color(arrowColor, cfg.IconPrompt)

	line1 := fmt.Sprintf("%s%s%s", folderSegment, venvSegment, gitSegment)

	return fmt.Sprintf("\n%s\n%s ", line1, line2)
}
