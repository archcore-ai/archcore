package cmd

// Conformance tests for the GLOBALS block (session-globals-disclosure.spec):
// presence and absence, both ceilings with named drops, the warning merge, the
// precedence sentence, the CORPUS label, and the banner suffix.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/display"
)

// globalsFixture builds a project with one local document and n declared global
// sources, each holding the given documents (slash-relative path -> written).
// Returns the project base directory.
func globalsFixture(t *testing.T, sourceDocs ...[]string) string {
	t.Helper()
	root := t.TempDir()
	base := filepath.Join(root, "project")
	writeArchcoreDoc(t, base, "knowledge/local-rule.rule.md", "---\ntitle: Local Rule\nstatus: accepted\n---\nBody.\n")

	entries := make([]string, 0, len(sourceDocs))
	for i, docs := range sourceDocs {
		id := fmt.Sprintf("src-%02d", i)
		gsDir := filepath.Join(root, id, ".archcore")
		if err := os.MkdirAll(gsDir, 0o755); err != nil {
			t.Fatal(err)
		}
		for _, p := range docs {
			full := filepath.Join(gsDir, filepath.FromSlash(p))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, []byte("---\ntitle: G\nstatus: accepted\n---\nBody.\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		entries = append(entries, fmt.Sprintf(`{"id":%q,"path":%q}`, id, "../"+id+"/.archcore"))
	}
	settings := fmt.Sprintf(`{"sync":"none","globals":[%s]}`, strings.Join(entries, ","))
	writeArchcoreDoc(t, base, "settings.json", settings)
	return base
}

// TestGlobalsBlock_HealthySource: clauses 1, 3-5, 13-15 — the line carries id,
// count, categories, and directories; the precedence sentence closes the block;
// CORPUS switches to the "local documents" label; the banner appends the total.
func TestGlobalsBlock_HealthySource(t *testing.T) {
	t.Parallel()
	base := globalsFixture(t, []string{
		"concepts/a.rule.md",
		"concepts/b.doc.md",
		"product/c.idea.md",
	})

	ctx, counts := buildSessionContext(bg(), base)

	if !strings.Contains(ctx, "GLOBALS (read-only, query via MCP read tools):") {
		t.Fatalf("GLOBALS heading missing; ctx=%q", ctx)
	}
	if !strings.Contains(ctx, "- src-00 — 3 docs (knowledge 2, vision 1) · concepts/ 2, product/ 1") {
		t.Errorf("source line wrong; ctx=%q", ctx)
	}
	if !strings.Contains(ctx, globalsPrecedenceLine) {
		t.Error("precedence sentence missing")
	}
	if !strings.Contains(ctx, "CORPUS: 1 local documents") {
		t.Errorf("CORPUS label must read \"local documents\" when the block renders; ctx=%q", ctx)
	}
	if counts.global != 3 {
		t.Errorf("global doc count = %d, want 3", counts.global)
	}
	if banner := display.HookConnectedLine("v0.0.0", counts.local, counts.global); !strings.Contains(banner, "1 docs + 3 global") {
		t.Errorf("banner = %q, want the global total appended", banner)
	}
	// The block must not leak global document paths or titles.
	if strings.Contains(ctx, "a.rule.md") {
		t.Error("global document filename leaked into the session context")
	}
}

// TestGlobalsBlock_AbsentWithoutGlobals: clause 2 — no declaration, no block, no
// label change, no banner suffix.
func TestGlobalsBlock_AbsentWithoutGlobals(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "knowledge/a.rule.md", "---\ntitle: A\nstatus: accepted\n---\nBody.\n")

	ctx, counts := buildSessionContext(bg(), base)

	if strings.Contains(ctx, "GLOBALS") {
		t.Errorf("no GLOBALS block without declared sources; ctx=%q", ctx)
	}
	if !strings.Contains(ctx, "CORPUS: 1 documents") {
		t.Errorf("CORPUS label must stay unchanged; ctx=%q", ctx)
	}
	if counts.global != 0 {
		t.Errorf("global doc count = %d, want 0", counts.global)
	}
	if banner := display.HookConnectedLine("v0.0.0", counts.local, counts.global); strings.Contains(banner, "global") {
		t.Errorf("banner = %q, want no global suffix", banner)
	}
}

// TestGlobalsBlock_WarningMergeAndPrecedence: clause 12 — a fatal and an empty
// source render as ⚠ lines inside the block, healthy sources still render, and
// the precedence sentence appears because one healthy line rendered.
func TestGlobalsBlock_WarningMergeAndPrecedence(t *testing.T) {
	t.Parallel()
	base := globalsFixture(t,
		[]string{"concepts/a.rule.md"}, // src-00: healthy
		[]string{},                     // src-01: empty
	)
	// src-02: declared but missing on disk.
	settings := `{"sync":"none","globals":[` +
		`{"id":"src-00","path":"../src-00/.archcore"},` +
		`{"id":"src-01","path":"../src-01/.archcore"},` +
		`{"id":"gone","path":"../gone/.archcore"}]}`
	writeArchcoreDoc(t, base, "settings.json", settings)

	ctx, _ := buildSessionContext(bg(), base)

	idx := strings.Index(ctx, "GLOBALS")
	if idx < 0 {
		t.Fatalf("GLOBALS block missing; ctx=%q", ctx)
	}
	block := ctx[idx:]
	blockEnd := strings.Index(block, "\n\n")
	if blockEnd < 0 {
		blockEnd = len(block)
	}
	block = block[:blockEnd]

	if !strings.Contains(block, "- src-00 — 1 docs") {
		t.Errorf("healthy source missing from block; block=%q", block)
	}
	if !strings.Contains(block, `⚠ global source "src-01"`) || !strings.Contains(block, "contains no documents") {
		t.Errorf("empty source warning missing from block; block=%q", block)
	}
	if !strings.Contains(block, `⚠ global source "gone" not found`) || !strings.Contains(block, "clone it or fix .archcore/settings.json") {
		t.Errorf("fatal source warning missing from block; block=%q", block)
	}
	if !strings.Contains(block, globalsPrecedenceLine) {
		t.Error("precedence sentence must render when one healthy line rendered")
	}
}

// TestGlobalsBlock_Ceilings: clauses 8-11 and 15 — at most 6 directories per
// source and 8 source lines, each drop named; the banner total still counts
// every healthy source, including the truncated ones.
func TestGlobalsBlock_Ceilings(t *testing.T) {
	t.Parallel()
	manyDirs := make([]string, 0, 8)
	for i := range 8 {
		manyDirs = append(manyDirs, fmt.Sprintf("dir%02d/d.rule.md", i))
	}
	sources := make([][]string, 10)
	for i := range sources {
		sources[i] = manyDirs
	}
	base := globalsFixture(t, sources...)

	ctx, counts := buildSessionContext(bg(), base)

	if counts.global != 80 {
		t.Errorf("global doc count = %d, want 80 — the banner total must not stop at the render ceiling", counts.global)
	}
	if got := strings.Count(ctx, "  - src-"); got != maxGlobalsSources {
		t.Errorf("rendered %d source lines, want %d", got, maxGlobalsSources)
	}
	if !strings.Contains(ctx, "… and 2 more sources") {
		t.Errorf("dropped sources must be named; ctx=%q", ctx)
	}
	firstLine := ctx[strings.Index(ctx, "  - src-00"):]
	firstLine = firstLine[:strings.Index(firstLine, "\n")]
	if got := strings.Count(firstLine, "/"); got != maxGlobalsDirs {
		t.Errorf("rendered %d directories on the line, want %d; line=%q", got, maxGlobalsDirs, firstLine)
	}
	if !strings.Contains(firstLine, "… and 2 more dirs") {
		t.Errorf("dropped directories must be named; line=%q", firstLine)
	}
}

// TestGlobalsBlock_InvalidSettings: clause 18 — an unparsable settings.json
// renders only the invalid-settings warning, never a block.
func TestGlobalsBlock_InvalidSettings(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "knowledge/a.rule.md", "---\ntitle: A\nstatus: accepted\n---\nBody.\n")
	writeArchcoreDoc(t, base, "settings.json", `{"sync":"none","globals":[{"id":""}]}`)

	ctx, counts := buildSessionContext(bg(), base)

	if !strings.Contains(ctx, "⚠ invalid .archcore/settings.json") {
		t.Errorf("invalid-settings warning missing; ctx=%q", ctx)
	}
	if strings.Contains(ctx, "GLOBALS (") {
		t.Errorf("no block may render on invalid settings; ctx=%q", ctx)
	}
	if counts.global != 0 {
		t.Errorf("global doc count = %d, want 0", counts.global)
	}
}
