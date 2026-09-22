package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/wiring"
)

func decodeWiringReport(t *testing.T, raw []byte) wiringReport {
	t.Helper()
	var report wiringReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("report is not valid JSON: %v\n%s", err, raw)
	}
	return report
}

// The executor is the backend of the install_host_config MCP tool: it must
// initialize .archcore/ when absent, write every artifact under the server's
// baseDir, and return a machine-readable report.
func TestHostWiringExecutor_ClaudeCode_FreshProject(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	raw, err := hostWiringExecutor(base, "claude-code", false)
	if err != nil {
		t.Fatalf("executor: %v", err)
	}
	report := decodeWiringReport(t, raw)

	if !report.ArchcoreInitialized {
		t.Error("fresh project: report must flag archcore_initialized")
	}
	if len(report.Agents) != 1 || report.Agents[0].Agent != "claude-code" {
		t.Fatalf("want single claude-code agent report, got %+v", report.Agents)
	}
	r := report.Agents[0]
	if len(r.Errors) != 0 {
		t.Errorf("unexpected errors: %v", r.Errors)
	}
	if !r.HooksSupported {
		t.Error("claude-code supports hooks")
	}

	for _, rel := range []string{
		".archcore/settings.json",
		".mcp.json",
		filepath.Join(".claude", "settings.json"),
		"CLAUDE.md",
		"AGENTS.md",
	} {
		if _, err := os.Stat(filepath.Join(base, rel)); err != nil {
			t.Errorf("expected artifact %s: %v", rel, err)
		}
	}
	if r.MCPConfigPath != ".mcp.json" {
		t.Errorf("mcp_config_path = %q, want project-relative %q", r.MCPConfigPath, ".mcp.json")
	}
	// The report must name EVERY instruction file the write touched: CLAUDE.md
	// as the primary path and AGENTS.md as the extra — an MCP client (the
	// plugin) relays this report, so an unnamed file would be written to the
	// user's repo silently.
	if r.Instructions != "CLAUDE.md" {
		t.Errorf("instructions_path = %q, want %q", r.Instructions, "CLAUDE.md")
	}
	if len(r.InstructionsExtra) != 1 || r.InstructionsExtra[0] != "AGENTS.md" {
		t.Errorf("instructions_extra_paths = %v, want [AGENTS.md]", r.InstructionsExtra)
	}
}

