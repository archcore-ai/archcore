package wiring

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedCodexConfig writes ~/.codex/config.toml against an isolated HOME.
func seedCodexConfig(t *testing.T, content string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	if content == "" {
		return
	}
	dir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestCodexHooksEnabled reads the one condition in this file that can lie.
//
// Reporting "hooks are written and will run" when they cannot is the failure
// this whole path exists to prevent, so a false positive here is worse than no
// check at all.
func TestCodexHooksEnabled(t *testing.T) {
	tests := []struct {
		name   string
		config string
		want   bool
	}{
		{name: "no config file at all", config: ""},
		{name: "current key under features", config: "[features]\nhooks = true\n", want: true},
		{name: "legacy key under features", config: "[features]\ncodex_hooks = true\n", want: true},
		{name: "dotted form", config: "features.hooks = true\n", want: true},
		{name: "explicitly disabled", config: "[features]\nhooks = false\n"},
		{name: "key present in an unrelated table", config: "[experimental]\nhooks = true\n"},
		{name: "a later table does not inherit features", config: "[features]\nother = 1\n\n[sandbox]\nhooks = true\n"},
		{name: "indented under features still counts", config: "[features]\n  hooks = true\n", want: true},
		{name: "unrelated config", config: "model = \"gpt-5\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seedCodexConfig(t, tt.config)
			if got := codexHooksEnabled(); got != tt.want {
				t.Errorf("codexHooksEnabled() = %v, want %v for:\n%s", got, tt.want, tt.config)
			}
		})
	}
}

// TestDetectInstalledPlugin: hosts nest their plugin checkouts, so a scan that
// stops at the first level finds nothing and the overlap notice never fires.
func TestDetectInstalledPlugin(t *testing.T) {
	tests := []struct {
		name      string
		layout    string
		wantFound bool
	}{
		{name: "flat layout", layout: ".claude/plugins/archcore", wantFound: true},
		{name: "nested cache layout", layout: ".claude/plugins/cache/archcore-plugins/archcore/0.6.2", wantFound: true},
		{name: "cursor layout", layout: ".cursor/plugins/cache/archcore", wantFound: true},
		{name: "unrelated plugin", layout: ".claude/plugins/cache/other-plugins/other/1.0.0"},
		{name: "nothing installed", layout: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			if tt.layout != "" {
				if err := os.MkdirAll(filepath.Join(home, filepath.FromSlash(tt.layout)), 0o755); err != nil {
					t.Fatal(err)
				}
			}

			_, found := detectInstalledPlugin()

			if found != tt.wantFound {
				t.Errorf("detectInstalledPlugin() found = %v, want %v for layout %q", found, tt.wantFound, tt.layout)
			}
		})
	}
}

// TestDescribeSelfCausedPluginConflictDropsTheUpdateAdvice covers §15 of
// plugin-delivery.spec. After this CLI installed or updated the plugin itself,
// "until it is updated" is false — the plugin IS current, and the overlap that
// is left belongs to the host session that loaded the old hooks. The two
// wordings are separate functions because requirement 4 of
// plugin-cli-compatibility.rule binds the original for `doctor` and
// `hooks install`, whose callers detect a plugin they did not touch.
func TestDescribeSelfCausedPluginConflictDropsTheUpdateAdvice(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.MkdirAll(filepath.Join(home, ".claude", "plugins", "archcore"), 0o755); err != nil {
		t.Fatal(err)
	}

	note := DescribeSelfCausedPluginConflict()
	if note == "" {
		t.Fatal("no notice for a detected plugin")
	}
	if strings.Contains(note, "Updating the plugin") || strings.Contains(note, "until it is updated") {
		t.Errorf("the self-caused notice tells the user to update the plugin this run just updated:\n%s", note)
	}
	// Compared slash-normalized: DescribeSelfCausedPluginConflict builds the
	// path with filepath.Join, so a literal forward-slash expectation fails on
	// windows. The rest of this file already sets USERPROFILE for that platform.
	if !strings.Contains(filepath.ToSlash(note), ".claude/plugins/archcore") {
		t.Errorf("the notice names no install path:\n%s", note)
	}
	if note == DescribePluginConflict() {
		t.Error("the self-caused notice repeats the wording requirement 4 binds for doctor and hooks install")
	}
}

// TestPluginConflictNoticesAreEmptyWithoutADetection covers requirement 5 of
// plugin-cli-compatibility.rule for both wordings: a detection that finds
// nothing produces no notice, and never a sentence about a plugin that is not
// there.
func TestPluginConflictNoticesAreEmptyWithoutADetection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if note := DescribePluginConflict(); note != "" {
		t.Errorf("DescribePluginConflict() = %q, want no notice", note)
	}
	if note := DescribeSelfCausedPluginConflict(); note != "" {
		t.Errorf("DescribeSelfCausedPluginConflict() = %q, want no notice", note)
	}
}

// TestDetectInstalledPlugin_StopsAtTheDepthCap keeps the deeper scan from
// becoming an unbounded walk of the user's home directory.
func TestDetectInstalledPlugin_StopsAtTheDepthCap(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	deep := filepath.Join(home, ".claude", "plugins", "a", "b", "c", "d", "e", "archcore")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, found := detectInstalledPlugin(); found {
		t.Error("detection reached past its depth cap")
	}
}
