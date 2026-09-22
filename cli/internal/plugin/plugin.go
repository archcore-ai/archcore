// Package plugin decides what the CLI does about the Archcore plugin on each
// host that ships one. This half of the package is pure. It holds the frozen
// plugin identifiers, the per-host command table, and one planning function
// that turns observed evidence into per-host actions. It runs no command, reads
// no file, and reads no environment variable. The collector that gathers
// evidence and the executor that runs the plan live beside it and carry every
// impure concern.
//
// The failure bias is silence. A machine that never installed the plugin must
// pay no mutating host command and see no output, and that silence comes from
// the shape of the evidence, never from parsing a host's error text. Evidence
// that is missing, unreadable, or unparsed counts as "the plugin is not here".
//
// The purity is what makes the invariant of plugin-delivery.spec a test
// rather than a convention: the plugin step of `archcore update` and
// `archcore plugin update` produce identical per-host actions, because both
// call Plan with the same evidence.
package plugin

import (
	"slices"
	"strings"
)

// The three public identifiers of the Archcore plugin. Requirement 11 of
// plugin-cli-compatibility.rule freezes them: the CLI MUST NOT change them
// except in step with the archcore-ai/archcore repository, because a released CLI
// carrying a renamed identifier addresses a plugin that no longer answers to
// it. Every host command in hosts.go spells them through these constants.
const (
	RepoID        = canonicalRepoID
	MarketplaceID = "archcore-plugins"
	PluginID      = "archcore@archcore-plugins"
)

// The repository was renamed to the canonical name on 2026-09-22;
// plugin-source-migration.spec gates source edits on the canonical target, so
// hosts that still declare the legacy locator migrate on their next update.
const (
	legacyRepoID    = "archcore-ai/plugin"
	canonicalRepoID = "archcore-ai/archcore"
)

// pluginName is PluginID without its marketplace: the name Copilot lists a
// direct install under and the only one its update and uninstall accept for it.
const pluginName = "archcore"

// Host is one AI coding host with a shipping Archcore plugin. The values match
// the AgentID values of internal/agents, so a host selected for wiring and a
// host targeted for plugin delivery are named by the same string.
type Host string

const (
	HostClaudeCode Host = "claude-code"
	HostCursor     Host = "cursor"
	HostCodexCLI   Host = "codex-cli"
	HostCopilot    Host = "copilot"
)

// hostOrder is the canonical order of the four plugin hosts. Plan emits actions
// in this order whatever order the evidence arrived in, so two entry points
// that collect evidence differently still produce comparable plans.
var hostOrder = []Host{HostClaudeCode, HostCursor, HostCodexCLI, HostCopilot}

// Hosts returns the four plugin-capable hosts in a stable order. The slice is a
// copy, so a caller may sort or filter it freely.
func Hosts() []Host {
	return slices.Clone(hostOrder)
}

// agentToHost maps the agent IDs of internal/agents onto the hosts that ship a
// plugin. The other registered agents — gemini-cli, opencode, roo-code, cline —
// ship none and never map, which is what lets `--agent <id>` reject them while
// naming only the four supported hosts.
var agentToHost = map[string]Host{
	string(HostClaudeCode): HostClaudeCode,
	string(HostCursor):     HostCursor,
	string(HostCodexCLI):   HostCodexCLI,
	string(HostCopilot):    HostCopilot,
}

// HostFromAgent maps an internal/agents ID onto a plugin host. The second
// result is false for an agent with no shipping plugin and for an unknown ID.
func HostFromAgent(id string) (Host, bool) {
	h, ok := agentToHost[id]
	return h, ok
}

// Verb is the action a caller asked for. It selects the tier table Plan applies,
// and nothing else: one planner serves every entry point.
type Verb int

const (
	VerbInstall Verb = iota
	VerbUpdate
	VerbRemove
	VerbStatus
)

// String names the verb for diagnostics and test output.
func (v Verb) String() string {
	switch v {
	case VerbInstall:
		return "install"
	case VerbUpdate:
		return "update"
	case VerbRemove:
		return "remove"
	case VerbStatus:
		return "status"
	}
	return "unknown"
}

// Evidence is what was observed about one host. The collector fills it in
// production; tests construct it literally, which is what keeps Plan pure.
//
// The zero value means "nothing was observed", and every tier reads it as "the
// plugin is not installed here". A listing that failed to run, exited nonzero,
// or did not parse leaves ListingOK false, so a broken host CLI is
// indistinguishable from a host without the plugin. That is deliberate:
// requirement 1 of the update spec's failure behavior demands it.
type Evidence struct {
	Host           Host
	CLIPresent     bool   // the host binary resolved on PATH
	ListingOK      bool   // the read-only listing ran and parsed
	Listed         bool   // that listing names the plugin (meaningful only when ListingOK)
	ListedVersion  string // when the host reports one
	RegistryListed bool   // the host's on-disk registry names the plugin

	// Installs holds every installation the listing reported, ordered and bounded
	// by the collector. It is empty when the host reports one installation without
	// naming it, and the update tier then runs the host's plain update sequence.
	Installs []Install
}

