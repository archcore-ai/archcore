package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeSettings writes a Claude Code settings file and returns its path.
func writeSettings(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing the settings fixture: %v", err)
	}
	return path
}

// readSettings decodes a settings file for assertions on its content.
func readSettings(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the settings file: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("the settings file is not valid JSON after the merge: %v\n%s", err, data)
	}
	return decoded
}

// marketplaceEntry returns the entry this surface owns, or nil when it is
// absent.
func marketplaceEntry(t *testing.T, path string) map[string]any {
	t.Helper()
	section, _ := readSettings(t, path)[extraKnownMarketplacesKey].(map[string]any)
	entry, _ := section[MarketplaceID].(map[string]any)
	return entry
}

// ageFile backdates a file's modification time, so a later write is visible
// without waiting for the clock. It is how "no write happened" is proved
// without inspecting content that a rewrite would reproduce byte for byte.
func ageFile(t *testing.T, path string) time.Time {
	t.Helper()
	aged := time.Now().Add(-time.Hour)
	if err := os.Chtimes(path, aged, aged); err != nil {
		t.Fatalf("backdating the settings file: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stating the settings file: %v", err)
	}
	return info.ModTime()
}

func modTime(t *testing.T, path string) time.Time {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stating the settings file: %v", err)
	}
	return info.ModTime()
}

// TestEnsureClaudeAutoUpdateWritesTheEntry covers requirement 14 of
// plugin-delivery.spec on a machine with no settings file at all.
func TestEnsureClaudeAutoUpdateWritesTheEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")

	if _, err := EnsureClaudeAutoUpdate(path); err != nil {
		t.Fatalf("EnsureClaudeAutoUpdate: %v", err)
	}
	entry := marketplaceEntry(t, path)
	if entry[marketplaceAutoUpdateKey] != true {
		t.Errorf("entry = %+v, want autoUpdate enabled", entry)
	}
	source, _ := entry[marketplaceSourceKey].(map[string]any)
	if source["repo"] != RepoID {
		t.Errorf("source = %+v, want the frozen repository %q", source, RepoID)
	}
}

// TestClaudeSettingsWireFormat pins every key of the entry as a literal.
//
// The constants above name them, and a test written against the constants
// passes through a rename — but the reader of this file is Claude Code, not this
// process. A renamed key ships a CLI that writes a declaration the host ignores,
// so autoUpdate silently stops happening and nothing in this repository can see
// it. The keys and the marketplace id come from plugin-delivery.spec, Surface,
// and requirement 11 of plugin-cli-compatibility.rule freezes the id.
//
// [assumption] the source object's own spelling is unverified, the same way the
// spec marks it; it is pinned here so a change to it is a deliberate edit.
func TestClaudeSettingsWireFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if _, err := EnsureClaudeAutoUpdate(path); err != nil {
		t.Fatalf("EnsureClaudeAutoUpdate: %v", err)
	}

	section, ok := readSettings(t, path)["extraKnownMarketplaces"].(map[string]any)
	if !ok {
		t.Fatalf("the file carries no extraKnownMarketplaces object:\n%+v", readSettings(t, path))
	}
	entry, ok := section["archcore-plugins"].(map[string]any)
	if !ok {
		t.Fatalf("the section carries no archcore-plugins entry:\n%+v", section)
	}
	if entry["autoUpdate"] != true {
		t.Errorf("entry = %+v, want the autoUpdate key spelled exactly", entry)
	}
	source, ok := entry["source"].(map[string]any)
	if !ok {
		t.Fatalf("entry = %+v, want a source object", entry)
	}
	if source["source"] != "github" || source["repo"] != "archcore-ai/plugin" {
		t.Errorf("source = %+v, want the github source of archcore-ai/plugin", source)
	}
}

