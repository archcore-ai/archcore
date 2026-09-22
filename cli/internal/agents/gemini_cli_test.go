package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGeminiCLI_WriteMCPConfig_NewFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	agent := ByID(GeminiCLI)

	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".gemini", "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["mcpServers"]; !ok {
		t.Error("missing mcpServers key")
	}
}

func TestGeminiCLI_WriteMCPConfig_Idempotent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	agent := ByID(GeminiCLI)

	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("first WriteMCPConfig: %v", err)
	}
	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("second WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".gemini", "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw["mcpServers"], &servers); err != nil {
		t.Fatalf("Unmarshal mcpServers: %v", err)
	}

	if len(servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(servers))
	}
}

func TestGeminiCLI_WriteMCPConfig_MergesExisting(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	geminiDir := filepath.Join(base, ".gemini")
	if err := os.MkdirAll(geminiDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Pre-populate with hooks (to simulate shared settings file).
	existing := map[string]any{
		"hooks": map[string]any{
			"SessionStart": []any{},
		},
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(geminiDir, "settings.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	agent := ByID(GeminiCLI)
	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	result, err2 := os.ReadFile(filepath.Join(geminiDir, "settings.json"))
	if err2 != nil {
		t.Fatalf("ReadFile: %v", err2)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(result, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if _, ok := raw["hooks"]; !ok {
		t.Error("existing hooks key was lost")
	}
	if _, ok := raw["mcpServers"]; !ok {
		t.Error("mcpServers not added")
	}
}

func TestGeminiCLI_Detect_True(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".gemini"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if !ByID(GeminiCLI).DetectFn(base) {
		t.Error("expected detection")
	}
}

func TestGeminiCLI_Detect_False(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if ByID(GeminiCLI).DetectFn(base) {
		t.Error("expected no detection")
	}
}
