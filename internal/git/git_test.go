package git

import (
	"context"
	"os/exec"
	"testing"
)

func TestDetectRepoURL_NonGitDir(t *testing.T) {
	dir := t.TempDir()
	got := DetectRepoURL(context.Background(), dir)
	if got != "" {
		t.Errorf("DetectRepoURL(non-git) = %q, want empty string", got)
	}
}

func TestDetectRepoURL_WithRemote(t *testing.T) {
	dir := t.TempDir()

	// Initialize a git repo and set a remote.
	for _, args := range [][]string{
		{"init"},
		{"remote", "add", "origin", "https://github.com/example/repo.git"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	got := DetectRepoURL(context.Background(), dir)
	want := "https://github.com/example/repo.git"
	if got != want {
		t.Errorf("DetectRepoURL = %q, want %q", got, want)
	}
}

func TestDetectRepoURL_NoOriginRemote(t *testing.T) {
	dir := t.TempDir()

	// Initialize a git repo without any remote.
	cmd := exec.Command("git", "-C", dir, "init")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	got := DetectRepoURL(context.Background(), dir)
	if got != "" {
		t.Errorf("DetectRepoURL(no origin) = %q, want empty string", got)
	}
}
