package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/api"
	"archcore-cli/internal/config"
	"archcore-cli/internal/display"
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
	outcomePicked         pickOutcome = iota // user picked one or more real agents (len(agents) > 0)
	outcomeSkipped                           // user explicitly chose the "Skip" sentinel
	outcomeAborted                           // user pressed Ctrl+C (huh.ErrUserAborted)
	outcomeNonInteractive                    // no /dev/tty available, picker was not run
)

// agentSelection captures the outcome of the interactive agent picker.
// agents is non-empty iff outcome == outcomePicked.
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

func newInitCmd(version string) *cobra.Command {
	var (
		agentFlags  []string
		projectFlag string
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
				return runInitForAgents(baseDir, agentFlags, version)
			}

			fmt.Println(display.WelcomeBanner(version))
			fmt.Println()

			cwd, err := resolveProjectRoot(projectFlag, os.Getenv("ARCHCORE_PROJECT_ROOT"))
			if err != nil {
				return err
			}

			if config.DirExists(cwd) {
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

			settings := config.NewNoneSettings()

			result, err := runInit(ctx, cwd, settings)
			if err != nil {
				return err
			}

			fmt.Println(display.CheckLine("Created .archcore/ directory"))
			if result.serverReachable {
				fmt.Println(display.CheckLine("Server is reachable"))
			}
			fmt.Println(display.CheckLine("Settings saved to .archcore/settings.json"))

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
				installAgents(cwd, sel.agents)
				maybeInstallInstructions(cwd, sel.agents)
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
	return cmd
}

// runInitForAgents is the non-interactive init path behind --agent. Contract
// (init_agent_flag_spec_test.go): validate every agent id before any write;
// never open a TTY prompt; write all artifacts under baseDir regardless of
// process cwd; keep existing .archcore/ settings untouched (idempotent pass);
// explicit --agent implies consent for the usage-hint instructions.
func runInitForAgents(baseDir string, agentIDs []string, version string) error {
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

	installAgents(baseDir, list)
	if targets := wiring.DedupeByInstructionsPath(baseDir, list); len(targets) > 0 {
		installInstructionsForAgents(baseDir, targets)
	}

	fmt.Println()
	fmt.Println(display.Success.Render("  Ready! Run 'archcore status' to verify."))
	return nil
}

// resolveAgents detects installed agents in baseDir; if none are found and
// stdin is interactive, it prompts the user via pickAgents. Returns an
// agentSelection that callers translate into user-facing messages via
// printAgentSelectionStatus.
func resolveAgents(baseDir string) (agentSelection, error) {
	detected := agents.Detect(baseDir)
	if len(detected) != 0 {
		return agentSelection{outcome: outcomePicked, agents: detected}, nil
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
		options = append(options, huh.NewOption(a.DisplayName, a.ID))
	}
	options = append(options, huh.NewOption("Skip — configure later", skipAgentSentinel))

	var picked []agents.AgentID
	err := huh.NewMultiSelect[agents.AgentID]().
		Title("No AI agent auto-detected. Select agents to configure (space to toggle, enter to confirm)").
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
