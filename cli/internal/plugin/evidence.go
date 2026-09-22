package plugin

import (
	"cmp"
	"context"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// CollectEvidence observes each host and returns what Plan needs to decide.
// Hosts that ship no plugin are dropped, so a caller may pass its whole agent
// selection without filtering it first.
//
// Per host, in order: resolve the binary on PATH; when it is there, ask the
// host its own read-only listing; then read the host's on-disk registry. The
// host's own answer comes first because it is the only evidence that authorizes
// a mutating command — updating-the-plugin.spec §5.
//
// Every failure lands as absent evidence. A listing that could not start, exited
// nonzero, printed more than the cap, or did not parse leaves ListingOK false,
// and the planner then reads the host as not listed. Nothing here inspects a
// host's error text: the silence a machine without the plugin gets has to come
// from the shape of the evidence, because error wording is another project's to
// change — updating-the-plugin.spec, Failure Behavior 1.
func CollectEvidence(ctx context.Context, hosts []Host) []Evidence {
	out := make([]Evidence, 0, len(hosts))
	for _, host := range hosts {
		spec, ok := SpecFor(host)
		if !ok {
			continue
		}
		ev := Evidence{Host: host}
		if spec.hasCLI() {
			if _, err := lookPath(spec.CLI); err == nil {
				ev.CLIPresent = true
				answer := readListing(ctx, spec)
				ev.ListingOK = answer.ok
				ev.Listed = answer.listed
				ev.ListedVersion = answer.version
				ev.Installs = locateProjects(answer.installs)
			}
		}
		ev.RegistryListed = registryListsPlugin(spec)
		out = append(out, ev)
	}
	return out
}

// listing is one host's answer to its read-only plugin listing. The zero value
// is "the host did not answer", which is what every failure collapses to.
type listing struct {
	ok       bool
	listed   bool
	version  string
	installs []Install
}

// locateProjects records whether each installation's project directory exists.
//
// Guard, not advisory: the answer decides whether a scoped update runs, and
// Claude Code resolves its working directory before matching it, so a command
// started anywhere but the resolved project updates a different one — verified
// 2026-09-16. A path that is relative, missing, unreadable, or not fully resolved
// therefore reads as absent — fail-open-or-fail-closed-reads.rule §6.
func locateProjects(installs []Install) []Install {
	for i, install := range installs {
		if install.ProjectPath == "" || !filepath.IsAbs(install.ProjectPath) {
			continue
		}
		resolved, err := filepath.EvalSymlinks(install.ProjectPath)
		if err != nil || resolved != install.ProjectPath {
			continue
		}
		info, err := os.Stat(resolved)
		installs[i].ProjectPresent = err == nil && info.IsDir()
	}
	return installs
}

// readListing runs a host's listing command and parses the result. A truncated
// answer counts as no answer: parsing a prefix would let a cap that fired
// halfway through the entries report the plugin as absent, and absence is the
// verdict that silently skips the host.
//
// Guard, not advisory: this answer is the only evidence that authorizes a
// mutating host command, so an unreadable or ambiguous listing refuses it
// rather than degrading into a presence claim —
// fail-open-or-fail-closed-reads.rule §2.
func readListing(ctx context.Context, spec HostSpec) listing {
	if spec.Listing.Name == "" {
		return listing{}
	}
	out := runCommand(ctx, spec.Listing)
	if out.Failed || out.Truncated {
		return listing{}
	}
	if spec.ListingJSON {
		return parseJSONListing(out.Stdout)
	}
	return parseTextListing(out.Stdout)
}

const (
	// maxListingDepth bounds how deep the listing walk descends. It protects the
	// step budget against a host answer that nests without end, and it is deep
	// enough for the envelopes the hosts use today: a top-level object, a
	// marketplace map, an entry array, and the entry itself.
	maxListingDepth = 8

	// maxListingNodes bounds how many values one listing walk visits. It protects
	// the same budget against a host with a very large answer, where the cost of
	// walking every entry buys nothing: the walk stops at the first match, so the
	// budget is only ever spent proving absence.
	maxListingNodes = 5000

	// maxListedVersion bounds the version token that leaves this package in
	// Evidence.ListedVersion. It protects the terminal a status report is printed
	// to and the context a report reaching a model would cost: the token is
	// another program's stdout, which this process does not size, and the output
	// cap alone would let a megabyte through as one field. A token past the
	// ceiling is dropped rather than shortened, because a shortened version reads
	// like a real one — bounded-and-deterministic-output.rule.
	maxListedVersion = 64
)

// listingIdentityKeys are the object fields a host may name a plugin in. The
// set is deliberately wide, because the listing shape is unverified on every
// host — see parseJSONListing.
var listingIdentityKeys = map[string]bool{
	"id": true, "name": true, "plugin": true, "pluginid": true,
	"plugin_id": true, "fullname": true, "slug": true,
}

// listingVersionKeys are the object fields a host may report a version in.
var listingVersionKeys = map[string]bool{
	"version": true, "pluginversion": true, "plugin_version": true,
}

// listingInstalledKeys are the object fields a host may report installation
// state in. A host that carries one answers the question this package asks
// directly; a host that omits it leaves the answer to the listing command,
// which enumerates installed plugins only — updating-the-plugin.spec, Surface.
var listingInstalledKeys = map[string]bool{
	"installed": true, "isinstalled": true, "is_installed": true,
}

// listingScopeKeys and listingProjectKeys are the object fields a host may
// report an installation's scope and project directory in.
var (
	listingScopeKeys   = map[string]bool{"scope": true}
	listingProjectKeys = map[string]bool{"projectpath": true, "project_path": true}
)

// listingVersionRe matches the version token a plain-text listing prints beside
// a plugin name. Extraction is best-effort: requirement 16 of
// plugin-delivery.spec asks for the version "when the host reports one", so a
// host that prints none simply leaves ListedVersion empty.
var listingVersionRe = regexp.MustCompile(`^v?\d+\.\d+(\.\d+)?[0-9A-Za-z.\-+]*$`)

// parseJSONListing finds the Archcore plugin inside a host's JSON listing.
//
// [assumption] the envelope each host wraps its entries in is unverified: the
// spec pins the command, not the schema. The walk is therefore tolerant rather
// than pinned to one shape — a parser bound to today's schema would read a
// changed answer as "not installed", and that verdict is the silent one, so the
// miss would never surface as an error. It is bounded and ordered instead:
// object keys are visited in sorted order, so two walks over one answer collect
// the same installations.
func parseJSONListing(stdout string) listing {
	var decoded any
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		return listing{}
	}
	var installs []Install
	budget := maxListingNodes
	findInstalls(decoded, maxListingDepth, &budget, &installs)
	return listingOf(installs)
}

