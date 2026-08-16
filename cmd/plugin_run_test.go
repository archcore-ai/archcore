package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/plugin"
	"archcore-cli/internal/telemetry"
)

// Every test in this file swaps a package var or the process environment, so
// none of them calls t.Parallel().

// isolatePluginRun points HOME, the state directory and PATH at empty temp
// directories, and returns the directory PATH names.
//
// PATH is the load-bearing one. A plan for a listed host runs mutating commands
// through exec.LookPath, and the developer running these tests may well have
// `claude` installed: without an isolated PATH a test would install, update or
// uninstall a plugin on the machine it is running on.
func isolatePluginRun(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	bin := t.TempDir()
	t.Setenv("PATH", bin)
	return bin
}

// writeHostFixture puts a stand-in host CLI on PATH and returns the file it
// appends each invocation to. The log is what makes "ran nothing" provable:
// printed output alone cannot tell a command that was skipped from one that ran
// and said nothing.
func writeHostFixture(t *testing.T, bin, name string, exitCode int) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fixture host CLIs are #!/bin/sh scripts; CI runs Linux only")
	}
	log := filepath.Join(t.TempDir(), name+".log")
	script := "#!/bin/sh\necho \"$@\" >> " + log + "\nexit " + strconv.Itoa(exitCode) + "\n"
	if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755); err != nil {
		t.Fatalf("writing the %s fixture: %v", name, err)
	}
	return log
}

// writeListingHostFixture puts a stand-in host CLI on PATH that answers every
// invocation with listing on stdout, and appends each one to a log shared by
// every host in the run.
//
// One log for the whole machine is the point. writeHostFixture gives each host
// its own file, which answers "did this host run?"; the invariant that a
// non-installer pays no mutating command is about the machine, so it needs the
// total — a host whose commands nobody counted is exactly where a stray
// subprocess would hide.
//
// The payload is embedded in a single-quoted shell word, so it may not carry a
// single quote itself. Every host listing this file stages is JSON or a plain
// name list, neither of which needs one.
func writeListingHostFixture(t *testing.T, bin, name, log, listing string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fixture host CLIs are #!/bin/sh scripts; CI runs Linux only")
	}
	if strings.Contains(listing, "'") {
		t.Fatalf("the %s listing carries a single quote, which the fixture cannot embed: %q", name, listing)
	}
	script := "#!/bin/sh\necho \"" + name + " $@\" >> " + log + "\nprintf '%s' '" + listing + "'\nexit 0\n"
	if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755); err != nil {
		t.Fatalf("writing the %s fixture: %v", name, err)
	}
}

// splitFixtureRuns sorts a shared fixture log into the read-only listings and
// everything else. Everything else is a mutating command by construction: the
// host table gives each host exactly one listing, and every other entry in it
// installs, updates or removes.
//
// The mutating half is counted rather than matched, because Execute appends a
// host's non-interactive flag on a session with no terminal — so the exact
// command line depends on whether the suite was started from one. The listings
// never take the flag: they run through the collector, which does not apply it.
func splitFixtureRuns(runs []string, listings []string) (readOnly, mutating []string) {
	for _, run := range runs {
		if slices.Contains(listings, run) {
			readOnly = append(readOnly, run)
			continue
		}
		mutating = append(mutating, run)
	}
	return readOnly, mutating
}

