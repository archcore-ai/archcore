package tools

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/config"
	"archcore-cli/internal/docs"

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

	docs, err := scanDocuments(base)
	if err != nil {
		t.Fatalf("scanDocuments: %v", err)
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
	if !(d.SourceKind == "global") {
		t.Error("want read_only=true for external global")
	}
}

// TestScanDocuments_MissingGlobalSkipped pins missing-global-degrades-to-local.adr:
// a global that is not cloned yet contributes nothing, while the local documents
// and a healthy global declared after it still come back.
func TestScanDocuments_MissingGlobalSkipped(t *testing.T) {
	base := setupTestArchcore(t)

	writeDoc(t, base, "knowledge", "local.rule.md",
		"---\ntitle: \"Local\"\nstatus: accepted\n---\n\nbody\n")
	writeGlobalDoc(t, base, "vendor/company", "knowledge", "shared.rule.md",
		"---\ntitle: \"Shared\"\nstatus: accepted\n---\n\nbody\n")
	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "ghost", Path: "vendor/ghost"},
		{ID: "company", Path: "vendor/company"},
	})

	docs, err := scanDocuments(base)
	if err != nil {
		t.Fatalf("scanDocuments should skip a missing global source, got %v", err)
	}
	bySource := make(map[string]int)
	for _, doc := range docs {
		bySource[doc.SourceID]++
	}
	if len(docs) != 2 || bySource["local"] != 1 || bySource["company"] != 1 {
		t.Errorf("scanDocuments = %+v, want the local document and the company document", docs)
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

	docs, err := scanDocuments(base)
	if err != nil {
		t.Fatalf("scanDocuments: %v", err)
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
			if d.SourceKind == "global" {
				t.Errorf("local doc %s: want read_only=false", d.Path)
			}
			continue
		}
		globalDocCount++
		if d.SourceID != "company-global" {
			t.Errorf("global doc %s: source_id = %q, want company-global", d.Path, d.SourceID)
		}
		if !(d.SourceKind == "global") {
			t.Errorf("global doc %s: want read_only=true", d.Path)
		}
		if !(d.SourceKind == "global") {
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
	writeGlobalDoc(t, base, "vendor/company", "knowledge", "react-query.rule.md", fm)

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
		{ID: "company", Path: "vendor/company"},
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
			relPath:  "vendor/company/knowledge/react-query.rule.md",
			wantPath: "vendor/company/knowledge/react-query.rule.md",
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

// setupSourceLayout builds root/app (the consumer) declaring two sources that sit
// beside it, root/company and root/platform, each with its own .archcore/.
func setupSourceLayout(t *testing.T) (base, root string) {
	t.Helper()
	root = t.TempDir()
	base = filepath.Join(root, "app")
	if err := os.MkdirAll(filepath.Join(base, ".archcore"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "company", Path: "../company/.archcore"},
		{ID: "platform", Path: "../platform/.archcore"},
	})
	const fm = "---\ntitle: \"Doc\"\nstatus: accepted\n---\n\nbody\n"
	writeDoc(t, base, "knowledge", "local-only.rule.md", fm)
	writeDoc(t, base, "knowledge", "react-query.rule.md", fm)
	writeGlobalDoc(t, root, "company/.archcore", "knowledge", "react-query.rule.md", fm)
	writeGlobalDoc(t, root, "company/.archcore", "knowledge", "security-policy.rule.md", fm)
	writeGlobalDoc(t, root, "platform/.archcore", "knowledge", "api-conventions.rule.md", fm)
	return base, root
}

func TestScanDocuments_WithGlobals(t *testing.T) {
	base, root := setupSourceLayout(t)

	got, err := scanDocuments(base)
	if err != nil {
		t.Fatalf("scanDocuments: %v", err)
	}
	want := map[string][3]string{ // path -> source_kind, source_id, source_root
		".archcore/knowledge/local-only.rule.md":                  {"local", "local", ""},
		".archcore/knowledge/react-query.rule.md":                 {"local", "local", ""},
		"../company/.archcore/knowledge/react-query.rule.md":      {"global", "company", filepath.Join(root, "company")},
		"../company/.archcore/knowledge/security-policy.rule.md":  {"global", "company", filepath.Join(root, "company")},
		"../platform/.archcore/knowledge/api-conventions.rule.md": {"global", "platform", filepath.Join(root, "platform")},
	}
	if len(got) != len(want) {
		t.Fatalf("want %d docs, got %d", len(want), len(got))
	}
	for _, d := range got {
		w, ok := want[d.Path]
		if !ok {
			t.Errorf("unexpected doc %s", d.Path)
			continue
		}
		if have := [3]string{string(d.SourceKind), d.SourceID, d.SourceRoot}; have != w {
			t.Errorf("%s: annotation = %v, want %v", d.Path, have, w)
		}
	}
}

func TestScanDocuments_UndeclaredGlobalDirIsLocal(t *testing.T) {
	base := setupTestArchcore(t)
	writeDoc(t, base, "knowledge", "local.rule.md",
		"---\ntitle: \"Local\"\nstatus: accepted\n---\n\nbody\n")
	writeGlobalDoc(t, base, ".archcore/global/company", "knowledge", "company.rule.md",
		"---\ntitle: \"Company\"\nstatus: accepted\n---\n\nbody\n")

	got, err := scanDocuments(base)
	if err != nil {
		t.Fatalf("scanDocuments: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 local docs, got %d", len(got))
	}
	for _, d := range got {
		if d.SourceKind != "local" {
			t.Errorf("%s: source_kind = %q, want local", d.Path, d.SourceKind)
		}
	}
}

func TestScanDocuments_SourceInsideArchcoreIsFatal(t *testing.T) {
	base := setupTestArchcore(t)
	writeGlobalDoc(t, base, ".archcore/globals/company", "knowledge", "company.rule.md",
		"---\ntitle: \"Company\"\nstatus: accepted\n---\n\nbody\n")
	writeGlobalsSettings(t, base, []config.GlobalSource{
		{ID: "company", Path: ".archcore/globals/company"},
	})

	_, err := scanDocuments(base)
	if err == nil || !strings.Contains(err.Error(), "overlaps the project's own .archcore") {
		t.Fatalf("want the self-overlap error, got %v", err)
	}

	// The local-only surfaces (hooks, status) do not count the misdeclared
	// source as the project's own documents while the user migrates it.
	local, err := docs.ScanLocal(base, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range local {
		if strings.Contains(d.Path, "globals/company") {
			t.Errorf("ScanLocal returned the misdeclared source document %s as local", d.Path)
		}
	}
}

// TestWriteTools_RefuseOtherProjectPath pins that a write addressed through a
// source row path is refused with the instruction to pass project, and names the
// source (source-mounts.spec, failure rules).
func TestWriteTools_RefuseOtherProjectPath(t *testing.T) {
	base, _ := setupSourceLayout(t)
	const path = "../company/.archcore/knowledge/react-query.rule.md"

	cases := map[string]struct {
		handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
		args    map[string]any
	}{
		"update_document": {HandleUpdateDocument(StaticRoot(base)), map[string]any{"path": path, "title": "Hijacked"}},
		"remove_document": {HandleRemoveDocument(StaticRoot(base)), map[string]any{"path": path}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := callTool(tc.handler, tc.args)
			if err != nil {
				t.Fatalf("transport error: %v", err)
			}
			if !result.IsError {
				t.Fatal("want an error result, got success")
			}
			text := result.Content[0].(mcp.TextContent).Text
			if !strings.Contains(text, "pass that project's absolute root as project") {
				t.Errorf("message = %q, want the pass-project instruction", text)
			}
			if !strings.Contains(text, `source "company"`) {
				t.Errorf("message = %q, want the source id", text)
			}
		})
	}
	if _, err := os.Stat(filepath.Join(base, "..", "company", ".archcore", "knowledge", "react-query.rule.md")); err != nil {
		t.Fatalf("source document must survive: %v", err)
	}
}

// TestWriteTools_WriteSourceThroughItsRoot pins that a source document is
// writable through its own root, with relations inside that project.
func TestWriteTools_WriteSourceThroughItsRoot(t *testing.T) {
	_, root := setupSourceLayout(t)
	company := StaticRoot(filepath.Join(root, "company"))

	mustCall(t, HandleUpdateDocument(company), map[string]any{
		"path": ".archcore/knowledge/react-query.rule.md", "title": "Renamed",
	})
	mustCall(t, HandleAddRelation(company), map[string]any{
		"source": ".archcore/knowledge/react-query.rule.md",
		"target": ".archcore/knowledge/security-policy.rule.md",
		"type":   "related",
	})
	mustCall(t, HandleRemoveDocument(company), map[string]any{"path": ".archcore/knowledge/security-policy.rule.md"})
}

// TestAddRelation_RejectsCrossProjectEndpoint pins that a relation stays inside
// one project: a source row path is never an endpoint.
func TestAddRelation_RejectsCrossProjectEndpoint(t *testing.T) {
	base, _ := setupSourceLayout(t)
	const (
		local  = ".archcore/knowledge/local-only.rule.md"
		global = "../company/.archcore/knowledge/react-query.rule.md"
	)
	for name, pair := range map[string][2]string{
		"global as target": {local, global},
		"global as source": {global, local},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := callTool(HandleAddRelation(StaticRoot(base)), map[string]any{
				"source": pair[0], "target": pair[1], "type": "related",
			})
			if err != nil {
				t.Fatalf("transport error: %v", err)
			}
			if !res.IsError {
				t.Errorf("expected error relating %q -> %q, got success", pair[0], pair[1])
			}
		})
	}
}
