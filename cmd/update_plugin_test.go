package cmd

import (
	"bytes"
	"context"
	"go/ast"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"archcore-cli/internal/plugin"
	"archcore-cli/internal/telemetry"
)

// The plugin step of `archcore update` — updating-the-plugin.spec. Every test
// here stages a machine through the evidence seam or the process environment,
// so none of them calls t.Parallel().

// runUpdateWithPluginStep executes `archcore update` with the plugin step live.
//
// It is deliberately not execUpdateCmd. That funnel empties PATH and the home
// directory so the rest of the update suite never reaches the developer's own
// host CLIs, and these are the tests that need the staged machine to survive to
// the step.
func runUpdateWithPluginStep(t *testing.T, version string, srv *httptest.Server, execPath string, tel *telemetry.Client, args ...string) (string, error) {
	t.Helper()

	if args == nil {
		args = []string{}
	}
	cmd := buildUpdateCmd(version, testUpdater(version, srv, execPath), tel)

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	cmd.SetContext(context.Background())
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	execErr := cmd.Execute()
	return buf.String(), execErr
}

// currentReleaseServer answers with the tag the caller already runs, which is
// the branch requirement 2 of updating-the-plugin.spec puts the step on.
func currentReleaseServer(t *testing.T, version string) *httptest.Server {
	t.Helper()
	return releaseArchiveServer(t, releaseFixture{tag: version})
}

// pluginStepOutput returns what the run printed after its binary phase reported
// "Already up to date" — everything below that line is the step's.
func pluginStepOutput(t *testing.T, out string) string {
	t.Helper()
	_, after, found := strings.Cut(out, "Already up to date")
	if !found {
		t.Fatalf("the run never reached the already-current branch:\n%s", out)
	}
	_, after, _ = strings.Cut(after, "\n")
	return after
}

// TestUpdateRunsThePluginStepWhenAlreadyCurrent covers requirement 2. A current
// binary is not a current plugin: the two ship apart, so the branch that
// replaces nothing still has a plugin to refresh.
func TestUpdateRunsThePluginStepWhenAlreadyCurrent(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, listedClaude())

	out, execErr := runUpdateWithPluginStep(t, "v1.0.0", currentReleaseServer(t, "v1.0.0"), "", nil)
	if execErr != nil {
		t.Fatalf("an already-current run must exit zero, got %v\n%s", execErr, out)
	}
	if runs := fixtureRuns(t, log); len(runs) != 2 {
		t.Errorf("the host CLI ran %d times %v, want both update commands", len(runs), runs)
	}
	if !strings.Contains(pluginStepOutput(t, out), "claude plugin update archcore@archcore-plugins") {
		t.Errorf("the step printed no progress line for the command it ran:\n%s", out)
	}
}

// TestUpdateRunsThePluginStepAfterReplacingTheBinary covers requirement 1, and
// the ordering the spec implies: the step runs after the binary phase has
// reported its own result, so a user reads the update they asked for before the
// one they did not.
func TestUpdateRunsThePluginStepAfterReplacingTheBinary(t *testing.T) {
	skipUnlessTarGz(t)
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, listedClaude())

	srv := releaseArchiveServer(t, releaseFixture{tag: "v2.0.0"})
	out, execErr := runUpdateWithPluginStep(t, "v1.0.0", srv, fakeBinary(t), nil)
	if execErr != nil {
		t.Fatalf("a successful update must exit zero, got %v\n%s", execErr, out)
	}
	if runs := fixtureRuns(t, log); len(runs) != 2 {
		t.Errorf("the host CLI ran %d times %v, want both update commands", len(runs), runs)
	}

	updated := strings.Index(out, "Updated to v2.0.0")
	step := strings.Index(out, "claude plugin marketplace update")
	if updated < 0 || step < 0 {
		t.Fatalf("output is missing the binary phase or the step:\n%s", out)
	}
	if step < updated {
		t.Errorf("the step printed before the binary phase reported its result:\n%s", out)
	}
}

