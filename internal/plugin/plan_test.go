package plugin

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

// The command lines each verb plans, stated once as literals. The tier tables
// below reference them, so a tier row says which tier fired and never restates
// the host table.
var (
	claudeUpdateLines = []string{
		"claude plugin marketplace update archcore-plugins",
		"claude plugin update archcore@archcore-plugins",
	}
	claudeInstallLines = []string{
		"claude plugin marketplace add archcore-ai/plugin",
		"claude plugin install archcore@archcore-plugins",
	}
	claudeRemoveLines = []string{"claude plugin uninstall archcore@archcore-plugins"}

	codexUpdateLines  = []string{"codex plugin marketplace upgrade archcore-plugins"}
	codexInstallLines = []string{
		"codex plugin marketplace add archcore-ai/plugin",
		"codex plugin add archcore@archcore-plugins",
	}
	codexRemoveLines = []string{"codex plugin remove archcore@archcore-plugins"}

	copilotUpdateLines  = []string{"copilot plugin update archcore@archcore-plugins"}
	copilotInstallLines = []string{"copilot plugin install archcore-ai/plugin:plugins/archcore"}
	copilotRemoveLines  = []string{"copilot plugin uninstall archcore"}
)

// wantAction is the expected shape of one host's planned action. A nil
// *wantAction means the tier table addresses nothing, which the specs spell as
// "skip that host silently".
type wantAction struct {
	kind      ActionKind
	commands  []string
	note      bool // the action must carry a non-empty instruction
	merge     bool // the caller merges the autoUpdate entry after success
	removeEnt bool // the caller takes the autoUpdate entry back after success
}

func wantRun(lines ...string) *wantAction {
	return &wantAction{kind: ActionRun, commands: lines}
}

// runMerging is the Claude Code install: the one run action whose success is
// followed by the autoUpdate marketplace entry merge.
func runMerging(lines ...string) *wantAction {
	return &wantAction{kind: ActionRun, commands: lines, merge: true}
}

// runRemovingEntry is the Claude Code removal: the run action that also takes
// back the autoUpdate marketplace entry this surface wrote.
func runRemovingEntry(lines ...string) *wantAction {
	return &wantAction{kind: ActionRun, commands: lines, removeEnt: true}
}

func printCmd(lines ...string) *wantAction {
	return &wantAction{kind: ActionPrintCommand, commands: lines}
}

func uiNote() *wantAction {
	return &wantAction{kind: ActionPrintUINote, note: true}
}

func reportInstalled() *wantAction {
	return &wantAction{kind: ActionReportInstalled}
}

// reportInstalledMerging is the Claude Code install over a plugin that is
// already there. It runs nothing and still merges: an install whose settings
// write failed once would otherwise report "already installed" forever and
// never converge on the entry.
func reportInstalledMerging() *wantAction {
	return &wantAction{kind: ActionReportInstalled, merge: true}
}

func reportStatus() *wantAction {
	return &wantAction{kind: ActionReportStatus}
}

// tierCase is one row of a verb's tier matrix: one evidence shape, and what
// each of the four hosts does with it.
type tierCase struct {
	ev      Evidence
	claude  *wantAction
	cursor  *wantAction
	codex   *wantAction
	copilot *wantAction
}

// shapeKey packs the four observed booleans of an evidence shape into one
// number, so the matrix below can be proved to cover every shape exactly once.
func shapeKey(ev Evidence) int {
	key := 0
	for i, set := range []bool{ev.CLIPresent, ev.ListingOK, ev.Listed, ev.RegistryListed} {
		if set {
			key |= 1 << i
		}
	}
	return key
}

// shapeName names an evidence shape by the observations it carries, so a row's
// subtest name cannot drift from the row's evidence.
func shapeName(ev Evidence) string {
	var parts []string
	if ev.CLIPresent {
		parts = append(parts, "cli")
	}
	if ev.ListingOK {
		parts = append(parts, "listing")
	}
	if ev.Listed {
		parts = append(parts, "listed")
	}
	if ev.RegistryListed {
		parts = append(parts, "registry")
	}
	if len(parts) == 0 {
		return "no evidence"
	}
	return strings.Join(parts, "+")
}

// allShapes is the number of evidence shapes the four observed booleans can
// take. Every verb's matrix states all of them: the tiers read the booleans in
// combination, so a shape left out is a tier no test constrains.
const allShapes = 16

