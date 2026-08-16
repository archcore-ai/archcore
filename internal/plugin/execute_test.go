package plugin

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

// recordedLine is one call the executor made on its Reporter, kept as text so a
// test asserts on what the user would see rather than on a call shape.
type recordedLine struct {
	kind string
	host Host
	text string
}

// recordingReporter stands in for an entry point. Execute writes to no stream
// itself, so this is the only place its output can be observed — which is the
// point of the interface: each entry point words the same decisions its own way.
type recordingReporter struct {
	lines []recordedLine
}

func (r *recordingReporter) add(kind string, host Host, text string) {
	r.lines = append(r.lines, recordedLine{kind: kind, host: host, text: text})
}

func (r *recordingReporter) Progress(host Host, c Command)      { r.add("progress", host, c.String()) }
func (r *recordingReporter) CommandFailed(host Host, c Command) { r.add("failed", host, c.String()) }
func (r *recordingReporter) UINote(host Host, note string)      { r.add("note", host, note) }
func (r *recordingReporter) AlreadyInstalled(host Host)         { r.add("installed", host, "") }
func (r *recordingReporter) Status(host Host, ev Evidence)      { r.add("status", host, ev.ListedVersion) }

func (r *recordingReporter) PrintCommand(host Host, cs []Command) {
	for _, c := range cs {
		r.add("print", host, c.String())
	}
}

// texts returns the text of every line of one kind, in order.
func (r *recordingReporter) texts(kind string) []string {
	var out []string
	for _, line := range r.lines {
		if line.kind == kind {
			out = append(out, line.text)
		}
	}
	return out
}

// lastProgress returns the text of the most recent progress line, or an empty
// string when none has been reported yet. It is how a test observes the order
// of a report against the command it announces, from inside the subprocess
// seam.
func (r *recordingReporter) lastProgress() string {
	for i := len(r.lines) - 1; i >= 0; i-- {
		if r.lines[i].kind == "progress" {
			return r.lines[i].text
		}
	}
	return ""
}

// forHost returns every line reported for one host, in order.
func (r *recordingReporter) forHost(host Host) []recordedLine {
	var out []recordedLine
	for _, line := range r.lines {
		if line.host == host {
			out = append(out, line)
		}
	}
	return out
}

// listedEvidence is a host that answered its own listing and named the plugin.
func listedEvidence(host Host) Evidence {
	return Evidence{Host: host, CLIPresent: true, ListingOK: true, Listed: true}
}

// TestExecuteStaysSilentWhenTheListingOmitsThePlugin is the invariant both
// specs share: a machine that never installed the plugin pays no mutating
// command and sees no output. The listing is the only command that runs, and
// its answer ends the step.
func TestExecuteStaysSilentWhenTheListingOmitsThePlugin(t *testing.T) {
	skipWithoutPOSIXShell(t)
	isolateHome(t)
	setInteractive(t, true)
	dir := t.TempDir()
	writeHostCLI(t, dir, "claude", listingScript(otherPluginListing, "exit 0"))
	useHostCLIs(t, dir, "claude")
	rec := recordRuns(t)

	evidence := CollectEvidence(t.Context(), []Host{HostClaudeCode})
	actions := Plan(VerbUpdate, evidence)
	if len(actions) != 0 {
		t.Fatalf("planned %+v for a host whose listing omits the plugin, want nothing", actions)
	}

	reporter := &recordingReporter{}
	results := Execute(t.Context(), actions, reporter, ExecuteOptions{})
	if len(results) != 0 || len(reporter.lines) != 0 {
		t.Errorf("results %+v and output %+v, want both empty", results, reporter.lines)
	}
	if want := []string{"claude plugin list --json"}; !slices.Equal(rec.lines(), want) {
		t.Errorf("ran %q, want only the read-only listing %q", rec.lines(), want)
	}
}

// TestExecuteProducesNothingWithoutEvidence covers the machine that has neither
// a host CLI nor a registry entry: no tier applies, so no host is addressed.
func TestExecuteProducesNothingWithoutEvidence(t *testing.T) {
	isolateHome(t)
	noHostCLIs(t)
	setInteractive(t, true)
	rec := recordRuns(t)

	evidence := CollectEvidence(t.Context(), Hosts())
	for _, verb := range []Verb{VerbUpdate, VerbRemove} {
		t.Run(verb.String(), func(t *testing.T) {
			reporter := &recordingReporter{}
			results := Execute(t.Context(), Plan(verb, evidence), reporter, ExecuteOptions{})
			if len(results) != 0 || len(reporter.lines) != 0 {
				t.Errorf("results %+v and output %+v, want both empty", results, reporter.lines)
			}
		})
	}
	if len(rec.commands) != 0 {
		t.Errorf("ran %q on a machine with no host CLI, want nothing", rec.lines())
	}
}

