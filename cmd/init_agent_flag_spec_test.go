package cmd

// TDD specs for the planned non-interactive init mode:
//
//	archcore init --agent <id> [--project <root>]
//
// Motivation (cwd-independence): the plugin's /archcore:init skill will
// delegate host-config installation to this command, and Cursor does not
// guarantee the cwd of agent-spawned shell commands. The contract therefore
// is: with --agent given, init never opens a TTY prompt, writes every
// artifact under the resolved --project root (flag > ARCHCORE_PROJECT_ROOT >
// cwd, same resolver as `archcore mcp`), is idempotent on re-run, and fails
// fast on an unknown agent id.
//
// Implemented: newInitCmd registers --agent (repeatable) and --project;
// runInitForAgents validates ids before any write and never prompts. No test
// here may be satisfied by prompting: they all run without a TTY.

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// runInitCmdForSpec executes the init command with args, capturing cobra's
// error. Output assertions go through the returned buffer once the
// implementation routes output via cmd.OutOrStdout().
func runInitCmdForSpec(t *testing.T, args ...string) (*bytes.Buffer, error) {
	t.Helper()
	cmd := newInitCmd("")
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs(args)
	return out, cmd.Execute()
}

// Spec 1: all artifacts land under --project regardless of process cwd.
func TestInitCmd_AgentAndProjectFlags_WritesUnderProjectRoot_Spec(t *testing.T) {

	// Arrange: run from an unrelated cwd — simulates Cursor spawning the
	// command inside the plugin cache. Cannot be parallel (t.Chdir).
	project := t.TempDir()
	elsewhere := t.TempDir()
	t.Chdir(elsewhere)

	_, err := runInitCmdForSpec(t, "--agent", "cursor", "--project", project)
	if err != nil {
		t.Fatalf("init --agent cursor --project: %v", err)
	}

	// Assert: artifacts under project root, and NOTHING under cwd.
	for _, rel := range []string{
		".archcore/settings.json",
		filepath.Join(".cursor", "mcp.json"),
		filepath.Join(".cursor", "hooks.json"),
	} {
		if _, err := os.Stat(filepath.Join(project, rel)); err != nil {
			t.Errorf("expected %s under --project root: %v", rel, err)
		}
		if _, err := os.Stat(filepath.Join(elsewhere, rel)); !os.IsNotExist(err) {
			t.Errorf("artifact %s leaked into process cwd %s", rel, elsewhere)
		}
	}
}

// Spec 2: --agent implies non-interactive — no reinit confirm, no picker,
// even when .archcore/ already exists. Verified indirectly: the command must
// complete without a TTY (tests have none) and exit 0.
func TestInitCmd_AgentFlag_NoPromptsWithoutTTY_Spec(t *testing.T) {

	project := setupArchcoreDir(t) // .archcore/ already initialized

	_, err := runInitCmdForSpec(t, "--agent", "claude-code", "--project", project)
	if err != nil {
		t.Fatalf("re-init over existing .archcore/ must succeed without prompting: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project, ".claude", "settings.json")); err != nil {
		t.Errorf("expected claude hooks config: %v", err)
	}
}

// Spec 3: idempotent — second run leaves every written file byte-identical.
func TestInitCmd_AgentFlag_IdempotentRerun_Spec(t *testing.T) {

	project := t.TempDir()
	if _, err := runInitCmdForSpec(t, "--agent", "cursor", "--project", project); err != nil {
		t.Fatalf("first run: %v", err)
	}
	mcpPath := filepath.Join(project, ".cursor", "mcp.json")
	first, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := runInitCmdForSpec(t, "--agent", "cursor", "--project", project); err != nil {
		t.Fatalf("second run: %v", err)
	}
	second, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("second run changed %s:\nfirst:\n%s\nsecond:\n%s", mcpPath, first, second)
	}
}

// Spec 4: unknown agent id errors out, names the id, and writes nothing.
func TestInitCmd_AgentFlag_UnknownAgentErrors_Spec(t *testing.T) {

	project := t.TempDir()

	_, err := runInitCmdForSpec(t, "--agent", "not-a-real-agent", "--project", project)

	if err == nil {
		t.Fatal("expected error for unknown agent id")
	}
	entries, readErr := os.ReadDir(project)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Errorf("unknown agent must not leave artifacts, found: %v", entries)
	}
}
