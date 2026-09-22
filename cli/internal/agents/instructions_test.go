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

func TestWriteClaudeInstructions_MigratesLegacyRulesFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	legacy := filepath.Join(base, ".claude", "rules", "archcore.md")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeFile(t, legacy, "legacy nudge\n")

	if err := writeClaudeInstructions(base); err != nil {
		t.Fatalf("writeClaudeInstructions: %v", err)
	}

	// Legacy owned file is migrated away...
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("legacy .claude/rules/archcore.md should be removed, stat err = %v", err)
	}
	// ...and both current targets carry the block.
	for _, f := range []string{"CLAUDE.md", "AGENTS.md"} {
		if got := readFile(t, filepath.Join(base, f)); !strings.Contains(got, instructionsMarkerStart) {
			t.Errorf("%s missing managed block after write", f)
		}
	}
}

// TestRemoveClaudeInstructions_DeletesLegacyRulesFile: uninstalling Claude Code
// must also delete the legacy .claude/rules/archcore.md, not just strip CLAUDE.md.
func TestRemoveClaudeInstructions_DeletesLegacyRulesFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	legacy := filepath.Join(base, legacyClaudeRulesRelPath)
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeFile(t, legacy, "legacy nudge\n")

	if err := removeClaudeInstructions(base); err != nil {
		t.Fatalf("removeClaudeInstructions: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("legacy .claude/rules/archcore.md should be removed on uninstall, stat err = %v", err)
	}
}

// TestWriteClaudeInstructions_PreservesExistingUserCLAUDEMD pins the common
// upgrade path: a hand-written CLAUDE.md plus the legacy rules file. User content
// must survive, the legacy file must migrate away, and re-runs stay byte-stable.
func TestWriteClaudeInstructions_PreservesExistingUserCLAUDEMD(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	user := "# My Project\n\nHand-written Claude guidance.\n"
	writeFile(t, filepath.Join(base, "CLAUDE.md"), user)
	legacy := filepath.Join(base, legacyClaudeRulesRelPath)
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeFile(t, legacy, "legacy nudge\n")

	if err := writeClaudeInstructions(base); err != nil {
		t.Fatalf("writeClaudeInstructions: %v", err)
	}

	claudeMD := readFile(t, filepath.Join(base, "CLAUDE.md"))
	if !strings.Contains(claudeMD, "Hand-written Claude guidance.") {
		t.Error("user CLAUDE.md content was lost")
	}
	if n := strings.Count(claudeMD, instructionsMarkerStart); n != 1 {
		t.Errorf("want exactly 1 managed block in CLAUDE.md, got %d", n)
	}
	if strings.Contains(claudeMD, "legacy nudge") {
		t.Error("legacy content must not be merged into CLAUDE.md")
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("legacy rules file should be migrated away, stat err = %v", err)
	}

	agentsMD := readFile(t, filepath.Join(base, "AGENTS.md"))
	if err := writeClaudeInstructions(base); err != nil {
		t.Fatalf("second writeClaudeInstructions: %v", err)
	}
	if got := readFile(t, filepath.Join(base, "CLAUDE.md")); got != claudeMD {
		t.Errorf("CLAUDE.md not byte-identical on re-run:\nfirst:\n%q\nsecond:\n%q", claudeMD, got)
	}
	if got := readFile(t, filepath.Join(base, "AGENTS.md")); got != agentsMD {
		t.Errorf("AGENTS.md not byte-identical on re-run")
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
		ClaudeCode: filepath.Join(base, "CLAUDE.md"),
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

func TestWriteInstructions_ClaudeWritesBothTargets(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	if err := ByID(ClaudeCode).WriteInstructions(base); err != nil {
		t.Fatalf("WriteInstructions: %v", err)
	}

	// CLAUDE.md — the file Claude Code reads natively — carries the fenced block.
	claudeMD := readFile(t, filepath.Join(base, "CLAUDE.md"))
	if !strings.Contains(claudeMD, instructionsMarkerStart) {
		t.Error("CLAUDE.md should carry the fenced block for Claude Code")
	}
	if !strings.Contains(claudeMD, "## Archcore — project context for this repo") {
		t.Error("CLAUDE.md missing nudge body")
	}

	// AGENTS.md also carries the block (standard the plugin + other hosts use).
	agentsMD := readFile(t, filepath.Join(base, "AGENTS.md"))
	if !strings.Contains(agentsMD, instructionsMarkerStart) {
		t.Error("AGENTS.md should carry the fenced block for Claude Code")
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

// TestInstructionsBody_FlagsGlobalSources guards the global-sources nudge: it must
// be present, phrased conditionally (most projects mount none), name the source_kind
// tag agents look for, and carry the read-only / local-overrides constraint. A
// regression that drops the paragraph — or hard-codes it as "this repo mounts …",
// which would be wrong for the many projects with no global source — fails here.
func TestInstructionsBody_FlagsGlobalSources(t *testing.T) {
	t.Parallel()
	for _, want := range []string{
		"global sources",              // the feature is named
		"may also mount",              // conditional framing: not every project has one
		"source_kind: \"global\"",     // the tag agents must look for to spot a global
		"never edit or relate to one", // read-only + relation constraint
	} {
		if !strings.Contains(instructionsBody, want) {
			t.Errorf("instructionsBody missing %q", want)
		}
	}
	// The nudge must stay conditional — it must not assert every repo mounts a global.
	for _, bad := range []string{"This repo mounts", "This repo also mounts"} {
		if strings.Contains(instructionsBody, bad) {
			t.Errorf("globals nudge must be conditional, found unconditional phrasing %q", bad)
		}
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
			got := readFile(t, path)
			if !strings.Contains(got, "## Archcore — project context for this repo") {
				t.Error("nudge body missing")
			}
			// Every host must ship the global-sources nudge, so it lands for all
			// agents at `archcore init` / `instructions install`.
			if !strings.Contains(got, "global sources") {
				t.Error("global-sources nudge missing from this host's instruction file")
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
