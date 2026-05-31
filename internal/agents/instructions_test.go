package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsertFencedBlock_NewFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "AGENTS.md")

	if err := upsertFencedBlock(path); err != nil {
		t.Fatalf("upsertFencedBlock: %v", err)
	}

	got := readFile(t, path)
	if !strings.Contains(got, instructionsMarkerStart) || !strings.Contains(got, instructionsMarkerEnd) {
		t.Error("missing markers")
	}
	if !strings.Contains(got, "## Archcore — project context for this repo") {
		t.Error("missing nudge body")
	}
	if !strings.HasSuffix(got, instructionsMarkerEnd+"\n") {
		t.Errorf("file should end with end-marker + newline, got:\n%q", got)
	}
}

func TestUpsertFencedBlock_Idempotent(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "AGENTS.md")

	if err := upsertFencedBlock(path); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	first := readFile(t, path)
	if err := upsertFencedBlock(path); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	second := readFile(t, path)

	if first != second {
		t.Errorf("not idempotent:\nfirst:\n%q\nsecond:\n%q", first, second)
	}
	if n := strings.Count(second, instructionsMarkerStart); n != 1 {
		t.Errorf("want exactly 1 start marker, got %d", n)
	}
}

func TestUpsertFencedBlock_PreservesUserContent(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	user := "# My Project\n\nHand-written guidance for agents.\n"
	writeFile(t, path, user)

	if err := upsertFencedBlock(path); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got := readFile(t, path)

	if !strings.Contains(got, "Hand-written guidance for agents.") {
		t.Error("user content was lost")
	}
	if !strings.Contains(got, instructionsMarkerStart) {
		t.Error("block not added")
	}
	if idx := strings.Index(got, instructionsMarkerStart); idx < strings.Index(got, "Hand-written") {
		t.Error("block should be appended after user content")
	}

	// A second upsert must not duplicate or drift.
	first := got
	if err := upsertFencedBlock(path); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if second := readFile(t, path); first != second {
		t.Errorf("not idempotent with user content:\nfirst:\n%q\nsecond:\n%q", first, second)
	}
}

func TestUpsertFencedBlock_UpdatesInPlace(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	seeded := "# Header\n\n" +
		instructionsMarkerStart + " managed by old version\n" +
		"OUTDATED BODY\n" +
		instructionsMarkerEnd + "\n\n" +
		"## Footer kept by user\n"
	writeFile(t, path, seeded)

	if err := upsertFencedBlock(path); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got := readFile(t, path)

	if strings.Contains(got, "OUTDATED BODY") {
		t.Error("stale block body was not replaced")
	}
	if !strings.Contains(got, "## Archcore — project context for this repo") {
		t.Error("fresh body not written")
	}
	if !strings.Contains(got, "# Header") || !strings.Contains(got, "## Footer kept by user") {
		t.Error("surrounding user content was lost")
	}
	if n := strings.Count(got, instructionsMarkerStart); n != 1 {
		t.Errorf("want exactly 1 start marker after in-place update, got %d", n)
	}
}

func TestUpsertFencedBlock_MissingTrailingNewline(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	writeFile(t, path, "no trailing newline")

	if err := upsertFencedBlock(path); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got := readFile(t, path)

	if !strings.Contains(got, "no trailing newline\n\n"+instructionsMarkerStart) {
		t.Errorf("expected blank-line separator before block, got:\n%q", got)
	}
}

func TestUpsertFencedBlock_CreatesDirs(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nested", "dir", "GEMINI.md")

	if err := upsertFencedBlock(path); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestWriteOwnedFile_CreatesAndOverwrites(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), ".claude", "rules", "archcore.md")

	if err := writeOwnedFile(path, "first\n"); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if got := readFile(t, path); got != "first\n" {
		t.Errorf("got %q, want %q", got, "first\n")
	}

	if err := writeOwnedFile(path, "second\n"); err != nil {
		t.Fatalf("second write: %v", err)
	}
	if got := readFile(t, path); got != "second\n" {
		t.Errorf("overwrite failed: got %q, want %q", got, "second\n")
	}
}

func TestRemoveFencedBlock_KeepsUserContent(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	writeFile(t, path, "# My Project\n\nKeep me.\n")
	if err := upsertFencedBlock(path); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if err := removeFencedBlock(path); err != nil {
		t.Fatalf("removeFencedBlock: %v", err)
	}
	got := readFile(t, path)

	if strings.Contains(got, instructionsMarkerStart) {
		t.Error("block not removed")
	}
	if !strings.Contains(got, "Keep me.") {
		t.Error("user content was lost")
	}
	if !strings.HasSuffix(got, "\n") || strings.HasSuffix(got, "\n\n") {
		t.Errorf("expected single trailing newline, got:\n%q", got)
	}
}

