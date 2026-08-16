package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/api"
	"archcore-cli/internal/config"
	"archcore-cli/internal/display"
	"archcore-cli/internal/plugin"
	"archcore-cli/internal/wiring"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// skipAgentSentinel is the AgentID value used to represent the "Skip" option in
// the agent picker. Empty AgentID is reserved for this sentinel and never
// matches a real agent.
const skipAgentSentinel agents.AgentID = ""

type initResult struct {
	serverReachable bool // only meaningful when ServerURL != ""
}

// pickOutcome enumerates the possible outcomes of the agent picker. Using a
// single enum (rather than parallel bools) makes invalid states like
// "aborted+picked" unrepresentable.
type pickOutcome int

const (
	outcomePicked pickOutcome = iota // user checked one or more real agents in the picker (len(agents) > 0)

	// outcomeDetected is not a second name for outcomePicked. Both carry agents
	// to wire and only one carries consent to install a plugin: a detected host
	// was never disclosed on a screen and no box was checked for it, so the
	// delivery step must stay off it — plugin-delivery.spec, Invariant. The two
	// arrive at installAgents with agent lists that look alike, so the fact is
	// recorded here instead of being re-derived downstream, where it cannot be.
	outcomeDetected // the project already carries a host's config (len(agents) > 0)

	outcomeSkipped        // user explicitly chose the "Skip" sentinel
	outcomeAborted        // user pressed Ctrl+C (huh.ErrUserAborted)
	outcomeNonInteractive // no /dev/tty available, picker was not run
)

// agentSelection captures the outcome of the interactive agent picker.
// agents is non-empty iff outcome is outcomePicked or outcomeDetected.
type agentSelection struct {
	outcome pickOutcome
	agents  []*agents.Agent
}

// agentPicker is the test seam for the interactive agent picker. Production
// uses defaultPickAgents; tests swap it with a stub.
type agentPicker func() (agentSelection, error)

// instructionsConfirmer asks the user whether to write the Archcore usage hint
// into the listed (relative) instruction-file paths.
type instructionsConfirmer func(paths []string) (bool, error)

var (
	isInteractive                             = defaultIsInteractive
	pickAgents          agentPicker           = defaultPickAgents
	confirmInstructions instructionsConfirmer = defaultConfirmInstructions
)

// runInit performs the init logic after prompts have been resolved.
// settings is a fully constructed Settings value (from NewNoneSettings, etc.)
func runInit(ctx context.Context, baseDir string, settings *config.Settings) (*initResult, error) {
	if err := config.InitDir(baseDir); err != nil {
		return nil, fmt.Errorf("creating .archcore/ directory: %w", err)
	}

	result := &initResult{}

	if err := config.Save(baseDir, settings); err != nil {
		return nil, fmt.Errorf("saving settings: %w", err)
	}

	if serverURL := settings.ServerURL(); serverURL != "" {
		client := api.NewClient(serverURL)
		if err := client.CheckHealth(ctx); err != nil {
			// Soft failure: an unreachable server must not abort init — the
			// directory and settings are already on disk and the remaining
			// setup (agent detection, hooks, MCP config) is local-only.
			fmt.Println(display.WarnLine(fmt.Sprintf("cannot reach server at %s: %v", serverURL, err)))
		} else {
			result.serverReachable = true
		}
	}

	return result, nil
}

// existingOrNewSettings returns the settings a reinitialize should write, and
// whether they came from the project rather than from the defaults.
//
// Reinitializing restores the directory and the host wiring. It is not a reset
// of the sync mode, project id, language and globals the user configured, and
// nothing on this path warned that it would be — `archcore init --yes` in a
// script silently discarded all four on a configured repository. The --agent
// path has always kept them (wiring.EnsureProjectInitialized), so overwriting
// here also made the two entry points disagree about what init means.
//
// Settings that cannot be read fall back to the defaults: the file is being
// rewritten either way, and refusing would leave a project unable to
// reinitialize out of exactly the state reinitializing repairs.
func existingOrNewSettings(baseDir string) (settings *config.Settings, kept bool) {
	if !config.DirExists(baseDir) {
		return config.NewNoneSettings(), false
	}
	existing, err := config.Load(baseDir)
	if err != nil || existing == nil {
		return config.NewNoneSettings(), false
	}
	return existing, true
}

