//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"strings"
	"testing"
)

// colored mirrors writeColored's output so tests state the expected wire format explicitly.
func colored(code, text string) string {
	return "%{\x1b[" + code + "m%}" + text + "%{\x1b[0m%}"
}

func TestShortenHome(t *testing.T) {
	tests := []struct {
		name string
		home string
		dir  string
		want string
	}{
		{"home itself", "/home/dev", "/home/dev", "~"},
		{"below home", "/home/dev", "/home/dev/projects/x", "~/projects/x"},
		{"sibling with the same prefix", "/home/dev", "/home/developer/x", "/home/developer/x"},
		{"sibling, exact prefix", "/home/dev", "/home/dev2", "/home/dev2"},
		{"outside home", "/home/dev", "/usr/lib", "/usr/lib"},
		{"trailing slash in HOME", "/home/dev/", "/home/dev/x", "~/x"},
		{"trailing slash, home itself", "/home/dev/", "/home/dev", "~"},
		{"HOME unset", "", "/home/dev/x", "/home/dev/x"},
		{"HOME is the root", "/", "/usr", "/usr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", tt.home)
			if got := shortenHome(tt.dir); got != tt.want {
				t.Errorf("shortenHome(%q) with HOME=%q = %q, want %q", tt.dir, tt.home, got, tt.want)
			}
		})
	}
}

