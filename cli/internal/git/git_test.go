package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"archcore-cli/internal/testsupport"
)

func requireGit(t *testing.T) {
	t.Helper()
	testsupport.RequireGit(t)
}

// initRepo creates a repository with one commit and returns its directory.
func initRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := testsupport.NewGitRepo(t, t.TempDir())
	for name, body := range files {
		testsupport.WriteFile(t, dir, name, body)
	}
	if len(files) > 0 {
		testsupport.GitCommit(t, dir, "initial")
	}
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	testsupport.RunGit(t, dir, args...)
}

func TestDetectRepoURL_NonGitDir(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	got := DetectRepoURL(context.Background(), dir)
	if got != "" {
		t.Errorf("DetectRepoURL(non-git) = %q, want empty string", got)
	}
}

func TestDetectRepoURL_WithRemote(t *testing.T) {
	dir := initRepo(t, nil)
	runGit(t, dir, "remote", "add", "origin", "https://github.com/example/repo.git")

	got := DetectRepoURL(context.Background(), dir)
	want := "https://github.com/example/repo.git"
	if got != want {
		t.Errorf("DetectRepoURL = %q, want %q", got, want)
	}
}

func TestDetectRepoURL_NoOriginRemote(t *testing.T) {
	dir := initRepo(t, nil)
	got := DetectRepoURL(context.Background(), dir)
	if got != "" {
		t.Errorf("DetectRepoURL(no origin) = %q, want empty string", got)
	}
}

// TestRun_GitAbsent pins the distinction the empty-string contract hides: a
// machine without git must be reportable as such, not as "no remote".
func TestRun_GitAbsent(t *testing.T) {
	original := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("not found") }
	t.Cleanup(func() { lookPath = original })

	if _, _, err := run(context.Background(), t.TempDir(), "status"); !errors.Is(err, ErrGitAbsent) {
		t.Errorf("run() error = %v, want ErrGitAbsent", err)
	}
	if got := DetectRepoURL(context.Background(), t.TempDir()); got != "" {
		t.Errorf("DetectRepoURL without git = %q, want empty string", got)
	}
	if _, err := LastCommitTouching(context.Background(), t.TempDir(), "."); !errors.Is(err, ErrGitAbsent) {
		t.Errorf("LastCommitTouching without git = %v, want ErrGitAbsent", err)
	}
}

// TestLastCommitTouching_OutsideARepo pins what replaced the IsRepo probe: the
// query itself reports "not a repository", so callers need no separate check.
func TestLastCommitTouching_OutsideARepo(t *testing.T) {
	requireGit(t)
	if _, err := LastCommitTouching(context.Background(), t.TempDir(), ".archcore/"); err == nil {
		t.Error("LastCommitTouching outside a repository returned no error")
	}
}

// TestCapWriter_TruncatesAtTheCap: git output is unbounded in principle, and
// ChangedSince over a long gap lists every changed file.
func TestCapWriter_TruncatesAtTheCap(t *testing.T) {
	t.Parallel()
	var w capWriter
	chunk := make([]byte, 4096)
	for range (maxOutput / len(chunk)) + 2 {
		if n, err := w.Write(chunk); n != len(chunk) || err != nil {
			t.Fatalf("Write() = %d, %v; want %d, nil", n, err, len(chunk))
		}
	}
	if len(w.buf) > maxOutput {
		t.Errorf("buffered %d bytes, want at most %d", len(w.buf), maxOutput)
	}
	if !w.truncated {
		t.Error("truncated flag not set after exceeding the cap")
	}
}

// TestCapWriter_KeepsSmallOutputWhole is the counterpart: an ordinary result
// must arrive intact and unflagged.
func TestCapWriter_KeepsSmallOutputWhole(t *testing.T) {
	t.Parallel()
	var w capWriter
	w.Write([]byte("a.txt\nb.txt\n"))
	if got := string(w.buf); got != "a.txt\nb.txt\n" {
		t.Errorf("buf = %q, want the input verbatim", got)
	}
	if w.truncated {
		t.Error("small output was flagged truncated")
	}
}

func TestCurrentBranch(t *testing.T) {
	dir := initRepo(t, map[string]string{"a.txt": "a"})
	branch, err := CurrentBranch(context.Background(), dir)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch == "" || branch == "HEAD" {
		t.Errorf("CurrentBranch = %q, want a branch name", branch)
	}
}

func TestCurrentBranch_NonGitDir(t *testing.T) {
	requireGit(t)
	if _, err := CurrentBranch(context.Background(), t.TempDir()); err == nil {
		t.Error("CurrentBranch(non-git) = nil error, want error")
	}
}

func TestLastCommitTouching(t *testing.T) {
	dir := initRepo(t, map[string]string{".archcore/a.adr.md": "doc", "src/main.go": "code"})

	sha, err := LastCommitTouching(context.Background(), dir, ".archcore/")
	if err != nil {
		t.Fatalf("LastCommitTouching: %v", err)
	}
	if len(sha) != 40 {
		t.Errorf("LastCommitTouching = %q, want a 40-char hash", sha)
	}

	// A pathspec with no history is an ordinary state, not a failure.
	empty, err := LastCommitTouching(context.Background(), dir, "no-such-dir/")
	if err != nil {
		t.Fatalf("LastCommitTouching(absent pathspec): %v", err)
	}
	if empty != "" {
		t.Errorf("LastCommitTouching(absent pathspec) = %q, want empty string", empty)
	}
}

func TestChangedSince(t *testing.T) {
	dir := initRepo(t, map[string]string{".archcore/a.adr.md": "doc", "src/main.go": "code"})
	sha, err := LastCommitTouching(context.Background(), dir, ".archcore/")
	if err != nil {
		t.Fatalf("LastCommitTouching: %v", err)
	}

	// Nothing has moved since the documentation commit.
	changed, err := ChangedSince(context.Background(), dir, sha, ":(exclude).archcore/")
	if err != nil {
		t.Fatalf("ChangedSince: %v", err)
	}
	if len(changed) != 0 {
		t.Errorf("ChangedSince(no new commits) = %v, want none", changed)
	}

	// Move the code, leave the documentation behind.
	if err := os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "code only")

	changed, err = ChangedSince(context.Background(), dir, sha, ":(exclude).archcore/")
	if err != nil {
		t.Fatalf("ChangedSince: %v", err)
	}
	if len(changed) != 1 || changed[0] != "src/main.go" {
		t.Errorf("ChangedSince = %v, want [src/main.go]", changed)
	}
}
