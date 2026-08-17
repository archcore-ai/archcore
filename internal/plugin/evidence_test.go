package plugin

import (
	"strings"
	"testing"
	"time"
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

			evidence := CollectEvidence(t.Context(), []Host{HostClaudeCode})
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

	evidence := CollectEvidence(t.Context(), []Host{HostClaudeCode})
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

	evidence := CollectEvidence(t.Context(), []Host{HostClaudeCode})
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

	evidence := CollectEvidence(t.Context(), []Host{HostClaudeCode})
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

	hosts := append(Hosts(), "gemini-cli", "opencode", "not-a-host")
	evidence := CollectEvidence(t.Context(), hosts)
	if len(evidence) != len(Hosts()) {
		t.Fatalf("collected %d observations for %d hosts, want %d", len(evidence), len(hosts), len(Hosts()))
	}
	for i, host := range Hosts() {
		if evidence[i].Host != host {
			t.Errorf("observation %d is for %q, want %q", i, evidence[i].Host, host)
		}
		if evidence[i] != (Evidence{Host: host}) {
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
		if got := parseJSONListing(answer); got != first {
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
