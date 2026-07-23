package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"archcore-cli/internal/display"
	"archcore-cli/internal/jsonfile"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// archcoreHookMarker identifies an archcore-owned hook entry regardless of
// the exact command string a given CLI version installed. Every hook command
// archcore has ever written STARTS with this prefix ("archcore hooks
// claude-code session-start", "archcore hooks cursor session-start", …), so
// probes use it to recognize a stale entry from an older or newer CLI and
// update it in place instead of appending a duplicate. Prefix (not substring)
// matching keeps user-wrapped commands — `sh -c 'archcore hooks … 2>&1'` —
// classified foreign and therefore untouched.
const archcoreHookMarker = "archcore hooks "

// isArchcoreHookCommand reports whether command is an archcore-written hook
// invocation (see archcoreHookMarker).
func isArchcoreHookCommand(command string) bool {
	return strings.HasPrefix(command, archcoreHookMarker)
}

// hookEntryClass classifies an existing raw hook entry with respect to the
// command archcore wants installed.
type hookEntryClass int

const (
	// entryForeign — not archcore's (or undecodable). Never touched.
	entryForeign hookEntryClass = iota
	// entryCurrent — already carries the exact command. Nothing to do.
	entryCurrent
	// entryStaleArchcore — archcore-owned (carries the marker) but with an
	// outdated command string. Updated in place, never duplicated.
	entryStaleArchcore
)

// hookEntryProbe classifies one raw hook entry. A probe that cannot decode
// the entry returns entryForeign — foreign or malformed entries are never
// treated as archcore's and never touched.
type hookEntryProbe func(entry json.RawMessage, command string) hookEntryClass

// hookEventInstall is one event archcore installs into an agent's hook config.
type hookEventInstall struct {
	Event   string // event key in the hooks section, e.g. "SessionStart"
	Command string // idempotency key probed against existing entries
	Entry   any    // marshaled and appended when Command is absent
}

// hookInstallSpec drives installHookEvents for one agent's config file.
type hookInstallSpec struct {
	Label         string // display prefix ("Cursor"); empty for Claude Code
	Path          string // config file path
	EnsureVersion bool   // add "version": 1 when the key is absent (cursor/copilot)
	Probe         hookEntryProbe
	Events        []hookEventInstall
}

func (s hookInstallSpec) alreadyInstalledLine(event string) string {
	if s.Label == "" {
		return fmt.Sprintf("Already installed: %s", event)
	}
	return fmt.Sprintf("%s: already installed: %s", s.Label, event)
}

func (s hookInstallSpec) installedLine(event string) string {
	if s.Label == "" {
		return fmt.Sprintf("Installed hook: %s", event)
	}
	return fmt.Sprintf("%s: installed hook: %s", s.Label, event)
}

func (s hookInstallSpec) updatedLine(event string) string {
	if s.Label == "" {
		return fmt.Sprintf("Updated hook: %s", event)
	}
	return fmt.Sprintf("%s: updated hook: %s", s.Label, event)
}