// TestExecutePrintsTheExactCommandForARegistryListedHost covers the tier that
// exists because a host's binary is often not on PATH — the Copilot row of the
// spec says so outright. The printed line is what the user pastes, so it is
// pinned literally.
func TestExecutePrintsTheExactCommandForARegistryListedHost(t *testing.T) {
	home := isolateHome(t)
	noHostCLIs(t)
	setInteractive(t, false)
	writeRegistryEntry(t, home, ".copilot/installed-plugins/_direct/archcore-ai--plugin--plugins-archcore")
	rec := recordRuns(t)

	evidence := CollectEvidence(t.Context(), []Host{HostCopilot})
	reporter := &recordingReporter{}
	results := Execute(t.Context(), Plan(VerbUpdate, evidence), reporter, ExecuteOptions{})

	want := []string{"copilot plugin update archcore@archcore-plugins"}
	if got := reporter.texts("print"); !slices.Equal(got, want) {
		t.Errorf("printed %q, want %q", got, want)
	}
	if len(rec.commands) != 0 {
		t.Errorf("ran %q for a host with no CLI, want nothing", rec.lines())
	}
	if len(results) != 1 || results[0].Kind != ActionPrintCommand || results[0].Failed {
		t.Errorf("results = %+v, want one printed command that did not fail", results)
	}
}

// TestExecuteReportsAFailedCommandAndContinues covers Failure Behavior 2 of
// updating-the-plugin.spec: the exact command, then the next host. One host's
// broken CLI must not cost the others their update.
func TestExecuteReportsAFailedCommandAndContinues(t *testing.T) {
	skipWithoutPOSIXShell(t)
	isolateHome(t)
	setInteractive(t, true)
	dir := t.TempDir()
	writeHostCLI(t, dir, "claude", listingScript(pluginListing, "exit 1"))
	writeHostCLI(t, dir, "codex", listingScript(pluginListing, "exit 0"))
	useHostCLIs(t, dir, "claude", "codex")

	reporter := &recordingReporter{}
	actions := Plan(VerbUpdate, []Evidence{listedEvidence(HostClaudeCode), listedEvidence(HostCodexCLI)})
	results := Execute(t.Context(), actions, reporter, ExecuteOptions{})

	wantFailed := []string{"claude plugin marketplace update archcore-plugins"}
	if got := reporter.texts("failed"); !slices.Equal(got, wantFailed) {
		t.Errorf("failure lines = %q, want the exact command %q", got, wantFailed)
	}
	// The second command of the sequence never runs: it had nothing to install
	// from, and a second failure line would report one cause twice.
	wantProgress := []string{
		"claude plugin marketplace update archcore-plugins",
		"codex plugin marketplace upgrade archcore-plugins",
	}
	if got := reporter.texts("progress"); !slices.Equal(got, wantProgress) {
		t.Errorf("progress lines = %q, want %q", got, wantProgress)
	}
	if len(results) != 2 || !results[0].Failed || results[1].Failed {
		t.Errorf("results = %+v, want Claude Code failed and Codex CLI not", results)
	}
}

// TestExecuteReportsATimedOutCommandAndContinues covers Failure Behavior 3 and
// 4: the subprocess is killed and the exact command is still reported, so a
// host CLI blocked on a prompt reads the same as one that refused.
func TestExecuteReportsATimedOutCommandAndContinues(t *testing.T) {
	skipWithoutPOSIXShell(t)
	isolateHome(t)
	setInteractive(t, true)
	// Two seconds, not milliseconds: the second host has to start a real process
	// inside the same budget, and a first exec of a freshly written script costs
	// a measurable fraction of a second on a developer machine.
	shrinkTimeouts(t, 2*time.Second, time.Minute)
	dir := t.TempDir()
	writeHostCLI(t, dir, "claude", `sleep 30`)
	writeHostCLI(t, dir, "codex", `exit 0`)
	useHostCLIs(t, dir, "claude", "codex")

	reporter := &recordingReporter{}
	actions := Plan(VerbUpdate, []Evidence{listedEvidence(HostClaudeCode), listedEvidence(HostCodexCLI)})
	started := time.Now()
	results := Execute(t.Context(), actions, reporter, ExecuteOptions{})

	if elapsed := time.Since(started); elapsed > 20*time.Second {
		t.Fatalf("the run took %s, want the command timeout to cut it", elapsed)
	}
	wantFailed := []string{"claude plugin marketplace update archcore-plugins"}
	if got := reporter.texts("failed"); !slices.Equal(got, wantFailed) {
		t.Errorf("failure lines = %q, want the exact command %q", got, wantFailed)
	}
	if len(results) != 2 || !results[0].Failed || results[1].Failed {
		t.Errorf("results = %+v, want Claude Code failed and Codex CLI not", results)
	}
}