// fixtureRuns returns one entry per invocation of a fixture host CLI.
func fixtureRuns(t *testing.T, log string) []string {
	t.Helper()
	data, err := os.ReadFile(log)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("reading the fixture log: %v", err)
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

// shrinkPluginStepBudget makes the step budget reachable inside a test. The
// real one is two minutes, which no test may spend.
func shrinkPluginStepBudget(t *testing.T, budget time.Duration) {
	t.Helper()
	original := pluginStepBudget
	pluginStepBudget = budget
	t.Cleanup(func() { pluginStepBudget = original })
}

// pluginEvidenceStub records what the run asked for and answers with what the
// test staged.
type pluginEvidenceStub struct {
	hosts    []plugin.Host
	evidence []plugin.Evidence
}

func stubPluginEvidence(t *testing.T, evidence ...plugin.Evidence) *pluginEvidenceStub {
	t.Helper()
	stub := &pluginEvidenceStub{evidence: evidence}
	original := collectPluginEvidence
	collectPluginEvidence = func(_ context.Context, hosts []plugin.Host) []plugin.Evidence {
		stub.hosts = hosts
		return stub.evidence
	}
	t.Cleanup(func() { collectPluginEvidence = original })
	return stub
}

// listedClaude is the evidence of a machine with Claude Code installed and the
// Archcore plugin in its own listing.
func listedClaude() plugin.Evidence {
	return plugin.Evidence{
		Host:          plugin.HostClaudeCode,
		CLIPresent:    true,
		ListingOK:     true,
		Listed:        true,
		ListedVersion: "1.4.0",
	}
}

// marketplaceEntry returns the archcore-plugins entry of a settings file, or nil
// when the file or the entry is absent.
func marketplaceEntry(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("reading the settings file: %v", err)
	}
	var settings struct {
		Marketplaces map[string]map[string]any `json:"extraKnownMarketplaces"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("the settings file is not valid JSON:\n%s", data)
	}
	return settings.Marketplaces[plugin.MarketplaceID]
}

// TestRunPluginActionsRunsAListedHostsCommands covers the tier that mutates:
// the host's own listing named the plugin, so the update runs, and every command
// is announced before it starts — updating-the-plugin.spec §6 and §11.
func TestRunPluginActionsRunsAListedHostsCommands(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, listedClaude())

	var out strings.Builder
	outcome := runPluginActions(t.Context(), &out, pluginRunOptions{
		Verb:  plugin.VerbUpdate,
		Hosts: []plugin.Host{plugin.HostClaudeCode},
	})

	if outcome.Failed {
		t.Errorf("a run whose commands all exited zero reported a failure:\n%s", out.String())
	}
	if runs := fixtureRuns(t, log); len(runs) != 2 {
		t.Errorf("the host CLI ran %d times %v, want both update commands", len(runs), runs)
	}
	// The progress lines are matched without the non-interactive flag: Execute
	// appends it only on a session with no terminal, and a test that pins it
	// passes or fails depending on where it ran.
	for _, want := range []string{
		"Claude Code: claude plugin marketplace update archcore-plugins",
		"Claude Code: claude plugin update archcore@archcore-plugins",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output is missing the progress line %q:\n%s", want, out.String())
		}
	}
}

// TestRunPluginActionsIsSilentForAHostWithoutThePlugin is the invariant the
// whole surface is shaped around: a user who never installed the plugin sees no
// output and pays no mutating host command — updating-the-plugin.spec §7.
func TestRunPluginActionsIsSilentForAHostWithoutThePlugin(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, plugin.Evidence{
		Host:       plugin.HostClaudeCode,
		CLIPresent: true,
		ListingOK:  true,
	})

	var out strings.Builder
	outcome := runPluginActions(t.Context(), &out, pluginRunOptions{
		Verb:  plugin.VerbUpdate,
		Hosts: []plugin.Host{plugin.HostClaudeCode},
	})

	if out.String() != "" {
		t.Errorf("a host without the plugin printed:\n%s", out.String())
	}
	if runs := fixtureRuns(t, log); len(runs) != 0 {
		t.Errorf("a host without the plugin ran %v", runs)
	}
	if len(outcome.Results) != 0 || outcome.Failed {
		t.Errorf("outcome = %+v, want no action and no failure", outcome)
	}
}

// TestRunPluginActionsReportsAFailedCommand covers Failure Behavior 2: the exact
// command line is printed and the run continues. The second command of the
// sequence never starts, because it depends on the one that failed.
func TestRunPluginActionsReportsAFailedCommand(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 1)
	stubPluginEvidence(t, listedClaude())

	var out strings.Builder
	outcome := runPluginActions(t.Context(), &out, pluginRunOptions{
		Verb:  plugin.VerbUpdate,
		Hosts: []plugin.Host{plugin.HostClaudeCode},
	})

	if !outcome.Failed {
		t.Error("a nonzero exit did not fail the run")
	}
	if !strings.Contains(out.String(), "claude plugin marketplace update archcore-plugins") {
		t.Errorf("the failure line names no command:\n%s", out.String())
	}
	if runs := fixtureRuns(t, log); len(runs) != 1 {
		t.Errorf("the host CLI ran %d times %v, want the sequence stopped at the first failure", len(runs), runs)
	}
}

// TestRunPluginActionsCountsOnlyAttemptedActionsAsFailures separates the two
// outcomes requirements 18 and 19 of plugin-delivery.spec pull apart: a host
// whose command failed fails the run, a host skipped for missing evidence does
// not, and the run reaches the second host either way.
func TestRunPluginActionsCountsOnlyAttemptedActionsAsFailures(t *testing.T) {
	bin := isolatePluginRun(t)
	writeHostFixture(t, bin, "claude", 1)
	stubPluginEvidence(t,
		listedClaude(),
		plugin.Evidence{Host: plugin.HostCopilot, CLIPresent: true, ListingOK: true},
	)

	var out strings.Builder
	outcome := runPluginActions(t.Context(), &out, pluginRunOptions{Verb: plugin.VerbUpdate})

	if !outcome.Failed {
		t.Error("a nonzero exit did not fail the run")
	}
	if len(outcome.Results) != 1 || outcome.Results[0].Host != plugin.HostClaudeCode {
		t.Errorf("results = %+v, want the skipped host to produce none", outcome.Results)
	}
	if strings.Contains(out.String(), "Copilot") {
		t.Errorf("the skipped host printed a line:\n%s", out.String())
	}
}

// TestRunPluginActionsPrintOnlyStartsNoSubprocess is what makes the CI and
// non-interactive tiers real — requirements 7 and 8 of plugin-delivery.spec ask
// for printed commands and nothing executed, and printing the same line a run
// would print is not evidence that nothing ran.
func TestRunPluginActionsPrintOnlyStartsNoSubprocess(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, listedClaude())

	var out strings.Builder
	outcome := runPluginActions(t.Context(), &out, pluginRunOptions{
		Verb:      plugin.VerbUpdate,
		Hosts:     []plugin.Host{plugin.HostClaudeCode},
		PrintOnly: true,
	})

	if runs := fixtureRuns(t, log); len(runs) != 0 {
		t.Errorf("a printing run started %v", runs)
	}
	if outcome.Failed {
		t.Error("a printing run reported a failure")
	}
	if !strings.Contains(out.String(), "claude plugin update archcore@archcore-plugins") {
		t.Errorf("the printed commands are missing:\n%s", out.String())
	}
}

// TestRunPluginActionsSpendsOneBudgetOnBothPhases pins the ceiling the step
// actually has. CollectEvidence asks each host CLI before Execute starts, so a
// bound that covers only the second phase lets the step cost twice what both
// specs allow it — updating-the-plugin.spec, Surface. A collector that spends
// the whole budget must therefore leave the executor an expired context and no
// host to run on.
func TestRunPluginActionsSpendsOneBudgetOnBothPhases(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	shrinkPluginStepBudget(t, 20*time.Millisecond)

	// stallPluginEvidence is this lever, and its doc comment already names this
	// test as its user; the body was written out here a second time.
	stallPluginEvidence(t, listedClaude())

	var out strings.Builder
	outcome := runPluginActions(t.Context(), &out, pluginRunOptions{
		Verb:  plugin.VerbUpdate,
		Hosts: []plugin.Host{plugin.HostClaudeCode},
	})

	if runs := fixtureRuns(t, log); len(runs) != 0 {
		t.Errorf("the executor ran %v on a budget the evidence phase had already spent", runs)
	}
	if len(outcome.Results) != 0 || outcome.Failed {
		t.Errorf("outcome = %+v, want no result and no failure for a host never reached", outcome)
	}
	if out.String() != "" {
		t.Errorf("a host the bound cut off printed:\n%s", out.String())
	}
}

// TestRunPluginActionsWritesNoSettingsForAHostItNeverReached keeps the
// marketplace entry tied to an install that happened. A run cut short by the
// step bound returns fewer results than it planned actions, and the merge reads
// the Result rather than the plan: an entry declaring autoUpdate for a plugin
// nobody installed points Claude Code at a marketplace it never got.
func TestRunPluginActionsWritesNoSettingsForAHostItNeverReached(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, plugin.Evidence{Host: plugin.HostClaudeCode, CLIPresent: true})
	settings := filepath.Join(t.TempDir(), ".claude", "settings.json")

	// A cancelled parent reaches the same line the elapsed bound does: Execute
	// checks ctx.Err() before the first action and returns the empty result list
	// the merge then has to read.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var out strings.Builder
	outcome := runPluginActions(ctx, &out, pluginRunOptions{
		Verb:         plugin.VerbInstall,
		Hosts:        []plugin.Host{plugin.HostClaudeCode},
		SettingsPath: settings,
	})

	if _, err := os.Stat(settings); err == nil {
		t.Error("a host the run never reached had its marketplace entry written anyway")
	}
	if runs := fixtureRuns(t, log); len(runs) != 0 {
		t.Errorf("a host the run never reached ran %v", runs)
	}
	// Requirement 19: nothing was attempted here, so there is nothing to fail.
	if outcome.Failed {
		t.Errorf("a host the step bound cut off failed the run:\n%s", out.String())
	}
}

// TestRunPluginActionsMergesTheSettingsEntryAfterAnInstall covers requirement 14
// of plugin-delivery.spec. The write happens here rather than in the executor,
// which words nothing and performs no settings merge.
func TestRunPluginActionsMergesTheSettingsEntryAfterAnInstall(t *testing.T) {
	bin := isolatePluginRun(t)
	writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, plugin.Evidence{Host: plugin.HostClaudeCode, CLIPresent: true})
	settings := filepath.Join(t.TempDir(), ".claude", "settings.json")

	var out strings.Builder
	outcome := runPluginActions(t.Context(), &out, pluginRunOptions{
		Verb:         plugin.VerbInstall,
		Hosts:        []plugin.Host{plugin.HostClaudeCode},
		SettingsPath: settings,
	})

	if outcome.Failed {
		t.Errorf("the install reported a failure:\n%s", out.String())
	}
	if entry := marketplaceEntry(t, settings); entry["autoUpdate"] != true {
		t.Errorf("entry = %+v, want autoUpdate enabled", entry)
	}
}

// TestRunPluginActionsSkipsTheSettingsEntryForAFailedInstall keeps the
// declaration and the plugin in step: a marketplace declared with autoUpdate for
// a plugin that failed to install points the host at nothing.
func TestRunPluginActionsSkipsTheSettingsEntryForAFailedInstall(t *testing.T) {
	bin := isolatePluginRun(t)
	writeHostFixture(t, bin, "claude", 1)
	stubPluginEvidence(t, plugin.Evidence{Host: plugin.HostClaudeCode, CLIPresent: true})
	settings := filepath.Join(t.TempDir(), ".claude", "settings.json")

	var out strings.Builder
	outcome := runPluginActions(t.Context(), &out, pluginRunOptions{
		Verb:         plugin.VerbInstall,
		Hosts:        []plugin.Host{plugin.HostClaudeCode},
		SettingsPath: settings,
	})

	if !outcome.Failed {
		t.Error("a failed install did not fail the run")
	}
	if _, err := os.Stat(settings); err == nil {
		t.Error("a failed install wrote the marketplace entry anyway")
	}
}

// TestRunPluginActionsReportsAFailedSettingsMerge covers Failure Behavior 3: the
// merge failure is reported, and the install that already completed keeps its
// result. Undoing it would uninstall a working plugin over a settings file.
func TestRunPluginActionsReportsAFailedSettingsMerge(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, plugin.Evidence{Host: plugin.HostClaudeCode, CLIPresent: true})

	// A directory where the settings file belongs: the read fails, so nothing is
	// written and nothing is backed up.
	settings := filepath.Join(t.TempDir(), "settings.json")
	if err := os.MkdirAll(settings, 0o755); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	outcome := runPluginActions(t.Context(), &out, pluginRunOptions{
		Verb:         plugin.VerbInstall,
		Hosts:        []plugin.Host{plugin.HostClaudeCode},
		SettingsPath: settings,
	})

	// Matched on the failure's own words, not on the host name: the reporter
	// prints "  Claude Code: claude plugin marketplace add …" for every command
	// it runs, so a host-name match passes on a run that reported nothing at
	// all. "is a directory" is what the wrapped read error renders as.
	if !strings.Contains(out.String(), "is a directory") {
		t.Errorf("the merge failure was not reported:\n%s", out.String())
	}
	if !outcome.Failed {
		t.Error("a failed settings merge left the run reporting success")
	}
	if len(outcome.Results) != 1 || outcome.Results[0].Failed {
		t.Errorf("results = %+v, want the completed install left as it was", outcome.Results)
	}
	if runs := fixtureRuns(t, log); len(runs) != 2 {
		t.Errorf("the install ran %d commands %v, want both of them", len(runs), runs)
	}
}

// TestRunPluginActionsPrintOnlyWritesNoSettings keeps the printing tier from
// mutating a file archcore does not own. Printing what an install would do must
// leave the machine as it was.
func TestRunPluginActionsPrintOnlyWritesNoSettings(t *testing.T) {
	bin := isolatePluginRun(t)
	writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, plugin.Evidence{Host: plugin.HostClaudeCode, CLIPresent: true})
	settings := filepath.Join(t.TempDir(), ".claude", "settings.json")

	runPluginActions(t.Context(), &strings.Builder{}, pluginRunOptions{
		Verb:         plugin.VerbInstall,
		Hosts:        []plugin.Host{plugin.HostClaudeCode},
		SettingsPath: settings,
		PrintOnly:    true,
	})

	if _, err := os.Stat(settings); err == nil {
		t.Error("a printing run wrote the Claude Code settings file")
	}
}

// TestRunPluginActionsHealsTheSettingsEntryOnAReportedInstall covers the case
// the planner carries the merge flag through: an install over a plugin the host
// already lists runs no command, and the entry a failed earlier write never
// produced is still written.
func TestRunPluginActionsHealsTheSettingsEntryOnAReportedInstall(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, listedClaude())
	settings := filepath.Join(t.TempDir(), ".claude", "settings.json")

	var out strings.Builder
	outcome := runPluginActions(t.Context(), &out, pluginRunOptions{
		Verb:         plugin.VerbInstall,
		Hosts:        []plugin.Host{plugin.HostClaudeCode},
		SettingsPath: settings,
	})

	if outcome.Failed {
		t.Errorf("the reported install failed:\n%s", out.String())
	}
	if runs := fixtureRuns(t, log); len(runs) != 0 {
		t.Errorf("an install over a listed plugin ran %v, want no command at all", runs)
	}
	if !strings.Contains(out.String(), "already installed") {
		t.Errorf("the no-op was not reported:\n%s", out.String())
	}
	if entry := marketplaceEntry(t, settings); entry["autoUpdate"] != true {
		t.Errorf("entry = %+v, want autoUpdate enabled", entry)
	}
}

// TestRunPluginActionsTakesTheSettingsEntryBackOnRemoval covers requirement 17:
// removal undoes what this surface wrote.
func TestRunPluginActionsTakesTheSettingsEntryBackOnRemoval(t *testing.T) {
	bin := isolatePluginRun(t)
	writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, listedClaude())

	settings := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(settings, []byte(`{"model":"opus","extraKnownMarketplaces":{"archcore-plugins":{"autoUpdate":true}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	outcome := runPluginActions(t.Context(), &out, pluginRunOptions{
		Verb:         plugin.VerbRemove,
		Hosts:        []plugin.Host{plugin.HostClaudeCode},
		SettingsPath: settings,
	})

	if outcome.Failed {
		t.Errorf("the removal reported a failure:\n%s", out.String())
	}
	if entry := marketplaceEntry(t, settings); entry != nil {
		t.Errorf("entry = %+v, want it taken back", entry)
	}
}

// TestRunPluginActionsReportsAFailedSettingsRemoval is the failure half of the
// branch above, and the twin of TestRunPluginActionsReportsAFailedSettingsMerge
// on the install side. Only the install half had a test: a removal whose
// settings write fails must report it and turn the run's answer failed, the same
// way, because requirement 17 makes taking the entry back part of the removal.
func TestRunPluginActionsReportsAFailedSettingsRemoval(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, listedClaude())

	// A directory where the settings file belongs: the read fails, so the entry
	// cannot be taken back.
	settings := filepath.Join(t.TempDir(), "settings.json")
	if err := os.MkdirAll(settings, 0o755); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	outcome := runPluginActions(t.Context(), &out, pluginRunOptions{
		Verb:         plugin.VerbRemove,
		Hosts:        []plugin.Host{plugin.HostClaudeCode},
		SettingsPath: settings,
	})

	if !strings.Contains(out.String(), "is a directory") {
		t.Errorf("the failed removal was not reported:\n%s", out.String())
	}
	if !outcome.Failed {
		t.Error("a failed settings removal left the run reporting success")
	}
	// The host's own uninstall still ran and still succeeded: the settings write
	// is a second step, and its failure does not rewrite what the host did.
	if len(outcome.Results) != 1 || outcome.Results[0].Failed {
		t.Errorf("results = %+v, want the completed uninstall left as it was", outcome.Results)
	}
	if runs := fixtureRuns(t, log); len(runs) != 1 {
		t.Errorf("the removal ran %d commands %v, want the host uninstall", len(runs), runs)
	}
}

// TestRunPluginActionsPrintsTheUINoteForAHostWithoutACLI covers requirement 5:
// Cursor manages plugins from its UI, so it gets a sentence and never a command.
func TestRunPluginActionsPrintsTheUINoteForAHostWithoutACLI(t *testing.T) {
	isolatePluginRun(t)
	stubPluginEvidence(t, plugin.Evidence{Host: plugin.HostCursor})

	var out strings.Builder
	outcome := runPluginActions(t.Context(), &out, pluginRunOptions{
		Verb:  plugin.VerbInstall,
		Hosts: []plugin.Host{plugin.HostCursor},
	})

	if outcome.Failed {
		t.Error("a printed instruction failed the run")
	}
	if !strings.Contains(out.String(), "Cursor plugin marketplace") {
		t.Errorf("the Cursor instruction is missing:\n%s", out.String())
	}
}

// TestRunPluginActionsCollectsEveryPluginHostByDefault keeps an empty host
// selection meaning "every host that ships a plugin" rather than "none".
func TestRunPluginActionsCollectsEveryPluginHostByDefault(t *testing.T) {
	isolatePluginRun(t)
	stub := stubPluginEvidence(t)

	runPluginActions(t.Context(), &strings.Builder{}, pluginRunOptions{Verb: plugin.VerbStatus})

	if !slices.Equal(stub.hosts, plugin.Hosts()) {
		t.Errorf("collected evidence for %v, want %v", stub.hosts, plugin.Hosts())
	}
}

// TestPluginStatusTextReportsTheEvidence covers requirement 16: the presence
// answer, which evidence produced it, and the version when the host reports one.
// The unanswered listing is the row that matters — the collector reads it as
// "not installed" for every mutating tier, and a status report that repeated
// that would state a fact it does not have.
func TestPluginStatusTextReportsTheEvidence(t *testing.T) {
	tests := []struct {
		name     string
		evidence plugin.Evidence
		want     string
	}{
		{
			name:     "listed with a version",
			evidence: listedClaude(),
			want:     "installed 1.4.0",
		},
		{
			name:     "listed without a version",
			evidence: plugin.Evidence{CLIPresent: true, ListingOK: true, Listed: true},
			want:     "installed (",
		},
		{
			name:     "the host answered and does not have it",
			evidence: plugin.Evidence{CLIPresent: true, ListingOK: true},
			want:     "not installed (",
		},
		{
			name:     "the host did not answer",
			evidence: plugin.Evidence{CLIPresent: true},
			want:     "unknown",
		},
		{
			name:     "no host CLI, but the registry names it",
			evidence: plugin.Evidence{RegistryListed: true},
			want:     "installed (on-disk registry",
		},
		{
			name: "no evidence at all",
			want: "not installed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pluginStatusText(tt.evidence); !strings.HasPrefix(got, tt.want) {
				t.Errorf("pluginStatusText(%+v) = %q, want it to start with %q", tt.evidence, got, tt.want)
			}
		})
	}
}