// runTierMatrix plans each row for each host in isolation and checks the tier
// that fired. Planning one host at a time is what makes a row readable; the
// order and purity guards below cover the whole-plan shape.
//
// The matrix must be exhaustive, not representative. Two of the tier guards read
// ListingOK and Listed together — a listing that ran and named the plugin is not
// the same evidence as one that never parsed but left Listed set — and a matrix
// that only walks the coherent shapes leaves that conjunction unconstrained.
func runTierMatrix(t *testing.T, verb Verb, cases []tierCase) {
	t.Helper()
	seen := make(map[int]bool, allShapes)
	for _, tt := range cases {
		key := shapeKey(tt.ev)
		if seen[key] {
			t.Errorf("evidence shape %q is stated twice", shapeName(tt.ev))
		}
		seen[key] = true
		t.Run(shapeName(tt.ev), func(t *testing.T) {
			checkHostPlan(t, verb, HostClaudeCode, tt.ev, tt.claude)
			checkHostPlan(t, verb, HostCursor, tt.ev, tt.cursor)
			checkHostPlan(t, verb, HostCodexCLI, tt.ev, tt.codex)
			checkHostPlan(t, verb, HostCopilot, tt.ev, tt.copilot)
		})
	}
	for key := 0; key < allShapes; key++ {
		if !seen[key] {
			ev := Evidence{
				CLIPresent:     key&1 != 0,
				ListingOK:      key&2 != 0,
				Listed:         key&4 != 0,
				RegistryListed: key&8 != 0,
			}
			t.Errorf("%s: evidence shape %q is not stated; the matrix must be exhaustive", verb, shapeName(ev))
		}
	}
}

func checkHostPlan(t *testing.T, verb Verb, host Host, ev Evidence, want *wantAction) {
	t.Helper()
	ev.Host = host
	actions := Plan(verb, []Evidence{ev})

	if want == nil {
		if len(actions) != 0 {
			t.Errorf("%s %s: planned %d actions (%v), want silence", host, verb, len(actions), actions)
		}
		return
	}
	if len(actions) != 1 {
		t.Fatalf("%s %s: planned %d actions, want 1", host, verb, len(actions))
	}
	got := actions[0]
	if got.Host != host {
		t.Errorf("%s %s: action host = %q, want %q", host, verb, got.Host, host)
	}
	if got.Kind != want.kind {
		t.Fatalf("%s %s: kind = %s, want %s", host, verb, got.Kind, want.kind)
	}
	if lines := commandLines(got.Commands); !slices.Equal(lines, want.commands) {
		t.Errorf("%s %s: commands = %q, want %q", host, verb, lines, want.commands)
	}
	if hasNote := got.Note != ""; hasNote != want.note {
		t.Errorf("%s %s: note = %q, want a note: %v", host, verb, got.Note, want.note)
	}
	if got.MergeAutoUpdate != want.merge {
		t.Errorf("%s %s: MergeAutoUpdate = %v, want %v", host, verb, got.MergeAutoUpdate, want.merge)
	}
	if got.RemoveAutoUpdate != want.removeEnt {
		t.Errorf("%s %s: RemoveAutoUpdate = %v, want %v", host, verb, got.RemoveAutoUpdate, want.removeEnt)
	}
	if got.Evidence != ev {
		t.Errorf("%s %s: evidence = %+v, want the evidence it was planned from %+v", host, verb, got.Evidence, ev)
	}
}

// TestPlanUpdateTiers walks the full evidence matrix for the update verb —
// requirements 5 to 10 of updating-the-plugin.spec. The rows that matter
// most are the silent ones: a present host CLI whose listing failed or omits
// the plugin runs nothing and prints nothing, which is how a machine that never
// installed the plugin pays no mutating command.
func TestPlanUpdateTiers(t *testing.T) {
	t.Parallel()
	runTierMatrix(t, VerbUpdate, []tierCase{
		// The CLI is absent. The on-disk registry is the only evidence that
		// counts, because the host itself never answered: a listing field left
		// set by a collector cannot stand in for the host's own reply.
		{
			ev: Evidence{},
		},
		{
			ev: Evidence{Listed: true},
		},
		{
			ev: Evidence{ListingOK: true},
		},
		{
			ev: Evidence{ListingOK: true, Listed: true},
		},
		{
			ev:      Evidence{RegistryListed: true},
			claude:  printCmd(claudeUpdateLines...),
			cursor:  uiNote(),
			codex:   printCmd(codexUpdateLines...),
			copilot: printCmd(copilotUpdateLines...),
		},
		{
			ev:      Evidence{Listed: true, RegistryListed: true},
			claude:  printCmd(claudeUpdateLines...),
			cursor:  uiNote(),
			codex:   printCmd(codexUpdateLines...),
			copilot: printCmd(copilotUpdateLines...),
		},
		{
			ev:      Evidence{ListingOK: true, RegistryListed: true},
			claude:  printCmd(claudeUpdateLines...),
			cursor:  uiNote(),
			codex:   printCmd(codexUpdateLines...),
			copilot: printCmd(copilotUpdateLines...),
		},
		{
			ev:      Evidence{ListingOK: true, Listed: true, RegistryListed: true},
			claude:  printCmd(claudeUpdateLines...),
			cursor:  uiNote(),
			codex:   printCmd(codexUpdateLines...),
			copilot: printCmd(copilotUpdateLines...),
		},

		// The CLI is present. Requirement 6 authorizes the mutating command from
		// the host's own listing alone, so requirement 7 skips every other shape
		// in silence — a registry that disagrees included.
		{
			ev: Evidence{CLIPresent: true},
		},
		// Listed without ListingOK is a listing that ran but never parsed.
		// Failure behavior 1 counts it as not listed, so it must stay silent:
		// dropping the ListingOK half of the guard turns this row into a
		// mutating command on a host that answered nothing.
		{
			ev: Evidence{CLIPresent: true, Listed: true},
		},
		{
			ev: Evidence{CLIPresent: true, ListingOK: true},
		},
		{
			ev:      Evidence{CLIPresent: true, ListingOK: true, Listed: true},
			claude:  wantRun(claudeUpdateLines...),
			codex:   wantRun(codexUpdateLines...),
			copilot: wantRun(copilotUpdateLines...),
		},
		{
			ev:     Evidence{CLIPresent: true, RegistryListed: true},
			cursor: uiNote(),
		},
		{
			ev:     Evidence{CLIPresent: true, Listed: true, RegistryListed: true},
			cursor: uiNote(),
		},
		{
			ev:     Evidence{CLIPresent: true, ListingOK: true, RegistryListed: true},
			cursor: uiNote(),
		},
		{
			ev:      Evidence{CLIPresent: true, ListingOK: true, Listed: true, RegistryListed: true},
			claude:  wantRun(claudeUpdateLines...),
			cursor:  uiNote(),
			codex:   wantRun(codexUpdateLines...),
			copilot: wantRun(copilotUpdateLines...),
		},
	})
}

