package cmd

import (
	"testing"

	"archcore-cli/internal/docs"
)

// Scan-budget proofs.
//
// "The write guard triggers no full scan" and "one scan serves the session" are
// budget claims, and a wall-clock assertion for either is flaky on shared CI.
// docs.ScanCount is the deterministic equivalent: it counts corpus walks, so the
// claim becomes an exact number rather than a timing.
//
// None of these may call t.Parallel(). The counter is process-global, and Go
// defers every parallel test until the sequential pass finishes — adding
// t.Parallel() later would make them race each other silently.

// scanDelta reports how many corpus walks fn performed.
func scanDelta(fn func()) uint64 {
	before := docs.ScanCount()
	fn()
	return docs.ScanCount() - before
}

// TestWriteGuard_PerformsNoCorpusScan: the pre-write guard runs on every file
// write the agent makes, inside a one-second host budget. Its verdict is a path
// judgement and needs no knowledge of the corpus.
func TestWriteGuard_PerformsNoCorpusScan(t *testing.T) {
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "knowledge/a.adr.md", "---\ntitle: \"A\"\nstatus: draft\n---\n\nBody.\n")

	tests := []struct {
		name     string
		filePath string
	}{
		{name: "a blocked document write", filePath: ".archcore/knowledge/a.adr.md"},
		{name: "an allowed source write", filePath: "src/main.go"},
		{name: "an allowed settings write", filePath: ".archcore/settings.json"},
		{name: "no file at all", filePath: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scanDelta(func() { writeGuardDecision(base, tt.filePath) })
			if got != 0 {
				t.Errorf("the write guard performed %d corpus scan(s), want 0", got)
			}
		})
	}
}

// TestBuildSessionContext_ScansTheCorpusOnce: session start used to scan three
// times — once including every global document only to discard them, once for
// the recap, and once more inside the staleness correlation.
func TestBuildSessionContext_ScansTheCorpusOnce(t *testing.T) {
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "knowledge/a.adr.md", "---\ntitle: \"A\"\nstatus: draft\n---\n\nBody.\n")

	got := scanDelta(func() { buildSessionContext(bg(), base) })

	if got != 1 {
		t.Errorf("buildSessionContext performed %d corpus scans, want 1", got)
	}
}

// TestCollectStatus_ScansTheCorpusOnce: the post-tool-use hook runs these checks
// after every document mutation. The structural checks used to re-read each file
// directly, bypassing the scan cache.
func TestCollectStatus_ScansTheCorpusOnce(t *testing.T) {
	base := setupArchcoreDir(t)
	for _, name := range []string{"a.adr.md", "b.rule.md", "c.guide.md"} {
		writeArchcoreDoc(t, base, "knowledge/"+name, "---\ntitle: \"T\"\nstatus: draft\n---\n\nBody.\n")
	}

	got := scanDelta(func() { collectStatus(base) })

	if got != 1 {
		t.Errorf("collectStatus performed %d corpus scans, want 1", got)
	}
}

// TestPostToolUse_ReadToolCostsNoScan is the payoff of gating on mutation: a
// read tool must not pay for the post-write checks at all.
func TestPostToolUse_ReadToolCostsNoScan(t *testing.T) {
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "knowledge/a.adr.md", "---\ntitle: \"A\"\nstatus: draft\n---\n\nBody.\n")

	read := scanDelta(func() {
		postToolUseHandler(bg(), postReq(t, base, "mcp__archcore__get_document", ".archcore/knowledge/a.adr.md"))
	})
	if read != 0 {
		t.Errorf("a read tool performed %d corpus scan(s), want 0", read)
	}

	// Positive control: without it this passes when the checks never run at all.
	wrote := scanDelta(func() {
		postToolUseHandler(bg(), postReq(t, base, "mcp__archcore__update_document", ".archcore/knowledge/a.adr.md"))
	})
	if wrote == 0 {
		t.Error("a document mutation performed no corpus scan, so no check ran")
	}
}
