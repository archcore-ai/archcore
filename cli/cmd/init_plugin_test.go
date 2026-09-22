package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/plugin"
	"archcore-cli/internal/telemetry"
)

// The delivery step of `archcore init` — plugin-delivery.spec. Every test here
// stages a machine through the evidence seam, a package var or the process
// environment, so none of them calls t.Parallel() except those that only read a
// pure function.

// withoutCIVars clears every variable pluginRunningInCI reads. Without it a test
// that stages consent proves nothing on a CI runner: the printing it asserts
// would come from the environment it inherited rather than from the branch under
// test, and the branch that installs could never run at all.
func withoutCIVars(t *testing.T) {
	t.Helper()
	for _, name := range telemetry.CIVars {
		t.Setenv(name, "")
	}
}

// claudeOnPATH is the machine both mutating tiers need: the host CLI resolves,
// and its listing does not name the plugin, so an install is planned.
func claudeOnPATH() plugin.Evidence {
	return plugin.Evidence{Host: plugin.HostClaudeCode, CLIPresent: true, ListingOK: true}
}

// countLinesWith returns how many of out's lines carry substr.
func countLinesWith(out, substr string) int {
	n := 0
	for line := range strings.SplitSeq(out, "\n") {
		if strings.Contains(line, substr) {
			n++
		}
	}
	return n
}

// runInitCapturingStdout runs the init command and returns what init printed.
// init writes its lines with fmt.Println, so the process stdout — not cobra's
// buffer — is where the delivery step's output lands.
func runInitCapturingStdout(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var execErr error
	out := captureStdout(t, func() {
		_, execErr = runInitCmdForSpec(t, args...)
	})
	return out, execErr
}

// TestAgentPickerLabelDisclosesThePluginInstall covers requirements 1 and 2: the
// screen states the install for every host that ships a plugin, and names it
// machine-level for the two whose stores sit outside the repository.
func TestAgentPickerLabelDisclosesThePluginInstall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		id              agents.AgentID
		wantDisclosure  bool
		wantMachineWide bool
	}{
		{name: "claude code", id: agents.ClaudeCode, wantDisclosure: true},
		{name: "cursor", id: agents.Cursor, wantDisclosure: true},
		{name: "codex cli is machine-scoped", id: agents.CodexCLI, wantDisclosure: true, wantMachineWide: true},
		{name: "copilot is machine-scoped", id: agents.Copilot, wantDisclosure: true, wantMachineWide: true},
		{name: "gemini cli ships no plugin", id: agents.GeminiCLI},
		{name: "opencode ships no plugin", id: agents.OpenCode},
		{name: "roo code ships no plugin", id: agents.RooCode},
		{name: "cline ships no plugin", id: agents.Cline},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			agent := agents.ByID(tt.id)
			if agent == nil {
				t.Fatalf("no agent registered for %q", tt.id)
			}
			label := agentPickerLabel(agent)

			// The label is what the user picks by, so the host's own name has to
			// stay in front of the disclosure.
			if !strings.HasPrefix(label, agent.DisplayName) {
				t.Errorf("label %q does not lead with %q", label, agent.DisplayName)
			}
			// Matched against the requirement's own words rather than the
			// constants, so rewording the constant does not reword the contract.
			if got := strings.Contains(label, "installs the Archcore plugin"); got != tt.wantDisclosure {
				t.Errorf("label %q discloses the plugin install = %v, want %v", label, got, tt.wantDisclosure)
			}
			if got := strings.Contains(label, "machine-level"); got != tt.wantMachineWide {
				t.Errorf("label %q names the install machine-level = %v, want %v", label, got, tt.wantMachineWide)
			}
		})
	}
}

// TestAgentPickerLabelMarksEveryPluginHost is the guard on the table rather than
// on today's four rows: a host added to internal/plugin without a label here
// would ship a screen that installs something it never mentioned.
func TestAgentPickerLabelMarksEveryPluginHost(t *testing.T) {
	t.Parallel()

	for _, host := range plugin.Hosts() {
		agent := agents.ByID(agents.AgentID(host))
		if agent == nil {
			t.Errorf("plugin host %q has no agent in the registry, so it can never be selected", host)
			continue
		}
		if !strings.Contains(agentPickerLabel(agent), "installs the Archcore plugin") {
			t.Errorf("the label for plugin host %q carries no disclosure: %q", host, agentPickerLabel(agent))
		}
	}
}