// TestEnsureClaudeAutoUpdatePreservesForeignFields is the property that makes
// this safe to run on a file archcore does not own: every key the user or the
// host put there survives, at every level.
func TestEnsureClaudeAutoUpdatePreservesForeignFields(t *testing.T) {
	path := writeSettings(t, `{
  "model": "opus",
  "permissions": {"allow": ["Bash(ls:*)"]},
  "extraKnownMarketplaces": {
    "someone-else": {"source": {"source": "github", "repo": "someone/else"}},
    "archcore-plugins": {"source": {"source": "github", "repo": "archcore-ai/plugin"}, "pinned": "0.4.0"}
  }
}`)

	if _, err := EnsureClaudeAutoUpdate(path); err != nil {
		t.Fatalf("EnsureClaudeAutoUpdate: %v", err)
	}

	settings := readSettings(t, path)
	if settings["model"] != "opus" {
		t.Errorf("model = %v, want the value the file already carried", settings["model"])
	}
	if _, ok := settings["permissions"]; !ok {
		t.Error("the permissions key was dropped")
	}
	section, _ := settings[extraKnownMarketplacesKey].(map[string]any)
	if _, ok := section["someone-else"]; !ok {
		t.Error("another marketplace entry was dropped")
	}
	entry := marketplaceEntry(t, path)
	if entry["pinned"] != "0.4.0" {
		t.Errorf("entry = %+v, want the unknown field it already carried", entry)
	}
	if entry[marketplaceAutoUpdateKey] != true {
		t.Errorf("entry = %+v, want autoUpdate enabled", entry)
	}
}

// TestEnsureClaudeAutoUpdateCreatesTheSettingsDirectory covers the case
// `--scope project` hits first: a project with no .claude/ in it yet. SaveDoc
// writes through a temp file created beside its target, so a missing parent is
// not a missing file — it is a failed install step on a machine where the
// plugin itself installed fine.
func TestEnsureClaudeAutoUpdateCreatesTheSettingsDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude", "settings.json")

	if _, err := EnsureClaudeAutoUpdate(path); err != nil {
		t.Fatalf("EnsureClaudeAutoUpdate: %v", err)
	}
	if entry := marketplaceEntry(t, path); entry[marketplaceAutoUpdateKey] != true {
		t.Errorf("entry = %+v, want autoUpdate enabled", entry)
	}
}

// TestEnsureClaudeAutoUpdateKeepsAUserEditedEntry is the half of "preserving
// unknown fields" (plugin-delivery.spec §14) that a merge does not give for
// free. MergeEntryFields keeps the fields the desired entry does not mention and
// overwrites every field it does, so an entry carrying the full desired shape
// rewrites a source the user pointed at a fork or a local checkout — silently,
// on the next install. Requirement 14 names one key; that is the only one this
// surface may ensure on an entry that is already there.
func TestEnsureClaudeAutoUpdateKeepsAUserEditedEntry(t *testing.T) {
	path := writeSettings(t, `{
  "extraKnownMarketplaces": {
    "archcore-plugins": {"source": {"source": "github", "repo": "someone/fork"}, "autoUpdate": false}
  }
}`)

	if _, err := EnsureClaudeAutoUpdate(path); err != nil {
		t.Fatalf("EnsureClaudeAutoUpdate: %v", err)
	}

	entry := marketplaceEntry(t, path)
	if entry[marketplaceAutoUpdateKey] != true {
		t.Errorf("entry = %+v, want autoUpdate enabled", entry)
	}
	source, _ := entry[marketplaceSourceKey].(map[string]any)
	if source["repo"] != "someone/fork" {
		t.Errorf("source = %+v, want the repository the user set", source)
	}
}

