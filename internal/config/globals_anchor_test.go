package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"archcore-cli/internal/git"
)

// withWorktreeRoots replaces the working-tree lookup and clears the memo, so a
// case drives resolution without building a git repository. Cases that use it
// cannot run in parallel: the seam and the memo are process-wide.
func withWorktreeRoots(t *testing.T, fn func(context.Context, string) (git.Roots, error)) {
	t.Helper()
	original := lookupWorktreeRoots
	lookupWorktreeRoots = fn
	resetMainCheckoutCache()
	t.Cleanup(func() {
		lookupWorktreeRoots = original
		resetMainCheckoutCache()
	})
}

// realDir evaluates symlinks the way the resolver does, so an expectation built
// from t.TempDir() (/var on macOS) matches an anchor git reports (/private/var).
func realDir(t *testing.T, dir string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", dir, err)
	}
	return resolved
}

// rootsFn answers every query with the same pair.
func rootsFn(current, main string) func(context.Context, string) (git.Roots, error) {
	return func(context.Context, string) (git.Roots, error) {
		return git.Roots{Current: current, Main: main}, nil
	}
}

func resetMainCheckoutCache() {
	mainCheckoutMu.Lock()
	defer mainCheckoutMu.Unlock()
	mainCheckoutCache = map[string]string{}
}

// newProject creates a directory holding an .archcore/ directory.
func newProject(t *testing.T, name string) string {
	t.Helper()
	return newProjectAt(t, filepath.Join(t.TempDir(), name))
}

// newProjectAt creates dir with an .archcore/ directory inside it.
func newProjectAt(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, dirName), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestResolveGlobalPathFrom(t *testing.T) {
	t.Parallel()
	base := filepath.FromSlash("/home/u/repo/.claude/worktrees/wt")
	main := filepath.FromSlash("/home/u/repo")
	tests := []struct {
		name, path, mainCheckout, want string
	}{
		{
			name: "escaping path anchors at the main checkout",
			path: "../global/.archcore", mainCheckout: main,
			want: filepath.FromSlash("/home/u/global/.archcore"),
		},
		{
			name: "in-tree path stays on the project root",
			path: ".archcore/global/company", mainCheckout: main,
			want: filepath.FromSlash("/home/u/repo/.claude/worktrees/wt/.archcore/global/company"),
		},
		{
			name: "absolute path is untouched",
			path: filepath.FromSlash("/elsewhere/.archcore"), mainCheckout: main,
			want: filepath.FromSlash("/elsewhere/.archcore"),
		},
		{
			name: "no main checkout falls back to the project root",
			path: "../global/.archcore", mainCheckout: "",
			want: filepath.FromSlash("/home/u/repo/.claude/worktrees/global/.archcore"),
		},
		{
			name: "main checkout equal to the project root changes nothing",
			path: "../global/.archcore", mainCheckout: base,
			want: filepath.FromSlash("/home/u/repo/.claude/worktrees/global/.archcore"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveGlobalPathFrom(base, tt.mainCheckout, tt.path); got != tt.want {
				t.Errorf("ResolveGlobalPathFrom(%q, %q, %q) = %q, want %q",
					base, tt.mainCheckout, tt.path, got, tt.want)
			}
		})
	}
}