// TestUpdateSkipsThePluginStepAfterAFailedBinaryPhase covers requirement 3 on
// both of the command's failure returns.
//
// The assertion is that the evidence seam was never reached, not merely that
// nothing was printed. A step that collected evidence and stayed silent is
// still a step: it queries every host CLI on the machine, and it does so after
// the command has already told the user the update failed.
func TestUpdateSkipsThePluginStepAfterAFailedBinaryPhase(t *testing.T) {
	tests := []struct {
		name     string
		tarGz    bool
		server   func(t *testing.T) *httptest.Server
		execPath func(t *testing.T) string
		wantLine string
	}{
		{
			name: "the check never resolved a release",
			server: func(t *testing.T) *httptest.Server {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "down", http.StatusInternalServerError)
				}))
				t.Cleanup(srv.Close)
				return srv
			},
			execPath: func(t *testing.T) string { return "" },
			wantLine: "Could not check for updates",
		},
		{
			name:  "the binary could not be replaced",
			tarGz: true,
			server: func(t *testing.T) *httptest.Server {
				return releaseArchiveServer(t, releaseFixture{tag: "v2.0.0"})
			},
			execPath: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "absent-dir", "archcore")
			},
			wantLine: "Update failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tarGz {
				skipUnlessTarGz(t)
			}
			bin := isolatePluginRun(t)
			log := writeHostFixture(t, bin, "claude", 0)
			stub := stubPluginEvidence(t, listedClaude())

			out, execErr := runUpdateWithPluginStep(t, "v1.0.0", tt.server(t), tt.execPath(t), nil)
			if execErr == nil {
				t.Fatalf("a failed binary phase must exit nonzero:\n%s", out)
			}
			if !strings.Contains(out, tt.wantLine) {
				t.Fatalf("output is missing %q, so this is not the failure branch:\n%s", tt.wantLine, out)
			}
			if stub.hosts != nil {
				t.Errorf("the step collected evidence for %v after the binary phase failed", stub.hosts)
			}
			if runs := fixtureRuns(t, log); len(runs) != 0 {
				t.Errorf("the step ran %v after the binary phase failed", runs)
			}
		})
	}
}

// TestUpdateCheckNeverRunsThePluginStep covers requirement 4. --check is the
// hook-facing probe behind the session-start advisory, bounded at two seconds
// and expected to print one line or nothing; a 120 s plugin step there would be
// paid on every session start of every project.
func TestUpdateCheckNeverRunsThePluginStep(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stub := stubPluginEvidence(t, listedClaude())

	out, execErr := runUpdateWithPluginStep(t, "v0.5.7", releaseServer(t, "v9.9.9"), "", nil, "--check")
	if execErr != nil {
		t.Fatalf("--check must always exit zero, got %v", execErr)
	}
	if out != "update available: v9.9.9\n" {
		t.Errorf("output = %q, want the advisory line alone", out)
	}
	if stub.hosts != nil {
		t.Errorf("--check collected plugin evidence for %v", stub.hosts)
	}
	if runs := fixtureRuns(t, log); len(runs) != 0 {
		t.Errorf("--check ran %v", runs)
	}
}

// TestUpdatePluginStepFailureLeavesTheExitCodeUnchanged covers requirement 13
// on both branches that host the step. `archcore update` exits with the binary
// phase's result; a host CLI that refuses is the plugin's problem to report,
// not a reason to tell a script that the update failed.
func TestUpdatePluginStepFailureLeavesTheExitCodeUnchanged(t *testing.T) {
	tests := []struct {
		name    string
		tarGz   bool
		version string
		tag     string
	}{
		{name: "already current", version: "v1.0.0", tag: "v1.0.0"},
		{name: "the binary was replaced", tarGz: true, version: "v1.0.0", tag: "v2.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tarGz {
				skipUnlessTarGz(t)
			}
			bin := isolatePluginRun(t)
			log := writeHostFixture(t, bin, "claude", 1)
			stubPluginEvidence(t, listedClaude())

			srv := releaseArchiveServer(t, releaseFixture{tag: tt.tag})
			out, execErr := runUpdateWithPluginStep(t, tt.version, srv, fakeBinary(t), nil)
			if execErr != nil {
				t.Fatalf("a failed plugin step changed the exit code: %v\n%s", execErr, out)
			}
			if runs := fixtureRuns(t, log); len(runs) != 1 {
				t.Fatalf("the host CLI ran %d times %v, want the sequence stopped at the failure", len(runs), runs)
			}
			// The failure is silent to the exit code, not to the user: the exact
			// command has to be there to rerun — updating-the-plugin.spec §12.
			if !strings.Contains(out, "claude plugin marketplace update archcore-plugins") {
				t.Errorf("the step swallowed the failing command line:\n%s", out)
			}
		})
	}
}