// TestPluginHostsForAgentNamesOnlyThePluginHosts covers the Constraint of
// plugin-delivery.spec: an agent without a shipping plugin is a valid agent
// everywhere else in the CLI, so the error may not answer with the registry.
func TestPluginHostsForAgentNamesOnlyThePluginHosts(t *testing.T) {
	if hosts, err := pluginHostsForAgent(""); err != nil || !slices.Equal(hosts, plugin.Hosts()) {
		t.Errorf("pluginHostsForAgent(\"\") = %v, %v, want every plugin host", hosts, err)
	}
	if hosts, err := pluginHostsForAgent("codex-cli"); err != nil || !slices.Equal(hosts, []plugin.Host{plugin.HostCodexCLI}) {
		t.Errorf("pluginHostsForAgent(\"codex-cli\") = %v, %v, want the one host", hosts, err)
	}

	for _, id := range []string{"gemini-cli", "opencode", "not-an-agent"} {
		t.Run(id, func(t *testing.T) {
			_, err := pluginHostsForAgent(id)
			if err == nil {
				t.Fatalf("pluginHostsForAgent(%q) returned no error", id)
			}
			for _, host := range plugin.Hosts() {
				if !strings.Contains(err.Error(), string(host)) {
					t.Errorf("the error does not name %q: %v", host, err)
				}
			}
			// The registry names every other agent; naming one here would offer a
			// selection this surface cannot act on.
			if strings.Contains(err.Error(), "roo-code") || strings.Contains(err.Error(), "cline") {
				t.Errorf("the error names an agent that ships no plugin: %v", err)
			}
		})
	}
}

