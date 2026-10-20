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

// --- integration tests with a real git ---

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
}

// isolateGit keeps the user's git configuration and identity out of the tests.
func isolateGit(t *testing.T) {
	t.Helper()
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE"} {
		unsetEnv(t, key)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// newRepo creates a repository on branch "main" with one commit of a.txt.
func newRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "symbolic-ref", "HEAD", "refs/heads/main")
	writeFile(t, filepath.Join(repo, "a.txt"), "a")
	gitIn(t, repo, "add", "a.txt")
	gitIn(t, repo, "commit", "-q", "-m", "init")
	return repo
}

func TestGetStatusRealRepository(t *testing.T) {
	requireGit(t)
	isolateGit(t)
	repo := newRepo(t)

	writeFile(t, filepath.Join(repo, "a.txt"), "changed") // unstaged
	writeFile(t, filepath.Join(repo, "b.txt"), "b")       // staged
	gitIn(t, repo, "add", "b.txt")
	writeFile(t, filepath.Join(repo, "c.txt"), "c") // untracked

	got := GetStatus(DefaultConfig, repo)
	want := Status{IsGit: true, Branch: "main", Staged: 1, Unstaged: 1, Untracked: 1}
	if got != want {
		t.Errorf("GetStatus() = %+v, want %+v", got, want)
	}
}

func TestGetStatusFromSubdirectory(t *testing.T) {
	requireGit(t)
	isolateGit(t)
	repo := newRepo(t)
	sub := filepath.Join(repo, "pkg")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo, "top.txt"), "x") // untracked, outside the subdirectory

	if got := GetStatus(DefaultConfig, sub); got.Untracked != 1 || got.Branch != "main" {
		t.Errorf("GetStatus() from a subdirectory = %+v", got)
	}
}

func TestGetStatusDetachedHead(t *testing.T) {
	requireGit(t)
	isolateGit(t)
	repo := newRepo(t)
	gitIn(t, repo, "checkout", "-q", "--detach")

	got := GetStatus(DefaultConfig, repo)
	if !got.Detached || len(got.Branch) != shortOIDLength {
		t.Errorf("GetStatus() = %+v, want a detached HEAD with a %d-char id", got, shortOIDLength)
	}
}

func TestGetStatusNotARepository(t *testing.T) {
	requireGit(t)
	isolateGit(t)
	dir := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))

	got := GetStatus(DefaultConfig, dir)
	if got.IsGit || got.Err == nil {
		t.Errorf("GetStatus() outside a repository = %+v, want IsGit=false and an error", got)
	}
}

func TestGetStatusTimeoutKeepsBranchMarker(t *testing.T) {
	requireGit(t)
	isolateGit(t)
	repo := newRepo(t)

	cfg := DefaultConfig
	cfg.Timeout = time.Nanosecond // always expired

	got := GetStatus(cfg, repo)
	if !got.IsGit || !got.IsTimeout || got.Err != nil || got.Branch != "HEAD" {
		t.Errorf("GetStatus() with an expired deadline = %+v", got)
	}
}

func TestQuickBranch(t *testing.T) {
	requireGit(t)
	isolateGit(t)
	repo := newRepo(t)

	if got := quickBranch(DefaultConfig, repo); got != "main" {
		t.Errorf("quickBranch() = %q, want main", got)
	}
	gitIn(t, repo, "checkout", "-q", "--detach")
	if got := quickBranch(DefaultConfig, repo); got != "" {
		t.Errorf("quickBranch() on a detached HEAD = %q, want empty", got)
	}
}

func TestGetStatusRapidChangesAfterAdd(t *testing.T) {
	requireGit(t)
	isolateGit(t)
	repo := newRepo(t)

	for i := 0; i < 5; i++ {
		fileName := "file" + strconv.Itoa(i) + ".txt"
		writeFile(t, filepath.Join(repo, fileName), "content")
		gitIn(t, repo, "add", fileName)

		got := GetStatus(DefaultConfig, repo)
		if !got.IsGit || got.Err != nil {
			t.Fatalf("iteration %d: GetStatus failed: %+v", i, got)
		}
		if got.Staged != i+1 {
			t.Errorf("iteration %d: got Staged = %d, want %d", i, got.Staged, i+1)
		}
	}
}

// A repository's own fsmonitor setting must not run when AllowFSMonitor is off.
func TestRepositoryFSMonitorHookIsNotExecuted(t *testing.T) {
	requireGit(t)
	isolateGit(t)
	repo := newRepo(t)

	marker := filepath.Join(t.TempDir(), "executed")
	hook := filepath.Join(t.TempDir(), "hook.sh")
	writeFile(t, hook, "#!/bin/sh\ntouch '"+marker+"'\n")
	if err := os.Chmod(hook, 0o700); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "config", "core.fsmonitor", hook)

	// Sanity check: a plain `git status` does run the repository's hook.
	plain := exec.Command("git", "status", "--porcelain")
	plain.Dir = repo
	_ = plain.Run()
	if _, err := os.Stat(marker); err != nil {
		t.Skip("this git version does not run the fsmonitor hook")
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}

	GetStatus(DefaultConfig, repo)
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the repository's core.fsmonitor command was executed")
	}

	// With AllowFSMonitor the opt-in works as documented.
	cfg := DefaultConfig
	cfg.AllowFSMonitor = true
	GetStatus(cfg, repo)
	if _, err := os.Stat(marker); err != nil {
		t.Error("AllowFSMonitor did not let git run the hook")
	}
}
