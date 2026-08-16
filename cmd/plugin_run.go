package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"archcore-cli/internal/display"
	"archcore-cli/internal/plugin"
	"archcore-cli/internal/telemetry"
	"archcore-cli/internal/wiring"
)

// The shared core of the plugin surface. `archcore plugin`, the plugin step of
// `archcore update`, and the delivery step of `archcore init` are each a thin
// wrapper over runPluginActions: one planner decides, one executor acts, and the
// entry points differ only in the lines they print around this one —
// plugin-delivery.spec. An entry point that re-decides what the planner already
// decided breaks that invariant, however reasonable the second decision looks.

// pluginStepBudget bounds the whole plugin step. It protects the latency
// `archcore update` and `archcore init` may spend on it: once it elapses the
// remaining hosts are skipped in silence rather than delaying the command that
// hosts the step — updating-the-plugin.spec, Surface and Failure Behavior 5.
//
// A var for the reason the two bounds of @internal/plugin/exec.go are: proving
// that ONE budget covers evidence and execution together means expiring it in
// milliseconds, and no test may spend the real budget to do it.
var pluginStepBudget = 120 * time.Second

// collectPluginEvidence is the seam a test swaps to state what a machine looks
// like. Everything below it is decision and wording; the observation is the one
// part a test cannot stage from the outside.
var collectPluginEvidence = plugin.CollectEvidence

// pluginRunOptions is one plugin run: which verb, over which hosts, acting or
// printing, and where the Claude Code settings entry goes.
type pluginRunOptions struct {
	Verb      plugin.Verb
	Hosts     []plugin.Host
	PrintOnly bool

	// SettingsPath is the Claude Code settings file the marketplace entry is
	// written into. An empty value resolves to user scope, the default
	// requirement 12 of plugin-delivery.spec states.
	SettingsPath string

	// PrintUnreachedCommands hands the user the commands for a host the step
	// bound never reached. It is the entry point's decision because the two
	// specs answer that one event with different verbs: the Constraints of
	// plugin-delivery.spec fall back to printed commands for the hosts the
	// delivery step did not reach, and Failure Behavior 5 of
	// updating-the-plugin.spec prints nothing for the hosts it skipped.
	//
	// Install carries the fallback and update does not, because they are asked
	// for differently. A user who typed an install, or checked a host on the
	// selection screen, has no other way to learn the plugin never arrived; the
	// update step runs behind a command that is about something else, and a
	// maintenance step that grows output on a slow machine is exactly what the
	// bound exists to prevent.
	PrintUnreachedCommands bool
}

// pluginRunOutcome is what one run did.
type pluginRunOutcome struct {
	Results []plugin.Result

	// Failed reports that an ATTEMPTED action failed. A host the evidence left
	// unaddressed produces no action and no result, so it can never land here:
	// requirement 19 of plugin-delivery.spec keeps a host skipped for missing
	// evidence from failing a direct invocation, and requirement 18 makes an
	// attempted one that failed exit nonzero.
	Failed bool
}

// runPluginActions collects evidence, plans it, executes the plan, and performs
// the Claude Code settings step the executor leaves to its caller.
func runPluginActions(ctx context.Context, w io.Writer, opts pluginRunOptions) pluginRunOutcome {
	// One budget over BOTH phases. plugin.Execute opens a bound of the same
	// length inside itself, which makes this one look redundant — it is not:
	// CollectEvidence runs a listing per host at 30 s each before Execute starts,
	// so two sequential bounds over four hosts is 240 s against a step both specs
	// bound at 120 s.
	ctx, cancel := context.WithTimeout(ctx, pluginStepBudget)
	defer cancel()

	hosts := opts.Hosts
	if len(hosts) == 0 {
		hosts = plugin.Hosts()
	}

	actions := plugin.Plan(opts.Verb, collectPluginEvidence(ctx, hosts))
	results := plugin.Execute(ctx, actions, &pluginReporter{w: w}, plugin.ExecuteOptions{PrintOnly: opts.PrintOnly})

	outcome := pluginRunOutcome{Results: results}
	for _, res := range results {
		outcome.Failed = outcome.Failed || res.Failed
	}
	if applyClaudeAutoUpdate(w, actions, results, opts) {
		outcome.Failed = true
	}
	if opts.PrintUnreachedCommands {
		printUnreachedCommands(w, actions, results)
	}
	return outcome
}

