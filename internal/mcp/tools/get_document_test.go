package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/sync"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleGetDocument_Success(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	content := "---\ntitle: My ADR\nstatus: accepted\n---\n\n## Context\nDetails here."
	writeDoc(t, base, "knowledge", "my-adr.adr.md", content)

	result, err := callTool(HandleGetDocument(base), map[string]any{
		"path": ".archcore/knowledge/my-adr.adr.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("unexpected error")
	}

	var doc LocalDocument
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Title != "My ADR" {
		t.Errorf("title = %q, want %q", doc.Title, "My ADR")
	}
	if doc.Content != content {
		t.Error("content mismatch")
	}
}

func TestHandleGetDocument_PathTraversal(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, err := callTool(HandleGetDocument(base), map[string]any{
		"path": ".archcore/../etc/passwd",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for path traversal")
	}
	// Must be rejected BY THE GUARD, not because the file happens to be absent
	// (a filesystem miss returns "document not found", which would still set
	// IsError and hide a removed guard). See test-oracle health check.
	if got := resultText(t, result); !strings.Contains(got, "invalid path") {
		t.Errorf("error = %q, want a path-guard rejection (\"invalid path: ...\")", got)
	}
}

func TestHandleGetDocument_PathTraversalNoLeak(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	// A real document that lives OUTSIDE .archcore/. Traversal must never reach it.
	secret := filepath.Join(base, "secret.adr.md")
	if err := os.WriteFile(secret, []byte("---\ntitle: Secret\nstatus: draft\n---\n\ntop-secret-body"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := callTool(HandleGetDocument(base), map[string]any{
		"path": ".archcore/../secret.adr.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("traversal to an existing out-of-tree file must be rejected, not read")
	}
	if strings.Contains(resultText(t, result), "top-secret-body") {
		t.Error("out-of-tree file content leaked through path traversal")
	}
}

func TestHandleGetDocument_InvalidPrefix(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, err := callTool(HandleGetDocument(base), map[string]any{
		"path": "etc/passwd",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for invalid prefix")
	}
	// Rejected by the prefix guard, not by an incidental filesystem miss.
	if got := resultText(t, result); !strings.Contains(got, "invalid path") {
		t.Errorf("error = %q, want a path-guard rejection (\"invalid path: ...\")", got)
	}
}

func TestHandleGetDocument_NotFound(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, err := callTool(HandleGetDocument(base), map[string]any{
		"path": ".archcore/knowledge/nonexistent.adr.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing file")
	}
}

func TestHandleGetDocument_MissingPath(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, err := callTool(HandleGetDocument(base), map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing path")
	}
}

func TestHandleGetDocument_WithRelations(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "source.adr.md", "---\ntitle: Source\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "vision", "target.prd.md", "---\ntitle: Target\nstatus: draft\n---\n\nbody")

	// Add a relation via manifest.
	m := sync.NewManifest()
	m.AddRelation("knowledge/source.adr.md", "vision/target.prd.md", sync.RelImplements)
	if err := sync.SaveManifest(base, m); err != nil {
		t.Fatal(err)
	}

	// Check source doc has outgoing.
	result, err := callTool(HandleGetDocument(base), map[string]any{
		"path": ".archcore/knowledge/source.adr.md",
	})
	if err != nil {
		t.Fatal(err)
	}

	var enriched EnrichedDocument
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &enriched); err != nil {
		t.Fatal(err)
	}
	if len(enriched.OutgoingRelations) != 1 {
		t.Fatalf("expected 1 outgoing, got %d", len(enriched.OutgoingRelations))
	}
	if enriched.OutgoingRelations[0].Path != ".archcore/vision/target.prd.md" {
		t.Errorf("outgoing path = %q", enriched.OutgoingRelations[0].Path)
	}
	if enriched.OutgoingRelations[0].Type != "implements" {
		t.Errorf("outgoing type = %q", enriched.OutgoingRelations[0].Type)
	}

	// Check target doc has incoming.
	result2, _ := callTool(HandleGetDocument(base), map[string]any{
		"path": ".archcore/vision/target.prd.md",
	})
	var enriched2 EnrichedDocument
	if err := json.Unmarshal([]byte(result2.Content[0].(mcp.TextContent).Text), &enriched2); err != nil {
		t.Fatal(err)
	}
	if len(enriched2.IncomingRelations) != 1 {
		t.Fatalf("expected 1 incoming, got %d", len(enriched2.IncomingRelations))
	}
	if enriched2.IncomingRelations[0].Path != ".archcore/knowledge/source.adr.md" {
		t.Errorf("incoming path = %q", enriched2.IncomingRelations[0].Path)
	}
}

func TestHandleGetDocument_NoRelations(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "solo.adr.md", "---\ntitle: Solo\nstatus: draft\n---\n\nbody")

	result, _ := callTool(HandleGetDocument(base), map[string]any{
		"path": ".archcore/knowledge/solo.adr.md",
	})

	var enriched EnrichedDocument
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &enriched); err != nil {
		t.Fatal(err)
	}
	if len(enriched.OutgoingRelations) != 0 {
		t.Errorf("expected 0 outgoing, got %d", len(enriched.OutgoingRelations))
	}
	if len(enriched.IncomingRelations) != 0 {
		t.Errorf("expected 0 incoming, got %d", len(enriched.IncomingRelations))
	}
}

func TestHandleGetDocument_NoManifest(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "no-manifest.adr.md", "---\ntitle: No Manifest\nstatus: draft\n---\n\nbody")

	result, err := callTool(HandleGetDocument(base), map[string]any{
		"path": ".archcore/knowledge/no-manifest.adr.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("unexpected error")
	}

	// Should still return the doc without relations.
	var enriched EnrichedDocument
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &enriched); err != nil {
		t.Fatal(err)
	}
	if enriched.Title != "No Manifest" {
		t.Errorf("title = %q", enriched.Title)
	}
}
