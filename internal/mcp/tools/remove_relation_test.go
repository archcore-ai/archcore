package tools

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleRemoveRelation_Success(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "vision", "b.prd.md", "---\ntitle: B\nstatus: draft\n---\n\nbody")

	// First add.
	callTool(HandleAddRelation(StaticRoot(base)), map[string]any{
		"source": "knowledge/a.adr.md",
		"target": "vision/b.prd.md",
		"type":   "implements",
	})

	// Then remove.
	result, err := callTool(HandleRemoveRelation(StaticRoot(base)), map[string]any{
		"source": "knowledge/a.adr.md",
		"target": "vision/b.prd.md",
		"type":   "implements",
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
	if resp["removed"] != true {
		t.Error("expected removed=true")
	}
	if resp["source"] != "knowledge/a.adr.md" {
		t.Errorf("source = %q, want %q", resp["source"], "knowledge/a.adr.md")
	}
	if resp["target"] != "vision/b.prd.md" {
		t.Errorf("target = %q, want %q", resp["target"], "vision/b.prd.md")
	}
	if resp["type"] != "implements" {
		t.Errorf("type = %q, want %q", resp["type"], "implements")
	}
}

func TestHandleRemoveRelation_NotFound(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, _ := callTool(HandleRemoveRelation(StaticRoot(base)), map[string]any{
		"source": "a.adr.md",
		"target": "b.prd.md",
		"type":   "related",
	})
	if result.IsError {
		t.Fatal("should not error, just return removed=false")
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["removed"] != false {
		t.Error("expected removed=false")
	}
	if resp["source"] != "a.adr.md" {
		t.Errorf("source = %q, want %q", resp["source"], "a.adr.md")
	}
	if resp["target"] != "b.prd.md" {
		t.Errorf("target = %q, want %q", resp["target"], "b.prd.md")
	}
	if resp["type"] != "related" {
		t.Errorf("type = %q, want %q", resp["type"], "related")
	}
}

func TestHandleRemoveRelation_InvalidType(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, _ := callTool(HandleRemoveRelation(StaticRoot(base)), map[string]any{
		"source": "a.adr.md",
		"target": "b.prd.md",
		"type":   "blocks",
	})
	if !result.IsError {
		t.Error("expected error for invalid type")
	}
}

func TestHandleRemoveRelation_PathTraversal(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, _ := callTool(HandleRemoveRelation(StaticRoot(base)), map[string]any{
		"source": "../etc/passwd",
		"target": "b.prd.md",
		"type":   "related",
	})
	if !result.IsError {
		t.Error("expected error for path traversal")
	}
}
