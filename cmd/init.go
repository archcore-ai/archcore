package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/api"
	"archcore-cli/internal/config"
	"archcore-cli/internal/display"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var ErrServerUnreachable = errors.New("server unreachable")

type serverUnreachableError struct {
	url string
	err error
}

func (e *serverUnreachableError) Error() string {
	return fmt.Sprintf("cannot reach server at %s: %v", e.url, e.err)
}

func (e *serverUnreachableError) Unwrap() error {
	return e.err
}

func (e *serverUnreachableError) Is(target error) bool {
	return target == ErrServerUnreachable
}

type initResult struct {
	serverReachable bool // only meaningful when ServerURL != ""
}

var isInteractive = defaultIsInteractive

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
			return nil, &serverUnreachableError{url: serverURL, err: err}
		}
		result.serverReachable = true
	}

	return result, nil
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize archcore in the current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			fmt.Println(display.WelcomeBanner())
			fmt.Println()

			cwd, err := os.Getwd()
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
				if errors.Is(err, ErrServerUnreachable) {
					fmt.Println(display.FailLine(err.Error()))
					return nil
				}
				return err
			}

			fmt.Println(display.CheckLine("Created .archcore/ directory"))
			if result.serverReachable {
				fmt.Println(display.CheckLine("Server is reachable"))
			}
			fmt.Println(display.CheckLine("Settings saved to .archcore/settings.json"))

			// Auto-detect agents and install hooks + MCP config for all found.
			// If none detected, ask the user to pick from supported agents.
			detected, err := resolveAgents(cwd)
			if err != nil {
				return err
			}
			if len(detected) == 0 {
				fmt.Println(display.Dim.Render("  No AI agent selected. Run 'archcore hooks install' or 'archcore mcp install' later."))
			}
			for _, agent := range detected {
				if _, err := installHooksForAgent(cwd, agent); err != nil {
					fmt.Println(display.WarnLine(fmt.Sprintf("%s hooks: %v", agent.DisplayName, err)))
				}
				if err := installMCPForAgent(cwd, agent); err != nil {
					fmt.Println(display.WarnLine(fmt.Sprintf("%s MCP: %v", agent.DisplayName, err)))
				}
			}

			fmt.Println()
			fmt.Println(display.Success.Render("  Ready! Run 'archcore status' to verify."))
			return nil
		},
	}
}

func resolveAgents(baseDir string) ([]*agents.Agent, error) {
	detected := agents.Detect(baseDir)
	if len(detected) != 0 {
		return detected, nil
	}
	return promptSelectAgents()
}

// promptSelectAgents asks the user to pick which agents to install hooks + MCP
// config for. Empty selection means "skip". Returns empty in non-interactive
// environments (CI, tests) and on user cancel.
func promptSelectAgents() ([]*agents.Agent, error) {
	if !isInteractive() {
		return nil, nil
	}

	all := agents.All()
	options := make([]huh.Option[agents.AgentID], 0, len(all))
	for _, a := range all {
		options = append(options, huh.NewOption(a.DisplayName, a.ID))
	}

	var picked []agents.AgentID
	err := huh.NewMultiSelect[agents.AgentID]().
		Title("No AI agent auto-detected. Select agents to configure (space to toggle, enter to confirm)").
		Options(options...).
		Value(&picked).
		Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, nil
		}
		return nil, err
	}

	result := make([]*agents.Agent, 0, len(picked))
	for _, id := range picked {
		if a := agents.ByID(id); a != nil {
			result = append(result, a)
		}
	}
	return result, nil
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