// TestClaudeSettingsPathScopes covers requirement 12: user scope is the default,
// and only `--scope project` writes into the repository.
func TestClaudeSettingsPathScopes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	got, err := claudeSettingsPath("/somewhere/else", false)
	if err != nil {
		t.Fatalf("user scope: %v", err)
	}
	if want := filepath.Join(home, ".claude", "settings.json"); got != want {
		t.Errorf("user scope = %q, want %q", got, want)
	}

	root := t.TempDir()
	got, err = claudeSettingsPath(root, true)
	if err != nil {
		t.Fatalf("project scope: %v", err)
	}
	if want := filepath.Join(root, ".claude", "settings.json"); got != want {
		t.Errorf("project scope = %q, want %q", got, want)
	}

	if _, err := claudeSettingsPath("", true); err == nil {
		t.Error("project scope with no project root was accepted")
	}
}

// TestPluginRunningInCI reads the same six variables the unattended policy
// enumerates. Requirement 8 of plugin-delivery.spec turns a CI run into printed
// commands, so a variable this misses is a mutating host command on a runner.
func TestPluginRunningInCI(t *testing.T) {
	for _, name := range telemetry.CIVars {
		t.Run(name, func(t *testing.T) {
			for _, other := range telemetry.CIVars {
				t.Setenv(other, "")
			}
			if pluginRunningInCI() {
				t.Fatal("a machine with no CI variable set reads as CI")
			}
			t.Setenv(name, "1")
			if !pluginRunningInCI() {
				t.Errorf("%s set does not read as CI", name)
			}
		})
	}
}

