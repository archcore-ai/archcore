package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/api"
	"archcore-cli/internal/config"
	"archcore-cli/internal/display"
	"archcore-cli/internal/update"
	"archcore-cli/internal/wiring"

	"github.com/spf13/cobra"
)

func newDoctorCmd(version string) *cobra.Command {
	var (
		fix         bool
		fixAgents   []string
		projectFlag string
	)
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check your archcore setup and fix issues",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if len(fixAgents) > 0 && !fix {
				return errors.New("--agent requires --fix")
			}

			fmt.Println(display.Banner(version))
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
			//
			// The update advisory goes above it, not below with the other
			// checks: it diagnoses the tool rather than the project, so it is
			// the one line here that an uninitialized directory must not
			// suppress. That is also where an out-of-date binary is most
			// likely to be met — a machine running `doctor` somewhere it has
			// not run `init`.
			if !config.DirExists(cwd) {
				reportCachedUpdate(os.Stdout, version, update.CachePath())
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

			// Whether each wired host will actually read what was written for
			// it. This belongs in the diagnosis, not in --fix: a config that a
			// host silently ignores is exactly what doctor exists to surface,
			// and reporting it here means a plain `doctor` says so too.
			reportEffectiveHooks(cwd)

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

			// A newer release, when some other path already resolved one. Last
			// of the checks because it diagnoses the tool rather than the
			// project. The uninitialized path prints it above instead, so the
			// one check that does not depend on .archcore/ is never lost to
			// the early return.
			reportCachedUpdate(os.Stdout, version, update.CachePath())

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

// reportEffectiveHooks prints, for every host that actually carries archcore
// hooks, whether it can act on them. Silence means every wired host will.
//
// Wired, not merely detected. agents.Detect answers "is this host used here?"
// from the presence of a .claude/ or .codex/ directory, which says nothing about
// whether archcore ever wrote hooks into it — so an unwired project used to warn
// about Codex configuration it does not have, and, with nothing wired at all,
// still claimed its wired hosts were healthy. The rule scopes this to a wired
// host (report-effective-hook-state.rule, requirement 3).
func reportEffectiveHooks(baseDir string) {
	printed, wired := false, 0
	for _, agent := range agents.Detect(baseDir) {
		if !wiring.CarriesArchcoreHooks(baseDir, agent.ID) {
			continue
		}
		wired++
		for _, note := range wiring.DescribeEffectiveHooks(baseDir, agent.ID) {
			fmt.Println(display.WarnLine(note))
			printed = true
		}
	}
	if note := wiring.DescribePluginConflict(); note != "" {
		fmt.Println(display.WarnLine(note))
		printed = true
	}
	// The reassurance is a claim about the hosts examined, so it is only honest
	// when at least one was.
	if !printed && wired > 0 {
		fmt.Println(display.CheckLine("Wired hosts can act on their hook configs"))
	}
}

// reportCachedUpdate prints one line when the freshness cache already holds a
// version newer than the running one. It reads that cache and nothing else: a
// lookup here would put a local health check behind github.com being reachable,
// and the answer another path already paid for is the one the doctor advisory is
// meant to spend — unattended-update.spec.
//
// The writer is a parameter, as in warnUnknownConfigFields, so the advisory is
// exercisable without a doctor run.
//
// It never touches the issue counter. A user whose only finding is "a newer
// archcore exists" has a healthy project and must still get exit 0 —
// reportEffectiveHooks warns on the same terms.
func reportCachedUpdate(w io.Writer, current, cachePath string) {
	latest, fresh := update.ReadCachedLatest(cachePath)
	// Empty content inside the window is the update path's failure stamp — a
	// recent lookup failed — and not a release. Reading it as an answer would
	// advertise an upgrade to a version nobody ever resolved
	// — unattended-update.spec §8.
	if !fresh || latest == "" {
		return
	}
	// NewerSemver, not NeedsUpdate: NeedsUpdate falls back to "dev is always
	// behind", which nags every locally built binary, and a version that does not
	// parse on either side must stay silent rather than read as one to move to
	// — unattended-update.spec §12.
	if newer, ok := update.NewerSemver(current, latest); !ok || !newer {
		return
	}
	fmt.Fprintln(w, display.WarnLine(fmt.Sprintf(
		"A newer archcore is available: %s (current: %s) — run 'archcore update'", latest, current)))
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
			return 0
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
	// Copilot rides the standard shape: its only project-level MCP source is
	// the workspace-root .mcp.json, the same file (and key, and entry) as
	// claude-code — see copilotMCPPath.
	case agents.ClaudeCode, agents.GeminiCLI, agents.RooCode, agents.Copilot:
		return agents.ConvergeStandardMCPJSON(path)
	default:
		return false, agent.WriteMCPConfig(baseDir)
	}
}
