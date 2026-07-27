package cmd

import (
	"github.com/spf13/cobra"
)

func newHooksCopilotCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "copilot",
		Short: "Handle GitHub Copilot hook events",
	}
	cmd.AddCommand(
		newSessionStartHookCmd("session-start", "Handle Copilot SessionStart hook event", version, shapeCopilotNative),
	)
	return cmd
}
