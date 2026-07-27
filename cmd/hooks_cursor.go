package cmd

import (
	"github.com/spf13/cobra"
)

func newHooksCursorCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cursor",
		Short: "Handle Cursor hook events",
	}
	cmd.AddCommand(
		newSessionStartHookCmd("session-start", "Handle Cursor SessionStart hook event", version, shapeClaudeCompat),
	)
	return cmd
}
