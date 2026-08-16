package plugin

import (
	"context"
	"os"
	"slices"
)

// The one executor. Every entry point — the plugin step of `archcore update`,
// the delivery step of `archcore init`, and each verb of `archcore plugin` —
// runs its plan through this function, which is the other half of the invariant
// of plugin-delivery.spec: one planner decides, one executor acts, and the
// entry points differ only in wording.

// Reporter receives every line the executor would otherwise print. Execute
// writes to no stream at all, because the wording is the entry point's to
// choose: `archcore init` reports a delivery inside a setup summary, and
// `archcore plugin status` reports the same evidence as a report of its own.
// The decision is shared; the sentences are not.
type Reporter interface {
	// Progress announces one command before it runs — updating-the-plugin.spec
	// §11. The command is the one that actually runs, non-interactive flag
	// included.
	Progress(host Host, c Command)

	// CommandFailed reports a nonzero exit or a timeout with the exact command
	// that produced it — updating-the-plugin.spec §12.
	CommandFailed(host Host, c Command)

	// PrintCommand hands the user the commands to run for a host this process
	// cannot reach, or for a run that prints instead of acting.
	PrintCommand(host Host, cs []Command)

	// UINote states the one-line instruction for a host with no CLI mechanism.
	UINote(host Host, note string)

	// AlreadyInstalled reports the no-op an install over a present plugin is.
	AlreadyInstalled(host Host)

	// Status reports the evidence gathered for one host.
	Status(host Host, ev Evidence)
}

// Result is what happened to one host. Host is the correlation key: the caller
// matches a Result back to the Action it planned by host, because a run cut
// short by the step bound returns fewer results than it was given actions.
//
// The settings merge is not performed here. Failure behavior 3 of
// plugin-delivery.spec requires a failed merge to be reported, and Execute
// words nothing; the entry point holds the settings path, reads MergeAutoUpdate
// or RemoveAutoUpdate off the action, and calls EnsureClaudeAutoUpdate or
// RemoveClaudeAutoUpdate for a host whose Result did not fail.
type Result struct {
	Host Host

	// Kind is what the executor did, which is not always what was planned: under
	// PrintOnly a planned ActionRun is reported as ActionPrintCommand, so a
	// caller can tell a printed command from an executed one.
	Kind ActionKind

	// Failed is true only for an attempted ActionRun whose command exited
	// nonzero or timed out.
	Failed bool
}

// ExecuteOptions carries the decisions the environment makes, read at the edge
// rather than inside the planner.
type ExecuteOptions struct {
	// PrintOnly downgrades every ActionRun to its printed command without
	// running anything. This is how the delivery surface satisfies the
	// requirement that a non-interactive init without --agent, and any CI run,
	// print the commands and execute nothing — one planner, one executor, and
	// the environment read at the edge instead of inside Plan.
	PrintOnly bool
}

// interactiveSession reports whether a terminal is attached. It is the seam a
// test drives to prove the non-interactive flag is appended, and it answers
// false wherever /dev/tty cannot be opened — Windows included, where the
// conservative answer is the safe one: an appended confirmation flag costs
// nothing, and a missing one hangs a host command on a prompt nobody can see.
var interactiveSession = defaultInteractiveSession

func defaultInteractiveSession() bool {
	f, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

// Execute performs a plan and reports what it did. It bounds the whole call at
// the step timeout: hosts not reached inside it are skipped with no output at
// all, which is what keeps a slow host from delaying the command that hosts the
// step — updating-the-plugin.spec, Failure Behavior 5.
//
// One host's failure never ends the run. A nonzero exit or a timeout reports the
// exact command and moves to the next host, because the hosts are independent
// and a machine with three of them should not lose two to the first.
func Execute(ctx context.Context, actions []Action, r Reporter, opts ExecuteOptions) []Result {
	ctx, cancel := context.WithTimeout(ctx, stepTimeout)
	defer cancel()

	results := make([]Result, 0, len(actions))
	for _, action := range actions {
		if ctx.Err() != nil {
			return results
		}
		result, reached := executeAction(ctx, action, r, opts)
		if !reached {
			return results
		}
		results = append(results, result)
	}
	return results
}

// executeAction performs one host's action. The second result is false when the
// step bound elapsed inside it, which ends the run.
func executeAction(ctx context.Context, a Action, r Reporter, opts ExecuteOptions) (Result, bool) {
	switch a.Kind {
	case ActionRun:
		if opts.PrintOnly {
			r.PrintCommand(a.Host, a.Commands)
			return Result{Host: a.Host, Kind: ActionPrintCommand}, true
		}
		return runHost(ctx, a, r)
	case ActionPrintCommand:
		r.PrintCommand(a.Host, a.Commands)
	case ActionPrintUINote:
		r.UINote(a.Host, a.Note)
	case ActionReportInstalled:
		r.AlreadyInstalled(a.Host)
	case ActionReportStatus:
		r.Status(a.Host, a.Evidence)
	}
	return Result{Host: a.Host, Kind: a.Kind}, true
}

// runHost runs one host's command sequence in order and stops the sequence at
// the first failure: the later commands of a sequence depend on the earlier
// ones — a plugin install after a marketplace add that did not happen has
// nothing to install from — so continuing would only produce a second failure
// line for one cause.
func runHost(ctx context.Context, a Action, r Reporter) (Result, bool) {
	// A host outside the command table carries no non-interactive flag. The
	// planned commands still run: the executor performs the plan it was handed.
	spec, _ := SpecFor(a.Host)

	result := Result{Host: a.Host, Kind: ActionRun}
	for _, planned := range a.Commands {
		if ctx.Err() != nil {
			return Result{}, false
		}
		c := effectiveCommand(spec, planned)
		r.Progress(a.Host, c)

		if out := runCommand(ctx, c); out.Failed {
			// A command cut short by the step bound belongs to the bound, not to
			// the host: Failure Behavior 5 asks for silence once it elapses, and
			// Failure Behavior 2 and 4 ask for the command line in every other
			// case — updating-the-plugin.spec.
			if ctx.Err() != nil {
				return Result{}, false
			}
			r.CommandFailed(a.Host, c)
			result.Failed = true
			break
		}
	}
	return result, true
}

// effectiveCommand returns the command line that actually runs. A host whose
// spec names a non-interactive flag gets it appended here, on a session with no
// terminal, rather than in the plan: the printed tiers hand the user a line to
// type at a prompt that does exist, and a flag baked into the plan would reach
// those lines too.
func effectiveCommand(spec HostSpec, c Command) Command {
	if spec.NonInteractiveFlag == "" || interactiveSession() {
		return c
	}
	return Command{Name: c.Name, Args: append(slices.Clone(c.Args), spec.NonInteractiveFlag)}
}
