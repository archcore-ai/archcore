package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"archcore-cli/internal/display"
	"archcore-cli/internal/jsonfile"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// hookEntryProbe reports whether one raw hook entry already carries command.
// A probe that cannot decode the entry returns false — foreign or malformed
// entries are never treated as archcore's and never touched.
type hookEntryProbe func(entry json.RawMessage, command string) bool

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
		if slices.ContainsFunc(entries, func(e json.RawMessage) bool {
			return spec.Probe(e, ev.Command)
		}) {
			fmt.Println(display.WarnLine(spec.alreadyInstalledLine(ev.Event)))
			continue
		}
		entryJSON, err := json.Marshal(ev.Entry)
		if err != nil {
			return fmt.Errorf("marshaling hook entry: %w", err)
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

// matcherEntryHasCommand probes {"hooks": [{"command": …}]} entries
// (Claude Code and Gemini CLI matcher shape).
func matcherEntryHasCommand(entry json.RawMessage, command string) bool {
	var m struct {
		Hooks []struct {
			Command string `json:"command"`
		} `json:"hooks"`
	}
	if json.Unmarshal(entry, &m) != nil {
		return false
	}
	for _, h := range m.Hooks {
		if h.Command == command {
			return true
		}
	}
	return false
}

// commandEntryHasCommand probes flat {"command": …} entries (Cursor).
func commandEntryHasCommand(entry json.RawMessage, command string) bool {
	var e struct {
		Command string `json:"command"`
	}
	return json.Unmarshal(entry, &e) == nil && e.Command == command
}

// bashEntryHasCommand probes flat {"bash": …} entries (Copilot).
func bashEntryHasCommand(entry json.RawMessage, command string) bool {
	var e struct {
		Bash string `json:"bash"`
	}
	return json.Unmarshal(entry, &e) == nil && e.Bash == command
}
