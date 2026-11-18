package integration

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"archcore-cli/internal/advisory"
)

// hookRowPathRe captures the document path of one hook row: "- rule: Title [path]".
var hookRowPathRe = regexp.MustCompile(`(?m)^- [^:]+: .* \[(\.archcore/[^\]]+)\]`)

// TestFileContext_HookAndForPathAgree pins file-context-resolution.spec
// behavior 13: one file, one corpus, the same matched rows in the same order.
func TestFileContext_HookAndForPathAgree(t *testing.T) {
	t.Parallel()
	base := initArchcore(t)
	docs := map[string]string{
		"web/seam.doc.md":       "PageShell.tsx renders the shell.",
		"web/layout.rule.md":    "Applies to src/app/_components/ code.",
		"web/extra-3.adr.md":    "Covers src/app/_components/ state.",
		"web/extra-4.adr.md":    "Covers src/app/_components/ styles.",
		"web/app.adr.md":        "Covers src/app/ routing.",
		"web/draft.spec.md":     "Covers src/app/_components/ props.",
		"web/broad.guide.md":    "Everything under src/ builds with one script.",
		"web/other.rule.md":     "Applies to src/app/_components/ and src/app/_components/ again.",
		"web/extra-1.rule.md":   "Applies to src/app/ code.",
		"web/extra-2.cpat.md":   "Applies to src/app/ code.",
		"general/guard.rule.md": "Return early.",
	}
	for rel, body := range docs {
		status := "accepted"
		if strings.Contains(rel, "draft") {
			status = "draft"
		}
		writeFixtureFile(t, filepath.Join(base, ".archcore", filepath.FromSlash(rel)),
			fmt.Sprintf("---\ntitle: %q\nstatus: %s\n---\n\n%s\n", rel, status, body))
	}
	file := "src/app/_components/PageShell.tsx"
	c := newTestClient(t, base)

	type row struct {
		Path   string `json:"path"`
		Reason string `json:"reason"`
	}
	resp := decodeJSON[struct {
		Results []row `json:"results"`
	}](t, mustCallTool(t, c, "search_documents", map[string]any{"for_path": file}))
	// The tool lists the matched rows first; a rule can be a matched row too, so
	// the count comes from the resolver, not from the file suffix.
	fc, err := advisory.ResolveFileContext(base, file)
	if err != nil {
		t.Fatal(err)
	}
	var fromTool []string
	for i, r := range resp.Results {
		if i < min(len(fc.Matched), advisory.MaxContextMatched) {
			fromTool = append(fromTool, r.Path)
		}
	}

	hook := advisory.CodeAlignment(base, file)
	matchedPart, _, _ := strings.Cut(hook, "Rules that apply to this file")
	var fromHook []string
	for _, m := range hookRowPathRe.FindAllStringSubmatch(matchedPart, -1) {
		fromHook = append(fromHook, m[1])
	}

	if len(fromTool) != advisory.MaxContextMatched {
		t.Fatalf("for_path returned %d matched rows, want %d: %v", len(fromTool), advisory.MaxContextMatched, fromTool)
	}
	if !slices.Equal(fromTool, fromHook) {
		t.Errorf("hook rows %v differ from for_path rows %v\nhook:\n%s", fromHook, fromTool, hook)
	}
}
