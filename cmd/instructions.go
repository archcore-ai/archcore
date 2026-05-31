package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/config"
	"archcore-cli/internal/display"
)

func newInstructionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instructions",
		Short: "Manage the Archcore usage hint in agent instruction files",
		Long: "Writes a short, always-on \"use Archcore\" hint into each detected agent's " +
			"instruction file (AGENTS.md, GEMINI.md, or .claude/rules/archcore.md) so agents " +
			"discover and use the Archcore MCP tools even without the Archcore plugin.",
	}
	cmd.AddCommand(newInstructionsInstallCmd(), newInstructionsRemoveCmd())
	return cmd
}

func newInstructionsInstallCmd() *cobra.Command {
	var agentFlag string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Write the Archcore usage hint into agent instruction files",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if !config.DirExists(cwd) {
				return errors.New(".archcore/ not found — run 'archcore init' first")
			}

			if agentFlag != "" {
				return runInstructionsInstallForAgent(cwd, agents.AgentID(agentFlag))
			}
			return runInstructionsInstallAutoDetect(cwd)
		},
	}

	cmd.Flags().StringVar(&agentFlag, "agent", "", "install for a specific agent (e.g. cursor, gemini-cli)")
	return cmd
}

func newInstructionsRemoveCmd() *cobra.Command {
	var agentFlag string

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove the Archcore usage hint from agent instruction files",
		// Unlike install, remove does not require .archcore/ — it is an uninstall
		// step that must still work after the project has been de-initialized.
		// removeFencedBlock only touches archcore's marked span, so it is safe
		// to run anywhere.
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			if agentFlag != "" {
				return runInstructionsRemoveForAgent(cwd, agents.AgentID(agentFlag))
			}
			// No --agent: clean up every possible target. removeFencedBlock only
			// touches archcore's marked span, so this never harms user content.
			removeInstructionsForAgents(cwd, agents.All())
			return nil
		},
	}

	cmd.Flags().StringVar(&agentFlag, "agent", "", "remove for a specific agent (e.g. cursor, gemini-cli)")
	return cmd
}

// runInstructionsInstallForAgent installs the usage hint for a specific agent.
func runInstructionsInstallForAgent(baseDir string, id agents.AgentID) error {
	agent := agents.ByID(id)
	if agent == nil {
		return fmt.Errorf("unknown agent %q — valid agents: %v", id, agents.AllIDs())
	}
	return installInstructionsForAgent(baseDir, agent)
}

// runInstructionsInstallAutoDetect detects agents and writes the usage hint for
// all found. If none detected, prompts the user to pick.
func runInstructionsInstallAutoDetect(baseDir string) error {
	sel, err := resolveAgents(baseDir)
	if err != nil {
		fmt.Println(display.WarnLine(fmt.Sprintf("agent picker failed: %v", err)))
		fmt.Println(display.Dim.Render(
			"  Run 'archcore instructions install --agent <id>' to install for a specific agent."))
		return nil
	}
	if len(sel.agents) == 0 {
		printInstructionsAgentSelectionStatus(sel)
		return nil
	}

	installInstructionsForAgents(baseDir, sel.agents)
	return nil
}

// runInstructionsRemoveForAgent removes the usage hint for a specific agent.
func runInstructionsRemoveForAgent(baseDir string, id agents.AgentID) error {
	agent := agents.ByID(id)
	if agent == nil {
		return fmt.Errorf("unknown agent %q — valid agents: %v", id, agents.AllIDs())
	}
	return removeInstructionsForAgent(baseDir, agent)
}

func printInstructionsAgentSelectionStatus(sel agentSelection) {
	switch sel.outcome {
	case outcomeAborted:
		fmt.Println(display.Dim.Render(
			"  Cancelled. Run 'archcore instructions install --agent <id>' later."))
	case outcomeSkipped:
		fmt.Println(display.Dim.Render(
			"  Skipped Archcore usage hint. Run 'archcore instructions install --agent <id>' later."))
	case outcomeNonInteractive:
		fmt.Println(display.Dim.Render(
			"  No AI agent selected (non-interactive). Run 'archcore instructions install --agent <id>' later."))
	}
}

// installInstructionsForAgents writes the Archcore usage hint for each agent in
// list, deduped by instruction-file path so the six AGENTS.md agents trigger a
// single write. Per-file failures are warnings, not aborts.
func installInstructionsForAgents(baseDir string, list []*agents.Agent) {
	for _, agent := range dedupeByInstructionsPath(baseDir, list) {
		if err := installInstructionsForAgent(baseDir, agent); err != nil {
			fmt.Println(display.WarnLine(err.Error()))
			continue
		}
	}
}

func installInstructionsForAgent(baseDir string, agent *agents.Agent) error {
	if err := agent.WriteInstructions(baseDir); err != nil {
		return fmt.Errorf("writing %s instructions: %w", agent.DisplayName, err)
	}
	fmt.Println(display.CheckLine(fmt.Sprintf(
		"Added Archcore usage hint to %s", displayPath(baseDir, agent.InstructionsPath(baseDir)))))
	return nil
}

// removeInstructionsForAgents strips the Archcore usage hint for each agent in
// list, deduped by instruction-file path.
func removeInstructionsForAgents(baseDir string, list []*agents.Agent) {
	for _, agent := range dedupeByInstructionsPath(baseDir, list) {
		if err := removeInstructionsForAgent(baseDir, agent); err != nil {
			fmt.Println(display.WarnLine(err.Error()))
			continue
		}
	}
}

func removeInstructionsForAgent(baseDir string, agent *agents.Agent) error {
	if err := agent.RemoveInstructions(baseDir); err != nil {
		return fmt.Errorf("removing %s instructions: %w", agent.DisplayName, err)
	}
	fmt.Println(display.CheckLine(fmt.Sprintf(
		"Removed Archcore usage hint from %s", displayPath(baseDir, agent.InstructionsPath(baseDir)))))
	return nil
}

// dedupeByInstructionsPath returns one agent per unique instruction-file path,
// preserving registry order. Agents without an instruction target are skipped.
func dedupeByInstructionsPath(baseDir string, list []*agents.Agent) []*agents.Agent {
	seen := make(map[string]bool, len(list))
	out := make([]*agents.Agent, 0, len(list))
	for _, agent := range list {
		// Skip agents missing any instruction hook. The registry wires all three
		// together (enforced by TestAllAgents_RequiredFields), so this guard
		// makes the dedup the single safe gate before callers deref the others.
		if agent.InstructionsPath == nil || agent.WriteInstructions == nil || agent.RemoveInstructions == nil {
			continue
		}
		path := agent.InstructionsPath(baseDir)
		if seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, agent)
	}
	return out
}

// displayPath renders an instruction-file path relative to baseDir with forward
// slashes, for stable user-facing and test output. If path is not under baseDir
// (Rel fails or escapes upward), it falls back to the cleaned path unchanged.
func displayPath(baseDir, path string) string {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