// listingOf is a parsed answer built from the installations it named. The
// plugin is listed exactly when one installation was found, so presence and the
// installations the update addresses are never two readings of one answer.
func listingOf(installs []Install) listing {
	if len(installs) == 0 {
		return listing{ok: true}
	}
	ordered := orderInstalls(installs)
	return listing{ok: true, listed: true, version: boundedVersion(ordered[0].Version), installs: ordered}
}

// findInstalls collects every installation a decoded listing reports. Claude
// Code lists the plugin once per scope and project, so the walk does not stop at
// the first entry.
//
// Keys name a plugin as readily as values do — a host may answer
// `{"archcore@archcore-plugins": {…}}`, where nothing inside the entry repeats
// the id. A walk that read values alone would answer "not installed" there, and
// that verdict is silent — updating-the-plugin.spec §7.
func findInstalls(value any, depth int, budget *int, out *[]Install) {
	if depth <= 0 || *budget <= 0 {
		return
	}
	*budget--

	switch node := value.(type) {
	case map[string]any:
		if name, named := entryPluginName(node); named {
			// The host has answered for this entry, so the walk stops here either
			// way: descending into a rejected entry would read its id again as a
			// bare string, with no installation flag beside it.
			if !entryReportsUninstalled(node) {
				*out = append(*out, installFromEntry(name, node))
			}
			return
		}
		for _, key := range slices.Sorted(maps.Keys(node)) {
			if identifiesPlugin(key) {
				installsUnderKey(key, node[key], depth-1, budget, out)
				continue
			}
			findInstalls(node[key], depth-1, budget, out)
		}
	case []any:
		for _, item := range node {
			findInstalls(item, depth-1, budget, out)
		}
	case string:
		// A host that answers a bare list of plugin ids reports no version.
		if identifiesPlugin(node) {
			*out = append(*out, Install{Name: node})
		}
	}
}

