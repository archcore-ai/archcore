// Package testsupport holds helpers shared by tests in more than one package.
// Go cannot export helpers declared in _test.go files, so anything two packages
// need lives here as ordinary code.
package testsupport

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// commitSeq gives every commit in a process run a distinct, increasing
// timestamp. Fixed dates alone would make ordering depend on the commit graph;
// wall-clock dates would make it depend on how fast the machine is.
var commitSeq atomic.Int64

// IsolateGit detaches every git subprocess in this process — the helpers below
// and the production runner in internal/git alike — from the developer's global
// configuration. Call it from TestMain.
//
// This is not cosmetic: internal/git spawns git with the test process's
// environment, so a global commit.gpgsign, core.hooksPath or log.showSignature
// changes what the code under test observes.
func IsolateGit() {
	os.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	os.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	os.Setenv("GIT_TERMINAL_PROMPT", "0")
}

// RequireGit skips instead of failing when the machine has no git — an absent
// tool must not read as a broken build.
func RequireGit(tb testing.TB) {
	tb.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		tb.Skip("git not available on PATH")
	}
}

// NewGitRepo initializes a repository in dir and returns dir. The branch name is
// pinned because it reaches user-visible output through the session context.
func NewGitRepo(tb testing.TB, dir string) string {
	tb.Helper()
	RequireGit(tb)
	RunGit(tb, dir, "init", "-b", "main")
	return dir
}

// GitCommit stages everything in dir and commits it. --no-verify keeps a user's
// global hooksPath out of the run.
func GitCommit(tb testing.TB, dir, msg string) {
	tb.Helper()
	RunGit(tb, dir, "add", "-A")
	RunGit(tb, dir, "-c", "commit.gpgsign=false", "commit", "--no-verify", "-m", msg)
}

// GitDetachHead moves HEAD off the branch, the state in which CurrentBranch must
// report no branch rather than a placeholder.
func GitDetachHead(tb testing.TB, dir string) {
	tb.Helper()
	RunGit(tb, dir, "checkout", "--detach", "HEAD")
}

// RunGit runs one git command against dir and fails the test on a non-zero exit.
// The environment is built per command rather than with t.Setenv, so callers
// stay safe under t.Parallel().
func RunGit(tb testing.TB, dir string, args ...string) {
	tb.Helper()
	stamp := fmt.Sprintf("2024-01-01T00:00:%02dZ", commitSeq.Add(1)%60)
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_SYSTEM="+os.DevNull,
		"GIT_TERMINAL_PROMPT=0",
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_AUTHOR_DATE="+stamp,
		"GIT_COMMITTER_DATE="+stamp,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		tb.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// WriteFile writes content at a slash-separated path relative to dir, creating
// parents. For documents prefer BuildCorpus or the caller's own document helper.
func WriteFile(tb testing.TB, dir, relPath, content string) {
	tb.Helper()
	full := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		tb.Fatal(err)
	}
}
