package wiring

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// hookInstallers enumerates the four generic-installer consumers for
// cross-agent table tests.
var hookInstallers = []struct {
	name    string
	install func(string) error
	relPath string
}{
	{"claude", InstallClaudeCodeHooks, ".claude/settings.json"},
	{"cursor", InstallCursorHooks, ".cursor/hooks.json"},
	{"gemini", InstallGeminiCLIHooks, ".gemini/settings.json"},
	{"copilot", InstallCopilotHooks, ".github/hooks/archcore.json"},
	{"codex", InstallCodexCLIHooks, ".codex/hooks.json"},
}

func configPathFor(base, relPath string) string {
	return filepath.Join(base, filepath.FromSlash(relPath))
}

func seedConfig(t *testing.T, base, relPath, content string) string {
	t.Helper()
	path := configPathFor(base, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestHooksInstall_PreservesUnknownHookFields is the regression pin for the
// field-stripping defect: foreign hook entries (other events, unknown fields
// like "timeout") must survive an archcore install byte-for-content.
func TestHooksInstall_PreservesUnknownHookFields(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	seedConfig(t, base, ".claude/settings.json", `{
  "permissions": {"allow": ["Bash(go test:*)"]},
  "hooks": {
    "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "lint.sh", "timeout": 120}], "custom": true}]
  }
}`)

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatalf("InstallClaudeCodeHooks: %v", err)
	}

	data, err := os.ReadFile(configPathFor(base, ".claude/settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		`"timeout": 120`,
		`"custom": true`,
		`"permissions"`,
		`"lint.sh"`,
		`"SessionStart"`,
		`"archcore hooks claude-code session-start"`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("settings.json lost %s after install:\n%s", want, text)
		}
	}
}

// TestHooksInstall_PreservesUnknownFieldsOnArchcoreEntries covers the half the
// test above does not: unknown fields on archcore's OWN entries.
//
// An entry is classified stale whenever it differs from what we would write, so
// a user-set "timeout" or a key from a newer archcore made every entry stale —
// and the update replaced it wholesale, deleting both, silently, on `init`,
// `hooks install`, and `doctor --fix`. Two rows, because they take different
// paths through installHookEvents: an entry that is otherwise current must not
// be rewritten at all, and one carrying a stale command must be merged, not
// replaced.
func TestHooksInstall_PreservesUnknownFieldsOnArchcoreEntries(t *testing.T) {
	t.Parallel()
	for _, tt := range hookInstallers {
		for _, stale := range []bool{false, true} {
			name := tt.name + "/current-command"
			if stale {
				name = tt.name + "/stale-command"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				base := setupArchcoreDir(t)
				if err := tt.install(base); err != nil {
					t.Fatalf("first install: %v", err)
				}
				path := configPathFor(base, tt.relPath)
				injectHookExtras(t, path, stale)
				before, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}

				if err := tt.install(base); err != nil {
					t.Fatalf("second install: %v", err)
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				text := string(data)

				for _, want := range []string{`"futureKey"`, `"futureInnerKey"`} {
					if !strings.Contains(text, want) {
						t.Errorf("install dropped %s from an archcore-owned entry:\n%s", want, text)
					}
				}
				if !strings.Contains(text, archcoreHookMarker) {
					t.Errorf("archcore hook command is gone after install:\n%s", text)
				}
				if strings.Contains(text, legacyCommandSuffix) {
					t.Errorf("a stale archcore command survived the merge:\n%s", text)
				}
				if !stale && !bytes.Equal(before, data) {
					t.Errorf("an entry differing only by unknown fields was rewritten:\nbefore:\n%s\nafter:\n%s", before, data)
				}
			})
		}
	}
}

// TestHooksInstall_PreservesUserTimeoutOnArchcoreEntry is the concrete form of
// the test above: "timeout" is a documented Claude Code hook field that archcore
// does not write for that host, so a user who raised it on our SessionStart hook
// is holding real configuration in an entry we own. It has to survive an update
// that changes our command.
func TestHooksInstall_PreservesUserTimeoutOnArchcoreEntry(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	seedConfig(t, base, ".claude/settings.json", `{
  "hooks": {
    "SessionStart": [{"hooks": [{"type": "command", "command": "archcore hooks claude-code session-start --legacy", "timeout": 60}]}]
  }
}`)

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatalf("InstallClaudeCodeHooks: %v", err)
	}

	data, err := os.ReadFile(configPathFor(base, ".claude/settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `"timeout": 60`) {
		t.Errorf("a user-set timeout was deleted by the update:\n%s", text)
	}
	if !strings.Contains(text, `"command": "archcore hooks claude-code session-start"`) {
		t.Errorf("the stale command was not updated:\n%s", text)
	}
	if strings.Contains(text, legacyCommandSuffix) {
		t.Errorf("the stale command survived:\n%s", text)
	}
	// The merge must not leave a second entry behind.
	if n := strings.Count(text, "archcore hooks claude-code session-start"); n != 1 {
		t.Errorf("SessionStart command appears %d times, want 1:\n%s", n, text)
	}
}

// legacyCommandSuffix makes an installed archcore command look like one written
// by an older CLI: still marker-prefixed, so it classifies as stale-archcore.
const legacyCommandSuffix = " --legacy"

var archcoreCommandRe = regexp.MustCompile(`"(archcore hooks [^"]*)"`)

