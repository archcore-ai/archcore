package cmd

import (
	"github.com/spf13/cobra"
)

func newHooksGeminiCLICmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gemini-cli",
		Short: "Handle Gemini CLI hook events",
	}
	cmd.AddCommand(
		newSessionStartHookCmd("session-start", "Handle Gemini CLI SessionStart hook event", version, shapeClaudeCompat),
	)
	return cmd
}