// TestPlanInstallTiers walks the full evidence matrix for the install verb.
// Install carries explicit consent, so a failed listing does not refuse it —
// the asymmetry with update. The one refusal is a listing that names the
// plugin: that install is a reported no-op, which is what keeps a rerun of
// `archcore init` from reinstalling.
func TestPlanInstallTiers(t *testing.T) {
	t.Parallel()
	runTierMatrix(t, VerbInstall, []tierCase{
		// The CLI is absent. Failure behavior 1 prints the exact install command,
		// unless the registry already names the plugin — then the install is the
		// reported no-op the idempotence invariant asks for. Cursor answers the
		// same question the same way, from the same registry evidence.
		{
			ev:      Evidence{},
			claude:  printCmd(claudeInstallLines...),
			cursor:  uiNote(),
			codex:   printCmd(codexInstallLines...),
			copilot: printCmd(copilotInstallLines...),
		},
		{
			ev:      Evidence{Listed: true},
			claude:  printCmd(claudeInstallLines...),
			cursor:  uiNote(),
			codex:   printCmd(codexInstallLines...),
			copilot: printCmd(copilotInstallLines...),
		},
		{
			ev:      Evidence{ListingOK: true},
			claude:  printCmd(claudeInstallLines...),
			cursor:  uiNote(),
			codex:   printCmd(codexInstallLines...),
			copilot: printCmd(copilotInstallLines...),
		},
		{
			ev:      Evidence{ListingOK: true, Listed: true},
			claude:  printCmd(claudeInstallLines...),
			cursor:  uiNote(),
			codex:   printCmd(codexInstallLines...),
			copilot: printCmd(copilotInstallLines...),
		},
		{
			ev:      Evidence{RegistryListed: true},
			claude:  reportInstalledMerging(),
			cursor:  reportInstalled(),
			codex:   reportInstalled(),
			copilot: reportInstalled(),
		},
		{
			ev:      Evidence{Listed: true, RegistryListed: true},
			claude:  reportInstalledMerging(),
			cursor:  reportInstalled(),
			codex:   reportInstalled(),
			copilot: reportInstalled(),
		},
		{
			ev:      Evidence{ListingOK: true, RegistryListed: true},
			claude:  reportInstalledMerging(),
			cursor:  reportInstalled(),
			codex:   reportInstalled(),
			copilot: reportInstalled(),
		},
		{
			ev:      Evidence{ListingOK: true, Listed: true, RegistryListed: true},
			claude:  reportInstalledMerging(),
			cursor:  reportInstalled(),
			codex:   reportInstalled(),
			copilot: reportInstalled(),
		},

		// The CLI is present. Consent is already given, so a listing that failed
		// does not refuse the install — only a listing that ran and named the
		// plugin does, which is requirement 9 and the rerun that never nags.
		{
			ev:      Evidence{CLIPresent: true},
			claude:  runMerging(claudeInstallLines...),
			cursor:  uiNote(),
			codex:   wantRun(codexInstallLines...),
			copilot: wantRun(copilotInstallLines...),
		},
		// Listed without ListingOK is a listing that never parsed. It is not the
		// evidence requirement 9 asks for, so the install still runs: dropping
		// the ListingOK half of the guard turns this row into a silent no-op and
		// a selected host never gets its plugin.
		{
			ev:      Evidence{CLIPresent: true, Listed: true},
			claude:  runMerging(claudeInstallLines...),
			cursor:  uiNote(),
			codex:   wantRun(codexInstallLines...),
			copilot: wantRun(copilotInstallLines...),
		},
		{
			ev:      Evidence{CLIPresent: true, ListingOK: true},
			claude:  runMerging(claudeInstallLines...),
			cursor:  uiNote(),
			codex:   wantRun(codexInstallLines...),
			copilot: wantRun(copilotInstallLines...),
		},
		{
			ev:      Evidence{CLIPresent: true, ListingOK: true, Listed: true},
			claude:  reportInstalledMerging(),
			cursor:  uiNote(),
			codex:   reportInstalled(),
			copilot: reportInstalled(),
		},
		{
			ev:      Evidence{CLIPresent: true, RegistryListed: true},
			claude:  runMerging(claudeInstallLines...),
			cursor:  reportInstalled(),
			codex:   wantRun(codexInstallLines...),
			copilot: wantRun(copilotInstallLines...),
		},
		{
			ev:      Evidence{CLIPresent: true, Listed: true, RegistryListed: true},
			claude:  runMerging(claudeInstallLines...),
			cursor:  reportInstalled(),
			codex:   wantRun(codexInstallLines...),
			copilot: wantRun(copilotInstallLines...),
		},
		{
			ev:      Evidence{CLIPresent: true, ListingOK: true, RegistryListed: true},
			claude:  runMerging(claudeInstallLines...),
			cursor:  reportInstalled(),
			codex:   wantRun(codexInstallLines...),
			copilot: wantRun(copilotInstallLines...),
		},
		{
			ev:      Evidence{CLIPresent: true, ListingOK: true, Listed: true, RegistryListed: true},
			claude:  reportInstalledMerging(),
			cursor:  reportInstalled(),
			codex:   reportInstalled(),
			copilot: reportInstalled(),
		},
	})
}