// TestInitYesWithoutAgentPrintsThePluginCommands covers the --yes half of the
// release criterion: zero plugin subprocesses, the per-host commands printed.
//
// The project already carries .archcore/, so the run also has to pass the
// reinitialize confirm. That is the other half of what --yes does, and it is
// asserted by the run completing at all: without the flag the confirm opens a
// prompt no test can answer.
func TestInitYesWithoutAgentPrintsThePluginCommands(t *testing.T) {
	withoutCIVars(t)
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, claudeOnPATH())
	withPickAgents(t, agentSelection{outcome: outcomePicked, agents: []*agents.Agent{agents.ByID(agents.ClaudeCode)}})
	withConfirmInstructions(t, func([]string) (bool, error) { return false, nil })

	project := setupArchcoreDir(t)
	out, execErr := runInitCapturingStdout(t, "--yes", "--project", project)

	if execErr != nil {
		t.Fatalf("init --yes exited nonzero: %v\n%s", execErr, out)
	}
	if runs := fixtureRuns(t, log); len(runs) != 0 {
		t.Errorf("--yes ran %v, want every command printed and none executed", runs)
	}
	for _, want := range []string{
		"claude plugin marketplace add archcore-ai/plugin",
		"claude plugin install archcore@archcore-plugins",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the printed commands are missing %q:\n%s", want, out)
		}
	}
}

// TestInitInCIPrintsThePluginCommands covers requirement 8. The selection is a
// checked host — full consent — and the CI variable still turns the step into
// printed commands, because an unattended runner has nobody to answer for the
// machine it would change.
func TestInitInCIPrintsThePluginCommands(t *testing.T) {
	withoutCIVars(t)
	t.Setenv("GITHUB_ACTIONS", "true")
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, claudeOnPATH())
	withPickAgents(t, agentSelection{outcome: outcomePicked, agents: []*agents.Agent{agents.ByID(agents.ClaudeCode)}})
	withConfirmInstructions(t, func([]string) (bool, error) { return false, nil })

	out, execErr := runInitCapturingStdout(t, "--project", t.TempDir())

	if execErr != nil {
		t.Fatalf("init on a CI runner exited nonzero: %v\n%s", execErr, out)
	}
	if runs := fixtureRuns(t, log); len(runs) != 0 {
		t.Errorf("a CI run executed %v", runs)
	}
	if !strings.Contains(out, "claude plugin install archcore@archcore-plugins") {
		t.Errorf("the CI run printed no install command:\n%s", out)
	}
}

// TestInitWithAgentInstallsThePlugin covers requirement 6: the flag carries the
// consent, so the step acts on a session with no terminal — the shape the plugin
// skill and every script invoke init in.
//
// The settings assertions are requirement 12 as an end-to-end fact: the entry
// lands in the user's own Claude Code settings, and the repository's copy — which
// init just wrote hooks into — gains nothing.
func TestInitWithAgentInstallsThePlugin(t *testing.T) {
	withoutCIVars(t)
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, claudeOnPATH())
	// No withInteractive here: --agent routes through runInitForAgents, which
	// passes printOnly=false outright and never reads the seam. Setting it would
	// read as if this case pinned the no-terminal branch, which it does not —
	// the flag carries consent with or without a terminal (§6).

	project := t.TempDir()
	out, execErr := runInitCapturingStdout(t, "--agent", "claude-code", "--project", project)

	if execErr != nil {
		t.Fatalf("init --agent claude-code exited nonzero: %v\n%s", execErr, out)
	}
	if runs := fixtureRuns(t, log); len(runs) != 2 {
		t.Fatalf("the host CLI ran %d times %v, want both install commands", len(runs), runs)
	}
	if entry := marketplaceEntry(t, filepath.Join(os.Getenv("HOME"), ".claude", "settings.json")); entry["autoUpdate"] != true {
		t.Errorf("the user's settings entry = %+v, want autoUpdate enabled", entry)
	}
	if entry := marketplaceEntry(t, filepath.Join(project, ".claude", "settings.json")); entry != nil {
		t.Errorf("the repository's settings gained a marketplace entry %+v; init has no --scope project", entry)
	}
}

