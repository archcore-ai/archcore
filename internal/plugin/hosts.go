package plugin

// The per-host command table. Every command line was probed live on 2026-08-15
// against claude 2.1.232, copilot 1.0.76, and codex 0.147.0, and matches the
// Surface tables of updating-the-plugin.spec and plugin-delivery.spec.
// The table is data on purpose: correcting a host is a one-line edit here, not
// a change in the planner.
//
// Entries the specs tag [assumption] are marked inline, so the first live
// verification is a cheap edit rather than a hunt.
//
// OpenCode, Roo Code, Cline, and Gemini CLI ship no plugin and have no row.

const (
	// pluginDirName is the directory a host checks the plugin out as, below the
	// marketplace checkout: cache/<marketplace>/<plugin>. Verified live on
	// 2026-08-16 for Claude Code (~/.claude/plugins/cache/archcore-plugins/
	// archcore), Cursor, and Codex CLI.
	pluginDirName = "archcore"

	// copilotPluginDir is the directory Copilot creates for the install spec
	// RepoID + ":plugins/archcore". Copilot flattens the spec into one name
	// rather than nesting it, so it shares nothing with the other three hosts.
	// Verified live on 2026-08-16 at
	// ~/.copilot/installed-plugins/_direct/archcore-ai--plugin--plugins-archcore.
	copilotPluginDir = "archcore-ai--plugin--plugins-archcore"
)

// UINotes carries the one-line instruction printed for a host with no CLI
// mechanism. Cursor is the only such host today: it manages plugins from its
// UI, so every verb it takes part in resolves to a printed sentence.
type UINotes struct {
	Install string
	Update  string
	Remove  string
}

// HostSpec is one row of the host command table. The planner reads the command
// slices; the collector reads Listing, ListingJSON, and RegistryPath; the
// executor reads NonInteractiveFlag and MergeAutoUpdate.
type HostSpec struct {
	// Host names the row.
	Host Host

	// DisplayName is the host's name in output lines.
	DisplayName string

	// CLI is the executable resolved on PATH. An empty CLI means the host has no
	// command-line plugin mechanism at all, and every verb resolves to a UI note.
	CLI string

	// Listing is the read-only command that reports the host's installed
	// plugins. It runs before any mutating command, and it is the only evidence
	// that authorizes one.
	Listing Command

	// ListingJSON reports whether Listing answers JSON rather than plain text.
	ListingJSON bool

	// RegistryPath is the host's on-disk plugin registry, relative to the user's
	// home directory. The collector reads it only when the host CLI is absent.
	RegistryPath string

	// RegistryEntry is the entry name under RegistryPath that means the plugin
	// itself is installed, compared with the extension trimmed and case folded.
	//
	// It is a per-host field rather than one shared predicate because the hosts
	// disagree: three of them check out the plugin as a directory called
	// "archcore" below the marketplace, and Copilot names its directory after
	// the install spec instead. Matching the marketplace id here would report
	// the plugin installed on a machine that only ever ran `marketplace add`.
	RegistryEntry string

	// Update, Install, and Remove are the mutating command sequences, run in
	// order. An empty sequence belongs to a host with no CLI mechanism.
	Update  []Command
	Install []Command
	Remove  []Command

	// NonInteractiveFlag is appended by the executor to a mutating command when
	// the session has no TTY. It lives here as one field, so a host that needs a
	// different flag is a one-line edit and never a concatenation at the call
	// site.
	NonInteractiveFlag string

	// MergeAutoUpdate marks the host whose successful install is followed by the
	// autoUpdate marketplace entry merge. Claude Code is the only one.
	MergeAutoUpdate bool

	// MachineScoped marks a host whose plugin store has no repository scope, so
	// installing changes the machine rather than this project. The selection
	// screen discloses that difference — plugin-delivery.spec §2.
	//
	// It lives in the table with every other per-host fact. Spelled as a host
	// comparison at the call site instead, a fifth machine-scoped host would get
	// the project-scoped disclosure with nothing to catch it.
	MachineScoped bool

	// Notes carries the printed instruction for a host with no CLI mechanism.
	Notes UINotes
}

// hasCLI reports whether the host has any command-line plugin mechanism. A host
// without one never runs and never prints a command, whatever the evidence
// says about a binary on PATH.
func (s HostSpec) hasCLI() bool {
	return s.CLI != ""
}