// TestEnsureClaudeAutoUpdateWritesNothingWhenTheEntryIsThere is what makes a
// repeated install a genuine no-op. A rerun of `archcore init` must not touch
// the user's settings file at all, and identical content is not proof of that:
// a rewrite reproduces it byte for byte.
func TestEnsureClaudeAutoUpdateWritesNothingWhenTheEntryIsThere(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if _, err := EnsureClaudeAutoUpdate(path); err != nil {
		t.Fatalf("the first merge failed: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the settings file: %v", err)
	}
	aged := ageFile(t, path)

	if _, err := EnsureClaudeAutoUpdate(path); err != nil {
		t.Fatalf("the second merge failed: %v", err)
	}

	if got := modTime(t, path); !got.Equal(aged) {
		t.Errorf("the settings file was rewritten (mtime %s, want %s)", got, aged)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the settings file: %v", err)
	}
	if string(after) != string(before) {
		t.Errorf("the settings file changed:\n%s\n%s", before, after)
	}
}

// TestEnsureClaudeAutoUpdateBacksUpInvalidJSON covers Failure Behavior 4 of
// plugin-delivery.spec through backup-invalid-configs.adr: the unreadable file
// is kept before anything replaces it, and the caller is told so.
//
// The report is the half that was missing. The user's live settings now hold
// only what this surface wrote; a backup nothing announces leaves them looking
// for a file they were never told about.
func TestEnsureClaudeAutoUpdateBacksUpInvalidJSON(t *testing.T) {
	const broken = `{ "model": "opus", `
	path := writeSettings(t, broken)

	backedUp, err := EnsureClaudeAutoUpdate(path)
	if err != nil {
		t.Fatalf("EnsureClaudeAutoUpdate: %v", err)
	}
	if !backedUp {
		t.Error("backedUp = false after replacing an unparsable settings file")
	}

	backup, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("reading the backup: %v", err)
	}
	if string(backup) != broken {
		t.Errorf("the backup holds %q, want the original bytes %q", backup, broken)
	}
	if entry := marketplaceEntry(t, path); entry[marketplaceAutoUpdateKey] != true {
		t.Errorf("entry = %+v, want autoUpdate enabled in the replacement file", entry)
	}
}