// TestInitInstallsForACheckedHostAndSurvivesItsFailure is requirement 3 with the
// Constraint that bounds it. A checked host is consent, so the step runs the
// host's own commands with no second prompt; and a host CLI that refuses is still
// not a reason to tell a script that the project was not initialized, when
// .archcore/ and every host config are on disk.
//
// The subprocess log carries both halves: one invocation proves the selection
// installed rather than printed, and exactly one proves the sequence stopped at
// the failure.
func TestInitInstallsForACheckedHostAndSurvivesItsFailure(t *testing.T) {
	withoutCIVars(t)
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 1)
	stubPluginEvidence(t, claudeOnPATH())
	withPickAgents(t, agentSelection{outcome: outcomePicked, agents: []*agents.Agent{agents.ByID(agents.ClaudeCode)}})
	withConfirmInstructions(t, func([]string) (bool, error) { return false, nil })

	project := t.TempDir()
	out, execErr := runInitCapturingStdout(t, "--project", project)

	if execErr != nil {
		t.Fatalf("a failed delivery changed init's exit code: %v\n%s", execErr, out)
	}
	if runs := fixtureRuns(t, log); len(runs) != 1 {
		t.Fatalf("the host CLI ran %d times %v, want the checked host installed and the sequence stopped at the failure", len(runs), runs)
	}
	if !strings.Contains(out, "claude plugin marketplace add archcore-ai/plugin") {
		t.Errorf("the failing command line was swallowed:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(project, ".claude", "settings.json")); err != nil {
		t.Errorf("the wiring the selection asked for is missing after a failed delivery: %v", err)
	}
}

// TestInitPrintsThePluginCommandsWhenTheTerminalIsGone covers requirement 7 at
// the line that enforces it. The consent a checked host carries is consent to
// install, not permission to act unattended, so the step asks about the terminal
// itself rather than inheriting the picker's answer.
//
// The terminal goes away as the picker returns, because that is the only way to
// reach this branch: resolveAgents refuses to open the picker without one, so a
// selection and a missing terminal cannot otherwise coexist. What the case proves
// is that the answer is read where it is used — the guard is the last thing
// standing between a selection and an unattended install.
func TestInitPrintsThePluginCommandsWhenTheTerminalIsGone(t *testing.T) {
	withoutCIVars(t)
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, claudeOnPATH())
	withPickAgentsFn(t, func() (agentSelection, error) {
		isInteractive = func() bool { return false }
		return agentSelection{outcome: outcomePicked, agents: []*agents.Agent{agents.ByID(agents.ClaudeCode)}}, nil
	})

	out, execErr := runInitCapturingStdout(t, "--project", t.TempDir())

	if execErr != nil {
		t.Fatalf("init without a terminal exited nonzero: %v\n%s", execErr, out)
	}
	if runs := fixtureRuns(t, log); len(runs) != 0 {
		t.Errorf("a session with no terminal ran %v", runs)
	}
	if !strings.Contains(out, "claude plugin install archcore@archcore-plugins") {
		t.Errorf("the commands were not printed:\n%s", out)
	}
}

// TestPluginHostsForAgentsFollowsTheCanonicalHostOrder is
// bounded-and-deterministic-output.rule on a selection carrying more than one
// plugin host: the offer line names them and the step observes them in one fixed
// order, whatever order they were selected in. A map's iteration order would
// reshuffle both between runs of the same command.
func TestPluginHostsForAgentsFollowsTheCanonicalHostOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		selected []agents.AgentID
		want     []plugin.Host
	}{
		{
			name:     "every plugin host, selected backwards",
			selected: []agents.AgentID{agents.Copilot, agents.CodexCLI, agents.Cursor, agents.ClaudeCode},
			want:     plugin.Hosts(),
		},
		{
			name:     "two hosts among agents that ship no plugin",
			selected: []agents.AgentID{agents.CodexCLI, agents.GeminiCLI, agents.ClaudeCode, agents.OpenCode},
			want:     []plugin.Host{plugin.HostClaudeCode, plugin.HostCodexCLI},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			selected := make([]*agents.Agent, 0, len(tt.selected))
			for _, id := range tt.selected {
				agent := agents.ByID(id)
				if agent == nil {
					t.Fatalf("no agent registered for %q", id)
				}
				selected = append(selected, agent)
			}

			// Repeated, because one map walk can agree with the canonical order by
			// chance and a single pass would call that determinism.
			for range 16 {
				if got := pluginHostsForAgents(selected); !slices.Equal(got, tt.want) {
					t.Fatalf("pluginHostsForAgents(%v) = %v, want %v", tt.selected, got, tt.want)
				}
			}
		})
	}
}

