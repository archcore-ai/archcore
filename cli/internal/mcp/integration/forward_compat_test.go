package integration

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestServer_ServesLocalDocsWithUnknownField proves the keep-serving property
// end-to-end through the real MCP server: a settings.json that carries a field
// this binary does not recognize (as a newer archcore would add) must not break
// the read path — list_documents still returns the project's local documents.
func TestServer_ServesLocalDocsWithUnknownField(t *testing.T) {
	base := t.TempDir()
	writeFixtureFile(t, filepath.Join(base, ".archcore", "service.doc.md"),
		"---\ntitle: \"Service\"\nstatus: accepted\n---\n\nbody\n")
	// settings.json with a field unknown to this binary.
	writeFixtureFile(t, filepath.Join(base, ".archcore", "settings.json"),
		"{\n  \"sync\": \"none\",\n  \"future_flag\": true\n}\n")

	c := newTestClient(t, base)

	res := mustCallTool(t, c, "list_documents", map[string]any{})
	docs := decodeListedDocs(t, res)
	if len(docs) != 1 {
		t.Fatalf("want 1 local doc served despite an unknown config field, got %d: %+v", len(docs), docs)
	}
	if docs[0].SourceKind != "local" {
		t.Errorf("source_kind = %q, want local", docs[0].SourceKind)
	}
}

func TestUpdateDocument_RetainsUnknownMetadataAcrossReads(t *testing.T) {
	t.Parallel()
	base := initArchcore(t)
	path := ".archcore/material.evidence.md"
	input := "---\ntitle: Material\nstatus: draft\ncustom: {nested: [1, true]}\nreviewed: null\n---\n\nOriginal body"
	writeFixtureFile(t, filepath.Join(base, path), input)
	c := newTestClient(t, base)
	mustCallTool(t, c, "get_document", map[string]any{"path": path})
	mustCallTool(t, c, "list_documents", nil)
	mustCallTool(t, c, "update_document", map[string]any{"path": path, "content": "Changed body"})
	mustCallTool(t, c, "update_document", map[string]any{"path": path, "title": "Revised Material"})
	doc := decodeJSON[map[string]any](t, mustCallTool(t, c, "get_document", map[string]any{"path": path}))
	content, ok := doc["content"].(string)
	if !ok {
		t.Fatalf("get_document content = %v", doc)
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) != 3 {
		t.Fatalf("missing frontmatter: %q", content)
	}
	var fields map[string]any
	if err := yaml.Unmarshal([]byte(parts[1]), &fields); err != nil {
		t.Fatal(err)
	}
	if fields["title"] != "Revised Material" || fields["status"] != "draft" {
		t.Errorf("owned fields = %v", fields)
	}
	if !reflect.DeepEqual(fields["custom"], map[string]any{"nested": []any{1, true}}) {
		t.Errorf("custom metadata = %#v", fields["custom"])
	}
	if v, ok := fields["reviewed"]; !ok || v != nil {
		t.Errorf("null metadata = %v (present %v)", v, ok)
	}
	if strings.TrimSpace(parts[2]) != "Changed body" {
		t.Errorf("body = %q", parts[2])
	}
	hits := decodeJSON[struct {
		Results []struct {
			Body string `json:"body"`
		} `json:"results"`
	}](t, mustCallTool(t, c, "search_documents", map[string]any{"types": []string{"evidence"}, "content": "Changed body", "mode": "full"})).Results
	if len(hits) != 1 || !strings.Contains(hits[0].Body, "Changed body") {
		t.Errorf("search sees stale data: %+v", hits)
	}
}
