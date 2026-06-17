package tools

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/config"

	"github.com/mark3labs/mcp-go/mcp"
)

// writeGlobalDoc creates a document inside a global source tree:
// base/<gsPath>/.archcore/<subdir>/<filename>
func writeGlobalDoc(t *testing.T, base, gsPath, subdir, filename, content string) {
	t.Helper()
	dir := filepath.Join(base, gsPath, subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeGlobalsSettings writes settings.json declaring the given global sources.
func writeGlobalsSettings(t *testing.T, base string, globals []config.GlobalSource) {
	t.Helper()
	type settingsJSON struct {
		Sync    string                `json:"sync"`
		Globals []config.GlobalSource `json:"globals,omitempty"`
	}
	data, err := json.MarshalIndent(settingsJSON{Sync: "none", Globals: globals}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(base, ".archcore", "settings.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanDocuments_WithGlobals(t *testing.T) {
	base := setupTestArchcore(t)

	// Local documents.
	writeDoc(t, base, "knowledge", "local-only.rule.md",
		"---\ntitle: \"Local Only\"\nstatus: accepted\n---\n\nbody\n")
	writeDoc(t, base, "knowledge", "react-query.rule.md",
		"---\ntitle: \"React Query (Local Override)\"\nstatus: accepted\n---\n\nbody\n")

	// Global: company source.
	writeGlobalDoc(t, base, ".archcore/global/company", "knowledge", "react-query.rule.md",
		"---\ntitle: \"React Query (Company)\"\nstatus: accepted\n---\n\nbody\n")
	writeGlobalDoc(t, base, ".archcore/global/company", "knowledge", "security-policy.rule.md",
		"---\ntitle: \"Security Policy\"\nstatus: accepted\n---\n\nbody\n")

	// Global: platform source.
	writeGlobalDoc(t, base, ".archcore/global/platform", "knowledge", "api-conventions.rule.md",
		"---\ntitle: \"API Conventions\"\nstatus: accepted\n---\n\nbody\n")

	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "company", Path: ".archcore/global/company"},
		{ID: "platform", Path: ".archcore/global/platform"},
	})

	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatalf("ScanDocuments: %v", err)
	}
	if len(docs) != 5 {
		t.Fatalf("want 5 docs, got %d", len(docs))
	}

	byPath := make(map[string]LocalDocument, len(docs))
	for _, d := range docs {
		byPath[d.Path] = d
	}

	// Local docs must be writable.
	for _, p := range []string{
		".archcore/knowledge/local-only.rule.md",
		".archcore/knowledge/react-query.rule.md",
	} {
		d, ok := byPath[p]
		if !ok {
			t.Errorf("missing local doc %s", p)
			continue
		}
		if d.SourceKind != "local" {
			t.Errorf("%s: want source_kind=local, got %q", p, d.SourceKind)
		}
		if d.SourceID != "local" {
			t.Errorf("%s: want source_id=local, got %q", p, d.SourceID)
		}
		if d.ReadOnly {
			t.Errorf("%s: want read_only=false", p)
		}
	}

	// Company global docs must be read-only with correct source id.
	for _, p := range []string{
		".archcore/global/company/knowledge/react-query.rule.md",
		".archcore/global/company/knowledge/security-policy.rule.md",
	} {
		d, ok := byPath[p]
		if !ok {
			t.Errorf("missing company doc %s", p)
			continue
		}
		if d.SourceKind != "global" {
			t.Errorf("%s: want source_kind=global, got %q", p, d.SourceKind)
		}
		if d.SourceID != "company" {
			t.Errorf("%s: want source_id=company, got %q", p, d.SourceID)
		}
		if !d.ReadOnly {
			t.Errorf("%s: want read_only=true", p)
		}
	}

	// Platform global doc.
	p := ".archcore/global/platform/knowledge/api-conventions.rule.md"
	d, ok := byPath[p]
	if !ok {
		t.Errorf("missing platform doc %s", p)
	} else {
		if d.SourceID != "platform" {
			t.Errorf("%s: want source_id=platform, got %q", p, d.SourceID)
		}
		if !d.ReadOnly {
			t.Errorf("%s: want read_only=true", p)
		}
	}
}

