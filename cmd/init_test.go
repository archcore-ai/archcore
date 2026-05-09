package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/config"
)

func healthyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ready":true}`))
	}
}

func TestRunInit_SyncNone(t *testing.T) {
	base := t.TempDir()
	settings := config.NewNoneSettings()
	result, err := runInit(context.Background(), base, settings)
	if err != nil {
		t.Fatalf("runInit: %v", err)
	}
	if result.serverReachable {
		t.Error("serverReachable should be false for sync none")
	}
	if !config.DirExists(base) {
		t.Error(".archcore/ directory not created")
	}
	s, err := config.Load(base)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Sync != config.SyncTypeNone {
		t.Errorf("Sync = %q, want %q", s.Sync, config.SyncTypeNone)
	}

	// Verify exact JSON format.
	data, err := os.ReadFile(filepath.Join(base, ".archcore", "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["project_id"]; ok {
		t.Error("none settings should not have project_id field")
	}
	if _, ok := raw["archcore_url"]; ok {
		t.Error("none settings should not have archcore_url field")
	}
}

func TestRunInit_SyncCloud(t *testing.T) {
	srv := httptest.NewServer(healthyHandler())
	defer srv.Close()

	// Override CloudServerURL for test.
	orig := config.CloudServerURL
	config.CloudServerURL = srv.URL
	defer func() { config.CloudServerURL = orig }()

	base := t.TempDir()
	settings := config.NewCloudSettings()
	result, err := runInit(context.Background(), base, settings)
	if err != nil {
		t.Fatalf("runInit: %v", err)
	}
	if !result.serverReachable {
		t.Error("serverReachable should be true")
	}
	s, err := config.Load(base)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Sync != config.SyncTypeCloud {
		t.Errorf("Sync = %q, want %q", s.Sync, config.SyncTypeCloud)
	}

	// Verify exact JSON format — should not have project_id (nil) or archcore_url.
	data, err := os.ReadFile(filepath.Join(base, ".archcore", "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["project_id"]; ok {
		t.Error("cloud settings should not have project_id field when nil")
	}
	if _, ok := raw["archcore_url"]; ok {
		t.Error("cloud settings should not have archcore_url field")
	}
}

func TestRunInit_SyncOnPrem(t *testing.T) {
	srv := httptest.NewServer(healthyHandler())
	defer srv.Close()

	base := t.TempDir()
	settings := config.NewOnPremSettings(srv.URL)
	result, err := runInit(context.Background(), base, settings)
	if err != nil {
		t.Fatalf("runInit: %v", err)
	}
	if !result.serverReachable {
		t.Error("serverReachable should be true")
	}
	s, err := config.Load(base)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Sync != config.SyncTypeOnPrem {
		t.Errorf("Sync = %q, want %q", s.Sync, config.SyncTypeOnPrem)
	}
	if s.ArchcoreURL != srv.URL {
		t.Errorf("ArchcoreURL = %q, want %q", s.ArchcoreURL, srv.URL)
	}

	// Verify exact JSON format — should not have project_id (nil), should have archcore_url.
	data, err := os.ReadFile(filepath.Join(base, ".archcore", "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["project_id"]; ok {
		t.Error("on-prem settings should not have project_id field when nil")
	}
	if _, ok := raw["archcore_url"]; !ok {
		t.Error("on-prem settings should have archcore_url field")
	}
}

func TestRunInit_ServerUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close() // close immediately

	base := t.TempDir()
	settings := config.NewOnPremSettings(srv.URL)
	result, err := runInit(context.Background(), base, settings)
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
	if result != nil {
		t.Fatal("expected nil result on server error")
	}
	if !errors.Is(err, ErrServerUnreachable) {
		t.Fatalf("expected ErrServerUnreachable, got: %v", err)
	}
	// Dirs should still be created even though server is unreachable.
	if !config.DirExists(base) {
		t.Error(".archcore/ directory should be created even when server is unreachable")
	}
}

func TestRunInit_InstallsHooksAndMCP(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	settings := config.NewNoneSettings()
	if _, err := runInit(context.Background(), base, settings); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	if err := runHooksInstallForAgent(base, agents.ClaudeCode); err != nil {
		t.Fatalf("runHooksInstallForAgent: %v", err)
	}

	// Verify .claude/settings.json has all 3 hook events.
	data, err := os.ReadFile(filepath.Join(base, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile .claude/settings.json: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var hooks map[string][]hookMatcher
	if err := json.Unmarshal(raw["hooks"], &hooks); err != nil {
		t.Fatalf("Unmarshal hooks: %v", err)
	}
	for _, event := range []string{"SessionStart"} {
		matchers, ok := hooks[event]
		if !ok {
			t.Errorf("missing hook event %s", event)
			continue
		}
		if len(matchers) != 1 {
			t.Errorf("event %s: want 1 matcher, got %d", event, len(matchers))
		}
	}

	// Verify .mcp.json has the archcore server entry.
	mcpData, err := os.ReadFile(filepath.Join(base, ".mcp.json"))
	if err != nil {
		t.Fatalf("ReadFile .mcp.json: %v", err)
	}
	var mcpRaw map[string]json.RawMessage
	if err := json.Unmarshal(mcpData, &mcpRaw); err != nil {
		t.Fatalf("Unmarshal .mcp.json: %v", err)
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(mcpRaw["mcpServers"], &servers); err != nil {
		t.Fatalf("Unmarshal mcpServers: %v", err)
	}
	if _, ok := servers["archcore"]; !ok {
		t.Error("missing archcore entry in .mcp.json mcpServers")
	}
}

func TestRunInit_HooksIdempotent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	settings := config.NewNoneSettings()
	if _, err := runInit(context.Background(), base, settings); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	// Run hooks install twice.
	for i := range 2 {
		if err := runHooksInstallForAgent(base, agents.ClaudeCode); err != nil {
			t.Fatalf("runHooksInstallForAgent call %d: %v", i+1, err)
		}
	}

	// Verify exactly 1 matcher per hook event (no duplicates).
	data, err := os.ReadFile(filepath.Join(base, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var hooks map[string][]hookMatcher
	if err := json.Unmarshal(raw["hooks"], &hooks); err != nil {
		t.Fatalf("Unmarshal hooks: %v", err)
	}
	for _, event := range []string{"SessionStart"} {
		if len(hooks[event]) != 1 {
			t.Errorf("event %s: want 1 matcher after idempotent install, got %d", event, len(hooks[event]))
		}
	}

	// Verify exactly 1 archcore entry in .mcp.json.
	mcpData, err := os.ReadFile(filepath.Join(base, ".mcp.json"))
	if err != nil {
		t.Fatalf("ReadFile .mcp.json: %v", err)
	}
	var mcpRaw map[string]json.RawMessage
	if err := json.Unmarshal(mcpData, &mcpRaw); err != nil {
		t.Fatalf("Unmarshal .mcp.json: %v", err)
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(mcpRaw["mcpServers"], &servers); err != nil {
		t.Fatalf("Unmarshal mcpServers: %v", err)
	}
	if len(servers) != 1 {
		t.Errorf("want 1 MCP server entry, got %d", len(servers))
	}
}

func TestRunInit_Idempotent(t *testing.T) {
	base := t.TempDir()
	for i := range 2 {
		_, err := runInit(context.Background(), base, config.NewNoneSettings())
		if err != nil {
			t.Fatalf("runInit call %d: %v", i+1, err)
		}
	}
	s, err := config.Load(base)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Sync != config.SyncTypeNone {
		t.Errorf("Sync = %q, want %q", s.Sync, config.SyncTypeNone)
	}
}

func TestInit_DetectsMultipleAgents(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	// Create agent marker directories before init.
	if err := os.MkdirAll(filepath.Join(base, ".cursor"), 0o755); err != nil {
		t.Fatalf("MkdirAll .cursor: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, ".gemini"), 0o755); err != nil {
		t.Fatalf("MkdirAll .gemini: %v", err)
	}

	settings := config.NewNoneSettings()
	if _, err := runInit(context.Background(), base, settings); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	// Now detect and install for detected agents.
	detected := agents.Detect(base)
	for _, agent := range detected {
		switch agent.ID {
		case agents.Cursor:
			runCursorHooksInstall(base)
		case agents.GeminiCLI:
			runGeminiCLIHooksInstall(base)
		}
	}

	// Verify .cursor/hooks.json exists.
	if _, err := os.Stat(filepath.Join(base, ".cursor", "hooks.json")); err != nil {
		t.Error("expected .cursor/hooks.json to exist")
	}
	// Verify .gemini/settings.json has hooks.
	data, err := os.ReadFile(filepath.Join(base, ".gemini", "settings.json"))
	if err != nil {
		t.Fatal("expected .gemini/settings.json to exist")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["hooks"]; !ok {
		t.Error("expected hooks in .gemini/settings.json")
	}
}

func TestResolveAgents_NoAgents_NonInteractive(t *testing.T) {
	base := t.TempDir()
	withInteractive(t, false)

	settings := config.NewNoneSettings()
	if _, err := runInit(context.Background(), base, settings); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	sel, err := resolveAgents(base)
	if err != nil {
		t.Fatalf("resolveAgents: %v", err)
	}
	if len(sel.agents) != 0 {
		t.Fatalf("expected no resolved agents, got %d", len(sel.agents))
	}
	if sel.outcome != outcomeNonInteractive {
		t.Errorf("outcome = %d, want outcomeNonInteractive(%d)", sel.outcome, outcomeNonInteractive)
	}
}

// withInteractive temporarily overrides the package-level isInteractive hook
// for tests. The package-level hooks are not safe under t.Parallel(), so
// callers must run serially.
func withInteractive(t *testing.T, v bool) {
	t.Helper()
	orig := isInteractive
	isInteractive = func() bool { return v }
	t.Cleanup(func() { isInteractive = orig })
}

// withPickAgents temporarily swaps the package-level pickAgents and
// isInteractive hooks for tests. Forces isInteractive=true so resolveAgents
// reaches the picker. Like withInteractive, it is not safe under t.Parallel().
func withPickAgents(t *testing.T, sel agentSelection) {
	t.Helper()
	withPickAgentsFn(t, func() (agentSelection, error) { return sel, nil })
}

// withPickAgentsFn is the lower-level seam that lets a test inject an
// arbitrary picker function (e.g. one that returns an error to exercise the
// error-propagation path).
func withPickAgentsFn(t *testing.T, fn agentPicker) {
	t.Helper()
	origPick := pickAgents
	origInteractive := isInteractive
	pickAgents = fn
	isInteractive = func() bool { return true }
	t.Cleanup(func() {
		pickAgents = origPick
		isInteractive = origInteractive
	})
}

// captureStdout redirects os.Stdout for the duration of fn and returns the
// captured bytes. Mirrors the pattern used in cmd/status_test.go and
// cmd/update_test.go. NOT safe under t.Parallel(): mutates the global
// os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	fn()

	w.Close()
	var out bytes.Buffer
	if _, err := out.ReadFrom(r); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return out.String()
}

func TestValidateAgentSelection(t *testing.T) {
	tests := []struct {
		name    string
		input   []agents.AgentID
		wantErr bool
	}{
		{name: "nil slice", input: nil, wantErr: true},
		{name: "empty slice", input: []agents.AgentID{}, wantErr: true},
		{name: "single real agent", input: []agents.AgentID{agents.ClaudeCode}, wantErr: false},
		{name: "skip sentinel only", input: []agents.AgentID{skipAgentSentinel}, wantErr: false},
		{name: "two real agents", input: []agents.AgentID{agents.ClaudeCode, agents.Cursor}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAgentSelection(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolveAgents_PicksTwoAgents(t *testing.T) {
	base := t.TempDir()
	want := []*agents.Agent{agents.ByID(agents.ClaudeCode), agents.ByID(agents.Cursor)}
	withPickAgents(t, agentSelection{outcome: outcomePicked, agents: want})

	sel, err := resolveAgents(base)
	if err != nil {
		t.Fatalf("resolveAgents: %v", err)
	}
	if sel.outcome != outcomePicked {
		t.Errorf("outcome = %d, want outcomePicked(%d)", sel.outcome, outcomePicked)
	}
	if len(sel.agents) != 2 {
		t.Fatalf("len(agents) = %d, want 2", len(sel.agents))
	}
	if sel.agents[0].ID != agents.ClaudeCode {
		t.Errorf("agents[0].ID = %q, want %q", sel.agents[0].ID, agents.ClaudeCode)
	}
	if sel.agents[1].ID != agents.Cursor {
		t.Errorf("agents[1].ID = %q, want %q", sel.agents[1].ID, agents.Cursor)
	}
}

func TestResolveAgents_UserSkipped(t *testing.T) {
	base := t.TempDir()
	withPickAgents(t, agentSelection{outcome: outcomeSkipped})

	sel, err := resolveAgents(base)
	if err != nil {
		t.Fatalf("resolveAgents: %v", err)
	}
	if sel.outcome != outcomeSkipped {
		t.Errorf("outcome = %d, want outcomeSkipped(%d)", sel.outcome, outcomeSkipped)
	}
	if len(sel.agents) != 0 {
		t.Errorf("len(agents) = %d, want 0", len(sel.agents))
	}
}

func TestResolveAgents_UserAborted(t *testing.T) {
	base := t.TempDir()
	withPickAgents(t, agentSelection{outcome: outcomeAborted})

	sel, err := resolveAgents(base)
	if err != nil {
		t.Fatalf("resolveAgents: %v", err)
	}
	if sel.outcome != outcomeAborted {
		t.Errorf("outcome = %d, want outcomeAborted(%d)", sel.outcome, outcomeAborted)
	}
	if len(sel.agents) != 0 {
		t.Errorf("len(agents) = %d, want 0", len(sel.agents))
	}
}

// TestResolveAgents_PickerError covers the path where the picker returns a
// non-aborted error: resolveAgents must propagate it unchanged so the caller
// (init/hooks/mcp RunE) can present a recovery hint.
func TestResolveAgents_PickerError(t *testing.T) {
	base := t.TempDir()
	wantErr := errors.New("huh exploded")
	withPickAgentsFn(t, func() (agentSelection, error) { return agentSelection{}, wantErr })

	sel, err := resolveAgents(base)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if len(sel.agents) != 0 {
		t.Errorf("len(agents) = %d, want 0 on error", len(sel.agents))
	}
}

func TestAgentsFromPicked(t *testing.T) {
	tests := []struct {
		name        string
		input       []agents.AgentID
		wantOutcome pickOutcome
		wantIDs     []agents.AgentID
	}{
		{name: "skip only", input: []agents.AgentID{skipAgentSentinel}, wantOutcome: outcomeSkipped},
		{name: "single real", input: []agents.AgentID{agents.ClaudeCode}, wantOutcome: outcomePicked, wantIDs: []agents.AgentID{agents.ClaudeCode}},
		{
			name:        "real plus skip — skip is filtered",
			input:       []agents.AgentID{agents.ClaudeCode, skipAgentSentinel},
			wantOutcome: outcomePicked,
			wantIDs:     []agents.AgentID{agents.ClaudeCode},
		},
		{
			name:        "two real",
			input:       []agents.AgentID{agents.ClaudeCode, agents.Cursor},
			wantOutcome: outcomePicked,
			wantIDs:     []agents.AgentID{agents.ClaudeCode, agents.Cursor},
		},
		{name: "unknown id only", input: []agents.AgentID{"nonsense"}, wantOutcome: outcomeSkipped},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agentsFromPicked(tt.input)
			if got.outcome != tt.wantOutcome {
				t.Errorf("outcome = %d, want %d", got.outcome, tt.wantOutcome)
			}
			if len(got.agents) != len(tt.wantIDs) {
				t.Fatalf("len(agents) = %d, want %d", len(got.agents), len(tt.wantIDs))
			}
			for i, want := range tt.wantIDs {
				if got.agents[i].ID != want {
					t.Errorf("agents[%d].ID = %q, want %q", i, got.agents[i].ID, want)
				}
			}
		})
	}
}

// installFromPicker drives the same path newInitCmd's RunE walks: resolve the
// picker, print status if empty, otherwise call installAgents. Tests use this
// to assert the install side-effects of the cobra closure without booting
// cobra itself.
func installFromPicker(t *testing.T, baseDir string) {
	t.Helper()
	sel, err := resolveAgents(baseDir)
	if err != nil {
		t.Fatalf("resolveAgents: %v", err)
	}
	if len(sel.agents) == 0 {
		printAgentSelectionStatus(sel)
		return
	}
	installAgents(baseDir, sel.agents)
}

func TestRunInit_PicksAgentsAndInstalls(t *testing.T) {
	base := t.TempDir()
	withPickAgents(t, agentSelection{outcome: outcomePicked, agents: []*agents.Agent{agents.ByID(agents.ClaudeCode)}})

	if _, err := runInit(context.Background(), base, config.NewNoneSettings()); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	installFromPicker(t, base)

	if _, err := os.Stat(filepath.Join(base, ".claude", "settings.json")); err != nil {
		t.Errorf(".claude/settings.json missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, ".mcp.json")); err != nil {
		t.Errorf(".mcp.json missing: %v", err)
	}
}

func TestRunInit_AbortShowsRecoveryHint(t *testing.T) {
	base := t.TempDir()
	withPickAgents(t, agentSelection{outcome: outcomeAborted})

	out := captureStdout(t, func() {
		if _, err := runInit(context.Background(), base, config.NewNoneSettings()); err != nil {
			t.Fatalf("runInit: %v", err)
		}
		installFromPicker(t, base)
	})

	if !strings.Contains(out, "Cancelled") {
		t.Errorf("output does not contain 'Cancelled':\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(base, ".archcore", "settings.json")); err != nil {
		t.Errorf("settings.json missing: %v", err)
	}
}

func TestRunInit_SkipShowsRecoveryHint(t *testing.T) {
	base := t.TempDir()
	withPickAgents(t, agentSelection{outcome: outcomeSkipped})

	out := captureStdout(t, func() {
		if _, err := runInit(context.Background(), base, config.NewNoneSettings()); err != nil {
			t.Fatalf("runInit: %v", err)
		}
		installFromPicker(t, base)
	})

	if !strings.Contains(out, "Skipped") {
		t.Errorf("output does not contain 'Skipped':\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(base, ".archcore", "settings.json")); err != nil {
		t.Errorf("settings.json missing: %v", err)
	}
}
