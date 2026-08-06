package wiring

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/config"
)

// Host hook wiring.
//
// Every host runs the same three archcore events and spells all three
// differently — event names, matcher syntax, entry shape, even the timeout
// unit. There is no shared map to factor out; the table below IS the knowledge,
// and each row is the host's documented contract.
//
// One archcore entry per (host, event). The command dispatches by tool name
// inside the process, so three checks after a document write cost one process
// start instead of three, and the ownership marker stays unambiguous: one entry
// per event means "ours" is a question with one answer.

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

// cursorHookEntry represents a single hook in Cursor config.
type cursorHookEntry struct {
	Command string `json:"command"`
	Type    string `json:"type"`
	Matcher string `json:"matcher,omitempty"`
}

// copilotHookEntry represents a single hook in Copilot config.
type copilotHookEntry struct {
	Type       string `json:"type"`
	Bash       string `json:"bash"`
	Matcher    string `json:"matcher,omitempty"`
	TimeoutSec int    `json:"timeoutSec,omitempty"`
}

// geminiHookEntry mirrors hookMatcher but carries a millisecond timeout —
// Gemini is the one host that does not measure hook timeouts in seconds.
type geminiHookEntry struct {
	Matcher string            `json:"matcher"`
	Hooks   []geminiInnerHook `json:"hooks"`
}

type geminiInnerHook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// mcpDocumentTools matches the archcore document tools under every spelling one
// server arrives as. A host filters on this before starting the process; the
// process folds the name again, so a spelling missed here costs a skipped check,
// never a wrong one.
const mcpDocumentTools = `mcp__archcore__(create_document|update_document|remove_document|add_relation|remove_relation)|` +
	`mcp__plugin_archcore_archcore__(create_document|update_document|remove_document|add_relation|remove_relation)|` +
	`mcp_archcore_(create_document|update_document|remove_document|add_relation|remove_relation)|` +
	`archcore-(create_document|update_document|remove_document|add_relation|remove_relation)`

// Per-host budgets. Pre-write blocks the user, so it is short; post-write only
// reports, so it may scan.
const (
	preToolTimeoutSec  = 1
	postToolTimeoutSec = 3
)

// hooksInstallers maps agent IDs to their hooks install functions.
// This is the single source of truth for which agents support hooks.
var hooksInstallers = map[agents.AgentID]func(string) error{
	agents.ClaudeCode: InstallClaudeCodeHooks,
	agents.Cursor:     InstallCursorHooks,
	agents.GeminiCLI:  InstallGeminiCLIHooks,
	agents.Copilot:    InstallCopilotHooks,
	agents.CodexCLI:   InstallCodexCLIHooks,
}

// hookConfigPaths maps an agent to the file its hooks are written into. It sits
// beside hooksInstallers because the two have to agree: the installer writes the
// file, and the effective-state report reads it back to answer "is anything of
// ours actually wired here?".
var hookConfigPaths = map[agents.AgentID]func(string) string{
	agents.ClaudeCode: func(b string) string { return filepath.Join(b, ".claude", "settings.json") },
	agents.Cursor:     func(b string) string { return filepath.Join(b, ".cursor", "hooks.json") },
	agents.GeminiCLI:  func(b string) string { return filepath.Join(b, ".gemini", "settings.json") },
	agents.Copilot:    func(b string) string { return filepath.Join(b, ".github", "hooks", "archcore.json") },
	agents.CodexCLI:   func(b string) string { return filepath.Join(b, ".codex", "hooks.json") },
}

// HookConfigPath returns the file archcore writes hooks into for one agent, and
// whether that agent supports hooks at all.
func HookConfigPath(baseDir string, id agents.AgentID) (string, bool) {
	fn, ok := hookConfigPaths[id]
	if !ok {
		return "", false
	}
	return fn(baseDir), true
}