func TestScanDocuments_SkipsGlobalDirWithoutDeclaration(t *testing.T) {
	base := setupTestArchcore(t)

	// Local doc.
	writeDoc(t, base, "knowledge", "local.rule.md",
		"---\ntitle: \"Local\"\nstatus: accepted\n---\n\nbody\n")

	// Global tree present on disk but NOT declared in settings.
	writeGlobalDoc(t, base, ".archcore/global/company", "knowledge", "company.rule.md",
		"---\ntitle: \"Company\"\nstatus: accepted\n---\n\nbody\n")

	// No settings.json → no globals declared.

	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatalf("ScanDocuments: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("want 1 doc (global/ skipped), got %d", len(docs))
	}
	if len(docs) > 0 && docs[0].SourceKind != "local" {
		t.Errorf("doc should be local, got source_kind=%q", docs[0].SourceKind)
	}
}

func TestCreateDocument_RejectsGlobalPath(t *testing.T) {
	base := setupTestArchcore(t)

	// Declare company so the write guard recognises the path as global.
	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "company", Path: ".archcore/global/company"},
	})

	result, err := callTool(HandleCreateDocument(base), map[string]any{
		"type":      "rule",
		"filename":  "injected",
		"directory": "global/company/knowledge",
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for global path, got success")
	}
}

func TestUpdateDocument_RejectsGlobalPath(t *testing.T) {
	base := setupTestArchcore(t)

	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "company", Path: ".archcore/global/company"},
	})
	writeGlobalDoc(t, base, ".archcore/global/company", "knowledge", "react-query.rule.md",
		"---\ntitle: \"React Query (Company)\"\nstatus: accepted\n---\n\nbody\n")

	result, err := callTool(HandleUpdateDocument(base), map[string]any{
		"path":  ".archcore/global/company/knowledge/react-query.rule.md",
		"title": "Hijacked",
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result updating a global doc, got success")
	}
	// In-tree global path reaches the read-only guard directly, so the message is
	// the clean read-only one (spec §5.5), not a path-validation error.
	if got := result.Content[0].(mcp.TextContent).Text; got != "cannot update a read-only global source document" {
		t.Errorf("message = %q, want clean read-only message (spec §5.5)", got)
	}
}

func TestRemoveDocument_RejectsGlobalPath(t *testing.T) {
	base := setupTestArchcore(t)

	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "company", Path: ".archcore/global/company"},
	})
	writeGlobalDoc(t, base, ".archcore/global/company", "knowledge", "security-policy.rule.md",
		"---\ntitle: \"Security Policy\"\nstatus: accepted\n---\n\nbody\n")

	result, err := callTool(HandleRemoveDocument(base), map[string]any{
		"path": ".archcore/global/company/knowledge/security-policy.rule.md",
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result removing a global doc, got success")
	}
}

// TestCreateDocument_FailsClosedOnUnreadableSettings verifies the write guard
// rejects rather than silently allows a write when settings.json is present but
// unparseable — otherwise a corrupt config could let a write slip into a global.
func TestCreateDocument_FailsClosedOnUnreadableSettings(t *testing.T) {
	base := setupTestArchcore(t)

	// Corrupt settings.json (present but invalid JSON).
	settingsPath := filepath.Join(base, ".archcore", "settings.json")
	if err := os.WriteFile(settingsPath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := callTool(HandleCreateDocument(base), map[string]any{
		"type":      "rule",
		"filename":  "doc",
		"directory": "knowledge",
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result when settings.json is unreadable, got success")
	}
}

func TestIsGlobalPath_LookupBased(t *testing.T) {
	baseDir := t.TempDir()
	globals := []config.GlobalSource{
		{ID: "company", Path: ".archcore/global/company"},
		{ID: "platform", Path: "vendor/platform"},
	}
	tests := []struct {
		name    string
		relPath string
		want    bool
	}{
		{"embedded global doc", ".archcore/global/company/knowledge/doc.rule.md", true},
		{"embedded global nested doc", ".archcore/global/company/knowledge/nested/doc.rule.md", true},
		{"external global doc", "vendor/platform/knowledge/api.rule.md", true},
		{"local doc", ".archcore/knowledge/local.rule.md", false},
		{"undeclared source", ".archcore/global/other/knowledge/doc.rule.md", false},
		{"exact match on source root", ".archcore/global/company", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGlobalPath(baseDir, tt.relPath, globals); got != tt.want {
				t.Errorf("isGlobalPath(%q) = %v, want %v", tt.relPath, got, tt.want)
			}
		})
	}
}

func TestScanDocuments_WithExternalGlobal(t *testing.T) {
	base := setupTestArchcore(t)

	writeDoc(t, base, "knowledge", "local.rule.md",
		"---\ntitle: \"Local\"\nstatus: accepted\n---\n\nbody\n")

	// External global: lives at vendor/platform, outside .archcore/.
	writeGlobalDoc(t, base, "vendor/platform", "knowledge", "api-conventions.rule.md",
		"---\ntitle: \"API Conventions\"\nstatus: accepted\n---\n\nbody\n")

	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "platform", Path: "vendor/platform"},
	})

	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatalf("ScanDocuments: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("want 2 docs (1 local + 1 external global), got %d", len(docs))
	}

	byPath := make(map[string]LocalDocument, len(docs))
	for _, d := range docs {
		byPath[d.Path] = d
	}

	p := "vendor/platform/knowledge/api-conventions.rule.md"
	d, ok := byPath[p]
	if !ok {
		t.Fatalf("missing external global doc %s", p)
	}
	if d.SourceID != "platform" {
		t.Errorf("source_id = %q, want %q", d.SourceID, "platform")
	}
	if d.SourceKind != "global" {
		t.Errorf("source_kind = %q, want %q", d.SourceKind, "global")
	}
	if !d.ReadOnly {
		t.Error("want read_only=true for external global")
	}
}

