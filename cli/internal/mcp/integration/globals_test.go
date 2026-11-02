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
	SourceID   string `json:"source_id"`
	SourceKind string `json:"source_kind"`
	SourceRoot string `json:"source_root"`
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
// server's list_documents — tagged with its source — so the agent never touches the
// filesystem directly. The per-tool unit suite exercises ScanDocuments() but not
// the registered handler over the protocol; this closes that gap.
func TestGlobals_SurfacedThroughTools(t *testing.T) {
	base := setupPrimaryWithSiblingGlobal(t)
	c := newTestClient(t, base)

	res := mustCallTool(t, c, "list_documents", map[string]any{})
	docs := decodeListedDocs(t, res)
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
	} else if locals[0].SourceRoot != "" {
		t.Errorf("local doc: source_root = %q, want none", locals[0].SourceRoot)
	}

	if len(globals) != 2 {
		t.Fatalf("want 2 global docs surfaced through the tool, got %d", len(globals))
	}
	gotGlobalFiles := make(map[string]bool, len(globals))
	for _, g := range globals {
		gotGlobalFiles[filepath.Base(g.Path)] = true
		if g.SourceID != "company" {
			t.Errorf("global %s: source_id = %q, want company", g.Path, g.SourceID)
		}
		if want := filepath.Join(filepath.Dir(base), "company-global"); g.SourceRoot != want {
			t.Errorf("global %s: source_root = %q, want %q", g.Path, g.SourceRoot, want)
		}
	}
	for _, want := range []string{"error-handling.rule.md", "logging.rule.md"} {
		if !gotGlobalFiles[want] {
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
	hits := decodeJSON[struct {
		Results []struct {
			Path string `json:"path"`
		} `json:"results"`
	}](t, res).Results

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

// TestGlobals_WriteToGlobalRejected verifies through the real server that a
// write addressed by a source row path is refused with the pass-project
// instruction and the source id (source-mounts.spec, failure rules).
func TestGlobals_WriteToGlobalRejected(t *testing.T) {
	base := setupPrimaryWithSiblingGlobal(t)
	c := newTestClient(t, base)

	errText := expectToolError(t, c, "update_document", map[string]any{
		"path":  "../company-global/.archcore/knowledge/logging.rule.md",
		"title": "Hijacked",
	})
	for _, want := range []string{"pass that project's absolute root as project", `source "company"`} {
		if !strings.Contains(errText, want) {
			t.Errorf("error = %q, want %q", errText, want)
		}
	}
}

// TestGlobals_WriteThroughSourceProject verifies the write path the refusal
// names: project set to the row's source_root edits the source document, and
// the response names that project and its source id.
func TestGlobals_WriteThroughSourceProject(t *testing.T) {
	base := setupPrimaryWithSiblingGlobal(t)
	c := newTestClient(t, base)
	sourceRoot := filepath.Join(filepath.Dir(base), "company-global")

	res := mustCallTool(t, c, "update_document", map[string]any{
		"project": sourceRoot,
		"path":    ".archcore/knowledge/logging.rule.md",
		"title":   "Logging (edited)",
	})
	got := decodeJSON[struct {
		Project string `json:"project"`
		Source  string `json:"source"`
	}](t, res)
	if got.Source != "company" {
		t.Errorf("source = %q, want company", got.Source)
	}
	if got.Project == "" {
		t.Error("project missing from the response")
	}
	data, err := os.ReadFile(filepath.Join(sourceRoot, ".archcore", "knowledge", "logging.rule.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Logging (edited)") {
		t.Errorf("source document not updated:\n%s", data)
	}

	relation := firstText(mustCallTool(t, c, "add_relation", map[string]any{
		"project": sourceRoot,
		"source":  ".archcore/knowledge/logging.rule.md",
		"target":  ".archcore/knowledge/error-handling.rule.md",
		"type":    "related",
	}))
	if n := strings.Count(relation, `"source":`); n != 1 {
		t.Errorf("add_relation response holds %d \"source\" keys, want 1 (the endpoint): %s", n, relation)
	}

	local := decodeJSON[struct {
		Source string `json:"source"`
	}](t, mustCallTool(t, c, "list_documents", map[string]any{}))
	if local.Source != "local" {
		t.Errorf("session-root response source = %q, want local", local.Source)
	}
}

// TestGlobals_GetExternalGlobalReadable verifies the read-path relaxation end to
// end: a sibling global declared via a "../" path — the canonical examples/05
// layout — must be readable through get_document, tagged with its source id and
// source_root, with its body inlined. Before the relaxation this returned `invalid path: must
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
		SourceRoot string `json:"source_root"`
		Content    string `json:"content"`
	}](t, res)
	wantRoot := filepath.Join(filepath.Dir(base), "company-global")
	if doc.SourceKind != "global" || doc.SourceID != "company" || doc.SourceRoot != wantRoot {
		t.Errorf("got source_kind=%q source_id=%q source_root=%q; want global/company/%s",
			doc.SourceKind, doc.SourceID, doc.SourceRoot, wantRoot)
	}
	if !strings.Contains(doc.Content, "structured logging") {
		t.Errorf("global body not inlined; content=%q", doc.Content)
	}
}

// TestGlobals_RelationToGlobalRejected verifies through the real server that a
// relation touching a global document (here as target) is refused.
func TestGlobals_RelationToGlobalRejected(t *testing.T) {
	base := setupPrimaryWithSiblingGlobal(t)
	c := newTestClient(t, base)

	errText := expectToolError(t, c, "add_relation", map[string]any{
		"source": ".archcore/error-handling.rule.md",
		"target": "../company-global/.archcore/knowledge/logging.rule.md",
		"type":   "related",
	})
	if errText == "" {
		t.Error("expected a non-empty error rejecting the relation to a global")
	}
}
