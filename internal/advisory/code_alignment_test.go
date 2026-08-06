package advisory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Case names mirror test/unit/check-code-alignment.bats in the plugin.

func writeAlignmentDoc(t *testing.T, base, relPath, title, body string) {
	t.Helper()
	full := filepath.Join(base, ".archcore", filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf("---\ntitle: %q\nstatus: accepted\n---\n\n%s\n", title, body)
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCodeAlignment_SilentCases(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		setup    func(t *testing.T, base string)
		env      map[string]string
	}{
		{name: "no file_path", filePath: ""},
		{name: "non-source-root path", filePath: "docs/readme.md"},
		{name: ".archcore/*.md path", filePath: ".archcore/knowledge/a.adr.md"},
		{name: "a file named like a root but not inside one", filePath: "src"},
		{
			name:     "escape hatch disables injection",
			filePath: "src/api/handlers.go",
			setup: func(t *testing.T, base string) {
				writeAlignmentDoc(t, base, "knowledge/api.rule.md", "API Rule", "Applies to src/api/ handlers.")
			},
			env: map[string]string{"ARCHCORE_DISABLE_INJECTION": "1"},
		},
		{
			name:     "source edit with no matching docs",
			filePath: "src/api/handlers.go",
			setup: func(t *testing.T, base string) {
				writeAlignmentDoc(t, base, "knowledge/unrelated.rule.md", "Unrelated", "Nothing about that tree.")
			},
		},
		{
			name:     "types outside the allowlist are ignored",
			filePath: "src/api/handlers.go",
			setup: func(t *testing.T, base string) {
				writeAlignmentDoc(t, base, "vision/roadmap.plan.md", "Roadmap", "Touches src/api/ eventually.")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := setupArchcoreDir(t)
			if tt.setup != nil {
				tt.setup(t, base)
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if got := CodeAlignment(base, tt.filePath); got != "" {
				t.Errorf("CodeAlignment = %q, want empty", got)
			}
		})
	}
}

func TestCodeAlignment_InjectsMatchingDocs(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeAlignmentDoc(t, base, "knowledge/api.rule.md", "API Handler Rule", "Everything under src/api/ returns an envelope.")

	got := CodeAlignment(base, "src/api/handlers.go")

	for _, want := range []string{"[Archcore Context] Before editing src/api/handlers.go", "rule: API Handler Rule", "knowledge/api.rule.md"} {
		if !strings.Contains(got, want) {
			t.Errorf("advisory missing %q:\n%s", want, got)
		}
	}
}

// TestCodeAlignment_LongerPrefixWins: a document about the exact package must
// outrank one about the whole tree, or the specific advice is never seen.
func TestCodeAlignment_LongerPrefixWins(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeAlignmentDoc(t, base, "knowledge/broad.rule.md", "Broad", "Applies across src/.")
	writeAlignmentDoc(t, base, "knowledge/narrow.rule.md", "Narrow", "Applies to src/api/handlers/ only.")

	got := CodeAlignment(base, "src/api/handlers/users.go")

	narrow := strings.Index(got, "Narrow")
	broad := strings.Index(got, "Broad")
	if narrow < 0 || broad < 0 {
		t.Fatalf("expected both documents:\n%s", got)
	}
	if narrow > broad {
		t.Errorf("the more specific document ranked below the broader one:\n%s", got)
	}
}

// TestCodeAlignment_TypePriority: at equal specificity a rule constrains an
// edit more than a guide does.
func TestCodeAlignment_TypePriority(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeAlignmentDoc(t, base, "knowledge/z.guide.md", "Some Guide", "Working in src/api/.")
	writeAlignmentDoc(t, base, "knowledge/a.rule.md", "Some Rule", "Working in src/api/.")

	got := CodeAlignment(base, "src/api/handlers.go")

	// Presence first: strings.Index returns -1 for an absent document, and
	// "-1 > 0" is false — so comparing indices alone passes just as happily when
	// rules stopped being injected at all.
	rule := strings.Index(got, "Some Rule")
	guide := strings.Index(got, "Some Guide")
	if rule < 0 || guide < 0 {
		t.Fatalf("expected both documents:\n%s", got)
	}
	if rule > guide {
		t.Errorf("guide outranked rule at equal specificity:\n%s", got)
	}
}

// TestCodeAlignment_SourceRootsAreNormalized: config validation accepts "./src"
// and "src/", but document paths are slash-separated and unprefixed. Those roots
// used to validate cleanly and then match nothing, so the advisory went silent
// for a settings.json that looked correct and reported no error. Normalization
// at load makes the accepted set and the matching set the same set.
//
// The Windows separator is normalized by the same filepath.ToSlash and is not
// tabled here: on Unix a backslash is an ordinary filename character, so
// converting it would break a directory legitimately named that way.
func TestCodeAlignment_SourceRootsAreNormalized(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		root string // as written in settings.json
		file string // an edit that must produce an injection
	}{
		{name: "bare", root: "backend", file: "backend/api/h.go"},
		{name: "dot-slash prefixed", root: "./backend", file: "backend/api/h.go"},
		{name: "trailing slash", root: "backend/", file: "backend/api/h.go"},
		{name: "dot-slash and trailing slash", root: "./backend/", file: "backend/api/h.go"},
		{name: "nested root", root: "./backend/api/", file: "backend/api/h.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)
			settings := `{"sync":"none","codeAlignment":{"sourceRoots":["` + tt.root + `"]}}`
			if err := os.WriteFile(filepath.Join(base, ".archcore", "settings.json"), []byte(settings), 0o644); err != nil {
				t.Fatal(err)
			}
			writeAlignmentDoc(t, base, "knowledge/be.rule.md", "Backend Rule", "Applies to backend/api/ code.")

			if got := CodeAlignment(base, tt.file); !strings.Contains(got, "Backend Rule") {
				t.Errorf("root %q produced no injection for %s:\n%s", tt.root, tt.file, got)
			}
		})
	}
}