// TestInitOverAnInstalledPluginReportsAndChangesNothing covers requirement 9 and
// the idempotence invariant: repeated inits never nag and never re-install.
func TestInitOverAnInstalledPluginReportsAndChangesNothing(t *testing.T) {
	withoutCIVars(t)
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, listedClaude())

	out, execErr := runInitCapturingStdout(t, "--agent", "claude-code", "--project", t.TempDir())

	if execErr != nil {
		t.Fatalf("a rerun exited nonzero: %v\n%s", execErr, out)
	}
	if runs := fixtureRuns(t, log); len(runs) != 0 {
		t.Errorf("an install over a listed plugin ran %v, want no command at all", runs)
	}
	if !strings.Contains(out, "already installed") {
		t.Errorf("the no-op was not reported:\n%s", out)
	}
}

// TestInitSurvivesAFailedPluginDelivery covers the Constraint that keeps the step
// out of init's exit code. Requirement 18 binds `archcore plugin`, not this: a
// host CLI that refuses is not a reason to tell a script that the project was not
// initialized, when .archcore/ and every host config are on disk.
func TestInitSurvivesAFailedPluginDelivery(t *testing.T) {
	withoutCIVars(t)
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 1)
	stubPluginEvidence(t, claudeOnPATH())

	project := t.TempDir()
	out, execErr := runInitCapturingStdout(t, "--agent", "claude-code", "--project", project)

	if execErr != nil {
		t.Fatalf("a failed delivery changed init's exit code: %v\n%s", execErr, out)
	}
	if runs := fixtureRuns(t, log); len(runs) != 1 {
		t.Fatalf("the host CLI ran %d times %v, want the sequence stopped at the failure", len(runs), runs)
	}
	// Silent to the exit code, not to the user: Failure Behavior 2 wants the
	// exact command back so it can be rerun.
	if !strings.Contains(out, "claude plugin marketplace add archcore-ai/plugin") {
		t.Errorf("the failing command line was swallowed:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(project, ".claude", "settings.json")); err != nil {
		t.Errorf("the wiring init was asked for is missing after a failed delivery: %v", err)
	}
}

// TestInitDetectedAgentsGetTheOfferAndNoInstall is the subtle half of the
// consent invariant. A project that already carries .claude/ skips the selection
// screen entirely, so the user saw no disclosure and checked no box: the step
// must offer the plugin and install nothing.
//
// The evidence seam is asserted untouched rather than merely silent. A step that
// collected evidence and printed nothing still queries every host CLI on the
// machine for a selection that never consented to being asked.
func TestInitDetectedAgentsGetTheOfferAndNoInstall(t *testing.T) {
	withoutCIVars(t)
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stub := stubPluginEvidence(t, claudeOnPATH())
	withInteractive(t, false)

	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, execErr := runInitCapturingStdout(t, "--project", project)

	if execErr != nil {
		t.Fatalf("init over a detected host exited nonzero: %v\n%s", execErr, out)
	}
	if stub.hosts != nil {
		t.Errorf("the detected path collected plugin evidence for %v", stub.hosts)
	}
	if runs := fixtureRuns(t, log); len(runs) != 0 {
		t.Errorf("the detected path ran %v", runs)
	}
	if n := countLinesWith(out, "archcore plugin install"); n != 1 {
		t.Errorf("the offer appears on %d lines, want exactly one:\n%s", n, out)
	}
	if !strings.Contains(out, "Claude Code") {
		t.Errorf("the offer names no host:\n%s", out)
	}
}

