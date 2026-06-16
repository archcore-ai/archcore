---
title: "MCP Prompts Are Reserved for Multi-Document Track Orchestration"
status: accepted
tags:
  - "mcp"
---

## Rule

MCP `prompts` exposed by `archcore mcp` MUST orchestrate a multi-document track. A track is a sequence of two or more linked documents created together (e.g., `brs → strs → syrs → srs`, `adr → rule → guide`).

A new prompt MAY be added only when ALL of the following hold:

1. The workflow creates two or more documents in a defined sequence.
2. The documents are linked by `implements`, `depends_on`, or `extends` relations defined by the prompt.
3. The flow benefits from user confirmation between steps (not a one-shot tool call).
4. The orchestration cannot be expressed cleanly as a single tool plus instruction text.

A new prompt MUST NOT be added for:

- Single-document creation (`write_brs`, `write_adr`, etc.) — templates and instructions cover these.
- Read-only or one-shot reporting (`status`, `graph`, `find_duplicates`) — use tools (or future resources).
- Diagnostic or maintenance flows that don't follow a fixed multi-doc cascade (`audit_health`, `actualize`, `propose_relations`) — these belong in plugin skills or future tools.
- Onboarding/help/tour content — instructions and external docs cover this.
- Capture or migration workflows (`capture_module`, `migrate_prd_to_iso`) — currently scoped to plugin skills; reconsider only via a new ADR.

The current authorized set is exactly: `iso_track`, `sources_track`, `product_track`, `standard_track`, `architecture_track` (per `mcp/mcp-prompts-for-tracks-only.adr.md`). Adding a sixth prompt requires a new ADR.

## Rationale

- The MCP `tools` layer is intentionally CRUD-primitive (`mcp/no-auto-relations-on-create-document.adr.md` upholds the same principle for relations). Prompts are the canonical place to put orchestrated multi-step workflows so tools stay primitive.
- Without a clear scope rule, the prompt list grows into "everything that could be a workflow," duplicating plugin skills and bloating the slash-menu.
- Single-doc creation has zero orchestration value beyond what templates plus the existing `WHEN TO CREATE` instructions already provide. A `write_brs` prompt gives no measurable lift over calling `create_document(type="brs")` against the standard template.
- Diagnostic flows (`audit_health`, `actualize`) need richer agent capabilities (`Read`, `Grep`, sub-agents) that headless MCP clients lack — they belong in plugin skills, not in the protocol-portable layer.
- Restricting the surface keeps `prompts/list` payload small (token cost) and the user-facing slash-menu uncluttered.

## Examples

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

Why it qualifies: four documents, three `implements` edges, four confirmation points, cannot be a single tool call.

### Bad — single-document creation as a prompt

```go
// internal/mcp/prompts/write_brs.go  ← do NOT add
func NewWriteBRSPrompt() mcp.Prompt {
    return mcp.NewPrompt("write_brs", ...)
}
```

Why it fails: one document, no linking, no orchestration. Use `create_document(type="brs")` with the standard template instead.

### Bad — one-shot diagnostic as a prompt

```go
// internal/mcp/prompts/find_duplicates.go  ← do NOT add
func NewFindDuplicatesPrompt() mcp.Prompt {
    return mcp.NewPrompt("find_duplicates", ...)
}
```

Why it fails: one analysis pass, no document creation cascade. If needed, add a tool (e.g., `find_similar_documents`) — not a prompt.

### Bad — onboarding tour as a prompt

```go
// internal/mcp/prompts/tour.go  ← do NOT add
func NewTourPrompt() mcp.Prompt { ... }
```

Why it fails: no document orchestration. Belongs in `instructions` or external documentation.

## Enforcement

- **Code review:** any new file in `internal/mcp/prompts/` must register a prompt that creates two or more documents and adds at least one relation in its message script. Reject PRs that add prompts violating the criteria above.
- **ADR gate:** adding a sixth prompt (or any non-track prompt) requires a new ADR amending `mcp/mcp-prompts-for-tracks-only.adr.md`. Reviewers MUST block the PR if the ADR is missing.
- **Lint check (manual):** during `/archcore:review`, audit `internal/mcp/prompts/` against the authorized set. Flag deviations.
- **Description sanity:** every prompt's `WithPromptDescription` must mention the cascade chain (e.g., "BRS → StRS → SyRS → SRS"). A description that reads like a single action ("Write a BRS document") is a smell.
