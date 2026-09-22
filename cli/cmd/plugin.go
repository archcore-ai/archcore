package cmd

import (
	"fmt"
	"io"
	"os"

	"archcore-cli/internal/display"
	"archcore-cli/internal/plugin"
	"archcore-cli/internal/wiring"

	"github.com/spf13/cobra"
)

// `archcore plugin` is one of the three entry points over the shared core in
// @cmd/plugin_run.go. It decides nothing the planner decides: it names a verb,
// names the hosts, and words the lines around the run — plugin-delivery.spec.
//
// What separates it from the init step is consent. A typed verb IS the consent
// for every host it targets (§10), so this surface never sets PrintOnly and
// never asks whether a terminal or a CI runner is present: §11 requires a
// non-interactive invocation to run exactly as an interactive one. The two
// questions `archcore init` must ask before it mutates a host are already
// answered here by the word the user typed.
//
// Output goes to cmd.OutOrStdout(), not to os.Stdout. That is the deviation the
// display convention allows for a command whose output is one self-contained
// report a caller may redirect: every line of a verb belongs to that verb, so a
// caller that redirects the command must see all of them or none. `archcore
// update` states the same reason. The commands that print through os.Stdout —
// init, doctor, hooks, status — interleave their lines with helpers that write
// to the process handle, and there redirecting the cobra writer would capture
// only half the run.

// pluginScope selects the Claude Code settings file the marketplace entry is
// written into, and nothing else — plugin-delivery.spec §12 and §13.
//
// A typed alias because the set is closed and finite: untyped string constants
// let a third spelling reach pluginInstallTargetFor as an ordinary string and
// fall out the bottom as an error at run time, where the compiler could have
// said so.
type pluginScope string

const (
	pluginScopeUser    pluginScope = "user"
	pluginScopeProject pluginScope = "project"
)

// pluginFlags is the host selection every verb shares.
//
// project is read only by pluginInstallTargetFor under `--scope project`, so
// only `archcore plugin install` declares the flag. The other three verbs write
// no repository file at all, and a flag they accepted and ignored would answer
// a user who passed it with silence instead of an unknown-flag error —
// cmd/project_root_flag_test.go lists them as rootless for the same reason.
type pluginFlags struct {
	agent   string
	project string
}

func newPluginCmd() *cobra.Command {
	f := &pluginFlags{}

	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage the Archcore plugin on the hosts that ship one",
		Long: "Install, update, remove, or report the Archcore plugin on the hosts that ship one: " +
			"Claude Code, Cursor, Codex CLI, and GitHub Copilot.\n\n" +
			"A typed verb is the consent for every host it targets, so each verb acts the same way " +
			"with or without a terminal.",
		// A bare `archcore plugin` is a question and answers with help. NoArgs is
		// what turns a mistyped verb into cobra's unknown-command error: without
		// it, `archcore plugin instal` would print help and exit zero, reporting
		// success for a command that did nothing.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	bindPluginFlags(cmd, f)
	bindPluginProjectFlag(cmd, f)

	cmd.AddCommand(
		newPluginInstallCmd(f),
		newPluginVerbCmd(f, "update", "Update the Archcore plugin on the hosts that have it", plugin.VerbUpdate),
		// Removal undoes what this surface wrote and runs the host's own
		// uninstall wherever the host CLI is present — plugin-delivery.spec §17.
		// Both halves are the planner's and the shared core's; the verb only
		// names them.
		newPluginVerbCmd(f, "remove", "Remove the Archcore plugin from the hosts that have it", plugin.VerbRemove),
		newPluginStatusCmd(f),
	)
	return cmd
}

// bindPluginFlags declares the host-selection flags on one command.
//
// Every verb carries its own copy, because cobra parses only the flag set of
// the command it resolved. The group carries one too, so `archcore plugin
// --help` lists them and a flag typed with no verb at all is not an unknown
// flag. One variable stands behind both declarations, so the two spellings are
// one invocation rather than two that can drift apart.
func bindPluginFlags(cmd *cobra.Command, f *pluginFlags) {
	cmd.Flags().StringVar(&f.agent, "agent", "",
		"act on one host (claude-code, cursor, codex-cli, copilot); default: every host that ships a plugin")
}

// bindPluginProjectFlag declares --project on the one command that reads it.
// The group carries it too, so `archcore plugin --project X install` parses:
// cobra consumes a parent's flag value before it resolves the subcommand.
func bindPluginProjectFlag(cmd *cobra.Command, f *pluginFlags) {
	cmd.Flags().StringVar(&f.project, "project", "",
		"project root containing .archcore/ under --scope project (default: current directory; env: ARCHCORE_PROJECT_ROOT)")
}

// newPluginVerbCmd builds a verb whose whole input is the host selection.
func newPluginVerbCmd(f *pluginFlags, use, short string, verb plugin.Verb) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		// Hosts are selected with --agent, never as a positional. A stray word
		// would otherwise be ignored and the verb would silently act on every
		// host instead of the one named.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			hosts, err := pluginHostsForAgent(f.agent)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			outcome := runPluginActions(cmd.Context(), out, pluginRunOptions{
				Verb:  verb,
				Hosts: hosts,
			})
			// Update owes the overlap notice and removal does not. The helper
			// reads that off the verb, so the two commands this constructor
			// builds stay one line of code rather than two that can drift.
			//
			// Neither verb hands over the commands for a host the step bound cut
			// off: Failure Behavior 5 of updating-the-plugin.spec prints nothing
			// for the hosts an update skipped, and a removal that did not happen
			// leaves the machine as the user already has it.
			reportSelfCausedPluginConflict(out, verb, outcome)
			return pluginExit(outcome)
		},
	}
	bindPluginFlags(cmd, f)
	return cmd
}

