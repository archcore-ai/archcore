package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/config"
	"archcore-cli/internal/display"

	"github.com/spf13/cobra"
)

// hookEntry represents a single hook command configuration.
type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// hookMatcher represents a matcher with its hooks array.
type hookMatcher struct {
	Matcher string      `json:"matcher"`
	Hooks   []hookEntry `json:"hooks"`
}

// archcoreHooks defines the hooks we install for Claude Code, in deterministic order.
var archcoreHooks = []struct {
	Event   string
	Matcher hookMatcher
}{
	{"SessionStart", hookMatcher{Matcher: "", Hooks: []hookEntry{{Type: "command", Command: "archcore hooks claude-code session-start"}}}},
}

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
	var agentFlag string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install archcore hooks for coding agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
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

	cmd.Flags().StringVar(&agentFlag, "agent", "", "install hooks for a specific agent (e.g. cursor, gemini-cli)")
	return cmd
}

// hooksInstallers maps agent IDs to their hooks install functions.
// This is the single source of truth for which agents support hooks.
var hooksInstallers = map[agents.AgentID]func(string) error{
	agents.ClaudeCode: runHooksInstall,
	agents.Cursor:     runCursorHooksInstall,
	agents.GeminiCLI:  runGeminiCLIHooksInstall,
	agents.Copilot:    runCopilotHooksInstall,
}

// installHooksForAgent installs hooks for a single agent that supports them.
// Returns (false, nil) if the agent doesn't support hooks.
func installHooksForAgent(baseDir string, agent *agents.Agent) (bool, error) {
	installer, ok := hooksInstallers[agent.ID]
	if !ok {
		return false, nil
	}
	return true, installer(baseDir)
}

// runHooksInstallForAgent installs hooks for a specific agent by ID.
func runHooksInstallForAgent(baseDir string, id agents.AgentID) error {
	agent := agents.ByID(id)
	if agent == nil {
		return fmt.Errorf("unknown agent %q — valid agents: %v", id, agents.AllIDs())
	}
	installed, err := installHooksForAgent(baseDir, agent)
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
		installed, err := installHooksForAgent(baseDir, agent)
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

// runHooksInstall installs Claude Code hooks into .claude/settings.json.
func runHooksInstall(baseDir string) error {
	if !config.DirExists(baseDir) {
		return errors.New(".archcore/ not found — run 'archcore init' first")
	}

	events := make([]hookEventInstall, len(archcoreHooks))
	for i, h := range archcoreHooks {
		events[i] = hookEventInstall{Event: h.Event, Command: h.Matcher.Hooks[0].Command, Entry: h.Matcher}
	}
	return installHookEvents(hookInstallSpec{
		Path:   filepath.Join(baseDir, ".claude", "settings.json"),
		Probe:  matcherEntryHasCommand,
		Events: events,
	})
}
