package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// writeRegistryEntry creates a home-relative directory, the way a host lays out
// an installed plugin.
func writeRegistryEntry(t *testing.T, home, relPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(home, filepath.FromSlash(relPath)), 0o755); err != nil {
		t.Fatalf("writing the registry entry %s: %v", relPath, err)
	}
}

// writeRegistryFile creates a home-relative file, the way a host stores a
// marketplace snapshot.
func writeRegistryFile(t *testing.T, home, relPath string) {
	t.Helper()
	full := filepath.Join(home, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("writing the registry file %s: %v", relPath, err)
	}
	if err := os.WriteFile(full, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("writing the registry file %s: %v", relPath, err)
	}
}

// TestRegistryListsPluginPerHost covers every host's registry path, because the
// path is the whole evidence when the host CLI is not on PATH — and three of
// the four paths are marked unverified in the spec, so the layout each one
// answers is worth pinning.
// Every layout below was read off a real machine on 2026-08-16, not derived
// from the spec: three hosts check the plugin out as cache/<marketplace>/
// <plugin>, and Copilot flattens the install spec into one directory name.
func TestRegistryListsPluginPerHost(t *testing.T) {
	tests := []struct {
		name  string
		host  Host
		entry string
	}{
		{name: "claude-code", host: HostClaudeCode, entry: ".claude/plugins/cache/archcore-plugins/archcore"},
		{name: "cursor", host: HostCursor, entry: ".cursor/plugins/cache/archcore-plugins/archcore"},
		{name: "codex-cli", host: HostCodexCLI, entry: ".codex/plugins/cache/archcore-plugins/archcore"},
		{name: "copilot", host: HostCopilot, entry: ".copilot/installed-plugins/_direct/archcore-ai--plugin--plugins-archcore"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := isolateHome(t)
			spec, ok := SpecFor(tt.host)
			if !ok {
				t.Fatalf("no host table row for %q", tt.host)
			}
			if registryListsPlugin(spec) {
				t.Error("an empty home reported the plugin installed")
			}
			writeRegistryEntry(t, home, tt.entry)
			if !registryListsPlugin(spec) {
				t.Errorf("the registry entry %s was not found", tt.entry)
			}
		})
	}
}

// TestRegistryFindsAPluginSnapshotFile keeps files in scope. The update spec
// names snapshots without naming a file, so the scan has to answer the same for
// a directory called archcore and a snapshot stored as archcore.json.
func TestRegistryFindsAPluginSnapshotFile(t *testing.T) {
	home := isolateHome(t)
	spec, _ := SpecFor(HostCodexCLI)
	writeRegistryFile(t, home, ".codex/plugins/cache/archcore-plugins/archcore.json")

	if !registryListsPlugin(spec) {
		t.Error("a plugin snapshot file was not found")
	}
}

// TestRegistryIgnoresAMarketplaceWithNothingInstalled is the false positive that
// made this scan useless. Every host stores the marketplace checkout one level
// above the plugin, so a scan that matched the marketplace id returned at the
// marketplace level and never read the plugin level at all — reporting the
// plugin installed on a machine that only ever ran `marketplace add`, which
// install then treats as a no-op and never installs.
//
// All three arrangements below exist on a real machine that has run
// `marketplace add` and nothing else.
func TestRegistryIgnoresAMarketplaceWithNothingInstalled(t *testing.T) {
	tests := []struct {
		name  string
		host  Host
		entry string
	}{
		{name: "claude cache", host: HostClaudeCode, entry: ".claude/plugins/cache/archcore-plugins"},
		{name: "claude marketplaces", host: HostClaudeCode, entry: ".claude/plugins/marketplaces/archcore-plugins"},
		{name: "codex tmp snapshot", host: HostCodexCLI, entry: ".codex/plugins/.tmp/marketplaces/archcore-plugins"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := isolateHome(t)
			spec, _ := SpecFor(tt.host)
			writeRegistryEntry(t, home, tt.entry)

			if registryListsPlugin(spec) {
				t.Errorf("a registered marketplace at %s was read as an installed plugin", tt.entry)
			}
		})
	}
}

// TestRegistryIgnoresACodexSiblingDirectory pins the registry root. ~/.codex
// holds sessions/, cache/, log/ and .tmp/ beside plugins/, so rooting the scan
// there would spend the entry budget on unrelated subtrees and match the
// marketplace snapshot under .tmp/marketplaces on the way.
func TestRegistryIgnoresACodexSiblingDirectory(t *testing.T) {
	home := isolateHome(t)
	spec, _ := SpecFor(HostCodexCLI)
	writeRegistryEntry(t, home, ".codex/.tmp/marketplaces/archcore-plugins/archcore")

	if registryListsPlugin(spec) {
		t.Error("a snapshot outside ~/.codex/plugins was read as an installed plugin")
	}
}