// --- the hosts the step bound never reached --------------------------------

// stallPluginEvidence answers with the staged evidence only once the step budget
// has elapsed, which is what leaves the executor an expired context and every
// planned host unreached.
//
// It is the lever TestRunPluginActionsSpendsOneBudgetOnBothPhases uses, for the
// same reason: the budget is a package var precisely because no test may spend
// the real two minutes. Waiting the budget out rather than sleeping a guessed
// interval costs exactly the budget, and the second arm turns a run that hands
// the collector no bound at all into a failure instead of a hang.
func stallPluginEvidence(t *testing.T, evidence ...plugin.Evidence) {
	t.Helper()
	original := collectPluginEvidence
	collectPluginEvidence = func(ctx context.Context, _ []plugin.Host) []plugin.Evidence {
		select {
		case <-ctx.Done():
		case <-time.After(2 * time.Second):
			t.Error("the run collected evidence under a context the step budget does not bound")
		}
		return evidence
	}
	t.Cleanup(func() { collectPluginEvidence = original })
}

// TestUnreachedHostsFallBackToPrintedCommandsForInstallOnly is the two specs
// answering one event with different verbs, and the entry point choosing between
// them: the Constraints of plugin-delivery.spec fall back to printed commands
// for a host the delivery step did not reach, and Failure Behavior 5 of
// updating-the-plugin.spec prints nothing for the hosts an update skipped.
//
// Each case stages the machine its own verb plans a command for — an install
// needs a host whose listing does not name the plugin, an update needs one whose
// listing does — because no single machine puts both verbs on their
// command-carrying tier. The bound then cuts the run short of every host, so the
// only thing that separates the four entry points is the fallback.
func TestUnreachedHostsFallBackToPrintedCommandsForInstallOnly(t *testing.T) {
	tests := []struct {
		name         string
		evidence     plugin.Evidence
		run          func(t *testing.T) string
		wantHandover bool
	}{
		{
			name:     "`archcore plugin install`",
			evidence: claudeOnPATH(),
			run: func(t *testing.T) string {
				out, err := runPluginCmd(t, "install", "--agent", "claude-code")
				// Requirement 19: nothing was attempted, so nothing failed.
				if err != nil {
					t.Fatalf("a host the bound cut off failed the run: %v\n%s", err, out)
				}
				return out
			},
			wantHandover: true,
		},
		{
			name:     "the delivery step of `archcore init`",
			evidence: claudeOnPATH(),
			run: func(t *testing.T) string {
				var out strings.Builder
				deliverPlugins(t.Context(), &out, []*agents.Agent{agents.ByID(agents.ClaudeCode)}, false)
				return out.String()
			},
			wantHandover: true,
		},
		{
			name:     "`archcore plugin update`",
			evidence: listedClaude(),
			run: func(t *testing.T) string {
				out, err := runPluginCmd(t, "update", "--agent", "claude-code")
				if err != nil {
					t.Fatalf("a host the bound cut off failed the run: %v\n%s", err, out)
				}
				return out
			},
		},
		{
			name:     "the plugin step of `archcore update`",
			evidence: listedClaude(),
			run: func(t *testing.T) string {
				var out strings.Builder
				runPluginUpdateStep(t.Context(), &out)
				return out.String()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withoutCIVars(t)
			bin := isolatePluginRun(t)
			log := writeHostFixture(t, bin, "claude", 0)
			shrinkPluginStepBudget(t, 20*time.Millisecond)
			stallPluginEvidence(t, tt.evidence)

			out := tt.run(t)

			// The premise of the whole case: the bound really did cut the run
			// short of the host, rather than the plan simply being empty.
			if runs := fixtureRuns(t, log); len(runs) != 0 {
				t.Fatalf("a host the bound was meant to cut off ran %v", runs)
			}
			if !tt.wantHandover {
				if out != "" {
					t.Errorf("an update printed for a host it never reached:\n%s", out)
				}
				return
			}
			for _, want := range []string{
				"yourself",
				"claude plugin marketplace add " + plugin.RepoID,
				"claude plugin install " + plugin.PluginID,
			} {
				if !strings.Contains(out, want) {
					t.Errorf("the handover is missing %q:\n%s", want, out)
				}
			}
		})
	}
}

