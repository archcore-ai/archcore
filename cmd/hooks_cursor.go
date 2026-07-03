package cmd

import (
	"fmt"
	"path/filepath"

	"archcore-cli/internal/config"

	"github.com/spf13/cobra"
)

// cursorHookEntry represents a single hook in Cursor config.
type cursorHookEntry struct {
	Command string `json:"command"`
	Type    string `json:"type"`
}

// cursorHookEvents maps event names to archcore commands.
var cursorHookEvents = []struct {
	Event   string
	Command string
}{
	{"sessionStart", "archcore hooks cursor session-start"},
}

func newHooksCursorCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cursor",
		Short: "Handle Cursor hook events",
	}
	cmd.AddCommand(
		newSessionStartHookCmd("session-start", "Handle Cursor SessionStart hook event", version),
	)
	return cmd
}

// runCursorHooksInstall writes hooks config to .cursor/hooks.json.
func runCursorHooksInstall(baseDir string) error {
	if !config.DirExists(baseDir) {
		return fmt.Errorf(".archcore/ not found — run 'archcore init' first")
	}

	events := make([]hookEventInstall, len(cursorHookEvents))
	for i, ev := range cursorHookEvents {
		events[i] = hookEventInstall{
			Event:   ev.Event,
			Command: ev.Command,
			Entry:   cursorHookEntry{Command: ev.Command, Type: "command"},
		}
	}
	return installHookEvents(hookInstallSpec{
		Label:         "Cursor",
		Path:          filepath.Join(baseDir, ".cursor", "hooks.json"),
		EnsureVersion: true,
		Probe:         commandEntryHasCommand,
		Events:        events,
	})
}
