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
// install cache. Hosts (notably Cursor — forum #99215) have been observed
// spawning agent processes with cwd inside the plugin cache instead of the
// user's workspace; treating such a directory as the project root would read
// the plugin's own bundled files as if they were the user's project
// (cursor-mcp-architecture ADR). The guard keys on install-cache path
// fragments only — a plugin *developer* repo, whose root merely contains
// .claude-plugin/ or .cursor-plugin/ manifests, is a perfectly valid project.
var pluginCacheFragments = []string{
	".cursor/plugins/",
	".claude/plugins/",
	".codex/plugins/",
	"plugins/cache/",
}

// isPluginCachePath reports whether abs sits inside a plugin install cache.
// Matching is done case-insensitively (macOS/Windows filesystems are
// case-insensitive) on the slash-normalized path with a trailing separator
// appended, so a root that IS the cache directory itself also matches.
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
			"refusing project root %q: path is inside an AI-host plugin install cache, not a user project (the host likely misrouted the working directory — pass --project to the real project root)", abs)
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
