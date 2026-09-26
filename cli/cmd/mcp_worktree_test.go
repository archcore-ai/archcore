package cmd

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"archcore-cli/internal/config"
	"archcore-cli/internal/docs"
	"archcore-cli/internal/testsupport"
)

// newWorktreeFixture builds the layout issue #30 describes: a main checkout
// declaring a sibling global source through a "../"-relative path, the global
// itself next to it, and a linked worktree elsewhere on disk. It returns the
// main checkout and the worktree.
func newWorktreeFixture(t *testing.T) (main, worktree string) {
	t.Helper()
	testsupport.RequireGit(t)

	parent := t.TempDir()
	main = filepath.Join(parent, "primary")
	if err := os.MkdirAll(main, 0o755); err != nil {
		t.Fatal(err)
	}
	writeMCPSettings(t, main,
		`{"sync":"none","globals":[{"id":"company","path":"../company/.archcore"}]}`)
	testsupport.WriteFile(t, filepath.Join(parent, "company", ".archcore"),
		"standards/naming.rule.md", "---\ntitle: \"Naming\"\nstatus: accepted\n---\n\n## Rule\n\n1. Name things.\n")

	testsupport.NewGitRepo(t, main)
	testsupport.GitCommit(t, main, "initial")

	worktree = filepath.Join(t.TempDir(), "wt")
	testsupport.RunGit(t, main, "worktree", "add", "-b", "probe", worktree)
	return main, worktree
}

// TestCheckGlobals_WorktreeResolvesFromMainCheckout is the startup half of
// issue #30: a worktree does not share the main checkout's parent directory, so
// a "../"-relative global resolves to nothing when anchored on the worktree.
// Anchoring an escaping path on the main checkout lets the server start.
func TestCheckGlobals_WorktreeResolvesFromMainCheckout(t *testing.T) {
	t.Parallel()
	main, worktree := newWorktreeFixture(t)

	// Premise of the case: anchored on the worktree the declared path resolves
	// to a directory that does not exist. Without it the test would pass for the
	// wrong reason.
	onWorktree := config.ResolveGlobalPathFrom(worktree, "", "../company/.archcore")
	if _, err := os.Stat(onWorktree); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("premise broken: %q exists (stat err %v)", onWorktree, err)
	}

	if err := checkGlobals(main); err != nil {
		t.Fatalf("checkGlobals in the main checkout: %v", err)
	}
	if err := checkGlobals(worktree); err != nil {
		t.Errorf("checkGlobals in a worktree = %v, want nil", err)
	}
	// A missing source no longer stops startup, so the state is what proves the
	// worktree found the source rather than skipped it.
	inspections, err := docs.InspectGlobals(worktree)
	if err != nil {
		t.Fatalf("InspectGlobals in a worktree: %v", err)
	}
	if len(inspections) != 1 || inspections[0].State != docs.GlobalOK {
		t.Errorf("worktree inspections = %+v, want one GlobalOK source", inspections)
	}
}
