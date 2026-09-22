---
title: "add_relation Reports Pair Overlaps as Non-Blocking Warnings"
status: accepted
tags:
  - "component:cli"
  - "mcp"
  - "relations"
---

## Context

The global research `relation-generation-quality.research` counted 1,966 unique relations in six corpora on 2026-09-21. Of them 1,519 are `related`; in this repository 254 of 353 are. The same snapshot holds 12 pairs with a `related` edge in both directions and 20 pairs with `related` beside another type.

The research classifies both shapes as review candidates, not defects. Its read examples R06 and R07 show a pair where two types can record distinct claims, and its synthesis forbids automatic deletion.

The relation policy in the tool description tells the agent not to add either shape solely for navigation. Until this decision `add_relation` validated structure and returned `added` only (`@cli/internal/mcp/tools/add_relation.go`), so the agent learned about an overlap only through an extra `list_relations` call. `tool-descriptions-fit-the-host-cap.adr` records that prompt text does not reach the model whole on every host.

Both shapes are a deterministic function of the manifest and the new edge.

## Decision

After a write, `add_relation` returns a `warnings` array that names each overlap the new edge takes part in — `reverse_related` and `related_beside_specific` — and the write always succeeds.

`Manifest.RelationOverlaps` computes the classification in `@cli/internal/sync/manifest.go`. The MCP boundary renders the message in `@cli/internal/mcp/tools/add_relation.go`. The response omits the `warnings` key when no overlap exists and when `added` is false.

## Alternatives Considered

1. Refuse the write — rejected because the research found justified pairs of both shapes. A refusal pushes the agent to drop a true claim or to pick a wrong type.
2. Prompt text alone — rejected as the sole measure because the text does not reach the model whole on every host, and it depends on the agent calling `list_relations` first.
3. Remove the overlapping edge automatically — rejected because the research policy sends an ambiguous edge to review, never to automatic deletion.
4. Report overlaps in `list_relations` or `archcore doctor` — deferred. A graph-wide audit is a separate surface; this decision covers the moment of the write.

## Consequences

Positive:

- The agent sees the overlap in the response to the write, with no extra call.
- The check does not depend on insertion order: `related` after a specific edge and a specific edge after `related` produce the same warning.
- The response shape is additive. `source`, `target`, `type`, and `added` are unchanged.
- The array holds at most two entries in a fixed order; both together are 418 bytes (measured 2026-09-21).
- No known consumer reads the response keys of `add_relation`. The plugin at commit `3c2d2c9` names the tool only in hook matchers (`plugins/archcore/hooks/hooks.json`, `plugins/archcore/hooks/codex.hooks.json`, `plugins/archcore/bin/lib/normalize-stdin.sh`), and the CLI hooks do not parse the response (checked 2026-09-22).

Negative:

- A justified overlap produces a warning each time one is written. The cost is one read of a short message.
- The change does not report overlaps that a manifest already holds.
- "Specific" means every type other than `related`, so `related` beside `supports`, `contradicts`, or `supersedes` warns too.
- [assumption] A consumer outside these two repositories ignores an unknown key.

## Superseded when

- A relation gains a stored justification, so the engine can tell a distinct claim from a navigation edge.
- A graph-wide audit surface reports the same overlaps, and the write-time warning proves redundant in transcripts.
