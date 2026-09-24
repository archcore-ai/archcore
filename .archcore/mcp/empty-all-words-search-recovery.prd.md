---
title: "Recover the Governing Decision After an Empty All-Words Search"
status: draft
tags:
  - "component:cli"
  - "mcp"
---

## Vision

An agent that searches `.archcore/` with several words reaches the governing decision even when one word of its query is absent from that decision, and never reads a word mismatch as "no decision exists".

## Problem Statement

An agent that calls `search_documents` with a multi-word query in the default all-words mode gets no rows whenever one query word is absent from the relevant document, and the response calls this a verified absence. The agent then concludes that no decision governs the code. In integration-bench experiment `20260923T153932Z-98ef5bbd20`, one agent recommended deleting code that an accepted ADR requires after such an empty result. The cost falls on the maintainers who review the agent's change and on every MCP client that trusts the absence claim.

## Goals and Success Metrics

- In a replay of the failing bench conversation with the new response, at least 8 of 10 samples open the governing ADR before recommending a change.
- `BenchmarkReadToolsScaling` at N=10,000 adds less than 15% time to an empty two-word query, measured against the release before the change.
- Real sessions: the share of empty multi-word all-words queries that end without another search is tracked, with no target, because about 6 such events occur per month on the maintainer's machine.

## Requirements

1. An agent whose all-words search returns no rows learns, in the same response, which documents came close and which query words they lack.
2. An agent tells "nothing came close" apart from "the CLI predates this feature".
3. The absence claim in the tool text and in the contract matches what the search checked, including the active filters.
4. Clients and agents that read today's responses keep working without change.
5. The tool stays a matching primitive: the addition carries match data, not guidance text.

## Out of Scope

- Agents that never call `search_documents`; in the bench they were 3 of the 4 runs that deleted decided code, and recipes cover them.
- Vocabulary gaps where every query word misses.
- `path_ref` searches with no rows.

## Clarifications

- 2026-09-23, maintainer: the design follows the RFC on near misses — the field name `near_misses`, the half-of-the-words rule, the `[]` presence rule, the addendum to the matching-primitive ADR, and a trigger from two words.
- 2026-09-24, maintainer: the replay target is at least 8 of 10; the real-session share is tracked without a target; the benchmark budget is less than 15%; the wording correction and the near-miss data ship in one release.