func newPluginInstallCmd(f *pluginFlags) *cobra.Command {
	// Declared without a value: cobra's StringVar below writes the default, and
	// a second one here would be dead the moment the two disagreed.
	var scope string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the Archcore plugin on the selected hosts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			hosts, err := pluginHostsForAgent(f.agent)
			if err != nil {
				return err
			}
			target, err := pluginInstallTargetFor(f, scope)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			outcome := runPluginActions(cmd.Context(), out, pluginRunOptions{
				Verb:         plugin.VerbInstall,
				Hosts:        hosts,
				SettingsPath: target.SettingsPath,
				// A typed install the step bound cut short falls back to the
				// printed commands for the hosts it never reached — the
				// Constraints of plugin-delivery.spec. The user asked for the
				// plugin on those hosts by name, and dropping them in silence
				// leaves nothing to say it never got there.
				PrintUnreachedCommands: true,
			})
			target.reportReach(out, outcome)
			reportSelfCausedPluginConflict(out, plugin.VerbInstall, outcome)
			return pluginExit(outcome)
		},
	}
	bindPluginFlags(cmd, f)
	bindPluginProjectFlag(cmd, f)
	cmd.Flags().StringVar(&scope, "scope", string(pluginScopeUser),
		"which Claude Code settings file gains the marketplace entry: 'user' for the user's own, 'project' for the repository's")
	return cmd
}

func newPluginStatusCmd(f *pluginFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Report the Archcore plugin state on each host",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			hosts, err := pluginHostsForAgent(f.agent)
			if err != nil {
				return err
			}
			// The outcome is dropped on purpose. Status runs no mutating command,
			// so it has no attempted action to fail, and §20 makes the report the
			// whole contract: a host that answered nothing is a row in it, never
			// an exit code.
			runPluginActions(cmd.Context(), cmd.OutOrStdout(), pluginRunOptions{
				Verb:  plugin.VerbStatus,
				Hosts: hosts,
			})
			return nil
		},
	}
	bindPluginFlags(cmd, f)
	return cmd
}

// pluginExit turns one run into this command's exit contract. An attempted
// action that failed exits nonzero with its details already printed
// (plugin-delivery.spec §18); a host skipped for missing evidence exits zero
// (§19), and needs nothing here to do so — it produced no action, so the
// outcome never carries it.
func pluginExit(outcome pluginRunOutcome) error {
	if outcome.Failed {
		return ErrAlreadyReported
	}
	return nil
}

// pluginInstallTarget is where an install writes the Claude Code marketplace
// entry.
type pluginInstallTarget struct {
	// SettingsPath is empty under user scope, which the shared core resolves to
	// the user's own settings file — plugin-delivery.spec §12.
	SettingsPath string

	// ProjectRoot is set only under `--scope project`, where it is what the
	// printed path is shown relative to.
	ProjectRoot string
}

// pluginInstallTargetFor resolves the scope selection into a settings file.
//
// A user-scope install resolves no project root at all. resolveProjectRoot
// refuses a working directory inside a host's plugin install cache
// (host-cwd-misrouting.adr), and an install that writes nothing into the
// repository must not fail on a directory it never reads — the Constraint of
// plugin-delivery.spec puts the resolved root behind the `--scope project`
// write alone.
func pluginInstallTargetFor(f *pluginFlags, scope string) (pluginInstallTarget, error) {
	switch pluginScope(scope) {
	case pluginScopeUser:
		return pluginInstallTarget{}, nil
	case pluginScopeProject:
		root, err := resolveProjectRoot(f.project, os.Getenv("ARCHCORE_PROJECT_ROOT"))
		if err != nil {
			return pluginInstallTarget{}, err
		}
		path, err := claudeSettingsPath(root, true)
		if err != nil {
			return pluginInstallTarget{}, err
		}
		return pluginInstallTarget{SettingsPath: path, ProjectRoot: root}, nil
	}
	return pluginInstallTarget{}, fmt.Errorf(
		"unknown --scope %q — the scopes are %q and %q", scope, pluginScopeUser, pluginScopeProject)
}

// reportReach states what a repository-scoped entry reaches —
// plugin-delivery.spec §13. It follows the run rather than preceding it,
// because a preamble states an intention and this sentence is about a file that
// now exists.
//
// The write itself belongs to the shared core, which performs it for the one
// host that carries the marketplace entry and only after that host's action
// completed. The two facts read off the outcome here are the same two: a Claude
// Code result that did not fail, of one of the kinds the planner attaches the
// entry to. A printed command writes no file, and a note about a file nobody
// wrote would be this entry point inventing an answer of its own.
//
// One case over-reports: a settings merge that failed leaves the host's result
// as it was, so the note still prints. It prints under the failure line the
// core already wrote, which is where the user reads what actually happened.
func (t pluginInstallTarget) reportReach(w io.Writer, outcome pluginRunOutcome) {
	if t.ProjectRoot == "" {
		return
	}
	for _, res := range outcome.Results {
		if res.Host != plugin.HostClaudeCode || res.Failed {
			continue
		}
		if res.Kind != plugin.ActionRun && res.Kind != plugin.ActionReportInstalled {
			continue
		}
		fmt.Fprintln(w, display.WarnLine(fmt.Sprintf(
			"%s is committed with the repository, so this marketplace declaration reaches every teammate who checks it out.",
			wiring.DisplayPath(t.ProjectRoot, t.SettingsPath))))
		return
	}
}
