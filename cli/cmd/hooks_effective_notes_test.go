package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/agents"
)

// seedInstalledPlugin puts an Archcore plugin checkout in an isolated HOME,
// laid out the way a host actually nests one.
func seedInstalledPlugin(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := filepath.Join(home, ".claude", "plugins", "cache", "archcore-plugins", "archcore", "0.6.2")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

// TestInstallAgents_PluginNoticePrintedOnce: the plugin overlap describes the
// machine, not one agent. Reporting it inside the per-agent notes repeated the
// same paragraph once per detected agent and re-scanned the plugin caches each
// time.
func TestInstallAgents_PluginNoticePrintedOnce(t *testing.T) {
	seedInstalledPlugin(t)
	base := setupArchcoreDir(t)

	list := []*agents.Agent{
		agents.ByID(agents.ClaudeCode),
		agents.ByID(agents.Cursor),
		agents.ByID(agents.GeminiCLI),
	}
	out := captureStdout(t, func() { installAgents(base, list, true) })

	if n := strings.Count(out, "An Archcore plugin is installed"); n != 1 {
		t.Errorf("plugin notice printed %d times, want 1:\n%s", n, out)
	}
}

// TestInstallAgents_NoPluginNoPluginNotice is the negative control: without it
// the test above passes when the notice is removed entirely.
func TestInstallAgents_NoPluginNoPluginNotice(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	base := setupArchcoreDir(t)

	out := captureStdout(t, func() {
		installAgents(base, []*agents.Agent{agents.ByID(agents.ClaudeCode)}, true)
	})

	if strings.Contains(out, "An Archcore plugin is installed") {
		t.Errorf("plugin notice printed with no plugin installed:\n%s", out)
	}
}

// TestReportEffectiveHooks_SaysSomethingEitherWay: doctor must state the
// effective state, not stay silent. A written config the host ignores is
// exactly what the command exists to surface.
func TestReportEffectiveHooks_SaysSomethingEitherWay(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	base := setupArchcoreDir(t)
	// doctor reports on detected agents, so the host's marker has to be there.
	seedConfig(t, base, ".github/copilot-instructions.md", "# instructions\n")
	if err := runHooksInstallForAgent(base, agents.Copilot); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() { reportEffectiveHooks(base) })

	if strings.TrimSpace(out) == "" {
		t.Error("doctor said nothing about whether the wired hosts can act on their hooks")
	}
	if !strings.Contains(out, "Copilot") {
		t.Errorf("Copilot's known limitation was not reported:\n%s", out)
	}
}

// TestReportEffectiveHooks_OnlyWiredHosts: detecting a host is not the same as
// having wired it.
//
// The report iterated agents.Detect, which answers "is this host used here?"
// from the presence of a .claude/ or .codex/ directory. So a project that never
// ran an install still got Codex's feature-flag and trust warnings — about
// wiring that does not exist — and a project with nothing wired at all still got
// the green line claiming its wired hosts were healthy.
func TestReportEffectiveHooks_OnlyWiredHosts(t *testing.T) {
	const reassurance = "Wired hosts can act on their hook configs"

	t.Run("nothing detected and nothing wired", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		base := setupArchcoreDir(t)

		out := captureStdout(t, func() { reportEffectiveHooks(base) })

		if strings.Contains(out, reassurance) {
			t.Errorf("claimed the wired hosts are healthy with no host wired:\n%s", out)
		}
	})

	t.Run("hosts detected but not wired", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		base := setupArchcoreDir(t)
		// Bare host directories: detected, but archcore never wrote hooks here.
		seedConfig(t, base, ".codex/config.toml", "# codex\n")
		seedConfig(t, base, ".claude/settings.json", `{"permissions":{"allow":[]}}`)

		out := captureStdout(t, func() { reportEffectiveHooks(base) })

		if strings.TrimSpace(out) != "" {
			t.Errorf("reported on hooks that were never installed:\n%s", out)
		}
	})

	t.Run("a wired host with no caveats", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		base := setupArchcoreDir(t)
		if err := runHooksInstallForAgent(base, agents.ClaudeCode); err != nil {
			t.Fatal(err)
		}

		out := captureStdout(t, func() { reportEffectiveHooks(base) })

		if !strings.Contains(out, reassurance) {
			t.Errorf("a wired, healthy host produced no report:\n%s", out)
		}
	})
}
