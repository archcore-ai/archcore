package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"archcore-cli/internal/sync"

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

func TestHandleRemoveRelation_ResearchBoundaries(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, source, target, settings string }{
		{name: "global source", source: "global/source.evidence.md", target: "local.research.md"},
		{name: "global target", source: "local.evidence.md", target: "global/target.research.md"},
		{name: "traversal", source: "../source.evidence.md", target: "local.research.md"},
		{name: "absolute", source: "/source.evidence.md", target: "local.research.md"},
		{name: "unreadable settings", source: "local.evidence.md", target: "local.research.md", settings: "{"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, value := range []string{"supports", "contradicts", "supersedes"} {
				base := setupTestArchcore(t)
				if tt.settings != "" {
					if err := os.WriteFile(filepath.Join(base, ".archcore", "settings.json"), []byte(tt.settings), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				manifestPath := filepath.Join(base, ".archcore", ".sync-state.json")
				original := `{"version":1,"files":{},"relations":[]}`
				if err := os.WriteFile(manifestPath, []byte(original), 0o644); err != nil {
					t.Fatal(err)
				}
				result, err := callTool(HandleRemoveRelation(StaticRoot(base)), map[string]any{"source": tt.source, "target": tt.target, "type": value})
				if err != nil {
					t.Fatal(err)
				}
				if !result.IsError {
					t.Fatalf("%s accepted unsafe relation: %+v", value, result)
				}
				data, err := os.ReadFile(manifestPath)
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != original {
					t.Error("refusal changed manifest")
				}
			}
		})
	}
}

func TestHandleRemoveRelation_ResearchCanonicalAndLegacyPaths(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, storedSource, storedTarget, source, target string }{
		{"canonical", "nested/source.evidence.md", "nested/target.research.md", "./nested/source.evidence.md", "nested//target.research.md"},
		{"legacy", "./nested/source.evidence.md", "nested//target.research.md", "./nested/source.evidence.md", "nested//target.research.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, value := range []string{"supports", "contradicts", "supersedes"} {
				base := setupTestArchcore(t)
				manifest := sync.NewManifest()
				manifest.Relations = []sync.Relation{{Source: tt.storedSource, Target: tt.storedTarget, Type: sync.RelationType(value)}}
				if err := sync.SaveManifest(base, manifest); err != nil {
					t.Fatal(err)
				}
				result, err := callTool(HandleRemoveRelation(StaticRoot(base)), map[string]any{"source": tt.source, "target": tt.target, "type": value})
				if err != nil || result.IsError {
					t.Fatalf("remove relation: %v, %+v", err, result)
				}
				var response struct {
					Removed        bool
					Source, Target string
				}
				if err := json.Unmarshal([]byte(resultText(t, result)), &response); err != nil {
					t.Fatal(err)
				}
				if !response.Removed || response.Source != tt.storedSource || response.Target != tt.storedTarget {
					t.Fatalf("removed result = %+v, want stored triple removed", response)
				}
				manifest, err = sync.LoadManifest(base)
				if err != nil {
					t.Fatal(err)
				}
				if len(manifest.Relations) != 0 {
					t.Fatalf("relations after removal = %+v", manifest.Relations)
				}
			}
		})
	}
}

func TestRelationResearch_InvalidManifestUnchanged(t *testing.T) {
	t.Parallel()
	for _, operation := range []string{"add", "remove"} {
		t.Run(operation, func(t *testing.T) {
			t.Parallel()
			for _, value := range []string{"supports", "contradicts", "supersedes"} {
				base := setupTestArchcore(t)
				writeDoc(t, base, "", "source.evidence.md", "---\ntitle: Source\nstatus: draft\n---\n\nBody")
				writeDoc(t, base, "", "target.research.md", "---\ntitle: Target\nstatus: draft\n---\n\nBody")
				original := `{"version":999,"files":{},"relations":[]}`
				manifestPath := filepath.Join(base, ".archcore", sync.ManifestFile)
				if err := os.WriteFile(manifestPath, []byte(original), 0o644); err != nil {
					t.Fatal(err)
				}
				handler := HandleAddRelation(StaticRoot(base))
				if operation == "remove" {
					handler = HandleRemoveRelation(StaticRoot(base))
				}
				result, err := callTool(handler, map[string]any{"source": "source.evidence.md", "target": "target.research.md", "type": value})
				if err != nil {
					t.Fatal(err)
				}
				if !result.IsError {
					t.Fatal("invalid manifest was accepted")
				}
				data, err := os.ReadFile(manifestPath)
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != original {
					t.Fatalf("refusal changed manifest: %s", data)
				}
			}
		})
	}
}
