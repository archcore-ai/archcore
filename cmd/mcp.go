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
	"archcore-cli/internal/projectroot"
)

func newMCPCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP stdio server for archcore documents",
		Long:  "Starts an MCP (Model Context Protocol) stdio server that exposes archcore document tools.",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := resolveProjectRoot(cmd, projectroot.ModeRuntime)
			if err != nil {
				return err
			}
			cwd := res.Path

			fmt.Fprintln(os.Stderr, display.WelcomeBanner())
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, mcpStartupBanner(version, res))
			if !config.DirExists(cwd) {
				fmt.Fprintln(os.Stderr, display.Dim.Render("  MCP server running on stdio (uninitialized project — only init_project tool is useful until the agent initializes .archcore/)..."))
			} else {
				fmt.Fprintln(os.Stderr, display.Dim.Render("  MCP server running on stdio..."))
			}

			return mcpserver.RunStdio(cwd,
				mcpserver.WithResolution(res),
				mcpserver.WithVersion(version),
			)
		},
	}

	cmd.AddCommand(newMCPInstallCmd())
	return cmd
}

// mcpStartupBanner formats the single-line stderr banner that MCP host logs
// capture. Plain text (grep-stable, no ANSI codes) — host log viewers don't
// render styles and the line should be copy-pasteable into bug reports.
func mcpStartupBanner(version string, res *projectroot.Resolution) string {
	guards := "strict"
	if res != nil && res.LegacyMode {
		guards = "legacy"
	}
	v := version
	if v == "" {
		v = "dev"
	}
	path := ""
	source := ""
	if res != nil {
		path = res.Path
		source = string(res.Source)
	}
	return fmt.Sprintf("archcore %s | base=%s | source=%s | guards=%s", v, path, source, guards)
}

func newMCPInstallCmd() *cobra.Command {
	var agentFlag string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install MCP server config for coding agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := resolveProjectRoot(cmd, projectroot.ModeRuntime)
			if err != nil {
				return err
			}
			cwd := res.Path
			if !config.DirExists(cwd) {
				return errors.New(".archcore/ not found — run 'archcore init' first")
			}

			if agentFlag != "" {
				return runMCPInstallForAgent(cwd, agents.AgentID(agentFlag))
			}
			return runMCPInstallAutoDetect(cwd)
		},
	}

	cmd.Flags().StringVar(&agentFlag, "agent", "", "install for a specific agent (e.g. cursor, gemini-cli)")
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
	detected, err := resolveAgents(baseDir)
	if err != nil {
		return err
	}
	if len(detected) == 0 {
		fmt.Println(display.Dim.Render("  No AI agent selected."))
		return nil
	}

	for _, agent := range detected {
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
