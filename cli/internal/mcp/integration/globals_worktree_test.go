package integration

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"archcore-cli/internal/config"
	"archcore-cli/internal/testsupport"
)

// setupWorktreeOfPrimaryWithSiblingGlobal commits the sibling-global fixture and
// adds a linked worktree of it somewhere else on disk — the layout worktree
// tooling produces, where the worktree does not share the main checkout's parent
// directory. Returns the worktree base dir.
func setupWorktreeOfPrimaryWithSiblingGlobal(t *testing.T) string {
	t.Helper()
	testsupport.RequireGit(t)

	base := setupPrimaryWithSiblingGlobal(t)
	testsupport.NewGitRepo(t, base)
	testsupport.GitCommit(t, base, "initial")

	worktree := filepath.Join(t.TempDir(), "wt")
	testsupport.RunGit(t, base, "worktree", "add", "-b", "probe", worktree)
	return worktree
}

// TestGlobals_SurfacedFromAWorktree is issue #30 end to end: a session started
// inside a git worktree must see the same corpus as the main checkout. Before
// the escaping-path anchor, list_documents failed the whole scan here because
// "../company-global/.archcore" resolved next to the worktree, where nothing
// exists.
func TestGlobals_SurfacedFromAWorktree(t *testing.T) {
	base := setupWorktreeOfPrimaryWithSiblingGlobal(t)

	// Premise: anchored on the worktree the declared path resolves to nothing.
	onWorktree := config.ResolveGlobalPathFrom(base, "", "../company-global/.archcore")
	if _, err := os.Stat(onWorktree); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("premise broken: %q exists (stat err %v)", onWorktree, err)
	}

	c := newTestClient(t, base)
	res := mustCallTool(t, c, "list_documents", map[string]any{})
	docs := decodeListedDocs(t, res)
	if len(docs) != 3 {
		t.Fatalf("list_documents from a worktree: want 3 docs (1 local + 2 global), got %d: %+v", len(docs), docs)
	}

	var globals int
	for _, d := range docs {
		if d.SourceKind == "global" {
			globals++
			if !d.ReadOnly {
				t.Errorf("global doc %q is not read_only", d.Path)
			}
			if d.SourceID != "company" {
				t.Errorf("global doc %q source_id = %q, want %q", d.Path, d.SourceID, "company")
			}
		}
	}
	if globals != 2 {
		t.Errorf("global docs from a worktree = %d, want 2", globals)
	}
}
