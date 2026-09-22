package advisory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// secretToken appears only in a file planted outside the project. Any of it
// surfacing in an advisory means the guard read a file it had no business
// opening, and put the result in front of the model.
const secretToken = "ZZTOPSECRETZZ"

// outsideDoc is deliberately full of findings — vague wording, missing
// sections, a short body, a cross-document link — so a traversal that succeeds
// produces loud, easily-asserted output rather than an empty string.
const outsideDoc = "---\n---\n" +
	"This " + secretToken + " approach is robust and various.\n" +
	"See .archcore/other/thing.adr.md for more.\n"

// TestPrecisionAdvisory_PathHandling pins which document paths the post-write
// advisory will open.
//
// docPath arrives from hook stdin, so it is host-controlled input. The advisory
// prefixed it with ".archcore/" when absent and read it with no validation,
// which made "../../outside/secret.adr.md" read straight out of the project and
// inject the derived findings into the model's context.
func TestPrecisionAdvisory_PathHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		docPath  string
		tool     string
		wantText bool // an advisory is expected
	}{
		{name: "a bare relative path is prefixed", docPath: "knowledge/in.adr.md", wantText: true},
		{name: "an already-prefixed path is used as-is", docPath: ".archcore/knowledge/in.adr.md", wantText: true},
		{name: "a traversal path is refused", docPath: "../../outside/secret.adr.md"},
		{name: "a traversal path below .archcore is refused", docPath: ".archcore/../../outside/secret.adr.md"},
		{name: "a missing file is silent", docPath: "knowledge/absent.adr.md"},
		{name: "an empty docPath is silent", docPath: ""},
		{name: "a non-mutating tool is silent", docPath: "knowledge/in.adr.md", tool: "mcp__archcore__get_document"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base, outsideDir := precisionFixture(t)
			_ = outsideDir

			tool := tt.tool
			if tool == "" {
				tool = "mcp__archcore__update_document"
			}
			got := Precision(base, tool, tt.docPath)

			if strings.Contains(got, secretToken) {
				t.Errorf("advisory leaked content from outside the project:\n%s", got)
			}
			if strings.Contains(got, "..") {
				t.Errorf("advisory echoed a traversal path:\n%s", got)
			}
			if tt.wantText && got == "" {
				t.Error("expected an advisory for a document inside the store")
			}
			if !tt.wantText && got != "" {
				t.Errorf("expected no advisory, got:\n%s", got)
			}
		})
	}
}

// TestPrecisionAdvisory_AbsolutePathIsRefused keeps the absolute-path case out
// of the table: it needs the fixture's real path to build the argument.
func TestPrecisionAdvisory_AbsolutePathIsRefused(t *testing.T) {
	t.Parallel()
	base, outsideDir := precisionFixture(t)

	abs := filepath.Join(outsideDir, "secret.adr.md")
	got := Precision(base, "mcp__archcore__update_document", abs)

	if strings.Contains(got, secretToken) {
		t.Errorf("an absolute path outside the project was read:\n%s", got)
	}
	if got != "" {
		t.Errorf("expected no advisory for an absolute path, got:\n%s", got)
	}
}

// TestPrecisionAdvisory_SymlinkedDocumentIsRefused: a document inside the store
// whose ancestor leaves it is the same escape by another route.
//
// The assertion is on the whole result, not on the planted token. The advisory
// derives findings rather than quoting the body, so a token check passes while
// the escape succeeds — which is exactly how this went unnoticed. What an
// escape does leak is the file's own vocabulary and its document links, and the
// only way to assert none of it surfaced is to require silence.
func TestPrecisionAdvisory_SymlinkedDocumentIsRefused(t *testing.T) {
	t.Parallel()
	base, outsideDir := precisionFixture(t)

	link := filepath.Join(base, ".archcore", "escape")
	if err := os.Symlink(outsideDir, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	got := Precision(base, "mcp__archcore__update_document", ".archcore/escape/secret.adr.md")

	if got != "" {
		t.Errorf("a symlinked path reached outside the store:\n%s", got)
	}
}

// precisionFixture returns a project holding one real document, plus a sibling
// directory outside it holding the planted secret.
func precisionFixture(t *testing.T) (base, outsideDir string) {
	t.Helper()
	root := t.TempDir()
	base = filepath.Join(root, "project")
	outsideDir = filepath.Join(root, "outside")
	if err := os.MkdirAll(filepath.Join(base, ".archcore", "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeArchcoreDoc(t, base, "knowledge/in.adr.md", "---\n---\nA robust and various body.\n")
	if err := os.WriteFile(filepath.Join(outsideDir, "secret.adr.md"), []byte(outsideDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	return base, outsideDir
}