func newInitCmd(version string) *cobra.Command {
	var (
		agentFlags  []string
		projectFlag string
		assumeYes   bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize archcore in the current directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// --agent switches to the non-interactive mode used by the plugin
			// skill, CI, and scripts: no TTY prompts, explicit consent implied
			// by the flag, project root resolved flag > env > cwd.
			if len(agentFlags) > 0 {
				baseDir, err := resolveProjectRoot(projectFlag, os.Getenv("ARCHCORE_PROJECT_ROOT"))
				if err != nil {
					return err
				}
				return runInitForAgents(ctx, baseDir, agentFlags, version)
			}

			fmt.Println(display.WelcomeBanner(version))
			fmt.Println()

			cwd, err := resolveProjectRoot(projectFlag, os.Getenv("ARCHCORE_PROJECT_ROOT"))
			if err != nil {
				return err
			}

			// --yes answers this one prompt, and grants nothing beyond it. The
			// selection screen still decides which hosts are wired, and the
			// delivery step below turns --yes into printed commands rather than
			// installs: a flag is not a checked host, and consent to install a
			// plugin is carried by nothing else — plugin-delivery.spec, Invariant.
			if config.DirExists(cwd) && !assumeYes {
				var reinit bool
				err := huh.NewConfirm().
					Title(".archcore/ already exists. Reinitialize?").
					Value(&reinit).
					Run()
				if err != nil {
					if errors.Is(err, huh.ErrUserAborted) {
						fmt.Println(display.Dim.Render("  Cancelled."))
						return nil
					}
					return err
				}
				if !reinit {
					fmt.Println(display.Dim.Render("  Cancelled."))
					return nil
				}
			}

			settings, kept := existingOrNewSettings(cwd)

			result, err := runInit(ctx, cwd, settings)
			if err != nil {
				return err
			}

			fmt.Println(display.CheckLine("Created .archcore/ directory"))
			if result.serverReachable {
				fmt.Println(display.CheckLine("Server is reachable"))
			}
			if kept {
				fmt.Println(display.CheckLine("Existing .archcore/settings.json kept"))
			} else {
				fmt.Println(display.CheckLine("Settings saved to .archcore/settings.json"))
			}

			// Auto-detect agents and install hooks + MCP config for all found.
			// If none detected, ask the user to pick from supported agents. A
			// picker error is shown as a warning so the user still sees the
			// "Ready!" line — .archcore/ is already on disk and they can rerun
			// 'archcore mcp install --agent <id>' later.
			sel, err := resolveAgents(cwd)
			if err != nil {
				fmt.Println(display.WarnLine(fmt.Sprintf("agent picker failed: %v", err)))
				fmt.Println(display.Dim.Render(
					"  Run 'archcore mcp install --agent <id>' (or 'archcore hooks install --agent <id>') later."))
			} else if len(sel.agents) == 0 {
				printAgentSelectionStatus(sel)
			} else {
				// The overlap notice is deferred: which of its two wordings is
				// true depends on whether the delivery step below installs a
				// plugin, and printing from both places emitted the two
				// contradicting each other.
				anyHooks := installAgents(cwd, sel.agents, false)
				maybeInstallInstructions(cwd, sel.agents)
				// Wiring is the same for both selections; delivery is not. A
				// detected host reached this line without a screen and without a
				// checked box, so it gets the offer and never the install. The
				// two agent lists are indistinguishable by now, which is why the
				// outcome carries the difference. Output goes where every line
				// above went — the writer parameter exists so a test can read the
				// step's own lines.
				// A switch on the outcome that grants consent, not on the one
				// that withholds it. Written the other way round — hint for
				// outcomeDetected, install for everything else — every outcome
				// added later lands in the install arm by default, which is the
				// wrong direction for a branch the Invariant of
				// plugin-delivery.spec makes load-bearing.
				//
				reportedOverlap := false
				//exhaustive:ignore // outcomeNonInteractive and outcomeSkipped carry no agents and cannot reach this block; the default answers them anyway.
				switch sel.outcome {
				case outcomePicked:
					// --yes and a session with no terminal mean the same thing
					// here: nobody was present to check the box this install
					// would stand on, so the step prints the commands instead —
					// plugin-delivery.spec §7.
					//
					// The terminal check is defensive rather than reachable:
					// resolveAgents opens the picker only when isInteractive
					// answered true, so a picked outcome and a missing terminal
					// cannot coexist on this path today. It is read here anyway
					// because this line, not the picker, is the last thing
					// between a selection and an unattended install — the seam
					// is swappable, and requirement 7 binds the step that acts.
					reportedOverlap = deliverPlugins(ctx, os.Stdout, sel.agents, assumeYes || !isInteractive())
				default:
					printPluginInstallHint(os.Stdout, sel.agents)
				}
				// The other wording, for a plugin this run found rather than
				// installed. Skipped when the step above already spoke.
				if anyHooks && !reportedOverlap {
					printPluginConflictNote()
				}
			}

			fmt.Println()
			fmt.Println(display.Success.Render("  Ready! Run 'archcore status' to verify."))
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&agentFlags, "agent", nil,
		"non-interactive: initialize and install hooks + MCP config + usage hint for the given agent id (repeatable; e.g. claude-code, cursor, codex-cli, copilot)")
	cmd.Flags().StringVar(&projectFlag, "project", "",
		"project root to initialize (default: current directory; env: ARCHCORE_PROJECT_ROOT)")
	cmd.Flags().BoolVar(&assumeYes, "yes", false,
		"answer the reinitialize prompt with yes; without --agent the plugin step prints the per-host install commands instead of running them")
	return cmd
}

