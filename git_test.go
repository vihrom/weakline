//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// v2 joins porcelain v2 lines the way git prints them.
func v2(lines ...string) []byte {
	return []byte(strings.Join(lines, "\n") + "\n")
}

const fakeOID = "0123456789abcdef0123456789abcdef01234567"

func TestParseStatusV2(t *testing.T) {
	entry := "N... 100644 100644 100644 " + fakeOID + " " + fakeOID
	tests := []struct {
		name string
		out  []byte
		want Status
	}{
		{
			name: "initial commit, no upstream",
			out:  v2("# branch.oid (initial)", "# branch.head main"),
			want: Status{IsGit: true, Branch: "main"},
		},
		{
			name: "clean, in sync with upstream",
			out: v2("# branch.oid "+fakeOID, "# branch.head main",
				"# branch.upstream origin/main", "# branch.ab +0 -0"),
			want: Status{IsGit: true, Branch: "main"},
		},
		{
			name: "ahead and behind",
			out: v2("# branch.oid "+fakeOID, "# branch.head feature/x",
				"# branch.upstream origin/feature/x", "# branch.ab +2 -13"),
			want: Status{IsGit: true, Branch: "feature/x", Ahead: 2, Behind: 13},
		},
		{
			name: "upstream gone has no ab line",
			out: v2("# branch.oid "+fakeOID, "# branch.head feat",
				"# branch.upstream origin/feat"),
			want: Status{IsGit: true, Branch: "feat"},
		},
		{
			name: "detached HEAD shows the short commit id",
			out:  v2("# branch.oid "+fakeOID, "# branch.head (detached)"),
			want: Status{IsGit: true, Detached: true, Branch: "0123456"},
		},
		{
			name: "staged, unstaged, renamed, untracked, ignored",
			out: v2("# branch.oid "+fakeOID, "# branch.head main",
				"1 .M "+entry+" a.txt",
				"2 R. "+entry+" R100 b2.txt\tb.txt",
				"1 MM "+entry+" c.txt",
				"1 A. "+entry+" new file.txt",
				"? untracked.txt",
				"! ignored.txt"),
			want: Status{IsGit: true, Branch: "main", Staged: 3, Unstaged: 2, Untracked: 1},
		},
		{
			name: "merge conflict counts once",
			out: v2("# branch.oid "+fakeOID, "# branch.head main",
				"u UU N... 100644 100644 100644 100644 "+fakeOID+" "+fakeOID+" "+fakeOID+" a.txt"),
			want: Status{IsGit: true, Branch: "main", Unstaged: 1},
		},
		{
			name: "no trailing newline",
			out:  []byte("# branch.head main\n? x"),
			want: Status{IsGit: true, Branch: "main", Untracked: 1},
		},
		{
			name: "empty output falls back to HEAD",
			out:  nil,
			want: Status{IsGit: true, Branch: "HEAD"},
		},
		{
			name: "short entry line is ignored",
			out:  v2("# branch.head main", "1 ."),
			want: Status{IsGit: true, Branch: "main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseStatusV2(tt.out); got != tt.want {
				t.Errorf("parseStatusV2() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFindGitRoot(t *testing.T) {
	unsetEnv(t, "GIT_DIR")

	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findGitRoot(sub); got != repo {
		t.Errorf("findGitRoot(%q) = %q, want %q", sub, got, repo)
	}
	if got := findGitRoot(repo); got != repo {
		t.Errorf("findGitRoot(%q) = %q, want %q", repo, got, repo)
	}

	// A ".git" file marks worktrees and submodules.
	worktree := t.TempDir()
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte("gitdir: /elsewhere"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := findGitRoot(worktree); got != worktree {
		t.Errorf("findGitRoot(%q) = %q, want the directory itself", worktree, got)
	}

	if got := findGitRoot(t.TempDir()); got != "" {
		t.Errorf("findGitRoot outside a repository = %q, want empty", got)
	}

	t.Setenv("GIT_DIR", "/somewhere/.git")
	if got := findGitRoot(sub); got != sub {
		t.Errorf("with GIT_DIR set, findGitRoot = %q, want %q", got, sub)
	}
}

func TestFindGitRootHonorsCeilingDirectories(t *testing.T) {
	unsetEnv(t, "GIT_DIR")

	// A dotfiles repository in "home" must not swallow every directory below it.
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(home, "proj", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	if got := findGitRoot(sub); got != home {
		t.Fatalf("without a ceiling findGitRoot(%q) = %q, want %q", sub, got, home)
	}

	// Relative entries are ignored, as in git; the list is colon-separated.
	t.Setenv("GIT_CEILING_DIRECTORIES", "relative/dir:/nonexistent:"+home+"/")
	if got := findGitRoot(sub); got != "" {
		t.Errorf("with the ceiling findGitRoot(%q) = %q, want empty", sub, got)
	}
	if got := findGitRoot(home); got != home {
		t.Errorf("the ceiling directory itself is still checked, got %q, want %q", got, home)
	}

	// A repository below the ceiling is still found.
	if err := os.Mkdir(filepath.Join(sub, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findGitRoot(sub); got != sub {
		t.Errorf("findGitRoot(%q) = %q, want %q", sub, got, sub)
	}
}

func TestGitCommandFlags(t *testing.T) {
	cfg := DefaultConfig
	args := gitCommand(context.Background(), cfg, "", "status").Args

	joined := " " + strings.Join(args, " ") + " "
	for _, want := range []string{" --no-optional-locks ", " -c core.fsmonitor=false ", " status "} {
		if !strings.Contains(joined, want) {
			t.Errorf("git args %v are missing %q", args, want)
		}
	}

	cfg.AllowFSMonitor = true
	joined = " " + strings.Join(gitCommand(context.Background(), cfg, "", "status").Args, " ") + " "
	if strings.Contains(joined, "core.fsmonitor") {
		t.Error("core.fsmonitor must not be overridden when AllowFSMonitor is set")
	}
}