// printUnreachedCommands hands over the commands for the hosts the step bound
// cut the run short of. Host is the correlation key the comment on
// plugin.Result names: a run cut short returns fewer results than it was given
// actions, so the actions with no result are exactly the hosts nothing was
// attempted on.
//
// It prints last, after every line about a host the run did reach, because a
// handover is what the user acts on and the lines above it are what already
// happened.
//
// An action carrying no command hands over nothing. A UI note and an
// already-installed report ask for no command line, so there is none to fall
// back to, and a host the run never reached still fails nothing —
// plugin-delivery.spec §19 keeps the exit code zero and leaves these lines as
// the only thing that tells the user the plugin did not arrive.
func printUnreachedCommands(w io.Writer, actions []plugin.Action, results []plugin.Result) {
	r := &pluginReporter{w: w}
	for _, action := range actions {
		if len(action.Commands) == 0 || pluginActionReached(results, action.Host) {
			continue
		}
		r.PrintCommand(action.Host, action.Commands)
	}
}

// pluginActionReached reports whether the executor got as far as this host. A
// result of any kind means it did; a missing one means the bound elapsed first.
func pluginActionReached(results []plugin.Result, host plugin.Host) bool {
	for _, res := range results {
		if res.Host == host {
			return true
		}
	}
	return false
}

// reportSelfCausedPluginConflict states the duplicate-hook overlap a run that
// installed or updated a plugin has just caused — plugin-delivery.spec §15.
//
// Every entry point that mutated a plugin owes this line, and owes exactly one
// of it per invocation, never one per host: the reason the notice exists is that
// the host session the user already has open now carries two sets of hooks, and
// that is equally true whichever command caused it. Removal is the verb that
// owes nothing — it ends the overlap rather than starting one — and status
// mutates nothing at all.
//
// A detection failure prints nothing and changes nothing else, which is
// requirement 5 of plugin-cli-compatibility.rule.
// DescribeSelfCausedPluginConflict already answers "" there, so the empty
// string is passed over here rather than guarded a second time.
//
// The other wording — DescribePluginConflict, for a plugin the process did not
// touch — belongs to `doctor` and `hooks install` and is not reachable from
// here.
func reportSelfCausedPluginConflict(w io.Writer, verb plugin.Verb, outcome pluginRunOutcome) (printed bool) {
	if !pluginVerbLeavesAPluginInstalled(verb) || !mutatedAPlugin(outcome) {
		return false
	}
	note := wiring.DescribeSelfCausedPluginConflict()
	if note == "" {
		return false
	}
	fmt.Fprintln(w, display.WarnLine(note))
	return true
}

// pluginVerbLeavesAPluginInstalled separates the verbs that end with a plugin on
// the machine from removal, which takes one away.
func pluginVerbLeavesAPluginInstalled(verb plugin.Verb) bool {
	return verb == plugin.VerbInstall || verb == plugin.VerbUpdate
}

// mutatedAPlugin reports whether the run changed a host. Only an executed
// command did: a printed command, a UI note and an already-installed report all
// leave the machine exactly as it was, and a failed command left nothing new to
// fire. None of them puts a second set of hooks into the open session, so none
// of them owes the restart sentence.
func mutatedAPlugin(outcome pluginRunOutcome) bool {
	for _, res := range outcome.Results {
		if res.Kind == plugin.ActionRun && !res.Failed {
			return true
		}
	}
	return false
}

