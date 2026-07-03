package cmd

import (
	"fmt"
	"net/http"
	"runtime"

	"archcore-cli/internal/display"
	"archcore-cli/internal/update"

	"github.com/spf13/cobra"
)

func newUpdateCmd(version string) *cobra.Command {
	u := update.NewUpdater(version, "archcore-ai/cli", "archcore")
	return buildUpdateCmd(version, u)
}

// newUpdateCmdWithClient creates an update command that uses a custom HTTP
// client. This is used for testing to inject a mock server.
func newUpdateCmdWithClient(version string, client *http.Client) *cobra.Command {
	u := &update.Updater{
		CurrentVersion: version,
		GitHubRepo:     "archcore-ai/cli",
		BinaryName:     "archcore",
		HTTPClient:     client,
	}
	return buildUpdateCmd(version, u)
}

func buildUpdateCmd(version string, u *update.Updater) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update archcore to the latest version",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			fmt.Println(display.Banner())
			fmt.Println()
			fmt.Println(display.Dim.Render("  Checking for updates..."))

			latest, err := u.CheckLatest(ctx)
			if err != nil {
				fmt.Println(display.FailLine("Could not check for updates"))
				fmt.Println(display.HintLine(err.Error()))
				// Updating is this command's sole job — scripts must see a
				// non-zero exit on failure. The details are already printed, so
				// signal exit-only (mirrors status/doctor) and let main avoid a
				// second stderr copy.
				return ErrAlreadyReported
			}

			fmt.Println(display.CheckLine(fmt.Sprintf("Current: %s", version)))
			fmt.Println(display.CheckLine(fmt.Sprintf("Latest:  %s", latest)))

			if !update.NeedsUpdate(version, latest) {
				fmt.Println()
				fmt.Println(display.CheckLine(fmt.Sprintf("Already up to date (%s)", version)))
				return nil
			}

			fmt.Println()

			archive := update.ArchiveName("archcore", runtime.GOOS, runtime.GOARCH)
			fmt.Println(display.Dim.Render(fmt.Sprintf("  Downloading %s...", archive)))

			if err := u.Apply(ctx, latest); err != nil {
				fmt.Println(display.FailLine("Update failed"))
				fmt.Println(display.HintLine(err.Error()))
				return ErrAlreadyReported
			}

			fmt.Println(display.CheckLine("Checksum verified"))
			fmt.Println(display.CheckLine(fmt.Sprintf("Updated to %s", latest)))

			return nil
		},
	}
}