// TestInitWithAgentWithoutAPluginDeliversNothing keeps `--agent gemini-cli` a
// valid wiring run. Only a typed `archcore plugin --agent gemini-cli` errors —
// here the agent simply contributes no host, and a step with no host says
// nothing at all.
func TestInitWithAgentWithoutAPluginDeliversNothing(t *testing.T) {
	withoutCIVars(t)
	isolatePluginRun(t)
	stub := stubPluginEvidence(t, claudeOnPATH())

	project := t.TempDir()
	out, execErr := runInitCapturingStdout(t, "--agent", "gemini-cli", "--project", project)

	if execErr != nil {
		t.Fatalf("init --agent gemini-cli exited nonzero: %v\n%s", execErr, out)
	}
	if stub.hosts != nil {
		t.Errorf("an agent that ships no plugin collected evidence for %v", stub.hosts)
	}
	if strings.Contains(strings.ToLower(out), "plugin") {
		t.Errorf("a run that delivers no plugin mentioned one:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(project, ".gemini", "settings.json")); err != nil {
		t.Errorf("the wiring the flag asked for is missing: %v", err)
	}
}

// TestDeliverPluginsIsSilentWithoutAPluginHost states the same rule against the
// step directly, for a selection where no agent maps to a host at all.
func TestDeliverPluginsIsSilentWithoutAPluginHost(t *testing.T) {
	withoutCIVars(t)
	isolatePluginRun(t)
	stub := stubPluginEvidence(t, claudeOnPATH())

	var out strings.Builder
	deliverPlugins(t.Context(), &out, []*agents.Agent{
		agents.ByID(agents.GeminiCLI),
		agents.ByID(agents.OpenCode),
	}, false)

	if out.String() != "" {
		t.Errorf("a selection with no plugin host printed:\n%s", out.String())
	}
	if stub.hosts != nil {
		t.Errorf("a selection with no plugin host collected evidence for %v", stub.hosts)
	}
}

// TestDeliverPluginsReportsTheOverlapItCaused lived here and was subsumed.
// TestEveryMutatingEntryPointReportsTheOverlapItCaused in plugin_run_test.go
// carries a "delivery step of `archcore init`" row for the install case, and
// TestTheSelfCausedOverlapNoticeNeedsAMutationThatSucceeded carries one for the
// print-only case — the two halves this test asserted, in the file that covers
// all four entry points together.

// TestDeliveryPathOpensNoPrompt covers the half of requirement 3 that running
// the code cannot reach: a checked host installs "with no second prompt".
//
// No behavioral test can hold it. huh needs a terminal and the suite has none,
// so a confirm added to the delivery path would fail immediately and — handled
// the way every other prompt error in this file is, by carrying on — leave the
// install running and every assertion green. The user who does have a terminal
// is the only one who would ever see it, which is exactly the person the
// requirement protects.
//
// So the property is pinned where it is decidable: the selection screen is the
// consent surface, and nothing below it asks again. The guard names each
// function directly, so a rename fails here loudly rather than passing on
// nothing.
func TestDeliveryPathOpensNoPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		file string
		fn   string
	}{
		// The init step itself: everything between the selection and the host
		// commands.
		{file: "init.go", fn: "deliverPlugins"},
		// The shared core all three entry points run. A prompt here would reach
		// the init step too, and `archcore plugin` besides, where requirement 11
		// makes a non-interactive invocation act exactly like an interactive one.
		{file: "plugin_run.go", fn: "runPluginActions"},
	}

	for _, tt := range tests {
		t.Run(tt.fn, func(t *testing.T) {
			t.Parallel()
			if pkg := promptPackageUsedBy(t, tt.file, tt.fn); pkg != "" {
				t.Errorf("%s calls into %q: a checked host installs with no second prompt — plugin-delivery.spec §3", tt.fn, pkg)
			}
		})
	}
}

// funcDeclIn returns the top-level function named name in file, failing the test
// when it is absent.
//
// Both source guards in this package — the prompt check below and the
// plugin-reference check in update_plugin_test.go — need it, and each had
// written the parse-and-search out in full, including the same failure message.
// A guard that names a function must move with that function, and two copies of
// the lookup are two places to forget.
func funcDeclIn(t *testing.T, file, name string) *ast.FuncDecl {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	for _, decl := range parsed.Decls {
		if d, ok := decl.(*ast.FuncDecl); ok && d.Name.Name == name && d.Recv == nil {
			return d
		}
	}
	t.Fatalf("no func %s in %s; this guard names it and must move with the code", name, file)
	return nil
}

