---
title: "Track/Cascade Invocation for Grouped Documents via MCP (Plugin-Free)"
status: accepted
tags:
  - "mcp"
---

## Idea

Give users a way to invoke multi-document tracks (ISO 29148 cascade `brs → strs → syrs → srs`, Sources track `mrd → brd → urd`, Product track `prd → plan`, Standard track `adr → rule → guide`, Architecture track `adr → spec → plan`) directly through the `archcore mcp` server — without requiring the Claude Code Archcore plugin and without adding orchestration tools or new document types.

Current state: `internal/mcp/server.go` already describes the three requirements tracks and ISO cascade in `mcpServerInstructions`, but there is no explicit entry point for "run this track end-to-end". The agent only triggers cascade behavior if the user explicitly says "ISO 29148" or uses strong semantic cues (ConOps, verification matrix, per-stakeholder class). Lightweight MRD/PRD prompts default to the Product track. Plugin skills like `/archcore:iso-track` fill this gap, but users without the plugin lose the orchestrated flow.

Proposed layered approach, keeping `create_document` / `update_document` / `add_relation` strictly as CRUD primitives and the type list flat:

1. **MCP `prompts` capability** — expose named prompts (`iso_track`, `sources_track`, `product_track`, `standard_track`, `architecture_track`) via the native MCP `prompts/list` + `prompts/get` protocol methods. Each prompt returns a scripted message sequence that walks the agent through the cascade, asking for user confirmation between steps and linking documents via `implements`/`depends_on`. Implemented in `mark3labs/mcp-go` via `server.WithPromptCapabilities(true)` and `s.AddPrompt(...)`.
2. **Tool response hints** — extend `create_document` response with a `suggested_next` field for track-member types (e.g., after creating a `brs`, suggest `{type: "strs", relation: "implements", target: <brs path>, reason: "ISO cascade: StRS decomposes BRS"}`). Works in any MCP client because it rides on existing tool responses.
3. **Signals section in MCP instructions** — add an explicit "CASCADE INVOCATION SIGNALS" block listing trigger keywords per track (regulated/compliance/ConOps/verification → ISO; personas/journeys → Sources; team standard → Standard). Pure prompt engineering, no code on the wire.

## Value

- Removes plugin-only advantage — `archcore mcp` becomes self-sufficient for orchestrated workflows in Claude Desktop, Claude Code, VS Code MCP, Cursor, Cline, Continue, and any other client that reads server instructions.
- Preserves the primitive-tool design: no `run_track` / `create_cascade` mega-tool, no composite types.
- Makes it discoverable: today a user has to know the phrase "ISO 29148" to reach the cascade; named prompts put the tracks into the client's slash-menu.
- Reuses work already done: requirement tracks, relation semantics, and type definitions already exist in `internal/mcp/server.go` and `internal/mcp/tools/create_document.go`.

## Possible Implementation

- **Prompts layer (primary)**
  - Register prompts in `internal/mcp/server.go` alongside existing tool registration, using `server.WithPromptCapabilities(true)`.
  - New package `internal/mcp/prompts/` with one file per track (`iso_track.go`, `sources_track.go`, `product_track.go`, `standard_track.go`, `architecture_track.go`).
  - Each handler returns `*mcp.GetPromptResult` with `PromptMessage` list: system framing + step-by-step user messages referencing `create_document`/`add_relation` calls and the required `implements` edges. Accept arguments like `feature_name`, `scope`, `project_context` for templating.
  - Content sourced from the existing `working-with-requirements-tracks.guide.md` and track-specific plugin skill definitions (mirror, don't duplicate conceptually).

- **Response hints layer**
  - Extend the JSON result returned from `create_document` with an optional `suggested_next` object when the created type participates in a known track. Mapping defined in one place (e.g., `internal/mcp/tools/track_hints.go`).
  - Examples: `brs → strs`, `strs → syrs`, `syrs → srs`, `mrd → brd`, `brd → urd`, `urd → brs`, `adr → rule`, `rule → guide`.

- **Instructions layer**
  - Append a `CASCADE INVOCATION SIGNALS` section to `mcpServerInstructions` in `internal/mcp/server.go`, listing explicit keywords per track, with a short rule like: "When user mentions any signal for track X, propose running the X cascade; confirm before creating the first document."

- **Documentation**
  - Add a guide (`using-tracks-via-mcp.guide.md`) showing how to invoke each track in plugin-free clients — screenshots of prompt discovery in Claude Desktop and Cursor slash-menus.

- **Testing**
  - In-process MCP integration tests (per `in-process-mcp-integration-tests.adr.md`) for `prompts/list` and `prompts/get` handlers.
  - Unit tests for `suggested_next` mapping per input type.

## Risks and Constraints

- **Uneven `prompts` client support** — tools are universally supported; prompts are not. Claude Desktop, Claude Code, VS Code MCP, Zed support prompts; Cursor and Continue are partial; headless agents built on raw SDKs often ignore `prompts/*`. Mitigation: the response-hint + instructions layers give cascade nudges to 100% of clients, prompts are additive UX for the richer ones.
- **Drift from plugin skills** — if plugin skills and MCP prompts diverge, users get different cascade flows depending on entry point. Mitigation: treat MCP prompts as the canonical source; plugin skills call them (or mirror the same text).
- **Response-hint noise** — adding `suggested_next` everywhere may encourage over-linking. Mitigation: emit the hint only when the created type has a clear downstream counterpart in a track.
- **Instruction bloat** — `mcpServerInstructions` is already long; adding another section has a token cost. Pair this work with `mcp-token-optimization.idea.md` if shipped together.
- **Implicit confirmation UX in prompts** — a single `prompts/get` call returns a message sequence, but a true multi-step cascade with user confirmations depends on the client replaying those messages as a conversation. Not all clients do this identically. Mitigation: keep prompts as "opinionated starting instruction + first step", rely on the model + response hints to drive subsequent steps.

## Related

- `.archcore/mcp/mcp-server-starts-without-archcore-dir.adr.md` — same philosophy: MCP must work standalone.
- `.archcore/document-types/prd-vs-iso-29148-requirements-strategy.idea.md` — defines the three tracks this idea surfaces.
- `.archcore/document-types/mrd-brd-urd-requirement-sources.idea.md` — Sources track.
- `.archcore/requirements/working-with-requirements-tracks.guide.md` — human-facing track guidance that prompt content should mirror.
- `.archcore/cli/mcp-token-optimization.idea.md` — coordinate to avoid instruction bloat.
- `.archcore/knowledge/in-process-mcp-integration-tests.adr.md` — test strategy for the new prompts handlers.