// runInitForAgents is the non-interactive init path behind --agent. Contract
// (init_agent_flag_spec_test.go): validate every agent id before any write;
// never open a TTY prompt; write all artifacts under baseDir regardless of
// process cwd; keep existing .archcore/ settings untouched (idempotent pass);
// explicit --agent implies consent for the usage-hint instructions, and for the
// plugin of every host it names — plugin-delivery.spec §6.
func runInitForAgents(ctx context.Context, baseDir string, agentIDs []string, version string) error {
	list := make([]*agents.Agent, 0, len(agentIDs))
	for _, id := range agentIDs {
		agent := agents.ByID(agents.AgentID(id))
		if agent == nil {
			return fmt.Errorf("unknown agent %q — valid agents: %v", id, agents.AllIDs())
		}
		list = append(list, agent)
	}

	fmt.Println(display.WelcomeBanner(version))
	fmt.Println()

	created, err := wiring.EnsureProjectInitialized(baseDir)
	if err != nil {
		return err
	}
	if created {
		fmt.Println(display.CheckLine("Created .archcore/ directory"))
		fmt.Println(display.CheckLine("Settings saved to .archcore/settings.json"))
	} else {
		fmt.Println(display.CheckLine("Existing .archcore/ kept (settings untouched)"))
	}

	anyHooks := installAgents(baseDir, list, false)
	if targets := wiring.DedupeByInstructionsPath(baseDir, list); len(targets) > 0 {
		installInstructionsForAgents(baseDir, targets)
	}
	// The flag named the hosts, so it carries their plugins too, with or without
	// a terminal — plugin-delivery.spec §6 and §11. A CI variable is the one
	// thing that turns this into printed commands, and deliverPlugins reads it
	// itself so both call sites cannot answer §8 differently.
	if !deliverPlugins(ctx, os.Stdout, list, false) && anyHooks {
		// No plugin was installed here, so the overlap — if any — belongs to a
		// plugin that was already in place, and takes the other wording.
		printPluginConflictNote()
	}

	fmt.Println()
	fmt.Println(display.Success.Render("  Ready! Run 'archcore status' to verify."))
	return nil
}

