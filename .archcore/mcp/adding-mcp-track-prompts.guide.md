---
title: "How to Add a New MCP Track Prompt"
status: accepted
tags:
  - "mcp"
---

## Prerequisites

- An accepted ADR authorizing the new prompt. The current authorized set is fixed at five (`iso_track`, `sources_track`, `product_track`, `standard_track`, `architecture_track`) per `mcp/mcp-prompts-for-tracks-only.adr.md`. Adding a sixth requires a new ADR amending it.
- The new workflow satisfies `mcp/mcp-prompts-orchestration-only.rule.md`: two or more documents, `implements`/`depends_on`/`extends` linkage, multi-step with confirmations.
- Familiarity with `mark3labs/mcp-go` server API (`server.WithPromptCapabilities`, `s.AddPrompt`, `mcp.NewPrompt`, `mcp.GetPromptResult`, `mcp.PromptMessage`).
- Reading `internal/mcp/server.go` to understand current registration patterns for tools.

## Steps

### 1. Enable the prompts capability (once, if not already on)

In `internal/mcp/server.go`, ensure the server is constructed with prompts capability:

```go
s := server.NewMCPServer(
    "archcore",
    "1.0.0",
    server.WithInstructions(buildInstructions(language)),
    server.WithPromptCapabilities(true), // add this if missing
)
```

### 2. Create the handler file

Add `internal/mcp/prompts/<track_name>.go`. One file per track.

```go
package prompts

import (
    "context"
    "fmt"

    "github.com/mark3labs/mcp-go/mcp"
)

func NewISOTrackPrompt() mcp.Prompt {
    return mcp.NewPrompt("iso_track",
        mcp.WithPromptDescription(
            "Run ISO 29148 cascade: BRS → StRS → SyRS → SRS, "+
            "linking each level via 'implements'.",
        ),
        mcp.WithArgument("feature_name",
            mcp.ArgumentDescription("Short name of the feature being specified"),
            mcp.RequiredArgument(),
        ),
        mcp.WithArgument("scope",
            mcp.ArgumentDescription("One-line scope statement (optional)"),
        ),
    )
}

func HandleISOTrack(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
    feature, _ := req.Params.Arguments["feature_name"].(string)
    if feature == "" {
        return nil, fmt.Errorf("feature_name is required")
    }
    scope, _ := req.Params.Arguments["scope"].(string)

    return &mcp.GetPromptResult{
        Description: fmt.Sprintf("ISO 29148 cascade for %q", feature),
        Messages:    isoTrackMessages(feature, scope),
    }, nil
}
```

### 3. Compose the message sequence

Return a `PromptMessage` list that walks the agent through the cascade. Required structure:

1. **System framing** — one message setting the task and confirming the cascade chain.
2. **One user message per phase** — phase number, document type to create, what to fill, which relation to add to the previous phase, the confirmation gate before the next phase.
3. **Final verification message** — instruct the agent to call `list_relations` on the root document and confirm the full chain.

```go
func isoTrackMessages(feature, scope string) []mcp.PromptMessage {
    return []mcp.PromptMessage{
        {
            Role: "system",
            Content: mcp.NewTextContent(
                "You are running the ISO 29148 cascade for \""+feature+"\". " +
                "Follow phases sequentially. After EACH phase, summarize what " +
                "was created and ask the user 'ok to continue?' before the next.",
            ),
        },
        {
            Role: "user",
            Content: mcp.NewTextContent(
                "Phase 1 — BRS. Call create_document(type=\"brs\", filename=..., " +
                "directory=..., title=...). Use the standard BRS template; fill " +
                "Mission, Goals, Operational Concept, Success Criteria based on the scope: " +
                scope + ". Summarize and wait for confirmation.",
            ),
        },
        {
            Role: "user",
            Content: mcp.NewTextContent(
                "Phase 2 — StRS. Create the StRS document, then call " +
                "add_relation(source=<strs path>, target=<brs path>, type=\"implements\"). " +
                "Summarize and wait for confirmation.",
            ),
        },
        // Phases 3 & 4 follow the same pattern for SyRS and SRS.
        {
            Role: "user",
            Content: mcp.NewTextContent(
                "Final — call list_relations on the BRS path and confirm the chain " +
                "BRS ← StRS ← SyRS ← SRS is complete.",
            ),
        },
    }
}
```

