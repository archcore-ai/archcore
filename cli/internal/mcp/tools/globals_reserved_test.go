package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/config"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestGlobalDirectory_IsOrdinaryLocalSpace pins that no directory name is
// reserved: a document under a "global" directory, at any depth and in any
// case, is a local document that every tool reads and writes
// (source-mounts.spec, invariant: one address per document).
func TestGlobalDirectory_IsOrdinaryLocalSpace(t *testing.T) {
	const doc = "---\ntitle: \"Deep\"\nstatus: accepted\n---\n\nbody\n"
	for _, dir := range []string{"global", "integrations/global", "Global"} {
		t.Run(dir, func(t *testing.T) {
			base := setupTestArchcore(t)
			writeDoc(t, base, "knowledge", "local.rule.md", doc)
			path := ".archcore/" + dir + "/deep.rule.md"

			created := mustCall(t, HandleCreateDocument(StaticRoot(base)), map[string]any{
				"type": "rule", "filename": "deep", "directory": dir, "content": doc,
			})
			if !strings.Contains(created, `"path":"`+path+`"`) {
				t.Fatalf("create_document = %s, want path %s", created, path)
			}

			got := mustCall(t, HandleGetDocument(StaticRoot(base)), map[string]any{"path": path})
			var read LocalDocument
			if err := json.Unmarshal([]byte(got), &read); err != nil {
				t.Fatal(err)
			}
			if read.SourceKind != "local" || read.SourceID != "local" || read.SourceRoot != "" {
				t.Errorf("get_document: source_kind=%q source_id=%q source_root=%q, want local/local/empty",
					read.SourceKind, read.SourceID, read.SourceRoot)
			}

			mustCall(t, HandleUpdateDocument(StaticRoot(base)), map[string]any{"path": path, "title": "Renamed"})
			mustCall(t, HandleAddRelation(StaticRoot(base)), map[string]any{
				"source": ".archcore/knowledge/local.rule.md", "target": path, "type": "related",
			})
			mustCall(t, HandleRemoveDocument(StaticRoot(base)), map[string]any{"path": path})
		})
	}
}

// TestSearchDocuments_AnnotatesSource pins that search rows carry the same
// source annotation as list_documents and get_document, across the body-loading
// branch (content, mode=full) and the metadata-only branch (types): a source row
// names its source_root, the project argument that writes it.
func TestSearchDocuments_AnnotatesSource(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "app")
	if err := os.MkdirAll(filepath.Join(base, ".archcore"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "company", Path: "../company/.archcore"},
	})
	writeDoc(t, base, "knowledge", "local-thing.rule.md",
		"---\ntitle: \"Local Thing\"\nstatus: accepted\n---\n\nsharedterm here\n")
	writeGlobalDoc(t, root, "company/.archcore", "knowledge", "global-thing.rule.md",
		"---\ntitle: \"Global Thing\"\nstatus: accepted\n---\n\nsharedterm here\n")
	wantRoot := filepath.Join(root, "company")

	assertAnnotated := func(t *testing.T, results []searchResult) {
		t.Helper()
		var local, global *searchResult
		for i := range results {
			switch {
			case strings.HasSuffix(results[i].Path, "local-thing.rule.md"):
				local = &results[i]
			case strings.HasSuffix(results[i].Path, "global-thing.rule.md"):
				global = &results[i]
			}
		}
		if local == nil || global == nil {
			t.Fatalf("expected both local and global hits, got %d results", len(results))
		}
		if local.SourceKind != "local" || local.SourceID != "local" || local.SourceRoot != "" {
			t.Errorf("local hit: source_kind=%q source_id=%q source_root=%q, want local/local/empty",
				local.SourceKind, local.SourceID, local.SourceRoot)
		}
		if global.SourceKind != "global" || global.SourceID != "company" || global.SourceRoot != wantRoot {
			t.Errorf("global hit: source_kind=%q source_id=%q source_root=%q, want global/company/%s",
				global.SourceKind, global.SourceID, global.SourceRoot, wantRoot)
		}
	}

	for name, args := range map[string]map[string]any{
		"content branch":  {"content": "sharedterm"},
		"full mode":       {"content": "sharedterm", "mode": "full"},
		"metadata branch": {"types": []any{"rule"}},
	} {
		t.Run(name, func(t *testing.T) {
			result, err := callTool(HandleSearchDocuments(StaticRoot(base)), args)
			if err != nil {
				t.Fatal(err)
			}
			assertAnnotated(t, unmarshalSearch(t, result))
		})
	}
}

// mustCall runs one tool call and returns its text, failing the test on an
// error result.
func mustCall(t *testing.T, handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) string {
	t.Helper()
	result, err := callTool(handler, args)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if result.IsError {
		t.Fatalf("tool error: %s", text)
	}
	return text
}