// resolveAgents detects installed agents in baseDir; if none are found and
// stdin is interactive, it prompts the user via pickAgents. Returns an
// agentSelection that callers translate into user-facing messages via
// printAgentSelectionStatus.
//
// The detection branch reports outcomeDetected rather than outcomePicked. Every
// caller that only wires hosts treats the two alike; the one that also delivers
// plugins may not, because detection is not consent.
func resolveAgents(baseDir string) (agentSelection, error) {
	detected := agents.Detect(baseDir)
	if len(detected) != 0 {
		return agentSelection{outcome: outcomeDetected, agents: detected}, nil
	}
	if !isInteractive() {
		return agentSelection{outcome: outcomeNonInteractive}, nil
	}
	return pickAgents()
}

// printAgentSelectionStatus prints a user-facing message describing the
// outcome of the agent picker when no agents will be installed. Each branch
// uses a distinct anchor word ("Cancelled", "Skipped", "non-interactive") so
// tests can assert on stable substrings.
func printAgentSelectionStatus(sel agentSelection) {
	//exhaustive:ignore // outcomePicked and outcomeDetected are the success paths and print nothing — this switch reports only the ways selection ended without agents.
	switch sel.outcome {
	case outcomeAborted:
		fmt.Println(display.Dim.Render(
			"  Cancelled — .archcore/ is in place. Run 'archcore mcp install --agent <id>' (or 'archcore hooks install --agent <id>') later."))
	case outcomeSkipped:
		fmt.Println(display.Dim.Render(
			"  Skipped agent setup. Run 'archcore mcp install --agent <id>' (or 'archcore hooks install --agent <id>') later."))
	case outcomeNonInteractive:
		fmt.Println(display.Dim.Render(
			"  No AI agent selected (non-interactive). Run 'archcore mcp install --agent <id>' later."))
	}
}

// validateAgentSelection enforces that the user picked at least one option
// (real agent or the Skip sentinel) before submitting the MultiSelect. Without
// this, hitting Enter without pressing Space yields an empty selection that
// silently bypasses the entire agent install loop.
func validateAgentSelection(v []agents.AgentID) error {
	if len(v) == 0 {
		return errors.New("press space (or x) to toggle, then enter; or pick \"Skip — configure later\"")
	}
	return nil
}

// defaultPickAgents asks the user to pick which agents to install hooks + MCP
// config for. The "Skip — configure later" sentinel is the last option and is
// filtered out of the returned agents; if it was the only thing picked, the
// selection's outcome is outcomeSkipped.
func defaultPickAgents() (agentSelection, error) {
	all := agents.All()
	options := make([]huh.Option[agents.AgentID], 0, len(all)+1)
	for _, a := range all {
		options = append(options, huh.NewOption(agentPickerLabel(a), a.ID))
	}
	options = append(options, huh.NewOption("Skip — configure later", skipAgentSentinel))

	var picked []agents.AgentID
	err := huh.NewMultiSelect[agents.AgentID]().
		Title("No AI agent auto-detected. Select agents to configure (space to toggle, enter to confirm)").
		// The disclosure that is the same for every marked host. The per-host
		// half is in the labels — plugin-delivery.spec §1.
		Description("A checked host is also the consent to install the Archcore plugin on it. Nothing is installed for a host you leave unchecked.").
		Options(options...).
		Validate(validateAgentSelection).
		Value(&picked).
		Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return agentSelection{outcome: outcomeAborted}, nil
		}
		return agentSelection{}, err
	}

	return agentsFromPicked(picked), nil
}

// agentsFromPicked translates a list of picked AgentIDs (from huh's
// MultiSelect) into an agentSelection. The Skip sentinel is filtered out, and
// unknown IDs (defensive — the picker only offers registered IDs) are
// dropped. If nothing real survives, returns outcomeSkipped: validateAgentSelection
// already guarantees the user pressed *something*, so an empty result here
// means they chose only Skip.
func agentsFromPicked(picked []agents.AgentID) agentSelection {
	result := make([]*agents.Agent, 0, len(picked))
	for _, id := range picked {
		if id == skipAgentSentinel {
			continue
		}
		if a := agents.ByID(id); a != nil {
			result = append(result, a)
		}
	}
	if len(result) == 0 {
		return agentSelection{outcome: outcomeSkipped}
	}
	return agentSelection{outcome: outcomePicked, agents: result}
}

