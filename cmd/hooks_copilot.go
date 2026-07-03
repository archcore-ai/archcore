package cmd

import (
	"errors"
	"path/filepath"

	"archcore-cli/internal/config"

	"github.com/spf13/cobra"
)

// copilotHookEntry represents a single hook in Copilot config.
type copilotHookEntry struct {
	Type string `json:"type"`
	Bash string `json:"bash"`
}

// copilotHookEvents maps event names to archcore commands.
var copilotHookEvents = []struct {
	Event   string
	Command string
}{
	{"sessionStart", "archcore hooks copilot session-start"},
}

func newHooksCopilotCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "copilot",
		Short: "Handle GitHub Copilot hook events",
	}
	cmd.AddCommand(
		newSessionStartHookCmd("session-start", "Handle Copilot SessionStart hook event", version),
	)
	return cmd
}

// runCopilotHooksInstall writes hooks config to .github/hooks/archcore.json.
func runCopilotHooksInstall(baseDir string) error {
	if !config.DirExists(baseDir) {
		return errors.New(".archcore/ not found — run 'archcore init' first")
	}

	events := make([]hookEventInstall, len(copilotHookEvents))
	for i, ev := range copilotHookEvents {
		events[i] = hookEventInstall{
			Event:   ev.Event,
			Command: ev.Command,
			Entry:   copilotHookEntry{Type: "command", Bash: ev.Command},
		}
	}
	return installHookEvents(hookInstallSpec{
		Label:         "Copilot",
		Path:          filepath.Join(baseDir, ".github", "hooks", "archcore.json"),
		EnsureVersion: true,
		Probe:         bashEntryHasCommand,
		Events:        events,
	})
}