// applyClaudeAutoUpdate performs the settings write Execute deliberately does
// not (see the comment on plugin.Result). The flag is read off the ACTION and
// the outcome off the Result the executor produced for that host, so the merge
// runs only where the install it belongs to completed — a marketplace declared
// for a plugin that failed to install points the host at nothing.
//
// A failed merge is reported and never undoes the install: the Result stays as
// Execute reported it (Failure Behavior 3 of plugin-delivery.spec). The run's
// own answer still turns failed, because requirement 14 makes the entry part of
// the install, and a zero exit would report an install that half happened as a
// whole one.
func applyClaudeAutoUpdate(w io.Writer, actions []plugin.Action, results []plugin.Result, opts pluginRunOptions) bool {
	// PrintOnly runs nothing at all. The settings write is a mutation of a file
	// archcore does not own, so it belongs to the run that acts, never to the one
	// that shows what acting would look like.
	if opts.PrintOnly {
		return false
	}

	failed := false
	for _, action := range actions {
		if !action.MergeAutoUpdate && !action.RemoveAutoUpdate {
			continue
		}
		if !pluginActionSucceeded(results, action.Host) {
			continue
		}
		path, err := opts.settingsPath()
		if err == nil {
			if action.MergeAutoUpdate {
				var backedUp bool
				backedUp, err = plugin.EnsureClaudeAutoUpdate(path)
				// The user's settings file was not valid JSON and has been
				// moved aside. Saying so is the other half of the remedy —
				// backup-invalid-configs.adr — because the live file now holds
				// only what this surface wrote.
				if backedUp {
					fmt.Fprintln(w, display.WarnLine(fmt.Sprintf(
						"%s: settings.json was not valid JSON — the original is saved as settings.json.bak",
						pluginHostName(action.Host))))
				}
			} else {
				err = plugin.RemoveClaudeAutoUpdate(path)
			}
		}
		if err != nil {
			fmt.Fprintln(w, display.FailLine(fmt.Sprintf("%s: %v", pluginHostName(action.Host), err)))
			failed = true
		}
	}
	return failed
}

// pluginActionSucceeded reports whether one host's action completed. Host is the
// correlation key because a run cut short by the step bound returns fewer
// results than it was given actions, and a host that was never reached must not
// have its settings written on the strength of a plan alone.
func pluginActionSucceeded(results []plugin.Result, host plugin.Host) bool {
	for _, res := range results {
		if res.Host == host {
			return !res.Failed
		}
	}
	return false
}

// settingsPath resolves where this run writes the marketplace entry.
func (o pluginRunOptions) settingsPath() (string, error) {
	if o.SettingsPath != "" {
		return o.SettingsPath, nil
	}
	return claudeSettingsPath("", false)
}