func TestScanDocuments_MissingGlobalFails(t *testing.T) {
	base := setupTestArchcore(t)

	writeDoc(t, base, "knowledge", "local.rule.md",
		"---\ntitle: \"Local\"\nstatus: accepted\n---\n\nbody\n")

	// Every declared global is mandatory; a missing one fails the scan.
	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "ghost", Path: ".archcore/global/ghost"},
	})

	_, err := ScanDocuments(base)
	if err == nil {
		t.Fatal("ScanDocuments should fail for a missing global source")
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("error %q should mention the global id", err.Error())
	}
}

// TestScanDocuments_WithRelativeExternalGlobal verifies that a global source
// declared in settings.json with a "../"-relative path pointing at a sibling
// project's .archcore directory is scanned read-only.
func TestScanDocuments_WithRelativeExternalGlobal(t *testing.T) {
	// Shared parent so base and the sibling global live side by side.
	parent := t.TempDir()
	base := filepath.Join(parent, "primary")
	if err := os.MkdirAll(filepath.Join(base, ".archcore"), 0o755); err != nil {
		t.Fatal(err)
	}

	writeDoc(t, base, "knowledge", "local.rule.md",
		"---\ntitle: \"Local Rule\"\nstatus: accepted\n---\n\nbody\n")

	// Sibling global project: parent/company-global/.archcore/knowledge/*.rule.md.
	globalArchcore := filepath.Join(parent, "company-global", ".archcore")
	if err := os.MkdirAll(filepath.Join(globalArchcore, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, fname := range []string{"error-handling.rule.md", "git-commits.rule.md"} {
		content := "---\ntitle: \"" + fname + "\"\nstatus: accepted\n---\n\nbody\n"
		if err := os.WriteFile(
			filepath.Join(globalArchcore, "knowledge", fname),
			[]byte(content), 0o644,
		); err != nil {
			t.Fatal(err)
		}
	}

	// Declare the sibling via a "../"-relative path in settings.json.
	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "company-global", Path: "../company-global/.archcore"},
	})

	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatalf("ScanDocuments: %v", err)
	}

	// Expect 1 local + 2 external global docs.
	if len(docs) != 3 {
		t.Fatalf("want 3 docs (1 local + 2 external global), got %d", len(docs))
	}

	// Local doc must be writable.
	var foundLocal bool
	globalDocCount := 0
	for _, d := range docs {
		if d.SourceKind == "local" {
			foundLocal = true
			if d.ReadOnly {
				t.Errorf("local doc %s: want read_only=false", d.Path)
			}
			continue
		}
		globalDocCount++
		if d.SourceID != "company-global" {
			t.Errorf("global doc %s: source_id = %q, want company-global", d.Path, d.SourceID)
		}
		if !d.ReadOnly {
			t.Errorf("global doc %s: want read_only=true", d.Path)
		}
		if !d.Global {
			t.Errorf("global doc %s: want global=true", d.Path)
		}
	}
	if !foundLocal {
		t.Error("missing local doc")
	}
	if globalDocCount != 2 {
		t.Errorf("want 2 global docs, got %d", globalDocCount)
	}
}

