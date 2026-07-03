package cmd

import (
	"errors"
	"path/filepath"

	"archcore-cli/internal/config"

	"github.com/spf13/cobra"
)

// geminiHookEvents maps Gemini CLI event names to archcore commands.
var geminiHookEvents = []struct {
	Event   string
	Command string
}{
	{"SessionStart", "archcore hooks gemini-cli session-start"},
}

func newHooksGeminiCLICmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gemini-cli",
		Short: "Handle Gemini CLI hook events",
	}
	cmd.AddCommand(
		newSessionStartHookCmd("session-start", "Handle Gemini CLI SessionStart hook event", version),
	)
	return cmd
}

// runGeminiCLIHooksInstall writes hooks config into .gemini/settings.json.
func runGeminiCLIHooksInstall(baseDir string) error {
	if !config.DirExists(baseDir) {
		return errors.New(".archcore/ not found — run 'archcore init' first")
	}

	events := make([]hookEventInstall, len(geminiHookEvents))
	for i, ev := range geminiHookEvents {
		events[i] = hookEventInstall{
			Event:   ev.Event,
			Command: ev.Command,
			Entry:   hookMatcher{Matcher: "", Hooks: []hookEntry{{Type: "command", Command: ev.Command}}},
		}
	}
	return installHookEvents(hookInstallSpec{
		Label:  "Gemini CLI",
		Path:   filepath.Join(baseDir, ".gemini", "settings.json"),
		Probe:  matcherEntryHasCommand,
		Events: events,
	})
}