// claudeSettingsPath returns the Claude Code settings file the marketplace
// entry belongs in: the user's own by default, and the project's under
// `--scope project` — plugin-delivery.spec §12 and §13.
//
// The path is spelled here rather than taken from the hook path map of
// internal/wiring. They name the same file today and answer different
// questions: a change to where this host's hooks are written must not silently
// move the marketplace entry with them.
func claudeSettingsPath(projectRoot string, projectScope bool) (string, error) {
	if projectScope {
		if projectRoot == "" {
			return "", errors.New("--scope project needs a project root")
		}
		return filepath.Join(projectRoot, ".claude", "settings.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating the home directory for the Claude Code settings file: %w", err)
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// pluginHostsForAgent maps an `--agent` selection onto the hosts a run targets.
// An empty id targets every host that ships a plugin.
//
// The error names the four plugin hosts and not the agent registry: gemini-cli,
// opencode, roo-code and cline are valid agents for wiring and simply ship no
// plugin, so listing them would offer names this surface cannot act on — a
// Constraint of plugin-delivery.spec.
func pluginHostsForAgent(agentID string) ([]plugin.Host, error) {
	if agentID == "" {
		return plugin.Hosts(), nil
	}
	host, ok := plugin.HostFromAgent(agentID)
	if !ok {
		return nil, fmt.Errorf("no Archcore plugin ships for %q — the plugin hosts are %s", agentID, pluginHostNames())
	}
	return []plugin.Host{host}, nil
}

// pluginHostNames lists the plugin hosts in the canonical order, for the one
// error that names them.
func pluginHostNames() string {
	hosts := plugin.Hosts()
	names := make([]string, 0, len(hosts))
	for _, h := range hosts {
		names = append(names, string(h))
	}
	return strings.Join(names, ", ")
}

// pluginRunningInCI reports whether a human is present to consent to a mutating
// host command. `archcore init` would otherwise run the delivery step unattended
// on a CI runner, and requirement 8 of plugin-delivery.spec answers a CI
// environment with printed commands and nothing executed.
//
// The question is this file's own; the set of variables that answers it is not.
// internal/update/policy.go asks a third question off the same set — whether to
// replace this binary at all — and all three once carried a private copy of the
// list, so adding a CI provider took three edits with nothing to catch a missed
// one. telemetry.CIVars is the single declaration; the decisions stay here.
func pluginRunningInCI() bool { return telemetry.DetectedCI() }

// pluginHostName is a host's name in an output line. A host outside the command
// table never reaches one, so its id is a sufficient last resort.
func pluginHostName(h plugin.Host) string {
	if spec, ok := plugin.SpecFor(h); ok {
		return spec.DisplayName
	}
	return string(h)
}

// pluginReporter is the one Reporter implementation the surface has. Execute
// words nothing, and these per-action lines are the same wherever the run came
// from; the lines around them are the entry point's, and it prints those itself.
type pluginReporter struct{ w io.Writer }

// Progress names the host before its command starts —
// updating-the-plugin.spec §11. The line carries the command that actually
// runs, non-interactive flag included, because that is what Execute reports.
func (r *pluginReporter) Progress(host plugin.Host, c plugin.Command) {
	fmt.Fprintln(r.w, display.Dim.Render(fmt.Sprintf("  %s: %s", pluginHostName(host), c)))
}

// CommandFailed prints the exact command line, so the user can rerun it and see
// the host's own error — updating-the-plugin.spec §12 with plugin-delivery.spec,
// Failure Behavior 2.
func (r *pluginReporter) CommandFailed(host plugin.Host, c plugin.Command) {
	fmt.Fprintln(r.w, display.FailLine(fmt.Sprintf("%s: this command failed: %s", pluginHostName(host), c)))
}

// PrintCommand hands over the commands for a host this process cannot reach, or
// for a run that prints instead of acting — plugin-delivery.spec, Failure
// Behavior 1 and 5.
func (r *pluginReporter) PrintCommand(host plugin.Host, cs []plugin.Command) {
	what := "this command"
	if len(cs) > 1 {
		what = "these commands"
	}
	fmt.Fprintln(r.w, display.WarnLine(fmt.Sprintf("%s: run %s yourself:", pluginHostName(host), what)))
	for _, c := range cs {
		fmt.Fprintln(r.w, display.HintLine(c.String()))
	}
}

// UINote states the instruction for a host with no CLI mechanism.
func (r *pluginReporter) UINote(host plugin.Host, note string) {
	fmt.Fprintln(r.w, display.WarnLine(fmt.Sprintf("%s: %s", pluginHostName(host), note)))
}

// AlreadyInstalled reports the no-op an install over a present plugin is. It is
// what keeps a rerun of `archcore init` from nagging.
func (r *pluginReporter) AlreadyInstalled(host plugin.Host) {
	fmt.Fprintln(r.w, display.CheckLine(fmt.Sprintf("%s: the Archcore plugin is already installed", pluginHostName(host))))
}

// Status reports one host's evidence.
func (r *pluginReporter) Status(host plugin.Host, ev plugin.Evidence) {
	fmt.Fprintln(r.w, display.KeyValue(pluginHostName(host), pluginStatusText(ev)))
}

// pluginStatusText words one host's evidence for `archcore plugin status`: the
// presence answer, which evidence produced it, and the version when the host
// reported one — plugin-delivery.spec §16.
//
// A host CLI that answered nothing reports "unknown" rather than "not
// installed". The collector collapses a failed listing into the same shape as an
// absent plugin on purpose, because the update tiers must treat them alike; a
// status report is the one place that collapse would mislead, and the CLI being
// present is what tells the two apart here.
func pluginStatusText(ev plugin.Evidence) string {
	switch {
	case ev.CLIPresent && ev.ListingOK && ev.Listed:
		if ev.ListedVersion != "" {
			return "installed " + ev.ListedVersion + " (the host reports it)"
		}
		return "installed (the host reports it)"
	case ev.CLIPresent && ev.ListingOK:
		return "not installed (the host reports it)"
	case ev.CLIPresent:
		return "unknown — the host listing did not answer"
	case ev.RegistryListed:
		return "installed (on-disk registry; the host CLI is not on PATH)"
	}
	return "not installed"
}
