package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/agents"
)

const instructionsStartMarker = "<!-- archcore:start -->"

func TestInstallInstructionsForAgents_DedupesAndWrites(t *testing.T) {
	// Not parallel: captureStdout reassigns the global os.Stdout.
	base := t.TempDir()
	// Three AGENTS.md agents + Claude (CLAUDE.md + AGENTS.md). Claude dedupes on
	// its own CLAUDE.md path; the AGENTS.md block is written once (idempotent
	// upsert), and Claude's CLAUDE.md adds one more install line.
	list := []*agents.Agent{
		agents.ByID(agents.CodexCLI),
		agents.ByID(agents.Cursor),
		agents.ByID(agents.Cline),
		agents.ByID(agents.ClaudeCode),
	}

	out := captureStdout(t, func() {
		installInstructionsForAgents(base, list)
	})

	agentsMD := readFileString(t, filepath.Join(base, "AGENTS.md"))
	if n := strings.Count(agentsMD, instructionsStartMarker); n != 1 {
		t.Errorf("AGENTS.md should have exactly 1 managed block, got %d", n)
	}
	if _, err := os.Stat(filepath.Join(base, "CLAUDE.md")); err != nil {
		t.Errorf("CLAUDE.md missing: %v", err)
	}

	if n := strings.Count(out, "Added Archcore usage hint"); n != 2 {
		t.Errorf("want 2 install lines (AGENTS.md + CLAUDE.md), got %d:\n%s", n, out)
	}
	if !strings.Contains(out, "AGENTS.md") {
		t.Errorf("output missing AGENTS.md:\n%s", out)
	}
	if !strings.Contains(out, "CLAUDE.md") {
		t.Errorf("output missing Claude path:\n%s", out)
	}
}

func TestRunInstructionsInstallForAgent_Unknown(t *testing.T) {
	t.Parallel()
	err := runInstructionsInstallForAgent(t.TempDir(), agents.AgentID("nope"))
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
	if !strings.Contains(err.Error(), "unknown agent") {
		t.Errorf("error %q should mention 'unknown agent'", err.Error())
	}
}

