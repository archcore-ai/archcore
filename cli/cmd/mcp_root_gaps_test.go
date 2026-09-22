package cmd

// Coverage-gap closures and behavior pins for resolveProjectRoot, motivated by
// the cwd-independence invariant: hosts do not guarantee the working directory
// of agent-spawned processes (Cursor — forum #99215; Copilot launches plugin
// MCP children in the install root — github/copilot-cli#4234), so the
// resolver's flag/env precedence and its plugin-cache rejection are the safety
// net that makes host cwd behavior irrelevant.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mustCacheDir creates the slash-separated relative path rel under a fresh
// temp directory and returns the absolute result — the fixture every
// plugin-cache guard test needs.
func mustCacheDir(t *testing.T, rel string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), filepath.FromSlash(rel))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q): %v", dir, err)
	}
	return dir
}

// TestResolveProjectRoot_RelativeEnvValue verifies that a relative
// ARCHCORE_PROJECT_ROOT value is resolved against the current working
// directory, mirroring the existing relative-flag test. Cannot run in
// parallel because it calls t.Chdir.
func TestResolveProjectRoot_RelativeEnvValue(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "envchild")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatalf("os.Mkdir(%q): %v", child, err)
	}
	t.Chdir(parent)

	got, err := resolveProjectRoot("", "envchild")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parentCanonical, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	want := filepath.Join(parentCanonical, "envchild")
	if got != want {
		t.Errorf("resolveProjectRoot(\"\", \"envchild\") = %q, want %q", got, want)
	}
}

// TestResolveProjectRoot_EnvNonexistentErrors verifies the env source is
// validated exactly like the flag source: a missing directory is rejected
// with the path named in the error, never silently swapped for cwd.
func TestResolveProjectRoot_EnvNonexistentErrors(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "gone")

	_, err := resolveProjectRoot("", missing)

	if err == nil {
		t.Fatal("expected error for nonexistent env path, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error %q should contain %q", err.Error(), "does not exist")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("error %q should name the env path %q", err.Error(), missing)
	}
}

// TestResolveProjectRoot_FlagWinsOverNonexistentEnv pins that precedence
// short-circuits before validation: a valid --project flag must succeed even
// when ARCHCORE_PROJECT_ROOT points at garbage. This is the recovery path when
// a host exports a stale/broken env value.
func TestResolveProjectRoot_FlagWinsOverNonexistentEnv(t *testing.T) {
	t.Parallel()
	flagDir := t.TempDir()
	badEnv := filepath.Join(t.TempDir(), "does-not-exist")
	wantAbs, err := filepath.Abs(flagDir)
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}

	got, err := resolveProjectRoot(flagDir, badEnv)

	if err != nil {
		t.Fatalf("valid flag must win over invalid env, got error: %v", err)
	}
	if got != wantAbs {
		t.Errorf("resolveProjectRoot(flag, badEnv) = %q, want %q", got, wantAbs)
	}
}

// TestResolveProjectRoot_AcceptsPluginDeveloperRepo pins — true today and
// required to STAY true alongside the plugin-cache rejection — that a plugin
// *developer* repo (root carries .claude-plugin/marketplace.json or
// .cursor-plugin/plugin.json manifests) is a perfectly valid project root.
// The cache guard keys on install-cache path fragments, not on the mere
// presence of plugin manifests.
func TestResolveProjectRoot_AcceptsPluginDeveloperRepo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		manifest string // relative path created under the project root
	}{
		{"claude marketplace manifest", ".claude-plugin/marketplace.json"},
		{"cursor plugin manifest", ".cursor-plugin/plugin.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			manifest := filepath.Join(root, filepath.FromSlash(tt.manifest))
			if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(manifest, []byte(`{"name":"dev-plugin"}`), 0o644); err != nil {
				t.Fatal(err)
			}
			wantAbs, err := filepath.Abs(root)
			if err != nil {
				t.Fatalf("filepath.Abs: %v", err)
			}

			got, err := resolveProjectRoot(root, "")

			if err != nil {
				t.Fatalf("plugin developer repo must be accepted as project root, got error: %v", err)
			}
			if got != wantAbs {
				t.Errorf("resolveProjectRoot(%q, \"\") = %q, want %q", root, got, wantAbs)
			}
		})
	}
}

// Case-insensitive matching: macOS and Windows filesystems are
// case-insensitive, so `.Cursor/Plugins/…` must not bypass the guard.
func TestResolveProjectRoot_PluginCacheGuardIsCaseInsensitive(t *testing.T) {
	t.Parallel()
	for _, p := range []string{
		".Cursor/Plugins/cache/archcore",
		".Copilot/Installed-Plugins/_direct/archcore",
	} {
		t.Run(p, func(t *testing.T) {
			t.Parallel()
			root := mustCacheDir(t, p)

			if _, err := resolveProjectRoot("", root); err == nil {
				t.Errorf("mixed-case plugin-cache path %q must be rejected for implicit sources", root)
			}
		})
	}
}

