package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/config"
	"archcore-cli/internal/display"
	mcpserver "archcore-cli/internal/mcp"
)

func newMCPCmd() *cobra.Command {
	var projectFlag string

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP stdio server for archcore documents",
		Long:  "Starts an MCP (Model Context Protocol) stdio server that exposes archcore document tools.",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir, err := resolveProjectRoot(projectFlag, os.Getenv("ARCHCORE_PROJECT_ROOT"))
			if err != nil {
				return err
			}

			fmt.Fprintln(os.Stderr, display.WelcomeBanner())
			fmt.Fprintln(os.Stderr)
			if !config.DirExists(baseDir) {
				fmt.Fprintln(os.Stderr, display.Dim.Render("  MCP server running on stdio (uninitialized project — only init_project tool is useful until the agent initializes .archcore/)..."))
			} else {
				fmt.Fprintln(os.Stderr, display.Dim.Render("  MCP server running on stdio..."))
			}

			return mcpserver.RunStdio(baseDir)
		},
	}

	cmd.Flags().StringVar(&projectFlag, "project", "",
		"project root containing .archcore/ (default: current directory; env: ARCHCORE_PROJECT_ROOT)")

	cmd.AddCommand(newMCPInstallCmd())
	return cmd
}

func newMCPInstallCmd() *cobra.Command {
	var (
		agentFlag   string
		projectFlag string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install MCP server config for coding agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir, err := resolveProjectRoot(projectFlag, os.Getenv("ARCHCORE_PROJECT_ROOT"))
			if err != nil {
				return err
			}
			if !config.DirExists(baseDir) {
				return errors.New(".archcore/ not found — run 'archcore init' first")
			}

			if agentFlag != "" {
				return runMCPInstallForAgent(baseDir, agents.AgentID(agentFlag))
			}
			return runMCPInstallAutoDetect(baseDir)
		},
	}

	cmd.Flags().StringVar(&agentFlag, "agent", "", "install for a specific agent (e.g. cursor, gemini-cli)")
	cmd.Flags().StringVar(&projectFlag, "project", "",
		"project root containing .archcore/ (default: current directory; env: ARCHCORE_PROJECT_ROOT)")
	return cmd
}

// runMCPInstallForAgent installs MCP config for a specific agent.
func runMCPInstallForAgent(baseDir string, id agents.AgentID) error {
	if !config.DirExists(baseDir) {
		return errors.New(".archcore/ not found — run 'archcore init' first")
	}
	agent := agents.ByID(id)
	if agent == nil {
		return fmt.Errorf("unknown agent %q — valid agents: %v", id, agents.AllIDs())
	}
	return installMCPForAgent(baseDir, agent)
}

// runMCPInstallAutoDetect detects agents and installs MCP config for all found.
// If none detected, prompts the user to pick from supported agents.
func runMCPInstallAutoDetect(baseDir string) error {
	sel, err := resolveAgents(baseDir)
	if err != nil {
		fmt.Println(display.WarnLine(fmt.Sprintf("agent picker failed: %v", err)))
		fmt.Println(display.Dim.Render(
			"  Run 'archcore mcp install --agent <id>' to install for a specific agent."))
		return nil
	}
	if len(sel.agents) == 0 {
		printAgentSelectionStatus(sel)
		return nil
	}

	for _, agent := range sel.agents {
		if err := installMCPForAgent(baseDir, agent); err != nil {
			fmt.Println(display.WarnLine(fmt.Sprintf("%s MCP install: %v", agent.DisplayName, err)))
		}
	}
	return nil
}

// installMCPForAgent installs MCP config for a single agent.
func installMCPForAgent(baseDir string, agent *agents.Agent) error {
	if agent.ManualMCPInstallHint != "" {
		fmt.Println(display.WarnLine(fmt.Sprintf("%s: %s", agent.DisplayName, agent.ManualMCPInstallHint)))
		return nil
	}

	if err := agent.WriteMCPConfig(baseDir); err != nil {
		return err
	}
	if agent.MCPConfigPath != nil {
		fmt.Println(display.CheckLine(fmt.Sprintf("Installed MCP config for %s", agent.DisplayName)))
	}
	return nil
}