// Scope is the installation scope a host reports for one installation. Only
// Claude Code reports one today; the values are its wire spelling.
type Scope string

const (
	ScopeUser    Scope = "user"
	ScopeProject Scope = "project"
	ScopeLocal   Scope = "local"
	ScopeManaged Scope = "managed"
)

// Install is one installation of the plugin a host listing reported. A host
// that installs per project lists the plugin once per project, and each of
// those needs its own update command — updating-the-plugin.spec §19.
type Install struct {
	// Name is the identity the host listed the plugin under. Copilot lists a
	// direct install as the bare plugin name and refuses the marketplace id for it.
	Name string

	// Scope is empty on a host that reports none.
	Scope Scope

	// ProjectPath is the absolute project directory a project or local
	// installation belongs to, as the host recorded it.
	ProjectPath string

	// ProjectPresent reports that ProjectPath is an existing, fully resolved
	// directory.
	ProjectPresent bool

	// Version is the version the host reported for this installation, if any.
	Version string
}

// Command is one host command line. Name is the executable as it is looked up
// on PATH; Args are passed through without a shell.
type Command struct {
	Name string
	Args []string

	// Dir is the working directory the command runs in; empty inherits the
	// process's own. Claude Code picks the project a scoped update addresses by
	// the working directory, so for that command the directory is part of what
	// it means — updating-the-plugin.spec §20.
	Dir string

	// ContinueOnFailure marks a command whose failure does not end its sequence,
	// because the commands after it address other installations and do not
	// depend on it — updating-the-plugin.spec §23.
	ContinueOnFailure bool

	// EmptyStdoutFails marks a command whose host exits zero on failure and
	// prints nothing on stdout, so an empty stdout is the only failure signal —
	// updating-the-plugin.spec §24.
	EmptyStdoutFails bool

	// Prompts marks a command that can stop for confirmation, and so takes the
	// host's non-interactive flag off a terminal. A host rejects the flag on a
	// command that never prompts: `claude plugin marketplace update -y` exits
	// with "unknown option" — verified 2026-09-16 on claude 2.1.273.
	Prompts bool
}

// String returns the exact command line. The print tier prints it, and so does
// every failure line the specs require. Arguments are not quoted: the host table
// holds none with a space, and a name taken from a listing is one whitespace-free
// field. A working directory is a user's path and is quoted.
func (c Command) String() string {
	if c.Name == "" {
		return ""
	}
	line := c.Name
	if len(c.Args) > 0 {
		line += " " + strings.Join(c.Args, " ")
	}
	if c.Dir == "" {
		return line
	}
	return "cd " + shellQuote(c.Dir) + " && " + line
}

// shellQuote quotes a path for a POSIX shell when it holds anything beyond
// characters every shell reads literally.
func shellQuote(path string) string {
	if path != "" && strings.Trim(path, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/._-+@:") == "" {
		return path
	}
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}

// ActionKind is what the executor does with one planned action.
type ActionKind int

const (
	ActionRun             ActionKind = iota // run Commands
	ActionPrintCommand                      // print the exact command for the user to run
	ActionPrintUINote                       // a host with no CLI mechanism
	ActionReportInstalled                   // an install over an already-listed plugin
	ActionReportStatus
)

// String names the kind for diagnostics and test output.
func (k ActionKind) String() string {
	switch k {
	case ActionRun:
		return "run"
	case ActionPrintCommand:
		return "print-command"
	case ActionPrintUINote:
		return "print-ui-note"
	case ActionReportInstalled:
		return "report-installed"
	case ActionReportStatus:
		return "report-status"
	}
	return "unknown"
}

// Action is one host's share of a plan. Commands is non-empty only for
// ActionRun and ActionPrintCommand; Note is non-empty only for
// ActionPrintUINote. The report kinds carry Evidence alone — the executor words
// them, because the wording differs between `archcore init` and
// `archcore plugin`, while the decision does not.
type Action struct {
	Host          Host
	Kind          ActionKind
	Commands      []Command
	Note          string
	MigrateSource bool

	// MergeAutoUpdate marks a Claude Code install whose settings entry the
	// caller writes after the action succeeds.
	MergeAutoUpdate bool

	// RemoveAutoUpdate marks a Claude Code removal whose settings entry the
	// caller takes back. It is stated rather than derived, so the executor never
	// has to reconstruct the verb it was never handed: it receives actions, not
	// the request they came from.
	RemoveAutoUpdate bool

	Evidence Evidence
}

// cloneCommands copies a command slice and every argument slice inside it. The
// host table is package state shared by every caller, so nothing that leaves
// this package may alias it: one append into a returned Args would rewrite the
// table for the rest of the process.
func cloneCommands(cmds []Command) []Command {
	if len(cmds) == 0 {
		return nil
	}
	out := make([]Command, len(cmds))
	for i, c := range cmds {
		out[i] = c
		out[i].Args = slices.Clone(c.Args)
	}
	return out
}
