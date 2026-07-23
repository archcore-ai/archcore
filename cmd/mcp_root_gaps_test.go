package cmd

// Coverage-gap closures and TDD specs for resolveProjectRoot, motivated by the
// cwd-independence invariant: hosts (notably Cursor) do not guarantee the
// working directory of agent-spawned processes, so the resolver's flag/env
// precedence and its (planned) plugin-cache rejection are the safety net that
// makes host cwd behavior irrelevant.
//
// Runnable now: TestResolveProjectRoot_RelativeEnvValue,
// TestResolveProjectRoot_EnvNonexistentErrors,
// TestResolveProjectRoot_FlagWinsOverNonexistentEnv,
// TestResolveProjectRoot_AcceptsPluginDeveloperRepo.
//
// TDD spec (skipped until the guard lands):
// TestResolveProjectRoot_RejectsPluginCachePaths_Spec.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

// TestResolveProjectRoot_AcceptsPluginDeveloperRepo pins — already true today
// and required to STAY true once the planned plugin-cache rejection lands —
// that a plugin *developer* repo (root carries .claude-plugin/marketplace.json
// or .cursor-plugin/plugin.json manifests) is a perfectly valid project root.
// The planned cache guard must key on install-cache path fragments, not on the
// mere presence of plugin manifests.
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
	root := filepath.Join(t.TempDir(), filepath.FromSlash(".Cursor/Plugins/cache/archcore"))
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := resolveProjectRoot("", root); err == nil {
		t.Errorf("mixed-case plugin-cache path %q must be rejected for implicit sources", root)
	}
}

// TestResolveProjectRoot_RejectsPluginCachePaths_Spec is a TDD spec for the
// planned plugin-install-cache guard. Cursor has been observed launching
// agent-spawned processes with cwd inside ~/.cursor/plugins/cache/…, which
// once made the resolver treat the plugin cache as the user's project and
// leak the plugin's bundled .archcore/. The guard must reject any resolved
// root whose path contains an install-cache fragment:
//
//	.cursor/plugins/   .claude/plugins/   .codex/plugins/   plugins/cache/
//
// Implemented: isPluginCachePath (mcp_root.go) rejects these fragments on the
// slash-normalized absolute path; each subtest produces an error naming the
// offending path. The guard applies to implicit sources only (env, cwd) —
// an explicit --project flag is trusted user intent and bypasses it.
func TestResolveProjectRoot_RejectsPluginCachePaths_Spec(t *testing.T) {
	t.Parallel()
	fragments := []string{
		filepath.FromSlash(".cursor/plugins/cache/archcore/abc123"),
		filepath.FromSlash(".claude/plugins/cache/archcore"),
		filepath.FromSlash(".codex/plugins/archcore"),
		filepath.FromSlash("plugins/cache/archcore"),
	}
	for _, frag := range fragments {
		t.Run(frag, func(t *testing.T) {
			t.Parallel()
			// Arrange: a real directory whose path contains the cache fragment.
			root := filepath.Join(t.TempDir(), frag)
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatal(err)
			}

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