// TestResolveProjectRoot_RejectsPluginCachePaths_Spec pins the
// plugin-install-cache guard. Cursor has been observed launching
// agent-spawned processes with cwd inside ~/.cursor/plugins/cache/…, and
// Copilot launches a plugin's MCP children inside
// ~/.copilot/installed-plugins/… (github/copilot-cli#4234) — either would
// make the resolver treat the plugin cache as the user's project and write
// the user's documents where no git repository will ever see them. The guard
// rejects any resolved root containing an install-cache fragment as a whole
// path segment:
//
//	/.cursor/plugins/   /.claude/plugins/   /.codex/plugins/
//	/.copilot/installed-plugins/   /plugins/cache/
//
// isPluginCachePath (mcp_root.go) rejects these fragments on the
// slash-normalized absolute path; each subtest produces an error naming the
// offending path. The guard applies to implicit sources only (env, cwd) —
// an explicit --project flag is trusted user intent and bypasses it.
func TestResolveProjectRoot_RejectsPluginCachePaths_Spec(t *testing.T) {
	t.Parallel()
	fragments := []string{
		".cursor/plugins/cache/archcore/abc123",
		".claude/plugins/cache/archcore",
		".codex/plugins/archcore",
		"plugins/cache/archcore",
		".copilot/installed-plugins/archcore",
		".copilot/installed-plugins/_direct/archcore-ai--plugin--plugins-archcore",
	}
	for _, frag := range fragments {
		t.Run(frag, func(t *testing.T) {
			t.Parallel()
			// Arrange: a real directory whose path contains the cache fragment.
			root := mustCacheDir(t, frag)

			// Act: resolve via the implicit source a host can poison (env; the
			// cwd fallback shares the code path). The explicit --project flag
			// states user intent and must BYPASS the guard — it is the
			// recovery path the guard's own error message recommends.
			_, envErr := resolveProjectRoot("", root)
			flagGot, flagErr := resolveProjectRoot(root, "")

			// Assert: env refuses and names the path; flag is trusted.
			if envErr == nil {
				t.Errorf("env source: expected plugin-cache path %q to be rejected", root)
			} else if !strings.Contains(envErr.Error(), root) {
				t.Errorf("env source: error %q should name the rejected path %q", envErr.Error(), root)
			}
			if flagErr != nil {
				t.Errorf("explicit --project must be trusted even inside a cache path: %v", flagErr)
			} else if flagGot != root {
				t.Errorf("flag source: got %q, want %q", flagGot, root)
			}
		})
	}
}

// The cache directory itself — not just paths under it — must be rejected:
// isPluginCachePath appends a trailing separator before matching, and Copilot
// (github/copilot-cli#4234) launches a plugin's MCP child with cwd set to the
// install root, i.e. exactly this directory.
func TestResolveProjectRoot_CopilotCacheRootItself(t *testing.T) {
	t.Parallel()
	root := mustCacheDir(t, ".copilot/installed-plugins")

	if _, err := resolveProjectRoot("", root); err == nil {
		t.Errorf("the copilot install-cache directory itself %q must be rejected for implicit sources", root)
	}
}

// A user project whose path merely CONTAINS a fragment as a substring
// ("…/my.copilot/installed-plugins/app") is NOT a cache and must resolve
// normally. Fragments are delimited by "/" at both ends, so ".copilot/" only
// matches a whole segment — "my.copilot" does not qualify. Anchoring costs no
// real detection: filepath.Abs returns a Cleaned absolute path, so a genuine
// cache hit always has a separator before the fragment. Both sources are
// asserted because the implicit arm is the one the anchoring buys.
func TestResolveProjectRoot_CopilotLookalikeUserPath_NotRejected(t *testing.T) {
	t.Parallel()
	root := mustCacheDir(t, "my.copilot/installed-plugins/app")

	for _, tt := range []struct {
		name      string
		flag, env string
	}{
		{"implicit env source", "", root},
		{"explicit --project", root, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveProjectRoot(tt.flag, tt.env)
			if err != nil {
				t.Fatalf("lookalike path %q must not be mistaken for a plugin cache: %v", root, err)
			}
			if got != root {
				t.Errorf("resolveProjectRoot(%q, %q) = %q, want %q", tt.flag, tt.env, got, root)
			}
		})
	}
}

// The guard error is all a user sees when a host misroutes cwd — it must name
// both recovery paths: --project for direct invocations, and project-level
// wiring (archcore init --agent) for hosts like Copilot that auto-discover a
// plugin-root .mcp.json and spawn the server from the cache, where no flag
// can be edited.
func TestResolveProjectRoot_GuardErrorNamesRecovery(t *testing.T) {
	t.Parallel()
	root := mustCacheDir(t, ".copilot/installed-plugins/_direct/x")

	_, err := resolveProjectRoot("", root)

	if err == nil {
		t.Fatal("expected plugin-cache rejection")
	}
	for _, want := range []string{"--project", "archcore init --agent"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("guard error %q should carry the recovery hint %q", err.Error(), want)
		}
	}
}