// TestPlanRemoveTiers walks the full evidence matrix for the remove verb.
// Removal reaches a host only from a typed verb, so a failed listing does not
// refuse it. Proof of absence does: a listing that ran and omits the plugin, or
// an absent CLI over a registry that omits it, means there is nothing to
// remove and nothing to say.
func TestPlanRemoveTiers(t *testing.T) {
	t.Parallel()
	runTierMatrix(t, VerbRemove, []tierCase{
		// The CLI is absent. Failure behavior 5 prints the exact uninstall
		// command, but only where the registry says there is something to
		// remove. Removing nothing is not news.
		{
			ev: Evidence{},
		},
		{
			ev: Evidence{Listed: true},
		},
		{
			ev: Evidence{ListingOK: true},
		},
		{
			ev: Evidence{ListingOK: true, Listed: true},
		},
		{
			ev:      Evidence{RegistryListed: true},
			claude:  printCmd(claudeRemoveLines...),
			cursor:  uiNote(),
			codex:   printCmd(codexRemoveLines...),
			copilot: printCmd(copilotRemoveLines...),
		},
		{
			ev:      Evidence{Listed: true, RegistryListed: true},
			claude:  printCmd(claudeRemoveLines...),
			cursor:  uiNote(),
			codex:   printCmd(codexRemoveLines...),
			copilot: printCmd(copilotRemoveLines...),
		},
		{
			ev:      Evidence{ListingOK: true, RegistryListed: true},
			claude:  printCmd(claudeRemoveLines...),
			cursor:  uiNote(),
			codex:   printCmd(codexRemoveLines...),
			copilot: printCmd(copilotRemoveLines...),
		},
		{
			ev:      Evidence{ListingOK: true, Listed: true, RegistryListed: true},
			claude:  printCmd(claudeRemoveLines...),
			cursor:  uiNote(),
			codex:   printCmd(codexRemoveLines...),
			copilot: printCmd(copilotRemoveLines...),
		},

		// The CLI is present. Only proof of absence refuses the verb: a listing
		// that ran and omits the plugin. A listing that never parsed proves
		// nothing, so the typed verb still reaches the host.
		{
			ev:      Evidence{CLIPresent: true},
			claude:  runRemovingEntry(claudeRemoveLines...),
			codex:   wantRun(codexRemoveLines...),
			copilot: wantRun(copilotRemoveLines...),
		},
		{
			ev:      Evidence{CLIPresent: true, Listed: true},
			claude:  runRemovingEntry(claudeRemoveLines...),
			codex:   wantRun(codexRemoveLines...),
			copilot: wantRun(copilotRemoveLines...),
		},
		{
			ev: Evidence{CLIPresent: true, ListingOK: true},
		},
		{
			ev:      Evidence{CLIPresent: true, ListingOK: true, Listed: true},
			claude:  runRemovingEntry(claudeRemoveLines...),
			codex:   wantRun(codexRemoveLines...),
			copilot: wantRun(copilotRemoveLines...),
		},
		{
			ev:      Evidence{CLIPresent: true, RegistryListed: true},
			claude:  runRemovingEntry(claudeRemoveLines...),
			cursor:  uiNote(),
			codex:   wantRun(codexRemoveLines...),
			copilot: wantRun(copilotRemoveLines...),
		},
		{
			ev:      Evidence{CLIPresent: true, Listed: true, RegistryListed: true},
			claude:  runRemovingEntry(claudeRemoveLines...),
			cursor:  uiNote(),
			codex:   wantRun(codexRemoveLines...),
			copilot: wantRun(copilotRemoveLines...),
		},
		{
			ev:     Evidence{CLIPresent: true, ListingOK: true, RegistryListed: true},
			cursor: uiNote(),
		},
		{
			ev:      Evidence{CLIPresent: true, ListingOK: true, Listed: true, RegistryListed: true},
			claude:  runRemovingEntry(claudeRemoveLines...),
			cursor:  uiNote(),
			codex:   wantRun(codexRemoveLines...),
			copilot: wantRun(copilotRemoveLines...),
		},
	})
}

