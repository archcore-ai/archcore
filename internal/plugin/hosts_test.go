package plugin

import (
	"slices"
	"strings"
	"testing"
)

// commandLines renders a command sequence for comparison against literals.
func commandLines(cmds []Command) []string {
	if len(cmds) == 0 {
		return nil
	}
	lines := make([]string, len(cmds))
	for i, c := range cmds {
		lines[i] = c.String()
	}
	return lines
}

// equalLines was a hand-rolled slices.Equal; execute_test.go in this same
// package already imports slices for exactly this comparison.

// TestSpecForPinsTheCommandTable states every host row as literal command
// lines. The lines were probed live on 2026-08-15 against claude 2.1.232,
// copilot 1.0.76, and codex 0.147.0, and the specs repeat them verbatim. A
// change here is a change to what the CLI runs on a user's machine.
func TestSpecForPinsTheCommandTable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		host            Host
		wantCLI         string
		wantDisplay     string
		wantListing     string
		wantListingJSON bool
		wantRegistry    string
		wantRegEntry    string
		wantUpdate      []string
		wantInstall     []string
		wantRemove      []string
		wantFlag        string
		wantMerge       bool
		wantNotes       bool
	}{
		{
			name:            "claude code",
			host:            HostClaudeCode,
			wantCLI:         "claude",
			wantDisplay:     "Claude Code",
			wantListing:     "claude plugin list --json",
			wantListingJSON: true,
			wantRegistry:    ".claude/plugins",
			wantRegEntry:    "archcore",
			wantUpdate: []string{
				"claude plugin marketplace update archcore-plugins",
				"claude plugin update archcore@archcore-plugins",
			},
			wantInstall: []string{
				"claude plugin marketplace add archcore-ai/plugin",
				"claude plugin install archcore@archcore-plugins",
			},
			wantRemove: []string{"claude plugin uninstall archcore@archcore-plugins"},
			wantFlag:   "-y",
			wantMerge:  true,
		},
		{
			name:         "cursor has no cli mechanism",
			host:         HostCursor,
			wantDisplay:  "Cursor",
			wantRegistry: ".cursor/plugins",
			wantRegEntry: "archcore",
			wantNotes:    true,
		},
		{
			name:            "codex cli",
			host:            HostCodexCLI,
			wantCLI:         "codex",
			wantDisplay:     "Codex CLI",
			wantListing:     "codex plugin list --json",
			wantListingJSON: true,
			wantRegistry:    ".codex/plugins",
			wantRegEntry:    "archcore",
			wantUpdate:      []string{"codex plugin marketplace upgrade archcore-plugins"},
			wantInstall: []string{
				"codex plugin marketplace add archcore-ai/plugin",
				"codex plugin add archcore@archcore-plugins",
			},
			wantRemove: []string{"codex plugin remove archcore@archcore-plugins"},
		},
		{
			name:         "github copilot",
			host:         HostCopilot,
			wantCLI:      "copilot",
			wantDisplay:  "GitHub Copilot",
			wantListing:  "copilot plugin list",
			wantRegistry: ".copilot/installed-plugins",
			wantRegEntry: "archcore-ai--plugin--plugins-archcore",
			wantUpdate:   []string{"copilot plugin update archcore@archcore-plugins"},
			wantInstall:  []string{"copilot plugin install archcore-ai/plugin:plugins/archcore"},
			wantRemove:   []string{"copilot plugin uninstall archcore"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, ok := SpecFor(tt.host)
			if !ok {
				t.Fatalf("SpecFor(%q) reported no row", tt.host)
			}
			if spec.Host != tt.host {
				t.Errorf("Host = %q, want %q", spec.Host, tt.host)
			}
			if spec.CLI != tt.wantCLI {
				t.Errorf("CLI = %q, want %q", spec.CLI, tt.wantCLI)
			}
			if spec.DisplayName != tt.wantDisplay {
				t.Errorf("DisplayName = %q, want %q", spec.DisplayName, tt.wantDisplay)
			}
			if got := spec.Listing.String(); got != tt.wantListing {
				t.Errorf("Listing = %q, want %q", got, tt.wantListing)
			}
			if spec.ListingJSON != tt.wantListingJSON {
				t.Errorf("ListingJSON = %v, want %v", spec.ListingJSON, tt.wantListingJSON)
			}
			if spec.RegistryPath != tt.wantRegistry {
				t.Errorf("RegistryPath = %q, want %q", spec.RegistryPath, tt.wantRegistry)
			}
			if spec.RegistryEntry != tt.wantRegEntry {
				t.Errorf("RegistryEntry = %q, want %q", spec.RegistryEntry, tt.wantRegEntry)
			}
			if got := commandLines(spec.Update); !slices.Equal(got, tt.wantUpdate) {
				t.Errorf("Update = %q, want %q", got, tt.wantUpdate)
			}
			if got := commandLines(spec.Install); !slices.Equal(got, tt.wantInstall) {
				t.Errorf("Install = %q, want %q", got, tt.wantInstall)
			}
			if got := commandLines(spec.Remove); !slices.Equal(got, tt.wantRemove) {
				t.Errorf("Remove = %q, want %q", got, tt.wantRemove)
			}
			if spec.NonInteractiveFlag != tt.wantFlag {
				t.Errorf("NonInteractiveFlag = %q, want %q", spec.NonInteractiveFlag, tt.wantFlag)
			}
			if spec.MergeAutoUpdate != tt.wantMerge {
				t.Errorf("MergeAutoUpdate = %v, want %v", spec.MergeAutoUpdate, tt.wantMerge)
			}
			hasNotes := spec.Notes.Install != "" && spec.Notes.Update != "" && spec.Notes.Remove != ""
			if hasNotes != tt.wantNotes {
				t.Errorf("all UI notes present = %v, want %v", hasNotes, tt.wantNotes)
			}
			if spec.hasCLI() != (tt.wantCLI != "") {
				t.Errorf("hasCLI() = %v, want %v", spec.hasCLI(), tt.wantCLI != "")
			}
		})
	}
}