// CarriesArchcoreHooks reports whether the agent's hook config actually holds an
// entry archcore wrote (recognized by the command marker).
//
// Detecting an agent is not the same as having wired it: a repository with a
// .claude/ directory and no archcore hooks is a common state, and reporting on
// its "effective hook state" describes wiring that was never written. An
// unreadable or absent config answers no, which matches the fail-closed reading
// requirement 7 of report-effective-hook-state.rule asks for.
func CarriesArchcoreHooks(baseDir string, id agents.AgentID) bool {
	path, ok := HookConfigPath(baseDir, id)
	if !ok {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), archcoreHookMarker)
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

func requireArchcoreDir(baseDir string) error {
	if !config.DirExists(baseDir) {
		return errors.New(".archcore/ not found — run 'archcore init' first")
	}
	return nil
}

// InstallClaudeCodeHooks installs Claude Code hooks into .claude/settings.json.
func InstallClaudeCodeHooks(baseDir string) error {
	if err := requireArchcoreDir(baseDir); err != nil {
		return err
	}
	return installHookEvents(hookInstallSpec{
		Path:   hookConfigPaths[agents.ClaudeCode](baseDir),
		Probe:  matcherEntryHasCommand,
		Events: claudeShapedEvents("claude-code", `Write|Edit`),
	})
}

// InstallCodexCLIHooks writes hooks config to .codex/hooks.json.
//
// Codex reads either hooks.json or an inline [hooks] table in config.toml. The
// separate file is chosen so this never has to rewrite the TOML the MCP wiring
// already owns — one writer per file, no merge to get wrong.
func InstallCodexCLIHooks(baseDir string) error {
	if err := requireArchcoreDir(baseDir); err != nil {
		return err
	}
	return installHookEvents(hookInstallSpec{
		Label: "Codex CLI",
		Path:  hookConfigPaths[agents.CodexCLI](baseDir),
		Probe: matcherEntryHasCommand,
		// Codex edits files through apply_patch as well as Write and Edit.
		Events: claudeShapedEvents("codex-cli", `Write|Edit|apply_patch`),
	})
}

// claudeShapedEvents builds the three events for a host that speaks Claude
// Code's schema: PascalCase event names and matcher-wrapped entries.
func claudeShapedEvents(host, writeMatcher string) []hookEventInstall {
	cmd := func(event string) string { return "archcore hooks " + host + " " + event }
	return []hookEventInstall{
		{
			Event:   "SessionStart",
			Command: cmd("session-start"),
			Entry:   hookMatcher{Hooks: []hookEntry{{Type: "command", Command: cmd("session-start")}}},
		},
		{
			Event:   "PreToolUse",
			Command: cmd("pre-tool-use"),
			Matcher: writeMatcher,
			Entry:   hookMatcher{Matcher: writeMatcher, Hooks: []hookEntry{{Type: "command", Command: cmd("pre-tool-use")}}},
		},
		{
			Event:   "PostToolUse",
			Command: cmd("post-tool-use"),
			Matcher: mcpDocumentTools,
			Entry:   hookMatcher{Matcher: mcpDocumentTools, Hooks: []hookEntry{{Type: "command", Command: cmd("post-tool-use")}}},
		},
	}
}

// InstallCursorHooks writes hooks config to .cursor/hooks.json.
func InstallCursorHooks(baseDir string) error {
	if err := requireArchcoreDir(baseDir); err != nil {
		return err
	}
	cmd := func(event string) string { return "archcore hooks cursor " + event }
	return installHookEvents(hookInstallSpec{
		Label:         "Cursor",
		Path:          hookConfigPaths[agents.Cursor](baseDir),
		EnsureVersion: true,
		Probe:         commandEntryHasCommand,
		Events: []hookEventInstall{
			{
				Event:   "sessionStart",
				Command: cmd("session-start"),
				Entry:   cursorHookEntry{Command: cmd("session-start"), Type: "command"},
			},
			{
				// Cursor has no Edit tool; a file change arrives as Write.
				Event:   "preToolUse",
				Command: cmd("pre-tool-use"),
				Matcher: "Write",
				Entry:   cursorHookEntry{Command: cmd("pre-tool-use"), Type: "command", Matcher: "Write"},
			},
			{
				// MCP results arrive on their own event here, and it takes no
				// matcher — the process filters by tool name instead.
				Event:   "afterMCPExecution",
				Command: cmd("post-tool-use"),
				Entry:   cursorHookEntry{Command: cmd("post-tool-use"), Type: "command"},
			},
		},
	})
}