// TestPlanStatusTiers walks the full evidence matrix for the status verb.
// Status has one tier and no condition: report what was found, for every host
// the caller gave evidence for. Every shape therefore expects the same action,
// which is the one row the matrix may state once and apply to all sixteen —
// stating it per shape would only repeat the same literal sixteen times.
func TestPlanStatusTiers(t *testing.T) {
	t.Parallel()
	cases := make([]tierCase, 0, allShapes)
	for key := 0; key < allShapes; key++ {
		cases = append(cases, tierCase{
			ev: Evidence{
				CLIPresent:     key&1 != 0,
				ListingOK:      key&2 != 0,
				Listed:         key&4 != 0,
				RegistryListed: key&8 != 0,
			},
			claude:  reportStatus(),
			cursor:  reportStatus(),
			codex:   reportStatus(),
			copilot: reportStatus(),
		})
	}
	runTierMatrix(t, VerbStatus, cases)
}

// TestPlanStatusCarriesTheReportedVersion proves the version a host reports
// survives into the action. Requirement 16 asks status to report it, and the
// planner is the only place it could be dropped.
func TestPlanStatusCarriesTheReportedVersion(t *testing.T) {
	t.Parallel()
	ev := Evidence{Host: HostClaudeCode, CLIPresent: true, ListingOK: true, Listed: true, ListedVersion: "1.4.0"}
	actions := Plan(VerbStatus, []Evidence{ev})
	if len(actions) != 1 {
		t.Fatalf("planned %d actions, want 1", len(actions))
	}
	if actions[0].Evidence.ListedVersion != "1.4.0" {
		t.Errorf("reported version = %q, want %q", actions[0].Evidence.ListedVersion, "1.4.0")
	}
}

// TestPlanCarriesTheFrozenIdentifiers is the guard requirement 11 of
// plugin-cli-compatibility.rule needs: the identifiers must survive into
// what the planner actually emits, not only into the constants. A refactor that
// renamed the marketplace would pass a constants-only test and fail here.
func TestPlanCarriesTheFrozenIdentifiers(t *testing.T) {
	t.Parallel()
	listed := Evidence{CLIPresent: true, ListingOK: true, Listed: true}
	notListed := Evidence{CLIPresent: true, ListingOK: true}

	tests := []struct {
		name            string
		host            Host
		verb            Verb
		ev              Evidence
		wantLines       []string
		wantIdentifiers []string
	}{
		{
			name: "claude code update", host: HostClaudeCode, verb: VerbUpdate, ev: listed,
			wantLines:       claudeUpdateLines,
			wantIdentifiers: []string{MarketplaceID, PluginID},
		},
		{
			name: "claude code install", host: HostClaudeCode, verb: VerbInstall, ev: notListed,
			wantLines:       claudeInstallLines,
			wantIdentifiers: []string{RepoID, PluginID},
		},
		{
			name: "claude code remove", host: HostClaudeCode, verb: VerbRemove, ev: listed,
			wantLines:       claudeRemoveLines,
			wantIdentifiers: []string{PluginID},
		},
		{
			name: "codex cli update", host: HostCodexCLI, verb: VerbUpdate, ev: listed,
			wantLines:       codexUpdateLines,
			wantIdentifiers: []string{MarketplaceID},
		},
		{
			name: "codex cli install", host: HostCodexCLI, verb: VerbInstall, ev: notListed,
			wantLines:       codexInstallLines,
			wantIdentifiers: []string{RepoID, PluginID},
		},
		{
			name: "codex cli remove", host: HostCodexCLI, verb: VerbRemove, ev: listed,
			wantLines:       codexRemoveLines,
			wantIdentifiers: []string{PluginID},
		},
		{
			name: "copilot update", host: HostCopilot, verb: VerbUpdate, ev: listed,
			wantLines:       copilotUpdateLines,
			wantIdentifiers: []string{PluginID},
		},
		{
			name: "copilot install", host: HostCopilot, verb: VerbInstall, ev: notListed,
			wantLines:       copilotInstallLines,
			wantIdentifiers: []string{RepoID},
		},
		{
			// Copilot's uninstall takes the bare plugin name, so no frozen
			// identifier appears. The line is pinned instead.
			name: "copilot remove", host: HostCopilot, verb: VerbRemove, ev: listed,
			wantLines: copilotRemoveLines,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ev.Host = tt.host
			actions := Plan(tt.verb, []Evidence{tt.ev})
			if len(actions) != 1 {
				t.Fatalf("planned %d actions, want 1", len(actions))
			}
			if actions[0].Kind != ActionRun {
				t.Fatalf("kind = %s, want %s", actions[0].Kind, ActionRun)
			}
			lines := commandLines(actions[0].Commands)
			if !slices.Equal(lines, tt.wantLines) {
				t.Fatalf("planned commands = %q, want %q", lines, tt.wantLines)
			}
			joined := strings.Join(lines, " ")
			for _, id := range tt.wantIdentifiers {
				if !strings.Contains(joined, id) {
					t.Errorf("planned commands %q do not carry the frozen identifier %q", joined, id)
				}
			}
		})
	}
}

