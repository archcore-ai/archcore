package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// listedDoc is the subset of the list_documents JSON payload the globals
// scenario asserts on. The handler marshals tools.LocalDocument, so these tags
// match that struct.
type listedDoc struct {
	Path       string `json:"path"`
	Slug       string `json:"slug"`
	SourceID   string `json:"source_id"`
	SourceKind string `json:"source_kind"`
	ReadOnly   bool   `json:"read_only"`
}

// writeFixtureFile writes content to path, creating parent directories.
func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// setupPrimaryWithSiblingGlobal builds, under a shared parent:
//
//	primary/.archcore/            — the writable project (one local rule)
//	company-global/.archcore/...  — a sibling global (two rules)
//
// and declares the global in primary via a "../"-relative path, exactly as the
// examples/ fixtures do. Returns the primary base dir.
func setupPrimaryWithSiblingGlobal(t *testing.T) string {
	t.Helper()
	parent := t.TempDir()
	base := filepath.Join(parent, "primary")

	// Local doc — same slug as a global one, to exercise override visibility.
	writeFixtureFile(t, filepath.Join(base, ".archcore", "error-handling.rule.md"),
		"---\ntitle: \"Error Handling (local override)\"\nstatus: accepted\n---\n\nbody\n")

	// Sibling global holding the two standards an agent would ask about.
	globalKnowledge := filepath.Join(parent, "company-global", ".archcore", "knowledge")
	writeFixtureFile(t, filepath.Join(globalKnowledge, "error-handling.rule.md"),
		"---\ntitle: \"Error Handling (company)\"\nstatus: accepted\n---\n\nwrap errors with %w\n")
	writeFixtureFile(t, filepath.Join(globalKnowledge, "logging.rule.md"),
		"---\ntitle: \"Logging (company)\"\nstatus: accepted\n---\n\nstructured logging only\n")

	// Declare the sibling global via a relative path, like examples/05 and /07.
	writeFixtureFile(t, filepath.Join(base, ".archcore", "settings.json"),
		"{\n  \"sync\": \"none\",\n  \"globals\": [\n    { \"id\": \"company\", \"path\": \"../company-global/.archcore\" }\n  ]\n}\n")

	return base
}

// TestGlobals_SurfacedThroughTools is the regression guard for the failure mode
// behind this test's existence: an agent asks "what are our X standards?", the
// MCP tools return only the local docs, and the agent falls back to reading the
// mounted global off disk. A mounted sibling global MUST appear through the real
// server's list_documents — tagged read-only — so the agent never touches the
// filesystem directly. The per-tool unit suite exercises ScanDocuments() but not
// the registered handler over the protocol; this closes that gap.
func TestGlobals_SurfacedThroughTools(t *testing.T) {
	base := setupPrimaryWithSiblingGlobal(t)
	c := newTestClient(t, base)

	res := mustCallTool(t, c, "list_documents", map[string]any{})
	docs := decodeJSON[[]listedDoc](t, res)
	if len(docs) != 3 {
		t.Fatalf("list_documents: want 3 docs (1 local + 2 global), got %d: %+v", len(docs), docs)
	}

	var locals, globals []listedDoc
	for _, d := range docs {
		switch d.SourceKind {
		case "local":
			locals = append(locals, d)
		case "global":
			globals = append(globals, d)
		default:
			t.Errorf("doc %s has unexpected source_kind %q", d.Path, d.SourceKind)
		}
	}

	if len(locals) != 1 {
		t.Errorf("want 1 local doc, got %d", len(locals))
	} else if locals[0].ReadOnly {
		t.Error("local doc must be writable (read_only=false)")
	}

	if len(globals) != 2 {
		t.Fatalf("want 2 global docs surfaced through the tool, got %d", len(globals))
	}
	gotGlobalSlugs := make(map[string]bool, len(globals))
	for _, g := range globals {
		gotGlobalSlugs[g.Slug] = true
		if g.SourceID != "company" {
			t.Errorf("global %s: source_id = %q, want company", g.Slug, g.SourceID)
		}
		if !g.ReadOnly {
			t.Errorf("global %s: want read_only=true", g.Slug)
		}
	}
	for _, want := range []string{"error-handling", "logging"} {
		if !gotGlobalSlugs[want] {
			t.Errorf("global standard %q not surfaced through list_documents", want)
		}
	}
}

// TestGlobals_SearchSeesGlobals verifies search_documents scans the mounted
// global too: a phrase that exists ONLY in the global logging rule must surface
// that global document. (search results carry no source tags by design — see
// search-documents-matching-not-presentation.adr — so we assert on the path.)
func TestGlobals_SearchSeesGlobals(t *testing.T) {
	base := setupPrimaryWithSiblingGlobal(t)
	c := newTestClient(t, base)

	res := mustCallTool(t, c, "search_documents", map[string]any{
		"content": "structured logging",
	})
	hits := decodeJSON[[]struct {
		Path string `json:"path"`
	}](t, res)

	found := false
	for _, h := range hits {
		if strings.HasSuffix(h.Path, "logging.rule.md") {
			found = true
		}
	}
	if !found {
		t.Errorf("search_documents(content=%q) did not surface the global logging rule; hits=%+v",
			"structured logging", hits)
	}
}