// TestExecutePrintOnlyRunsNothing is the requirement that a non-interactive
// init without --agent and any CI run print the commands and execute nothing.
// The subprocess seam is what proves it: identical output would also come from
// a run that printed the lines and ran them anyway.
func TestExecutePrintOnlyRunsNothing(t *testing.T) {
	skipWithoutPOSIXShell(t)
	isolateHome(t)
	setInteractive(t, true)
	dir := t.TempDir()
	writeHostCLI(t, dir, "claude", `exit 0`)
	writeHostCLI(t, dir, "codex", `exit 0`)
	useHostCLIs(t, dir, "claude", "codex")

	actions := Plan(VerbUpdate, []Evidence{listedEvidence(HostClaudeCode), listedEvidence(HostCodexCLI)})

	// One recorder across both phases, snapshotted between them. Installing a
	// second recordRuns wrapped the first, so the running phase recorded into
	// both and the PrintOnly assertion only held because it was made before the
	// second Execute — one reordered line and it would silently invert.
	rec := recordRuns(t)

	printing := &recordingReporter{}
	printResults := Execute(t.Context(), actions, printing, ExecuteOptions{PrintOnly: true})
	if len(rec.commands) != 0 {
		t.Errorf("PrintOnly ran %q, want no subprocess at all", rec.lines())
	}
	for _, result := range printResults {
		if result.Kind != ActionPrintCommand || result.Failed {
			t.Errorf("result = %+v, want a printed command under PrintOnly", result)
		}
	}

	running := &recordingReporter{}
	Execute(t.Context(), actions, running, ExecuteOptions{})
	runRec := rec

	// The printed lines and the executed ones are the same lines: one planner,
	// one executor, and only the environment differs.
	if got, want := printing.texts("print"), running.texts("progress"); !slices.Equal(got, want) {
		t.Errorf("printed %q, want the commands the run executes %q", got, want)
	}
	if got := runRec.lines(); !slices.Equal(got, printing.texts("print")) {
		t.Errorf("ran %q, want the printed commands %q", got, printing.texts("print"))
	}
}

// TestExecuteStopsAtTheStepBound covers Failure Behavior 5: once the step bound
// elapses the remaining hosts are skipped and print nothing. The host cut by
// the bound is silent too — the bound ended the step, and reporting it as a
// host failure would blame the host for the budget.
// The slow command is a stub rather than a `sleep 30` fixture on PATH. A real
// fixture makes the case depend on fork+execve of a freshly written shell script
// outrunning the bound, and on `sleep` existing: a fixture that failed to start
// would return before the bound elapsed, the second host would run, and the run
// would fail for a reason that has nothing to do with the budget. The stub
// spends the bound by construction. Its sibling below already had this shape.
//
// The stub reports a failure on purpose: the claim is that a host cut by the
// bound is silent even then, because the bound ended the step and blaming the
// host would blame it for the budget.
func TestExecuteStopsAtTheStepBound(t *testing.T) {
	isolateHome(t)
	setInteractive(t, true)
	shrinkTimeouts(t, time.Minute, 50*time.Millisecond)

	var ran []Command
	stubRuns(t, func(c Command) commandOutcome {
		ran = append(ran, c)
		time.Sleep(250 * time.Millisecond)
		return commandOutcome{Failed: true}
	})

	reporter := &recordingReporter{}
	actions := Plan(VerbUpdate, []Evidence{listedEvidence(HostClaudeCode), listedEvidence(HostCodexCLI)})
	results := Execute(t.Context(), actions, reporter, ExecuteOptions{})

	if len(results) != 0 {
		t.Errorf("results = %+v, want none once the step bound elapsed", results)
	}
	if got := reporter.texts("failed"); len(got) != 0 {
		t.Errorf("failure lines = %q, want silence once the step bound elapsed", got)
	}
	if got := reporter.forHost(HostCodexCLI); len(got) != 0 {
		t.Errorf("Codex CLI reported %+v, want nothing for a host the step never reached", got)
	}
	if len(ran) != 1 {
		t.Errorf("ran %d command(s) %+v, want only the one the bound cut short", len(ran), ran)
	}
}