// maybeInstallInstructions offers to write the Archcore usage hint after agent
// setup. It is opt-in: interactive users get a single confirm (default yes);
// non-interactive runs skip with a hint, since the hint lands in user-curated
// files (AGENTS.md, GEMINI.md) that should not be touched without consent. The
// dedicated 'archcore instructions install' command covers automation.
func maybeInstallInstructions(baseDir string, list []*agents.Agent) {
	targets := wiring.DedupeByInstructionsPath(baseDir, list)
	if len(targets) == 0 {
		return
	}

	if !isInteractive() {
		fmt.Println(display.Dim.Render(
			"  Skipped Archcore usage hint (non-interactive). Run 'archcore instructions install' later."))
		return
	}

	paths := make([]string, len(targets))
	for i, agent := range targets {
		paths[i] = wiring.DisplayPath(baseDir, agent.InstructionsPath(baseDir))
	}

	ok, err := confirmInstructions(paths)
	if err != nil || !ok {
		fmt.Println(display.Dim.Render(
			"  Skipped Archcore usage hint. Run 'archcore instructions install' later."))
		return
	}
	installInstructionsForAgents(baseDir, targets)
}

// defaultConfirmInstructions shows the opt-in confirm for writing the usage
// hint. Returns (false, nil) on user abort (Ctrl+C) so init still completes.
func defaultConfirmInstructions(paths []string) (bool, error) {
	add := true
	err := huh.NewConfirm().
		Title("Add an Archcore usage hint so agents auto-discover .archcore/?").
		Description("Writes a managed block to: " + strings.Join(paths, ", ")).
		Value(&add).
		Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, nil
		}
		return false, err
	}
	return add, nil
}

// --- the plugin delivery step -------------------------------------------------
//
// The third entry point over the shared core in @cmd/plugin_run.go, beside the
// `archcore plugin` verbs and the plugin step of `archcore update`: one planner
// decides, one executor acts, and this file chooses only the hosts and the words
// — plugin-delivery.spec.
//
// What is particular to init is consent. A typed verb carries it for the direct
// command; here it is carried by the host selection, and by nothing else — a box
// checked on a screen that disclosed the install, or a host named with --agent.
// The detected-agents path reaches installAgents with a non-empty list too, and
// installs nothing.

// The selection screen's disclosure — plugin-delivery.spec §1 and §2. Two
// constants because the marking is per host: every plugin host states the
// install, and requirement 2 puts the machine-level half on the two whose plugin
// stores have no repository scope at all.
const (
	pluginSelectionNote = "also installs the Archcore plugin"

	// Codex CLI registers its marketplace in ~/.codex and Copilot installs into
	// its user-level store; neither takes a project scope, so checking one in a
	// command about this repository changes the machine.
	pluginSelectionMachineNote = pluginSelectionNote + " machine-level, outside this project"
)

// agentPickerLabel is one option's text on the selection screen. It is a pure
// function of the agent because the disclosure has to be provable: the picker
// itself needs a terminal, and no test here has one.
func agentPickerLabel(a *agents.Agent) string {
	host, ok := plugin.HostFromAgent(string(a.ID))
	if !ok {
		return a.DisplayName
	}
	// The scope comes from the host table, which owns every other per-host fact.
	// Named here as a comparison against two hosts instead, a fifth
	// machine-scoped host would silently get the project-scoped disclosure.
	spec, ok := plugin.SpecFor(host)
	if ok && spec.MachineScoped {
		return a.DisplayName + " — " + pluginSelectionMachineNote
	}
	return a.DisplayName + " — " + pluginSelectionNote
}