// InstallGeminiCLIHooks writes hooks config into .gemini/settings.json.
//
// [assumption] Gemini's tool events are wired from its published reference and
// have not been probed against a live host. Its event names (BeforeTool /
// AfterTool), its tool names (write_file, not Write), and its millisecond
// timeouts all differ from every other host, so this row is the most likely one
// to be wrong.
func InstallGeminiCLIHooks(baseDir string) error {
	if err := requireArchcoreDir(baseDir); err != nil {
		return err
	}
	cmd := func(event string) string { return "archcore hooks gemini-cli " + event }
	inner := func(command string, timeoutSec int) []geminiInnerHook {
		return []geminiInnerHook{{Type: "command", Command: command, Timeout: timeoutSec * 1000}}
	}
	return installHookEvents(hookInstallSpec{
		Label: "Gemini CLI",
		Path:  hookConfigPaths[agents.GeminiCLI](baseDir),
		Probe: matcherEntryHasCommand,
		Events: []hookEventInstall{
			{
				Event:   "SessionStart",
				Command: cmd("session-start"),
				Entry:   geminiHookEntry{Hooks: inner(cmd("session-start"), postToolTimeoutSec)},
			},
			{
				Event:   "BeforeTool",
				Command: cmd("pre-tool-use"),
				Matcher: "write_file",
				Entry:   geminiHookEntry{Matcher: "write_file", Hooks: inner(cmd("pre-tool-use"), preToolTimeoutSec)},
			},
			{
				Event:   "AfterTool",
				Command: cmd("post-tool-use"),
				Matcher: mcpDocumentTools,
				Entry:   geminiHookEntry{Matcher: mcpDocumentTools, Hooks: inner(cmd("post-tool-use"), postToolTimeoutSec)},
			},
		},
	})
}

// InstallCopilotHooks writes hooks config to .github/hooks/archcore.json.
func InstallCopilotHooks(baseDir string) error {
	if err := requireArchcoreDir(baseDir); err != nil {
		return err
	}
	cmd := func(event string) string { return "archcore hooks copilot " + event }
	return installHookEvents(hookInstallSpec{
		Label:         "Copilot",
		Path:          hookConfigPaths[agents.Copilot](baseDir),
		EnsureVersion: true,
		Probe:         bashEntryHasCommand,
		Events: []hookEventInstall{
			{
				Event:   "sessionStart",
				Command: cmd("session-start"),
				Entry:   copilotHookEntry{Type: "command", Bash: cmd("session-start"), TimeoutSec: postToolTimeoutSec},
			},
			{
				// Copilot's editing tools carry their own names, and its
				// preToolUse carries only a permission decision — the write
				// guard is all this event can usefully do here.
				Event:   "preToolUse",
				Command: cmd("pre-tool-use"),
				Matcher: `create|edit|str_replace_editor|apply_patch`,
				Entry: copilotHookEntry{
					Type: "command", Bash: cmd("pre-tool-use"),
					Matcher: `create|edit|str_replace_editor|apply_patch`, TimeoutSec: preToolTimeoutSec,
				},
			},
			{
				Event:   "postToolUse",
				Command: cmd("post-tool-use"),
				Matcher: mcpDocumentTools,
				Entry: copilotHookEntry{
					Type: "command", Bash: cmd("post-tool-use"),
					Matcher: mcpDocumentTools, TimeoutSec: postToolTimeoutSec,
				},
			},
		},
	})
}
