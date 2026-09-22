package git

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"archcore-cli/internal/testsupport"
)

// addWorktree creates a linked worktree of repo at dir on a new branch.
func addWorktree(t *testing.T, repo, dir, branch string) string {
	t.Helper()
	runGit(t, repo, "worktree", "add", "-b", branch, dir)
	return dir
}

func TestMainCheckout(t *testing.T) {
	requireGit(t)

	tests := []struct {
		name string
		// setup returns the directory to query and the expected checkout. An
		// empty want means the query must fail.
		setup func(t *testing.T) (dir, want string)
	}{
		{
			name: "main checkout answers itself",
			setup: func(t *testing.T) (string, string) {
				repo := initRepo(t, map[string]string{"a.txt": "a"})
				return repo, repo
			},
		},
		{
			name: "linked worktree answers the main checkout",
			setup: func(t *testing.T) (string, string) {
				repo := initRepo(t, map[string]string{"a.txt": "a"})
				wt := addWorktree(t, repo, filepath.Join(t.TempDir(), "wt"), "probe")
				return wt, repo
			},
		},
		{
			name: "subdirectory of a worktree answers the main checkout",
			setup: func(t *testing.T) (string, string) {
				repo := initRepo(t, map[string]string{"a.txt": "a"})
				wt := addWorktree(t, repo, filepath.Join(t.TempDir(), "wt"), "probe")
				sub := filepath.Join(wt, "nested")
				testsupport.WriteFile(t, sub, "b.txt", "b")
				return sub, repo
			},
		},
		{
			name: "non-git directory fails",
			setup: func(t *testing.T) (string, string) {
				return t.TempDir(), ""
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, want := tt.setup(t)
			got, err := mainCheckout(context.Background(), dir)
			if want == "" {
				if err == nil {
					t.Fatalf("mainCheckout(%q) = %q, want an error", dir, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("mainCheckout(%q): %v", dir, err)
			}
			if resolved(t, got) != resolved(t, want) {
				t.Errorf("mainCheckout(%q) = %q, want %q", dir, got, want)
			}
		})
	}
}

// TestMainCheckout_Submodule pins the answer git gives inside a submodule: the
// reported worktree is the module directory under the superproject's .git, not
// the checkout. Callers must validate the path rather than trust it.
func TestMainCheckout_Submodule(t *testing.T) {
	requireGit(t)
	sub := initRepo(t, map[string]string{"s.txt": "s"})
	super := initRepo(t, map[string]string{"top.txt": "top"})
	runGit(t, super, "-c", "protocol.file.allow=always", "submodule", "add", sub, "sub")
	testsupport.GitCommit(t, super, "add submodule")

	got, err := mainCheckout(context.Background(), filepath.Join(super, "sub"))
	if err != nil {
		t.Fatalf("mainCheckout inside submodule: %v", err)
	}
	if got == filepath.Join(super, "sub") {
		t.Fatalf("mainCheckout inside submodule = %q; the test's premise no longer holds", got)
	}
}

func TestMainCheckout_GitAbsent(t *testing.T) {
	original := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("not found") }
	t.Cleanup(func() { lookPath = original })

	if _, err := mainCheckout(context.Background(), t.TempDir()); !errors.Is(err, ErrGitAbsent) {
		t.Errorf("mainCheckout without git: got %v, want ErrGitAbsent", err)
	}
}

// resolved evaluates symlinks so a comparison survives macOS /tmp -> /private/tmp.
func resolved(t *testing.T, dir string) string {
	t.Helper()
	out, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", dir, err)
	}
	return out
}

// TestWorktreeRoots covers the pair config.deriveAnchor actually calls: inside a
// linked worktree the two answers must differ, and inside the main checkout they
// must agree. Only mainCheckout was pinned before, so nothing held the pairing.
func TestWorktreeRoots(t *testing.T) {
	requireGit(t)

	tests := []struct {
		name string
		// setup returns the directory to query, and whether Current and Main are
		// expected to name the same tree.
		setup func(t *testing.T) (dir string, wantSame bool)
	}{
		{
			name: "main checkout reports one tree twice",
			setup: func(t *testing.T) (string, bool) {
				return initRepo(t, map[string]string{"a.txt": "a"}), true
			},
		},
		{
			name: "linked worktree reports both trees",
			setup: func(t *testing.T) (string, bool) {
				repo := initRepo(t, map[string]string{"a.txt": "a"})
				return addWorktree(t, repo, filepath.Join(t.TempDir(), "wt"), "probe"), false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, wantSame := tt.setup(t)
			roots, err := WorktreeRoots(context.Background(), dir)
			if err != nil {
				t.Fatalf("WorktreeRoots(%q): %v", dir, err)
			}
			if roots.Current == "" || roots.Main == "" {
				t.Fatalf("WorktreeRoots(%q) = %+v, want both trees named", dir, roots)
			}
			if same := resolved(t, roots.Current) == resolved(t, roots.Main); same != wantSame {
				t.Errorf("WorktreeRoots(%q) = %+v; Current == Main is %v, want %v", dir, roots, same, wantSame)
			}
		})
	}
}

// TestWorktreeRoots_Bare pins the state deriveAnchor falls back from: a bare
// repository has history but no checkout, so the pair must fail rather than
// hand back a directory that cannot anchor a relative path.
func TestWorktreeRoots_Bare(t *testing.T) {
	requireGit(t)
	bare := filepath.Join(t.TempDir(), "b.git")
	runGit(t, t.TempDir(), "init", "--bare", "-q", bare)

	if roots, err := WorktreeRoots(context.Background(), bare); err == nil {
		t.Errorf("WorktreeRoots on a bare repository = %+v, want an error", roots)
	}
}

func TestWorktreeRoots_NotARepository(t *testing.T) {
	requireGit(t)
	if roots, err := WorktreeRoots(context.Background(), t.TempDir()); err == nil {
		t.Errorf("WorktreeRoots outside a repository = %+v, want an error", roots)
	}
}

// TestToplevel covers the half of the pair that answers about the caller's own
// tree: inside a linked worktree it must name that worktree, not the checkout
// `git worktree add` was run from.
func TestToplevel(t *testing.T) {
	requireGit(t)
	repo := initRepo(t, map[string]string{"a.txt": "a"})
	wt := addWorktree(t, repo, filepath.Join(t.TempDir(), "wt"), "probe")

	got, err := toplevel(context.Background(), wt)
	if err != nil {
		t.Fatalf("toplevel(%q): %v", wt, err)
	}
	if resolved(t, got) != resolved(t, wt) {
		t.Errorf("toplevel(%q) = %q, want the linked worktree itself", wt, got)
	}
}
