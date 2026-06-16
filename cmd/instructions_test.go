package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/agents"
)

const instructionsStartMarker = "<!-- archcore:start -->"

func TestDedupeByInstructionsPath_AllAgents(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	got := dedupeByInstructionsPath(base, agents.All())

	// Eight agents collapse to three unique targets, in registry order:
	// Claude Code (owned file), Cursor (AGENTS.md), Gemini CLI (GEMINI.md).
	if len(got) != 3 {
		t.Fatalf("want 3 unique targets, got %d", len(got))
	}
	wantOrder := []agents.AgentID{agents.ClaudeCode, agents.Cursor, agents.GeminiCLI}
	for i, want := range wantOrder {
		if got[i].ID != want {
			t.Errorf("target[%d] = %q, want %q", i, got[i].ID, want)
		}
	}
}

func TestInstallInstructionsForAgents_DedupesAndWrites(t *testing.T) {
	// Not parallel: captureStdout reassigns the global os.Stdout.
	base := t.TempDir()
	// Three AGENTS.md agents + Claude (owned file). AGENTS.md must be written once.
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
	if _, err := os.Stat(filepath.Join(base, ".claude", "rules", "archcore.md")); err != nil {
		t.Errorf(".claude/rules/archcore.md missing: %v", err)
	}

	if n := strings.Count(out, "Added Archcore usage hint"); n != 2 {
		t.Errorf("want 2 install lines (AGENTS.md + Claude), got %d:\n%s", n, out)
	}
	if !strings.Contains(out, "AGENTS.md") {
		t.Errorf("output missing AGENTS.md:\n%s", out)
	}
	if !strings.Contains(out, ".claude/rules/archcore.md") {
		t.Errorf("output missing Claude path (forward slashes):\n%s", out)
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

	got := readFileString(t, filepath.Join(base, ".claude", "rules", "archcore.md"))
	if strings.Contains(got, instructionsStartMarker) {
		t.Error("owned Claude file must not contain markers")
	}
	if !strings.Contains(got, "## Archcore — project context for this repo") {
		t.Error("Claude file missing nudge body")
	}
	if !strings.Contains(got, "global sources") {
		t.Error("Claude file missing global-sources nudge")
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

	// AGENTS.md held only our block → deleted; Claude owned file → deleted.
	if _, err := os.Stat(filepath.Join(base, "AGENTS.md")); !os.IsNotExist(err) {
		t.Errorf("AGENTS.md should be deleted after remove, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, ".claude", "rules", "archcore.md")); !os.IsNotExist(err) {
		t.Errorf(".claude file should be deleted after remove, stat err = %v", err)
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

func TestDisplayPath(t *testing.T) {
	t.Parallel()
	base := filepath.Join("home", "repo")
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "agents md", path: filepath.Join(base, "AGENTS.md"), want: "AGENTS.md"},
		{name: "nested claude", path: filepath.Join(base, ".claude", "rules", "archcore.md"), want: ".claude/rules/archcore.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := displayPath(base, tt.path); got != tt.want {
				t.Errorf("displayPath(%q, %q) = %q, want %q", base, tt.path, got, tt.want)
			}
		})
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
	if _, err := os.Stat(filepath.Join(base, ".claude", "rules", "archcore.md")); !os.IsNotExist(err) {
		t.Errorf("file should be removed, stat err = %v", err)
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
