package cmd

import (
	"errors"
	"fmt"
	"os"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/config"
	"archcore-cli/internal/display"
	"archcore-cli/internal/wiring"

	"github.com/spf13/cobra"
)

func newHooksCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage agent hooks integration",
	}
	cmd.AddCommand(
		newHooksInstallCmd(),
		newHooksClaudeCodeCmd(version),
		newHooksCursorCmd(version),
		newHooksGeminiCLICmd(version),
		newHooksCopilotCmd(version),
	)
	return cmd
}

func newHooksInstallCmd() *cobra.Command {
	var (
		agentFlag   string
		projectFlag string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install archcore hooks for coding agents",
		// Agent selection is via --agent; reject stray positional args so a
		// mistyped `hooks install cursor` fails loudly instead of silently
		// running auto-detect.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := resolveProjectRoot(projectFlag, os.Getenv("ARCHCORE_PROJECT_ROOT"))
			if err != nil {
				return err
			}
			if !config.DirExists(cwd) {
				return errors.New(".archcore/ not found — run 'archcore init' first")
			}

			if agentFlag != "" {
				return runHooksInstallForAgent(cwd, agents.AgentID(agentFlag))
			}
			return runHooksInstallAutoDetect(cwd)
		},
	}

	cmd.Flags().StringVar(&agentFlag, "agent", "", "install hooks for a single agent (e.g. cursor, gemini-cli); not repeatable — the last value wins")
	cmd.Flags().StringVar(&projectFlag, "project", "",
		"project root containing .archcore/ (default: current directory; env: ARCHCORE_PROJECT_ROOT)")
	return cmd
}

// runHooksInstallForAgent installs hooks for a specific agent by ID.
func runHooksInstallForAgent(baseDir string, id agents.AgentID) error {
	agent := agents.ByID(id)
	if agent == nil {
		return fmt.Errorf("unknown agent %q — valid agents: %v", id, agents.AllIDs())
	}
	installed, err := wiring.InstallHooksForAgent(baseDir, agent)
	if err != nil {
		return err
	}
	if !installed {
		// Hookless agents still get their MCP config, matching the
		// auto-detect path (installAgents).
		fmt.Println(display.WarnLine(fmt.Sprintf("%s does not support hooks", agent.DisplayName)))
	}
	if err := installMCPForAgent(baseDir, agent); err != nil {
		return err
	}
	return nil
}

// runHooksInstallAutoDetect detects agents and installs hooks + MCP config
// for all that support them. If none detected, prompts the user.
func runHooksInstallAutoDetect(baseDir string) error {
	sel, err := resolveAgents(baseDir)
	if err != nil {
		fmt.Println(display.WarnLine(fmt.Sprintf("agent picker failed: %v", err)))
		fmt.Println(display.Dim.Render(
			"  Run 'archcore hooks install --agent <id>' to install for a specific agent."))
		return nil
	}
	if len(sel.agents) == 0 {
		printAgentSelectionStatus(sel)
		return nil
	}

	installAgents(baseDir, sel.agents)
	return nil
}

// installAgents installs hooks (when supported) and MCP config for each agent
// in list, logging per-agent failures as warnings without aborting the loop.
// Shared by 'archcore init' and 'archcore hooks install' auto-detect paths.
func installAgents(baseDir string, list []*agents.Agent) {
	for _, agent := range list {
		installed, err := wiring.InstallHooksForAgent(baseDir, agent)
		if err != nil {
			fmt.Println(display.WarnLine(fmt.Sprintf("%s hooks: %v", agent.DisplayName, err)))
		} else if !installed {
			fmt.Println(display.Dim.Render(fmt.Sprintf("  Skipping hooks for %s (not supported)", agent.DisplayName)))
		}
		if err := installMCPForAgent(baseDir, agent); err != nil {
			fmt.Println(display.WarnLine(fmt.Sprintf("%s MCP: %v", agent.DisplayName, err)))
		}
	}
}
