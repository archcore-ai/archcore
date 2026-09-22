package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllAgents_UniqueIDs(t *testing.T) {
	t.Parallel()
	seen := make(map[AgentID]bool)
	for _, a := range All() {
		if seen[a.ID] {
			t.Errorf("duplicate agent ID: %s", a.ID)
		}
		seen[a.ID] = true
	}
}

func TestAllAgents_RequiredFields(t *testing.T) {
	t.Parallel()
	for _, a := range All() {
		t.Run(string(a.ID), func(t *testing.T) {
			t.Parallel()
			if a.DisplayName == "" {
				t.Error("missing DisplayName")
			}
			if a.DetectFn == nil {
				t.Error("missing DetectFn")
			}
			if a.WriteMCPConfig == nil {
				t.Error("missing WriteMCPConfig")
			}
			if a.InstructionsPath == nil {
				t.Error("missing InstructionsPath")
			}
			if a.WriteInstructions == nil {
				t.Error("missing WriteInstructions")
			}
			if a.RemoveInstructions == nil {
				t.Error("missing RemoveInstructions")
			}
		})
	}
}

func TestByID_Found(t *testing.T) {
	t.Parallel()
	for _, id := range AllIDs() {
		t.Run(string(id), func(t *testing.T) {
			t.Parallel()
			a := ByID(id)
			if a == nil {
				t.Fatalf("ByID(%q) returned nil", id)
			}
			if a.ID != id {
				t.Errorf("ByID(%q).ID = %q", id, a.ID)
			}
		})
	}
}

func TestByID_NotFound(t *testing.T) {
	t.Parallel()
	if a := ByID("nonexistent"); a != nil {
		t.Errorf("expected nil for unknown ID, got %v", a)
	}
}

func TestAllIDs_Complete(t *testing.T) {
	t.Parallel()
	ids := AllIDs()
	agents := All()
	if len(ids) != len(agents) {
		t.Fatalf("AllIDs() returned %d IDs, but All() has %d agents", len(ids), len(agents))
	}
	for i, id := range ids {
		if id != agents[i].ID {
			t.Errorf("AllIDs()[%d] = %q, want %q", i, id, agents[i].ID)
		}
	}
}

func TestDetect_NoAgents(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	detected := Detect(base)
	if len(detected) != 0 {
		t.Errorf("expected 0 agents, got %d", len(detected))
	}
}

func TestDetect_SingleAgent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}

	detected := Detect(base)
	if len(detected) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(detected))
	}
	if detected[0].ID != Cursor {
		t.Errorf("expected Cursor, got %s", detected[0].ID)
	}
}

func TestDetect_MultipleAgents(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	for _, dir := range []string{".claude", ".gemini", ".roo"} {
		if err := os.MkdirAll(filepath.Join(base, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	detected := Detect(base)
	if len(detected) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(detected))
	}

	ids := make(map[AgentID]bool)
	for _, a := range detected {
		ids[a.ID] = true
	}
	for _, want := range []AgentID{ClaudeCode, GeminiCLI, RooCode} {
		if !ids[want] {
			t.Errorf("expected %s in detected agents", want)
		}
	}
}

func TestDetect_AllAgents(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	for _, dir := range []string{".claude", ".cursor", ".gemini", ".codex", ".roo", ".clinerules", ".github"} {
		if err := os.MkdirAll(filepath.Join(base, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(base, "opencode.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, ".github", "copilot-instructions.md"), []byte("# Copilot"), 0o644); err != nil {
		t.Fatal(err)
	}

	detected := Detect(base)
	if len(detected) != 8 {
		t.Errorf("expected 8 agents, got %d", len(detected))
	}
}

func TestDetect_OpenCode_JsonFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "opencode.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	detected := Detect(base)
	if len(detected) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(detected))
	}
	if detected[0].ID != OpenCode {
		t.Errorf("expected OpenCode, got %s", detected[0].ID)
	}
}

func TestDetect_OpenCode_Dir(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".opencode"), 0o755); err != nil {
		t.Fatal(err)
	}

	detected := Detect(base)
	if len(detected) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(detected))
	}
	if detected[0].ID != OpenCode {
		t.Errorf("expected OpenCode, got %s", detected[0].ID)
	}
}
