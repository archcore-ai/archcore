package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// reqFor builds the hookRequest a leaf would hand a handler.
func reqFor(t *testing.T, host, baseDir, payload string) hookRequest {
	t.Helper()
	return hookRequest{
		baseDir: baseDir,
		dialect: dialectByID(t, host),
		event:   eventPreToolUse,
		payload: decodeHookPayload(strings.NewReader(payload)),
	}
}

// bg is the context a test hands a handler directly.
func bg() context.Context { return context.Background() }

// postReq builds a PostToolUse request carrying an MCP document payload.
func postReq(t *testing.T, baseDir, tool, docPath string) hookRequest {
	t.Helper()
	return hookRequest{
		baseDir: baseDir,
		dialect: hookDialects[0],
		event:   eventPostToolUse,
		payload: mcpPayload(tool, docPath),
	}
}

// emitSessionStart runs the session-start path for a host and returns exactly
// what that host would receive on stdout. It goes through emitDecision because
// that is now the only writer — the shape is a property of the envelope, not of
// a per-host output struct.
func emitSessionStart(t *testing.T, host, baseDir, version string) string {
	t.Helper()
	d := dialectByID(t, host)
	dec := sessionStartHandler(version)(bg(), hookRequest{
		baseDir: baseDir,
		dialect: d,
		event:   eventSessionStart,
		payload: decodeHookPayload(strings.NewReader("{}")),
	})
	return captureStdout(t, func() { emitDecision(d, eventSessionStart, dec) })
}

// sessionContextOf returns just the injected context text for a host.
func sessionContextOf(t *testing.T, host, baseDir, version string) string {
	t.Helper()
	var wrapper struct {
		HookSpecificOutput map[string]any `json:"hookSpecificOutput"`
		AdditionalContext  string         `json:"additionalContext"`
	}
	if err := json.Unmarshal([]byte(emitSessionStart(t, host, baseDir, version)), &wrapper); err != nil {
		t.Fatalf("session-start output is not JSON: %v", err)
	}
	if s, ok := wrapper.HookSpecificOutput["additionalContext"].(string); ok {
		return s
	}
	return wrapper.AdditionalContext
}
