// Per-host event wiring.
//
// Three properties this file guards, each a bug that would otherwise stay
// invisible until a user reported that hooks "stopped working":
//
//  1. A matcher change reaches the host. The ownership probe used to compare
//     only the command, so widening a matcher between releases printed
//     "already installed" and the host kept filtering on the old pattern.
//  2. One archcore entry per event. Two of ours under one key would each
//     classify the other as a stale duplicate, and the second would silently
//     overwrite the first.
//  3. Each host gets ITS event names. They are not variations on a theme —
//     Gemini's tool events are called something else entirely.
package wiring

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readHooksSection(t *testing.T, base, relPath string) map[string][]json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(configPathFor(base, relPath))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Hooks map[string][]json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parsing %s: %v\n%s", relPath, err, data)
	}
	return doc.Hooks
}

func TestHooksInstall_PerHostEventNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		install  func(string) error
		relPath  string
		wantKeys []string
	}{
		{
			name: "claude code", install: InstallClaudeCodeHooks, relPath: ".claude/settings.json",
			wantKeys: []string{"SessionStart", "PreToolUse", "PostToolUse"},
		},
		{
			name: "codex cli", install: InstallCodexCLIHooks, relPath: ".codex/hooks.json",
			wantKeys: []string{"SessionStart", "PreToolUse", "PostToolUse"},
		},
		{
			name: "cursor", install: InstallCursorHooks, relPath: ".cursor/hooks.json",
			// MCP results arrive on a dedicated event here, not on postToolUse.
			wantKeys: []string{"sessionStart", "preToolUse", "afterMCPExecution"},
		},
		{
			name: "copilot", install: InstallCopilotHooks, relPath: ".github/hooks/archcore.json",
			wantKeys: []string{"sessionStart", "preToolUse", "postToolUse"},
		},
		{
			name: "gemini cli", install: InstallGeminiCLIHooks, relPath: ".gemini/settings.json",
			// Gemini names its tool events BeforeTool / AfterTool.
			wantKeys: []string{"SessionStart", "BeforeTool", "AfterTool"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)
			if err := tt.install(base); err != nil {
				t.Fatalf("install: %v", err)
			}
			hooks := readHooksSection(t, base, tt.relPath)
			for _, key := range tt.wantKeys {
				entries, ok := hooks[key]
				if !ok {
					t.Errorf("missing event %q; got %v", key, keysOf(hooks))
					continue
				}
				if len(entries) != 1 {
					t.Errorf("event %q has %d entries, want exactly 1", key, len(entries))
				}
			}
		})
	}
}

// TestHooksInstall_MatcherChangeReachesConfig is the regression guard for the
// probe fix. Before it, only the command was compared.
func TestHooksInstall_MatcherChangeReachesConfig(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	// An entry with the CURRENT command but a stale matcher, as a narrower
	// earlier release would have written it.
	seedConfig(t, base, ".claude/settings.json", `{
  "hooks": {
    "PreToolUse": [
      {"matcher": "Write", "hooks": [{"type": "command", "command": "archcore hooks claude-code pre-tool-use"}]}
    ]
  }
}`)

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatal(err)
	}

	entries := readHooksSection(t, base, ".claude/settings.json")["PreToolUse"]
	if len(entries) != 1 {
		t.Fatalf("PreToolUse has %d entries, want 1", len(entries))
	}
	if !strings.Contains(string(entries[0]), `Write|Edit`) {
		t.Errorf("matcher change did not reach the config:\n%s", entries[0])
	}
}

// TestInstallHookEvents_RejectsDuplicateEventKeys: the writer's one-entry-per-
// event invariant, enforced rather than assumed.
func TestInstallHookEvents_RejectsDuplicateEventKeys(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	spec := hookInstallSpec{
		Path:  filepath.Join(base, ".claude", "settings.json"),
		Probe: matcherEntryHasCommand,
		Events: []hookEventInstall{
			{Event: "PostToolUse", Command: "archcore hooks claude-code post-tool-use", Entry: hookMatcher{}},
			{Event: "PostToolUse", Command: "archcore hooks claude-code other", Entry: hookMatcher{}},
		},
	}

	err := installHookEvents(spec)

	if err == nil {
		t.Fatal("duplicate event keys were accepted")
	}
	if !strings.Contains(err.Error(), "twice") {
		t.Errorf("error should name the duplication: %v", err)
	}
}

// TestHooksInstall_ForeignEntriesSurviveEveryEvent: adding two events must not
// weaken the promise that someone else's hooks are never touched.
func TestHooksInstall_ForeignEntriesSurviveEveryEvent(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	seedConfig(t, base, ".claude/settings.json", `{
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "other-tool guard"}]}
    ],
    "PostToolUse": [
      {"matcher": "", "hooks": [{"type": "command", "command": "other-tool report"}]}
    ]
  }
}`)

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatal(err)
	}

	hooks := readHooksSection(t, base, ".claude/settings.json")
	for event, wantForeign := range map[string]string{"PreToolUse": "other-tool guard", "PostToolUse": "other-tool report"} {
		joined := ""
		for _, e := range hooks[event] {
			joined += string(e)
		}
		if !strings.Contains(joined, wantForeign) {
			t.Errorf("%s: foreign entry %q was dropped:\n%s", event, wantForeign, joined)
		}
		if !strings.Contains(joined, "archcore hooks claude-code") {
			t.Errorf("%s: archcore entry was not added:\n%s", event, joined)
		}
	}
}

func keysOf(m map[string][]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
