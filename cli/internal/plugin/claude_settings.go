package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"archcore-cli/internal/jsonfile"
)

// Claude Code refreshes a plugin on its own once the marketplace it came from
// is declared with autoUpdate enabled, so the install writes that declaration
// and the removal takes it back — plugin-delivery.spec §14 and §17.
//
// The redundancy with the deterministic update step is deliberate: the step
// covers the hosts with no host-side auto-update, and Claude Code gets both.

const (
	// extraKnownMarketplacesKey is the settings key Claude Code reads its known
	// marketplaces from.
	extraKnownMarketplacesKey = "extraKnownMarketplaces"

	// marketplaceSourceKey and marketplaceAutoUpdateKey are the two fields the
	// entry carries.
	marketplaceSourceKey     = "source"
	marketplaceAutoUpdateKey = "autoUpdate"
)

// autoUpdateOverlay is everything this surface ensures on an entry the file
// already carries: the one key requirement 14 of plugin-delivery.spec names.
//
// MergeEntryFields keeps the fields the overlay does not mention and overwrites
// every field it does, so an overlay carrying the source object would rewrite a
// source the user pointed at a fork or a local checkout — on the next install,
// silently, in a file archcore does not own. The fuller entry below is for an
// entry that does not exist yet, where there is nothing of the user's to lose.
var autoUpdateOverlay = json.RawMessage(`{"` + marketplaceAutoUpdateKey + `":true}`)

// desiredMarketplaceEntry is the entry this surface writes when the file
// declares no Archcore marketplace at all.
//
// It is a constant rather than a function returning an error: the value is a
// literal, json.Marshal over it cannot fail, and the error return only added a
// branch no input could reach. Key order is fixed here for the same reason it
// was stable before — a marshalled Go map sorts its keys.
//
// [assumption] the source object is spelled as Claude Code's GitHub marketplace
// source. plugin-delivery.spec pins the autoUpdate key and the marketplace id,
// not the schema around them; the source is written anyway, because an entry
// without one names no repository and a fresh settings file would carry a
// declaration Claude Code cannot resolve.
var desiredMarketplaceEntry = json.RawMessage(
	`{"` + marketplaceAutoUpdateKey + `":true,` +
		`"` + marketplaceSourceKey + `":{"repo":"` + RepoID + `","source":"github"}}`)

// EnsureClaudeAutoUpdate declares the archcore-plugins marketplace with
// autoUpdate enabled in settingsPath, preserving every field the file already
// carries. An entry that is already there gains the one key requirement 14 of
// plugin-delivery.spec names and keeps the rest of itself; a marketplace the
// file does not declare at all is written whole.
//
// A file that already carries the entry is not rewritten. That is what makes a
// repeated install a genuine no-op rather than a no-op that still touches the
// user's settings file and moves its modification time.
//
// backedUp reports that the settings file was not valid JSON and was moved to a
// .bak before this surface replaced it. The caller must print that: the user's
// own settings are gone from the live file, and the ADR's backup is only half
// the remedy if nothing says it happened. Every other caller of
// jsonfile.ReadOrBackup prints the same notice — @internal/agents/mcp_helpers.go
// and @internal/wiring/hooks_install.go.
func EnsureClaudeAutoUpdate(settingsPath string) (backedUp bool, err error) {
	// The write path, so an unreadable file is backed up before it is replaced —
	// backup-invalid-configs.adr with plugin-delivery.spec, Failure Behavior 4.
	doc, marketplaces, backedUp, err := readClaudeMarketplaces(settingsPath, jsonfile.ReadOrBackup)
	if err != nil {
		return false, err
	}

	existing, found := marketplaces.Get(MarketplaceID)
	if !found {
		marketplaces.Set(MarketplaceID, desiredMarketplaceEntry)
		return backedUp, saveClaudeMarketplaces(settingsPath, doc, marketplaces)
	}

	merged, changed := jsonfile.MergeEntryFields(existing, autoUpdateOverlay)
	if !changed {
		return backedUp, nil
	}
	marketplaces.Set(MarketplaceID, merged)
	return backedUp, saveClaudeMarketplaces(settingsPath, doc, marketplaces)
}

// RemoveClaudeAutoUpdate deletes only the entry this surface wrote.
//
// The extraKnownMarketplaces object stays behind even when it empties out. The
// entry is ours to take back; the object may predate us, and an empty one costs
// the user nothing.
func RemoveClaudeAutoUpdate(settingsPath string) error {
	// jsonfile.Read, not ReadOrBackup. Failure Behavior 4 of plugin-delivery.spec
	// backs a file up before a write, and a removal that finds nothing to remove
	// performs none: backing the file up here would leave a .bak of a file this
	// call never touched, and report success for a file it could not read.
	doc, marketplaces, _, err := readClaudeMarketplaces(settingsPath, readWithoutBackup)
	if err != nil {
		return err
	}
	if _, found := marketplaces.Delete(MarketplaceID); !found {
		return nil
	}
	return saveClaudeMarketplaces(settingsPath, doc, marketplaces)
}

// readWithoutBackup adapts jsonfile.Read to the reader signature below. It never
// reports a backup because it never makes one — the removal path refuses an
// invalid file rather than replacing it.
func readWithoutBackup(settingsPath string) (*jsonfile.Doc, bool, error) {
	doc, err := jsonfile.Read(settingsPath)
	return doc, false, err
}

// readClaudeMarketplaces reads the settings file and its marketplace section.
// Both are returned as ordered documents holding opaque values, so a rewrite
// keeps every unknown key and the user's key order — the surface writes one
// entry and must be invisible everywhere else.
//
// read is the corrupt-file policy the caller is entitled to: the write path
// backs an invalid file up (and says so through backedUp, which the entry point
// turns into a warning), the removal path refuses it.
func readClaudeMarketplaces(settingsPath string, read func(string) (*jsonfile.Doc, bool, error)) (doc, marketplaces *jsonfile.Doc, backedUp bool, err error) {
	doc, backedUp, err = read(settingsPath)
	if err != nil {
		return nil, nil, false, fmt.Errorf("reading Claude Code settings: %w", err)
	}
	marketplaces = jsonfile.NewDoc()
	if err := jsonfile.UnmarshalSection(doc, extraKnownMarketplacesKey, marketplaces); err != nil {
		return nil, nil, false, fmt.Errorf("reading Claude Code settings: %w", err)
	}
	return doc, marketplaces, backedUp, nil
}

// saveClaudeMarketplaces writes the marketplace section back into the settings
// document and saves it atomically.
func saveClaudeMarketplaces(settingsPath string, doc, marketplaces *jsonfile.Doc) error {
	encoded, err := json.Marshal(marketplaces)
	if err != nil {
		return fmt.Errorf("encoding %s: %w", extraKnownMarketplacesKey, err)
	}
	doc.Set(extraKnownMarketplacesKey, encoded)

	// SaveDoc writes through a temp file created beside its target, so a missing
	// parent fails the write rather than creating it. `--scope project` reaches
	// that case on the first project with no .claude/ in it — the same explicit
	// MkdirAll @internal/agents/mcp_helpers.go performs before every host config
	// write.
	dir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating the Claude Code settings directory: %w", err)
	}
	if err := jsonfile.SaveDoc(settingsPath, doc); err != nil {
		return fmt.Errorf("writing Claude Code settings: %w", err)
	}
	return nil
}