// TestRegistryIgnoresAProjectDirectory is the false positive this scan is one
// substring away from. Codex keys entries by project directory, so a checkout
// of this repository sits under ~/.codex — and reading that as an installed
// plugin would print an update command on a machine that never installed one.
func TestRegistryIgnoresAProjectDirectory(t *testing.T) {
	home := isolateHome(t)
	spec, _ := SpecFor(HostCodexCLI)
	writeRegistryEntry(t, home, ".codex/plugins/projects/-Users-someone-Documents-archcore-cli")
	writeRegistryEntry(t, home, ".codex/plugins/sessions/archcore-notes")

	if registryListsPlugin(spec) {
		t.Error("a project directory named after this repository was read as an installed plugin")
	}
}

// TestRegistryStopsAtTheDepthBound proves the walk is bounded rather than a
// full-tree search. Three levels covers the nesting the hosts use; anything
// deeper costs a miss, which is the safe direction — the miss keeps a host
// silent, it never invents an install.
func TestRegistryStopsAtTheDepthBound(t *testing.T) {
	home := isolateHome(t)
	spec, _ := SpecFor(HostClaudeCode)

	writeRegistryEntry(t, home, ".claude/plugins/a/b/c/archcore")
	if registryListsPlugin(spec) {
		t.Error("the scan descended past the depth bound")
	}
	writeRegistryEntry(t, home, ".claude/plugins/a/b/archcore")
	if !registryListsPlugin(spec) {
		t.Error("the scan missed an entry inside the depth bound")
	}
}

// TestRegistryStopsAtTheEntryBudget proves the budget holds and that the cut is
// deterministic: os.ReadDir sorts by name, so the same tree always spends the
// budget on the same entries and always answers the same way.
func TestRegistryStopsAtTheEntryBudget(t *testing.T) {
	home := isolateHome(t)
	spec, _ := SpecFor(HostClaudeCode)
	for i := 0; i < registryScanBudget; i++ {
		writeRegistryFile(t, home, fmt.Sprintf(".claude/plugins/aaa-%04d", i))
	}
	writeRegistryFile(t, home, ".claude/plugins/archcore")

	if registryListsPlugin(spec) {
		t.Error("the scan spent more than its entry budget")
	}
	for i := 0; i < 20; i++ {
		if registryListsPlugin(spec) {
			t.Fatalf("scan %d answered differently from the first, so the cut is not deterministic", i)
		}
	}
}

// TestRegistryAnswersNoForAnUnreadableRoot keeps every failure on the silent
// side: a registry that cannot be read is no evidence, never an assumed
// install. All three unreadable shapes answer the same way.
func TestRegistryAnswersNoForAnUnreadableRoot(t *testing.T) {
	home := isolateHome(t)
	spec, _ := SpecFor(HostClaudeCode)

	if registryListsPlugin(spec) {
		t.Error("a home without the registry directory reported the plugin installed")
	}
	// A registry path that exists as a file, not a directory.
	writeRegistryFile(t, home, ".claude/plugins")
	if registryListsPlugin(spec) {
		t.Error("a registry path that is a file reported the plugin installed")
	}
}

// TestRegistryAnswersNoForADirectoryItCannotOpen is the permission half of the
// failure range, kept separate because it cannot run as root.
func TestRegistryAnswersNoForADirectoryItCannotOpen(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}
	home := isolateHome(t)
	spec, _ := SpecFor(HostClaudeCode)

	// The plugin is really there; only the mode stops the scan reading it.
	writeRegistryEntry(t, home, ".claude/plugins/cache/archcore-plugins/archcore")
	root := filepath.Join(home, ".claude", "plugins")
	if err := os.Chmod(root, 0o000); err != nil {
		t.Fatalf("making the registry root unreadable: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })

	if registryListsPlugin(spec) {
		t.Error("an unreadable registry root reported the plugin installed")
	}
}

// TestRegistryAnswersNoWithoutARegistryPath covers a host table row that names
// no registry, and one that names a registry without naming the entry that
// means the plugin. Either half missing is nothing to look for.
func TestRegistryAnswersNoWithoutARegistryPath(t *testing.T) {
	tests := []struct {
		name string
		spec HostSpec
	}{
		{name: "neither", spec: HostSpec{Host: "nowhere"}},
		{name: "no entry name", spec: HostSpec{Host: "nowhere", RegistryPath: ".claude/plugins"}},
		{name: "no path", spec: HostSpec{Host: "nowhere", RegistryEntry: pluginDirName}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := isolateHome(t)
			writeRegistryEntry(t, home, ".claude/plugins/cache/archcore-plugins/archcore")
			if registryListsPlugin(tt.spec) {
				t.Error("an incomplete host row reported the plugin installed")
			}
		})
	}
}
