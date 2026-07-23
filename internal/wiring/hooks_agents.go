package wiring

import (
	"errors"
	"path/filepath"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/config"
)

// hookEntry represents a single hook command configuration.
type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// hookMatcher represents a matcher with its hooks array.
type hookMatcher struct {
	Matcher string      `json:"matcher"`
	Hooks   []hookEntry `json:"hooks"`
}

// archcoreHooks defines the hooks we install for Claude Code, in deterministic order.
var archcoreHooks = []struct {
	Event   string
	Matcher hookMatcher
}{
	{"SessionStart", hookMatcher{Matcher: "", Hooks: []hookEntry{{Type: "command", Command: "archcore hooks claude-code session-start"}}}},
}

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

// geminiHookEvents maps Gemini CLI event names to archcore commands.
var geminiHookEvents = []struct {
	Event   string
	Command string
}{
	{"SessionStart", "archcore hooks gemini-cli session-start"},
}

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

// hooksInstallers maps agent IDs to their hooks install functions.
// This is the single source of truth for which agents support hooks.
var hooksInstallers = map[agents.AgentID]func(string) error{
	agents.ClaudeCode: InstallClaudeCodeHooks,
	agents.Cursor:     InstallCursorHooks,
	agents.GeminiCLI:  InstallGeminiCLIHooks,
	agents.Copilot:    InstallCopilotHooks,
}

// InstallHooksForAgent installs hooks for a single agent that supports them.
// Returns (false, nil) if the agent doesn't support hooks.
func InstallHooksForAgent(baseDir string, agent *agents.Agent) (bool, error) {
	installer, ok := hooksInstallers[agent.ID]
	if !ok {
		return false, nil
	}
	return true, installer(baseDir)
}

// InstallClaudeCodeHooks installs Claude Code hooks into .claude/settings.json.
func InstallClaudeCodeHooks(baseDir string) error {
	if !config.DirExists(baseDir) {
		return errors.New(".archcore/ not found — run 'archcore init' first")
	}

	events := make([]hookEventInstall, len(archcoreHooks))
	for i, h := range archcoreHooks {
		events[i] = hookEventInstall{Event: h.Event, Command: h.Matcher.Hooks[0].Command, Entry: h.Matcher}
	}
	return installHookEvents(hookInstallSpec{
		Path:   filepath.Join(baseDir, ".claude", "settings.json"),
		Probe:  matcherEntryHasCommand,
		Events: events,
	})
}

// InstallCursorHooks writes hooks config to .cursor/hooks.json.
func InstallCursorHooks(baseDir string) error {
	if !config.DirExists(baseDir) {
		return errors.New(".archcore/ not found — run 'archcore init' first")
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

// InstallGeminiCLIHooks writes hooks config into .gemini/settings.json.
func InstallGeminiCLIHooks(baseDir string) error {
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

// InstallCopilotHooks writes hooks config to .github/hooks/archcore.json.
func InstallCopilotHooks(baseDir string) error {
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
