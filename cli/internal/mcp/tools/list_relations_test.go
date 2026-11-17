package tools

import (
	"encoding/json"
	"fmt"
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

	var resp listRelationsResponse
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Relations) != 0 {
		t.Errorf("expected 0 relations, got %d", len(resp.Relations))
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

	var resp listRelationsResponse
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Relations) != 3 {
		t.Errorf("expected 3 relations, got %d", len(resp.Relations))
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

	var resp listRelationsResponse
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Relations) != 2 {
		t.Errorf("expected 2 relations for b.prd.md, got %d", len(resp.Relations))
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

	var resp listRelationsResponse
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Relations) != 0 {
		t.Errorf("expected 0 relations, got %d", len(resp.Relations))
	}
}

func TestHandleListRelations_NormalizesPrefix(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "", "a.adr.md", "---\ntitle: A\nstatus: draft\n---\n\nbody")
	writeDoc(t, base, "", "b.prd.md", "---\ntitle: B\nstatus: draft\n---\n\nbody")

	callTool(HandleAddRelation(StaticRoot(base)), map[string]any{"source": "a.adr.md", "target": "b.prd.md", "type": "related"})

	result, _ := callTool(HandleListRelations(StaticRoot(base)), map[string]any{"path": ".archcore/a.adr.md"})

	var resp listRelationsResponse
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Relations) != 1 {
		t.Errorf("expected 1 relation with normalized prefix, got %d", len(resp.Relations))
	}
}

func TestHandleListRelations_ResearchVocabulary(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"supports", "contradicts", "supersedes"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			base := setupTestArchcore(t)
			manifest := sync.NewManifest()
			want := sync.Relation{Source: "source.evidence.md", Target: "target.research.md", Type: sync.RelationType(value)}
			manifest.Relations = []sync.Relation{want}
			if err := sync.SaveManifest(base, manifest); err != nil {
				t.Fatal(err)
			}
			result, err := callTool(HandleListRelations(StaticRoot(base)), map[string]any{"path": ".archcore/target.research.md"})
			if err != nil || result.IsError {
				t.Fatalf("list: %v, %+v", err, result)
			}
			var response struct{ Relations []sync.Relation }
			if err := json.Unmarshal([]byte(resultText(t, result)), &response); err != nil {
				t.Fatal(err)
			}
			if len(response.Relations) != 1 || response.Relations[0] != want {
				t.Fatalf("relations = %+v, want %+v", response.Relations, want)
			}
		})
	}
}

type listRelationsResponse struct {
	Total     int             `json:"total"`
	Truncated bool            `json:"truncated"`
	Relations []sync.Relation `json:"relations"`
}

func TestHandleListRelations_CapsTheListing(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	m := sync.NewManifest()
	for i := range listRelationsLimit + 5 {
		m.Relations = append(m.Relations, sync.Relation{Source: fmt.Sprintf("s%d.adr.md", i), Target: "t.adr.md", Type: "related"})
	}
	if err := sync.SaveManifest(base, m); err != nil {
		t.Fatal(err)
	}

	result, err := callTool(HandleListRelations(StaticRoot(base)), map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	var resp listRelationsResponse
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Total != listRelationsLimit+5 || !resp.Truncated || len(resp.Relations) != listRelationsLimit {
		t.Errorf("total=%d truncated=%v listed=%d, want %d/true/%d", resp.Total, resp.Truncated, len(resp.Relations), listRelationsLimit+5, listRelationsLimit)
	}
	if resp.Relations[0].Source != "s0.adr.md" {
		t.Errorf("the cut did not keep manifest order: first = %s", resp.Relations[0].Source)
	}

	// Filtering by the hub document reaches every edge; otherwise the five
	// beyond the cap could not be listed by any call.
	result, err = callTool(HandleListRelations(StaticRoot(base)), map[string]any{"path": "t.adr.md"})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Truncated || len(resp.Relations) != listRelationsLimit+5 {
		t.Errorf("filtered listing: truncated=%v listed=%d, want false/%d", resp.Truncated, len(resp.Relations), listRelationsLimit+5)
	}
}
