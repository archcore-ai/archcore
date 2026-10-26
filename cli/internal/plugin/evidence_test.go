package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"archcore-cli/internal/testsupport"
)

// TestCollectEvidenceReadsTheHostsOwnAnswer walks the answers a host CLI can
// give. The rows that matter are the failing ones: a listing that exits
// nonzero, does not parse, or names nothing must all leave ListingOK or Listed
// clear, because that is the evidence the planner reads as "not installed" —
// updating-the-plugin.spec, Failure Behavior 1. Nothing here inspects the
// host's error text.
func TestCollectEvidenceReadsTheHostsOwnAnswer(t *testing.T) {
	skipWithoutPOSIXShell(t)

	tests := []struct {
		name        string
		script      string
		wantListing bool
		wantListed  bool
		wantVersion string
	}{
		{
			name:        "a listing that names the plugin",
			script:      listingScript(pluginListing, "exit 0"),
			wantListing: true,
			wantListed:  true,
			wantVersion: "1.4.0",
		},
		{
			name:        "a listing that names other plugins",
			script:      listingScript(otherPluginListing, "exit 0"),
			wantListing: true,
		},
		{
			name:        "an empty listing",
			script:      listingScript("[]", "exit 0"),
			wantListing: true,
		},
		{
			name:   "malformed output",
			script: listingScript("not json at all {", "exit 0"),
		},
		{
			name:   "a listing that exits nonzero",
			script: "exit 1",
		},
		{
			name:   "a listing that prints nothing at all",
			script: "exit 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateHome(t)
			dir := t.TempDir()
			writeHostCLI(t, dir, "claude", tt.script)
			useHostCLIs(t, dir, "claude")

			evidence := CollectEvidence(t.Context(), []Host{HostClaudeCode}, "")
			if len(evidence) != 1 {
				t.Fatalf("collected %d observations, want 1", len(evidence))
			}
			got := evidence[0]
			if !got.CLIPresent {
				t.Error("CLIPresent = false for a host whose binary resolved")
			}
			if got.ListingOK != tt.wantListing {
				t.Errorf("ListingOK = %v, want %v", got.ListingOK, tt.wantListing)
			}
			if got.Listed != tt.wantListed {
				t.Errorf("Listed = %v, want %v", got.Listed, tt.wantListed)
			}
			if got.ListedVersion != tt.wantVersion {
				t.Errorf("ListedVersion = %q, want %q", got.ListedVersion, tt.wantVersion)
			}
		})
	}
}

// TestCollectEvidenceTimesOutALockedListing proves a host CLI that never
// returns cannot hold the collection: the command bound cuts it and the host
// reads as unanswered.
func TestCollectEvidenceTimesOutALockedListing(t *testing.T) {
	skipWithoutPOSIXShell(t)
	isolateHome(t)
	dir := t.TempDir()
	writeHostCLI(t, dir, "claude", `sleep 30`)
	useHostCLIs(t, dir, "claude")
	shrinkTimeouts(t, 50*time.Millisecond, time.Minute)

	evidence := CollectEvidence(t.Context(), []Host{HostClaudeCode}, "")
	if len(evidence) != 1 {
		t.Fatalf("collected %d observations, want 1", len(evidence))
	}
	if evidence[0].ListingOK || evidence[0].Listed {
		t.Errorf("a timed-out listing produced %+v, want an unanswered host", evidence[0])
	}
}

// TestCollectEvidenceRefusesATruncatedListing keeps the output ceiling from
// becoming a silent shortening: a listing cut at the cap is no answer, because
// a prefix that lost the entry would report the plugin absent and skip the host
// — bounded-and-deterministic-output.rule.
func TestCollectEvidenceRefusesATruncatedListing(t *testing.T) {
	isolateHome(t)
	useHostCLIs(t, t.TempDir(), "claude")
	stubRuns(t, func(Command) commandOutcome {
		return commandOutcome{Stdout: pluginListing, Truncated: true}
	})

	evidence := CollectEvidence(t.Context(), []Host{HostClaudeCode}, "")
	if len(evidence) != 1 {
		t.Fatalf("collected %d observations, want 1", len(evidence))
	}
	if evidence[0].ListingOK || evidence[0].Listed {
		t.Errorf("a truncated listing produced %+v, want an unanswered host", evidence[0])
	}
}

// TestCollectEvidenceAsksNothingOfAnAbsentCLI proves the order: with no binary
// on PATH there is no listing to run, and the on-disk registry is the only
// evidence left — updating-the-plugin.spec §8.
func TestCollectEvidenceAsksNothingOfAnAbsentCLI(t *testing.T) {
	home := isolateHome(t)
	noHostCLIs(t)
	rec := recordRuns(t)
	writeRegistryEntry(t, home, ".claude/plugins/cache/archcore-plugins/archcore")

	evidence := CollectEvidence(t.Context(), []Host{HostClaudeCode}, "")
	if len(rec.commands) != 0 {
		t.Errorf("ran %q with no host CLI on PATH, want no command at all", rec.lines())
	}
	if len(evidence) != 1 {
		t.Fatalf("collected %d observations, want 1", len(evidence))
	}
	got := evidence[0]
	if got.CLIPresent || got.ListingOK {
		t.Errorf("evidence = %+v, want no host answer", got)
	}
	if !got.RegistryListed {
		t.Error("RegistryListed = false over a registry that names the plugin")
	}
}