func TestResolveGlobalPath_AnchorLookup(t *testing.T) {
	tests := []struct {
		name string
		// setup returns the project root, the lookup answer, the declared path,
		// and the expected resolution.
		setup func(t *testing.T) (base string, lookup func(context.Context, string) (git.Roots, error), path, want string)
	}{
		{
			name: "worktree resolves an escaping path against the main checkout",
			setup: func(t *testing.T) (string, func(context.Context, string) (git.Roots, error), string, string) {
				main := newProject(t, "main")
				wt := newProject(t, "wt")
				return wt, rootsFn(wt, main),
					"../global/.archcore",
					filepath.Join(realDir(t, filepath.Dir(main)), "global", ".archcore")
			},
		},
		{
			name: "a project below the working tree root keeps its own anchor",
			setup: func(t *testing.T) (string, func(context.Context, string) (git.Roots, error), string, string) {
				repo := t.TempDir()
				nested := newProjectAt(t, filepath.Join(repo, "examples", "05"))
				return nested, rootsFn(repo, repo),
					"../_global_/company/.archcore",
					filepath.Join(repo, "examples", "_global_", "company", ".archcore")
			},
		},
		{
			name: "a nested project in a worktree maps onto the same position",
			setup: func(t *testing.T) (string, func(context.Context, string) (git.Roots, error), string, string) {
				parent := t.TempDir()
				mainRepo := filepath.Join(parent, "main")
				newProjectAt(t, filepath.Join(mainRepo, "examples", "05"))
				wtRepo := filepath.Join(parent, "wt")
				wtNested := newProjectAt(t, filepath.Join(wtRepo, "examples", "05"))
				return wtNested, rootsFn(wtRepo, mainRepo),
					"../_global_/company/.archcore",
					filepath.Join(realDir(t, mainRepo), "examples", "_global_", "company", ".archcore")
			},
		},
		{
			name: "an anchor without .archcore is rejected",
			setup: func(t *testing.T) (string, func(context.Context, string) (git.Roots, error), string, string) {
				wt := newProject(t, "wt")
				modules := t.TempDir() // stands in for <super>/.git/modules/<name>
				return wt, rootsFn(wt, modules),
					"../global/.archcore",
					filepath.Join(filepath.Dir(wt), "global", ".archcore")
			},
		},
		{
			// The third fallback the ADR enumerates: filepath.Rel yields a "../"
			// answer, so the project root is not inside the tree git reported and
			// no position can be mapped onto the main checkout.
			name: "a project root outside the reported worktree is rejected",
			setup: func(t *testing.T) (string, func(context.Context, string) (git.Roots, error), string, string) {
				elsewhere := newProject(t, "elsewhere")
				current := newProject(t, "current")
				main := newProject(t, "main")
				return elsewhere, rootsFn(current, main),
					"../global/.archcore",
					filepath.Join(filepath.Dir(elsewhere), "global", ".archcore")
			},
		},
		{
			name: "a failed lookup falls back to the project root",
			setup: func(t *testing.T) (string, func(context.Context, string) (git.Roots, error), string, string) {
				wt := newProject(t, "wt")
				return wt, func(context.Context, string) (git.Roots, error) {
						return git.Roots{}, errors.New("not a repository")
					},
					"../global/.archcore",
					filepath.Join(filepath.Dir(wt), "global", ".archcore")
			},
		},
		{
			name: "an in-tree path never consults the lookup",
			setup: func(t *testing.T) (string, func(context.Context, string) (git.Roots, error), string, string) {
				wt := newProject(t, "wt")
				return wt, func(context.Context, string) (git.Roots, error) {
						t.Error("lookup called for an in-tree path")
						return git.Roots{}, nil
					},
					".archcore/global/company",
					filepath.Join(wt, ".archcore", "global", "company")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, lookup, path, want := tt.setup(t)
			withWorktreeRoots(t, lookup)
			if got := ResolveGlobalPath(base, path); got != want {
				t.Errorf("ResolveGlobalPath(%q, %q) = %q, want %q", base, path, got, want)
			}
		})
	}
}

// TestResolveGlobalPath_LookupMemoized pins that the git query runs once per
// project root: the MCP server resolves global paths on every read.
func TestResolveGlobalPath_LookupMemoized(t *testing.T) {
	main := newProject(t, "main")
	wt := newProject(t, "wt")
	calls := 0
	withWorktreeRoots(t, func(context.Context, string) (git.Roots, error) {
		calls++
		return git.Roots{Current: wt, Main: main}, nil
	})

	for range 3 {
		ResolveGlobalPath(wt, "../global/.archcore")
	}
	if calls != 1 {
		t.Errorf("lookup ran %d times, want 1", calls)
	}
}
