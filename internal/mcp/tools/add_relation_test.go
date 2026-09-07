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

func TestHandleAddRelation_ResearchBoundaries(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, source, target, settings, want string }{
		{name: "missing source", source: "missing.evidence.md", target: "local.research.md"},
		{name: "missing target", source: "local.evidence.md", target: "missing.research.md"},
		{name: "same document", source: "local.evidence.md", target: "local.evidence.md"},
		{name: "same dotted document", source: "./local.evidence.md", target: "local.evidence.md", want: "different documents"},
		{name: "same nested document", source: "nested//./one.evidence.md", target: "nested/one.evidence.md", want: "different documents"},
		{name: "global source", source: "global/source.evidence.md", target: "local.research.md", want: "read-only global source"},
		{name: "global target", source: "local.evidence.md", target: "global/target.research.md", want: "read-only global source"},
		{name: "traversal", source: "../source.evidence.md", target: "local.research.md"},
		{name: "absolute", source: "/source.evidence.md", target: "local.research.md"},
		{name: "unreadable settings", source: "local.evidence.md", target: "local.research.md", settings: "{"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, value := range []string{"supports", "contradicts", "supersedes"} {
				base := setupTestArchcore(t)
				writeDoc(t, base, "", "local.evidence.md", "---\ntitle: Local\nstatus: draft\n---\n\nBody")
				writeDoc(t, base, "", "local.research.md", "---\ntitle: Local\nstatus: draft\n---\n\nBody")
				writeDoc(t, base, "global", "source.evidence.md", "---\ntitle: Global\nstatus: draft\n---\n\nBody")
				writeDoc(t, base, "global", "target.research.md", "---\ntitle: Global\nstatus: draft\n---\n\nBody")
				writeDoc(t, base, "nested", "one.evidence.md", "---\ntitle: Nested\nstatus: draft\n---\n\nBody")
				if tt.settings != "" {
					if err := os.WriteFile(filepath.Join(base, ".archcore", "settings.json"), []byte(tt.settings), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				manifestPath := filepath.Join(base, ".archcore", sync.ManifestFile)
				original := `{"version":1,"files":{},"relations":[]}`
				if err := os.WriteFile(manifestPath, []byte(original), 0o644); err != nil {
					t.Fatal(err)
				}
				result, err := callTool(HandleAddRelation(StaticRoot(base)), map[string]any{"source": tt.source, "target": tt.target, "type": value})
				if err != nil {
					t.Fatal(err)
				}
				if !result.IsError {
					t.Fatalf("%s accepted unsafe relation: %+v", value, result)
				}
				if got := resultText(t, result); tt.want != "" && !strings.Contains(got, tt.want) {
					t.Fatalf("refusal = %q, want %q", got, tt.want)
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

func TestHandleAddRelation_ResearchCanonicalPaths(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"supports", "contradicts", "supersedes"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			base := setupTestArchcore(t)
			writeDoc(t, base, "nested", "source.evidence.md", "---\ntitle: Source\nstatus: draft\n---\n\nBody")
			writeDoc(t, base, "nested", "target.research.md", "---\ntitle: Target\nstatus: draft\n---\n\nBody")
			args := map[string]any{"source": "nested//./source.evidence.md", "target": "nested/./target.research.md", "type": value}
			result, err := callTool(HandleAddRelation(StaticRoot(base)), args)
			if err != nil || result.IsError {
				t.Fatalf("add relation: %v, %+v", err, result)
			}
			manifest, err := sync.LoadManifest(base)
			if err != nil {
				t.Fatal(err)
			}
			want := sync.Relation{Source: "nested/source.evidence.md", Target: "nested/target.research.md", Type: sync.RelationType(value)}
			if len(manifest.Relations) != 1 || manifest.Relations[0] != want {
				t.Fatalf("persisted relations = %+v, want %+v", manifest.Relations, want)
			}
		})
	}
}
