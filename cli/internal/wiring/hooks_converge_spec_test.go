package wiring

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Convergence of archcore-owned hook entries.
//
// The ownership probe decides whether an existing entry is already current. It
// used to compare only the command and the matcher, so every other field we
// write — Gemini's millisecond timeout, Copilot's timeoutSec — could never
// reach a config that already carried the right command: the probe reported
// "already installed" and the host kept the old budget forever.

// TestHooksInstall_TimeoutChangeConverges seeds an entry that is ours and
// current except for its timeout, then installs over it.
func TestHooksInstall_TimeoutChangeConverges(t *testing.T) {
	tests := []struct {
		name     string
		relPath  string
		install  func(string) error
		seed     string
		wantHave string
		wantGone string
	}{
		{
			name:    "gemini millisecond timeout",
			relPath: ".gemini/settings.json",
			install: InstallGeminiCLIHooks,
			seed: `{"hooks":{"BeforeTool":[{"matcher":"write_file","hooks":` +
				`[{"type":"command","command":"archcore hooks gemini-cli pre-tool-use","timeout":999}]}]}}`,
			wantHave: `"timeout":1000`,
			wantGone: `"timeout":999`,
		},
		{
			name:    "copilot second timeout",
			relPath: ".github/hooks/archcore.json",
			install: InstallCopilotHooks,
			seed: `{"version":1,"hooks":{"sessionStart":[{"type":"command",` +
				`"bash":"archcore hooks copilot session-start","timeoutSec":99}]}}`,
			wantHave: `"timeoutSec":3`,
			wantGone: `"timeoutSec":99`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := setupArchcoreDir(t)
			path := seedConfig(t, base, tt.relPath, tt.seed)

			if err := tt.install(base); err != nil {
				t.Fatal(err)
			}

			got := compactFile(t, path)
			if !strings.Contains(got, tt.wantHave) {
				t.Errorf("the current timeout did not reach the config; want %s in:\n%s", tt.wantHave, got)
			}
			if strings.Contains(got, tt.wantGone) {
				t.Errorf("the stale timeout survived; %s still present in:\n%s", tt.wantGone, got)
			}
		})
	}
}

// TestHooksInstall_MixedEntryNeverDuplicates guards the case a whole-entry
// comparison alone would break: a user has hand-merged their own hook into the
// same matcher entry that carries ours.
//
// The entry no longer equals what we would write, but it already carries our
// command — so it is current. Classifying it otherwise appends a second entry
// and the host runs the hook twice.
func TestHooksInstall_MixedEntryNeverDuplicates(t *testing.T) {
	base := setupArchcoreDir(t)
	seed := `{"hooks":{"PreToolUse":[{"matcher":"Write|Edit","hooks":[` +
		`{"type":"command","command":"archcore hooks claude-code pre-tool-use"},` +
		`{"type":"command","command":"my-own-linter"}]}]}}`
	path := seedConfig(t, base, ".claude/settings.json", seed)

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatal(err)
	}

	got := compactFile(t, path)
	if n := strings.Count(got, "archcore hooks claude-code pre-tool-use"); n != 1 {
		t.Errorf("archcore command appears %d times, want 1:\n%s", n, got)
	}
	if !strings.Contains(got, "my-own-linter") {
		t.Errorf("a hand-merged foreign hook was dropped:\n%s", got)
	}
}

// TestHooksInstall_ForeignEntrySurvives: an entry that is not ours at all must
// be left alone, however the probe is written.
func TestHooksInstall_ForeignEntrySurvives(t *testing.T) {
	base := setupArchcoreDir(t)
	seed := `{"hooks":{"PreToolUse":[{"matcher":"Write","hooks":[{"type":"command","command":"someone-elses-tool"}]}]}}`
	path := seedConfig(t, base, ".claude/settings.json", seed)

	if err := InstallClaudeCodeHooks(base); err != nil {
		t.Fatal(err)
	}

	got := compactFile(t, path)
	if !strings.Contains(got, "someone-elses-tool") {
		t.Errorf("a foreign entry was dropped:\n%s", got)
	}
	if !strings.Contains(got, "archcore hooks claude-code pre-tool-use") {
		t.Errorf("our entry was not added alongside it:\n%s", got)
	}
}

// compactFile reads a JSON file and strips its formatting, so assertions can
// look for a key/value pair without depending on indentation.
func compactFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var dst bytes.Buffer
	if err := json.Compact(&dst, data); err != nil {
		t.Fatalf("compact %s: %v", path, err)
	}
	return dst.String()
}