// TestCollectEvidenceCoversEveryPluginHost keeps the collector aligned with the
// host table: every host that ships a plugin is observed, in the canonical
// order, and a host that ships none is dropped rather than reported empty.
func TestCollectEvidenceCoversEveryPluginHost(t *testing.T) {
	isolateHome(t)
	noHostCLIs(t)

	hosts := append(Hosts(), "gemini-cli", "roo-code", "not-a-host")
	evidence := CollectEvidence(t.Context(), hosts, "")
	if len(evidence) != len(Hosts()) {
		t.Fatalf("collected %d observations for %d hosts, want %d", len(evidence), len(hosts), len(Hosts()))
	}
	for i, host := range Hosts() {
		if evidence[i].Host != host {
			t.Errorf("observation %d is for %q, want %q", i, evidence[i].Host, host)
		}
		if !reflect.DeepEqual(evidence[i], Evidence{Host: host}) {
			t.Errorf("observation for %q = %+v, want no evidence at all", host, evidence[i])
		}
	}
}

// TestParseJSONListingToleratesTheEnvelope pins the tolerance the unverified
// listing schema needs. A parser bound to one shape would answer "not
// installed" the first time a host moved its entries, and that verdict is the
// silent one, so the miss would never surface.
func TestParseJSONListingToleratesTheEnvelope(t *testing.T) {
	tests := []struct {
		name        string
		answer      string
		wantOK      bool
		wantListed  bool
		wantVersion string
	}{
		{
			name:   "an empty array",
			answer: `[]`,
			wantOK: true,
		},
		{
			name:       "a bare array of ids",
			answer:     `["archcore@archcore-plugins","other@else"]`,
			wantOK:     true,
			wantListed: true,
		},
		{
			name:        "an array of entries",
			answer:      `[{"name":"archcore","version":"2.0.1","enabled":true}]`,
			wantOK:      true,
			wantListed:  true,
			wantVersion: "2.0.1",
		},
		{
			name:        "entries under a top-level key",
			answer:      `{"plugins":[{"id":"archcore@archcore-plugins","version":"3.1.0"}]}`,
			wantOK:      true,
			wantListed:  true,
			wantVersion: "3.1.0",
		},
		{
			name:        "entries grouped by marketplace",
			answer:      `{"marketplaces":{"archcore-plugins":{"plugins":[{"name":"archcore","pluginVersion":"0.9"}]}}}`,
			wantOK:      true,
			wantListed:  true,
			wantVersion: "0.9",
		},
		{
			name:       "a plugin with no version reported",
			answer:     `[{"name":"archcore@archcore-plugins"}]`,
			wantOK:     true,
			wantListed: true,
		},
		{
			name:   "another project's plugin",
			answer: `[{"name":"archcore-cli-helper@somewhere-else","version":"1.0.0"}]`,
			wantOK: true,
		},
		{
			name:   "not JSON",
			answer: `plugin list is not implemented`,
		},
		{
			name:   "nothing at all",
			answer: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJSONListing(tt.answer)
			if got.ok != tt.wantOK || got.listed != tt.wantListed || got.version != tt.wantVersion {
				t.Errorf("parsed %+v, want ok=%v listed=%v version=%q",
					got, tt.wantOK, tt.wantListed, tt.wantVersion)
			}
		})
	}
}

