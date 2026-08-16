package plugin

import (
	"os"
	"path/filepath"
	"strings"
)

// The on-disk half of the evidence: each host keeps its installed plugins under
// a home-relative directory, and that directory is the only answer available
// when the host's own CLI is not on PATH — updating-the-plugin.spec §8.
//
// @internal/wiring/hooks_effective.go runs a comparable scan for the
// duplicate-hook notice. It is deliberately not shared. That one is cross-host
// and informational, and requirement 3 of plugin-cli-compatibility.rule keeps
// reads of host plugin state inside this surface, where they are allowed to
// decide something.

const (
	// registryScanDepth bounds how far below one host's registry root the scan
	// descends. It protects the step budget, and it is deep enough for the
	// nesting the hosts use: a checkout lands as
	// cache/<marketplace>/<plugin>/<version>, so a shallower scan would see only
	// the first level and never find the plugin.
	registryScanDepth = 3

	// registryScanBudget bounds how many directory entries one host's scan
	// visits. It protects the latency of `archcore init` and `archcore update`,
	// where the scan runs before any output: a host cache with thousands of
	// entries must cost a miss, never a visible pause.
	registryScanBudget = 400
)

// registryListsPlugin reports whether a host's on-disk registry names the
// Archcore plugin.
//
// Every failure answers no. An unreadable home directory, a registry that does
// not exist, a directory the user cannot read — all of them mean "no evidence
// the plugin is here", which is the reading that keeps a machine without the
// plugin silent — updating-the-plugin.spec, Failure Behavior 1.
func registryListsPlugin(spec HostSpec) bool {
	if spec.RegistryPath == "" || spec.RegistryEntry == "" {
		return false
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	budget := registryScanBudget
	root := filepath.Join(home, filepath.FromSlash(spec.RegistryPath))
	return scanRegistry(root, spec.RegistryEntry, registryScanDepth, &budget)
}

// registryNamesPlugin reports whether one registry entry is the host's installed
// plugin.
//
// It compares against the host's own RegistryEntry rather than calling
// namesPlugin, which answers true for the marketplace id as well. Every host
// stores the marketplace checkout one level above the plugin
// (cache/<marketplace>/<plugin>), so a namesPlugin test matched the marketplace
// directory and returned before the plugin level was ever read — reporting the
// plugin installed on a machine that only ever ran `marketplace add`, which
// install then treats as a no-op and never installs. keyNamesPlugin draws the
// same line for the JSON listing; this is its on-disk twin.
//
// The extension is trimmed so a marketplace snapshot stored as archcore.json
// answers the same as a directory called archcore.
func registryNamesPlugin(name, want string) bool {
	lowered := strings.ToLower(name)
	return strings.TrimSuffix(lowered, filepath.Ext(lowered)) == want
}

// scanRegistry walks up to depth levels below root, looking for an entry that
// names the plugin. Files count as well as directories: a host that keeps
// marketplace snapshots stores them as files, and the spec names the directory
// without naming what is inside it.
//
// The order is os.ReadDir's, which sorts by filename, so the same tree answers
// the same way on every run and a budget that runs out cuts the same entries.
// A symlink is never followed — ReadDir reports the link itself, so a loop
// cannot form.
func scanRegistry(root, want string, depth int, budget *int) bool {
	if depth <= 0 || *budget <= 0 {
		return false
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if *budget <= 0 {
			return false
		}
		*budget--
		if registryNamesPlugin(entry.Name(), want) {
			return true
		}
		if entry.IsDir() && scanRegistry(filepath.Join(root, entry.Name()), want, depth-1, budget) {
			return true
		}
	}
	return false
}
