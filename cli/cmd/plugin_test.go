package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"archcore-cli/internal/plugin"

	"github.com/spf13/cobra"
)

// Every test in this file swaps a package var or the process environment
// (isolatePluginRun, stubPluginEvidence), so none of them calls t.Parallel().

// runPluginCmd executes `archcore plugin ...` through the real root command, so
// every case here also proves the command is reachable where a user types it.
//
// The working directory is moved somewhere disposable first. `--scope project`
// with no `--project` resolves the working directory, and under `go test` that
// is the repository's own cmd/ directory: a regression in the scope default
// writes .claude/settings.json into the checkout, which is how this suite once
// left one behind.
func runPluginCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	t.Chdir(t.TempDir())

	var buf bytes.Buffer
	root := NewRootCmd("0.0.0-test")
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append([]string{"plugin"}, args...))
	root.SetContext(context.Background())

	err := root.Execute()
	return buf.String(), err
}

// TestPluginCommandOffersTheFourVerbs pins the surface plugin-delivery.spec
// names. A verb that stops being registered is not a compile error anywhere:
// the constructor keeps building, the root keeps ten commands, and the user
// simply loses the word.
func TestPluginCommandOffersTheFourVerbs(t *testing.T) {
	var group *cobra.Command
	for _, c := range NewRootCmd("0.0.0-test").Commands() {
		if c.Name() == "plugin" {
			group = c
		}
	}
	if group == nil {
		t.Fatal("the root command carries no `plugin` command")
	}

	var verbs []string
	for _, c := range group.Commands() {
		verbs = append(verbs, c.Name())
		// --agent and --project sit on the group AND on every verb, so both
		// spellings of `archcore plugin --agent x <verb>` reach the same
		// selection. TestCommands_OfferProjectFlag walks the tree for --project;
		// nothing else walks it for --agent.
		if c.Flags().Lookup("agent") == nil {
			t.Errorf("`plugin %s` declares no --agent flag", c.Name())
		}
	}
	slices.Sort(verbs)
	if want := []string{"install", "remove", "status", "update"}; !slices.Equal(verbs, want) {
		t.Errorf("verbs = %v, want %v", verbs, want)
	}

	install, _, err := group.Find([]string{"install"})
	if err != nil {
		t.Fatalf("finding `plugin install`: %v", err)
	}
	if install.Flags().Lookup("scope") == nil {
		t.Error("`plugin install` declares no --scope flag; user scope is the default it selects away from")
	}
}

// TestPluginRefusesAStrayWord keeps a mistyped verb from reporting success.
// cobra's legacyArgs accepts a positional on every non-root command, so without
// an explicit Args each of these exits zero: `archcore plugin instal` prints
// help for a command that did nothing, and `archcore plugin update claude-code`
// silently acts on every host instead of the one the user meant to name — hosts
// are selected with --agent.
func TestPluginRefusesAStrayWord(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "a mistyped verb", args: []string{"instal"}},
		{name: "a host as a positional to update", args: []string{"update", "claude-code"}},
		{name: "a host as a positional to install", args: []string{"install", "claude-code"}},
		{name: "a host as a positional to remove", args: []string{"remove", "claude-code"}},
		{name: "a host as a positional to status", args: []string{"status", "claude-code"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin := isolatePluginRun(t)
			log := writeHostFixture(t, bin, "claude", 0)
			stubPluginEvidence(t, listedClaude())

			out, err := runPluginCmd(t, tt.args...)
			if err == nil {
				t.Fatalf("`archcore plugin %s` exited zero:\n%s", strings.Join(tt.args, " "), out)
			}
			if runs := fixtureRuns(t, log); len(runs) != 0 {
				t.Errorf("a rejected invocation still ran %v", runs)
			}
		})
	}
}