// TestScanDocuments_InTreeGlobalOutsideReservedDir_NotDoubleScanned guards the
// dedup fix: a global vendored in-tree under a directory NOT named "global" (e.g.
// the plausible plural "globals/" typo) must be scanned once — as a read-only
// global, never also as a writable local — so it carries exactly one source_id.
func TestScanDocuments_InTreeGlobalOutsideReservedDir_NotDoubleScanned(t *testing.T) {
	base := setupTestArchcore(t)

	writeDoc(t, base, "knowledge", "local.rule.md",
		"---\ntitle: \"Local\"\nstatus: accepted\n---\n\nbody\n")

	// In-tree global under a NON-reserved directory name ("globals", not "global").
	writeGlobalDoc(t, base, ".archcore/globals/company", "knowledge", "company.rule.md",
		"---\ntitle: \"Company\"\nstatus: accepted\n---\n\nbody\n")
	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "company", Path: ".archcore/globals/company"},
	})

	docs, err := ScanDocuments(base)
	if err != nil {
		t.Fatalf("ScanDocuments: %v", err)
	}

	const companyPath = ".archcore/globals/company/knowledge/company.rule.md"
	count := 0
	for _, d := range docs {
		if d.Path == companyPath {
			count++
			if d.SourceKind != "global" {
				t.Errorf("%s: source_kind = %q, want global", d.Path, d.SourceKind)
			}
			if !d.ReadOnly {
				t.Errorf("%s: want read_only=true", d.Path)
			}
		}
	}
	if count != 1 {
		t.Fatalf("in-tree global scanned %d time(s), want exactly 1", count)
	}
	if len(docs) != 2 { // 1 local + 1 global
		t.Errorf("want 2 docs total, got %d", len(docs))
	}
}

// TestAddRelation_RejectsGlobalEndpoint verifies a relation may never touch a
// global document, on either endpoint — whether declared, or merely sitting in the
// reserved .archcore/global/ tree without a declaration.
func TestAddRelation_RejectsGlobalEndpoint(t *testing.T) {
	base := setupTestArchcore(t)

	writeDoc(t, base, "knowledge", "local.rule.md",
		"---\ntitle: \"Local\"\nstatus: accepted\n---\n\nbody\n")
	writeGlobalDoc(t, base, ".archcore/global/company", "knowledge", "std.rule.md",
		"---\ntitle: \"Std\"\nstatus: accepted\n---\n\nbody\n")
	// A doc physically present in the reserved tree but NOT declared in settings.
	writeGlobalDoc(t, base, ".archcore/global/elsewhere", "knowledge", "x.rule.md",
		"---\ntitle: \"X\"\nstatus: accepted\n---\n\nbody\n")
	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "company", Path: ".archcore/global/company"},
	})

	const (
		local      = ".archcore/knowledge/local.rule.md"
		global     = ".archcore/global/company/knowledge/std.rule.md"
		undeclared = ".archcore/global/elsewhere/knowledge/x.rule.md"
	)
	cases := []struct {
		name           string
		source, target string
	}{
		{"global as target", local, global},
		{"global as source", global, local},
		{"undeclared reserved-dir target", local, undeclared},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := callTool(HandleAddRelation(base), map[string]any{
				"source": tc.source, "target": tc.target, "type": "related",
			})
			if err != nil {
				t.Fatalf("transport error: %v", err)
			}
			if !res.IsError {
				t.Errorf("expected error relating %q -> %q, got success", tc.source, tc.target)
			}
		})
	}
}

// TestCreateDocument_RejectsReservedGlobalDir verifies the reserved .archcore/global/
// tree is write-protected even with nothing declared — otherwise create_document
// would write an invisible document into global mount space.
func TestCreateDocument_RejectsReservedGlobalDir(t *testing.T) {
	base := setupTestArchcore(t)

	result, err := callTool(HandleCreateDocument(base), map[string]any{
		"type":      "rule",
		"filename":  "sneaky",
		"directory": "global/foo",
	})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error creating under reserved .archcore/global/, got success")
	}
}