var hostTable = map[Host]HostSpec{
	HostClaudeCode: {
		Host:          HostClaudeCode,
		DisplayName:   "Claude Code",
		CLI:           "claude",
		Listing:       Command{Name: "claude", Args: []string{"plugin", "list", "--json"}},
		ListingJSON:   true,
		RegistryPath:  ".claude/plugins",
		RegistryEntry: pluginDirName,
		Update: []Command{
			{Name: "claude", Args: []string{"plugin", "marketplace", "update", MarketplaceID}},
			{Name: "claude", Args: []string{"plugin", "update", PluginID}},
		},
		Install: []Command{
			{Name: "claude", Args: []string{"plugin", "marketplace", "add", RepoID}},
			{Name: "claude", Args: []string{"plugin", "install", PluginID}},
		},
		// [assumption] plugin-delivery.spec names the Claude Code uninstall a
		// "Claude Code equivalent" of the verified Copilot and Codex forms. The
		// marketplace registration is deliberately left in place: removal undoes
		// the plugin and the autoUpdate entry this surface wrote, not the
		// marketplace another plugin may still use.
		Remove: []Command{
			{Name: "claude", Args: []string{"plugin", "uninstall", PluginID}},
		},
		// [assumption] `-y` for non-TTY safety, per the update spec's Surface table.
		NonInteractiveFlag: "-y",
		MergeAutoUpdate:    true,
	},

	HostCursor: {
		Host:        HostCursor,
		DisplayName: "Cursor",
		// Cursor manages plugins from its UI only (cursor.com/docs/plugins). It
		// has no CLI, so it has no listing and no command of any kind.
		RegistryPath:  ".cursor/plugins",
		RegistryEntry: pluginDirName,
		Notes: UINotes{
			Install: "Install the Archcore plugin from the Cursor plugin marketplace, or run /add-plugin in Cursor.",
			Update:  "Update the Archcore plugin from the Cursor plugin marketplace.",
			Remove:  "Remove the Archcore plugin from the Cursor plugin marketplace.",
		},
	},

	HostCodexCLI: {
		Host:        HostCodexCLI,
		DisplayName: "Codex CLI",
		CLI:         "codex",
		Listing:     Command{Name: "codex", Args: []string{"plugin", "list", "--json"}},
		ListingJSON: true,
		// The update spec names "~/.codex marketplace snapshots" as the registry
		// evidence. Verified live on 2026-08-16: Codex mirrors Claude Code at
		// ~/.codex/plugins/cache/<marketplace>/<plugin>. The root is the plugins
		// subdirectory, not ~/.codex — that directory also holds sessions/,
		// cache/, log/ and .tmp/, which would spend the scan budget before
		// plugins/ was reached and match the marketplace snapshot under
		// .tmp/marketplaces/ on the way.
		RegistryPath:  ".codex/plugins",
		RegistryEntry: pluginDirName,
		// Codex has no per-plugin update. Refreshing the marketplace snapshot is
		// the update.
		Update: []Command{
			{Name: "codex", Args: []string{"plugin", "marketplace", "upgrade", MarketplaceID}},
		},
		Install: []Command{
			{Name: "codex", Args: []string{"plugin", "marketplace", "add", RepoID}},
			{Name: "codex", Args: []string{"plugin", "add", PluginID}},
		},
		Remove: []Command{
			{Name: "codex", Args: []string{"plugin", "remove", PluginID}},
		},
		// Codex registers its marketplace under ~/.codex and takes no project
		// scope.
		MachineScoped: true,
	},

	HostCopilot: {
		Host:        HostCopilot,
		DisplayName: "GitHub Copilot",
		CLI:         "copilot",
		Listing:     Command{Name: "copilot", Args: []string{"plugin", "list"}},
		ListingJSON: false,
		// The update spec marked ~/.copilot/installed-plugins unverified; the path
		// is confirmed as of 2026-08-16. The binary is often absent from PATH on a
		// VS Code-managed install, so this row reaches the print tier more often
		// than the others, which makes the entry name below load-bearing.
		RegistryPath:  ".copilot/installed-plugins",
		RegistryEntry: copilotPluginDir,
		Update: []Command{
			{Name: "copilot", Args: []string{"plugin", "update", PluginID}},
		},
		// [assumption] plugin-delivery.spec marks the plugins/archcore subpath
		// unverified until the first live install.
		Install: []Command{
			{Name: "copilot", Args: []string{"plugin", "install", RepoID + ":plugins/archcore"}},
		},
		// Copilot's uninstall takes the bare plugin name, not the marketplace-
		// qualified id. plugin-delivery.spec records the form verbatim.
		Remove: []Command{
			{Name: "copilot", Args: []string{"plugin", "uninstall", "archcore"}},
		},
		// Copilot installs into its user-level store, outside any repository.
		MachineScoped: true,
	},
}

// SpecFor returns the command-table row for a host. The second result is false
// for a host that ships no plugin. Command slices in the result are copies, so
// a caller cannot rewrite the table.
func SpecFor(h Host) (HostSpec, bool) {
	spec, ok := hostTable[h]
	if !ok {
		return HostSpec{}, false
	}
	return copySpec(spec), true
}

// Specs returns every row of the command table in the canonical host order.
func Specs() []HostSpec {
	out := make([]HostSpec, 0, len(hostOrder))
	for _, h := range hostOrder {
		if spec, ok := hostTable[h]; ok {
			out = append(out, copySpec(spec))
		}
	}
	return out
}

// copySpec deep-copies the command slices of a row so the shared table stays
// immutable outside this file.
func copySpec(spec HostSpec) HostSpec {
	spec.Listing = Command{Name: spec.Listing.Name, Args: cloneArgs(spec.Listing.Args)}
	spec.Update = cloneCommands(spec.Update)
	spec.Install = cloneCommands(spec.Install)
	spec.Remove = cloneCommands(spec.Remove)
	return spec
}

// cloneArgs copies an argument slice, keeping nil as nil so an absent listing
// compares equal to the zero Command.
func cloneArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, len(args))
	copy(out, args)
	return out
}

// commandsFor returns the mutating sequence a verb runs on a host.
func (s HostSpec) commandsFor(verb Verb) []Command {
	switch verb {
	case VerbInstall:
		return s.Install
	case VerbUpdate:
		return s.Update
	case VerbRemove:
		return s.Remove
	case VerbStatus:
		// Status reads evidence and runs nothing.
		return nil
	}
	return nil
}

// noteFor returns the UI instruction a verb prints on a host with no CLI
// mechanism.
func (s HostSpec) noteFor(verb Verb) string {
	switch verb {
	case VerbInstall:
		return s.Notes.Install
	case VerbUpdate:
		return s.Notes.Update
	case VerbRemove:
		return s.Notes.Remove
	case VerbStatus:
		// Status reports what is already there and prints no instruction.
		return ""
	}
	return ""
}