func TestRemoveFencedBlock_DeletesWhenOnlyBlock(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	if err := upsertFencedBlock(path); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if err := removeFencedBlock(path); err != nil {
		t.Fatalf("removeFencedBlock: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file holding only the block should be deleted, stat err = %v", err)
	}
}

func TestRemoveFencedBlock_Noop(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Missing file is a no-op.
	if err := removeFencedBlock(filepath.Join(dir, "missing.md")); err != nil {
		t.Errorf("missing file should be no-op, got %v", err)
	}

	// File without a managed block is left untouched.
	path := filepath.Join(dir, "AGENTS.md")
	user := "# Only user content\n"
	writeFile(t, path, user)
	if err := removeFencedBlock(path); err != nil {
		t.Fatalf("removeFencedBlock: %v", err)
	}
	if got := readFile(t, path); got != user {
		t.Errorf("file without block changed: got %q, want %q", got, user)
	}
}

func TestRemoveOwnedFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Missing file is a no-op.
	if err := removeOwnedFile(filepath.Join(dir, "missing.md")); err != nil {
		t.Errorf("missing file should be no-op, got %v", err)
	}

	path := filepath.Join(dir, "archcore.md")
	writeFile(t, path, "owned\n")
	if err := removeOwnedFile(path); err != nil {
		t.Fatalf("removeOwnedFile: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file should be removed, stat err = %v", err)
	}
}

func TestAgentInstructionsPath(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	want := map[AgentID]string{
		ClaudeCode: filepath.Join(base, ".claude", "rules", "archcore.md"),
		GeminiCLI:  filepath.Join(base, "GEMINI.md"),
		Cursor:     filepath.Join(base, "AGENTS.md"),
		OpenCode:   filepath.Join(base, "AGENTS.md"),
		CodexCLI:   filepath.Join(base, "AGENTS.md"),
		RooCode:    filepath.Join(base, "AGENTS.md"),
		Cline:      filepath.Join(base, "AGENTS.md"),
		Copilot:    filepath.Join(base, "AGENTS.md"),
	}
	for id, wantPath := range want {
		t.Run(string(id), func(t *testing.T) {
			t.Parallel()
			got := ByID(id).InstructionsPath(base)
			if got != wantPath {
				t.Errorf("InstructionsPath = %q, want %q", got, wantPath)
			}
		})
	}
}

func TestWriteInstructions_ClaudeOwnedNoMarkers(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	if err := ByID(ClaudeCode).WriteInstructions(base); err != nil {
		t.Fatalf("WriteInstructions: %v", err)
	}
	got := readFile(t, filepath.Join(base, ".claude", "rules", "archcore.md"))

	if strings.Contains(got, instructionsMarkerStart) {
		t.Error("owned Claude file must not contain HTML markers")
	}
	if !strings.Contains(got, "## Archcore — project context for this repo") {
		t.Error("owned Claude file missing nudge body")
	}
}

func TestWriteInstructions_AgentsMDHasMarkers(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	if err := ByID(Cursor).WriteInstructions(base); err != nil {
		t.Fatalf("WriteInstructions: %v", err)
	}
	got := readFile(t, filepath.Join(base, "AGENTS.md"))

	if !strings.Contains(got, instructionsMarkerStart) {
		t.Error("AGENTS.md should use fenced markers")
	}
}

