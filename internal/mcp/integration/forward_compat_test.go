package integration

import (
	"path/filepath"
	"testing"
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
