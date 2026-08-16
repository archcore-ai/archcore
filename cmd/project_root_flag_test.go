package cmd

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Project-root resolution is one decision, taken in one place.
//
// A command that finds its own root with os.Getwd() ignores --project, ignores
// ARCHCORE_PROJECT_ROOT, and — the reason this matters rather than merely being
// untidy — skips the plugin-cache guard in resolveProjectRoot. Hosts have been
// observed spawning agent processes with cwd inside their own plugin install
// cache (host-cwd-misrouting.adr), so such a command reads a plugin's bundled
// files as if they were the user's project.
//
// `instructions remove` did exactly that while `instructions install` did not,
// which made one user-facing command answer two different questions about which
// repository it was operating on.

// rootlessCommands are the commands that legitimately resolve no project root,
// each with the reason. Anything else in the tree must offer --project.
//
// The hook subtree is excluded by shape rather than by name: every leaf under
// `hooks <host>` takes its root from the payload's cwd key, because there the
// host names the project and the process's own working directory is the host's.
// Listing the eighteen of them would turn a rule into an inventory.
var rootlessCommands = map[string]string{
	// Self-update replaces the binary; it never reads the store.
	"archcore update": "does not read .archcore/",
	// Gated off. Its RunE returns an error before resolving anything; the flag
	// arrives with the resolution when the gate lifts.
	"archcore sync":         "gated off, resolves nothing yet",
	"archcore":              "root",
	"archcore hooks":        "parent of install and the hook leaves",
	"archcore instructions": "parent of install and remove",
	// Only `plugin install --scope project` writes a repository file. The other
	// three verbs act on host stores alone, so a --project they accepted and
	// never read would answer a user who passed it with silence. The `plugin`
	// group keeps the flag because cobra parses a parent's flag value before it
	// resolves the subcommand, which is what makes
	// `archcore plugin --project X install` work.
	"archcore plugin update": "acts on host plugin stores, reads no project file",
	"archcore plugin remove": "acts on host plugin stores, reads no project file",
	"archcore plugin status": "reports host plugin state, reads no project file",
}

// isHookEventCommand reports whether path names a per-host hook leaf or the
// host group above it, both of which read their root from the payload.
func isHookEventCommand(path string) bool {
	return strings.HasPrefix(path, "archcore hooks ") && path != "archcore hooks install"
}

// TestCommands_OfferProjectFlag walks the real command tree, so a command added
// without --project fails here rather than in a user's plugin cache.
func TestCommands_OfferProjectFlag(t *testing.T) {
	t.Parallel()

	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		path := c.CommandPath()
		hasFlag := c.Flags().Lookup("project") != nil
		reason, listed := rootlessCommands[path]

		switch {
		case isHookEventCommand(path):
			if hasFlag {
				t.Errorf("%q is a hook leaf and declares --project; its root comes from the payload", path)
			}
		case listed:
			if hasFlag {
				t.Errorf("%q is listed as rootless (%s) but declares --project; "+
					"remove it from rootlessCommands or drop the flag", path, reason)
			}
		case !hasFlag:
			t.Errorf("%q declares no --project flag. Resolve its root through "+
				"resolveProjectRoot, or add it to rootlessCommands with the reason", path)
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(NewRootCmd("0.0.0-test"))
}

// TestProjectFlag_IsHonored drives the commands that read the store against a
// project that is not the working directory. A command still calling os.Getwd()
// reads the wrong tree, and for `instructions remove` it edits it.
//
// Each case asserts on something only the named project could produce. An
// earlier version of this test ran the commands and discarded their output,
// which made two of its three cases prove nothing: reverting `status` and
// `config` to os.Getwd() left the whole suite green.
//
// t.Chdir marks the test as non-parallel for us; it also restores the directory.
func TestProjectFlag_IsHonored(t *testing.T) {
	elsewhere := t.TempDir()
	project := setupArchcoreDir(t)
	seedConfig(t, project, filepath.Join(".archcore", "settings.json"), `{"sync":"none"}`)
	writeArchcoreDoc(t, project, "knowledge/a.adr.md",
		"---\ntitle: \"A\"\nstatus: draft\n---\n\nBody long enough to read as a document rather than a placeholder.\n")

	// Identical instruction files in both trees, each with content outside the
	// managed block so a stripped file survives and can be told apart.
	const hinted = "# Notes\n\n<!-- archcore:start -->\nhint\n<!-- archcore:end -->\n"
	for _, dir := range []string{elsewhere, project} {
		if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(hinted), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// The working directory holds no .archcore/, so every command reading it
	// instead of the named project says so — which is the assertion.
	t.Chdir(elsewhere)

	tests := []struct {
		name string
		args []string
		// want appears only when the named project was read; reject appears only
		// when the working directory was.
		want   string
		reject string
	}{
		{
			name:   "config get",
			args:   []string{"config", "get", "sync", "--project", project},
			want:   "none",
			reject: "Settings not found",
		},
		{
			name:   "status",
			args:   []string{"status", "--project", project},
			want:   ".archcore/ exists",
			reject: ".archcore/ directory not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := NewRootCmd("0.0.0-test")
			root.SetArgs(tt.args)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)

			out, errOut := captureOutput(t, func() { _ = root.Execute() })
			combined := out + errOut

			if !strings.Contains(combined, tt.want) {
				t.Errorf("output does not carry %q, so the named project was not read:\n%s", tt.want, combined)
			}
			if strings.Contains(combined, tt.reject) {
				t.Errorf("output carries %q, so the working directory was read instead:\n%s", tt.reject, combined)
			}
		})
	}

	// instructions remove is the one that writes, so its proof is on disk: the
	// named project loses its block and the working directory keeps its own.
	t.Run("instructions remove", func(t *testing.T) {
		root := NewRootCmd("0.0.0-test")
		root.SetArgs([]string{"instructions", "remove", "--project", project})
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)

		captureOutput(t, func() { _ = root.Execute() })

		if got := readIfPresent(t, filepath.Join(project, "CLAUDE.md")); strings.Contains(got, "archcore:start") {
			t.Errorf("the named project still carries the managed block: %q", got)
		}
		if got := readIfPresent(t, filepath.Join(elsewhere, "CLAUDE.md")); !strings.Contains(got, "archcore:start") {
			t.Errorf("the working directory lost its block, so --project was ignored: %q", got)
		}
	})
}

// readIfPresent returns the file's content, or "" when it does not exist —
// removing the whole file is one valid outcome of stripping the block.
func readIfPresent(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