// TestUpdateDocument_FailsClosedOnUnreadableSettings mirrors the create_document
// fail-closed guard: a corrupt settings.json must block the write rather than let
// it proceed unverified.
func TestUpdateDocument_FailsClosedOnUnreadableSettings(t *testing.T) {
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "doc.rule.md",
		"---\ntitle: \"Doc\"\nstatus: accepted\n---\n\nbody\n")
	settingsPath := filepath.Join(base, ".archcore", "settings.json")
	if err := os.WriteFile(settingsPath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := callTool(HandleUpdateDocument(base), map[string]any{
		"path":  ".archcore/knowledge/doc.rule.md",
		"title": "New",
	})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error when settings.json is unreadable, got success")
	}
}

// TestRemoveDocument_FailsClosedOnUnreadableSettings is the remove_document analogue.
func TestRemoveDocument_FailsClosedOnUnreadableSettings(t *testing.T) {
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "doc.rule.md",
		"---\ntitle: \"Doc\"\nstatus: accepted\n---\n\nbody\n")
	settingsPath := filepath.Join(base, ".archcore", "settings.json")
	if err := os.WriteFile(settingsPath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := callTool(HandleRemoveDocument(base), map[string]any{
		"path": ".archcore/knowledge/doc.rule.md",
	})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error when settings.json is unreadable, got success")
	}
}

// TestValidateReadPath covers the read-path relaxation that lets get_document read
// a document inside a declared external global source, with its defense-in-depth
// guards (relative-only, .md-only, lexical containment, symlink containment).
func TestValidateReadPath(t *testing.T) {
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "local.rule.md",
		"---\ntitle: \"Local\"\nstatus: accepted\n---\n\nbody\n")
	// External global at base/vendor/platform (outside .archcore/): its document
	// paths do not start with ".archcore/", so they take the relaxed branch.
	writeGlobalDoc(t, base, "vendor/platform", "knowledge", "api.rule.md",
		"---\ntitle: \"API\"\nstatus: accepted\n---\n\nbody\n")
	globals := []config.GlobalSource{{ID: "platform", Path: "vendor/platform"}}

	const globalDoc = "vendor/platform/knowledge/api.rule.md"

	t.Run("local path passes through unchanged", func(t *testing.T) {
		got, err := validateReadPath(base, ".archcore/knowledge/local.rule.md", globals)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != ".archcore/knowledge/local.rule.md" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("external global document is readable", func(t *testing.T) {
		got, err := validateReadPath(base, globalDoc, globals)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != globalDoc {
			t.Errorf("got %q, want %q", got, globalDoc)
		}
	})

	t.Run("traversal escape outside the global is rejected", func(t *testing.T) {
		// Must be rejected by the containment guard — NOT merely because the
		// escaped path is absent on disk (fs.ErrNotExist would also satisfy
		// err != nil and would hide a removed containment check).
		_, err := validateReadPath(base, "vendor/platform/knowledge/../../../escape.rule.md", globals)
		if err == nil || errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("want a containment-guard rejection, got %v", err)
		}
	})

	t.Run("non-.md under the global is rejected", func(t *testing.T) {
		if _, err := validateReadPath(base, "vendor/platform/settings.json", globals); err == nil {
			t.Fatal("expected rejection of non-.md path")
		}
	})

	t.Run("absolute path is rejected", func(t *testing.T) {
		if _, err := validateReadPath(base, "/etc/passwd.md", globals); err == nil {
			t.Fatal("expected rejection of absolute path")
		}
	})

	t.Run("path under no declared global is rejected", func(t *testing.T) {
		// Guard rejection, not an incidental filesystem miss.
		_, err := validateReadPath(base, "vendor/other/api.rule.md", globals)
		if err == nil || errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("want a guard rejection, got %v", err)
		}
	})

	t.Run("missing document under the global reports not-found", func(t *testing.T) {
		_, err := validateReadPath(base, "vendor/platform/knowledge/missing.rule.md", globals)
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("want fs.ErrNotExist, got %v", err)
		}
	})

	t.Run("symlink escaping the global is rejected", func(t *testing.T) {
		secret := filepath.Join(base, "secret.rule.md")
		if err := os.WriteFile(secret, []byte("top secret"), 0o644); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(base, "vendor", "platform", "knowledge", "evil.rule.md")
		if err := os.Symlink(secret, link); err != nil {
			t.Skipf("symlinks unsupported on this platform: %v", err)
		}
		if _, err := validateReadPath(base, "vendor/platform/knowledge/evil.rule.md", globals); err == nil {
			t.Fatal("expected rejection of a symlink escaping the global root")
		}
	})
}

