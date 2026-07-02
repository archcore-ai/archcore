package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func callTool(handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) (*mcp.CallToolResult, error) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	return handler(context.Background(), req)
}

func TestHandleListDocuments_Empty(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	result, err := callTool(HandleListDocuments(base), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("unexpected error result")
	}

	var docs []LocalDocument
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &docs); err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0, got %d", len(docs))
	}
}

func TestHandleListDocuments_AllDocs(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n")
	writeDoc(t, base, "vision", "b.prd.md", "---\ntitle: B\nstatus: accepted\n---\n")

	result, err := callTool(HandleListDocuments(base), nil)
	if err != nil {
		t.Fatal(err)
	}

	var docs []LocalDocument
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &docs); err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2, got %d", len(docs))
	}
}

func TestHandleListDocuments_FilterByType(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n")
	writeDoc(t, base, "knowledge", "b.rfc.md", "---\ntitle: B\nstatus: draft\n---\n")

	result, err := callTool(HandleListDocuments(base), map[string]any{
		"types": []any{"adr"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var docs []LocalDocument
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &docs); err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1, got %d", len(docs))
	}
	if docs[0].Type != "adr" {
		t.Errorf("type = %q, want adr", docs[0].Type)
	}
}

func TestHandleListDocuments_FilterByCategory(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n")
	writeDoc(t, base, "vision", "b.prd.md", "---\ntitle: B\nstatus: draft\n---\n")

	result, err := callTool(HandleListDocuments(base), map[string]any{
		"category": "vision",
	})
	if err != nil {
		t.Fatal(err)
	}

	var docs []LocalDocument
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &docs); err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1, got %d", len(docs))
	}
	if docs[0].Category != "vision" {
		t.Errorf("category = %q, want vision", docs[0].Category)
	}
}

func TestHandleListDocuments_FilterByStatus(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n")
	writeDoc(t, base, "knowledge", "b.rfc.md", "---\ntitle: B\nstatus: accepted\n---\n")

	result, err := callTool(HandleListDocuments(base), map[string]any{
		"status": "accepted",
	})
	if err != nil {
		t.Fatal(err)
	}

	var docs []LocalDocument
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &docs); err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1, got %d", len(docs))
	}
	if docs[0].Status != "accepted" {
		t.Errorf("status = %q, want accepted", docs[0].Status)
	}
}

func TestHandleListDocuments_FilterByTags(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\ntags:\n  - frontend\n  - auth\n---\n")
	writeDoc(t, base, "knowledge", "b.rfc.md", "---\ntitle: B\nstatus: draft\ntags:\n  - backend\n---\n")
	writeDoc(t, base, "vision", "c.prd.md", "---\ntitle: C\nstatus: draft\n---\n")

	result, err := callTool(HandleListDocuments(base), map[string]any{
		"tags": []any{"frontend"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var docs []LocalDocument
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &docs); err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1, got %d", len(docs))
	}
	if docs[0].Type != "adr" {
		t.Errorf("type = %q, want adr", docs[0].Type)
	}

	// OR semantics: either tag matches.
	result2, err := callTool(HandleListDocuments(base), map[string]any{
		"tags": []any{"frontend", "backend"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var docs2 []LocalDocument
	if err := json.Unmarshal([]byte(result2.Content[0].(mcp.TextContent).Text), &docs2); err != nil {
		t.Fatal(err)
	}
	if len(docs2) != 2 {
		t.Fatalf("expected 2 (OR semantics), got %d", len(docs2))
	}
}

func TestHandleListDocuments_InvalidFilterTags(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, err := callTool(HandleListDocuments(base), map[string]any{
		"tags": []any{"INVALID"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for invalid filter tags")
	}
}

func TestHandleListDocuments_ScanError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}
	base := t.TempDir()
	archcoreDir := filepath.Join(base, ".archcore")
	subDir := filepath.Join(archcoreDir, "noperm")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(subDir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(subDir, 0o755) })

	result, err := callTool(HandleListDocuments(base), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected tool error when scan fails due to permissions")
	}
	msg := resultText(t, result)
	assertNoAbsPath(t, base, msg)
	if !strings.Contains(msg, "scanning documents: permission denied") {
		t.Errorf("message = %q, want sanitized scan error", msg)
	}
}

func TestHandleListDocuments_TagsAndTypeFilter(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\ntags:\n  - frontend\n---\n")
	writeDoc(t, base, "knowledge", "b.rfc.md", "---\ntitle: B\nstatus: draft\ntags:\n  - frontend\n---\n")

	result, err := callTool(HandleListDocuments(base), map[string]any{
		"types": []any{"adr"},
		"tags":  []any{"frontend"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var docs []LocalDocument
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &docs); err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 (type+tag filter), got %d", len(docs))
	}
	if docs[0].Type != "adr" {
		t.Errorf("type = %q, want adr", docs[0].Type)
	}
}