// injectHookExtras edits the config at path the way a user or a newer archcore
// would: an unknown field beside archcore's own entry fields, and a timeout on
// the hook itself. With stale set, the installed command is also aged so the
// entry takes the update path rather than the already-installed one.
func injectHookExtras(t *testing.T, path string, stale bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		data = archcoreCommandRe.ReplaceAll(data, []byte(`"$1`+legacyCommandSuffix+`"`))
	}

	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatal(err)
	}
	hooks, ok := root["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("no hooks section in %s:\n%s", path, data)
	}
	touched := 0
	for _, v := range hooks {
		entries, _ := v.([]any)
		for _, e := range entries {
			entry, ok := e.(map[string]any)
			if !ok {
				continue
			}
			raw, err := json.Marshal(entry)
			if err != nil || !strings.Contains(string(raw), archcoreHookMarker) {
				continue
			}
			touched++
			entry["futureKey"] = "written-by-a-newer-archcore"
			// Names archcore writes for no host: Gemini owns "timeout" on its
			// inner hooks, so using that here would assert against a field the
			// merge is supposed to overwrite.
			inner, nested := entry["hooks"].([]any)
			if !nested {
				entry["futureInnerKey"] = "kept"
				continue
			}
			for _, h := range inner {
				if hm, ok := h.(map[string]any); ok {
					hm["futureInnerKey"] = "kept"
				}
			}
		}
	}
	if touched == 0 {
		t.Fatalf("fixture found no archcore entries in %s:\n%s", path, data)
	}

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestHooksInstall_NullHooksSection pins the `"hooks": null` handling for all
// four installers (previously a panic for Gemini, an error for Claude).
func TestHooksInstall_NullHooksSection(t *testing.T) {
	t.Parallel()
	for _, tt := range hookInstallers {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)
			seedConfig(t, base, tt.relPath, `{"hooks": null}`)

			if err := tt.install(base); err != nil {
				t.Fatalf("install with null hooks section: %v", err)
			}
			data, err := os.ReadFile(configPathFor(base, tt.relPath))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.ToLower(string(data)), "sessionstart") {
				t.Errorf("hook not installed over a null hooks section:\n%s", data)
			}
		})
	}
}

// TestHooksInstall_SecondRunByteIdentical pins idempotency: an already-installed
// config must not be rewritten at all (byte-identical file, no tmp leftovers).
func TestHooksInstall_SecondRunByteIdentical(t *testing.T) {
	t.Parallel()
	for _, tt := range hookInstallers {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)

			if err := tt.install(base); err != nil {
				t.Fatalf("first install: %v", err)
			}
			path := configPathFor(base, tt.relPath)
			first, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			firstInfo, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}

			if err := tt.install(base); err != nil {
				t.Fatalf("second install: %v", err)
			}
			second, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(first, second) {
				t.Errorf("second run changed the file:\nfirst:\n%s\nsecond:\n%s", first, second)
			}
			secondInfo, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if !secondInfo.ModTime().Equal(firstInfo.ModTime()) {
				t.Error("second run rewrote an unchanged file (mtime moved)")
			}
			if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
				t.Error("tmp file left behind")
			}
		})
	}
}

// TestHooksInstall_BackupWriteFailureAborts pins the abort-on-backup-failure
// policy: if the .bak of a corrupted config cannot be written, the install
// must fail and the original (possibly hand-recoverable) file stay untouched.
func TestHooksInstall_BackupWriteFailureAborts(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	original := `{corrupted`
	path := seedConfig(t, base, ".claude/settings.json", original)
	// A directory at path+".bak" makes the backup write fail.
	if err := os.MkdirAll(path+".bak", 0o755); err != nil {
		t.Fatal(err)
	}

	if err := InstallClaudeCodeHooks(base); err == nil {
		t.Fatal("expected error when the backup cannot be written")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Error("original corrupted file must stay untouched when backup fails")
	}
}

// TestHooksInstall_PreservesUnknownTopLevelKeysAndVersion pins that cursor and
// copilot configs keep foreign top-level keys and an existing version value
// (previously "version": 0/2 was rewritten to 1 via typed structs).
func TestHooksInstall_PreservesUnknownTopLevelKeysAndVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		install func(string) error
		relPath string
	}{
		{"cursor", InstallCursorHooks, ".cursor/hooks.json"},
		{"copilot", InstallCopilotHooks, ".github/hooks/archcore.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)
			seedConfig(t, base, tt.relPath, `{"version": 2, "customKey": {"a": 1}, "hooks": {}}`)

			if err := tt.install(base); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(configPathFor(base, tt.relPath))
			if err != nil {
				t.Fatal(err)
			}
			text := string(data)
			if !strings.Contains(text, `"version": 2`) {
				t.Errorf("existing version must be preserved:\n%s", text)
			}
			if !strings.Contains(text, `"customKey"`) {
				t.Errorf("unknown top-level key must be preserved:\n%s", text)
			}
		})
	}
}

// TestHooksInstall_SkipsWriteWhenInstalled pins that "already installed" means
// no write at all: even a hand-formatted file stays byte-identical.
func TestHooksInstall_SkipsWriteWhenInstalled(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	path := seedConfig(t, base, ".claude/settings.json", `{"zebra": "first"}`+"\n")

	// Install once to reach the fully-wired state, then again. The fixture is
	// derived rather than written out: pinning a literal config here would make
	// this test fail every time an event is added, which is exactly the change
	// it is least interested in.
	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatal(err)
	}
	installed, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(installed) {
		t.Errorf("already-installed config must not be rewritten:\ngot:\n%s\nwant:\n%s", again, installed)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(info.ModTime()) {
		t.Error("already-installed config was rewritten (mtime changed)")
	}
	if !strings.Contains(string(again), `"zebra"`) {
		t.Errorf("a foreign top-level key was dropped:\n%s", again)
	}
}
