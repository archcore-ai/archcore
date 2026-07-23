package wiring

import (
	"path/filepath"
	"testing"

	"archcore-cli/internal/agents"
)

func TestDedupeByInstructionsPath_AllAgents(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	got := DedupeByInstructionsPath(base, agents.All())

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
			if got := DisplayPath(base, tt.path); got != tt.want {
				t.Errorf("DisplayPath(%q, %q) = %q, want %q", base, tt.path, got, tt.want)
			}
		})
	}
}
