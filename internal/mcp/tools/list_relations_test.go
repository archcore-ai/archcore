package tools

import (
	"encoding/json"
	"testing"

	"archcore-cli/internal/sync"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleListRelations_Empty(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, err := callTool(HandleListRelations(StaticRoot(base)), map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("unexpected error")
	}

	var resp map[string][]sync.Relation
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp["relations"]) != 0 {
		t.Errorf("expected 0 relations, got %d", len(resp["relations"]))
	}
}

func TestHandleListRelations_All(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "", "b.prd.md", "---\ntitle: B\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "", "c.rfc.md", "---\ntitle: C\nstatus: draft\n---\n\nbody")

	callTool(HandleAddRelation(StaticRoot(base)), map[string]any{"source": "a.adr.md", "target": "b.prd.md", "type": "implements"})
	callTool(HandleAddRelation(StaticRoot(base)), map[string]any{"source": "b.prd.md", "target": "c.rfc.md", "type": "related"})
	callTool(HandleAddRelation(StaticRoot(base)), map[string]any{"source": "a.adr.md", "target": "c.rfc.md", "type": "depends_on"})

	result, _ := callTool(HandleListRelations(StaticRoot(base)), map[string]any{})

	var resp map[string][]sync.Relation
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp["relations"]) != 3 {
		t.Errorf("expected 3 relations, got %d", len(resp["relations"]))
	}
}

func TestHandleListRelations_FilterByPath(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "", "b.prd.md", "---\ntitle: B\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "", "c.rfc.md", "---\ntitle: C\nstatus: draft\n---\n\nbody")

	callTool(HandleAddRelation(StaticRoot(base)), map[string]any{"source": "a.adr.md", "target": "b.prd.md", "type": "implements"})
	callTool(HandleAddRelation(StaticRoot(base)), map[string]any{"source": "b.prd.md", "target": "c.rfc.md", "type": "related"})
	callTool(HandleAddRelation(StaticRoot(base)), map[string]any{"source": "a.adr.md", "target": "c.rfc.md", "type": "depends_on"})

	result, _ := callTool(HandleListRelations(StaticRoot(base)), map[string]any{"path": "b.prd.md"})

	var resp map[string][]sync.Relation
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp["relations"]) != 2 {
		t.Errorf("expected 2 relations for b.prd.md, got %d", len(resp["relations"]))
	}
}

func TestHandleListRelations_NoManifest(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)

	result, err := callTool(HandleListRelations(StaticRoot(base)), map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("unexpected error")
	}

	var resp map[string][]sync.Relation
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp["relations"]) != 0 {
		t.Errorf("expected 0 relations, got %d", len(resp["relations"]))
	}
}

func TestHandleListRelations_NormalizesPrefix(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "", "b.prd.md", "---\ntitle: B\nstatus: draft\n---\n\nbody")

	callTool(HandleAddRelation(StaticRoot(base)), map[string]any{"source": "a.adr.md", "target": "b.prd.md", "type": "related"})

	result, _ := callTool(HandleListRelations(StaticRoot(base)), map[string]any{"path": ".archcore/a.adr.md"})

	var resp map[string][]sync.Relation
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp["relations"]) != 1 {
		t.Errorf("expected 1 relation with normalized prefix, got %d", len(resp["relations"]))
	}
}