// TestUpdatePluginStepSendsNoTelemetryEvent covers requirement 14 and the
// one-event-per-invocation invariant of cli-update-telemetry.spec. The step is
// the third thing in this command that could send an event, and the invariant
// holds only because it sends none.
func TestUpdatePluginStepSendsNoTelemetryEvent(t *testing.T) {
	tests := []struct {
		name       string
		tarGz      bool
		tag        string
		wantEvents int
	}{
		{name: "already current", tag: "v1.0.0", wantEvents: 0},
		{name: "the binary was replaced", tarGz: true, tag: "v2.0.0", wantEvents: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tarGz {
				skipUnlessTarGz(t)
			}
			isolateTelemetryEnv(t)
			bin := isolatePluginRun(t)
			log := writeHostFixture(t, bin, "claude", 0)
			stubPluginEvidence(t, listedClaude())

			rec := newTelemetryRecorder(t, http.StatusOK)
			srv := releaseArchiveServer(t, releaseFixture{tag: tt.tag})
			out, execErr := runUpdateWithPluginStep(t, "v1.0.0", srv, fakeBinary(t), rec.client(t, "v1.0.0"))
			if execErr != nil {
				t.Fatalf("the run exited nonzero: %v\n%s", execErr, out)
			}
			if runs := fixtureRuns(t, log); len(runs) != 2 {
				t.Fatalf("the step did not run, so this proves nothing about its events: %v", runs)
			}
			if n := rec.count(); n != tt.wantEvents {
				t.Errorf("the invocation sent %d event(s), want %d", n, tt.wantEvents)
			}
		})
	}
}

// TestUpdateIsSilentForAMachineWithoutThePlugin is the invariant the whole step
// is shaped around: a user who never installed the plugin sees no plugin output
// and pays no mutating host command.
//
// The evidence seam is left alone here. An empty PATH and an empty home ARE
// that machine, so the silence is produced by the real collector rather than
// asserted against a staged answer.
func TestUpdateIsSilentForAMachineWithoutThePlugin(t *testing.T) {
	isolatePluginRun(t)

	out, execErr := runUpdateWithPluginStep(t, "v1.0.0", currentReleaseServer(t, "v1.0.0"), "", nil)
	if execErr != nil {
		t.Fatalf("an already-current run must exit zero, got %v\n%s", execErr, out)
	}
	if step := strings.TrimSpace(pluginStepOutput(t, out)); step != "" {
		t.Errorf("a machine without the plugin was told about it:\n%s", step)
	}
}

