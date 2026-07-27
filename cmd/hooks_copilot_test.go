package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"archcore-cli/internal/agents"
)

func TestRunHooksInstallForAgent_CopilotAlsoInstallsMCP(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	if err := runHooksInstallForAgent(base, agents.Copilot); err != nil {
		t.Fatalf("runHooksInstallForAgent: %v", err)
	}

	// The workspace-root .mcp.json should exist with the "mcpServers" key —
	// the only project-level MCP source Copilot CLI reads.
	data, err := os.ReadFile(filepath.Join(base, ".mcp.json"))
	if err != nil {
		t.Fatalf("ReadFile .mcp.json: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["mcpServers"]; !ok {
		t.Error("missing 'mcpServers' in .mcp.json")
	}
}

// TestHandleSessionStart_CopilotNativeShape pins the output schema divergence.
// The hooks config we install registers camelCase "sessionStart" — Copilot's
// NATIVE format — whose documented output is a bare top-level
// additionalContext. Emitting Claude's hookSpecificOutput wrapper here parses
// as nothing: the hook exits 0, the user sees no error, and the session simply
// starts with no archcore context. That silence is why this is a test and not
// a comment.
func TestHandleSessionStart_CopilotNativeShape(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	out, err := handleSessionStart(base, "v0.0.0-test", shapeCopilotNative)
	if err != nil {
		t.Fatalf("handleSessionStart: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["hookSpecificOutput"]; ok {
		t.Error("hookSpecificOutput is Claude's wrapper — Copilot ignores it entirely")
	}
	if _, ok := raw["systemMessage"]; ok {
		t.Error("Copilot's sessionStart schema has no systemMessage slot")
	}
	ctx, ok := raw["additionalContext"]
	if !ok {
		t.Fatal("missing top-level additionalContext")
	}
	var s string
	if err := json.Unmarshal(ctx, &s); err != nil {
		t.Fatalf("additionalContext must be a JSON string: %v", err)
	}
	if s == "" {
		t.Error("additionalContext is empty — no context would reach the model")
	}
}

// The other hosts must keep the Claude-compatible wrapper: same context, same
// helper, different frame. A refactor that collapsed both onto one shape would
// break whichever host it did not pick.
func TestHandleSessionStart_ClaudeCompatShapeUnchanged(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	out, err := handleSessionStart(base, "v0.0.0-test", shapeClaudeCompat)
	if err != nil {
		t.Fatalf("handleSessionStart: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["additionalContext"]; ok {
		t.Error("top-level additionalContext is Copilot's shape, not Claude's")
	}
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(raw["hookSpecificOutput"], &wrapper); err != nil {
		t.Fatalf("Unmarshal hookSpecificOutput: %v", err)
	}
	if _, ok := wrapper["additionalContext"]; !ok {
		t.Error("missing additionalContext inside hookSpecificOutput")
	}
}