// TestPluginVerbRunsInCIWithoutATerminal is requirements 10 and 11 of
// plugin-delivery.spec together: the typed verb IS the consent, so a
// non-interactive invocation runs exactly as an interactive one.
//
// The assertion is on the subprocess log rather than on the output, because the
// tier this must not fall into prints the same command line it would have run.
// A CI runner is where `archcore init` is required to print instead of act, and
// that difference has to stay in init.
func TestPluginVerbRunsInCIWithoutATerminal(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, listedClaude())
	t.Setenv("CI", "true")

	out, err := runPluginCmd(t, "update", "--agent", "claude-code")
	if err != nil {
		t.Fatalf("the verb exited nonzero: %v\n%s", err, out)
	}
	if runs := fixtureRuns(t, log); len(runs) != 2 {
		t.Errorf("the host CLI ran %d times %v on a CI runner, want both update commands", len(runs), runs)
	}
	if strings.Contains(out, "yourself") {
		t.Errorf("a typed verb fell back to printing commands:\n%s", out)
	}
}

// TestPluginCommandsCarryTheFrozenIdentifiersVerbatim spells the three
// identifiers as literals rather than through the constants —
// plugin-cli-compatibility.rule §11 freezes the VALUES, and a test that threads
// the constant through both sides passes a rename that ships a CLI addressing a
// plugin nothing answers to.
//
// Two evidence shapes, because no single one reaches both verbs: install runs
// only where the listing does NOT name the plugin, and update only where it
// does.
func TestPluginCommandsCarryTheFrozenIdentifiersVerbatim(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)

	tests := []struct {
		name     string
		verb     string
		evidence plugin.Evidence
		want     []string
	}{
		{
			name:     "install plans both commands",
			verb:     "install",
			evidence: plugin.Evidence{Host: plugin.HostClaudeCode, CLIPresent: true},
			want: []string{
				"plugin marketplace add archcore-ai/plugin",
				"plugin install archcore@archcore-plugins",
			},
		},
		{
			name:     "update plans both commands",
			verb:     "update",
			evidence: listedClaude(),
			want: []string{
				"plugin marketplace update archcore-plugins",
				"plugin update archcore@archcore-plugins",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubPluginEvidence(t, tt.evidence)

			out, err := runPluginCmd(t, tt.verb, "--agent", "claude-code")
			if err != nil {
				t.Fatalf("`plugin %s` exited nonzero: %v\n%s", tt.verb, err, out)
			}
			ran := strings.Join(fixtureRuns(t, log), "\n")
			for _, want := range tt.want {
				if !strings.Contains(ran, want) {
					t.Errorf("the host never received %q; it ran:\n%s", want, ran)
				}
			}
		})
	}
}

// TestPluginVerbExitsNonzeroOnAFailedHostAction covers requirement 18. The
// details are already printed by the shared core, so the error is the
// exit-only marker and main prints nothing more.
func TestPluginVerbExitsNonzeroOnAFailedHostAction(t *testing.T) {
	bin := isolatePluginRun(t)
	writeHostFixture(t, bin, "claude", 1)
	stubPluginEvidence(t, listedClaude())

	out, err := runPluginCmd(t, "update", "--agent", "claude-code")
	if !errors.Is(err, ErrAlreadyReported) {
		t.Fatalf("error = %v, want ErrAlreadyReported\n%s", err, out)
	}
	if !strings.Contains(out, "claude plugin marketplace update archcore-plugins") {
		t.Errorf("the failure names no command:\n%s", out)
	}
}

// TestPluginVerbExitsZeroForAHostSkippedForMissingEvidence covers requirement
// 19, the other half of the pair. A host whose listing does not name the plugin
// produced no action at all, so there is nothing for the exit code to report —
// and a nonzero exit here would make `archcore plugin update` fail on every
// machine that has one host without the plugin.
func TestPluginVerbExitsZeroForAHostSkippedForMissingEvidence(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 1)
	stubPluginEvidence(t, plugin.Evidence{
		Host:       plugin.HostClaudeCode,
		CLIPresent: true,
		ListingOK:  true,
	})

	out, err := runPluginCmd(t, "update", "--agent", "claude-code")
	if err != nil {
		t.Fatalf("a skipped host failed the command: %v\n%s", err, out)
	}
	if runs := fixtureRuns(t, log); len(runs) != 0 {
		t.Errorf("a host without the plugin ran %v", runs)
	}
}