// TestBackgroundUpdateTaskNamesNothingOfThePluginSurface closes the one gap the
// import graph cannot see.
//
// TestPackage_DoesNotLinkThePluginSurface in @internal/update holds the seam
// where the unattended policy lives, and holds it absolutely: that package
// cannot reach a host command because it does not link one. The closure the MCP
// trigger runs is the exception — it is built here in cmd, which links the
// plugin surface on purpose for the three entry points that are allowed to use
// it. Nothing structural stops a line in this one function from reaching the
// same code, and nothing would fail if it did: the trigger's own tests stub the
// policy and assert it ran once, which a closure that also installed a plugin
// would satisfy.
//
// The result would be mutating subprocesses on host CLIs — `claude plugin
// install`, `codex plugin marketplace upgrade` — sixty seconds into a session,
// on a machine with nobody watching, from a command the user started to serve
// documents. Both specs forbid it in the same words: the unattended update
// policy and the MCP trigger MUST NOT reach this surface.
//
// The rule is deliberately blunt — no mention of the plugin surface at all,
// under any spelling. This function resolves a version, waits, calls the policy
// and prints one line; there is no legitimate reason for the word to appear in
// it, so anything subtler than "not at all" would only be a rule with a hole in
// it.
func TestBackgroundUpdateTaskNamesNothingOfThePluginSurface(t *testing.T) {
	t.Parallel()

	const (
		file = "mcp.go"
		fn   = "backgroundUpdateTask"
	)

	decl := funcDeclIn(t, file, fn)

	ast.Inspect(decl, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok || !strings.Contains(strings.ToLower(id.Name), "plugin") {
			return true
		}
		t.Errorf("%s names %q: the unattended trigger must not reach the plugin surface — "+
			"updating-the-plugin.spec and plugin-delivery.spec, Constraints", fn, id.Name)
		return true
	})
}

// hostListings are the read-only commands the collector runs, one per host with
// a CLI. Cursor has none, which is why three names cover four hosts.
var hostListings = []string{
	"claude plugin list --json",
	"codex plugin list --json",
	"copilot plugin list",
}

// emptyListing and installedListing are what a host answers about itself. The
// text tier reads the JSON one as a line of fields and finds the marketplace id
// in it, so one payload serves every host.
const (
	emptyListing     = "[]"
	installedListing = `[{"id":"archcore@archcore-plugins","version":"1.4.0"}]`
)

// TestUpdatePluginStepCostsANonInstallerOnlyItsListings is the release criterion
// counted rather than read: a user who never installed the plugin pays no
// mutating host command.
//
// It differs from TestUpdateIsSilentForAMachineWithoutThePlugin in the machine
// it stages and in what it asserts. That test empties PATH, where no subprocess
// can start whatever the code does, and reads the output. This one puts every
// host CLI on PATH with the real collector in play — the case the invariant is
// actually about, since a machine with `claude` installed and no plugin is the
// common one — and counts what ran. Silence is not the evidence here: a step
// that ran `claude plugin update` against a plugin nobody installed would print
// its progress line, but a step that ran it and swallowed the line would pass an
// output assertion while having already changed the machine.
//
// The installed arm is what keeps the empty one honest. Two rows over one
// fixture separate "the step decided not to mutate" from "the step never got far
// enough to mutate", which a single row of zeroes cannot tell apart.
func TestUpdatePluginStepCostsANonInstallerOnlyItsListings(t *testing.T) {
	tests := []struct {
		name         string
		listing      string
		wantMutating int
	}{
		{
			name:    "no host has the plugin",
			listing: emptyListing,
		},
		{
			// Every host answers the same payload, so all three mutate: Claude
			// Code runs two commands, Codex CLI and Copilot one each.
			name:         "every host has the plugin",
			listing:      installedListing,
			wantMutating: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin := isolatePluginRun(t)
			log := filepath.Join(t.TempDir(), "hosts.log")
			for _, host := range []string{"claude", "codex", "copilot"} {
				writeListingHostFixture(t, bin, host, log, tt.listing)
			}

			out, execErr := runUpdateWithPluginStep(t, "v1.0.0", currentReleaseServer(t, "v1.0.0"), "", nil)
			if execErr != nil {
				t.Fatalf("an already-current run must exit zero, got %v\n%s", execErr, out)
			}

			readOnly, mutating := splitFixtureRuns(fixtureRuns(t, log), hostListings)

			// Asserted first: a run that asked nothing proves nothing about what
			// it then declined to do.
			if !slices.Equal(readOnly, hostListings) {
				t.Fatalf("the step ran %v, want one read-only listing per host with a CLI %v", readOnly, hostListings)
			}
			if len(mutating) != tt.wantMutating {
				t.Errorf("the step ran %d mutating command(s) %v, want %d", len(mutating), mutating, tt.wantMutating)
			}
		})
	}
}