// TestInstallHandsOverNoCommandsForAHostItReached keeps the fallback tied to the
// hosts nothing was attempted on. Host is the correlation key, so a fallback
// that walked the plan instead of the results would hand a user the commands for
// an install that had just completed in front of them.
func TestInstallHandsOverNoCommandsForAHostItReached(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, claudeOnPATH())

	var out strings.Builder
	outcome := runPluginActions(t.Context(), &out, pluginRunOptions{
		Verb:                   plugin.VerbInstall,
		Hosts:                  []plugin.Host{plugin.HostClaudeCode},
		SettingsPath:           filepath.Join(t.TempDir(), ".claude", "settings.json"),
		PrintUnreachedCommands: true,
	})

	if outcome.Failed {
		t.Errorf("an install whose commands all exited zero reported a failure:\n%s", out.String())
	}
	if runs := fixtureRuns(t, log); len(runs) != 2 {
		t.Fatalf("the host CLI ran %d times %v, want both install commands", len(runs), runs)
	}
	if strings.Contains(out.String(), "yourself") {
		t.Errorf("a host the run installed on had its commands handed over as well:\n%s", out.String())
	}
}

// --- the overlap this process caused ----------------------------------------

// stageInstalledPluginCache creates the on-disk state the overlap notice detects.
// The fixture host CLI installs no plugin of its own, so what a real install
// would leave behind is staged here.
func stageInstalledPluginCache(t *testing.T) {
	t.Helper()
	cache := filepath.Join(os.Getenv("HOME"), ".claude", "plugins", "archcore@archcore-plugins")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatalf("staging the installed plugin cache: %v", err)
	}
}

