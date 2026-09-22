package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/config"
	"archcore-cli/internal/docs"
)

// assertNoAbsPath fails if msg embeds the absolute base path — a regression guard
// for no-absolute-paths-in-mcp-errors.rule on the global scan error path.
func assertNoAbsPath(t *testing.T, base, msg string) {
	t.Helper()
	if strings.Contains(msg, base) {
		t.Errorf("message leaked an absolute path (%q): %q", base, msg)
	}
}

// TestScanDocuments_EmptyGlobalIsNotFatal: a declared global that exists and is
// readable but holds no documents must NOT fail the scan — it simply contributes
// zero documents (the "empty" warning is surfaced by the reporting surfaces).
func TestScanDocuments_EmptyGlobalIsNotFatal(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "local.rule.md",
		"---\ntitle: Local\nstatus: accepted\n---\n\nbody\n")
	if err := os.MkdirAll(filepath.Join(base, ".archcore", "global", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeGlobalsSettings(t, base, []config.GlobalSource{{ID: "empty", Path: ".archcore/global/empty"}})

	docs, err := scanDocuments(base)
	if err != nil {
		t.Fatalf("empty global must not fail the scan: %v", err)
	}
	for _, d := range docs {
		if d.SourceKind == "global" {
			t.Errorf("empty global mounted a document: %+v", d)
		}
	}
	if len(docs) != 1 {
		t.Errorf("want 1 local doc, got %d", len(docs))
	}
}

// TestScanDocuments_FileAsPathFails: a global path pointing at a file (not a
// directory) must fail cleanly, without leaking an absolute path.
func TestScanDocuments_FileAsPathFails(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	if err := os.WriteFile(filepath.Join(base, "notadir.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeGlobalsSettings(t, base, []config.GlobalSource{{ID: "x", Path: "notadir.txt"}})

	_, err := scanDocuments(base)
	if err == nil {
		t.Fatal("a global path pointing at a file must fail the scan")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("error %q should say 'not a directory'", err)
	}
	assertNoAbsPath(t, base, err.Error())
}

// TestScanDocuments_UnreadableGlobalFails: a directory the process cannot read
// must fail at scan time (matching the startup gate) with a clean message — no
// raw "open /abs/path: permission denied" leak.
func TestScanDocuments_UnreadableGlobalFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: file permissions are not enforced")
	}
	t.Parallel()
	base := setupTestArchcore(t)
	writeGlobalDoc(t, base, ".archcore/global/noperm", "knowledge", "a.rule.md",
		"---\ntitle: A\nstatus: accepted\n---\n\nbody\n")
	noperm := filepath.Join(base, ".archcore", "global", "noperm")
	if err := os.Chmod(noperm, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(noperm, 0o755) })
	writeGlobalsSettings(t, base, []config.GlobalSource{{ID: "noperm", Path: ".archcore/global/noperm"}})

	_, err := scanDocuments(base)
	if err == nil {
		t.Fatal("an unreadable global must fail the scan")
	}
	if !strings.Contains(err.Error(), "not readable") {
		t.Errorf("error %q should say 'not readable'", err)
	}
	assertNoAbsPath(t, base, err.Error())
}

// TestScanDocuments_SelfOverlapFails: a global resolving to the project's own
// .archcore must be rejected — it would re-mount the primary's local documents
// as read-only globals.
func TestScanDocuments_SelfOverlapFails(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "local.rule.md",
		"---\ntitle: Local\nstatus: accepted\n---\n\nbody\n")
	writeGlobalsSettings(t, base, []config.GlobalSource{{ID: "self", Path: ".archcore"}})

	_, err := scanDocuments(base)
	if err == nil {
		t.Fatal("a global resolving to the project's own .archcore must fail")
	}
	if !strings.Contains(err.Error(), "own .archcore") {
		t.Errorf("error %q should mention self-overlap", err)
	}
	assertNoAbsPath(t, base, err.Error())
}

// TestScanDocuments_DuplicatePathFails: two globals resolving to the same
// directory must fail rather than mount every document twice.
func TestScanDocuments_DuplicatePathFails(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeGlobalDoc(t, base, ".archcore/global/dup", "knowledge", "a.rule.md",
		"---\ntitle: A\nstatus: accepted\n---\n\nbody\n")
	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "a", Path: ".archcore/global/dup"},
		{ID: "b", Path: ".archcore/global/dup"},
	})

	_, err := scanDocuments(base)
	if err == nil {
		t.Fatal("two globals resolving to the same path must fail")
	}
	if !strings.Contains(err.Error(), "same path") {
		t.Errorf("error %q should mention the duplicate path", err)
	}
}

// TestScanDocuments_GlobalValidTypeFilter: the global phase mounts only files
// with a recognized document type, so a misconfigured path cannot surface stray
// .md files (README, untyped notes) as malformed read-only documents.
func TestScanDocuments_GlobalValidTypeFilter(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeGlobalDoc(t, base, ".archcore/global/co", "knowledge", "good.rule.md",
		"---\ntitle: Good\nstatus: accepted\n---\n\nbody\n")
	writeGlobalDoc(t, base, ".archcore/global/co", "knowledge", "notes.bogus.md", "stray\n")
	writeGlobalsSettings(t, base, []config.GlobalSource{{ID: "co", Path: ".archcore/global/co"}})

	docs, err := scanDocuments(base)
	if err != nil {
		t.Fatal(err)
	}
	var globals []string
	for _, d := range docs {
		if d.SourceKind == "global" {
			globals = append(globals, d.Filename)
		}
	}
	if len(globals) != 1 || globals[0] != "good.rule.md" {
		t.Errorf("global phase should mount only valid-type docs; got %v", globals)
	}
}

// TestInspectGlobals_States exercises the reporting classifier across one ok,
// empty, missing, and duplicate source in a single settings.json.
func TestInspectGlobals_States(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeGlobalDoc(t, base, ".archcore/global/ok", "knowledge", "a.rule.md",
		"---\ntitle: A\nstatus: accepted\n---\n\nbody\n")
	if err := os.MkdirAll(filepath.Join(base, ".archcore", "global", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "ok", Path: ".archcore/global/ok"},
		{ID: "empty", Path: ".archcore/global/empty"},
		{ID: "gone", Path: ".archcore/global/gone"},
		{ID: "dup", Path: ".archcore/global/ok"},
	})

	got, err := inspectGlobals(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("want 4 inspections, got %d", len(got))
	}
	want := []docs.GlobalState{docs.GlobalOK, docs.GlobalEmpty, docs.GlobalMissing, docs.GlobalDuplicate}
	for i, w := range want {
		if got[i].State != w {
			t.Errorf("inspection[%d] (%s) state = %v, want %v", i, got[i].ID, got[i].State, w)
		}
	}
	if got[0].Docs != 1 {
		t.Errorf("ok source Docs = %d, want 1", got[0].Docs)
	}
}

// TestInspectGlobals_InvalidSettings: a present-but-invalid settings.json must
// return an error so callers fail closed instead of seeing "no globals".
func TestInspectGlobals_InvalidSettings(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	if err := os.WriteFile(filepath.Join(base, ".archcore", "settings.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := inspectGlobals(base); err == nil {
		t.Fatal("inspectGlobals must return an error for invalid settings.json")
	}
}