// TestPlanFrozenIdentifiersReachEveryPrintedTier proves the print tier carries
// the same identifiers as the run tier. The printed line is what a user pastes,
// so a rename that survived only there would still address the wrong plugin.
//
// Each verb needs the evidence that actually reaches its print tier: install
// prints when the host is unreachable and the registry holds nothing, while
// update and remove print on registry evidence alone. Driving all three from one
// evidence shape would leave install asserting nothing at all.
func TestPlanFrozenIdentifiersReachEveryPrintedTier(t *testing.T) {
	t.Parallel()
	unreachable := Evidence{}
	registryOnly := Evidence{RegistryListed: true}

	tests := []struct {
		name            string
		host            Host
		verb            Verb
		ev              Evidence
		wantLines       []string
		wantIdentifiers []string
	}{
		{
			name: "claude code install", host: HostClaudeCode, verb: VerbInstall, ev: unreachable,
			wantLines:       claudeInstallLines,
			wantIdentifiers: []string{RepoID, PluginID},
		},
		{
			name: "claude code update", host: HostClaudeCode, verb: VerbUpdate, ev: registryOnly,
			wantLines:       claudeUpdateLines,
			wantIdentifiers: []string{MarketplaceID, PluginID},
		},
		{
			name: "claude code remove", host: HostClaudeCode, verb: VerbRemove, ev: registryOnly,
			wantLines:       claudeRemoveLines,
			wantIdentifiers: []string{PluginID},
		},
		{
			name: "codex cli install", host: HostCodexCLI, verb: VerbInstall, ev: unreachable,
			wantLines:       codexInstallLines,
			wantIdentifiers: []string{RepoID, PluginID},
		},
		{
			name: "codex cli update", host: HostCodexCLI, verb: VerbUpdate, ev: registryOnly,
			wantLines:       codexUpdateLines,
			wantIdentifiers: []string{MarketplaceID},
		},
		{
			name: "codex cli remove", host: HostCodexCLI, verb: VerbRemove, ev: registryOnly,
			wantLines:       codexRemoveLines,
			wantIdentifiers: []string{PluginID},
		},
		{
			name: "copilot install", host: HostCopilot, verb: VerbInstall, ev: unreachable,
			wantLines:       copilotInstallLines,
			wantIdentifiers: []string{RepoID},
		},
		{
			name: "copilot update", host: HostCopilot, verb: VerbUpdate, ev: registryOnly,
			wantLines:       copilotUpdateLines,
			wantIdentifiers: []string{PluginID},
		},
		{
			// Copilot's uninstall takes the bare plugin name, so no frozen
			// identifier appears. The line is pinned instead.
			name: "copilot remove", host: HostCopilot, verb: VerbRemove, ev: registryOnly,
			wantLines: copilotRemoveLines,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ev.Host = tt.host
			actions := Plan(tt.verb, []Evidence{tt.ev})
			if len(actions) != 1 {
				t.Fatalf("planned %d actions, want 1", len(actions))
			}
			if actions[0].Kind != ActionPrintCommand {
				t.Fatalf("kind = %s, want %s", actions[0].Kind, ActionPrintCommand)
			}
			lines := commandLines(actions[0].Commands)
			if !slices.Equal(lines, tt.wantLines) {
				t.Fatalf("printed commands = %q, want %q", lines, tt.wantLines)
			}
			joined := strings.Join(lines, " ")
			for _, id := range tt.wantIdentifiers {
				if !strings.Contains(joined, id) {
					t.Errorf("printed commands %q do not carry the frozen identifier %q", joined, id)
				}
			}
		})
	}
}

// TestPlanCommandsDoNotAliasTheHostTable proves a planned command may be
// rewritten without rewriting the process-wide host table. The executor's own
// job makes this reachable: it appends NonInteractiveFlag to a planned command
// on a non-TTY, and an aliased argument slice would carry that edit into every
// later plan in the process — including the printed lines a user pastes.
func TestPlanCommandsDoNotAliasTheHostTable(t *testing.T) {
	t.Parallel()
	evidence := []Evidence{{Host: HostClaudeCode, CLIPresent: true, ListingOK: true, Listed: true}}

	first := Plan(VerbUpdate, evidence)
	if len(first) != 1 || len(first[0].Commands) != 2 {
		t.Fatalf("planned %+v, want one action carrying two commands", first)
	}
	first[0].Commands[0].Name = "rewritten"
	first[0].Commands[0].Args[3] = "rewritten"
	first[0].Commands[1].Args = append(first[0].Commands[1].Args, "-y")

	if got := commandLines(Plan(VerbUpdate, evidence)[0].Commands); !slices.Equal(got, claudeUpdateLines) {
		t.Errorf("second plan = %q after a caller rewrote the first, want %q", got, claudeUpdateLines)
	}
	spec, _ := SpecFor(HostClaudeCode)
	if got := commandLines(spec.Update); !slices.Equal(got, claudeUpdateLines) {
		t.Errorf("host table = %q after a caller rewrote a planned command, want %q", got, claudeUpdateLines)
	}
}