// deliverPlugins installs the Archcore plugin on the hosts a selection consents
// to.
//
// It hands back no failure at all: a Constraint of plugin-delivery.spec keeps a
// delivery failure out of init's exit code, and a step that returns no error is
// one no call site can accidentally return. Requirement 18 — an attempted action
// that failed exits nonzero — binds the direct command, not this step.
//
// reportedOverlap is not a failure and not an outcome. It says whether this step
// already printed the duplicate-hook notice in its own wording, so the caller
// knows not to print the other wording after it — the two contradict each other,
// and a run once emitted both.
//
// printOnly is the one fact the step cannot observe for itself: which sentence of
// the consent invariant this run stands on. The selection screen passes true when
// nobody was there to check a box — no terminal (§7), or --yes answering in their
// place — and --agent passes false, because the flag carries consent with or
// without one (§6). A CI variable prints over either (§8), and it is read here so
// the two call sites cannot answer that one differently.
func deliverPlugins(ctx context.Context, w io.Writer, selected []*agents.Agent, printOnly bool) (reportedOverlap bool) {
	hosts := pluginHostsForAgents(selected)
	if len(hosts) == 0 {
		// A selection of hosts that ship no plugin produces no plugin output at
		// all: `archcore init --agent gemini-cli` is a valid wiring run that
		// delivers nothing. Returning before the collector also keeps it from
		// listing plugins on hosts nobody selected.
		return false
	}

	outcome := runPluginActions(ctx, w, pluginRunOptions{
		Verb:      plugin.VerbInstall,
		Hosts:     hosts,
		PrintOnly: printOnly || pluginRunningInCI(),
		// A checked host whose install the step bound never reached gets its
		// commands printed — the Constraints of plugin-delivery.spec. Silence
		// here is the one outcome the selection screen cannot survive: the user
		// consented to the plugin on that host and would otherwise never learn it
		// never arrived.
		PrintUnreachedCommands: true,
		// SettingsPath is left empty, which resolves to the user's own Claude
		// Code settings — the default of §12. Init carries no --scope to move
		// the marketplace entry into the repository.
	})

	// The duplicate-hook notice in the wording that fits a plugin this process
	// just installed — plugin-delivery.spec §15. It answers nothing on a run
	// that mutated no plugin, and the caller then prints the other wording for
	// a plugin that was already in place. Exactly one of the two, never both.
	return reportSelfCausedPluginConflict(w, plugin.VerbInstall, outcome)
}

// printPluginInstallHint offers the plugin for hosts init detected instead of
// asked about. The user saw no disclosure and checked no box, so this path may
// state that a plugin exists and do nothing else — plugin-delivery.spec,
// Invariant.
//
// One line for the whole set. A line per host would nag a user who ran init to
// wire a repository and never asked about plugins at all.
func printPluginInstallHint(w io.Writer, detected []*agents.Agent) {
	hosts := pluginHostsForAgents(detected)
	if len(hosts) == 0 {
		return
	}

	names := make([]string, 0, len(hosts))
	for _, host := range hosts {
		names = append(names, pluginHostName(host))
	}
	fmt.Fprintln(w, display.Dim.Render(fmt.Sprintf(
		"  An Archcore plugin ships for %s — run 'archcore plugin install' to add it.",
		strings.Join(names, ", "))))
}

// pluginHostsForAgents maps a host selection onto the plugin hosts it consents
// to. An agent that ships no plugin contributes nothing: gemini-cli, opencode,
// roo-code and cline are ordinary wiring selections, and only a typed
// `archcore plugin --agent <id>` refuses them (@cmd/plugin_run.go).
//
// The result follows the canonical host order rather than the selection's or the
// dedupe map's, so the same set of hosts is observed and printed in the same
// order on every run — bounded-and-deterministic-output.rule.
func pluginHostsForAgents(selected []*agents.Agent) []plugin.Host {
	chosen := make(map[plugin.Host]bool, len(selected))
	for _, agent := range selected {
		if host, ok := plugin.HostFromAgent(string(agent.ID)); ok {
			chosen[host] = true
		}
	}

	hosts := make([]plugin.Host, 0, len(chosen))
	for _, host := range plugin.Hosts() {
		if chosen[host] {
			hosts = append(hosts, host)
		}
	}
	return hosts
}

// defaultIsInteractive reports whether prompts can be shown. Used to skip
// prompts in CI / tests where /dev/tty is unavailable. We probe /dev/tty
// directly because huh opens it explicitly — stdin being a char device isn't
// sufficient.
func defaultIsInteractive() bool {
	f, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