// installHookEvents installs the spec's hook entries into the JSON config at
// spec.Path. Everything archcore does not own is held as opaque RawMessage —
// unknown top-level keys, foreign events, and unknown fields on existing hook
// entries all survive a rewrite. The file is written only when something
// actually changed (a second run is a no-op with a byte-identical file), the
// write is atomic (tmp+rename), and a corrupted file is backed up as .bak
// before starting fresh — aborting if the backup itself cannot be written.
func installHookEvents(spec hookInstallSpec) error {
	if err := os.MkdirAll(filepath.Dir(spec.Path), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(spec.Path), err)
	}

	doc, backedUp, err := jsonfile.ReadOrBackup(spec.Path)
	if err != nil {
		return err
	}
	if backedUp {
		fmt.Println(display.WarnLine(fmt.Sprintf("Corrupted %s backed up, starting fresh", spec.Path)))
	}

	hooks := orderedmap.New[string, []json.RawMessage]()
	if err := jsonfile.UnmarshalSection(doc, "hooks", hooks); err != nil {
		return err
	}

	changed := false
	if spec.EnsureVersion {
		if _, ok := doc.Get("version"); !ok {
			doc.Set("version", json.RawMessage("1"))
			changed = true
		}
	}

	for _, ev := range spec.Events {
		entries, _ := hooks.Get(ev.Event)

		// Classify every existing entry once: is the exact command already
		// present, and where do stale archcore-owned entries sit?
		current := false
		var staleIdx []int
		for i, e := range entries {
			switch spec.Probe(e, ev.Command) {
			case entryCurrent:
				current = true
			case entryStaleArchcore:
				staleIdx = append(staleIdx, i)
			}
		}

		if current && len(staleIdx) == 0 {
			fmt.Println(display.WarnLine(spec.alreadyInstalledLine(ev.Event)))
			continue
		}

		if current {
			// Exact entry present plus stale leftovers (e.g. duplicates from a
			// pre-marker CLI): drop the stale ones, keep everything else.
			hooks.Set(ev.Event, deleteIndices(entries, staleIdx))
			changed = true
			fmt.Println(display.CheckLine(spec.updatedLine(ev.Event)))
			continue
		}

		entryJSON, err := json.Marshal(ev.Entry)
		if err != nil {
			return fmt.Errorf("marshaling hook entry: %w", err)
		}

		if len(staleIdx) > 0 {
			// Archcore-owned entry with an outdated command: update the first
			// in place, drop any further stale duplicates.
			entries[staleIdx[0]] = json.RawMessage(entryJSON)
			hooks.Set(ev.Event, deleteIndices(entries, staleIdx[1:]))
			changed = true
			fmt.Println(display.CheckLine(spec.updatedLine(ev.Event)))
			continue
		}

		hooks.Set(ev.Event, append(entries, json.RawMessage(entryJSON)))
		changed = true
		fmt.Println(display.CheckLine(spec.installedLine(ev.Event)))
	}

	if !changed {
		return nil
	}

	hooksJSON, err := json.Marshal(hooks)
	if err != nil {
		return fmt.Errorf("marshaling hooks section: %w", err)
	}
	doc.Set("hooks", json.RawMessage(hooksJSON))

	return jsonfile.SaveDoc(spec.Path, doc)
}

// deleteIndices returns entries with the given (ascending) indices removed.
// An empty index list returns entries unchanged.
func deleteIndices(entries []json.RawMessage, idx []int) []json.RawMessage {
	if len(idx) == 0 {
		return entries
	}
	kept := make([]json.RawMessage, 0, len(entries)-len(idx))
	for i, e := range entries {
		if len(idx) > 0 && i == idx[0] {
			idx = idx[1:]
			continue
		}
		kept = append(kept, e)
	}
	return kept
}

// matcherEntryHasCommand probes {"hooks": [{"command": …}]} entries
// (Claude Code and Gemini CLI matcher shape). The entry is stale-archcore
// only when EVERY inner hook carries the archcore marker — a hand-merged
// entry mixing archcore and foreign hooks is left alone (classified foreign)
// so an update never drops someone else's hook.
func matcherEntryHasCommand(entry json.RawMessage, command string) hookEntryClass {
	var m struct {
		Hooks []struct {
			Command string `json:"command"`
		} `json:"hooks"`
	}
	if json.Unmarshal(entry, &m) != nil || len(m.Hooks) == 0 {
		return entryForeign
	}
	allArchcore := true
	for _, h := range m.Hooks {
		if h.Command == command {
			return entryCurrent
		}
		if !isArchcoreHookCommand(h.Command) {
			allArchcore = false
		}
	}
	if allArchcore {
		return entryStaleArchcore
	}
	return entryForeign
}

// commandEntryHasCommand probes flat {"command": …} entries (Cursor).
func commandEntryHasCommand(entry json.RawMessage, command string) hookEntryClass {
	var e struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(entry, &e) != nil {
		return entryForeign
	}
	switch {
	case e.Command == command:
		return entryCurrent
	case isArchcoreHookCommand(e.Command):
		return entryStaleArchcore
	default:
		return entryForeign
	}
}

// bashEntryHasCommand probes flat {"bash": …} entries (Copilot).
func bashEntryHasCommand(entry json.RawMessage, command string) hookEntryClass {
	var e struct {
		Bash string `json:"bash"`
	}
	if json.Unmarshal(entry, &e) != nil {
		return entryForeign
	}
	switch {
	case e.Bash == command:
		return entryCurrent
	case isArchcoreHookCommand(e.Bash):
		return entryStaleArchcore
	default:
		return entryForeign
	}
}
