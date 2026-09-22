package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// The server used to register five track prompts. Track orchestration now lives
// entirely in the plugin layer, so the CLI exposes none — one owner for the
// flow, no drift between an MCP prompt and the skill that duplicates it.
//
// These two tests are the inverted canary: they fail if prompts come back.
// That matters because re-registration is easy to do by accident — AddPrompt
// implicitly turns the capability back on, so a single stray registration
// would silently re-advertise the whole surface.

// TestServerAdvertisesNoPromptsCapability pins the handshake: a client that
// asks what the server can do must not be told it serves prompts.
func TestServerAdvertisesNoPromptsCapability(t *testing.T) {
	t.Parallel()

	base := initArchcore(t)
	c := newTestClient(t, base)

	if caps := c.GetServerCapabilities(); caps.Prompts != nil {
		t.Fatalf("server advertises a prompts capability; caps=%+v", caps)
	}
}

// TestListPromptsIsUnsupported pins the wire behavior, which is NOT an empty
// list: with the capability undeclared, mcp-go answers prompts/list with a
// JSON-RPC "method not found" instead of an empty result. Assert the typed
// error rather than its text — the wording is the library's, not our contract.
func TestListPromptsIsUnsupported(t *testing.T) {
	t.Parallel()

	base := initArchcore(t)
	c := newTestClient(t, base)

	res, err := c.ListPrompts(context.Background(), mcp.ListPromptsRequest{})
	if err == nil {
		t.Fatalf("ListPrompts succeeded and returned %d prompt(s), want a method-not-found error", len(res.Prompts))
	}
	if !errors.Is(err, mcp.ErrMethodNotFound) {
		t.Errorf("ListPrompts error = %v, want mcp.ErrMethodNotFound", err)
	}
}