// TestExecuteSkipsEveryRemainingKindAtTheStepBound covers the half of Failure
// Behavior 5 the running host does not reach. A host cut mid-command is caught
// inside the run; a host whose action prints, notes, or reports runs no command
// at all, so the only thing that can keep it silent is the bound being checked
// before the action is performed — updating-the-plugin.spec, Failure Behavior 5.
func TestExecuteSkipsEveryRemainingKindAtTheStepBound(t *testing.T) {
	setInteractive(t, true)
	shrinkTimeouts(t, time.Minute, 50*time.Millisecond)
	stubRuns(t, func(Command) commandOutcome {
		// Long enough to spend the step bound, and the command still succeeds:
		// the remaining hosts are skipped by the budget, not by a failure.
		time.Sleep(250 * time.Millisecond)
		return commandOutcome{}
	})

	actions := []Action{
		{Host: HostClaudeCode, Kind: ActionRun, Commands: []Command{{Name: "claude", Args: []string{"plugin", "update"}}}},
		{Host: HostCursor, Kind: ActionPrintUINote, Note: "Update it in the Cursor UI."},
		{Host: HostCodexCLI, Kind: ActionPrintCommand, Commands: []Command{{Name: "codex", Args: []string{"plugin", "list"}}}},
		{Host: HostCopilot, Kind: ActionReportInstalled},
	}
	reporter := &recordingReporter{}
	results := Execute(t.Context(), actions, reporter, ExecuteOptions{})

	if len(results) != 1 || results[0].Host != HostClaudeCode {
		t.Errorf("results = %+v, want only the host the step reached", results)
	}
	for _, host := range []Host{HostCursor, HostCodexCLI, HostCopilot} {
		if got := reporter.forHost(host); len(got) != 0 {
			t.Errorf("%s reported %+v, want nothing once the step bound elapsed", host, got)
		}
	}
}

// TestExecuteAnnouncesEachCommandBeforeItRuns pins the order of requirement 11
// of updating-the-plugin.spec. The line is not decoration: a host command may
// spend the full 30 s bound, and a progress line printed afterwards leaves the
// user watching a terminal that says nothing for the whole wait. The subprocess
// seam is the only place the order is observable, so the assertion reads the
// reporter from inside the command it announces.
func TestExecuteAnnouncesEachCommandBeforeItRuns(t *testing.T) {
	setInteractive(t, true)
	reporter := &recordingReporter{}

	var announced []string
	stubRuns(t, func(Command) commandOutcome {
		announced = append(announced, reporter.lastProgress())
		return commandOutcome{}
	})

	actions := Plan(VerbInstall, []Evidence{{Host: HostClaudeCode, CLIPresent: true}})
	Execute(t.Context(), actions, reporter, ExecuteOptions{})

	want := []string{
		"claude plugin marketplace add archcore-ai/plugin",
		"claude plugin install archcore@archcore-plugins",
	}
	if !slices.Equal(announced, want) {
		t.Errorf("each command saw %q announced, want its own line %q already reported", announced, want)
	}
}

// TestExecuteReportsTheCommandItActuallyRan covers requirement 12 and Failure
// Behavior 2 and 4 literally: the line is "the exact command it ran", and off a
// terminal the command that ran carries the non-interactive flag. A failure line
// missing it hands the user a line that behaves differently from the one that
// failed.
func TestExecuteReportsTheCommandItActuallyRan(t *testing.T) {
	setInteractive(t, false)
	stubRuns(t, func(Command) commandOutcome { return commandOutcome{Failed: true} })

	reporter := &recordingReporter{}
	actions := Plan(VerbUpdate, []Evidence{listedEvidence(HostClaudeCode)})
	Execute(t.Context(), actions, reporter, ExecuteOptions{})

	want := []string{"claude plugin marketplace update archcore-plugins -y"}
	if got := reporter.texts("failed"); !slices.Equal(got, want) {
		t.Errorf("failure lines = %q, want the command as it ran %q", got, want)
	}
	if got := reporter.texts("progress"); !slices.Equal(got, want) {
		t.Errorf("progress lines = %q, want the command as it ran %q", got, want)
	}
}