// TestPluginStatusExitsZeroWithEveryHostAbsent covers requirement 20. The
// evidence seam is deliberately left alone: an empty PATH and an empty home ARE
// the machine with no host CLI and no registry, which is the case the exit code
// has to survive.
func TestPluginStatusExitsZeroWithEveryHostAbsent(t *testing.T) {
	isolatePluginRun(t)

	out, err := runPluginCmd(t, "status")
	if err != nil {
		t.Fatalf("status must exit zero, got %v\n%s", err, out)
	}
	for _, spec := range plugin.Specs() {
		if !strings.Contains(out, spec.DisplayName) {
			t.Errorf("the report skips %s:\n%s", spec.DisplayName, out)
		}
	}
	if !strings.Contains(out, "not installed") {
		t.Errorf("a machine with no plugin is not reported as such:\n%s", out)
	}
}

// TestPluginAgentFlagNamesOnlyThePluginHosts covers the Constraint of
// plugin-delivery.spec at the command surface. gemini-cli is a valid agent
// everywhere else in the CLI, so the error may not answer with the agent
// registry: it would offer names this command cannot act on.
func TestPluginAgentFlagNamesOnlyThePluginHosts(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, listedClaude())

	out, err := runPluginCmd(t, "update", "--agent", "gemini-cli")
	if err == nil {
		t.Fatalf("an agent with no shipping plugin was accepted:\n%s", out)
	}
	for _, host := range plugin.Hosts() {
		if !strings.Contains(err.Error(), string(host)) {
			t.Errorf("the error does not name %q: %v", host, err)
		}
	}
	for _, other := range []string{"roo-code", "cline", "opencode"} {
		if strings.Contains(err.Error(), other) {
			t.Errorf("the error names %q, which ships no plugin: %v", other, err)
		}
	}
	if runs := fixtureRuns(t, log); len(runs) != 0 {
		t.Errorf("a rejected selection still ran %v", runs)
	}
}

// TestPluginInstallDefaultsToUserScope covers requirement 12: the marketplace
// entry lands in the user's own settings file unless `--scope project` moved it.
func TestPluginInstallDefaultsToUserScope(t *testing.T) {
	bin := isolatePluginRun(t)
	writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, plugin.Evidence{Host: plugin.HostClaudeCode, CLIPresent: true})

	out, err := runPluginCmd(t, "install", "--agent", "claude-code")
	if err != nil {
		t.Fatalf("the install exited nonzero: %v\n%s", err, out)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if entry := marketplaceEntry(t, filepath.Join(home, ".claude", "settings.json")); entry["autoUpdate"] != true {
		t.Errorf("the user's settings entry = %+v, want autoUpdate enabled", entry)
	}
	// The teammate-reach sentence belongs to a repository file. Printing it for
	// a user-scope install would describe a file no teammate will ever see.
	if strings.Contains(out, "teammate") {
		t.Errorf("a user-scope install reported repository reach:\n%s", out)
	}
	// A default that slipped to project scope resolves the working directory
	// and writes there. runPluginCmd moved it somewhere disposable, so this
	// catches the slip instead of committing its result.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".claude")); err == nil {
		t.Error("a user-scope install wrote into the working directory")
	}
}

// TestPluginInstallProjectScopeWritesTheRepositoryFileAndStatesItsReach covers
// requirement 13. The sentence is the requirement, not a courtesy: a settings
// file inside the repository is committed, and the user chose the scope without
// necessarily choosing that consequence.
func TestPluginInstallProjectScopeWritesTheRepositoryFileAndStatesItsReach(t *testing.T) {
	bin := isolatePluginRun(t)
	writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, plugin.Evidence{Host: plugin.HostClaudeCode, CLIPresent: true})
	root := t.TempDir()

	out, err := runPluginCmd(t, "install", "--agent", "claude-code", "--scope", "project", "--project", root)
	if err != nil {
		t.Fatalf("the install exited nonzero: %v\n%s", err, out)
	}

	if entry := marketplaceEntry(t, filepath.Join(root, ".claude", "settings.json")); entry["autoUpdate"] != true {
		t.Errorf("the repository settings entry = %+v, want autoUpdate enabled", entry)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if entry := marketplaceEntry(t, filepath.Join(home, ".claude", "settings.json")); entry != nil {
		t.Errorf("--scope project also wrote the user's own settings file: %+v", entry)
	}
	if !strings.Contains(out, "teammate") {
		t.Errorf("the repository-reach sentence is missing:\n%s", out)
	}
}

