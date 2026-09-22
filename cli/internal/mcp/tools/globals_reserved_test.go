package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"archcore-cli/internal/config"
	"archcore-cli/internal/docs"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestIsReservedGlobalDir covers the any-depth segment match that keeps the
// write/relation guard in agreement with the local-scan skip (BUG-1).
func TestIsReservedGlobalDir(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "reserved root", path: ".archcore/global", want: true},
		{name: "document directly under the reserved root", path: ".archcore/global/x.rule.md", want: true},
		{name: "document deep under the reserved root", path: ".archcore/global/company/knowledge/react-query.rule.md", want: true},
		{name: "reserved segment one level deep", path: ".archcore/integrations/global/deep.rule.md", want: true},
		{name: "reserved segment at any depth", path: ".archcore/a/b/global/c.rule.md", want: true},
		{name: "substring is not a segment", path: ".archcore/global-ish/x.rule.md", want: false},
		{name: "global in the filename is not a directory", path: ".archcore/knowledge/global.rule.md", want: false},
		{name: "ordinary local document", path: ".archcore/knowledge/local.rule.md", want: false},
		{name: "outside .archcore/", path: "global/x.rule.md", want: false},
		{name: "empty path", path: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := docs.IsReservedGlobalDir(tt.path); got != tt.want {
				t.Errorf("IsReservedGlobalDir(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestScanDocuments_SkipsNestedGlobalDir mirrors the top-level skip test but with
// a global/ directory nested one level deep: it must still be invisible (BUG-1).
func TestScanDocuments_SkipsNestedGlobalDir(t *testing.T) {
	base := setupTestArchcore(t)

	writeDoc(t, base, "knowledge", "local.rule.md",
		"---\ntitle: \"Local\"\nstatus: accepted\n---\n\nbody\n")
	// A directory named "global" nested under a normal subdir, no declaration.
	writeGlobalDoc(t, base, ".archcore/integrations/global", "knowledge", "deep.rule.md",
		"---\ntitle: \"Deep Nested\"\nstatus: accepted\n---\n\nbody\n")

	docs, err := scanDocuments(base)
	if err != nil {
		t.Fatalf("scanDocuments: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("want 1 doc (nested global/ skipped), got %d", len(docs))
	}
	if len(docs) > 0 && docs[0].SourceKind != "local" {
		t.Errorf("doc should be local, got source_kind=%q", docs[0].SourceKind)
	}
}

// TestUpdateDocument_RejectsNestedReservedGlobalDir is the BUG-1 write hole: a doc
// under a nested global/ directory was invisible yet writable. It must now be
// rejected with the clean read-only message.
func TestUpdateDocument_RejectsNestedReservedGlobalDir(t *testing.T) {
	base := setupTestArchcore(t)

	writeGlobalDoc(t, base, ".archcore/integrations/global", "knowledge", "deep.rule.md",
		"---\ntitle: \"Deep Nested\"\nstatus: accepted\n---\n\nbody\n")

	result, err := callTool(HandleUpdateDocument(StaticRoot(base)), map[string]any{
		"path":  ".archcore/integrations/global/knowledge/deep.rule.md",
		"title": "Hijacked",
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error updating a nested reserved-global doc, got success")
	}
	if got := result.Content[0].(mcp.TextContent).Text; got != "cannot update a read-only global source document" {
		t.Errorf("message = %q, want clean read-only message", got)
	}
}

// TestGetDocument_ReservedDirIsReadOnly is the BUG-3 read mislabel: undeclared
// content in the reserved global/ tree must be annotated global/read_only so the
// read label matches the write guard.
func TestGetDocument_ReservedDirIsReadOnly(t *testing.T) {
	base := setupTestArchcore(t)

	writeGlobalDoc(t, base, ".archcore/global/vendored", "knowledge", "sample.rule.md",
		"---\ntitle: \"Sample Vendored\"\nstatus: accepted\n---\n\nbody\n")

	result, err := callTool(HandleGetDocument(StaticRoot(base)), map[string]any{
		"path": ".archcore/global/vendored/knowledge/sample.rule.md",
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success reading reserved-dir doc, got error: %s",
			result.Content[0].(mcp.TextContent).Text)
	}

	var doc LocalDocument
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SourceKind != "global" || !doc.ReadOnly || !doc.Global {
		t.Errorf("source_kind=%q read_only=%v global=%v, want global/true/true",
			doc.SourceKind, doc.ReadOnly, doc.Global)
	}
	if doc.SourceID != "__global__" {
		t.Errorf("source_id = %q, want %q (reserved sentinel)", doc.SourceID, "__global__")
	}
}

// TestSearchDocuments_AnnotatesSource is BUG-2: search results must carry the same
// source annotation as list_documents/get_document, across both the body-loading
// branch (content / mode=full) and the metadata-only branch (types).
func TestSearchDocuments_AnnotatesSource(t *testing.T) {
	base := setupTestArchcore(t)

	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "company", Path: ".archcore/global/company"},
	})
	writeDoc(t, base, "knowledge", "local-thing.rule.md",
		"---\ntitle: \"Local Thing\"\nstatus: accepted\n---\n\nsharedterm here\n")
	writeGlobalDoc(t, base, ".archcore/global/company", "knowledge", "global-thing.rule.md",
		"---\ntitle: \"Global Thing\"\nstatus: accepted\n---\n\nsharedterm here\n")

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
		if local.SourceKind != "local" || local.SourceID != "local" || local.ReadOnly {
			t.Errorf("local hit: source_kind=%q source_id=%q read_only=%v, want local/local/false",
				local.SourceKind, local.SourceID, local.ReadOnly)
		}
		if global.SourceKind != "global" || global.SourceID != "company" || !global.ReadOnly || !global.Global {
			t.Errorf("global hit: source_kind=%q source_id=%q read_only=%v global=%v, want global/company/true/true",
				global.SourceKind, global.SourceID, global.ReadOnly, global.Global)
		}
	}

	t.Run("content branch", func(t *testing.T) {
		result, err := callTool(HandleSearchDocuments(StaticRoot(base)), map[string]any{"content": "sharedterm"})
		if err != nil {
			t.Fatal(err)
		}
		assertAnnotated(t, unmarshalSearch(t, result))
	})

	t.Run("full mode", func(t *testing.T) {
		result, err := callTool(HandleSearchDocuments(StaticRoot(base)), map[string]any{
			"content": "sharedterm",
			"mode":    "full",
		})
		if err != nil {
			t.Fatal(err)
		}
		assertAnnotated(t, unmarshalSearch(t, result))
	})

	t.Run("metadata branch", func(t *testing.T) {
		result, err := callTool(HandleSearchDocuments(StaticRoot(base)), map[string]any{
			"types": []any{"rule"},
		})
		if err != nil {
			t.Fatal(err)
		}
		assertAnnotated(t, unmarshalSearch(t, result))
	})
}

// TestCreateDocument_RejectsNestedReservedGlobalDir is the BUG-1 write hole for
// create: a directory under a nested global/ path is reserved read-only space, so
// creating a document there must be rejected with the clean read-only message —
// even with no global source declared in settings.json.
func TestCreateDocument_RejectsNestedReservedGlobalDir(t *testing.T) {
	base := setupTestArchcore(t)

	result, err := callTool(HandleCreateDocument(StaticRoot(base)), map[string]any{
		"type":      "rule",
		"filename":  "injected",
		"directory": "integrations/global/knowledge",
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error creating under a nested reserved-global dir, got success")
	}
	if got := result.Content[0].(mcp.TextContent).Text; got != "cannot create document in a read-only global source" {
		t.Errorf("message = %q, want clean read-only message", got)
	}
}

// TestRemoveDocument_RejectsNestedReservedGlobalDir is the BUG-1 write hole for
// remove: a doc under a nested global/ directory is read-only, so removing it must
// be rejected with the clean read-only message.
func TestRemoveDocument_RejectsNestedReservedGlobalDir(t *testing.T) {
	base := setupTestArchcore(t)

	writeGlobalDoc(t, base, ".archcore/integrations/global", "knowledge", "deep.rule.md",
		"---\ntitle: \"Deep Nested\"\nstatus: accepted\n---\n\nbody\n")

	result, err := callTool(HandleRemoveDocument(StaticRoot(base)), map[string]any{
		"path": ".archcore/integrations/global/knowledge/deep.rule.md",
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error removing a nested reserved-global doc, got success")
	}
	if got := result.Content[0].(mcp.TextContent).Text; got != "cannot remove a read-only global source document" {
		t.Errorf("message = %q, want clean read-only message", got)
	}
}

// TestAddRelation_RejectsNestedReservedGlobalEndpoint is the BUG-1 write hole for
// relations: an endpoint under a nested global/ directory is read-only, so a
// relation touching it must be rejected in EITHER direction (source or target).
func TestAddRelation_RejectsNestedReservedGlobalEndpoint(t *testing.T) {
	base := setupTestArchcore(t)

	writeDoc(t, base, "knowledge", "local.rule.md",
		"---\ntitle: \"Local\"\nstatus: accepted\n---\n\nbody\n")
	writeGlobalDoc(t, base, ".archcore/integrations/global", "knowledge", "deep.rule.md",
		"---\ntitle: \"Deep Nested\"\nstatus: accepted\n---\n\nbody\n")

	const (
		local  = ".archcore/knowledge/local.rule.md"
		nested = ".archcore/integrations/global/knowledge/deep.rule.md"
	)
	cases := []struct {
		name           string
		source, target string
	}{
		{"nested as target", local, nested},
		{"nested as source", nested, local},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := callTool(HandleAddRelation(StaticRoot(base)), map[string]any{
				"source": tt.source, "target": tt.target, "type": "related",
			})
			if err != nil {
				t.Fatalf("unexpected transport error: %v", err)
			}
			if !result.IsError {
				t.Fatalf("expected error relating %q -> %q, got success", tt.source, tt.target)
			}
			if got := result.Content[0].(mcp.TextContent).Text; got != "cannot add a relation involving a read-only global source document — relations connect local documents only" {
				t.Errorf("message = %q, want clean read-only message", got)
			}
		})
	}
}
