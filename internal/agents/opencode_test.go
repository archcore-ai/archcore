package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCode_WriteMCPConfig_NewFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	agent := ByID(OpenCode)

	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, "opencode.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	var mcp map[string]json.RawMessage
	if err := json.Unmarshal(raw["mcp"], &mcp); err != nil {
		t.Fatalf("Unmarshal mcp: %v", err)
	}

	if _, ok := mcp["archcore"]; !ok {
		t.Error("missing 'archcore' in mcp section")
	}
}

func TestOpenCode_WriteMCPConfig_Format(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	agent := ByID(OpenCode)
	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, "opencode.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var mcp map[string]json.RawMessage
	if err := json.Unmarshal(raw["mcp"], &mcp); err != nil {
		t.Fatalf("Unmarshal mcp: %v", err)
	}

	var entry struct {
		Type    string   `json:"type"`
		Command []string `json:"command"`
	}
	if err := json.Unmarshal(mcp["archcore"], &entry); err != nil {
		t.Fatalf("Unmarshal entry: %v", err)
	}
	if entry.Type != "local" {
		t.Errorf("type = %q, want %q", entry.Type, "local")
	}
	if len(entry.Command) != 2 || entry.Command[0] != "archcore" || entry.Command[1] != "mcp" {
		t.Errorf("command = %v, want [archcore mcp]", entry.Command)
	}
}

func TestOpenCode_WriteMCPConfig_Idempotent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	agent := ByID(OpenCode)

	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("first WriteMCPConfig: %v", err)
	}
	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("second WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, "opencode.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var mcp map[string]json.RawMessage
	if err := json.Unmarshal(raw["mcp"], &mcp); err != nil {
		t.Fatalf("Unmarshal mcp: %v", err)
	}

	if len(mcp) != 1 {
		t.Errorf("expected 1 mcp entry, got %d", len(mcp))
	}
}

func TestOpenCode_WriteMCPConfig_MergesExisting(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	existing := map[string]any{
		"mcp": map[string]any{
			"other": map[string]any{"type": "local", "command": []string{"other"}},
		},
		"theme": "dark",
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "opencode.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	agent := ByID(OpenCode)
	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	result, err2 := os.ReadFile(filepath.Join(base, "opencode.json"))
	if err2 != nil {
		t.Fatalf("ReadFile: %v", err2)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(result, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if _, ok := raw["theme"]; !ok {
		t.Error("existing 'theme' key was lost")
	}

	var mcp map[string]json.RawMessage
	if err := json.Unmarshal(raw["mcp"], &mcp); err != nil {
		t.Fatalf("Unmarshal mcp: %v", err)
	}
	if _, ok := mcp["other"]; !ok {
		t.Error("existing 'other' mcp entry lost")
	}
	if _, ok := mcp["archcore"]; !ok {
		t.Error("archcore not added")
	}
}

func TestOpenCode_Detect_JsonFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "opencode.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if !ByID(OpenCode).DetectFn(base) {
		t.Error("expected detection with opencode.json")
	}
}

func TestOpenCode_Detect_Dir(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".opencode"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if !ByID(OpenCode).DetectFn(base) {
		t.Error("expected detection with .opencode/")
	}
}

func TestOpenCode_Detect_False(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if ByID(OpenCode).DetectFn(base) {
		t.Error("expected no detection")
	}
}
