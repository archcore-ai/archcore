package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

// TestOpenCode_WriteMCPConfig_PreservesExistingArchcore pins the "already
// configured" guard: a second write must not overwrite an existing, possibly
// user-customized archcore entry. The idempotency test (len==1) cannot detect
// the guard's removal because a guard-less re-marshal keeps the count at one.
func TestOpenCode_WriteMCPConfig_PreservesExistingArchcore(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	const sentinel = "SENTINEL-DO-NOT-OVERWRITE"
	seed := `{"mcp":{"archcore":{"type":"` + sentinel + `"}}}`
	if err := os.WriteFile(filepath.Join(base, "opencode.json"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ByID(OpenCode).WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), sentinel) {
		t.Errorf("existing archcore entry was overwritten; file = %s", data)
	}
}

// TestOpenCode_WriteMCPConfig_CorruptedJSON pins the delegation to the shared
// ADR policy: a corrupted opencode.json is backed up as .bak and replaced with
// a fresh archcore-only config (previously a hard failure).
func TestOpenCode_WriteMCPConfig_CorruptedJSON(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	original := `{not json`
	if err := os.WriteFile(filepath.Join(base, "opencode.json"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := writeOpenCodeMCPConfig(base); err != nil {
		t.Fatalf("corrupted config must be backed up, not fail: %v", err)
	}
	bak, err := os.ReadFile(filepath.Join(base, "opencode.json.bak"))
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}
	if string(bak) != original {
		t.Errorf(".bak = %q, want original bytes", bak)
	}
	data, err := os.ReadFile(filepath.Join(base, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		MCP map[string]json.RawMessage `json:"mcp"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.MCP["archcore"]; !ok {
		t.Error("fresh config must contain the archcore entry")
	}
}

// TestOpenCode_WriteMCPConfig_NullMCPSection pins `"mcp": null` handling
// (previously a nil-map assignment panic).
func TestOpenCode_WriteMCPConfig_NullMCPSection(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "opencode.json"), []byte(`{"mcp": null}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := writeOpenCodeMCPConfig(base); err != nil {
		t.Fatalf("null mcp section must not fail: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(base, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		MCP map[string]json.RawMessage `json:"mcp"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.MCP["archcore"]; !ok {
		t.Error("archcore entry must be added over a null section")
	}
}