// TestSpecForRejectsHostsWithoutAPlugin keeps the table to the four hosts the
// specs name. The other registered agents ship no plugin.
func TestSpecForRejectsHostsWithoutAPlugin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		host Host
	}{
		{name: "gemini cli", host: "gemini-cli"},
		{name: "opencode", host: "opencode"},
		{name: "roo code", host: "roo-code"},
		{name: "cline", host: "cline"},
		{name: "empty host", host: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, ok := SpecFor(tt.host)
			if ok {
				t.Fatalf("SpecFor(%q) returned a row, want none", tt.host)
			}
			if spec.Host != "" {
				t.Errorf("SpecFor(%q) returned %+v, want the zero spec", tt.host, spec)
			}
		})
	}
}

func TestSpecsFollowTheCanonicalHostOrder(t *testing.T) {
	t.Parallel()
	specs := Specs()
	hosts := Hosts()
	if len(specs) != len(hosts) {
		t.Fatalf("Specs() returned %d rows, want %d", len(specs), len(hosts))
	}
	for i, spec := range specs {
		if spec.Host != hosts[i] {
			t.Errorf("Specs()[%d].Host = %q, want %q", i, spec.Host, hosts[i])
		}
	}
}

// TestSpecCommandsAreCopies proves the shared table survives a caller that
// rewrites what it was handed. The table is process-wide state, so an aliased
// argument slice would rewrite what every later caller runs.
func TestSpecCommandsAreCopies(t *testing.T) {
	t.Parallel()
	first, ok := SpecFor(HostClaudeCode)
	if !ok {
		t.Fatal("SpecFor(claude-code) reported no row")
	}
	first.Update[0].Name = "rewritten"
	first.Update[0].Args[3] = "rewritten"
	first.Install[1].Args[2] = "rewritten"
	first.Listing.Args[0] = "rewritten"

	second, ok := SpecFor(HostClaudeCode)
	if !ok {
		t.Fatal("SpecFor(claude-code) reported no row on the second call")
	}
	if got := second.Update[0].String(); got != "claude plugin marketplace update archcore-plugins" {
		t.Errorf("Update[0] = %q after a caller rewrote an earlier copy", got)
	}
	if got := second.Install[1].String(); got != "claude plugin install archcore@archcore-plugins" {
		t.Errorf("Install[1] = %q after a caller rewrote an earlier copy", got)
	}
	if got := second.Listing.String(); got != "claude plugin list --json" {
		t.Errorf("Listing = %q after a caller rewrote an earlier copy", got)
	}
}

