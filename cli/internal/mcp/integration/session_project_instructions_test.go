package integration

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	mcpserver "archcore-cli/internal/mcp"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func instructionsOf(t *testing.T, baseDir string) string {
	t.Helper()
	c, err := client.NewInProcessClient(mcpserver.NewServer(baseDir, "test"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		t.Fatal(err)
	}
	req := mcp.InitializeRequest{}
	req.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	req.Params.ClientInfo = mcp.Implementation{Name: "archcore-integration-test", Version: "0.0.0"}
	result, err := c.Initialize(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	return result.Instructions
}

// TestSessionProject_InstructionsDescribeTheProjectAbove: the server started in
// a folder below a root carries the project's language in its instructions,
// exactly as at the root (session-project-is-nearest-store.adr).
func TestSessionProject_InstructionsDescribeTheProjectAbove(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	a := filepath.Join(w.dir, "a")
	writeFixtureFile(t, filepath.Join(a, ".archcore", "settings.json"), `{"sync":"none","language":"ru"}`)
	below := w.plain(t, "a/src/deep")

	atRoot := instructionsOf(t, a)
	if !strings.Contains(atRoot, "LANGUAGE:") {
		t.Fatalf("the root instructions carry no language notice:\n%s", atRoot)
	}
	if got := instructionsOf(t, below); got != atRoot {
		t.Errorf("instructions below the root differ from the root's:\n below: %s\n  root: %s", got, atRoot)
	}
}