// TestCodeAlignment_TopThreeTruncation: beyond three the injection stops being
// a pointer and becomes reading homework.
func TestCodeAlignment_TopThreeTruncation(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	for i := range 8 {
		writeAlignmentDoc(t, base, fmt.Sprintf("knowledge/r-%d.rule.md", i), fmt.Sprintf("Rule %d", i), "Applies to src/api/ code.")
	}

	got := CodeAlignment(base, "src/api/handlers.go")

	if n := strings.Count(got, "\n- "); n != maxAlignmentDocs {
		t.Errorf("injected %d documents, want %d:\n%s", n, maxAlignmentDocs, got)
	}
}

// TestCodeAlignment_SourceRootsOverride pins that settings.json replaces the
// defaults rather than extending them.
func TestCodeAlignment_SourceRootsOverride(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	settings := `{"sync":"none","codeAlignment":{"sourceRoots":["backend"]}}`
	if err := os.WriteFile(filepath.Join(base, ".archcore", "settings.json"), []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
	writeAlignmentDoc(t, base, "knowledge/be.rule.md", "Backend Rule", "Applies to backend/api/ code.")
	writeAlignmentDoc(t, base, "knowledge/fe.rule.md", "Src Rule", "Applies to src/api/ code.")

	if got := CodeAlignment(base, "backend/api/h.go"); !strings.Contains(got, "Backend Rule") {
		t.Errorf("declared root produced no injection:\n%s", got)
	}
	if got := CodeAlignment(base, "src/api/h.go"); got != "" {
		t.Errorf("a default root survived the override:\n%s", got)
	}
}

// TestCodeAlignment_RejectedExcluded: a refused decision is not a constraint.
func TestCodeAlignment_RejectedExcluded(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	full := filepath.Join(base, ".archcore", "knowledge", "old.rule.md")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\ntitle: \"Refused Rule\"\nstatus: rejected\n---\n\nApplies to src/api/ code.\n"
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := CodeAlignment(base, "src/api/h.go"); got != "" {
		t.Errorf("a rejected document was injected:\n%s", got)
	}
}
