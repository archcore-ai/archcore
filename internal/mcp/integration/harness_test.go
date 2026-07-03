// Package integration contains in-process MCP server integration tests.
//
// Unlike the per-tool unit tests under internal/mcp/tools/, these tests wire
// the real server.MCPServer (via internal/mcp.NewServer) to an in-process
// client.Client (via mcp-go's client.NewInProcessClient), exercising the
// full registration + JSON marshalling + handler path. They catch bugs the
// unit suite is structurally blind to: tool-registration regressions,
// request/response shape mismatches, and multi-tool composition where one
// handler writes state another reads.
package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	mcpserver "archcore-cli/internal/mcp"
	"archcore-cli/internal/mcp/tools"
	"archcore-cli/internal/sync"
)

// initArchcore creates a fresh tempdir with an empty .archcore/ directory and
// returns the project base path. Tests use this as the baseDir passed to the
// MCP server.
func initArchcore(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".archcore"), 0o755); err != nil {
		t.Fatalf("mkdir .archcore: %v", err)
	}
	return base
}

// newTestClient builds a real MCP server for baseDir, wires it to an
// in-process client, performs the Start + Initialize handshake, and registers
// Close on cleanup. The returned client is ready for CallTool / ListTools.
//
// Per mcp-go v0.44.0 (client/inprocess_test.go:96-110), Start AND Initialize
// must both run before any other request, otherwise calls hang or error.
func newTestClient(t *testing.T, baseDir string) *client.Client {
	t.Helper()
	srv := mcpserver.NewServer(baseDir)

	c, err := client.NewInProcessClient(srv)
	if err != nil {
		t.Fatalf("NewInProcessClient: %v", err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Errorf("client.Close: %v", err)
		}
	})

	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("client.Start: %v", err)
	}

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "archcore-integration-test",
		Version: "0.0.0",
	}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		t.Fatalf("client.Initialize: %v", err)
	}

	return c
}

// mustCallTool invokes name with args and fails the test on transport error
// or on a handler-level error (IsError == true). The first TextContent of a
// tool error is included in the failure message so triage is one read.
//
// Tool failures arrive as result.IsError == true, NOT as a Go-level error —
// the Go error only fires on transport/protocol failure. Tests that ignore
// IsError silently miss handler bugs.
func mustCallTool(t *testing.T, c *client.Client, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	result, err := c.CallTool(context.Background(), req)
	if err != nil {
		t.Fatalf("CallTool(%s) transport error: %v", name, err)
	}
	if result.IsError {
		t.Fatalf("CallTool(%s) handler error: %s", name, firstText(result))
	}
	return result
}

// expectToolError is the inverse of mustCallTool: it expects the call to come
// back with IsError == true and returns the error text for further assertion.
// Fails the test on transport error or unexpected success.
func expectToolError(t *testing.T, c *client.Client, name string, args map[string]any) string {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	result, err := c.CallTool(context.Background(), req)
	if err != nil {
		t.Fatalf("CallTool(%s) transport error: %v", name, err)
	}
	if !result.IsError {
		t.Fatalf("CallTool(%s) expected error, got success: %s", name, firstText(result))
	}
	return firstText(result)
}

// decodeJSON unmarshals the first TextContent payload of result into T.
// Archcore tools always return JSON via mcp.NewToolResultText, so the first
// TextContent is always the payload.
func decodeJSON[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()
	text := firstText(result)
	if text == "" {
		t.Fatalf("decodeJSON: result has no TextContent payload")
	}
	var out T
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("decodeJSON: %v\npayload: %s", err, text)
	}
	return out
}

// firstText extracts the first mcp.TextContent.Text from a tool result, or "".
// Archcore tool results have exactly one TextContent (a JSON string).
func firstText(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	for _, c := range result.Content {
		if tc, ok := mcp.AsTextContent(c); ok {
			return tc.Text
		}
	}
	return ""
}

// loadManifest reads .archcore/.sync-state.json and returns the decoded
// manifest. Used as a corroborating assertion in mutation scenarios — it
// pinpoints WHERE the loop broke when a client-side assertion fails.
func loadManifest(t *testing.T, baseDir string) *sync.Manifest {
	t.Helper()
	m, err := sync.LoadManifest(baseDir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	return m
}

// createADR is a thin wrapper around create_document for the multi-tool
// scenarios. It creates an ADR at .archcore/<dir>/<slug>.adr.md (dir may be
// empty for root) and returns the path the server reports. Going through
// the real tool — instead of writing files directly — keeps the test
// exercising the layer it exists to test.
func createADR(t *testing.T, c *client.Client, dir, slug string) string {
	t.Helper()
	args := map[string]any{
		"type":     "adr",
		"filename": slug,
		"title":    "Doc " + slug,
	}
	if dir != "" {
		args["directory"] = dir
	}
	res := mustCallTool(t, c, "create_document", args)
	payload := decodeJSON[map[string]any](t, res)
	path, _ := payload["path"].(string)
	if path == "" {
		t.Fatalf("create_document(%s) returned empty path; payload=%v", slug, payload)
	}
	return path
}

// decodeListDocuments unmarshals a list_documents envelope and returns the
// documents page.
func decodeListDocuments(t *testing.T, result *mcp.CallToolResult) []tools.LocalDocument {
	t.Helper()
	return decodeJSON[struct {
		Documents []tools.LocalDocument `json:"documents"`
	}](t, result).Documents
}

// decodeListedDocs is decodeListDocuments for the integration package's
// reduced listedDoc shape.
func decodeListedDocs(t *testing.T, result *mcp.CallToolResult) []listedDoc {
	t.Helper()
	return decodeJSON[struct {
		Documents []listedDoc `json:"documents"`
	}](t, result).Documents
}
