package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"archcore-cli/internal/sync"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleAddRelation_Success(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "vision", "b.prd.md", "---\ntitle: B\nstatus: draft\n---\n\nbody")

	result, err := callTool(HandleAddRelation(StaticRoot(base)), map[string]any{
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
	if resp["added"] != true {
		t.Error("expected added=true")
	}

	// Verify manifest on disk.
	m, _ := sync.LoadManifest(base)
	if len(m.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(m.Relations))
	}
}

func TestHandleAddRelation_Duplicate(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "vision", "b.prd.md", "---\ntitle: B\nstatus: draft\n---\n\nbody")

	callTool(HandleAddRelation(StaticRoot(base)), map[string]any{
		"source": "knowledge/a.adr.md",
		"target": "vision/b.prd.md",
		"type":   "implements",
	})

	result, _ := callTool(HandleAddRelation(StaticRoot(base)), map[string]any{
		"source": "knowledge/a.adr.md",
		"target": "vision/b.prd.md",
		"type":   "implements",
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["added"] != false {
		t.Error("expected added=false for duplicate")
	}
}

func TestHandleAddRelation_InvalidType(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, _ := callTool(HandleAddRelation(StaticRoot(base)), map[string]any{
		"source": "a.adr.md",
		"target": "b.prd.md",
		"type":   "blocks",
	})
	if !result.IsError {
		t.Error("expected error for invalid type")
	}
}

func TestHandleAddRelation_SourceEqualsTarget(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, _ := callTool(HandleAddRelation(StaticRoot(base)), map[string]any{
		"source": "a.adr.md",
		"target": "a.adr.md",
		"type":   "related",
	})
	if !result.IsError {
		t.Error("expected error when source equals target")
	}
}

func TestHandleAddRelation_SourceNotFound(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "vision", "b.prd.md", "---\ntitle: B\nstatus: draft\n---\n\nbody")

	result, _ := callTool(HandleAddRelation(StaticRoot(base)), map[string]any{
		"source": "nonexistent.adr.md",
		"target": "vision/b.prd.md",
		"type":   "related",
	})
	if !result.IsError {
		t.Error("expected error for missing source")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "source document not found") {
		t.Errorf("expected 'source document not found', got: %s", text)
	}
}

func TestHandleAddRelation_TargetNotFound(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n\nbody")

	result, _ := callTool(HandleAddRelation(StaticRoot(base)), map[string]any{
		"source": "knowledge/a.adr.md",
		"target": "nonexistent.prd.md",
		"type":   "related",
	})
	if !result.IsError {
		t.Error("expected error for missing target")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "target document not found") {
		t.Errorf("expected 'target document not found', got: %s", text)
	}
}

func TestHandleAddRelation_PathTraversal(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, _ := callTool(HandleAddRelation(StaticRoot(base)), map[string]any{
		"source": "../etc/passwd",
		"target": "b.prd.md",
		"type":   "related",
	})
	if !result.IsError {
		t.Error("expected error for path traversal")
	}
	// Rejected by the ".." guard specifically — not because the doc is missing.
	if got := resultText(t, result); !strings.Contains(got, "must not contain") {
		t.Errorf("error = %q, want the \"..\" path guard rejection", got)
	}
}

func TestHandleAddRelation_NormalizesPrefix(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "vision", "b.prd.md", "---\ntitle: B\nstatus: draft\n---\n\nbody")

	result, err := callTool(HandleAddRelation(StaticRoot(base)), map[string]any{
		"source": ".archcore/knowledge/a.adr.md",
		"target": ".archcore/vision/b.prd.md",
		"type":   "related",
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
	if resp["source"] != "knowledge/a.adr.md" {
		t.Errorf("source = %q, want normalized", resp["source"])
	}
}

func TestHandleAddRelation_AllTypes(t *testing.T) {
	t.Parallel()
	for _, rt := range []string{"related", "implements", "extends", "depends_on"} {
		t.Run(rt, func(t *testing.T) {
			t.Parallel()
			base := setupTestArchcore(t)
			writeDoc(t, base, "", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n\nbody")
			writeDoc(t, base, "", "b.prd.md", "---\ntitle: B\nstatus: draft\n---\n\nbody")

			result, err := callTool(HandleAddRelation(StaticRoot(base)), map[string]any{
				"source": "a.adr.md",
				"target": "b.prd.md",
				"type":   rt,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatalf("unexpected error for type %s: %s", rt, result.Content[0].(mcp.TextContent).Text)
			}
		})
	}
}