// selfCausedOverlapNotice is the sentence only DescribeSelfCausedPluginConflict
// produces. The wording `doctor` and `hooks install` print for a plugin this
// process did not touch says "Until it is updated" instead, so matching on the
// restart sentence tells the two notices apart.
const selfCausedOverlapNotice = "Restart the host session"

// TestEveryMutatingEntryPointReportsTheOverlapItCaused covers §15 of
// plugin-delivery.spec for all four entry points. The notice exists because the
// host session the user already has open now runs two sets of hooks, and that is
// true whichever command installed or updated the plugin — the person who typed
// `archcore plugin install` is the most likely of them to be sitting in front of
// a live session.
//
// The count is asserted rather than the presence: the overlap is one fact about
// the machine, so a run over four hosts owes one line, not four.
func TestEveryMutatingEntryPointReportsTheOverlapItCaused(t *testing.T) {
	tests := []struct {
		name     string
		evidence plugin.Evidence
		run      func(t *testing.T) string
	}{
		{
			name:     "`archcore plugin install`",
			evidence: claudeOnPATH(),
			run: func(t *testing.T) string {
				out, err := runPluginCmd(t, "install", "--agent", "claude-code")
				if err != nil {
					t.Fatalf("the install exited nonzero: %v\n%s", err, out)
				}
				return out
			},
		},
		{
			name:     "the delivery step of `archcore init`",
			evidence: claudeOnPATH(),
			run: func(t *testing.T) string {
				var out strings.Builder
				deliverPlugins(t.Context(), &out, []*agents.Agent{agents.ByID(agents.ClaudeCode)}, false)
				return out.String()
			},
		},
		{
			name:     "`archcore plugin update`",
			evidence: listedClaude(),
			run: func(t *testing.T) string {
				out, err := runPluginCmd(t, "update", "--agent", "claude-code")
				if err != nil {
					t.Fatalf("the update exited nonzero: %v\n%s", err, out)
				}
				return out
			},
		},
		{
			name:     "the plugin step of `archcore update`",
			evidence: listedClaude(),
			run: func(t *testing.T) string {
				var out strings.Builder
				runPluginUpdateStep(t.Context(), &out)
				return out.String()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withoutCIVars(t)
			bin := isolatePluginRun(t)
			writeHostFixture(t, bin, "claude", 0)
			stubPluginEvidence(t, tt.evidence)
			stageInstalledPluginCache(t)

			out := tt.run(t)

			if got := strings.Count(out, selfCausedOverlapNotice); got != 1 {
				t.Errorf("printed the self-caused overlap notice %d times, want exactly 1:\n%s", got, out)
			}
		})
	}
}

