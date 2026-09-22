package wiring

import (
	"os"
	"path/filepath"
	"testing"

	"archcore-cli/internal/agents"
)

// Apply's report must carry the FULL instruction touch-set per agent — the
// primary InstructionsPath plus every ExtraInstructionsPaths entry. This is
// the domain-level guarantee behind the install_host_config MCP report: the
// cmd adapter only renders what AgentResult carries, so an omission here is
// invisible to every MCP client.
func TestApply_ClaudeCode_ReportCarriesExtraInstructions(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	report, err := Apply(base, agents.ClaudeCode, false)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(report.Agents) != 1 {
		t.Fatalf("want 1 agent result, got %+v", report.Agents)
	}

	r := report.Agents[0]
	if len(r.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", r.Errors)
	}
	if want := filepath.Join(base, "CLAUDE.md"); r.Instructions != want {
		t.Errorf("Instructions = %q, want %q", r.Instructions, want)
	}
	if want := filepath.Join(base, "AGENTS.md"); len(r.ExtraInstructions) != 1 || r.ExtraInstructions[0] != want {
		t.Errorf("ExtraInstructions = %v, want [%s]", r.ExtraInstructions, want)
	}
}

// Single-target agents must NOT grow a spurious extras list — the field is
// omitted, not an empty echo of the primary path.
func TestApply_Cursor_ReportHasNoExtraInstructions(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	report, err := Apply(base, agents.Cursor, false)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(report.Agents) != 1 {
		t.Fatalf("want 1 agent result, got %+v", report.Agents)
	}

	r := report.Agents[0]
	if want := filepath.Join(base, "AGENTS.md"); r.Instructions != want {
		t.Errorf("Instructions = %q, want %q", r.Instructions, want)
	}
	if len(r.ExtraInstructions) != 0 {
		t.Errorf("ExtraInstructions = %v, want none for a single-target agent", r.ExtraInstructions)
	}
}

// A multi-file write that fails partway must report the file that DID land on
// disk, not treat the whole agent as unwritten. claude-code writes CLAUDE.md
// first, then AGENTS.md; with AGENTS.md forced to fail, the report must still
// name CLAUDE.md (it is in the user's repo) AND surface the error — a report
// that omits CLAUDE.md would be a silent write, exactly what the MCP contract
// forbids.
func TestApply_ClaudeCode_PartialWrite_ReportsWrittenFileAndError(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	// AGENTS.md is a directory → its upsert fails after CLAUDE.md succeeds.
	if err := os.MkdirAll(filepath.Join(base, "AGENTS.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	report, err := Apply(base, agents.ClaudeCode, false)
	if err != nil {
		t.Fatalf("Apply hard error: %v", err)
	}
	if len(report.Agents) != 1 {
		t.Fatalf("want 1 agent result, got %+v", report.Agents)
	}
	r := report.Agents[0]

	// Precondition: CLAUDE.md really was written.
	if _, statErr := os.Stat(filepath.Join(base, "CLAUDE.md")); statErr != nil {
		t.Fatalf("precondition: CLAUDE.md should have been written: %v", statErr)
	}
	if want := filepath.Join(base, "CLAUDE.md"); r.Instructions != want {
		t.Errorf("Instructions = %q, want %q (the file that landed on disk)", r.Instructions, want)
	}
	if len(r.ExtraInstructions) != 0 {
		t.Errorf("ExtraInstructions = %v, want none — AGENTS.md write failed", r.ExtraInstructions)
	}
	if !hasInstructionError(r) {
		t.Errorf("the AGENTS.md failure must be surfaced, got errors %+v", r.Errors)
	}
}

// A failed write to a SHARED instruction file must be attributed to every agent
// that shares it, not only the dedupe representative. cursor and opencode both
// map to AGENTS.md; the pre-fix failure path blamed only cursor (the registry-
// order representative) and left opencode with neither a path nor an error —
// a silent gap for a file that failed to write.
func TestApply_SharedAgentsMDFailure_AttributedToEveryAgent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, ".cursor"))   // detected: cursor
	mustMkdir(t, filepath.Join(base, ".opencode")) // detected: opencode
	// Force the single shared AGENTS.md write to fail.
	if err := os.MkdirAll(filepath.Join(base, "AGENTS.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	report, err := Apply(base, agents.Cursor, true)
	if err != nil {
		t.Fatalf("Apply hard error: %v", err)
	}

	for _, id := range []agents.AgentID{agents.Cursor, agents.OpenCode} {
		r := findAgentResult(t, report, id)
		if r.Instructions != "" || len(r.ExtraInstructions) != 0 {
			t.Errorf("%s: no instruction path should be reported for a failed write, got %q %v",
				id, r.Instructions, r.ExtraInstructions)
		}
		if !hasInstructionError(r) {
			t.Errorf("%s: the shared AGENTS.md failure must be attributed to this agent, got errors %+v",
				id, r.Errors)
		}
	}
}

func hasInstructionError(r AgentResult) bool {
	for _, e := range r.Errors {
		if e.Action == "instructions" {
			return true
		}
	}
	return false
}

func findAgentResult(t *testing.T, report Report, id agents.AgentID) AgentResult {
	t.Helper()
	for _, r := range report.Agents {
		if r.Agent == id {
			return r
		}
	}
	t.Fatalf("no result for agent %q in report %+v", id, report.Agents)
	return AgentResult{}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll %s: %v", path, err)
	}
}
