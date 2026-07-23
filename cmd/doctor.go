package cmd

import (
	"errors"
	"fmt"
	"os"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/api"
	"archcore-cli/internal/config"
	"archcore-cli/internal/display"
	"archcore-cli/internal/wiring"

	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	var (
		fix         bool
		fixAgents   []string
		projectFlag string
	)
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check your archcore setup and fix issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if len(fixAgents) > 0 && !fix {
				return errors.New("--agent requires --fix")
			}

			fmt.Println(display.Banner())
			fmt.Println()

			cwd, err := resolveProjectRoot(projectFlag, os.Getenv("ARCHCORE_PROJECT_ROOT"))
			if err != nil {
				return fmt.Errorf("resolving project root: %w", err)
			}

			issues := 0

			// Fix issues before checking (only when --fix is passed).
			if fix {
				removed, err := fixManifest(cwd)
				if err != nil {
					issues++
					fmt.Println(display.FailLine(err.Error()))
				} else if removed > 0 {
					fmt.Println(display.CheckLine(fmt.Sprintf("Removed %d orphaned relation(s)", removed)))
				}
				if config.DirExists(cwd) {
					issues += convergeHostWiring(cwd, fixAgents)
				}
			}

			// Run status checks (structure + documents).
			issues += runStatusChecks(cwd)

			// Early return if .archcore/ doesn't exist — settings and
			// server checks depend on it.
			if !config.DirExists(cwd) {
				fmt.Println()
				fmt.Println(display.Warn.Render(fmt.Sprintf("  %d issue(s) found", issues)))
				return ErrAlreadyReported
			}

			// Settings file valid.
			settings, err := config.Load(cwd)
			if err != nil {
				issues++
				fmt.Println(display.FailLine("Settings file missing or invalid"))
				fmt.Println(display.HintLine(err.Error()))
				fmt.Println(display.HintLine("Run 'archcore init' to reconfigure"))
			} else {
				fmt.Println(display.CheckLine(fmt.Sprintf("Settings valid (sync: %s)", settings.Sync)))
				// To stderr so structured doctor output on stdout is unaffected.
				warnUnknownConfigFields(os.Stderr, settings)
			}

			// Server reachable (only when a server URL is configured).
			if settings != nil {
				if serverURL := settings.ServerURL(); serverURL != "" {
					client := api.NewClient(serverURL)
					if err := client.CheckHealth(ctx); err != nil {
						issues++
						fmt.Println(display.FailLine("Server unreachable at " + serverURL))
						fmt.Println(display.HintLine(err.Error()))
					} else {
						fmt.Println(display.CheckLine("Server is reachable"))
					}
				}
			}

			fmt.Println()
			if issues == 0 {
				fmt.Println(display.Success.Render("  All checks passed!"))
			} else {
				fmt.Println(display.Warn.Render(fmt.Sprintf("  %d issue(s) found", issues)))
				return fmt.Errorf("%d issue(s) found", issues)
			}

			return nil
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "Automatically fix issues (orphaned relations, host-wiring drift)")
	cmd.Flags().StringArrayVar(&fixAgents, "agent", nil,
		"with --fix: converge host wiring for the given agent id(s) (repeatable; default: all auto-detected)")
	cmd.Flags().StringVar(&projectFlag, "project", "",
		"project root to check (default: current directory; env: ARCHCORE_PROJECT_ROOT)")
	return cmd
}

// convergeHostWiring re-runs the host-wiring installers in converge mode for
// the requested (or auto-detected) agents: stale hook commands are updated in
// place (marker-based probes), drifted MCP entries are rewritten to the
// desired shape (e.g. a Cursor config predating --project ${workspaceFolder}),
// and the instructions managed block is refreshed. Returns the number of
// failures; per-agent errors don't abort the loop.
func convergeHostWiring(baseDir string, agentIDs []string) int {
	var list []*agents.Agent
	if len(agentIDs) > 0 {
		for _, id := range agentIDs {
			agent := agents.ByID(agents.AgentID(id))
			if agent == nil {
				fmt.Println(display.FailLine(fmt.Sprintf("unknown agent %q — valid agents: %v", id, agents.AllIDs())))
				return 1
			}
			list = append(list, agent)
		}
	} else {
		list = agents.Detect(baseDir)
		if len(list) == 0 {
			return 0 // nothing detected, nothing to converge
		}
	}

	failures := 0
	for _, agent := range list {
		// Hooks: the installers are update-capable — a stale archcore command
		// is rewritten in place, duplicates are healed.
		if _, err := wiring.InstallHooksForAgent(baseDir, agent); err != nil {
			failures++
			fmt.Println(display.FailLine(fmt.Sprintf("%s hooks: %v", agent.DisplayName, err)))
		}

		changed, err := convergeMCPForAgent(baseDir, agent)
		switch {
		case err != nil:
			failures++
			fmt.Println(display.FailLine(fmt.Sprintf("%s MCP: %v", agent.DisplayName, err)))
		case changed:
			fmt.Println(display.CheckLine(fmt.Sprintf("Updated MCP config for %s", agent.DisplayName)))
		}
	}

	// Instructions managed blocks are converge-friendly by construction
	// (owned file / fenced-block upsert): re-installing refreshes content.
	installInstructionsForAgents(baseDir, list)

	return failures
}

// convergeMCPForAgent updates a drifted archcore MCP entry for hosts with a
// known desired shape; other hosts fall back to the plain (presence-ensuring)
// writer.
func convergeMCPForAgent(baseDir string, agent *agents.Agent) (bool, error) {
	if agent.ManualMCPInstallHint != "" || agent.MCPConfigPath == nil {
		return false, nil
	}
	path := agent.MCPConfigPath(baseDir)
	switch agent.ID {
	case agents.Cursor:
		return agents.ConvergeCursorMCPJSON(path)
	case agents.ClaudeCode, agents.GeminiCLI, agents.RooCode:
		return agents.ConvergeStandardMCPJSON(path)
	case agents.Copilot:
		return agents.ConvergeVSCodeMCPJSON(path)
	default:
		return false, agent.WriteMCPConfig(baseDir)
	}
}