// installsUnderKey reads the value of a key that names the plugin. The key is
// the identity when the entries below it do not repeat one, as in
// `{"archcore@archcore-plugins": [{"scope": "user"}, …]}`.
func installsUnderKey(key string, value any, depth int, budget *int, out *[]Install) {
	switch node := value.(type) {
	case map[string]any:
		if _, named := entryPluginName(node); named {
			findInstalls(node, depth, budget, out)
			return
		}
		if !entryReportsUninstalled(node) {
			*out = append(*out, installFromEntry(key, node))
		}
	case []any:
		for _, item := range node {
			if entry, isEntry := item.(map[string]any); isEntry {
				installsUnderKey(key, entry, depth, budget, out)
			}
		}
	default:
		*out = append(*out, Install{Name: key})
	}
}

// installFromEntry reads one installation out of a listing entry.
func installFromEntry(name string, entry map[string]any) Install {
	install := Install{Name: name}
	for _, key := range slices.Sorted(maps.Keys(entry)) {
		text, isText := entry[key].(string)
		if !isText {
			continue
		}
		switch lowered := strings.ToLower(key); {
		case listingScopeKeys[lowered]:
			install.Scope = Scope(strings.ToLower(text))
		case listingProjectKeys[lowered]:
			install.ProjectPath = text
		case listingVersionKeys[lowered] && install.Version == "":
			install.Version = text
		}
	}
	return install
}

// scopeRank orders installations so the machine-wide one runs first.
var scopeRank = map[Scope]int{ScopeUser: 0, ScopeProject: 1, ScopeLocal: 2, ScopeManaged: 3}

// scopeOrder ranks a scope for sorting. A host that reports no scope has one
// installation for the whole machine, so it ranks with user scope; an
// unrecognized scope sorts last.
func scopeOrder(s Scope) int {
	if s == "" {
		return scopeRank[ScopeUser]
	}
	if r, known := scopeRank[s]; known {
		return r
	}
	return len(scopeRank)
}

// orderInstalls sorts and deduplicates installations on every field, so the
// same listing always yields the same sequence. The planner cuts it, after it
// drops the installations it must not address — bounded-and-deterministic-output.rule §3.
func orderInstalls(installs []Install) []Install {
	sorted := slices.Clone(installs)
	slices.SortFunc(sorted, func(a, b Install) int {
		return cmp.Or(
			cmp.Compare(scopeOrder(a.Scope), scopeOrder(b.Scope)),
			cmp.Compare(a.Scope, b.Scope),
			cmp.Compare(a.ProjectPath, b.ProjectPath),
			cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.Version, b.Version),
		)
	})
	return slices.Compact(sorted)
}

// boundedVersion returns the token a host reported as its version, or nothing
// when the token is longer than any version is. Requirement 16 of
// plugin-delivery.spec asks for the version "when the host reports one", and a
// host that hands back more than maxListedVersion bytes reported something
// else.
func boundedVersion(version string) string {
	if len(version) > maxListedVersion {
		return ""
	}
	return version
}