Rules for message authoring:
- Reference the actual tool names (`create_document`, `add_relation`, `list_relations`) — the agent uses them as-is.
- Always specify the relation type and direction (source/target). Ambiguity here causes wrong edges.
- Always include a confirmation gate between phases. Without it, the agent runs the whole cascade without user input.
- Keep each phase message under ~400 chars. Long messages bury the action.

### 4. Register the prompt in the server

In `internal/mcp/server.go`, after the existing `s.AddTool(...)` block, add:

```go
import "archcore-cli/internal/mcp/prompts"

// ... inside NewServer, after AddTool calls:
s.AddPrompt(prompts.NewISOTrackPrompt(), prompts.HandleISOTrack)
```

Order: keep prompt registrations grouped together at the bottom of `NewServer`, separate from tool registrations.

### 5. Mention the prompt in instructions

In `mcpServerInstructions` (top of `internal/mcp/server.go`), the `WORKFLOW PROMPTS` section lists all available prompts. Append your new prompt name and a one-line summary.

```
WORKFLOW PROMPTS (when client supports MCP prompts):
  iso_track          — BRS → StRS → SyRS → SRS cascade
  sources_track      — MRD → BRD → URD discovery flow
  product_track      — PRD → plan
  standard_track     — ADR → rule → guide
  architecture_track — ADR → spec → plan
```

### 6. Add tests

Per `code-quality/in-process-mcp-integration-tests.adr.md`, extend `internal/mcp/integration/` with a prompt-flow test:

- `prompts_test.go` — call the in-process MCP client's `ListPrompts()`, assert the new prompt appears with expected name, description, and arguments.
- `prompts_test.go` — call `GetPrompt(name, args)`, assert message count and that the chain types appear in the returned text (e.g., the BRS message mentions `type="brs"`).
- Unit test in `internal/mcp/prompts/<track>_test.go` for `HandleXTrack`: missing required arg returns error, optional args have working defaults, message ordering matches phase order.

### 7. Document in user-facing guide

If a guide like `using-tracks-via-mcp.guide.md` exists, add a section for the new track with example client invocation. If not (first track prompt), create it as part of this work.

## Verification

After implementing, run:

```bash
go build -o archcore .
go test ./internal/mcp/...
```

Manual smoke test using Claude Desktop or any prompts-aware MCP client:

1. Connect the client to the local `archcore mcp` server.
2. Confirm the new prompt shows up in the slash menu (Claude Desktop: `+` button → MCP prompts).
3. Invoke the prompt with the required arguments.
4. Verify the agent walks through phases, asks for confirmation, creates the documents in the correct types, and adds the `implements` relations.
5. Run `list_relations` on the root document; confirm the full cascade chain.

CI/regression checks:
- `go test ./internal/mcp/integration/ -run TestPrompt` returns green.
- `prompts/list` includes the new prompt name.
- Token cost of `mcpServerInstructions` did not grow disproportionately (one line per prompt is the budget).

## Common Issues

**Prompt doesn't appear in the client's slash menu.**
The client may not implement the `prompts` capability. Confirm support for the target client per `mcp-prompts-for-tracks-only.adr.md`. Headless agents and parts of Cursor/Continue ignore prompts. This is expected — they fall back to instruction-driven cascades.

**Agent ignores the message sequence and free-styles.**
The model may compress multi-message prompts. Mitigation: keep the system message firm ("You MUST follow phases sequentially"), make each user message self-contained, and include the confirmation gate as an explicit instruction, not implied.

**Wrong relation direction (`implements` reversed).**
Always state source/target explicitly in the prompt text. Verify in tests that the message text contains the exact substring `source=<later phase>, target=<earlier phase>`.

**Naming collision with plugin skill (`/archcore:iso-track` vs `/mcp__archcore__iso_track`).**
The two coexist intentionally (per the parent ADR). Disambiguate in the `WithPromptDescription`: mention "portable, works in any MCP client" so users understand when to pick the MCP version vs the plugin skill.

**Argument extraction returns `nil` and panics.**
`req.Params.Arguments` is `map[string]interface{}`. Always type-assert with the comma-ok form: `feature, _ := req.Params.Arguments["feature_name"].(string)`. Check for empty string before use; required-argument enforcement happens at the protocol level only if the client respects it.

**Instruction bloat from listing prompts.**
The `WORKFLOW PROMPTS` block is one line per prompt. Resist the urge to add detailed descriptions inline — those belong in `WithPromptDescription`. Coordinate with `cli/mcp-token-optimization.idea.md` if total instruction size becomes a concern.