// pluginUpdateMachine is the machine both entry points observe. Three hosts on
// three different tiers — one runs, one prints a command, one prints a UI note
// — so a plan that differs between the entry points differs visibly rather than
// in a corner.
func pluginUpdateMachine() []plugin.Evidence {
	return []plugin.Evidence{
		listedClaude(),
		{Host: plugin.HostCursor, RegistryListed: true},
		{Host: plugin.HostCopilot, RegistryListed: true},
	}
}

// pluginEvidenceFor returns the machine's evidence for exactly the hosts one
// entry point asked about.
//
// This is what makes the comparison below a real one. Plan is pure, so the
// actions an entry point produced are a function of the verb and the evidence
// it collected, and the host set is the only part of that an entry point
// chooses — an entry point that narrowed the selection would plan a shorter
// list here.
func pluginEvidenceFor(machine []plugin.Evidence, hosts []plugin.Host) []plugin.Evidence {
	out := make([]plugin.Evidence, 0, len(hosts))
	for _, ev := range machine {
		if slices.Contains(hosts, ev.Host) {
			out = append(out, ev)
		}
	}
	return out
}

// TestUpdateStepAndPluginUpdateCommandActIdentically is the invariant of
// plugin-delivery.spec: `archcore update`'s plugin step and
// `archcore plugin update` produce identical per-host actions. The plan/execute
// split is what makes it a test rather than a convention.
//
// It is checked twice over. The plans are compared deeply, which is the
// invariant as the spec words it; the subprocesses and the printed lines are
// compared too, because a plan two entry points agree on is worth nothing if
// one of them then executes it differently.
func TestUpdateStepAndPluginUpdateCommandActIdentically(t *testing.T) {
	bin := isolatePluginRun(t)
	machine := pluginUpdateMachine()

	stepLog := writeHostFixture(t, bin, "claude", 0)
	stepStub := stubPluginEvidence(t, machine...)
	stepRun, execErr := runUpdateWithPluginStep(t, "v1.0.0", currentReleaseServer(t, "v1.0.0"), "", nil)
	if execErr != nil {
		t.Fatalf("the update run exited nonzero: %v\n%s", execErr, stepRun)
	}
	stepOut := pluginStepOutput(t, stepRun)
	stepRuns := fixtureRuns(t, stepLog)

	// A fresh fixture, so the second run's invocations land in their own log.
	cmdLog := writeHostFixture(t, bin, "claude", 0)
	cmdStub := stubPluginEvidence(t, machine...)
	cmdOut, err := runPluginCmd(t, "update")
	if err != nil {
		t.Fatalf("`archcore plugin update` exited nonzero: %v\n%s", err, cmdOut)
	}
	cmdRuns := fixtureRuns(t, cmdLog)

	if !slices.Equal(stepStub.hosts, cmdStub.hosts) {
		t.Fatalf("the entry points observed different hosts: step %v, command %v", stepStub.hosts, cmdStub.hosts)
	}

	// The plan is computed once, as a statement about the fixture rather than
	// about the two entry points: plugin.Plan is pure, so planning the same
	// evidence twice and comparing the results proves only that a function
	// returns what it returns. The equivalence claim rests on what the two runs
	// actually did — the commands below and the lines they printed.
	//
	// Two empty plans would compare equal and prove nothing, so the staged
	// machine's three tiers are asserted to have survived into the plan. That is
	// what makes the comparisons that follow a real test rather than two silences
	// agreeing.
	plan := plugin.Plan(plugin.VerbUpdate, pluginEvidenceFor(machine, stepStub.hosts))
	if len(plan) != len(machine) {
		t.Fatalf("the plan addresses %d host(s), want the %d the machine carries", len(plan), len(machine))
	}

	if !slices.Equal(stepRuns, cmdRuns) {
		t.Errorf("the entry points ran different commands:\nstep:    %v\ncommand: %v", stepRuns, cmdRuns)
	}
	if stepOut != cmdOut {
		t.Errorf("the entry points printed different lines:\nstep:\n%s\ncommand:\n%s", stepOut, cmdOut)
	}
}