// identifiesPlugin reports whether a name in a listing names the plugin itself.
//
// It is tighter than namesPlugin, which answers true for the marketplace id as
// well. A host that groups its entries under `{"archcore-plugins": {…}}` names
// a registered marketplace, and a registered marketplace with nothing installed
// under it is the opposite of an installation: reading it as presence would
// report the plugin installed on a machine that only ever added the
// marketplace, which install then treats as a no-op and never installs.
//
// Every place a listing name is read goes through here — object keys, identity
// values, and the bare strings of a host that answers a plain array of ids. The
// line was drawn for keys alone once, which left the same marketplace id
// counting as an installation when a host carried it in a value instead.
// registryNamesPlugin is the on-disk twin.
func identifiesPlugin(name string) bool {
	return namesPlugin(name) && strings.ToLower(name) != MarketplaceID
}

// entryPluginName returns the identity value an entry names the plugin by. It
// answers the identity question alone; whether that entry is an installation is
// entryReportsUninstalled's answer, and the two are separate because a host
// names the plugin in the very entry that says it is not installed.
func entryPluginName(entry map[string]any) (string, bool) {
	for _, key := range slices.Sorted(maps.Keys(entry)) {
		text, isText := entry[key].(string)
		if isText && listingIdentityKeys[strings.ToLower(key)] && identifiesPlugin(text) {
			return text, true
		}
	}
	return "", false
}

// entryReportsUninstalled reports whether a listing entry carries the host's own
// statement that this plugin is not installed.
//
// The listing is a guard (see readListing), so the affirmative reading is the
// narrow one: a host that answers anything other than a true installation flag
// has not confirmed the plugin, and the planner then skips that host in silence
// — fail-open-or-fail-closed-reads.rule §2.
//
// An entry with no such field is not a refusal. Three hosts list what is
// installed and nothing else, and Codex prints its uninstalled marketplace
// entries only under the `--available` flag this package never passes, so the
// absent field means "the command already answered that" rather than "unknown".
func entryReportsUninstalled(entry map[string]any) bool {
	for _, key := range slices.Sorted(maps.Keys(entry)) {
		if listingInstalledKeys[strings.ToLower(key)] {
			return !installedFlagIsTrue(entry[key])
		}
	}
	return false
}

// installedFlagIsTrue reads one host's installation flag. A bool is the shape
// every observed host uses; a string is read too, because the flag arrives as
// JSON another project writes and a quoted boolean costs nothing to accept.
func installedFlagIsTrue(value any) bool {
	switch flag := value.(type) {
	case bool:
		return flag
	case string:
		return strings.EqualFold(flag, "true")
	}
	return false
}

// parseTextListing finds the Archcore plugin inside a host's plain-text
// listing. A listing that ran and printed nothing recognizable is a parsed
// answer of "not installed", not a failure: the host was asked and replied.
func parseTextListing(stdout string) listing {
	var installs []Install
	for line := range strings.Lines(stdout) {
		fields := strings.Fields(line)
		if index := slices.IndexFunc(fields, identifiesPlugin); index >= 0 {
			installs = append(installs, Install{Name: fields[index], Version: versionInFields(fields)})
		}
	}
	return listingOf(installs)
}

// versionInFields returns the first version-shaped token on a listing line.
func versionInFields(fields []string) string {
	for _, field := range fields {
		if listingVersionRe.MatchString(field) {
			return field
		}
	}
	return ""
}

// namesPlugin reports whether a name — a registry entry, a listing field, a
// plugin id — names the Archcore plugin.
//
// The match is tight on purpose. A bare "archcore" substring also appears in
// paths a host keys by project directory, and ~/.codex holds one entry per
// project, so a substring test would read a checkout of this repository as an
// installed plugin. The marketplace id is the discriminating token; the plugin
// id contains it, so matching the marketplace id matches both.
func namesPlugin(name string) bool {
	lowered := strings.ToLower(name)
	if strings.Contains(lowered, MarketplaceID) {
		return true
	}
	// A bare entry names the plugin alone: a directory called archcore, or a
	// marketplace snapshot file called archcore.json. The extension is trimmed so
	// both forms answer the same.
	return strings.TrimSuffix(lowered, filepath.Ext(lowered)) == "archcore"
}
