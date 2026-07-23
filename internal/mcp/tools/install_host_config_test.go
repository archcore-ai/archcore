package tools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func callInstallHostConfig(t *testing.T, wire HostWiringFunc, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Name = "install_host_config"
	req.Params.Arguments = args
	res, err := HandleInstallHostConfig(wire)(context.Background(), req)
	if err != nil {
		t.Fatalf("handler must never return a Go error, got: %v", err)
	}
	return res
}

func TestHandleInstallHostConfig_MissingHostParam(t *testing.T) {
	t.Parallel()
	called := false
	wire := func(host string, allDetected bool) ([]byte, error) {
		called = true
		return nil, nil
	}

	res := callInstallHostConfig(t, wire, map[string]any{})

	if !res.IsError {
		t.Error("missing host must produce an error result")
	}
	if called {
		t.Error("executor must not run without a host")
	}
	if !strings.Contains(resultText(t, res), "host") {
		t.Errorf("error should name the missing parameter: %s", resultText(t, res))
	}
}

func TestHandleInstallHostConfig_PassesArgsThrough(t *testing.T) {
	t.Parallel()
	var gotHost string
	var gotAll bool
	wire := func(host string, allDetected bool) ([]byte, error) {
		gotHost, gotAll = host, allDetected
		return []byte(`{"agents":[]}`), nil
	}

	res := callInstallHostConfig(t, wire,
		map[string]any{"host": "cursor", "all_detected": true})

	if res.IsError {
		t.Fatalf("unexpected error result: %s", resultText(t, res))
	}
	if gotHost != "cursor" || !gotAll {
		t.Errorf("wire(%q, %v), want (\"cursor\", true)", gotHost, gotAll)
	}
	if resultText(t, res) != `{"agents":[]}` {
		t.Errorf("report must pass through verbatim, got %s", resultText(t, res))
	}
}

func TestHandleInstallHostConfig_ExecutorErrorSanitized(t *testing.T) {
	t.Parallel()
	// A *fs.PathError embedding an absolute path must reach the client as a
	// path-free I/O class (no-absolute-paths-in-mcp-errors.rule).
	pathErr := &fs.PathError{Op: "mkdir", Path: "/Users/someone/project/.claude", Err: fs.ErrPermission}
	wire := func(host string, allDetected bool) ([]byte, error) {
		return nil, fmt.Errorf("installing: %w", pathErr)
	}

	res := callInstallHostConfig(t, wire, map[string]any{"host": "claude-code"})

	if !res.IsError {
		t.Fatal("executor failure must produce an error result")
	}
	text := resultText(t, res)
	if strings.Contains(text, "/Users/") {
		t.Errorf("error leaks an absolute path: %s", text)
	}
	if !strings.Contains(text, "permission denied") {
		t.Errorf("error should carry the I/O class: %s", text)
	}
}

func TestHandleInstallHostConfig_PlainErrorTextKept(t *testing.T) {
	t.Parallel()
	wire := func(host string, allDetected bool) ([]byte, error) {
		return nil, errors.New(`unknown agent "nope"`)
	}

	res := callInstallHostConfig(t, wire, map[string]any{"host": "claude-code"})

	if !res.IsError {
		t.Fatal("expected error result")
	}
	if !strings.Contains(resultText(t, res), `unknown agent "nope"`) {
		t.Errorf("validation error text must survive sanitization: %s", resultText(t, res))
	}
}