func TestFindManagedSpans(t *testing.T) {
	t.Parallel()
	blk := instructionsFencedBlock
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"no markers", "# just user content\n", 0},
		{"one block", "pre\n\n" + blk + "\n", 1},
		{"two blocks", blk + "\nmid\n" + blk, 2},
		{"orphan start only", "x\n" + instructionsMarkerStart + " stray\ntail\n", 0},
		{"end only", instructionsMarkerEnd + "\ntail\n", 0},
		{"reversed end before start", instructionsMarkerEnd + "\n" + instructionsMarkerStart + "\n", 0},
		{"orphan start then real block", instructionsMarkerStart + " stray\nuser\n\n" + blk + "\n", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := len(findManagedSpans(tt.input)); got != tt.want {
				t.Errorf("findManagedSpans(%q) = %d spans, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// TestUpsertFencedBlock_OrphanStartPreservesUserContent is the regression test
// for review finding B1: a stray start marker (no matching end) followed by user
// content. A naive first-start/first-end pairing would, on the second upsert,
// span from the orphan to the appended block's end and delete the tail.
func TestUpsertFencedBlock_OrphanStartPreservesUserContent(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	writeFile(t, path, "# Doc\n\n"+instructionsMarkerStart+" stray\nUSER TAIL THAT MUST SURVIVE\n")

	if err := upsertFencedBlock(path); err != nil { // appends a block
		t.Fatalf("first upsert: %v", err)
	}
	if err := upsertFencedBlock(path); err != nil { // must not eat the tail
		t.Fatalf("second upsert: %v", err)
	}
	got := readFile(t, path)
	if !strings.Contains(got, "USER TAIL THAT MUST SURVIVE") {
		t.Errorf("user content was destroyed:\n%s", got)
	}
	if err := upsertFencedBlock(path); err != nil {
		t.Fatalf("third upsert: %v", err)
	}
	if third := readFile(t, path); third != got {
		t.Errorf("not idempotent:\nsecond:\n%q\nthird:\n%q", got, third)
	}
}

// TestUpsertFencedBlock_CollapsesDuplicateBlocks is the regression test for B2:
// two existing blocks must collapse to one, preserving content between them.
func TestUpsertFencedBlock_CollapsesDuplicateBlocks(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	writeFile(t, path, instructionsFencedBlock+"\n\nMIDDLE USER TEXT\n\n"+instructionsFencedBlock+"\n")

	if err := upsertFencedBlock(path); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got := readFile(t, path)
	if n := strings.Count(got, instructionsMarkerStart); n != 1 {
		t.Errorf("want exactly 1 block after collapse, got %d:\n%s", n, got)
	}
	if !strings.Contains(got, "MIDDLE USER TEXT") {
		t.Error("user text between duplicate blocks was lost")
	}
}

// TestUpsertFencedBlock_ReversedMarkersIdempotent is the regression test for B3:
// an end marker before a start marker must not cause unbounded block growth.
func TestUpsertFencedBlock_ReversedMarkersIdempotent(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	writeFile(t, path, instructionsMarkerEnd+"\n"+instructionsMarkerStart+"\n")

	if err := upsertFencedBlock(path); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	first := readFile(t, path)
	if err := upsertFencedBlock(path); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if second := readFile(t, path); first != second {
		t.Errorf("not idempotent with reversed markers:\nfirst:\n%q\nsecond:\n%q", first, second)
	}
}

func TestRemoveFencedBlock_RemovesAllDuplicates(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	writeFile(t, path, "TOP\n\n"+instructionsFencedBlock+"\n\nMIDDLE\n\n"+instructionsFencedBlock+"\n\nBOTTOM\n")

	if err := removeFencedBlock(path); err != nil {
		t.Fatalf("removeFencedBlock: %v", err)
	}
	got := readFile(t, path)
	if strings.Contains(got, instructionsMarkerStart) {
		t.Errorf("not all blocks removed:\n%s", got)
	}
	for _, want := range []string{"TOP", "MIDDLE", "BOTTOM"} {
		if !strings.Contains(got, want) {
			t.Errorf("user content %q was lost:\n%s", want, got)
		}
	}
}

// TestRemoveFencedBlock_CRLF is the regression test for S3: removing a block
// from a CRLF (Windows-edited) file must not leave stray blank lines.
func TestRemoveFencedBlock_CRLF(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	writeFile(t, path, "# Title\r\n\r\n"+instructionsFencedBlock+"\r\n\r\nfooter\r\n")

	if err := removeFencedBlock(path); err != nil {
		t.Fatalf("removeFencedBlock: %v", err)
	}
	got := readFile(t, path)
	if strings.Contains(got, instructionsMarkerStart) {
		t.Error("block not removed")
	}
	if !strings.Contains(got, "# Title") || !strings.Contains(got, "footer") {
		t.Errorf("user content lost:\n%q", got)
	}
	if strings.Contains(got, "\r\n\r\n\r\n") || strings.Contains(got, "\n\n\n") {
		t.Errorf("stray blank lines left after CRLF removal:\n%q", got)
	}
}

// TestAgentInstructions_WriteThenRemove exercises every agent's WriteInstructions
// and RemoveInstructions wrappers directly (they are otherwise only hit via the
// cmd package, which does not attribute coverage here).
func TestAgentInstructions_WriteThenRemove(t *testing.T) {
	t.Parallel()
	for _, id := range AllIDs() {
		t.Run(string(id), func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			a := ByID(id)
			if err := a.WriteInstructions(base); err != nil {
				t.Fatalf("WriteInstructions: %v", err)
			}
			path := a.InstructionsPath(base)
			if got := readFile(t, path); !strings.Contains(got, "## Archcore — project context for this repo") {
				t.Error("nudge body missing")
			}
			if err := a.RemoveInstructions(base); err != nil {
				t.Fatalf("RemoveInstructions: %v", err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("block-only instruction file should be deleted, stat err = %v", err)
			}
		})
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", path, err)
	}
	return string(data)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}
