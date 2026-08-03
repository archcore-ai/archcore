package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// pluginCacheFragments are path fragments that identify an AI-host plugin
// install cache. Hosts have been observed spawning agent processes with cwd
// inside the plugin cache instead of the user's workspace: Cursor for stdio
// hooks (forum #99215), and Copilot for a plugin's auto-discovered MCP
// children — launched in the install root with no project path passed
// (github/copilot-cli#4234). See host-cwd-misrouting.adr. Treating such a
// directory as the project root would read the plugin's own bundled files as
// the user's project, or worse, write the user's documents into the cache
// where no git repository will see them.
//
// The guard keys on install-cache path fragments only — a plugin *developer*
// repo, whose root merely contains .claude-plugin/ or .cursor-plugin/
// manifests, is a perfectly valid project.
//
// Each fragment must be lowercase and delimited by "/" at both ends:
// isPluginCachePath lowers the candidate and appends a trailing separator,
// and filepath.Abs always returns a Cleaned absolute path, so a real cache
// hit always carries a separator before the fragment. The leading "/" costs
// no detection and keeps a user project living at, say,
// ".../my.copilot/installed-plugins/app" from being refused.
var pluginCacheFragments = []string{
	"/.cursor/plugins/",
	"/.claude/plugins/",
	"/.codex/plugins/",
	"/.copilot/installed-plugins/",
	"/plugins/cache/",
}

// isPluginCachePath reports whether abs sits inside a plugin install cache.
// Matching is done case-insensitively (macOS/Windows filesystems are
// case-insensitive) on the slash-normalized path with a trailing separator
// appended, so a root that IS the cache directory itself also matches.
// Fragments are separator-delimited on both ends, so they match whole path
// segments and never a suffix of a longer segment name.
// Symlinks are not resolved — this is a heuristic against host cwd
// misrouting, not a security boundary.
func isPluginCachePath(abs string) bool {
	normalized := strings.ToLower(filepath.ToSlash(abs)) + "/"
	for _, frag := range pluginCacheFragments {
		if strings.Contains(normalized, frag) {
			return true
		}
	}
	return false
}

// resolveProjectRoot picks the project root from (in order):
//  1. flagValue if non-empty
//  2. envValue if non-empty
//  3. os.Getwd()
//
// The returned path is absolute. Errors if the source cannot be made absolute,
// does not exist, or is not a directory.
//
// The plugin-cache guard applies only to the implicit sources (env, cwd) —
// those are the ones a host can misroute. An explicit --project flag states
// user intent and is trusted as-is, which also keeps it usable as the
// recovery path the guard's own error message recommends.
//
// envValue is passed in (not read inside) so the caller controls precedence
// visibly and tests do not have to mutate process env.
func resolveProjectRoot(flagValue, envValue string) (string, error) {
	source := flagValue
	explicit := flagValue != ""
	if !explicit {
		if envValue != "" {
			source = envValue
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("failed to determine working directory: %w", err)
			}
			source = cwd
		}
	}

	abs, err := filepath.Abs(source)
	if err != nil {
		return "", fmt.Errorf("failed to resolve project root %q: %w", source, err)
	}

	if !explicit && isPluginCachePath(abs) {
		return "", fmt.Errorf(
			"refusing project root %q: path is inside an AI-host plugin install cache, not a user project (the host likely misrouted the working directory — pass --project to the real project root, or register a project-level server: archcore init --agent <agent> --project <path>)", abs)
	}

	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("project root does not exist: %q", abs)
		}
		return "", fmt.Errorf("failed to stat project root %q: %w", abs, err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("project root is not a directory: %q", abs)
	}

	return abs, nil
}