// promptPackageUsedBy returns the prompt package the named function reaches, or
// "" when it opens none. huh is the only one the binary links; naming it
// explicitly is what makes the guard readable, and a second prompt library
// arriving in this repository is a change large enough to revisit this list.
func promptPackageUsedBy(t *testing.T, file, name string) string {
	t.Helper()

	fn := funcDeclIn(t, file, name)

	found := ""
	ast.Inspect(fn, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "huh" {
			return true
		}
		found = pkg.Name
		return false
	})
	return found
}

// TestResolveAgentsSeparatesDetectionFromAPick locks the distinction the whole
// consent model rests on: both branches return agents to wire, and only the
// picker's carries consent to install a plugin.
func TestResolveAgentsSeparatesDetectionFromAPick(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	withInteractive(t, false)

	sel, err := resolveAgents(project)
	if err != nil {
		t.Fatalf("resolveAgents: %v", err)
	}
	if sel.outcome != outcomeDetected {
		t.Errorf("outcome = %d, want outcomeDetected(%d)", sel.outcome, outcomeDetected)
	}
	if len(sel.agents) == 0 {
		t.Error("detection returned no agents, so the outcome proves nothing")
	}
}

// TestInitPrintsOneOverlapNoticeAtMost is the regression guard for a run that
// printed both wordings of the duplicate-hook notice, contradicting itself.
//
// installAgents printed DescribePluginConflict ("Until it is updated … Updating
// the plugin resolves it") and the delivery step then printed
// DescribeSelfCausedPluginConflict ("installed and current … Restart the host
// session"). The two detectors ask the same question — is a plugin on this
// machine? — so a staged cache put both on the screen, telling the user to
// update a plugin the same run had just reported current. Each detector also
// re-ran the bounded plugin walk, so init paid it twice.
//
// Both cases stage the cache. The difference is whether this run installed
// anything, which is what decides which of the two wordings is true.
func TestInitPrintsOneOverlapNoticeAtMost(t *testing.T) {
	const (
		selfCaused  = "Restart the host session"
		preexisting = "Updating the plugin resolves it"
	)

	tests := []struct {
		name      string
		printOnly bool
		want      string
	}{
		{name: "the step installed the plugin", want: selfCaused},
		{name: "the step only printed the commands", printOnly: true, want: preexisting},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withoutCIVars(t)
			bin := isolatePluginRun(t)
			writeHostFixture(t, bin, "claude", 0)
			stubPluginEvidence(t, claudeOnPATH())
			stageInstalledPluginCache(t)

			// installAgents is what printed the second notice; running it here
			// is what makes this a test of the whole init sequence rather than
			// of deliverPlugins alone.
			//
			// Both streams are collected: deliverPlugins writes to the writer it
			// is handed, printPluginConflictNote writes to os.Stdout, and a test
			// that read only one of them could never see the two together — which
			// is the whole failure.
			var out strings.Builder
			list := []*agents.Agent{agents.ByID(agents.ClaudeCode)}
			// A real .archcore/, so the hook wiring actually installs: without
			// it installAgents reports "not found", anyHooks stays false, and
			// the notice under test is never reached at all.
			project := setupArchcoreDir(t)
			stdout := captureStdout(t, func() {
				anyHooks := installAgents(project, list, false)
				if !deliverPlugins(t.Context(), &out, list, tt.printOnly) && anyHooks {
					printPluginConflictNote()
				}
			})

			text := stdout + out.String()
			if got := strings.Contains(text, selfCaused); got != (tt.want == selfCaused) {
				t.Errorf("printed the self-caused wording = %v, want %v:\n%s", got, tt.want == selfCaused, text)
			}
			if got := strings.Contains(text, preexisting); got != (tt.want == preexisting) {
				t.Errorf("printed the pre-existing wording = %v, want %v:\n%s", got, tt.want == preexisting, text)
			}
			if strings.Contains(text, selfCaused) && strings.Contains(text, preexisting) {
				t.Errorf("printed both wordings of the overlap notice, which contradict each other:\n%s", text)
			}
		})
	}
}
