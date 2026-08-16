package wiring

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"archcore-cli/internal/agents"
)

// Effective-state reporting: say what was written, then whether it can take
// effect, and name the fix when it cannot.

// DescribeEffectiveHooks returns what the user needs to know about hooks just
// written for one agent. An empty result means the wiring works as installed.
//
// Nothing here depends on the other agents, so a caller installing several
// reports the plugin overlap separately (DescribePluginConflict) rather than
// repeating it per agent.
func DescribeEffectiveHooks(baseDir string, id agents.AgentID) []string {
	//exhaustive:ignore // Only hosts with a documented limitation have a note; the rest return nil, which is the default arm.
	switch id {
	case agents.CodexCLI:
		return codexHookNotes()
	case agents.Copilot:
		notes := []string{
			"Copilot: pre-write context injection is unavailable — its preToolUse event carries only a permission decision, so the write guard runs but the code-alignment hint does not.",
		}
		// Copilot reads .claude/settings.json too, so in a repository wired for
		// both hosts one event can start two processes.
		if CarriesArchcoreHooks(baseDir, agents.ClaudeCode) {
			notes = append(notes,
				"Copilot also reads .claude/settings.json, which carries archcore hooks for Claude Code in this project. Copilot therefore runs the hook twice per event: once in its own dialect and once in Claude Code's, which it ignores. Expect duplicate output, not a wrong verdict.")
		}
		return notes
	case agents.GeminiCLI:
		return []string{
			"Gemini CLI: tool events (BeforeTool / AfterTool) are wired from the published reference and not yet confirmed against a running host.",
		}
	}
	return nil
}

// DescribePluginConflict reports an installed Archcore plugin whose own hooks
// may fire alongside these, or "" when none is found. It describes the machine,
// not one agent, so a caller installing several reports it once.
func DescribePluginConflict() string {
	p, found := detectInstalledPlugin()
	if !found {
		return ""
	}
	return fmt.Sprintf(
		"An Archcore plugin is installed (%s). Until it is updated, its own hooks and these may both fire and you will see duplicated context. Updating the plugin resolves it.", p)
}

// DescribeSelfCausedPluginConflict reports the same overlap for a run that
// installed or updated the plugin itself, or "" when none is found —
// plugin-delivery.spec §15.
//
// The sentence above is false right after this surface acted: the plugin is the
// current one, so "until it is updated" points the user at work that is already
// done. What remains is the host session that started before the plugin
// changed, and the action that ends it is a restart, not an update.
//
// It is a sibling rather than a parameter on DescribePluginConflict because the
// two have different callers with different obligations. Requirement 4 of
// plugin-cli-compatibility.rule binds the original wording for `doctor` and
// `hooks install`, which detect a plugin this process did not touch, and those
// callers must not change.
func DescribeSelfCausedPluginConflict() string {
	p, found := detectInstalledPlugin()
	if !found {
		return ""
	}
	return fmt.Sprintf(
		"The Archcore plugin is installed and current (%s). Its own hooks and these may both fire in the session that is already open, so expect duplicated context there. Restart the host session to clear it.", p)
}

var (
	// codexTableRe matches a TOML table header, which scopes the keys below it.
	codexTableRe = regexp.MustCompile(`^\s*\[([^\]]+)\]`)
	// codexFeatureFlagRe matches either spelling of the hooks feature flag. The
	// key was renamed in Codex 0.129.0, and both are in the wild.
	codexFeatureFlagRe = regexp.MustCompile(`^\s*(codex_hooks|hooks)\s*=\s*true`)
	// codexDottedFlagRe matches the same flag written at the top level.
	codexDottedFlagRe = regexp.MustCompile(`^\s*features\s*\.\s*(codex_hooks|hooks)\s*=\s*true`)
)

// codexHookNotes reports the three conditions that can leave a written Codex
// config inert.
func codexHookNotes() []string {
	var notes []string

	if runtime.GOOS == "windows" {
		notes = append(notes,
			"Codex CLI: hooks are written but will not run — Codex does not support hooks on Windows.")
		return notes
	}

	if !codexHooksEnabled() {
		notes = append(notes,
			"Codex CLI: hooks are written but will not run — the hooks feature is experimental and off by default. "+
				"Enable it with `codex --enable hooks`, or add `[features]` with `hooks = true` (`codex_hooks = true` before Codex 0.129.0) to ~/.codex/config.toml.")
	}

	notes = append(notes,
		"Codex CLI: project-local hooks load only when the project's .codex/ layer is trusted. Trust the project in Codex if the hooks stay silent.")
	return notes
}

// codexHooksEnabled reports whether the user's Codex config turns hooks on. An
// unreadable config counts as disabled: the honest answer to "can this run?" is
// no when we cannot tell.
//
// The flag is only meaningful inside [features], so the scan tracks the current
// table. Matching it anywhere would report hooks as enabled for any config that
// happens to carry `hooks = true` under some other table — suppressing the
// warning the user needs. The inline form (`features = { hooks = true }`) stays
// unsupported; this is a best-effort reporter, not a TOML parser.
func codexHooksEnabled() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	data, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		return false
	}

	inFeatures := false
	for line := range strings.Lines(string(data)) {
		if m := codexTableRe.FindStringSubmatch(line); m != nil {
			inFeatures = strings.TrimSpace(m[1]) == "features"
			continue
		}
		if codexDottedFlagRe.MatchString(line) {
			return true
		}
		if inFeatures && codexFeatureFlagRe.MatchString(line) {
			return true
		}
	}
	return false
}

// pluginInstallDirs are the caches the Archcore plugin installs into, per host.
var pluginInstallDirs = []string{
	".claude/plugins",
	".cursor/plugins",
	".codex/plugins",
	".copilot/installed-plugins",
}

const (
	// pluginScanDepth is how far below an install root to look. Hosts nest a
	// checkout as cache/<marketplace>/<plugin>/<version>, so a depth-1 scan sees
	// only "cache" and never fires.
	pluginScanDepth = 3
	// pluginScanBudget bounds the walk. The notice is informational, so a large
	// cache must cost a miss rather than real time on a path that runs during
	// `hooks install` and `doctor`.
	pluginScanBudget = 400
)

// detectInstalledPlugin looks for an installed Archcore plugin. Detection is
// best-effort and informational only: a miss costs a missing notice, never a
// change in what the CLI writes. Gating behavior on it would make the CLI
// depend on the layout of someone else's cache.
func detectInstalledPlugin() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	budget := pluginScanBudget
	for _, dir := range pluginInstallDirs {
		if found, ok := scanForPlugin(filepath.Join(home, filepath.FromSlash(dir)), dir, pluginScanDepth, &budget); ok {
			return found, true
		}
	}
	return "", false
}

// scanForPlugin walks up to depth levels below root, returning the display path
// of the first directory whose name names archcore.
func scanForPlugin(root, display string, depth int, budget *int) (string, bool) {
	if depth == 0 || *budget <= 0 {
		return "", false
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if *budget <= 0 {
			return "", false
		}
		*budget--
		if !e.IsDir() {
			continue
		}
		child := filepath.Join(display, e.Name())
		if strings.Contains(strings.ToLower(e.Name()), "archcore") {
			return child, true
		}
		if found, ok := scanForPlugin(filepath.Join(root, e.Name()), child, depth-1, budget); ok {
			return found, true
		}
	}
	return "", false
}