// TestPlanIgnoresHostsWithoutAPlugin keeps evidence about a host that ships no
// plugin out of every plan, whatever the verb and whatever the evidence says.
func TestPlanIgnoresHostsWithoutAPlugin(t *testing.T) {
	t.Parallel()
	full := Evidence{CLIPresent: true, ListingOK: true, Listed: true, RegistryListed: true}
	hosts := []Host{"gemini-cli", "opencode", "roo-code", "cline", "", "not-a-host"}
	verbs := []Verb{VerbInstall, VerbUpdate, VerbRemove, VerbStatus}

	for _, host := range hosts {
		for _, verb := range verbs {
			t.Run(string(host)+" "+verb.String(), func(t *testing.T) {
				ev := full
				ev.Host = host
				if actions := Plan(verb, []Evidence{ev}); len(actions) != 0 {
					t.Errorf("planned %d actions for %q, want none", len(actions), host)
				}
			})
		}
	}
}

// TestPlanUnknownVerbIsSilent keeps a verb the tier tables do not cover from
// planning anything at all.
func TestPlanUnknownVerbIsSilent(t *testing.T) {
	t.Parallel()
	ev := []Evidence{{Host: HostClaudeCode, CLIPresent: true, ListingOK: true, Listed: true}}
	if actions := Plan(Verb(99), ev); len(actions) != 0 {
		t.Errorf("planned %d actions for an unknown verb, want none", len(actions))
	}
}

// TestPlanIsPure proves planning has no state: the same evidence plans the same
// actions every time, and no evidence plans nothing.
func TestPlanIsPure(t *testing.T) {
	t.Parallel()
	evidence := []Evidence{
		{Host: HostClaudeCode, CLIPresent: true, ListingOK: true, Listed: true},
		{Host: HostCursor, RegistryListed: true},
		{Host: HostCodexCLI, RegistryListed: true},
		{Host: HostCopilot, CLIPresent: true, ListingOK: true},
	}
	verbs := []Verb{VerbInstall, VerbUpdate, VerbRemove, VerbStatus}

	for _, verb := range verbs {
		t.Run("repeat "+verb.String(), func(t *testing.T) {
			first := Plan(verb, evidence)
			second := Plan(verb, evidence)
			if !reflect.DeepEqual(first, second) {
				t.Errorf("two plans from one evidence set differ:\n%+v\n%+v", first, second)
			}
		})
		t.Run("no evidence "+verb.String(), func(t *testing.T) {
			if actions := Plan(verb, nil); len(actions) != 0 {
				t.Errorf("planned %d actions from no evidence, want none", len(actions))
			}
			if actions := Plan(verb, []Evidence{}); len(actions) != 0 {
				t.Errorf("planned %d actions from empty evidence, want none", len(actions))
			}
		})
	}
}

// TestPlanDoesNotMutateItsInput proves the caller's evidence survives planning
// unchanged, which is half of what makes two entry points comparable.
func TestPlanDoesNotMutateItsInput(t *testing.T) {
	t.Parallel()
	evidence := []Evidence{
		{Host: HostCopilot, CLIPresent: true, ListingOK: true, Listed: true, ListedVersion: "2.0.0"},
		{Host: HostClaudeCode, RegistryListed: true},
	}
	before := make([]Evidence, len(evidence))
	copy(before, evidence)

	Plan(VerbUpdate, evidence)

	if !reflect.DeepEqual(before, evidence) {
		t.Errorf("Plan rewrote its input:\n%+v\n%+v", before, evidence)
	}
}

