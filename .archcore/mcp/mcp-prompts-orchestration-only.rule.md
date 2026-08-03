---
title: "MCP Prompts Are Reserved for Multi-Document Track Orchestration"
status: accepted
tags:
  - "mcp"
---

## Rule

A track is a sequence of two or more linked documents created together, for example `brs → strs → syrs → srs` or `adr → rule → guide`.

1. The Archcore MCP server MUST expose a prompt only when that prompt orchestrates a multi-document track.
2. The developer MAY add a prompt only WHEN the workflow creates two or more documents in a defined sequence.
3. The developer MAY add a prompt only WHEN the prompt itself links the documents with `implements`, `depends_on`, or `extends` relations.
4. The developer MAY add a prompt only WHEN the flow needs user confirmation between steps.
5. The developer MAY add a prompt only WHEN a single tool call plus instruction text cannot express the orchestration.
6. The developer MUST NOT add a prompt for single-document creation, such as `write_brs` or `write_adr`. Templates and instructions cover that case.
7. The developer MUST NOT add a prompt for read-only or one-shot reporting, such as `status`, `graph`, or `find_duplicates`. Tools cover that case.
8. The developer MUST NOT add a prompt for a diagnostic or maintenance flow that follows no fixed multi-document cascade, such as `audit_health`, `actualize`, or `propose_relations`. Plugin skills cover that case.
9. The developer MUST NOT add a prompt for onboarding, help, or tour content.
10. The developer MUST NOT add a prompt for a capture or migration workflow, such as `capture_module` or `migrate_prd_to_iso`. Plugin skills currently own those workflows; a new ADR is required to move one into the MCP surface.
11. IF a change adds a sixth prompt, THEN the developer MUST record a new ADR that amends the accepted track-only decision.

The current authorized set is exactly `iso_track`, `sources_track`, `product_track`, `standard_track`, and `architecture_track`.

## Rationale

- The MCP `tools` layer stays CRUD-primitive on purpose. Prompts are the place for orchestrated multi-step workflows, which keeps the tools primitive.
- Without a scope rule, the prompt list grows into "everything that could be a workflow", duplicates plugin skills, and fills the slash-menu.
- Single-document creation gains nothing from a prompt: `create_document(type="brs")` against the standard template already produces the same result as a `write_brs` prompt would.
- Diagnostic flows such as `audit_health` and `actualize` need agent capabilities that headless MCP clients lack (`Read`, `Grep`, sub-agents), so they belong in plugin skills rather than in the protocol-portable layer.
- A small prompt surface keeps the `prompts/list` payload small, which lowers token cost.

## Examples

Non-normative examples.

### Good — orchestrated multi-document track

```go
// internal/mcp/prompts/iso_track.go
func NewISOTrackPrompt() mcp.Prompt {
    return mcp.NewPrompt("iso_track",
        mcp.WithPromptDescription("Run ISO 29148 cascade: BRS → StRS → SyRS → SRS"),
        mcp.WithArgument("feature_name", mcp.RequiredArgument()),
    )
}

func HandleISOTrack(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
    feature := req.Params.Arguments["feature_name"]
    return &mcp.GetPromptResult{
        Description: fmt.Sprintf("ISO 29148 cascade for %q", feature),
        Messages: []mcp.PromptMessage{
            // Phase 1: BRS — confirm — Phase 2: StRS implements BRS — confirm — ...
        },
    }, nil
}
```

It qualifies: four documents, three `implements` edges, four confirmation points, and no single-tool-call equivalent.

### Bad — single-document creation as a prompt

```go
// internal/mcp/prompts/write_brs.go  ← do NOT add
func NewWriteBRSPrompt() mcp.Prompt {
    return mcp.NewPrompt("write_brs", ...)
}
```

It fails: one document, no linking, no orchestration. Use `create_document(type="brs")` with the standard template.

### Bad — one-shot diagnostic as a prompt

```go
// internal/mcp/prompts/find_duplicates.go  ← do NOT add
func NewFindDuplicatesPrompt() mcp.Prompt {
    return mcp.NewPrompt("find_duplicates", ...)
}
```

It fails: one analysis pass, no document cascade. IF the capability is needed, THEN add a tool such as `find_similar_documents`, not a prompt.

### Bad — onboarding tour as a prompt

```go
// internal/mcp/prompts/tour.go  ← do NOT add
func NewTourPrompt() mcp.Prompt { ... }
```

It fails: no document orchestration. This content belongs in `instructions` or external documentation.

## Enforcement

- Code review: the reviewer MUST reject a new file in `internal/mcp/prompts/` unless the prompt creates two or more documents and adds at least one relation in its message script.
- ADR gate: the reviewer MUST block a pull request that adds a sixth prompt, or any non-track prompt, while the amending ADR is missing.
- Manual audit: during `/archcore:review`, the reviewer compares `internal/mcp/prompts/` against the authorized set and flags deviations.
- Description check: every prompt's `WithPromptDescription` MUST name the cascade chain, for example "BRS → StRS → SyRS → SRS". A description that reads as a single action, such as "Write a BRS document", indicates a prompt that this rule forbids.