// TestHostWiringExecutor_ClaudeCode_UpgradeFromLegacy: rewiring a pre-migration
// project (hand-written CLAUDE.md + legacy .claude/rules/archcore.md) must
// migrate the legacy file away and upsert CLAUDE.md without losing user content.
func TestHostWiringExecutor_ClaudeCode_UpgradeFromLegacy(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	user := "# My Project\n\nHand-written Claude guidance.\n"
	if err := os.WriteFile(filepath.Join(base, "CLAUDE.md"), []byte(user), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	legacy := filepath.Join(base, ".claude", "rules", "archcore.md")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(legacy, []byte("legacy nudge\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	raw, err := hostWiringExecutor(base, "claude-code", false)
	if err != nil {
		t.Fatalf("executor: %v", err)
	}
	report := decodeWiringReport(t, raw)
	if len(report.Agents) != 1 || len(report.Agents[0].Errors) != 0 {
		t.Fatalf("unexpected agent reports: %+v", report.Agents)
	}

	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("legacy .claude/rules/archcore.md should be migrated away, stat err = %v", err)
	}
	claudeMD, err := os.ReadFile(filepath.Join(base, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("ReadFile CLAUDE.md: %v", err)
	}
	if !strings.Contains(string(claudeMD), "Hand-written Claude guidance.") {
		t.Error("user CLAUDE.md content was lost during upgrade")
	}
	if n := strings.Count(string(claudeMD), "<!-- archcore:start -->"); n != 1 {
		t.Errorf("want exactly 1 managed block in CLAUDE.md after upgrade, got %d", n)
	}
}

// The report is returned to an MCP client, so every path and error string in
// it must be project-relative (no-absolute-paths-in-mcp-errors.rule).
func TestHostWiringExecutor_ReportHasNoAbsolutePaths(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	raw, err := hostWiringExecutor(base, "claude-code", false)
	if err != nil {
		t.Fatalf("executor: %v", err)
	}
	if strings.Contains(string(raw), base) {
		t.Errorf("report leaks the absolute project root %q:\n%s", base, raw)
	}
}

// Per-agent installer failures land as error strings inside the report — they
// must be sanitized like top-level errors: an I/O failure yields a path-free
// class description, never the raw *fs.PathError text.
func TestHostWiringExecutor_AgentErrorsAreSanitized(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	// Pre-create .archcore/ so init succeeds, then block the hook/MCP writes:
	// a FILE where claude-code's .claude directory must go makes MkdirAll fail
	// with a *fs.PathError embedding the absolute path.
	if _, err := wiring.EnsureProjectInitialized(base); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, ".claude"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	raw, err := hostWiringExecutor(base, "claude-code", false)
	if err != nil {
		t.Fatalf("executor must report per-agent errors, not fail: %v", err)
	}
	report := decodeWiringReport(t, raw)
	if len(report.Agents) != 1 || len(report.Agents[0].Errors) == 0 {
		t.Fatalf("expected per-agent errors in report, got %+v", report.Agents)
	}
	if strings.Contains(string(raw), base) {
		t.Errorf("error strings leak the absolute project root %q:\n%s", base, raw)
	}
}

func TestHostWiringExecutor_UnknownHostErrorsBeforeWrites(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	_, err := hostWiringExecutor(base, "not-an-agent", false)

	if err == nil {
		t.Fatal("expected error for unknown host")
	}
	if !strings.Contains(err.Error(), "not-an-agent") {
		t.Errorf("error should name the id: %v", err)
	}
	entries, readErr := os.ReadDir(base)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Errorf("unknown host must not leave artifacts, found %v", entries)
	}
}

func TestHostWiringExecutor_AllDetectedAddsMarkedAgents(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	// A .cursor/ marker directory makes Cursor auto-detected.
	if err := os.MkdirAll(filepath.Join(base, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}

	raw, err := hostWiringExecutor(base, "claude-code", true)
	if err != nil {
		t.Fatalf("executor: %v", err)
	}
	report := decodeWiringReport(t, raw)

	got := map[string]bool{}
	for _, r := range report.Agents {
		got[r.Agent] = true
	}
	if !got["claude-code"] || !got["cursor"] {
		t.Errorf("all_detected must cover claude-code + detected cursor, got %+v", report.Agents)
	}
	// Cursor's project-level MCP config must carry the cwd-independence args.
	data, err := os.ReadFile(filepath.Join(base, ".cursor", "mcp.json"))
	if err != nil {
		t.Fatalf("expected .cursor/mcp.json: %v", err)
	}
	if !strings.Contains(string(data), "${workspaceFolder}") {
		t.Errorf(".cursor/mcp.json must pass --project ${workspaceFolder}:\n%s", data)
	}
}

// Instruction attribution in a mixed run: claude-code's entry carries its full
// touch-set (CLAUDE.md primary + AGENTS.md extra); cursor's entry carries
// AGENTS.md as its own primary with no extras. Both agents legitimately touch
// AGENTS.md — the report must say so per agent, not lose the file to the
// dedupe (the pre-fix behavior attributed AGENTS.md to nobody when claude-code
// ran alone, and only to cursor in a mixed run).
func TestHostWiringExecutor_MixedAgents_InstructionAttribution(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}

	raw, err := hostWiringExecutor(base, "claude-code", true)
	if err != nil {
		t.Fatalf("executor: %v", err)
	}
	report := decodeWiringReport(t, raw)

	byAgent := map[string]wiringAgentReport{}
	for _, r := range report.Agents {
		byAgent[r.Agent] = r
	}

	claude, ok := byAgent["claude-code"]
	if !ok {
		t.Fatalf("no claude-code entry in report: %+v", report.Agents)
	}
	if claude.Instructions != "CLAUDE.md" {
		t.Errorf("claude-code instructions_path = %q, want CLAUDE.md", claude.Instructions)
	}
	if len(claude.InstructionsExtra) != 1 || claude.InstructionsExtra[0] != "AGENTS.md" {
		t.Errorf("claude-code instructions_extra_paths = %v, want [AGENTS.md]", claude.InstructionsExtra)
	}

	cursor, ok := byAgent["cursor"]
	if !ok {
		t.Fatalf("no cursor entry in report: %+v", report.Agents)
	}
	if cursor.Instructions != "AGENTS.md" {
		t.Errorf("cursor instructions_path = %q, want AGENTS.md", cursor.Instructions)
	}
	if len(cursor.InstructionsExtra) != 0 {
		t.Errorf("cursor instructions_extra_paths = %v, want empty", cursor.InstructionsExtra)
	}
}

func TestHostWiringExecutor_IdempotentSecondRun(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	exec := hostWiringExecutor

	// Byte-stability is asserted for every file the executor writes on this
	// path — the MCP config AND both instruction targets. The instruction
	// upserts are the seam where a duplicate managed block would appear.
	tracked := []string{".mcp.json", "CLAUDE.md", "AGENTS.md"}

	if _, err := exec(base, "claude-code", false); err != nil {
		t.Fatal(err)
	}
	first := map[string][]byte{}
	for _, rel := range tracked {
		data, err := os.ReadFile(filepath.Join(base, rel))
		if err != nil {
			t.Fatalf("ReadFile %s after first run: %v", rel, err)
		}
		first[rel] = data
	}

	raw, err := exec(base, "claude-code", false)
	if err != nil {
		t.Fatal(err)
	}
	report := decodeWiringReport(t, raw)
	if report.ArchcoreInitialized {
		t.Error("second run must not report archcore_initialized")
	}
	for _, rel := range tracked {
		second, err := os.ReadFile(filepath.Join(base, rel))
		if err != nil {
			t.Fatalf("ReadFile %s after second run: %v", rel, err)
		}
		if string(first[rel]) != string(second) {
			t.Errorf("second run changed %s:\nfirst:\n%s\nsecond:\n%s", rel, first[rel], second)
		}
	}
}