// TestPlanIsDeterministicAcrossEvidenceOrder is the invariant of
// plugin-delivery.spec made testable: the plugin step of `archcore update`
// and `archcore plugin update` collect evidence separately, so their plans may
// only be compared if the order evidence arrives in cannot change the plan.
func TestPlanIsDeterministicAcrossEvidenceOrder(t *testing.T) {
	t.Parallel()
	claude := Evidence{Host: HostClaudeCode, CLIPresent: true, ListingOK: true, Listed: true}
	cursor := Evidence{Host: HostCursor, RegistryListed: true}
	codex := Evidence{Host: HostCodexCLI, RegistryListed: true}
	copilot := Evidence{Host: HostCopilot, CLIPresent: true, ListingOK: true, Listed: true}

	tests := []struct {
		name     string
		evidence []Evidence
	}{
		{name: "canonical order", evidence: []Evidence{claude, cursor, codex, copilot}},
		{name: "reversed", evidence: []Evidence{copilot, codex, cursor, claude}},
		{name: "shuffled", evidence: []Evidence{codex, claude, copilot, cursor}},
		{name: "cursor first", evidence: []Evidence{cursor, copilot, claude, codex}},
	}

	want := Plan(VerbUpdate, tests[0].evidence)
	wantHosts := []Host{HostClaudeCode, HostCursor, HostCodexCLI, HostCopilot}
	if len(want) != len(wantHosts) {
		t.Fatalf("the reference plan has %d actions, want %d", len(want), len(wantHosts))
	}
	for i, host := range wantHosts {
		if want[i].Host != host {
			t.Fatalf("plan[%d] is for %q, want the canonical order %q", i, want[i].Host, host)
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Plan(VerbUpdate, tt.evidence); !reflect.DeepEqual(got, want) {
				t.Errorf("plan differs from the canonical-order plan:\n%+v\n%+v", got, want)
			}
		})
	}
}

// TestPlanTakesTheFirstObservationOfAHost pins the answer to a collector that
// reported one host twice. The first entry wins, so a duplicate cannot flip the
// plan depending on which observation arrived last.
func TestPlanTakesTheFirstObservationOfAHost(t *testing.T) {
	t.Parallel()
	evidence := []Evidence{
		{Host: HostClaudeCode, CLIPresent: true, ListingOK: true, Listed: true},
		{Host: HostClaudeCode, CLIPresent: true, ListingOK: true},
	}
	actions := Plan(VerbUpdate, evidence)
	if len(actions) != 1 {
		t.Fatalf("planned %d actions, want 1", len(actions))
	}
	if actions[0].Kind != ActionRun {
		t.Errorf("kind = %s, want %s from the first observation", actions[0].Kind, ActionRun)
	}
}

// TestPlanFlagsTheAutoUpdateEntryOnlyForClaudeCode keeps both settings flags
// bound to the host and the verbs requirements 14 and 17 attach them to.
//
// The merge rides two install kinds, not one. A run merges after a successful
// install; a report-installed merges without running anything, because an
// install whose settings write failed once reports "already installed" on every
// later run, and the entry would then never be written by any path. Removal is
// the mirror, and only where a command actually runs: the print tier hands the
// user an uninstall line, and the entry goes when the removal does.
func TestPlanFlagsTheAutoUpdateEntryOnlyForClaudeCode(t *testing.T) {
	t.Parallel()
	shapes := []Evidence{
		{},
		{RegistryListed: true},
		{CLIPresent: true},
		{CLIPresent: true, ListingOK: true},
		{CLIPresent: true, ListingOK: true, Listed: true},
		{CLIPresent: true, ListingOK: true, Listed: true, RegistryListed: true},
	}
	for _, verb := range []Verb{VerbInstall, VerbUpdate, VerbRemove, VerbStatus} {
		for _, host := range Hosts() {
			for i, shape := range shapes {
				ev := shape
				ev.Host = host
				for _, action := range Plan(verb, []Evidence{ev}) {
					isClaude := host == HostClaudeCode
					wantMerge := isClaude && verb == VerbInstall &&
						(action.Kind == ActionRun || action.Kind == ActionReportInstalled)
					wantRemove := isClaude && verb == VerbRemove && action.Kind == ActionRun
					if action.MergeAutoUpdate != wantMerge {
						t.Errorf("%s %s shape %d (%s): MergeAutoUpdate = %v, want %v",
							host, verb, i, action.Kind, action.MergeAutoUpdate, wantMerge)
					}
					if action.RemoveAutoUpdate != wantRemove {
						t.Errorf("%s %s shape %d (%s): RemoveAutoUpdate = %v, want %v",
							host, verb, i, action.Kind, action.RemoveAutoUpdate, wantRemove)
					}
				}
			}
		}
	}
}

// TestPlanEmitsOneActionPerAddressedHost proves a mixed evidence set produces
// one action per host the tiers address and nothing for the rest.
func TestPlanEmitsOneActionPerAddressedHost(t *testing.T) {
	t.Parallel()
	evidence := []Evidence{
		{Host: HostClaudeCode, CLIPresent: true, ListingOK: true, Listed: true}, // runs
		{Host: HostCursor}, // no evidence: silent
		{Host: HostCodexCLI, CLIPresent: true, ListingOK: true}, // listing omits it: silent
		{Host: HostCopilot, RegistryListed: true},               // prints the command
	}
	actions := Plan(VerbUpdate, evidence)
	if len(actions) != 2 {
		t.Fatalf("planned %d actions, want 2: %+v", len(actions), actions)
	}
	if actions[0].Host != HostClaudeCode || actions[0].Kind != ActionRun {
		t.Errorf("first action = %q %s, want claude-code run", actions[0].Host, actions[0].Kind)
	}
	if actions[1].Host != HostCopilot || actions[1].Kind != ActionPrintCommand {
		t.Errorf("second action = %q %s, want copilot print-command", actions[1].Host, actions[1].Kind)
	}
}