func TestRunInstructionsInstallForAgent_WriteError(t *testing.T) {
	baseFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(baseFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := runInstructionsInstallForAgent(baseFile, agents.ClaudeCode)
	if err == nil {
		t.Fatal("expected write error")
	}
	if !strings.Contains(err.Error(), "writing Claude Code instructions") {
		t.Errorf("error %q should mention Claude instructions write", err.Error())
	}
}

func TestRunInstructionsInstallForAgent_Claude(t *testing.T) {
	// Not parallel: captureStdout reassigns the global os.Stdout.
	base := t.TempDir()

	captureStdout(t, func() {
		if err := runInstructionsInstallForAgent(base, agents.ClaudeCode); err != nil {
			t.Fatalf("runInstructionsInstallForAgent: %v", err)
		}
	})

	claudeMD := readFileString(t, filepath.Join(base, "CLAUDE.md"))
	if !strings.Contains(claudeMD, instructionsStartMarker) {
		t.Error("CLAUDE.md should carry the fenced block")
	}
	if !strings.Contains(claudeMD, "## Archcore — project context for this repo") {
		t.Error("CLAUDE.md missing nudge body")
	}
	if !strings.Contains(claudeMD, "global sources") {
		t.Error("CLAUDE.md missing global-sources nudge")
	}

	agentsMD := readFileString(t, filepath.Join(base, "AGENTS.md"))
	if !strings.Contains(agentsMD, instructionsStartMarker) {
		t.Error("AGENTS.md should carry the fenced block for Claude Code")
	}
}

// TestRemoveInstructionsForAgent_Claude_PreservesSharedAgentsMD pins the remove
// asymmetry: removing Claude Code alone strips its own CLAUDE.md block but must
// NOT strip the shared AGENTS.md block a co-installed AGENTS.md agent (Cursor
// here) still relies on. A "clean up everything" refactor of
// removeClaudeInstructions would break a co-installed user with no other test
// failing — this is the guard against it.
func TestRemoveInstructionsForAgent_Claude_PreservesSharedAgentsMD(t *testing.T) {
	// Not parallel: captureStdout reassigns the global os.Stdout.
	base := t.TempDir()
	captureStdout(t, func() {
		installInstructionsForAgents(base, []*agents.Agent{
			agents.ByID(agents.Cursor), agents.ByID(agents.ClaudeCode),
		})
		if err := removeInstructionsForAgent(base, agents.ByID(agents.ClaudeCode)); err != nil {
			t.Fatalf("remove claude: %v", err)
		}
	})

	// CLAUDE.md held only our block → deleted.
	if _, err := os.Stat(filepath.Join(base, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Errorf("CLAUDE.md should be removed, stat err = %v", err)
	}
	// The shared AGENTS.md block MUST survive for the still-installed Cursor.
	agentsMD := readFileString(t, filepath.Join(base, "AGENTS.md"))
	if n := strings.Count(agentsMD, instructionsStartMarker); n != 1 {
		t.Errorf("removing Claude alone must leave Cursor's AGENTS.md block intact, got %d blocks:\n%s", n, agentsMD)
	}
}

// TestRemoveInstructionsForAgent_ClaudeOnly_LeavesAgentsMD pins the claude-only
// remove residue: removing Claude Code alone deliberately leaves the AGENTS.md
// block; only the no-flag "remove all" path cleans it up (see the nudge ADR).
func TestRemoveInstructionsForAgent_ClaudeOnly_LeavesAgentsMD(t *testing.T) {
	// Not parallel: captureStdout reassigns the global os.Stdout.
	base := t.TempDir()
	captureStdout(t, func() {
		installInstructionsForAgents(base, []*agents.Agent{agents.ByID(agents.ClaudeCode)})
		if err := removeInstructionsForAgent(base, agents.ByID(agents.ClaudeCode)); err != nil {
			t.Fatalf("remove claude: %v", err)
		}
	})

	if _, err := os.Stat(filepath.Join(base, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Errorf("CLAUDE.md should be removed, stat err = %v", err)
	}
	agentsMD := readFileString(t, filepath.Join(base, "AGENTS.md"))
	if n := strings.Count(agentsMD, instructionsStartMarker); n != 1 {
		t.Errorf("claude-only remove leaves the AGENTS.md block by design, got %d blocks:\n%s", n, agentsMD)
	}
}

// TestInstallInstructionsForAgents_ClaudePlusAgentsMD_Idempotent guards that the
// composed dual write (Claude's CLAUDE.md + the AGENTS.md upsert) is byte-stable
// across re-installs — init is commonly re-run, so a second pass must not append
// or drift either file.
func TestInstallInstructionsForAgents_ClaudePlusAgentsMD_Idempotent(t *testing.T) {
	// Not parallel: captureStdout reassigns the global os.Stdout.
	base := t.TempDir()
	list := []*agents.Agent{agents.ByID(agents.Cursor), agents.ByID(agents.ClaudeCode)}

	captureStdout(t, func() { installInstructionsForAgents(base, list) })
	firstAgents := readFileString(t, filepath.Join(base, "AGENTS.md"))
	firstClaude := readFileString(t, filepath.Join(base, "CLAUDE.md"))

	captureStdout(t, func() { installInstructionsForAgents(base, list) })
	if got := readFileString(t, filepath.Join(base, "AGENTS.md")); got != firstAgents {
		t.Errorf("AGENTS.md not byte-identical on re-install:\n%q\n%q", firstAgents, got)
	}
	if got := readFileString(t, filepath.Join(base, "CLAUDE.md")); got != firstClaude {
		t.Errorf("CLAUDE.md not byte-identical on re-install")
	}
}

func TestRemoveInstructionsForAgents_RoundTrip(t *testing.T) {
	// Not parallel: captureStdout reassigns the global os.Stdout.
	base := t.TempDir()
	list := []*agents.Agent{agents.ByID(agents.Cursor), agents.ByID(agents.ClaudeCode)}

	captureStdout(t, func() {
		installInstructionsForAgents(base, list)
		removeInstructionsForAgents(base, agents.All())
	})

	// AGENTS.md held only our block → deleted; Claude's CLAUDE.md → deleted.
	if _, err := os.Stat(filepath.Join(base, "AGENTS.md")); !os.IsNotExist(err) {
		t.Errorf("AGENTS.md should be deleted after remove, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Errorf("CLAUDE.md should be deleted after remove, stat err = %v", err)
	}
}

func TestRemoveInstructionsForAgents_PreservesUserContent(t *testing.T) {
	// Not parallel: captureStdout reassigns the global os.Stdout.
	base := t.TempDir()
	agentsMD := filepath.Join(base, "AGENTS.md")
	if err := os.WriteFile(agentsMD, []byte("# Mine\n\nkeep this.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	captureStdout(t, func() {
		installInstructionsForAgents(base, []*agents.Agent{agents.ByID(agents.Cursor)})
		removeInstructionsForAgents(base, []*agents.Agent{agents.ByID(agents.Cursor)})
	})

	got := readFileString(t, agentsMD)
	if strings.Contains(got, instructionsStartMarker) {
		t.Error("block should be removed")
	}
	if !strings.Contains(got, "keep this.") {
		t.Error("user content should survive remove")
	}
}

func TestRunInstructionsRemoveForAgent_Unknown(t *testing.T) {
	t.Parallel()
	err := runInstructionsRemoveForAgent(t.TempDir(), agents.AgentID("nope"))
	if err == nil || !strings.Contains(err.Error(), "unknown agent") {
		t.Fatalf("want unknown-agent error, got %v", err)
	}
}

func TestRunInstructionsRemoveForAgent_RemoveError(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, ".claude", "rules", "archcore.md")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(path, "child"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := runInstructionsRemoveForAgent(base, agents.ClaudeCode)
	if err == nil {
		t.Fatal("expected remove error")
	}
	if !strings.Contains(err.Error(), "removing Claude Code instructions") {
		t.Errorf("error %q should mention Claude instructions remove", err.Error())
	}
}

func TestRunInstructionsRemoveForAgent_Claude(t *testing.T) {
	// Not parallel: captureStdout reassigns the global os.Stdout.
	base := t.TempDir()
	captureStdout(t, func() {
		if err := runInstructionsInstallForAgent(base, agents.ClaudeCode); err != nil {
			t.Fatalf("install: %v", err)
		}
		if err := runInstructionsRemoveForAgent(base, agents.ClaudeCode); err != nil {
			t.Fatalf("remove: %v", err)
		}
	})
	if _, err := os.Stat(filepath.Join(base, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Errorf("CLAUDE.md should be removed, stat err = %v", err)
	}
}

func TestRunInstructionsInstallAutoDetect_DetectedAgent(t *testing.T) {
	// Not parallel: captureStdout reassigns the global os.Stdout.
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".cursor"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	out := captureStdout(t, func() {
		if err := runInstructionsInstallAutoDetect(base); err != nil {
			t.Fatalf("runInstructionsInstallAutoDetect: %v", err)
		}
	})

	if _, err := os.Stat(filepath.Join(base, "AGENTS.md")); err != nil {
		t.Errorf("AGENTS.md not written for detected agent: %v", err)
	}
	if !strings.Contains(out, "Added Archcore usage hint") {
		t.Errorf("expected install confirmation:\n%s", out)
	}
}

func TestRunInstructionsInstallAutoDetect_SkipShowsInstructionsHint(t *testing.T) {
	base := t.TempDir()
	withPickAgents(t, agentSelection{outcome: outcomeSkipped})

	out := captureStdout(t, func() {
		if err := runInstructionsInstallAutoDetect(base); err != nil {
			t.Fatalf("runInstructionsInstallAutoDetect: %v", err)
		}
	})

	if !strings.Contains(out, "archcore instructions install --agent <id>") {
		t.Errorf("output missing instructions recovery hint:\n%s", out)
	}
	if strings.Contains(out, "archcore mcp install") || strings.Contains(out, "archcore hooks install") {
		t.Errorf("output should not mention mcp/hooks recovery hints:\n%s", out)
	}
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", path, err)
	}
	return string(data)
}
