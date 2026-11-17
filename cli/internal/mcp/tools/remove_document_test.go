package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	gosync "sync"
	"sync/atomic"
	"testing"

	"archcore-cli/internal/sync"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleRemoveDocument_Success(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", "---\ntitle: My ADR\nstatus: draft\n---\n\nbody")

	result, err := callTool(HandleRemoveDocument(StaticRoot(base)), map[string]any{
		"path": ".archcore/knowledge/my-adr.adr.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["path"] != ".archcore/knowledge/my-adr.adr.md" {
		t.Errorf("path = %q", resp["path"])
	}
	if resp["title"] != "My ADR" {
		t.Errorf("title = %q", resp["title"])
	}
	if resp["type"] != "adr" {
		t.Errorf("type = %q", resp["type"])
	}
	if resp["category"] != "knowledge" {
		t.Errorf("category = %q", resp["category"])
	}
	if resp["relations_removed"] != float64(0) {
		t.Errorf("relations_removed = %v", resp["relations_removed"])
	}

	// File should be gone.
	if _, err := os.Stat(filepath.Join(base, ".archcore", "knowledge", "my-adr.adr.md")); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestHandleRemoveDocument_WithRelationCleanup(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "", "b.prd.md", "---\ntitle: B\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "", "c.plan.md", "---\ntitle: C\nstatus: draft\n---\n\nbody")

	// Add relations: A→B and B→C.
	m := sync.NewManifest()
	m.AddRelation("a.adr.md", "b.prd.md", sync.RelImplements)
	m.AddRelation("b.prd.md", "c.plan.md", sync.RelDependsOn)
	if err := sync.SaveManifest(base, m); err != nil {
		t.Fatal(err)
	}

	// Remove A — should clean up A→B, but B→C survives.
	result, err := callTool(HandleRemoveDocument(StaticRoot(base)), map[string]any{
		"path": ".archcore/a.adr.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["relations_removed"] != float64(1) {
		t.Errorf("relations_removed = %v, want 1", resp["relations_removed"])
	}

	// Verify manifest.
	m2, err := sync.LoadManifest(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(m2.Relations) != 1 {
		t.Fatalf("expected 1 surviving relation, got %d", len(m2.Relations))
	}
	if m2.Relations[0].Source != "b.prd.md" || m2.Relations[0].Target != "c.plan.md" {
		t.Errorf("wrong surviving relation: %+v", m2.Relations[0])
	}
}

func TestHandleRemoveDocument_FileNotFound(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, err := callTool(HandleRemoveDocument(StaticRoot(base)), map[string]any{
		"path": ".archcore/knowledge/nonexistent.adr.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for nonexistent file")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "document not found") {
		t.Errorf("error = %q, want 'document not found'", text)
	}
}

func TestHandleRemoveDocument_InvalidPrefix(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, err := callTool(HandleRemoveDocument(StaticRoot(base)), map[string]any{
		"path": "src/main.go",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for non-.archcore path")
	}
}

func TestHandleRemoveDocument_PathTraversal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{"dotdot prefix", "../etc/passwd"},
		{"dotdot in archcore", ".archcore/../etc/passwd"},
		{"absolute path", "/etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupTestArchcore(t)

			result, err := callTool(HandleRemoveDocument(StaticRoot(base)), map[string]any{
				"path": tt.path,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !result.IsError {
				t.Error("expected error for path traversal")
			}
		})
	}
}

func TestHandleRemoveDocument_MissingPath(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, err := callTool(HandleRemoveDocument(StaticRoot(base)), map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing path")
	}
}

func TestHandleRemoveDocument_NoManifest(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "my-adr.adr.md", "---\ntitle: My ADR\nstatus: draft\n---\n\nbody")

	// No manifest file exists — should still succeed.
	result, err := callTool(HandleRemoveDocument(StaticRoot(base)), map[string]any{
		"path": ".archcore/knowledge/my-adr.adr.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["relations_removed"] != float64(0) {
		t.Errorf("relations_removed = %v, want 0", resp["relations_removed"])
	}

	// File should be gone.
	if _, err := os.Stat(filepath.Join(base, ".archcore", "knowledge", "my-adr.adr.md")); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
	if _, err := os.Stat(filepath.Join(base, ".archcore", sync.ManifestFile)); !os.IsNotExist(err) {
		t.Error("removal without relation changes wrote a manifest")
	}
}

func TestHandleRemoveDocument_BothDirectionRelations(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "", "b.prd.md", "---\ntitle: B\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "", "c.plan.md", "---\ntitle: C\nstatus: draft\n---\n\nbody")

	// A→B (outgoing from B's perspective: no), B→C (outgoing from B), C→B would be incoming to B.
	// Make B both a source and a target: A→B and B→C.
	m := sync.NewManifest()
	m.AddRelation("a.adr.md", "b.prd.md", sync.RelRelated)  // B is target
	m.AddRelation("b.prd.md", "c.plan.md", sync.RelExtends) // B is source
	if err := sync.SaveManifest(base, m); err != nil {
		t.Fatal(err)
	}

	// Remove B — both relations should be cleaned up.
	result, err := callTool(HandleRemoveDocument(StaticRoot(base)), map[string]any{
		"path": ".archcore/b.prd.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["relations_removed"] != float64(2) {
		t.Errorf("relations_removed = %v, want 2", resp["relations_removed"])
	}

	// Verify manifest has no relations left.
	m2, err := sync.LoadManifest(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(m2.Relations) != 0 {
		t.Errorf("expected 0 relations, got %d", len(m2.Relations))
	}
}

// TestHandleRemoveDocument_StaleRelationsNotCounted pins relations_removed
// semantics: only the deleted document's own edges are counted; a pre-existing
// stale edge (both endpoints already missing) is still cleaned opportunistically
// but does not inflate this deletion's count.
func TestHandleRemoveDocument_StaleRelationsNotCounted(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "", "b.prd.md", "---\ntitle: B\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "", "c.plan.md", "---\ntitle: C\nstatus: draft\n---\n\nbody")

	m := sync.NewManifest()
	m.AddRelation("a.adr.md", "b.prd.md", sync.RelImplements) // outgoing edge of A — counted
	m.AddRelation("c.plan.md", "a.adr.md", sync.RelRelated)   // incoming edge of A — counted
	m.AddRelation("x.adr.md", "y.adr.md", sync.RelRelated)    // stale: both endpoints missing
	if err := sync.SaveManifest(base, m); err != nil {
		t.Fatal(err)
	}

	result, err := callTool(HandleRemoveDocument(StaticRoot(base)), map[string]any{
		"path": ".archcore/a.adr.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["relations_removed"] != float64(2) {
		t.Errorf("relations_removed = %v, want 2 (stale edge must not be counted)", resp["relations_removed"])
	}

	m2, err := sync.LoadManifest(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(m2.Relations) != 0 {
		t.Errorf("expected all relations cleaned (incl. the stale edge), got %d", len(m2.Relations))
	}
}

// TestHandleRemoveDocument_NotResurrectedByAParallelUpdate pins the shared
// document lock: an update that read the file before a remove deleted it must
// not rename its edit back into place after the remove reported success.
func TestHandleRemoveDocument_NotResurrectedByAParallelUpdate(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	remove := HandleRemoveDocument(StaticRoot(base))
	update := HandleUpdateDocument(StaticRoot(base))
	const path = ".archcore/knowledge/race.adr.md"
	for round := range 40 {
		writeDoc(t, base, "knowledge", "race.adr.md", "---\ntitle: T\nstatus: draft\n---\n\nbody\n")
		var removed atomic.Bool
		var wg gosync.WaitGroup
		for i := range 4 {
			wg.Go(func() {
				_, _ = callTool(update, map[string]any{"path": path, "title": fmt.Sprintf("T%d-%d", round, i)})
			})
		}
		wg.Go(func() {
			result, err := callTool(remove, map[string]any{"path": path})
			removed.Store(err == nil && !result.IsError)
		})
		wg.Wait()
		if _, err := os.Stat(filepath.Join(base, path)); removed.Load() && err == nil {
			t.Fatalf("round %d: the document exists after remove_document succeeded", round)
		}
	}
}

// add_relation reads both documents before it takes the manifest lock. A
// remove_document that lands in that window must win: the edge would otherwise
// be added right behind the cleanup and dangle from its first second.
func TestHandleAddRelation_RefusesADocumentRemovedBeforeTheLock(t *testing.T) {
	// Not parallel: it reassigns the beforeRelationMutate seam.
	base := setupTestArchcore(t)
	writeDoc(t, base, "", "keep.adr.md", "---\ntitle: K\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "", "gone.adr.md", "---\ntitle: G\nstatus: draft\n---\n\nbody")
	remove := HandleRemoveDocument(StaticRoot(base))
	orig := beforeRelationMutate
	t.Cleanup(func() { beforeRelationMutate = orig })
	beforeRelationMutate = func() {
		beforeRelationMutate = orig // remove_document hits the same seam
		if result, err := callTool(remove, map[string]any{"path": ".archcore/gone.adr.md"}); err != nil || result.IsError {
			t.Errorf("remove inside the window failed: %v %+v", err, result)
		}
	}

	result, err := callTool(HandleAddRelation(StaticRoot(base)), map[string]any{"source": "keep.adr.md", "target": "gone.adr.md", "type": "related"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatalf("add_relation accepted an edge to a document removed before the lock: %+v", result)
	}
	m, err := sync.LoadManifest(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Relations) != 0 {
		t.Fatalf("a dangling relation was written: %+v", m.Relations)
	}
}

// remove_document stats the endpoints from a snapshot taken before the lock.
// An add_relation that lands in that window must still lose its edge: the
// document itself is marked missing by hand, not by the stat.
func TestHandleRemoveDocument_DropsARelationAddedBeforeTheLock(t *testing.T) {
	// Not parallel: it reassigns the beforeRelationMutate seam.
	base := setupTestArchcore(t)
	writeDoc(t, base, "", "keep.adr.md", "---\ntitle: K\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "", "gone.adr.md", "---\ntitle: G\nstatus: draft\n---\n\nbody")
	orig := beforeRelationMutate
	t.Cleanup(func() { beforeRelationMutate = orig })
	beforeRelationMutate = func() {
		beforeRelationMutate = orig
		if err := sharedManifestStore.mutate(base, func(m *sync.Manifest) bool {
			return m.AddRelation("keep.adr.md", "gone.adr.md", sync.RelRelated)
		}); err != nil {
			t.Fatal(err)
		}
	}

	result, err := callTool(HandleRemoveDocument(StaticRoot(base)), map[string]any{"path": ".archcore/gone.adr.md"})
	if err != nil || result.IsError {
		t.Fatalf("remove_document failed: %v %+v", err, result)
	}
	m, err := sync.LoadManifest(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Relations) != 0 {
		t.Fatalf("a relation added before the lock survived the removal: %+v", m.Relations)
	}
}
