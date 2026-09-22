---
title: "Remove the MCP Track Prompts and the Track Sections from Server Instructions"
status: accepted
tags:
  - "document-types"
  - "mcp"
---

## Context

Track orchestration had two owners. The CLI carried five MCP prompts
(`iso_track`, `sources_track`, `product_track`, `standard_track`,
`architecture_track`) plus three sections of `mcpServerInstructions` describing
the same flows: `REQUIREMENTS TRACKS`, `RESEARCH GATE (rnd)`, and
`WORKFLOW PROMPTS`. The plugin describes those flows too, in its skills.

Neither copy was canonical, and they had already started to diverge. A prompt's
per-phase gate is one fixed sentence of Go string, so its elicitation quality is
capped where the plugin's is not — the CLI copy could never win, but it could
still contradict.

The layer boundary that resolves this is recorded in the plugin ADR
"CLI Owns Layers 4–5": the CLI owns document types, templates, validation,
retrieval signals, and host guardrails; track and interview logic belongs to the
plugin.

## Decision

Delete `internal/mcp/prompts/` and stop declaring the prompt capability. Cut the
three track sections from the server instructions.

Keep everything that is knowledge about document TYPES rather than about a
workflow: `TYPE SELECTION RULES` (all 22), `REQUIREMENTS LAYERS`,
`DOCUMENT RELATIONS`, `TAGS`, and `VALID STATUS VALUES`.

Two fragments of the deleted `RESEARCH GATE` section carry type knowledge, not
workflow, so they move rather than disappear:

- the rnd verdict mapping (draft = investigating, accepted = proceed/refine,
  rejected = defer/stop) and the rule that a rejected rnd is a first-class
  outcome move into `VALID STATUS VALUES`;
- the rnd relation conventions move into `DOCUMENT RELATIONS`.

Remove the capability declaration and the registration call together. `AddPrompt`
turns the capability back on implicitly, so removing only one leaves the server
advertising a surface it no longer serves.

## Alternatives Considered

1. Keep the prompts and let the plugin defer to them — rejected because the
   fixed one-sentence gate caps elicitation quality below what a skill achieves,
   so the CLI copy would stay the weaker of two owners.
2. Keep the instruction sections and delete only the prompts — rejected because
   the sections describe the same flows in prose; two owners survive, only less
   visibly.
3. Split the release so the sections go now and the package later — considered
   and dropped by the maintainer. One release, with the window accepted.

## Consequences

- Layer 2 has one owner. No drift between an MCP prompt and the skill that
  duplicates it.
- `prompts/list` answers with a JSON-RPC method-not-found rather than an empty
  list, because the capability is undeclared. A client that never asks is
  unaffected.
- A CLI-only user loses the five prompts. The flow-guidance floor becomes the
  type-selection rules in the server instructions.
- The instructions lose 1907 bytes, about 12 percent. The bulk stays: type
  selection alone is roughly a third of the text.
- `examples/` needs no regeneration — the pinned fixture is the managed
  instruction block, which never named a prompt or a track.

## Superseded when

- Track orchestration becomes expressible in a host-portable MCP primitive that
  subagents can reach, making a CLI-side owner useful again.
