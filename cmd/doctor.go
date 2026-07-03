package cmd

import (
	"fmt"
	"os"

	"archcore-cli/internal/api"
	"archcore-cli/internal/config"
	"archcore-cli/internal/display"

	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	var fix bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check your archcore setup and fix issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			fmt.Println(display.Banner())
			fmt.Println()

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
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
	cmd.Flags().BoolVar(&fix, "fix", false, "Automatically fix issues (e.g., remove orphaned relations)")
	return cmd
}