// TestTheSelfCausedOverlapNoticeNeedsAMutationThatSucceeded holds the other half
// of §15: the notice reports a change this process made, so everything that
// changed nothing stays quiet. Every case stages the cache the detection finds,
// which is what makes the silence a decision rather than a miss.
//
// Removal is the mutation that still says nothing. It ends the overlap instead
// of starting one, and the sentence — "installed and current ... restart the
// host session" — would be false the moment it printed.
func TestTheSelfCausedOverlapNoticeNeedsAMutationThatSucceeded(t *testing.T) {
	tests := []struct {
		name     string
		evidence plugin.Evidence
		exitCode int
		run      func(t *testing.T) string
	}{
		{
			name:     "an install over a plugin the host already lists",
			evidence: listedClaude(),
			run: func(t *testing.T) string {
				out, err := runPluginCmd(t, "install", "--agent", "claude-code")
				if err != nil {
					t.Fatalf("a reported no-op exited nonzero: %v\n%s", err, out)
				}
				return out
			},
		},
		{
			name:     "an install whose command failed",
			evidence: claudeOnPATH(),
			exitCode: 1,
			run: func(t *testing.T) string {
				out, err := runPluginCmd(t, "install", "--agent", "claude-code")
				if err == nil {
					t.Fatalf("a failed install exited zero:\n%s", out)
				}
				return out
			},
		},
		{
			name:     "a delivery step that only printed the commands",
			evidence: claudeOnPATH(),
			run: func(t *testing.T) string {
				var out strings.Builder
				deliverPlugins(t.Context(), &out, []*agents.Agent{agents.ByID(agents.ClaudeCode)}, true)
				return out.String()
			},
		},
		{
			name:     "a removal",
			evidence: listedClaude(),
			run: func(t *testing.T) string {
				out, err := runPluginCmd(t, "remove", "--agent", "claude-code")
				if err != nil {
					t.Fatalf("the removal exited nonzero: %v\n%s", err, out)
				}
				return out
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withoutCIVars(t)
			bin := isolatePluginRun(t)
			writeHostFixture(t, bin, "claude", tt.exitCode)
			stubPluginEvidence(t, tt.evidence)
			stageInstalledPluginCache(t)

			if out := tt.run(t); strings.Contains(out, selfCausedOverlapNotice) {
				t.Errorf("a run that installed nothing new reported an overlap it did not cause:\n%s", out)
			}
		})
	}
}

// TestUnreachedCursorHandsOverNothing covers the guard the case above never
// reaches: an action carrying no command has nothing to hand over.
//
// Cursor manages plugins from its UI, so its install action is a printed note
// and its Commands slice is empty. Without the len(action.Commands) == 0 check
// in printUnreachedCommands the fallback would print "Cursor: run this command
// yourself:" followed by nothing at all — a handover naming no command, on the
// one host that has none to give.
//
// Claude Code is staged alongside it so the run is not empty: its commands must
// still be handed over, which is what separates "the guard skipped Cursor" from
// "the fallback printed nothing for anyone".
func TestUnreachedCursorHandsOverNothing(t *testing.T) {
	withoutCIVars(t)
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	shrinkPluginStepBudget(t, 20*time.Millisecond)
	stallPluginEvidence(t,
		claudeOnPATH(),
		// Cursor has no CLI, so the registry is its whole evidence.
		plugin.Evidence{Host: plugin.HostCursor, RegistryListed: true},
	)

	out, err := runPluginCmd(t, "install")
	if err != nil {
		t.Fatalf("a host the bound cut off failed the run: %v\n%s", err, out)
	}
	if runs := fixtureRuns(t, log); len(runs) != 0 {
		t.Fatalf("a host the bound was meant to cut off ran %v", runs)
	}

	// Claude Code's commands are handed over, so the fallback did run.
	if !strings.Contains(out, "claude plugin install "+plugin.PluginID) {
		t.Fatalf("the fallback printed nothing for Claude Code, so this proves nothing about Cursor:\n%s", out)
	}
	// Cursor is absent from it entirely: no header, and no empty handover.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Cursor") && strings.Contains(line, "yourself") {
			t.Errorf("the fallback offered a command handover for a host with no commands:\n%s", out)
		}
	}
}