// TestValidateReadPath_MultipleGlobals exercises the read relaxation when MORE THAN ONE
// global is declared — the case examples/06 relies on and that every other read-path
// test misses (each declares exactly one). It also covers the deep ("../../") and
// bare-".archcore"-root external shapes from examples/09 and /10. The two externals sit
// at indices 1 and 2, so a "matches only the first global" bug fails the readable cases;
// the rejection rows prove the loop still blocks escapes with three globals in play.
func TestValidateReadPath_MultipleGlobals(t *testing.T) {
	// Parent-temp layout: base is two levels deep so "../../" resolves inside the temp
	// tree (parent), not outside it.
	parent := t.TempDir()
	base := filepath.Join(parent, "apps", "api")
	if err := os.MkdirAll(filepath.Join(base, ".archcore"), 0o755); err != nil {
		t.Fatal(err)
	}

	const fm = "---\ntitle: Doc\nstatus: accepted\n---\n\nbody\n"

	// Local doc + in-tree global "company" (idx0).
	writeDoc(t, base, "knowledge", "local.rule.md", fm)
	writeGlobalDoc(t, base, ".archcore/global/company", "knowledge", "react-query.rule.md", fm)

	// External global roots must exist on disk before the table loop — validateReadPath
	// runs EvalSymlinks on both root and target.
	writeFile := func(p string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(fm), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// "monorepo" (idx1): bare-.archcore root at ../../.archcore.
	writeFile(filepath.Join(parent, ".archcore", "knowledge", "monorepo-std.rule.md"))
	// "shared" (idx2): deep root at ../../packages/shared-standards/.archcore.
	writeFile(filepath.Join(parent, "packages", "shared-standards", ".archcore", "knowledge", "shared-std.rule.md"))

	globals := []config.GlobalSource{
		{ID: "company", Path: ".archcore/global/company"},
		{ID: "monorepo", Path: "../../.archcore"},
		{ID: "shared", Path: "../../packages/shared-standards/.archcore"},
	}

	tests := []struct {
		name     string
		relPath  string
		wantPath string
		wantErr  bool
		notFound bool
	}{
		{
			name:     "in-tree company readable",
			relPath:  ".archcore/global/company/knowledge/react-query.rule.md",
			wantPath: ".archcore/global/company/knowledge/react-query.rule.md",
		},
		{
			name:     "bare-.archcore external (idx1) readable",
			relPath:  "../../.archcore/knowledge/monorepo-std.rule.md",
			wantPath: "../../.archcore/knowledge/monorepo-std.rule.md",
		},
		{
			name:     "deep external (idx2) readable",
			relPath:  "../../packages/shared-standards/.archcore/knowledge/shared-std.rule.md",
			wantPath: "../../packages/shared-standards/.archcore/knowledge/shared-std.rule.md",
		},
		{
			name:     "missing under idx1 reports not-found",
			relPath:  "../../.archcore/knowledge/ghost.rule.md",
			notFound: true,
		},
		{
			name:     "missing under idx2 reports not-found",
			relPath:  "../../packages/shared-standards/.archcore/knowledge/ghost.rule.md",
			notFound: true,
		},
		{
			name:    "traversal escape past idx2 rejected",
			relPath: "../../packages/shared-standards/.archcore/knowledge/../../../../../../etc/secret.rule.md",
			wantErr: true,
		},
		{
			name:    "path under no declared global rejected",
			relPath: "../../packages/other/.archcore/x.rule.md",
			wantErr: true,
		},
		{
			name:    "non-.md under a global rejected",
			relPath: "../../.archcore/knowledge/settings.json",
			wantErr: true,
		},
		{
			name:    "absolute path rejected",
			relPath: "/etc/passwd.md",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateReadPath(base, tt.relPath, globals)
			switch {
			case tt.notFound:
				if !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("want fs.ErrNotExist, got %v", err)
				}
			case tt.wantErr:
				if err == nil {
					t.Fatalf("want error, got path %q", got)
				}
			default:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.wantPath {
					t.Errorf("got %q, want %q", got, tt.wantPath)
				}
			}
			// No-absolute-path fence: any non-sentinel error must never embed the temp root.
			if err != nil && !errors.Is(err, fs.ErrNotExist) && strings.Contains(err.Error(), parent) {
				t.Errorf("error message leaked an absolute path: %v", err)
			}
		})
	}
}
