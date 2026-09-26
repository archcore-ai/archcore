package integration

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestUpdateDocumentEdits_RoundTrip drives edits through the real JSON-RPC path:
// the client marshals the array of objects, the server decodes it, and the old
// text is copied from get_document's own output, line endings included.
func TestUpdateDocumentEdits_RoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		onDisk string
	}{
		{name: "lf", onDisk: "---\ntitle: E\nstatus: draft\n---\n\n## Context\nalpha\n\n## Decision\nbeta\n"},
		{name: "crlf", onDisk: "---\r\ntitle: E\r\nstatus: draft\r\n---\r\n\r\n## Context\r\nalpha\r\n\r\n## Decision\r\nbeta\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := initArchcore(t)
			writeFixtureFile(t, filepath.Join(base, ".archcore", "e.adr.md"), tt.onDisk)
			c := newTestClient(t, base)
			const path = ".archcore/e.adr.md"

			content := getDoc(t, c, path).Content
			start := strings.Index(content, "alpha")
			end := strings.Index(content, "## Decision") + len("## Decision")
			mustCallTool(t, c, "update_document", map[string]any{
				"path":  path,
				"edits": []map[string]any{{"old_string": content[start:end], "new_string": "ALPHA\n\n## Outcome"}},
			})

			got := getDoc(t, c, path)
			if !strings.Contains(got.Content, "## Context\nALPHA\n\n## Outcome\nbeta\n") {
				t.Errorf("edited body missing, got:\n%q", got.Content)
			}
			if got.Title != "E" || got.Status != "draft" {
				t.Errorf("frontmatter changed: title %q, status %q", got.Title, got.Status)
			}
		})
	}
}