// TestExecuteAppendsTheNonInteractiveFlagOffATerminal proves the flag is a
// decision of the run, not of the plan: a host command that would stop on a
// confirmation prompt gets the flag where no terminal can answer it, and the
// same plan leaves it off where one can.
func TestExecuteAppendsTheNonInteractiveFlagOffATerminal(t *testing.T) {
	actions := Plan(VerbUpdate, []Evidence{listedEvidence(HostClaudeCode)})

	tests := []struct {
		name        string
		interactive bool
		want        []string
	}{
		{
			name:        "with a terminal",
			interactive: true,
			want: []string{
				"claude plugin marketplace update archcore-plugins",
				"claude plugin update archcore@archcore-plugins",
			},
		},
		{
			name: "without a terminal",
			want: []string{
				"claude plugin marketplace update archcore-plugins -y",
				"claude plugin update archcore@archcore-plugins -y",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setInteractive(t, tt.interactive)
			stubRuns(t, func(Command) commandOutcome { return commandOutcome{} })
			reporter := &recordingReporter{}

			Execute(t.Context(), actions, reporter, ExecuteOptions{})

			if got := reporter.texts("progress"); !slices.Equal(got, tt.want) {
				t.Errorf("ran %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExecuteDoesNotRewriteThePlan keeps the appended flag out of the caller's
// actions. The same plan is executed once and printed by another entry point,
// and a flag written back into the plan would reach the printed line a user
// pastes.
func TestExecuteDoesNotRewriteThePlan(t *testing.T) {
	setInteractive(t, false)
	stubRuns(t, func(Command) commandOutcome { return commandOutcome{} })

	actions := Plan(VerbUpdate, []Evidence{listedEvidence(HostClaudeCode)})
	before := commandLines(actions[0].Commands)

	Execute(t.Context(), actions, &recordingReporter{}, ExecuteOptions{})

	if got := commandLines(actions[0].Commands); !slices.Equal(got, before) {
		t.Errorf("the plan reads %q after execution, want %q", got, before)
	}
}

// TestExecuteRoutesEveryKindThroughTheReporter keeps every planned kind
// reachable. A kind the executor forgets is a host that silently does nothing,
// which is indistinguishable from the silence the specs ask for elsewhere.
func TestExecuteRoutesEveryKindThroughTheReporter(t *testing.T) {
	setInteractive(t, true)
	stubRuns(t, func(Command) commandOutcome { return commandOutcome{} })

	actions := []Action{
		{Host: HostClaudeCode, Kind: ActionRun, Commands: []Command{{Name: "claude", Args: []string{"plugin", "update"}}}},
		{Host: HostCursor, Kind: ActionPrintUINote, Note: "Update it in the Cursor UI."},
		{Host: HostCodexCLI, Kind: ActionPrintCommand, Commands: []Command{{Name: "codex", Args: []string{"plugin", "list"}}}},
		{Host: HostCopilot, Kind: ActionReportInstalled},
		{Host: HostCopilot, Kind: ActionReportStatus, Evidence: Evidence{Host: HostCopilot, ListedVersion: "1.4.0"}},
	}

	reporter := &recordingReporter{}
	results := Execute(t.Context(), actions, reporter, ExecuteOptions{})

	want := []recordedLine{
		{kind: "progress", host: HostClaudeCode, text: "claude plugin update"},
		{kind: "note", host: HostCursor, text: "Update it in the Cursor UI."},
		{kind: "print", host: HostCodexCLI, text: "codex plugin list"},
		{kind: "installed", host: HostCopilot},
		{kind: "status", host: HostCopilot, text: "1.4.0"},
	}
	if !reflect.DeepEqual(reporter.lines, want) {
		t.Errorf("reported\n%+v\nwant\n%+v", reporter.lines, want)
	}
	if len(results) != len(actions) {
		t.Fatalf("results = %+v, want one per action", results)
	}
	for i, result := range results {
		if result.Host != actions[i].Host || result.Kind != actions[i].Kind {
			t.Errorf("result %d = %+v, want it to name %q %s", i, result, actions[i].Host, actions[i].Kind)
		}
		if result.Failed {
			t.Errorf("result %d failed, want Failed only for an attempted run that failed", i)
		}
	}
}

// TestExecuteReportsAnEmptyPlan keeps the executor from inventing work for a
// caller that planned none.
func TestExecuteReportsAnEmptyPlan(t *testing.T) {
	reporter := &recordingReporter{}
	if results := Execute(t.Context(), nil, reporter, ExecuteOptions{}); len(results) != 0 {
		t.Errorf("results = %+v, want none", results)
	}
	if len(reporter.lines) != 0 {
		t.Errorf("output = %+v, want none", reporter.lines)
	}
}

// TestExecuteFailsOnlyAnAttemptedRun pins the meaning of Result.Failed, which
// the direct `archcore plugin` verbs turn into an exit code: a host skipped for
// missing evidence must not fail the command — plugin-delivery.spec §19.
func TestExecuteFailsOnlyAnAttemptedRun(t *testing.T) {
	setInteractive(t, true)
	stubRuns(t, func(Command) commandOutcome { return commandOutcome{Failed: true} })

	actions := []Action{
		{Host: HostClaudeCode, Kind: ActionRun, Commands: []Command{{Name: "claude"}}},
		{Host: HostCursor, Kind: ActionPrintUINote, Note: "note"},
		{Host: HostCodexCLI, Kind: ActionPrintCommand, Commands: []Command{{Name: "codex"}}},
		{Host: HostCopilot, Kind: ActionReportInstalled},
	}
	results := Execute(t.Context(), actions, &recordingReporter{}, ExecuteOptions{})

	for _, result := range results {
		if wantFailed := result.Kind == ActionRun; result.Failed != wantFailed {
			t.Errorf("%s %s: Failed = %v, want %v", result.Host, result.Kind, result.Failed, wantFailed)
		}
	}
}

// TestExecuteRunsAWholeSequenceInOrder keeps a multi-command host intact: the
// marketplace refresh runs before the plugin update, because the second reads
// what the first wrote.
func TestExecuteRunsAWholeSequenceInOrder(t *testing.T) {
	setInteractive(t, true)
	rec := &runRecorder{}
	stubRuns(t, func(c Command) commandOutcome {
		rec.commands = append(rec.commands, c)
		return commandOutcome{}
	})

	actions := Plan(VerbInstall, []Evidence{{Host: HostClaudeCode, CLIPresent: true}})
	Execute(t.Context(), actions, &recordingReporter{}, ExecuteOptions{})

	want := []string{
		"claude plugin marketplace add archcore-ai/plugin",
		"claude plugin install archcore@archcore-plugins",
	}
	if got := rec.lines(); !slices.Equal(got, want) {
		t.Errorf("ran %q, want %q in order", got, want)
	}
	if !strings.Contains(strings.Join(rec.lines(), " "), MarketplaceID) {
		t.Error("the executed commands lost the frozen marketplace identifier")
	}
}

// TestExecuteRunsAHostOutsideTheCommandTable covers the branch runHost documents
// and nothing exercised: "a host outside the command table carries no
// non-interactive flag; the planned commands still run".
//
// SpecFor answers no row, so the zero HostSpec is used and effectiveCommand
// appends nothing. The executor performs the plan it was handed rather than
// second-guessing it — a planner and an executor that disagree about which hosts
// exist is the failure this pins.
func TestExecuteRunsAHostOutsideTheCommandTable(t *testing.T) {
	setInteractive(t, false)

	var ran []Command
	stubRuns(t, func(c Command) commandOutcome {
		ran = append(ran, c)
		return commandOutcome{}
	})

	const unknown Host = "not-in-the-table"
	if _, ok := SpecFor(unknown); ok {
		t.Fatalf("%q is in the command table, so this test no longer covers the branch", unknown)
	}

	planned := Command{Name: "claude", Args: []string{"plugin", "update"}}
	reporter := &recordingReporter{}
	results := Execute(t.Context(), []Action{
		{Host: unknown, Kind: ActionRun, Commands: []Command{planned}},
	}, reporter, ExecuteOptions{})

	if len(ran) != 1 || ran[0].String() != planned.String() {
		t.Errorf("ran %+v, want the planned command %q unchanged", ran, planned.String())
	}
	// Claude Code would take "-y" here; a host with no row has no flag to take,
	// which is the whole content of the branch.
	if got := reporter.texts("progress"); !slices.Equal(got, []string{planned.String()}) {
		t.Errorf("progress = %q, want the command with no non-interactive flag appended", got)
	}
	if len(results) != 1 || results[0].Host != unknown || results[0].Failed {
		t.Errorf("results = %+v, want one successful run for %q", results, unknown)
	}
}