func TestPromptSafe(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "main", "main"},
		{"unicode is kept", "ветка-日本", "ветка-日本"},
		{"percent is doubled", "100%", "100%%"},
		{"prompt escape is neutralized", "%{x%}%F{red}", "%%{x%%}%%F{red}"},
		{"ESC becomes ?", "a\x1b[31mb", "a?[31mb"},
		{"newline and tab", "a\nb\tc", "a?b?c"},
		{"DEL", "a\x7fb", "a?b"},
		{"C1 control", "a\u009bb", "a?b"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := promptSafe(tt.in); got != tt.want {
				t.Errorf("promptSafe(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestWriteFormattedPath(t *testing.T) {
	cfg := DefaultConfig
	parent := func(text string) string { return colored(cfg.Colors.PathParent, text) }
	active := func(text string) string { return colored(cfg.Colors.PathActive, text) }

	tests := []struct {
		name string
		path string
		want string
	}{
		{"home", "~", active("~")},
		{"root", "/", active("/")},
		{"root level folder", "/usr", parent("/") + active("usr")},
		{"one level below home", "~/a", parent("~") + active("/a")},
		{"deep path", "~/projects/weakline", parent("~/projects") + active("/weakline")},
		{"no slash", "plain", active("plain")},
		{"percent in a directory name", "~/100%", parent("~") + active("/100%%")},
		{"escape in a directory name", "/tmp/\x1b[2J", parent("/tmp") + active("/?[2J")},
		{"empty (deleted cwd)", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			writeFormattedPath(&sb, cfg, tt.path)
			if got := sb.String(); got != tt.want {
				t.Errorf("writeFormattedPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestBuildGitStatusString(t *testing.T) {
	cfg := DefaultConfig
	branch := func(label string) string { return " " + colored(cfg.Colors.Branch, cfg.IconGitBranch+label) }

	tests := []struct {
		name string
		st   Status
		want string
	}{
		{"not a repository", Status{}, ""},
		{"git error", Status{IsGit: true, Err: errTest}, ""},
		{"clean branch", Status{IsGit: true, Branch: "main"}, branch("main")},
		{
			"every counter, in display order",
			Status{IsGit: true, Branch: "main", Ahead: 1, Behind: 2, Staged: 3, Unstaged: 4, Untracked: 5},
			branch("main") +
				" " + colored(cfg.Colors.Ahead, cfg.IconAhead+"1") +
				" " + colored(cfg.Colors.Behind, cfg.IconBehind+"2") +
				" " + colored(cfg.Colors.Staged, cfg.IconStaged+"3") +
				" " + colored(cfg.Colors.Unstaged, cfg.IconUnstaged+"4") +
				" " + colored(cfg.Colors.Untracked, cfg.IconUntracked+"5"),
		},
		{"detached HEAD", Status{IsGit: true, Branch: "a1b2c3d", Detached: true}, branch(cfg.IconDetached + "a1b2c3d")},
		{
			"timeout shows the branch and a marker",
			Status{IsGit: true, IsTimeout: true, Branch: "main"},
			branch("main") + " " + colored(cfg.Colors.Timeout, cfg.IconTimeout),
		},
		{"branch name with a prompt escape", Status{IsGit: true, Branch: "br%{x"}, branch("br%%{x")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildGitStatusString(cfg, tt.st); got != tt.want {
				t.Errorf("buildGitStatusString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVenvName(t *testing.T) {
	tests := []struct {
		venv string
		want string
	}{
		{"/home/dev/myenv", "myenv"},
		{"/home/dev/project/.venv", "project"},
		{"/home/dev/project/venv", "project"},
		{"/home/dev/project/env", "project"},
		{"/opt/venvs/data-science", "data-science"},
	}

	for _, tt := range tests {
		if got := venvName(tt.venv); got != tt.want {
			t.Errorf("venvName(%q) = %q, want %q", tt.venv, got, tt.want)
		}
	}
}

func TestWriteVenv(t *testing.T) {
	cfg := DefaultConfig

	t.Run("unset", func(t *testing.T) {
		unsetEnv(t, "VIRTUAL_ENV")
		var sb strings.Builder
		writeVenv(&sb, cfg)
		if sb.Len() != 0 {
			t.Errorf("writeVenv() with no venv = %q, want nothing", sb.String())
		}
	})

	t.Run("set", func(t *testing.T) {
		t.Setenv("VIRTUAL_ENV", "/home/dev/project/.venv")
		var sb strings.Builder
		writeVenv(&sb, cfg)
		want := " " + colored(cfg.Colors.Python, "("+cfg.IconPython+"project)")
		if got := sb.String(); got != want {
			t.Errorf("writeVenv() = %q, want %q", got, want)
		}
	})

	t.Run("hostile name", func(t *testing.T) {
		t.Setenv("VIRTUAL_ENV", "/home/dev/50%\x1b")
		var sb strings.Builder
		writeVenv(&sb, cfg)
		if got := sb.String(); !strings.Contains(got, "50%%?") {
			t.Errorf("writeVenv() = %q, want the name sanitized", got)
		}
	})
}

func TestRenderPrompt(t *testing.T) {
	cfg := DefaultConfig
	cfg.Colors.PromptOK = "32"
	cfg.Colors.PromptErr = "31"
	t.Setenv("HOME", "/home/dev")
	unsetEnv(t, "VIRTUAL_ENV")

	ok := renderPrompt(cfg, 0, "/home/dev/proj", []byte(" GIT"))
	if !strings.HasPrefix(ok, "\n") || !strings.HasSuffix(ok, " ") {
		t.Errorf("prompt must start with a newline and end with a space: %q", ok)
	}
	if !strings.Contains(ok, colored("32", cfg.IconPrompt)) || strings.Contains(ok, colored("31", cfg.IconPrompt)) {
		t.Errorf("exit code 0 must use the OK arrow: %q", ok)
	}
	if !strings.Contains(ok, " GIT\n") {
		t.Errorf("cached git segment is missing: %q", ok)
	}
	if !strings.Contains(ok, "~") || strings.Contains(ok, "/home/dev") {
		t.Errorf("home directory was not shortened: %q", ok)
	}

	failed := renderPrompt(cfg, 2, "/home/dev/proj", nil)
	if !strings.Contains(failed, colored("31", cfg.IconPrompt)) || strings.Contains(failed, colored("32", cfg.IconPrompt)) {
		t.Errorf("non-zero exit code must use the error arrow: %q", failed)
	}
}

func BenchmarkRenderPrompt(b *testing.B) {
	cfg := DefaultConfig
	cache := []byte(buildGitStatusString(cfg, Status{IsGit: true, Branch: "main", Staged: 2, Unstaged: 1, Untracked: 3}))

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = renderPrompt(cfg, 0, "/home/dev/projects/weakline", cache)
	}
}

func BenchmarkBuildGitStatusString(b *testing.B) {
	cfg := DefaultConfig
	st := Status{IsGit: true, Branch: "main", Ahead: 1, Staged: 2, Unstaged: 1, Untracked: 3}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = buildGitStatusString(cfg, st)
	}
}