// TestGlobals_WriteToGlobalRejected verifies the read-only invariant through the
// real server: update_document targeting a mounted global document must fail.
// (For a "../"-relative global the path-validation guard fires first — spec
// §5.5 — so we assert the call errors, not the exact message.)
func TestGlobals_WriteToGlobalRejected(t *testing.T) {
	base := setupPrimaryWithSiblingGlobal(t)
	c := newTestClient(t, base)

	errText := expectToolError(t, c, "update_document", map[string]any{
		"path":  "../company-global/.archcore/knowledge/logging.rule.md",
		"title": "Hijacked",
	})
	if errText == "" {
		t.Error("expected a non-empty error message rejecting the global write")
	}
}

// TestGlobals_GetExternalGlobalReadable verifies the read-path relaxation end to
// end: a sibling global declared via a "../" path — the canonical examples/05
// layout — must be readable through get_document, tagged read-only with its source
// id and its body inlined. Before the relaxation this returned `invalid path: must
// start with ".archcore/"` because the document renders with a leading "..".
func TestGlobals_GetExternalGlobalReadable(t *testing.T) {
	base := setupPrimaryWithSiblingGlobal(t)
	c := newTestClient(t, base)

	res := mustCallTool(t, c, "get_document", map[string]any{
		"path": "../company-global/.archcore/knowledge/logging.rule.md",
	})
	doc := decodeJSON[struct {
		SourceID   string `json:"source_id"`
		SourceKind string `json:"source_kind"`
		ReadOnly   bool   `json:"read_only"`
		Content    string `json:"content"`
	}](t, res)
	if doc.SourceKind != "global" || doc.SourceID != "company" || !doc.ReadOnly {
		t.Errorf("got source_kind=%q source_id=%q read_only=%v; want global/company/true",
			doc.SourceKind, doc.SourceID, doc.ReadOnly)
	}
	if !strings.Contains(doc.Content, "structured logging") {
		t.Errorf("global body not inlined; content=%q", doc.Content)
	}
}

// setupPrimaryWithInTreeGlobal builds a primary with a global vendored in-tree
// under the reserved .archcore/global/company directory. An in-tree global's
// documents have valid .archcore/ paths, so the relation guard — which validates
// endpoints with the strict path check — can be exercised against them directly.
// (External "../" globals are now readable via get_document too; see
// TestGlobals_GetExternalGlobalReadable.)
func setupPrimaryWithInTreeGlobal(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	writeFixtureFile(t, filepath.Join(base, ".archcore", "service.doc.md"),
		"---\ntitle: \"Service\"\nstatus: accepted\n---\n\nbody\n")
	writeFixtureFile(t, filepath.Join(base, ".archcore", "global", "company", "knowledge", "logging.rule.md"),
		"---\ntitle: \"Logging (company)\"\nstatus: accepted\n---\n\nstructured logging only\n")
	writeFixtureFile(t, filepath.Join(base, ".archcore", "settings.json"),
		"{\n  \"sync\": \"none\",\n  \"globals\": [\n    { \"id\": \"company\", \"path\": \".archcore/global/company\" }\n  ]\n}\n")
	return base
}

// TestGlobals_GetDocumentAnnotatesGlobal closes the §4.2/§4.3 coverage gap: reading
// a mounted global through get_document must tag it read-only with its source id.
func TestGlobals_GetDocumentAnnotatesGlobal(t *testing.T) {
	base := setupPrimaryWithInTreeGlobal(t)
	c := newTestClient(t, base)

	res := mustCallTool(t, c, "get_document", map[string]any{
		"path": ".archcore/global/company/knowledge/logging.rule.md",
	})
	doc := decodeJSON[struct {
		SourceID   string `json:"source_id"`
		SourceKind string `json:"source_kind"`
		ReadOnly   bool   `json:"read_only"`
	}](t, res)
	if doc.SourceKind != "global" {
		t.Errorf("source_kind = %q, want global", doc.SourceKind)
	}
	if doc.SourceID != "company" {
		t.Errorf("source_id = %q, want company", doc.SourceID)
	}
	if !doc.ReadOnly {
		t.Error("want read_only=true")
	}
}

// TestGlobals_RelationToGlobalRejected verifies through the real server that a
// relation touching a global document (here as target) is refused.
func TestGlobals_RelationToGlobalRejected(t *testing.T) {
	base := setupPrimaryWithInTreeGlobal(t)
	c := newTestClient(t, base)

	errText := expectToolError(t, c, "add_relation", map[string]any{
		"source": ".archcore/service.doc.md",
		"target": ".archcore/global/company/knowledge/logging.rule.md",
		"type":   "related",
	})
	if errText == "" {
		t.Error("expected a non-empty error rejecting the relation to a global")
	}
}