// TestPluginInstallStatesRepositoryReachOnlyForAnEntryItWrote is the other half
// of requirement 13. The sentence describes a file in the repository, so a run
// that wrote none must not print it: the merge follows the host's own result, and
// neither a command that failed nor one that was only printed produces a file for
// a teammate to check out.
func TestPluginInstallStatesRepositoryReachOnlyForAnEntryItWrote(t *testing.T) {
	tests := []struct {
		name     string
		evidence plugin.Evidence
		exitCode int
	}{
		{
			name:     "the host command failed",
			evidence: plugin.Evidence{Host: plugin.HostClaudeCode, CLIPresent: true},
			exitCode: 1,
		},
		{
			// No CLI on PATH and no registry: the install is handed to the user as
			// a command line, and this process writes nothing at all.
			name:     "the host CLI could not be reached",
			evidence: plugin.Evidence{Host: plugin.HostClaudeCode},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin := isolatePluginRun(t)
			writeHostFixture(t, bin, "claude", tt.exitCode)
			stubPluginEvidence(t, tt.evidence)
			root := t.TempDir()

			out, _ := runPluginCmd(t, "install", "--agent", "claude-code", "--scope", "project", "--project", root)

			if strings.Contains(out, "teammate") {
				t.Errorf("the reach sentence describes a file this run never wrote:\n%s", out)
			}
			if _, err := os.Stat(filepath.Join(root, ".claude", "settings.json")); err == nil {
				t.Error("the repository gained a marketplace entry from an install that did not complete")
			}
		})
	}
}

// TestPluginInstallRejectsAnUnknownScope keeps a typo from silently taking the
// default. `--scope porject` writing the user's settings file is a surprise the
// user cannot see: the command reports success and the repository gains nothing.
func TestPluginInstallRejectsAnUnknownScope(t *testing.T) {
	bin := isolatePluginRun(t)
	log := writeHostFixture(t, bin, "claude", 0)
	stubPluginEvidence(t, plugin.Evidence{Host: plugin.HostClaudeCode, CLIPresent: true})

	out, err := runPluginCmd(t, "install", "--agent", "claude-code", "--scope", "porject")
	if err == nil {
		t.Fatalf("an unknown scope was accepted:\n%s", out)
	}
	if runs := fixtureRuns(t, log); len(runs) != 0 {
		t.Errorf("a rejected scope still installed: %v", runs)
	}
}

// TestPluginInstallScopeProjectRefusesAHostPluginCache covers the branch
// host-cwd-misrouting.adr put behind `--scope project`, and the reason the scope
// resolves a root at all.
//
// Hosts have been observed spawning agent processes with cwd inside their own
// plugin install cache. Under `--scope project` the marketplace entry is written
// relative to the resolved root, so a misrouted working directory would write
// the user's Claude Code settings into a plugin's bundled files. The refusal has
// to happen before any host command runs — a partial install that then failed to
// record itself is worse than one that never started.
//
// User scope resolves no root at all and is the control: the same working
// directory must not refuse there, because that install writes nothing into the
// repository.
func TestPluginInstallScopeProjectRefusesAHostPluginCache(t *testing.T) {
	cache := filepath.Join(t.TempDir(), ".cursor", "plugins", "cache", "archcore-plugins", "archcore")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cache)

	f := &pluginFlags{}

	if _, err := pluginInstallTargetFor(f, string(pluginScopeProject)); err == nil {
		t.Error("--scope project accepted a working directory inside a host plugin cache")
	} else if !strings.Contains(err.Error(), "plugin install cache") {
		t.Errorf("error = %v, want the misrouted-cwd refusal", err)
	}

	target, err := pluginInstallTargetFor(f, string(pluginScopeUser))
	if err != nil {
		t.Fatalf("--scope user resolved a project root it does not need: %v", err)
	}
	if target.SettingsPath != "" || target.ProjectRoot != "" {
		t.Errorf("target = %+v, want the empty target user scope resolves to", target)
	}
}
