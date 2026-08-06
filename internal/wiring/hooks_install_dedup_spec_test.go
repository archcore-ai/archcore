package wiring

// TDD spec pinning a CONFIRMED defect in the hook installers' idempotency
// probes (matcherEntryHasCommand / commandEntryHasCommand /
// bashEntryHasCommand in hooks_install.go):
//
//	The probes match by EXACT command string. If the installed command ever
//	changes between CLI versions (e.g. "archcore hooks claude-code
//	session-start" gains a flag), a re-run of `archcore hooks install` does
//	not update the stale entry — it appends a second archcore entry, and the
//	host then fires the hook twice per session.
//
// Target behavior: a probe should recognize an EXISTING archcore-owned entry
// by a stable marker (command contains "archcore hooks") and the installer
// should update that entry in place. Foreign entries (no marker) keep the
// current never-touch semantics.
//
// Implemented: probes classify entries (entryForeign / entryCurrent /
// entryStaleArchcore) and installHookEvents updates the first stale entry in
// place, dropping further stale duplicates. Foreign entries keep the
// never-touch semantics.

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

// TestHooksInstall_UpdatesStaleArchcoreEntry_Spec: a config that already
// carries an archcore SessionStart entry with an OUTDATED command string must
// end up with exactly one archcore entry carrying the current command — never
// two archcore entries.
func TestHooksInstall_UpdatesStaleArchcoreEntry_Spec(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	// Arrange: an entry installed by a hypothetical older CLI — same marker
	// ("archcore hooks"), different exact string than today's
	// "archcore hooks claude-code session-start".
	seedConfig(t, base, ".claude/settings.json", `{
  "hooks": {
    "SessionStart": [
      {"matcher": "", "hooks": [{"type": "command", "command": "archcore hooks session-start"}]},
      {"matcher": "", "hooks": [{"type": "command", "command": "some-other-tool session-start"}]}
    ]
  }
}`)

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatalf("InstallClaudeCodeHooks: %v", err)
	}

	// Assert: exactly one archcore-marked SessionStart entry, updated to the
	// current command; the foreign entry untouched.
	data, err := os.ReadFile(configPathFor(base, ".claude/settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Hooks struct {
			SessionStart []struct {
				Hooks []struct {
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"SessionStart"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Unmarshal %s: %v", data, err)
	}

	archcoreEntries := 0
	foreignSurvives := false
	for _, entry := range doc.Hooks.SessionStart {
		for _, h := range entry.Hooks {
			switch {
			case strings.Contains(h.Command, "archcore hooks"):
				archcoreEntries++
				if h.Command != "archcore hooks claude-code session-start" {
					t.Errorf("stale archcore command not updated: %q", h.Command)
				}
			case h.Command == "some-other-tool session-start":
				foreignSurvives = true
			}
		}
	}
	if archcoreEntries != 1 {
		t.Errorf("want exactly 1 archcore SessionStart entry after reinstall over a stale one, got %d:\n%s", archcoreEntries, data)
	}
	if !foreignSurvives {
		t.Errorf("foreign SessionStart entry must survive untouched:\n%s", data)
	}
}

// TestHooksInstall_RemovesStaleDuplicateWhenCurrentPresent: a config bitten by
// the pre-marker bug (current entry AND a stale duplicate) heals to exactly
// one archcore entry.
func TestHooksInstall_RemovesStaleDuplicateWhenCurrentPresent(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	seedConfig(t, base, ".claude/settings.json", `{
  "hooks": {
    "SessionStart": [
      {"matcher": "", "hooks": [{"type": "command", "command": "archcore hooks session-start"}]},
      {"matcher": "", "hooks": [{"type": "command", "command": "archcore hooks claude-code session-start"}]}
    ]
  }
}`)

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatalf("InstallClaudeCodeHooks: %v", err)
	}

	data, err := os.ReadFile(configPathFor(base, ".claude/settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := countArchcoreEntriesUnder(t, data, "SessionStart"); got != 1 {
		t.Errorf("want exactly 1 archcore entry under SessionStart after healing a duplicate, got %d:\n%s", got, data)
	}
	if !strings.Contains(string(data), "archcore hooks claude-code session-start") {
		t.Errorf("surviving entry must carry the current command:\n%s", data)
	}
}

// TestHooksInstall_MixedEntryNeverTouched: a hand-merged matcher entry mixing
// an archcore hook with a foreign hook is classified foreign — the installer
// appends its own entry and never rewrites the mixed one.
func TestHooksInstall_MixedEntryNeverTouched(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	mixed := `{"matcher": "", "hooks": [{"type": "command", "command": "archcore hooks session-start"}, {"type": "command", "command": "my-linter check"}]}`
	seedConfig(t, base, ".claude/settings.json", `{
  "hooks": {
    "SessionStart": [
      `+mixed+`
    ]
  }
}`)

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatalf("InstallClaudeCodeHooks: %v", err)
	}

	data, err := os.ReadFile(configPathFor(base, ".claude/settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "my-linter check") {
		t.Errorf("foreign hook inside mixed entry must survive:\n%s", data)
	}
	if !strings.Contains(string(data), `"archcore hooks session-start"`) {
		t.Errorf("mixed entry must not be rewritten (stale command should remain):\n%s", data)
	}
	if !strings.Contains(string(data), "archcore hooks claude-code session-start") {
		t.Errorf("current command must be appended as a new entry:\n%s", data)
	}
}

// TestHooksInstall_UpdatesStaleCursorEntry: same update-in-place contract for
// the flat {"command": …} Cursor shape.
func TestHooksInstall_UpdatesStaleCursorEntry(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	seedConfig(t, base, ".cursor/hooks.json", `{
  "version": 1,
  "hooks": {
    "sessionStart": [
      {"command": "archcore hooks session-start", "type": "command"},
      {"command": "other-tool run", "type": "command"}
    ]
  }
}`)

	if err := InstallCursorHooks(base); err != nil {
		t.Fatalf("InstallCursorHooks: %v", err)
	}

	data, err := os.ReadFile(configPathFor(base, ".cursor/hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := countArchcoreEntriesUnder(t, data, "sessionStart"); got != 1 {
		t.Errorf("want exactly 1 archcore entry under sessionStart after update, got %d:\n%s", got, data)
	}
	if !strings.Contains(string(data), "archcore hooks cursor session-start") {
		t.Errorf("stale cursor command must be updated in place:\n%s", data)
	}
	if !strings.Contains(string(data), "other-tool run") {
		t.Errorf("foreign cursor entry must survive:\n%s", data)
	}
}

// A user-wrapped archcore command (`sh -c 'archcore hooks …'`) is a deliberate
// customization: it contains the marker text but does not START with it, so it
// must classify foreign and survive an install untouched — the installer adds
// its own entry alongside instead of clobbering the wrapper.
func TestHooksInstall_UserWrappedCommandSurvives(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	wrapped := `sh -c 'archcore hooks cursor session-start 2>/dev/null'`
	seedConfig(t, base, ".cursor/hooks.json", `{
  "version": 1,
  "hooks": {
    "sessionStart": [
      {"command": "sh -c 'archcore hooks cursor session-start 2>/dev/null'", "type": "command"}
    ]
  }
}`)

	if err := InstallCursorHooks(base); err != nil {
		t.Fatalf("InstallCursorHooks: %v", err)
	}

	data, err := os.ReadFile(configPathFor(base, ".cursor/hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Compare decoded values, not raw bytes: encoding/json HTML-escapes '>'
	// on rewrite, which is cosmetic, not a semantic change.
	var doc struct {
		Hooks struct {
			SessionStart []struct {
				Command string `json:"command"`
			} `json:"sessionStart"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, data)
	}
	commands := make([]string, 0, len(doc.Hooks.SessionStart))
	for _, e := range doc.Hooks.SessionStart {
		commands = append(commands, e.Command)
	}
	if !slices.Contains(commands, wrapped) {
		t.Errorf("user-wrapped command must survive untouched, got %q", commands)
	}
}

// TestHooksInstall_UpdatesStaleCopilotEntry: same contract for the flat
// {"bash": …} Copilot shape.
func TestHooksInstall_UpdatesStaleCopilotEntry(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	seedConfig(t, base, ".github/hooks/archcore.json", `{
  "version": 1,
  "hooks": {
    "sessionStart": [
      {"type": "command", "bash": "archcore hooks session-start"}
    ]
  }
}`)

	if err := InstallCopilotHooks(base); err != nil {
		t.Fatalf("InstallCopilotHooks: %v", err)
	}

	data, err := os.ReadFile(configPathFor(base, ".github/hooks/archcore.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := countArchcoreEntriesUnder(t, data, "sessionStart"); got != 1 {
		t.Errorf("want exactly 1 archcore entry under sessionStart after update, got %d:\n%s", got, data)
	}
	if !strings.Contains(string(data), "archcore hooks copilot session-start") {
		t.Errorf("stale copilot command must be updated in place:\n%s", data)
	}
}

// countArchcoreEntriesUnder counts the archcore-owned entries under one event
// key. Scoped to an event on purpose: "how many entries do we own here" is the
// ownership contract, while "how many appear in the file" only counts how many
// events archcore installs — a number these tests have no opinion about.
func countArchcoreEntriesUnder(t *testing.T, data []byte, event string) int {
	t.Helper()
	var doc struct {
		Hooks map[string][]json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parsing config: %v\n%s", err, data)
	}
	n := 0
	for _, entry := range doc.Hooks[event] {
		if strings.Contains(string(entry), "archcore hooks ") {
			n++
		}
	}
	return n
}
