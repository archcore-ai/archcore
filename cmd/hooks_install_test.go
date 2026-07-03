package cmd

import (
	"bytes"
	"os"
	"path/filepath"
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
	{"claude", runHooksInstall, ".claude/settings.json"},
	{"cursor", runCursorHooksInstall, ".cursor/hooks.json"},
	{"gemini", runGeminiCLIHooksInstall, ".gemini/settings.json"},
	{"copilot", runCopilotHooksInstall, ".github/hooks/archcore.json"},
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

	if err := runHooksInstall(base); err != nil {
		t.Fatalf("runHooksInstall: %v", err)
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

	if err := runHooksInstall(base); err == nil {
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
		{"cursor", runCursorHooksInstall, ".cursor/hooks.json"},
		{"copilot", runCopilotHooksInstall, ".github/hooks/archcore.json"},
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
	original := `{
    "zebra":    "first",
    "hooks": {"SessionStart": [{"matcher": "", "hooks": [{"type": "command", "command": "archcore hooks claude-code session-start"}]}]}
}
`
	path := seedConfig(t, base, ".claude/settings.json", original)

	if err := runHooksInstall(base); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Errorf("already-installed config must not be rewritten:\ngot:\n%s\nwant:\n%s", data, original)
	}
}
