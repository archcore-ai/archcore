// Package projectroot holds the checks a project root must pass before the CLI
// or the MCP server serves it. Both the start-time resolver in the cmd layer and
// the session-following root provider consult them, so a root a host reports
// over the protocol is trusted no further than a working directory is.
package projectroot

import (
	"path/filepath"
	"strings"
)

// pluginCacheFragments are path fragments that identify an AI-host plugin
// install cache, which hosts have been observed spawning agent processes into
// instead of the user's workspace (host-cwd-misrouting.adr records the incidents
// and why detection keys on the cache path rather than on a plugin manifest).
//
// A fragment added later MUST keep all three properties, because IsPluginCachePath
// depends on every one of them: lowercase, a leading "/", and a trailing "/".
var pluginCacheFragments = []string{
	"/.cursor/plugins/",
	"/.claude/plugins/",
	"/.codex/plugins/",
	"/.copilot/installed-plugins/",
	"/plugins/cache/",
}

// IsPluginCachePath reports whether abs sits inside a plugin install cache.
// Matching is done case-insensitively (macOS/Windows filesystems are
// case-insensitive) on the slash-normalized path with a trailing separator
// appended, so a root that IS the cache directory itself also matches.
// Fragments are separator-delimited on both ends, so they match whole path
// segments and never a suffix of a longer segment name.
// Symlinks are not resolved — this is a heuristic against host cwd
// misrouting, not a security boundary.
func IsPluginCachePath(abs string) bool {
	normalized := strings.ToLower(filepath.ToSlash(abs)) + "/"
	for _, frag := range pluginCacheFragments {
		if strings.Contains(normalized, frag) {
			return true
		}
	}
	return false
}
