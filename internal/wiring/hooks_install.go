package wiring

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
type hookEntryProbe func(entry, want json.RawMessage, ev hookEventInstall) hookEntryClass

// jsonEqual compares two encoded values by structure rather than by bytes. The
// stored entry has been through MarshalIndent, so its whitespace and key order
// differ from a freshly marshaled one even when nothing about it changed.
func jsonEqual(a, b json.RawMessage) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

// overlayEntry merges want onto existing and returns the result. Every field
// archcore writes takes want's value; every field it does not write — a host's
// own "timeout", a key an newer archcore added — keeps its value and its
// position. Objects merge by key, arrays merge element-wise (a shorter want
// truncates, which is how a duplicated inner hook is dropped), and anything else
// is replaced outright.
//
// Replacing a stale entry wholesale is what this exists to avoid. An entry is
// classified stale whenever it differs from want in any way, so a single extra
// field made `init`, `hooks install`, and `doctor --fix` delete real user data
// from a file the user had edited by hand — contradicting this writer's own
// contract above and backup-invalid-configs.adr.
//
// The trade it accepts: a field archcore used to write and no longer does is
// preserved rather than cleaned up. Keeping a stray key is the cheaper mistake.
func overlayEntry(existing, want json.RawMessage) json.RawMessage {
	if ex, exOK := decodeJSONObject(existing); exOK {
		if wa, waOK := decodeJSONObject(want); waOK {
			for pair := wa.Oldest(); pair != nil; pair = pair.Next() {
				if cur, ok := ex.Get(pair.Key); ok {
					ex.Set(pair.Key, overlayEntry(cur, pair.Value))
					continue
				}
				ex.Set(pair.Key, pair.Value)
			}
			merged, err := json.Marshal(ex)
			if err != nil {
				return want
			}
			return merged
		}
		return want
	}

	var exArr, waArr []json.RawMessage
	if json.Unmarshal(existing, &exArr) == nil && json.Unmarshal(want, &waArr) == nil {
		out := make([]json.RawMessage, 0, len(waArr))
		for i, w := range waArr {
			if i < len(exArr) {
				out = append(out, overlayEntry(exArr[i], w))
				continue
			}
			out = append(out, w)
		}
		merged, err := json.Marshal(out)
		if err != nil {
			return want
		}
		return merged
	}

	return want
}

// decodeJSONObject decodes raw as an order-preserving object. A non-object (an
// array, a scalar, null) reports false — json.Unmarshal into an OrderedMap
// accepts null and yields an empty map, which would silently turn a null entry
// into "{}".
func decodeJSONObject(raw json.RawMessage) (*jsonfile.Doc, bool) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{' {
		return nil, false
	}
	doc := jsonfile.NewDoc()
	if json.Unmarshal(raw, doc) != nil {
		return nil, false
	}
	return doc, true
}

// hookEventInstall is one event archcore installs into an agent's hook config.
type hookEventInstall struct {
	Event   string // event key in the hooks section, e.g. "SessionStart"
	Command string // idempotency key probed against existing entries
	Matcher string // expected matcher; "" when the event takes none
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

	// One archcore entry per event is an invariant of this writer: the ownership
	// probe answers "is this ours?" per entry, so two of our entries under one
	// key would each classify the other as a stale duplicate and the second
	// would silently overwrite the first.
	seenEvents := make(map[string]bool, len(spec.Events))
	for _, ev := range spec.Events {
		if seenEvents[ev.Event] {
			return fmt.Errorf("hook spec declares event %q twice; one archcore entry per event", ev.Event)
		}
		seenEvents[ev.Event] = true
	}

	for _, ev := range spec.Events {
		entries, _ := hooks.Get(ev.Event)

		entryJSON, err := json.Marshal(ev.Entry)
		if err != nil {
			return fmt.Errorf("marshaling hook entry: %w", err)
		}
		want := json.RawMessage(entryJSON)

		// Classify every existing entry once: is the exact command already
		// present, and where do stale archcore-owned entries sit?
		current := false
		var staleIdx []int
		for i, e := range entries {
			switch spec.Probe(e, want, ev) {
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

		if len(staleIdx) > 0 {
			// Archcore-owned entry with an outdated command: merge our fields
			// into the first in place, drop any further stale duplicates.
			merged := overlayEntry(entries[staleIdx[0]], want)
			if len(staleIdx) == 1 && jsonEqual(merged, entries[staleIdx[0]]) {
				// Only unknown fields set this entry apart, and the merge left
				// them alone — there is nothing to carry to the host, so the
				// file is not touched at all.
				fmt.Println(display.WarnLine(spec.alreadyInstalledLine(ev.Event)))
				continue
			}
			entries[staleIdx[0]] = merged
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
func matcherEntryHasCommand(entry, want json.RawMessage, ev hookEventInstall) hookEntryClass {
	if jsonEqual(entry, want) {
		return entryCurrent
	}
	var m struct {
		Matcher string `json:"matcher"`
		Hooks   []struct {
			Command string `json:"command"`
		} `json:"hooks"`
	}
	if json.Unmarshal(entry, &m) != nil || len(m.Hooks) == 0 {
		return entryForeign
	}
	allArchcore := true
	for _, h := range m.Hooks {
		if !isArchcoreHookCommand(h.Command) {
			allArchcore = false
			break
		}
	}
	if allArchcore {
		// Entirely ours but not what we would write now — the command or the
		// matcher has moved on, or the user set a field we do not write.
		// overlayEntry carries our fields to the host and leaves theirs alone;
		// classifying by "differs from want" alone is why the merge, and not a
		// replace, has to be what installHookEvents applies.
		return entryStaleArchcore
	}
	// A mixed entry: someone hand-merged their own hook beside ours. It cannot
	// be replaced without dropping theirs, so if it already carries our command
	// under our matcher it counts as current. Classifying it otherwise appends a
	// second entry and the host runs the hook twice.
	for _, h := range m.Hooks {
		if h.Command == ev.Command && m.Matcher == ev.Matcher {
			return entryCurrent
		}
	}
	return entryForeign
}

// commandEntryHasCommand probes flat {"command": …} entries (Cursor). A flat
// entry carries exactly one command, so there is no mixed case: it is either
// byte-for-byte what we would write, ours but outdated, or someone else's.
func commandEntryHasCommand(entry, want json.RawMessage, _ hookEventInstall) hookEntryClass {
	var e struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(entry, &e) != nil {
		return entryForeign
	}
	switch {
	case jsonEqual(entry, want):
		return entryCurrent
	case isArchcoreHookCommand(e.Command):
		return entryStaleArchcore
	default:
		return entryForeign
	}
}

// bashEntryHasCommand probes flat {"bash": …} entries (Copilot).
func bashEntryHasCommand(entry, want json.RawMessage, _ hookEventInstall) hookEntryClass {
	var e struct {
		Bash string `json:"bash"`
	}
	if json.Unmarshal(entry, &e) != nil {
		return entryForeign
	}
	switch {
	case jsonEqual(entry, want):
		return entryCurrent
	case isArchcoreHookCommand(e.Bash):
		return entryStaleArchcore
	default:
		return entryForeign
	}
}
