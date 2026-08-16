package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/config"
	"archcore-cli/internal/display"
	"archcore-cli/internal/docs"
	mcpserver "archcore-cli/internal/mcp"
	"archcore-cli/internal/update"

	"github.com/spf13/cobra"
)

func newMCPCmd(version string) *cobra.Command {
	var projectFlag string

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP stdio server for archcore documents",
		Long:  "Starts an MCP (Model Context Protocol) stdio server that exposes archcore document tools.",
		// Serving takes no positional args; a stray one (`mcp bogus`) must error,
		// not silently start the server ignoring it.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir, err := resolveProjectRoot(projectFlag, os.Getenv("ARCHCORE_PROJECT_ROOT"))
			if err != nil {
				return err
			}

			fmt.Fprintln(os.Stderr, display.WelcomeBanner(version))
			fmt.Fprintln(os.Stderr)
			if !config.DirExists(baseDir) {
				fmt.Fprintln(os.Stderr, display.Dim.Render("  MCP server running on stdio (uninitialized project — only init_project tool is useful until the agent initializes .archcore/)..."))
			} else {
				fmt.Fprintln(os.Stderr, display.Dim.Render("  MCP server running on stdio..."))
				if err := checkGlobals(baseDir); err != nil {
					return err
				}
				// Warn (to stderr — stdout is the JSON-RPC stream) if the config
				// carries fields this binary does not recognize. checkGlobals has
				// already confirmed settings.json is present and valid here.
				if s, lErr := config.Load(baseDir); lErr == nil {
					warnUnknownConfigFields(os.Stderr, s)
				}
			}

			return mcpserver.RunStdio(cmd.Context(), baseDir, version,
				backgroundUpdateTask(version),
				mcpserver.WithHostWiring(hostWiringExecutor(baseDir)))
		},
	}

	cmd.Flags().StringVar(&projectFlag, "project", "",
		"project root containing .archcore/ (default: current directory; env: ARCHCORE_PROJECT_ROOT)")

	cmd.AddCommand(newMCPInstallCmd())
	return cmd
}

// --- the background update trigger -------------------------------------------
//
// A long-lived `archcore mcp` is the one process that reliably outlives the
// moment a release lands, so it is where an unattended attempt is worth making
// — mcp-background-update.spec. Everything about the update stack stays on this
// side of the option: internal/mcp takes an opaque func and must never link
// internal/update.

// backgroundUpdateDelay keeps the attempt out of the session's opening moments,
// where a download would contend with the host's initialize and tools/list
// round-trips — mcp-background-update.spec §3.
//
// A variable so a test does not spend a minute per case.
var backgroundUpdateDelay = 60 * time.Second

// runUnattendedUpdate is a seam: the real policy resolves a release host, claims
// a cross-process stamp and replaces the running binary, none of which a test of
// the trigger may do.
var runUnattendedUpdate = update.RunUnattended

// backgroundUpdateTask builds the work RunStdio runs beside the session: wait
// out the delay, then hand the decision to the unattended policy. It writes to
// stderr only, and only when a replacement completed.
func backgroundUpdateTask(version string) func(context.Context) {
	// Cleaned here because root.go hands `mcp` the raw version while every other
	// command is constructed with the cleaned one. A build that reports
	// "1.2.3+dirty" reaches NewerSemver unparseable, which refuses the attempt —
	// silently, since a refusal sends no event — for the life of the install.
	// Cleaning also spells from_version the way the typed `archcore update`
	// spells it, and leaves "dev" alone so the policy's development-build
	// refusal still fires — unattended-update.spec §12, §4.
	ver := cleanVersion(version)

	// The delay is read here, on the caller's goroutine, rather than inside the
	// closure. Nothing joins the goroutine below, so it can still be starting
	// when a test's t.Cleanup restores this seam — a real data race the race
	// detector reports. Reading the value while the task is built puts the read
	// and the restore on the same goroutine and settles it.
	delay := backgroundUpdateDelay

	return func(ctx context.Context) {
		select {
		case <-ctx.Done():
			// The session ended inside the delay: no policy call, so no claim
			// stamp is spent on a process that is already going away —
			// mcp-background-update.spec §9.
			return
		case <-time.After(delay):
		}

		// updateDeps is now shared with `archcore update`: the release repo and
		// the binary name are spelled once, so an unattended attempt can never
		// resolve a different artifact than the command a user types.
		u, tel := updateDeps(ver)
		res := runUnattendedUpdate(ctx, update.UnattendedOptions{
			Updater:   u,
			Version:   ver,
			Telemetry: tel,
		})
		if !res.Updated {
			// A refusal, a skip and a failure are all silent. They are already
			// recorded in telemetry, and a line here would put update chatter in
			// the host's log for a session that got nothing —
			// mcp-background-update.spec §8.
			return
		}
		// stderr only: fd 1 carries JSON-RPC frames, and RunStdio's shield is not
		// guaranteed to still stand over this goroutine — §6. The write error is
		// dropped because an unwritable stderr costs the line and nothing else.
		_, _ = fmt.Fprintln(os.Stderr, display.Dim.Render(fmt.Sprintf(
			"  Updated archcore to %s — takes effect on the next launch", res.NewVersion)))
	}
}

// checkGlobals validates the project's declared global sources before the MCP
// server starts. Every declared global is mandatory: a source that is missing,
// not a directory, unreadable, self-overlapping (resolves to the project's own
// .archcore), or a duplicate path aborts startup so the agent never runs against
// a broken mount. An existing source that holds no documents is surfaced as a
// warning but does not block startup. A present-but-invalid settings.json also
// aborts, rather than starting with globals silently dropped from the read path.
func checkGlobals(baseDir string) error {
	inspections, err := docs.InspectGlobals(baseDir)
	if err != nil {
		// A present-but-invalid settings.json must not start silently: the read
		// path degrades to "no globals" without signal, the exact failure the
		// mandatory-globals decision exists to prevent.
		return fmt.Errorf("invalid .archcore/settings.json: %w", err)
	}
	for _, in := range inspections {
		switch {
		case in.State == docs.GlobalEmpty:
			fmt.Fprintln(os.Stderr, display.WarnLine(in.Message()+" — starting anyway"))
		case in.State.Fatal():
			suffix := " — fix .archcore/settings.json before starting the MCP server"
			if in.State == docs.GlobalMissing {
				suffix = " — clone it before starting the MCP server"
			}
			return errors.New(in.Message() + suffix)
		}
	}
	return nil
}

func newMCPInstallCmd() *cobra.Command {
	var (
		agentFlag   string
		projectFlag string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install MCP server config for coding agents",
		// Agent selection is via --agent; reject stray positional args.
		Args: cobra.NoArgs,
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

	cmd.Flags().StringVar(&agentFlag, "agent", "", "install for a single agent (e.g. cursor, gemini-cli); not repeatable — the last value wins")
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