// TestTableCarriesNoIdentifierVariant scans every argument of every command for
// a token that mentions archcore. Each one must be a frozen identifier, the
// Copilot uninstall's bare plugin name, or the Copilot install's repository
// subpath. The scan is what catches a near-miss spelling — archcore-plugin,
// archcore_ai/plugin, archcore@archcore-plugin — that a targeted assertion on
// one command would walk straight past.
func TestTableCarriesNoIdentifierVariant(t *testing.T) {
	t.Parallel()
	allowed := map[string]bool{
		RepoID:                       true,
		MarketplaceID:                true,
		PluginID:                     true,
		"archcore":                   true, // copilot plugin uninstall takes the bare name
		RepoID + ":plugins/archcore": true, // copilot install subpath [assumption]
	}
	for _, spec := range Specs() {
		sequences := map[string][]Command{
			"listing": {spec.Listing},
			"update":  spec.Update,
			"install": spec.Install,
			"remove":  spec.Remove,
		}
		for name, cmds := range sequences {
			for _, cmd := range cmds {
				for _, arg := range cmd.Args {
					if !strings.Contains(arg, "archcore") {
						continue
					}
					if !allowed[arg] {
						t.Errorf("%s %s: argument %q is not a frozen identifier", spec.Host, name, arg)
					}
				}
				if strings.Contains(cmd.Name, "archcore") {
					t.Errorf("%s %s: executable %q mentions the plugin identifiers", spec.Host, name, cmd.Name)
				}
			}
		}
	}
}

func TestCommandsForAndNoteFor(t *testing.T) {
	t.Parallel()
	claude, _ := SpecFor(HostClaudeCode)
	cursor, _ := SpecFor(HostCursor)

	tests := []struct {
		name      string
		spec      HostSpec
		verb      Verb
		wantCmds  []string
		wantEmpty bool // the note must be empty
	}{
		{name: "claude install", spec: claude, verb: VerbInstall, wantCmds: commandLines(claude.Install), wantEmpty: true},
		{name: "claude update", spec: claude, verb: VerbUpdate, wantCmds: commandLines(claude.Update), wantEmpty: true},
		{name: "claude remove", spec: claude, verb: VerbRemove, wantCmds: commandLines(claude.Remove), wantEmpty: true},
		{name: "claude status has no commands", spec: claude, verb: VerbStatus, wantEmpty: true},
		{name: "cursor install has a note only", spec: cursor, verb: VerbInstall},
		{name: "cursor update has a note only", spec: cursor, verb: VerbUpdate},
		{name: "cursor remove has a note only", spec: cursor, verb: VerbRemove},
		{name: "cursor status has neither", spec: cursor, verb: VerbStatus, wantEmpty: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commandLines(tt.spec.commandsFor(tt.verb)); !slices.Equal(got, tt.wantCmds) {
				t.Errorf("commandsFor(%s) = %q, want %q", tt.verb, got, tt.wantCmds)
			}
			note := tt.spec.noteFor(tt.verb)
			if tt.wantEmpty && note != "" {
				t.Errorf("noteFor(%s) = %q, want an empty note", tt.verb, note)
			}
			if !tt.wantEmpty && note == "" {
				t.Errorf("noteFor(%s) is empty, want an instruction", tt.verb)
			}
		})
	}
}