// TestEnsureClaudeAutoUpdateReportsNoBackupForAReadableFile is the other side of
// the same signal: a caller that printed the notice unconditionally would tell
// every user their settings had been replaced.
func TestEnsureClaudeAutoUpdateReportsNoBackupForAReadableFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "valid file", content: `{"model": "opus"}`},
		{name: "empty object", content: `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeSettings(t, tt.content)

			backedUp, err := EnsureClaudeAutoUpdate(path)
			if err != nil {
				t.Fatalf("EnsureClaudeAutoUpdate: %v", err)
			}
			if backedUp {
				t.Error("backedUp = true for a file that parsed")
			}
			if _, err := os.Stat(path + ".bak"); err == nil {
				t.Error("a readable settings file must not be backed up")
			}
		})
	}
}

// TestRemoveClaudeAutoUpdateDeletesOnlyOurEntry covers requirement 17: the
// removal undoes what this surface wrote and nothing else.
func TestRemoveClaudeAutoUpdateDeletesOnlyOurEntry(t *testing.T) {
	path := writeSettings(t, `{
  "model": "opus",
  "extraKnownMarketplaces": {
    "someone-else": {"source": {"source": "github", "repo": "someone/else"}},
    "archcore-plugins": {"source": {"source": "github", "repo": "archcore-ai/plugin"}, "autoUpdate": true}
  }
}`)

	if err := RemoveClaudeAutoUpdate(path); err != nil {
		t.Fatalf("RemoveClaudeAutoUpdate: %v", err)
	}

	settings := readSettings(t, path)
	if settings["model"] != "opus" {
		t.Errorf("model = %v, want the value the file already carried", settings["model"])
	}
	section, _ := settings[extraKnownMarketplacesKey].(map[string]any)
	if _, ok := section["someone-else"]; !ok {
		t.Error("another marketplace entry was deleted")
	}
	if _, ok := section[MarketplaceID]; ok {
		t.Errorf("the entry this surface wrote survived the removal: %+v", section)
	}
	if data, err := os.ReadFile(path); err == nil && strings.Contains(string(data), `"autoUpdate"`) {
		t.Errorf("the settings file still carries an autoUpdate key:\n%s", data)
	}
}

// TestRemoveClaudeAutoUpdateWritesNothingWhenTheEntryIsAbsent keeps a removal
// on a machine that never had the entry from touching the user's file.
func TestRemoveClaudeAutoUpdateWritesNothingWhenTheEntryIsAbsent(t *testing.T) {
	path := writeSettings(t, `{"model":"opus","extraKnownMarketplaces":{"someone-else":{}}}`)
	aged := ageFile(t, path)

	if err := RemoveClaudeAutoUpdate(path); err != nil {
		t.Fatalf("RemoveClaudeAutoUpdate: %v", err)
	}
	if got := modTime(t, path); !got.Equal(aged) {
		t.Errorf("the settings file was rewritten (mtime %s, want %s)", got, aged)
	}
}

// TestRemoveClaudeAutoUpdateOnAMissingFile keeps the removal from creating a
// settings file for a host that has none.
func TestRemoveClaudeAutoUpdateOnAMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")

	if err := RemoveClaudeAutoUpdate(path); err != nil {
		t.Fatalf("RemoveClaudeAutoUpdate: %v", err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("the removal created a settings file that did not exist")
	}
}

// TestRemoveClaudeAutoUpdateRefusesAnInvalidFile keeps the corrupt-file policy
// on the write path, where it belongs. Failure Behavior 4 of
// plugin-delivery.spec backs a file up "before writing"; a removal that finds
// nothing to remove writes nothing, so backing the file up would leave a .bak
// of a file this call never touched and report success for an operation that
// could not read what it was asked to change.
func TestRemoveClaudeAutoUpdateRefusesAnInvalidFile(t *testing.T) {
	const broken = `{ "extraKnownMarketplaces": `
	path := writeSettings(t, broken)

	if err := RemoveClaudeAutoUpdate(path); err == nil {
		t.Error("a settings file that is not valid JSON was accepted")
	}
	if _, err := os.Stat(path + ".bak"); err == nil {
		t.Error("the removal backed up a file it had nothing to remove from")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the settings file: %v", err)
	}
	if string(data) != broken {
		t.Errorf("the settings file reads\n%s\nwant the bytes it already carried\n%s", data, broken)
	}
}

// TestClaudeSettingsRoundTrip proves the pair converges: an install followed by
// a removal leaves the file as it was, so a user who tries the plugin and drops
// it keeps the settings they started with.
func TestClaudeSettingsRoundTrip(t *testing.T) {
	const original = `{
  "model": "opus",
  "extraKnownMarketplaces": {
    "someone-else": {
      "source": {
        "source": "github",
        "repo": "someone/else"
      }
    }
  }
}
`
	path := writeSettings(t, original)

	if _, err := EnsureClaudeAutoUpdate(path); err != nil {
		t.Fatalf("EnsureClaudeAutoUpdate: %v", err)
	}
	if err := RemoveClaudeAutoUpdate(path); err != nil {
		t.Fatalf("RemoveClaudeAutoUpdate: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the settings file: %v", err)
	}
	if string(after) != original {
		t.Errorf("the settings file reads\n%s\nwant\n%s", after, original)
	}
}

// TestEnsureClaudeAutoUpdateRejectsANonObjectSection keeps the surface from
// overwriting a key it cannot read. Failure Behavior 3 asks for a report, and a
// report is what an error is; silently replacing the value would lose whatever
// the user meant by it.
func TestEnsureClaudeAutoUpdateRejectsANonObjectSection(t *testing.T) {
	path := writeSettings(t, `{"extraKnownMarketplaces": "somewhere"}`)

	if _, err := EnsureClaudeAutoUpdate(path); err == nil {
		t.Error("a marketplaces key that is not an object was accepted")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the settings file: %v", err)
	}
	if !strings.Contains(string(data), `"somewhere"`) {
		t.Errorf("the value was overwritten:\n%s", data)
	}
}
