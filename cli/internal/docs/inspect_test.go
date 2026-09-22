package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/templates"
)

// TestInspectGlobals_NestedTraversalFailure pins the invariant the startup gate
// rests on: InspectGlobals and the runtime scan classify a source identically
// (global-sources.spec).
//
// config.CheckGlobalDir opens only the top directory, so a subdirectory nobody
// can enter passes it. The document walk is where that surfaces — and while its
// error was discarded, the count came back partial and the source was reported
// GlobalOK or GlobalEmpty. The MCP server then started on a source whose every
// read failed, and `status` printed a healthy line for it.
func TestInspectGlobals_NestedTraversalFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: file permissions are not enforced")
	}
	t.Parallel()

	tests := []struct {
		name string
		// docs written into the global source before the directory is locked.
		// The empty case reproduced GlobalEmpty, the populated one GlobalOK.
		docs []string
	}{
		{name: "no readable documents"},
		{name: "one readable document", docs: []string{"a.adr.md"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base, gsDir := externalGlobalFixture(t)
			for _, name := range tt.docs {
				writeFile(t, filepath.Join(gsDir, name), "---\ntitle: A\nstatus: accepted\n---\nBody.\n")
			}

			locked := filepath.Join(gsDir, "locked")
			if err := os.MkdirAll(locked, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(locked, 0o000); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(locked, 0o755) }) // so TempDir cleanup works

			inspections, err := InspectGlobals(base)
			if err != nil {
				t.Fatalf("InspectGlobals: %v", err)
			}
			if len(inspections) != 1 {
				t.Fatalf("got %d inspections, want 1", len(inspections))
			}

			in := inspections[0]
			if in.State != GlobalUnreadable {
				t.Errorf("State = %v, want GlobalUnreadable", in.State)
			}
			if !in.State.Fatal() {
				t.Error("an unreadable source must be fatal — the server would start and then fail every read")
			}
			if msg := in.Message(); !strings.Contains(msg, "not readable") {
				t.Errorf("Message() = %q, want it to name the source unreadable", msg)
			}

			// The other half of the invariant: the scan refuses the same source.
			if _, scanErr := Scan(base); scanErr == nil {
				t.Error("Scan accepted a source InspectGlobals classified unreadable")
			}
		})
	}
}

// TestInspectGlobals_ReadableSourceIsOK is the control: the same fixture without
// the locked directory classifies as OK and counts its documents, so the test
// above fails on the traversal error rather than on the fixture.
func TestInspectGlobals_ReadableSourceIsOK(t *testing.T) {
	t.Parallel()
	base, gsDir := externalGlobalFixture(t)
	writeFile(t, filepath.Join(gsDir, "a.adr.md"), "---\ntitle: A\nstatus: accepted\n---\nBody.\n")
	if err := os.MkdirAll(filepath.Join(gsDir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(gsDir, "nested", "b.rule.md"), "---\ntitle: B\nstatus: accepted\n---\nBody.\n")

	inspections, err := InspectGlobals(base)
	if err != nil {
		t.Fatalf("InspectGlobals: %v", err)
	}
	if len(inspections) != 1 {
		t.Fatalf("got %d inspections, want 1", len(inspections))
	}
	if in := inspections[0]; in.State != GlobalOK || in.Docs != 2 {
		t.Errorf("State = %v, Docs = %d; want GlobalOK with 2 documents", in.State, in.Docs)
	}
}

// TestInspectGlobals_CountBreakdowns pins the filename-derived breakdowns the
// session-start disclosure renders (session-globals-disclosure.spec): category
// from the type suffix, top-level directory from the path, root-level documents
// counted in the total but under no directory.
func TestInspectGlobals_CountBreakdowns(t *testing.T) {
	t.Parallel()
	base, gsDir := externalGlobalFixture(t)
	docBody := "---\ntitle: T\nstatus: accepted\n---\nBody.\n"
	for _, p := range []string{
		"root.adr.md",              // knowledge, no directory entry
		"concepts/a.rule.md",       // knowledge
		"concepts/b.doc.md",        // knowledge
		"product/c.idea.md",        // vision
		"product/nested/d.plan.md", // vision, still counted under product/
		"process/e.task-type.md",   // experience
		"concepts/skip.txt",        // not a document
		"concepts/no-type.md",      // unrecognized type: not mounted
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(gsDir, p)), 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(gsDir, p), docBody)
	}

	inspections, err := InspectGlobals(base)
	if err != nil {
		t.Fatalf("InspectGlobals: %v", err)
	}
	if len(inspections) != 1 {
		t.Fatalf("got %d inspections, want 1", len(inspections))
	}
	in := inspections[0]
	if in.Docs != 6 {
		t.Errorf("Docs = %d, want 6", in.Docs)
	}
	wantCategories := map[templates.Category]int{
		templates.CategoryKnowledge:  3,
		templates.CategoryVision:     2,
		templates.CategoryExperience: 1,
	}
	for cat, want := range wantCategories {
		if got := in.DocsByCategory[cat]; got != want {
			t.Errorf("DocsByCategory[%s] = %d, want %d", cat, got, want)
		}
	}
	wantDirs := map[string]int{"concepts": 2, "product": 2, "process": 1}
	if len(in.TopDirs) != len(wantDirs) {
		t.Errorf("TopDirs = %v, want %v", in.TopDirs, wantDirs)
	}
	for dir, want := range wantDirs {
		if got := in.TopDirs[dir]; got != want {
			t.Errorf("TopDirs[%s] = %d, want %d", dir, got, want)
		}
	}
}

// externalGlobalFixture builds a project declaring one global source mounted
// from a sibling directory, and returns the project root and the resolved
// source directory.
func externalGlobalFixture(t *testing.T) (base, gsDir string) {
	t.Helper()
	root := t.TempDir()
	base = filepath.Join(root, "project")
	gsDir = filepath.Join(root, "company", ".archcore")
	if err := os.MkdirAll(filepath.Join(base, ".archcore"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(gsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(base, ".archcore", "settings.json"),
		`{"sync":"none","globals":[{"id":"company","path":"../company/.archcore"}]}`)
	return base, gsDir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
