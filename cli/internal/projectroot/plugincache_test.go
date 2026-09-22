package projectroot

import (
	"strings"
	"testing"
)

func TestIsPluginCachePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		abs  string
		want bool
	}{
		{name: "cursor plugin cache", abs: "/Users/me/.cursor/plugins/archcore", want: true},
		{name: "claude plugin cache", abs: "/Users/me/.claude/plugins/archcore/skills", want: true},
		{name: "codex plugin cache", abs: "/Users/me/.codex/plugins/x", want: true},
		{name: "copilot install root", abs: "/Users/me/.copilot/installed-plugins/archcore", want: true},
		{name: "generic plugin cache", abs: "/opt/tool/plugins/cache/archcore", want: true},

		{name: "the cache directory itself", abs: "/Users/me/.claude/plugins", want: true},
		{name: "case variant on a case-insensitive filesystem", abs: "/Users/me/.Claude/Plugins/Archcore", want: true},
		// Backslash paths are covered by filepath.ToSlash, which is a no-op off
		// Windows — asserting them here would test the host OS, not this package.

		{name: "an ordinary project", abs: "/Users/me/src/archcore-cli"},
		{name: "a plugin developer repo", abs: "/Users/me/src/archcore-plugin"},
		// The leading separator is what keeps these out: the fragment must match
		// whole segments, not a suffix of a longer segment name.
		{name: "a segment ending in the host name", abs: "/Users/me/my.copilot/installed-plugins/app"},
		{name: "a segment ending in cursor", abs: "/Users/me/notcursor/plugins/app"},
		{name: "plugins without the cache segment", abs: "/Users/me/src/plugins/archcore"},
		{name: "empty path", abs: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsPluginCachePath(tt.abs); got != tt.want {
				t.Errorf("IsPluginCachePath(%q) = %v, want %v", tt.abs, got, tt.want)
			}
		})
	}
}

// TestPluginCacheFragments_Shape pins the two properties the fragment list's
// doc comment states and nothing else enforces: lowercase, so lowering the
// candidate can match; and "/" at both ends, so a fragment matches whole path
// segments rather than a suffix of a longer segment name.
func TestPluginCacheFragments_Shape(t *testing.T) {
	t.Parallel()
	for _, frag := range pluginCacheFragments {
		if frag != strings.ToLower(frag) {
			t.Errorf("fragment %q is not lowercase; IsPluginCachePath lowers the candidate and would never match it", frag)
		}
		if !strings.HasPrefix(frag, "/") || !strings.HasSuffix(frag, "/") {
			t.Errorf("fragment %q is not delimited by %q at both ends; it would match a suffix of a longer segment", frag, "/")
		}
	}
}
