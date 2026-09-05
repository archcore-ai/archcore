package wiring

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHooksInstall_NoArchcoreDir(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	err := InstallClaudeCodeHooks(base)
	if err == nil {
		t.Fatal("expected error without .archcore/")
	}
	if got := err.Error(); got != ".archcore/ not found — run 'archcore init' first" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestRunHooksInstall_FreshInstall(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatalf("InstallClaudeCodeHooks: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	var hooks map[string][]hookMatcherGroup
	if err := json.Unmarshal(raw["hooks"], &hooks); err != nil {
		t.Fatalf("Unmarshal hooks: %v", err)
	}

	for _, event := range []string{"SessionStart"} {
		matchers, ok := hooks[event]
		if !ok {
			t.Errorf("missing hook event %s", event)
			continue
		}
		if len(matchers) != 1 {
			t.Errorf("event %s: want 1 matcher, got %d", event, len(matchers))
			continue
		}
		if len(matchers[0].Hooks) != 1 {
			t.Errorf("event %s: want 1 hook entry, got %d", event, len(matchers[0].Hooks))
			continue
		}
		if matchers[0].Hooks[0].Type != "command" {
			t.Errorf("event %s: want type 'command', got %q", event, matchers[0].Hooks[0].Type)
		}
	}
}

func TestRunHooksInstall_Idempotent(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatalf("first install: %v", err)
	}
	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatalf("second install: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}

	var hooks map[string][]hookMatcherGroup
	if err := json.Unmarshal(raw["hooks"], &hooks); err != nil {
		t.Fatal(err)
	}

	// Each event should still have exactly 1 matcher (not duplicated).
	for _, event := range []string{"SessionStart"} {
		if len(hooks[event]) != 1 {
			t.Errorf("event %s: want 1 matcher after idempotent install, got %d", event, len(hooks[event]))
		}
	}
}

func TestRunHooksInstall_MergesExisting(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	// Pre-populate .claude/settings.json with an unrelated key.
	claudeDir := filepath.Join(base, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := map[string]any{
		"model": "claude-sonnet-4-6",
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatalf("InstallClaudeCodeHooks: %v", err)
	}

	resultData, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(resultData, &raw); err != nil {
		t.Fatal(err)
	}

	// "model" key should be preserved.
	if _, ok := raw["model"]; !ok {
		t.Error("existing 'model' key was lost during merge")
	}
	// "hooks" key should exist.
	if _, ok := raw["hooks"]; !ok {
		t.Error("'hooks' key missing after install")
	}
}

func TestRunHooksInstall_PreservesExistingHooks(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	// Pre-populate with a non-archcore hook.
	claudeDir := filepath.Join(base, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := map[string]any{
		"hooks": map[string]any{
			"SessionStart": []interface{}{
				map[string]any{
					"matcher": "",
					"hooks": []interface{}{
						map[string]any{
							"type":    "command",
							"command": "echo hello",
						},
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatalf("InstallClaudeCodeHooks: %v", err)
	}

	resultData, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(resultData, &raw); err != nil {
		t.Fatal(err)
	}

	var hooks map[string][]hookMatcherGroup
	if err := json.Unmarshal(raw["hooks"], &hooks); err != nil {
		t.Fatal(err)
	}

	// SessionStart should have 2 matchers: the existing "echo hello" + archcore.
	if len(hooks["SessionStart"]) != 2 {
		t.Errorf("SessionStart: want 2 matchers, got %d", len(hooks["SessionStart"]))
	}
}

func TestRunHooksInstall_PreservesKeyOrder(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	claudeDir := filepath.Join(base, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write settings with a specific key order: permissions, then model.
	original := `{
  "permissions": ["allow"],
  "model": "claude-sonnet-4-6"
}
`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatalf("InstallClaudeCodeHooks: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}

	result := string(data)

	// "permissions" should appear before "model", and "model" before "hooks" (appended).
	permIdx := strings.Index(result, `"permissions"`)
	modelIdx := strings.Index(result, `"model"`)
	hooksIdx := strings.Index(result, `"hooks"`)

	if permIdx == -1 || modelIdx == -1 || hooksIdx == -1 {
		t.Fatalf("missing expected keys in output:\n%s", result)
	}
	if permIdx >= modelIdx {
		t.Errorf("permissions (%d) should appear before model (%d):\n%s", permIdx, modelIdx, result)
	}
	if modelIdx >= hooksIdx {
		t.Errorf("model (%d) should appear before hooks (%d) (appended):\n%s", modelIdx, hooksIdx, result)
	}
}

func TestRunHooksInstall_CorruptedJSON(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	claudeDir := filepath.Join(base, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	corrupted := []byte("not json")
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, corrupted, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// .bak should contain original corrupted content.
	bakData, err := os.ReadFile(settingsPath + ".bak")
	if err != nil {
		t.Fatalf("ReadFile .bak: %v", err)
	}
	if string(bakData) != string(corrupted) {
		t.Errorf("bak content = %q, want %q", bakData, corrupted)
	}

	// settings.json should be valid with hooks installed.
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["hooks"]; !ok {
		t.Error("'hooks' key missing after recovery")
	}
}

func TestRunHooksInstall_PreservesHookEventOrder(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	claudeDir := filepath.Join(base, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write settings with hooks in a specific order: PreToolUse first.
	original := `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [{"type": "command", "command": "echo pre"}]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "",
        "hooks": [{"type": "command", "command": "echo post"}]
      }
    ]
  }
}
`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatalf("InstallClaudeCodeHooks: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}

	result := string(data)

	// PreToolUse and PostToolUse should keep their original order,
	// with archcore events appended after.
	preIdx := strings.Index(result, `"PreToolUse"`)
	postIdx := strings.Index(result, `"PostToolUse"`)
	sessionIdx := strings.Index(result, `"SessionStart"`)

	if preIdx == -1 || postIdx == -1 || sessionIdx == -1 {
		t.Fatalf("missing expected keys in output:\n%s", result)
	}
	if preIdx >= postIdx {
		t.Errorf("PreToolUse (%d) should appear before PostToolUse (%d):\n%s", preIdx, postIdx, result)
	}
	if postIdx >= sessionIdx {
		t.Errorf("PostToolUse (%d) should appear before SessionStart (%d) (appended):\n%s", postIdx, sessionIdx, result)
	}
}