// TestParseJSONListingReadsAKeyedEnvelope covers the host that names the plugin
// in an object KEY rather than in a value: `{"archcore@archcore-plugins": {…}}`.
// Reading only values answers "not installed" there, and that verdict is the
// silent one — an unlisted host is skipped with no output at all
// (updating-the-plugin.spec §7) — so a real installation would be skipped on
// every run and no error would ever point at it.
//
// The last row is the reason a key is not read with namesPlugin: that predicate
// also answers true for the marketplace id, and a registered marketplace with no
// plugins under it is the opposite of an installation.
func TestParseJSONListingReadsAKeyedEnvelope(t *testing.T) {
	tests := []struct {
		name        string
		answer      string
		wantListed  bool
		wantVersion string
	}{
		{
			name:        "keyed by plugin id",
			answer:      `{"archcore@archcore-plugins":{"version":"1.4.0","enabled":true}}`,
			wantListed:  true,
			wantVersion: "1.4.0",
		},
		{
			name:        "keyed by plugin id under a top-level key",
			answer:      `{"plugins":{"archcore@archcore-plugins":{"version":"2.0.0"}}}`,
			wantListed:  true,
			wantVersion: "2.0.0",
		},
		{
			name:        "keyed by the bare plugin name",
			answer:      `{"archcore":{"pluginVersion":"3.0.0"}}`,
			wantListed:  true,
			wantVersion: "3.0.0",
		},
		{
			name:       "keyed by plugin id with no version reported",
			answer:     `{"archcore@archcore-plugins":{"enabled":true}}`,
			wantListed: true,
		},
		{
			name:   "keyed by another project's plugin",
			answer: `{"someone-else@other-marketplace":{"version":"9.9.9"}}`,
		},
		{
			name:   "a registered marketplace with no plugins under it",
			answer: `{"marketplaces":{"archcore-plugins":{"plugins":[]}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJSONListing(tt.answer)
			if !got.ok {
				t.Fatalf("parsed %+v, want an answered host", got)
			}
			if got.listed != tt.wantListed || got.version != tt.wantVersion {
				t.Errorf("parsed %+v, want listed=%v version=%q", got, tt.wantListed, tt.wantVersion)
			}
		})
	}
}

// TestParseJSONListingCountsOnlyAnInstalledPlugin covers the two answers that
// name the plugin without installing it. Both once read as a presence claim,
// and presence is the reading that authorizes a mutating host command
// (updating-the-plugin.spec §6) and turns an install into a reported no-op that
// never installs (plugin-delivery.spec §9).
//
// The first is a host that lists what a registered marketplace offers beside
// what is installed: `codex plugin list --json --available` prints exactly that,
// with `"installed": false` on the offered entries. The second is the marketplace
// id carried in an identity value — the case the key reading already excluded
// while the value reading did not.
func TestParseJSONListingCountsOnlyAnInstalledPlugin(t *testing.T) {
	tests := []struct {
		name       string
		answer     string
		wantListed bool
	}{
		{
			name:   "an entry the host reports as not installed",
			answer: `{"installed":[],"available":[{"pluginId":"archcore@archcore-plugins","name":"archcore","installed":false}]}`,
		},
		{
			name:   "a quoted installation flag reading false",
			answer: `[{"id":"archcore@archcore-plugins","installed":"false","version":"1.0.0"}]`,
		},
		{
			name:   "an installation flag of an unreadable shape",
			answer: `[{"id":"archcore@archcore-plugins","installed":{"state":"pending"}}]`,
		},
		{
			name:   "the marketplace id in an identity value",
			answer: `[{"name":"archcore-plugins","version":"1.0.0"}]`,
		},
		{
			name:   "the marketplace id in a bare array of ids",
			answer: `["archcore-plugins"]`,
		},
		{
			name:   "an entry keyed by the plugin id and reported as not installed",
			answer: `{"archcore@archcore-plugins":{"version":"1.0.0","installed":false}}`,
		},
		{
			name:       "an installed entry beside the offered ones",
			answer:     `{"installed":[{"pluginId":"archcore@archcore-plugins","installed":true,"version":"0.8.0"}],"available":[{"pluginId":"other@else","installed":false}]}`,
			wantListed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJSONListing(tt.answer)
			if !got.ok {
				t.Fatalf("parsed %+v, want an answered host", got)
			}
			if got.listed != tt.wantListed {
				t.Errorf("parsed %+v, want listed=%v", got, tt.wantListed)
			}
		})
	}
}

// TestParseJSONListingKeepsItsBoundsOverKeys keeps the key reading inside the
// ceilings the walk already had: a keyed entry nested past the depth cap is not
// found, because the walk that would reach it is what the cap stops —
// bounded-and-deterministic-output.rule.
func TestParseJSONListingKeepsItsBoundsOverKeys(t *testing.T) {
	answer := `{"archcore@archcore-plugins":{"version":"1.0.0"}}`
	for range maxListingDepth {
		answer = `{"nested":` + answer + `}`
	}

	if got := parseJSONListing(answer); got.listed {
		t.Errorf("parsed %+v, want a walk that stopped at the depth cap", got)
	}
}

// TestParseJSONListingIsDeterministic proves two walks over one answer pick the
// same entry, and that the entry they pick is the one sorted order names.
//
// Object keys are visited in sorted order for exactly this reason: Go
// randomizes map iteration, and an unordered walk would report a different
// version between runs on an answer that carries more than one match.
//
// The literal expectation is what makes the case load-bearing. Comparing 50
// walks against a first walk of the same function proves self-consistency and
// nothing else — reversing the sort would satisfy it too. Key "a" sorts first,
// so "1.0.0" is the answer sorted order requires.
func TestParseJSONListingIsDeterministic(t *testing.T) {
	answer := `{"b":{"name":"archcore","version":"2.0.0"},"a":{"name":"archcore","version":"1.0.0"}}`

	const wantVersion = "1.0.0"
	first := parseJSONListing(answer)
	if !first.listed || first.version != wantVersion {
		t.Fatalf("parsed %+v, want the entry under the first sorted key, version %q", first, wantVersion)
	}
	for i := range 50 {
		if got := parseJSONListing(answer); !reflect.DeepEqual(got, first) {
			t.Fatalf("walk %d parsed %+v, want the same answer as the first walk %+v", i, got, first)
		}
	}
}

// TestParseTextListingReadsAPlainAnswer covers the one host whose listing is
// not JSON. A listing that ran and named nothing is a parsed answer, not a
// failure: the host was asked and it replied.
func TestParseTextListingReadsAPlainAnswer(t *testing.T) {
	tests := []struct {
		name        string
		answer      string
		wantListed  bool
		wantVersion string
	}{
		{
			name:        "a table naming the plugin",
			answer:      "PLUGIN                      VERSION\narchcore                    1.2.3\n",
			wantListed:  true,
			wantVersion: "1.2.3",
		},
		{
			name:        "the marketplace-qualified id",
			answer:      "archcore@archcore-plugins  v0.4.0  enabled\n",
			wantListed:  true,
			wantVersion: "v0.4.0",
		},
		{
			name:       "no version printed",
			answer:     "archcore  enabled\n",
			wantListed: true,
		},
		{
			name:   "other plugins only",
			answer: "some-other-plugin  1.0.0\n",
		},
		{
			// A marketplace named on its own line is a registration, not an
			// installation — the plain-text twin of the identity-value case.
			name:   "a marketplace section with nothing under it",
			answer: "MARKETPLACE       ROOT\narchcore-plugins  /home/someone/.codex/.tmp/marketplaces\n",
		},
		{
			name:   "no plugins installed",
			answer: "No plugins installed.\n",
		},
		{
			name:   "nothing at all",
			answer: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTextListing(tt.answer)
			if !got.ok {
				t.Error("a plain-text listing that ran was reported unparsed")
			}
			if got.listed != tt.wantListed || got.version != tt.wantVersion {
				t.Errorf("parsed %+v, want listed=%v version=%q", got, tt.wantListed, tt.wantVersion)
			}
		})
	}
}

// TestListedVersionIsBounded keeps a host's answer from leaving this package
// unbounded. ListedVersion is echoed into a status report, so its size is the
// host's choice until a ceiling makes it this program's — and the ceiling drops
// an oversized token rather than shortening it, because a shortened version
// reads like a real one — bounded-and-deterministic-output.rule.
func TestListedVersionIsBounded(t *testing.T) {
	// Version-shaped, because the plain-text extraction only recognizes a token
	// that already looks like a version.
	oversized := "1.0." + strings.Repeat("9", maxListedVersion)

	tests := []struct {
		name   string
		parse  func(string) listing
		answer string
	}{
		{
			name:   "json",
			parse:  parseJSONListing,
			answer: `[{"name":"archcore@archcore-plugins","version":"` + oversized + `"}]`,
		},
		{
			name:   "plain text",
			parse:  parseTextListing,
			answer: "archcore  " + oversized + "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.parse(tt.answer)
			if !got.ok || !got.listed {
				t.Fatalf("parsed %+v, want the plugin found", got)
			}
			if got.version != "" {
				t.Errorf("ListedVersion is %d bytes, want an oversized token dropped", len(got.version))
			}
			// A version at the ceiling is still a version.
			atCeiling := strings.Replace(tt.answer, oversized, oversized[:maxListedVersion], 1)
			if v := tt.parse(atCeiling).version; v != oversized[:maxListedVersion] {
				t.Errorf("a token at the ceiling parsed as %q, want it kept", v)
			}
		})
	}
}

// TestNamesPluginIsTightEnoughForAProjectDirectory is the guard against the
// substring match this scan is tempted into. ~/.codex keys entries by project
// directory, so a checkout of this repository sits under it — and a loose match
// would read that as an installed plugin and print an update command on a
// machine that never installed one.
func TestNamesPluginIsTightEnoughForAProjectDirectory(t *testing.T) {
	// name and input are separate fields: reusing one for both left the
	// empty-string row with a subtest called "#10", which names nothing a
	// failure could be read from.
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "the plugin directory", input: "archcore", want: true},
		{name: "case folded", input: "Archcore", want: true},
		{name: "a snapshot file", input: "archcore.json", want: true},
		{name: "the marketplace id", input: "archcore-plugins", want: true},
		{name: "the qualified plugin id", input: "archcore@archcore-plugins", want: true},
		{name: "a marketplace snapshot file", input: "archcore-plugins.json", want: true},

		{name: "a project directory keyed by path", input: "-Users-someone-Documents-archcore-cli", want: false},
		{name: "this repository", input: "archcore-cli", want: false},
		{name: "the organization", input: "archcore-ai", want: false},
		{name: "an unrelated directory", input: "my-archcore-notes", want: false},
		{name: "empty", input: "", want: false},
		{name: "another plugin", input: "other-plugin", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := namesPlugin(tt.input); got != tt.want {
				t.Errorf("namesPlugin(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestParseJSONListingCollectsEveryInstallation pins updating-the-plugin.spec
// §19: Claude Code lists the plugin once per scope and project, and every one
// of those entries is an installation the update has to address.
func TestParseJSONListingCollectsEveryInstallation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		answer string
		want   []Install
	}{
		{
			name: "one entry per scope, as claude plugin list --json answers",
			answer: `[
				{"id":"archcore@archcore-plugins","scope":"local","projectPath":"/work/b","version":"0.8.3"},
				{"id":"other@elsewhere","scope":"user"},
				{"id":"archcore@archcore-plugins","scope":"project","projectPath":"/work/a","version":"0.7.1"},
				{"id":"archcore@archcore-plugins","scope":"user","version":"0.9.0"},
				{"id":"archcore@archcore-plugins","scope":"local","projectPath":"/work/b","version":"0.8.3"}
			]`,
			want: []Install{
				{Name: PluginID, Scope: ScopeUser, Version: "0.9.0"},
				{Name: PluginID, Scope: ScopeProject, ProjectPath: "/work/a", Version: "0.7.1"},
				{Name: PluginID, Scope: ScopeLocal, ProjectPath: "/work/b", Version: "0.8.3"},
			},
		},
		{
			name:   "entries grouped under the plugin id",
			answer: `{"archcore@archcore-plugins":[{"scope":"local","projectPath":"/work/a"},{"scope":"user"}]}`,
			want: []Install{
				{Name: PluginID, Scope: ScopeUser},
				{Name: PluginID, Scope: ScopeLocal, ProjectPath: "/work/a"},
			},
		},
		{
			name: "an uninstalled entry and the marketplace id are not installations",
			answer: `{"installed":[{"pluginId":"archcore@archcore-plugins","name":"archcore","installed":true}],
				"available":[{"name":"archcore-plugins"},{"id":"archcore@archcore-plugins","installed":false}]}`,
			want: []Install{{Name: "archcore"}},
		},
		{
			name:   "a plugin id keyed to a bare value",
			answer: `{"archcore@archcore-plugins": true}`,
			want:   []Install{{Name: PluginID}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJSONListing(tt.answer)
			if !got.listed {
				t.Fatalf("listing not read as listed: %+v", got)
			}
			if !reflect.DeepEqual(got.installs, tt.want) {
				t.Errorf("installs = %+v, want %+v", got.installs, tt.want)
			}
		})
	}
}

// TestParseTextListingNamesTheInstallation pins the name Copilot lists a direct
// install under, which is the only name its update accepts — verified live on
// copilot 1.0.83.
func TestParseTextListingNamesTheInstallation(t *testing.T) {
	t.Parallel()
	got := parseTextListing("Installed plugins:\n  • archcore v0.6.1\n")
	want := []Install{{Name: "archcore", Version: "v0.6.1"}}
	if !got.listed || got.version != "v0.6.1" || !reflect.DeepEqual(got.installs, want) {
		t.Errorf("listing = %+v, want listed with installs %+v", got, want)
	}
}

// TestCollectEvidenceLocatesInstallationProjects pins updating-the-plugin.spec
// §21: only an absolute, fully resolved path to an existing directory counts as
// present. Claude Code matches the resolved working directory, so a recorded
// symlink would send a scoped update to a different project.
func TestCollectEvidenceLocatesInstallationProjects(t *testing.T) {
	isolateHome(t)
	useHostCLIs(t, t.TempDir(), "claude")
	present, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(present, "not-a-directory")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(present, link); err != nil {
		t.Fatal(err)
	}
	answer, err := json.Marshal([]map[string]string{
		{"id": PluginID, "scope": "local", "projectPath": present},
		{"id": PluginID, "scope": "local", "projectPath": link},
		{"id": PluginID, "scope": "local", "projectPath": filepath.Join(present, "gone")},
		{"id": PluginID, "scope": "local", "projectPath": file},
		{"id": PluginID, "scope": "project", "projectPath": "relative/project"},
	})
	if err != nil {
		t.Fatal(err)
	}
	stubRuns(t, func(Command) commandOutcome { return commandOutcome{Stdout: string(answer)} })

	evidence := CollectEvidence(t.Context(), []Host{HostClaudeCode}, "")
	if len(evidence) != 1 {
		t.Fatalf("collected %d observations, want 1", len(evidence))
	}
	got := make(map[string]bool, len(evidence[0].Installs))
	for _, install := range evidence[0].Installs {
		got[install.ProjectPath] = install.ProjectPresent
	}
	want := map[string]bool{
		present:                        true,
		link:                           false,
		filepath.Join(present, "gone"): false,
		file:                           false,
		"relative/project":             false,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("project presence = %v, want %v", got, want)
	}
}

// TestParseJSONListingReadsAnUninstalledGroupAsAbsent pins that presence and the
// installations come from one walk: a group keyed by the plugin id whose only
// entry is uninstalled lists nothing — updating-the-plugin.spec §17.
func TestParseJSONListingReadsAnUninstalledGroupAsAbsent(t *testing.T) {
	t.Parallel()
	got := parseJSONListing(`{"archcore@archcore-plugins":[{"installed":false,"scope":"user"}]}`)
	if !got.ok || got.listed || len(got.installs) != 0 {
		t.Errorf("listing = %+v, want parsed and not listed", got)
	}
}

// TestCollectEvidenceReadsTheOpenCodeConfig walks the config files OpenCode
// loads its plugins from. Each file name counts on its own, the files may be
// JSONC, and a pinned spec names the package as much as a bare one does. A
// lookalike package, a comment, or a key outside `plugin` names nothing.
func TestCollectEvidenceReadsTheOpenCodeConfig(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		content    string
		wantListed bool
	}{
		{
			name:       "the project config dir",
			file:       ".opencode/opencode.json",
			content:    `{"plugin": ["@archcore/opencode-plugin"]}`,
			wantListed: true,
		},
		{
			name: "a jsonc file with comments, a pinned spec, and trailing commas",
			file: ".opencode/opencode.jsonc",
			content: `{
  // the Archcore bridge
  "$schema": "https://opencode.ai/config.json",
  /* pinned on purpose */
  "plugin": [
    "other-plugin",
    "@archcore/opencode-plugin@1.2.3", // pinned
  ],
}`,
			wantListed: true,
		},
		{
			name:       "the root config",
			file:       "opencode.json",
			content:    `{"plugin":["@archcore/opencode-plugin@latest"]}`,
			wantListed: true,
		},
		{
			name:       "the root jsonc config",
			file:       "opencode.jsonc",
			content:    "{\n  \"plugin\": [\"@archcore/opencode-plugin\"] // bare\n}",
			wantListed: true,
		},
		{
			name:    "a lookalike package",
			file:    ".opencode/opencode.json",
			content: `{"plugin": ["@archcore/opencode-plugin-extra", "@archcore/opencode"]}`,
		},
		{
			name:    "the package only inside a comment",
			file:    ".opencode/opencode.json",
			content: "{\n  // \"plugin\": [\"@archcore/opencode-plugin\"]\n  \"plugin\": []\n}",
		},
		{
			name:    "the package outside the plugin array",
			file:    "opencode.json",
			content: `{"mcp": {"@archcore/opencode-plugin": {}}, "plugin": ["a//b", "/*c*/"]}`,
		},
		{
			name:    "a config that does not parse",
			file:    "opencode.json",
			content: `{"plugin": ["@archcore/opencode-plugin"`,
		},
		{
			name:    "a plugin key that is not an array",
			file:    "opencode.json",
			content: `{"plugin": "@archcore/opencode-plugin"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateHome(t)
			noHostCLIs(t)
			root := t.TempDir()
			writeProjectFile(t, root, tt.file, tt.content)

			evidence := CollectEvidence(t.Context(), []Host{HostOpenCode}, root)
			if len(evidence) != 1 {
				t.Fatalf("collected %d observations, want 1", len(evidence))
			}
			got := evidence[0]
			if got.ProjectRoot != root {
				t.Errorf("ProjectRoot = %q, want %q", got.ProjectRoot, root)
			}
			if got.RegistryListed != tt.wantListed {
				t.Errorf("RegistryListed = %v, want %v", got.RegistryListed, tt.wantListed)
			}
			if got.ListingOK || got.Listed {
				t.Errorf("evidence = %+v, want no listing answer without the CLI", got)
			}
		})
	}
}

// TestCollectEvidenceReadsTheOpenCodeConfigAsItsListing proves the config
// stands as the listing once the CLI is present, so the install tier can report
// a configured project instead of rerunning `opencode plugin`.
func TestCollectEvidenceReadsTheOpenCodeConfigAsItsListing(t *testing.T) {
	isolateHome(t)
	useHostCLIs(t, t.TempDir(), "opencode")
	rec := recordRuns(t)
	root := t.TempDir()
	writeProjectFile(t, root, ".opencode/opencode.json", `{"plugin":["@archcore/opencode-plugin"]}`)

	evidence := CollectEvidence(t.Context(), []Host{HostOpenCode}, root)
	if want := []string{"opencode --version"}; !slices.Equal(rec.lines(), want) {
		t.Errorf("ran %q, want the version read and the config read alone", rec.lines())
	}
	want := Evidence{Host: HostOpenCode, CLIPresent: true, ListingOK: true, Listed: true, RegistryListed: true, ProjectRoot: root}
	if len(evidence) != 1 || !reflect.DeepEqual(evidence[0], want) {
		t.Errorf("evidence = %+v, want %+v", evidence, want)
	}
}

// TestCollectEvidenceReadsNoOpenCodeConfigWithoutAProject keeps the read off
// the process's working directory: with no project root there is nowhere the
// config belongs, and the evidence names no project for Plan to run in.
func TestCollectEvidenceReadsNoOpenCodeConfigWithoutAProject(t *testing.T) {
	isolateHome(t)
	noHostCLIs(t)
	cwd := t.TempDir()
	writeProjectFile(t, cwd, "opencode.json", `{"plugin":["@archcore/opencode-plugin"]}`)
	t.Chdir(cwd)

	evidence := CollectEvidence(t.Context(), []Host{HostOpenCode}, "")
	want := []Evidence{{Host: HostOpenCode}}
	if !reflect.DeepEqual(evidence, want) {
		t.Errorf("evidence = %+v, want %+v", evidence, want)
	}
}

// TestCollectEvidenceReadsTheOpenCodeConfigFormsOpenCodeAccepts covers the
// spellings OpenCode itself loads: a `[spec, options]` pair names its package
// (packages/opencode/src/config/plugin.ts, pluginSpecifier), and a file saved
// with a UTF-8 byte order mark is still the same config.
func TestCollectEvidenceReadsTheOpenCodeConfigFormsOpenCodeAccepts(t *testing.T) {
	for name, content := range map[string]string{
		"a spec and options pair": `{"plugin": [["@archcore/opencode-plugin@1.0.0", {"x": 1}]]}`,
		"a byte order mark":       "\xef\xbb\xbf" + `{"plugin": ["@archcore/opencode-plugin"]}`,
	} {
		t.Run(name, func(t *testing.T) {
			isolateHome(t)
			noHostCLIs(t)
			root := t.TempDir()
			writeProjectFile(t, root, "opencode.json", content)

			evidence := CollectEvidence(t.Context(), []Host{HostOpenCode}, root)
			if len(evidence) != 1 || !evidence[0].RegistryListed {
				t.Errorf("evidence = %+v, want the package listed", evidence)
			}
		})
	}
}

// TestCollectEvidenceReadsTheOpenCodeConfigAtTheGitRoot follows OpenCode into
// a project below the root of its git working tree: `opencode plugin` writes
// .opencode/opencode.json at that root (install.ts, patchDir), and OpenCode
// loads every config between the session directory and the root. A config
// above the working tree belongs to no session in the project.
func TestCollectEvidenceReadsTheOpenCodeConfigAtTheGitRoot(t *testing.T) {
	tests := []struct {
		name       string
		file       string // relative to the directory above the repository
		wantListed bool
	}{
		{name: "the root of the working tree", file: "repo/.opencode/opencode.json", wantListed: true},
		{name: "a directory between the root and the project", file: "repo/services/opencode.jsonc", wantListed: true},
		{name: "the project itself", file: "repo/services/api/.opencode/opencode.json", wantListed: true},
		{name: "above the working tree", file: ".opencode/opencode.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateHome(t)
			useHostCLIs(t, t.TempDir(), "opencode")
			outer := t.TempDir()
			repo := filepath.Join(outer, "repo")
			if err := os.Mkdir(repo, 0o755); err != nil {
				t.Fatal(err)
			}
			testsupport.NewGitRepo(t, repo)
			project := filepath.Join(repo, "services", "api")
			writeProjectFile(t, project, ".archcore/settings.json", "{}")
			writeProjectFile(t, outer, tt.file, `{"plugin": ["@archcore/opencode-plugin"]}`)

			evidence := CollectEvidence(t.Context(), []Host{HostOpenCode}, project)
			want := Evidence{Host: HostOpenCode, CLIPresent: true, ListingOK: true, Listed: tt.wantListed, RegistryListed: tt.wantListed, ProjectRoot: project}
			if len(evidence) != 1 || !reflect.DeepEqual(evidence[0], want) {
				t.Errorf("evidence = %+v, want %+v", evidence, want)
			}
		})
	}
}

// TestCollectEvidenceLeavesAnUnreadableOpenCodeConfigUnanswered keeps a config
// the CLI could not read from passing for proof of absence. With the CLI
// present, absence is what lets removal stay silent (planRemove), so a broken
// or oversized config must answer nothing — the rule a host listing that did
// not parse already follows. A project with no config at all is a real answer.
func TestCollectEvidenceLeavesAnUnreadableOpenCodeConfigUnanswered(t *testing.T) {
	tests := []struct {
		name          string
		content       string // empty writes no config file
		wantListingOK bool
	}{
		{name: "no config file", wantListingOK: true},
		{name: "a config that lists other plugins", content: `{"plugin": ["other"]}`, wantListingOK: true},
		{name: "a config that does not parse", content: `{"plugin": ["@archcore/opencode-plugin"`},
		{name: "a config above the size bound", content: `{"plugin": [], "pad": "` + strings.Repeat("x", maxProjectConfigBytes) + `"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateHome(t)
			useHostCLIs(t, t.TempDir(), "opencode")
			root := t.TempDir()
			if tt.content != "" {
				writeProjectFile(t, root, ".opencode/opencode.json", tt.content)
			}

			evidence := CollectEvidence(t.Context(), []Host{HostOpenCode}, root)
			want := Evidence{Host: HostOpenCode, CLIPresent: true, ListingOK: tt.wantListingOK, ProjectRoot: root}
			if len(evidence) != 1 || !reflect.DeepEqual(evidence[0], want) {
				t.Fatalf("evidence = %+v, want %+v", evidence, want)
			}
			// The planner's side of the same answer: an unanswered listing
			// still prints the removal note, a real absence prints nothing.
			actions := Plan(VerbRemove, evidence)
			if gotNote := len(actions) == 1 && actions[0].Kind == ActionPrintUINote; gotNote == tt.wantListingOK {
				t.Errorf("remove plan = %+v, want a removal note: %v", actions, !tt.wantListingOK)
			}
		})
	}
}

// TestCollectEvidenceTakesAListingOverAnUnreadableNeighbour keeps one broken
// config from hiding another that lists the package: a listing is an answer
// whatever the other files hold.
func TestCollectEvidenceTakesAListingOverAnUnreadableNeighbour(t *testing.T) {
	isolateHome(t)
	useHostCLIs(t, t.TempDir(), "opencode")
	root := t.TempDir()
	writeProjectFile(t, root, ".opencode/opencode.json", `{"plugin": [`)
	writeProjectFile(t, root, "opencode.json", `{"plugin": ["@archcore/opencode-plugin"]}`)

	evidence := CollectEvidence(t.Context(), []Host{HostOpenCode}, root)
	want := Evidence{Host: HostOpenCode, CLIPresent: true, ListingOK: true, Listed: true, RegistryListed: true, ProjectRoot: root}
	if len(evidence) != 1 || !reflect.DeepEqual(evidence[0], want) {
		t.Errorf("evidence = %+v, want %+v", evidence, want)
	}
}

// TestCollectEvidenceReadsTheOpenCodeMajorVersion picks the command line of the
// OpenCode version on PATH: 1.x prints a bare version, 2.x prefixes its name,
// and an answer without a version keeps the 1.x commands.
func TestCollectEvidenceReadsTheOpenCodeMajorVersion(t *testing.T) {
	for _, tt := range []struct {
		out  commandOutcome
		want int
	}{
		{out: commandOutcome{Stdout: "1.18.34\n"}, want: 1},
		{out: commandOutcome{Stdout: "opencode v2.1.4\n"}, want: 2},
		{out: commandOutcome{Stdout: "dev build\n"}, want: 0},
		{out: commandOutcome{Failed: true, Stdout: "2.0.0"}, want: 0},
	} {
		t.Run(tt.out.Stdout, func(t *testing.T) {
			isolateHome(t)
			useHostCLIs(t, t.TempDir(), "opencode")
			stubRuns(t, func(Command) commandOutcome { return tt.out })

			evidence := CollectEvidence(t.Context(), []Host{HostOpenCode}, t.TempDir())
			if len(evidence) != 1 || evidence[0].HostMajor != tt.want {
				t.Errorf("evidence = %+v, want HostMajor %d", evidence, tt.want)
			}
		})
	}
}

// TestCollectEvidenceReadsTheGlobalOpenCodeConfig finds the entry that
// `opencode plugin add` (2.x) and `opencode plugin -g` (1.x) write: the global
// config, under the 2.x `plugins` key in its object form as well.
func TestCollectEvidenceReadsTheGlobalOpenCodeConfig(t *testing.T) {
	for name, content := range map[string]string{
		"2.x plugins with an object": `{"plugins": [{"package": "@archcore/opencode-plugin", "options": {}}]}`,
		"2.x plugins with a string":  `{"plugins": ["@archcore/opencode-plugin@0.15.0"]}`,
		"1.x plugin in a jsonc file": `{"plugin": ["@archcore/opencode-plugin"], // global
}`,
	} {
		t.Run(name, func(t *testing.T) {
			isolateHome(t)
			noHostCLIs(t)
			config := t.TempDir()
			t.Setenv("OPENCODE_CONFIG_DIR", "")
			t.Setenv("XDG_CONFIG_HOME", config)
			file := "opencode.json"
			if strings.Contains(name, "jsonc") {
				file = "opencode.jsonc"
			}
			writeProjectFile(t, config, "opencode/"+file, content)

			evidence := CollectEvidence(t.Context(), []Host{HostOpenCode}, t.TempDir())
			if len(evidence) != 1 || !evidence[0].RegistryListed {
				t.Errorf("evidence = %+v, want the global entry listed", evidence)
			}
		})
	}
}

// TestCollectEvidenceReadsTheGlobalOpenCodeConfigWithoutAProject is the
// evidence `archcore update` gets: no project, only the global config.
func TestCollectEvidenceReadsTheGlobalOpenCodeConfigWithoutAProject(t *testing.T) {
	isolateHome(t)
	useHostCLIs(t, t.TempDir(), "opencode")
	stubRuns(t, func(Command) commandOutcome { return commandOutcome{Stdout: "opencode v2.1.4"} })
	config := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", config)
	writeProjectFile(t, config, "opencode/opencode.json", `{"plugins": ["@archcore/opencode-plugin"]}`)

	evidence := CollectEvidence(t.Context(), []Host{HostOpenCode}, "")
	want := Evidence{Host: HostOpenCode, CLIPresent: true, ListingOK: true, Listed: true, RegistryListed: true, HostMajor: 2}
	if len(evidence) != 1 || !reflect.DeepEqual(evidence[0], want) {
		t.Errorf("evidence = %+v, want %+v", evidence, want)
	}
}

func writeProjectFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
