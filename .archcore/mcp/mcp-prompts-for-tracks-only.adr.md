---
title: "MCP prompts capability scoped to the five document tracks"
status: accepted
tags:
  - "mcp"
---

## Context

The `archcore mcp` server currently exposes only `tools` and `instructions`. Multi-document workflows (ISO 29148 cascade `brs → strs → syrs → srs`, Sources `mrd → brd → urd`, Product, Standard `adr → rule → guide`, Architecture `adr → spec → plan`) are described declaratively in `internal/mcp/server.go:116-134` but have no orchestrated entry point. Without the Claude Code Archcore plugin (`/archcore:iso-track`, `/archcore:sources-track`, etc.), users have no single-click way to launch a cascade — the agent must reconstruct the flow from instructions.

The MCP protocol natively supports a `prompts` capability — named, parameterized message templates discoverable by clients via `prompts/list` and invoked via `prompts/get`. Implementing it would give a portable, plugin-free entry point for orchestrated multi-document workflows in any prompts-aware MCP client (Claude Desktop, Claude Code, VS Code MCP, Zed, parts of Cursor/Cline). See `.archcore/mcp/track-cascade-invocation-via-mcp.idea.md` for the full design exploration.

The open question was scope: implement prompts only for tracks, or also for adjacent workflows (`capture_module`, `audit_health`, `tour`, per-type guided writers, migrations, diagnostics).

## Decision

We will add the MCP `prompts` capability to `archcore mcp` and ship **exactly five prompts**, all targeting multi-document tracks:

1. `iso_track` — `brs → strs → syrs → srs` (ISO 29148 cascade)
2. `sources_track` — `mrd → brd → urd`
3. `product_track` — `prd → plan`
4. `standard_track` — `adr → rule → guide`
5. `architecture_track` — `adr → spec → plan`

Implementation:

- Enable prompts in `internal/mcp/server.go` via `server.WithPromptCapabilities(true)`.
- New package `internal/mcp/prompts/` with one file per track and a thin registration helper.
- Each handler returns `*mcp.GetPromptResult` with a `PromptMessage` sequence: system framing + step-by-step instructions referencing `create_document`/`add_relation` and the required `implements` edges. Accept arguments such as `feature_name`, `scope`, `project_context`.
- Add a short `WORKFLOW PROMPTS` section to `mcpServerInstructions` listing the five prompts so the model can suggest them when the client supports prompts.

Out of scope for this decision (deliberately):

- No prompts for single-document workflows (`write_brs`, `write_adr`, etc.) — templates plus existing instructions are sufficient.
- No prompts for diagnostic or maintenance flows (`audit_health`, `actualize`, `find_duplicates`, `propose_relations`).
- No prompts for capture/migration flows (`capture_module`, `migrate_prd_to_iso`).
- No `tour`/`help` meta-prompts.
- MCP `resources` capability — not added.
- Tool response `suggested_next` hints and `CASCADE INVOCATION SIGNALS` instructions block — both still planned per the parent idea but tracked separately; not blocked by or coupled to this ADR.

## Alternatives Considered

**A. Don't implement prompts; rely only on instructions + tools.**
Rejected — instructions are passive and don't give a single-click entry point in client UIs. Users without the plugin would still need to phrase requests precisely to trigger a cascade. Discoverability stays poor.

**B. Implement prompts for the full plugin parity surface (tracks + capture + audit + tour + per-type writers).**
Rejected for now — large surface, much of it overlaps with plugin skills that already amplify with Claude Code-specific tools (Agent, TaskCreate, Read/Glob). Shipping all of them at once risks dilution and maintenance overhead with limited evidence of demand. Tracks are the highest-value subset because they involve multi-document orchestration that no single tool call covers.

**C. Implement only `iso_track` (the formal cascade) and skip lighter ones.**
Rejected — the five tracks share infrastructure (registration, message-template helpers, tests). Marginal cost of shipping all five vs one is low, and parity across tracks gives a coherent UX in the slash-menu.

**D. Add an orchestration tool (`run_track`) instead of prompts.**
Rejected — would violate the project's primitive-tool design (`.archcore/relations/no-auto-relations-on-create-document.adr.md` established the same principle for relations). Tools should remain CRUD primitives; orchestration belongs in the prompt layer where the protocol natively supports it.

**E. Move all canonical workflow definitions out of plugin skills into MCP prompts immediately.**
Rejected as one-shot — coordinated migration is for a later phase. This ADR establishes MCP prompts as the canonical source for the five tracks; plugin skill rework is a follow-up that can call `prompts/get` and add Claude Code-specific amplification.

## Consequences

**Positive:**
- Plugin-free users in any prompts-aware MCP client get a slash-menu entry point for the five tracks.
- The MCP server becomes the canonical source of cascade workflow definitions; plugin skills can later delegate to it instead of redefining.
- Bounded scope keeps the first prompt rollout small, focused, and easy to test/revert.
- Preserves the primitive-tool design — no new orchestration tools, no new document types.
- Sets a precedent: the `prompts` slot in MCP is reserved for multi-document orchestration, not for everything that could be a workflow.

**Negative / costs:**
- New code surface (`internal/mcp/prompts/` + capability flag + tests).
- `mcpServerInstructions` grows by a short `WORKFLOW PROMPTS` block — small token cost; coordinate with `.archcore/cli/mcp-token-optimization.idea.md`.
- Naming overlap with plugin skills (`/archcore:iso-track` vs `/mcp__archcore__iso_track`) — must be disambiguated in descriptions until plugin skills are refactored to delegate.
- Clients without prompts support (most headless agents, parts of Cursor/Continue) gain nothing from this work; they still depend on instructions and (future) `suggested_next` hints.

**Follow-ups (not part of this ADR):**
- `suggested_next` field in `create_document` responses (parent idea, layer 2).
- `CASCADE INVOCATION SIGNALS` block in instructions (parent idea, layer 3).
- Plugin skill refactor to delegate to MCP prompts (separate decision when prompts ship).
- Decisions about non-track prompts (capture, audit, tour, etc.) — explicitly deferred.

## Related

- `.archcore/mcp/track-cascade-invocation-via-mcp.idea.md` — parent idea with full layered design and rationale.
- `.archcore/mcp/mcp-server-starts-without-archcore-dir.adr.md` — prior decision aligning MCP to be self-sufficient without external setup.
- `.archcore/relations/no-auto-relations-on-create-document.adr.md` — same primitive-tool principle this ADR upholds.
- `.archcore/code-quality/in-process-mcp-integration-tests.adr.md` — test strategy that covers the new `prompts/list` and `prompts/get` handlers.
- `.archcore/cli/mcp-token-optimization.idea.md` — must coordinate to keep instruction additions cheap.

## Implementation Notes

Key files: @internal/mcp/server.go (capability flag + instruction block + prompt registration), @internal/mcp/prompts/ (new package), @internal/mcp/server_test.go (extend coverage).
