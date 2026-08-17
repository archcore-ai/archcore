---
title: "Plan Discharge by Deletion — a Completed Plan Leaves the Corpus, Not Its Status"
status: draft
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Context

Eighteen `plan` documents live in `.archcore/plugin/`; fifteen carry `status: rejected`, all written in one sweep on 2026-08-07 between 13:21 and 13:22, and every one of them still holds an `implements` or `depends_on` edge — the exact pair that `@plugins/archcore/skills/_shared/tracks/actualize.md` flags as temporal staleness, so drift detection returns fifteen findings that never resolve. The kernel supplies no other word: `@internal/mcp/server.go` defines `rejected` as "superseded, abandoned, or declined", and `@internal/mcp/tools/remove_document.go` instructs "A plan is abandoned → change status to rejected", leaving a completed plan with no status that describes it. `@plugins/archcore/skills/_shared/tracks/closeout.md` already specifies plan discharge into `task-type` or `guide` capture, but its Discharge report section blocks every transition on an `archived` value the kernel does not carry.

## Decision

A completed `plan` discharges by deletion across two gates: `closeout.capture` routes the plan's residue to the instrument that already owns that type — the actor rule for a procedure, the decision instrument's standard cascade for a settled standard — and `closeout.discharge` then calls `remove_document` under a per-document confirmation, blocked unless every plan task carries a `fulfilled` verdict from `closeout.verify` and the plan file is already committed.

## Alternatives Considered

1. The `archived` status value added to the kernel enum — rejected because it costs a CLI release, a version probe in the plugin, and skew handling across four MCP tool schemas plus `@internal/mcp/server.go` and the hook counters, to buy residual read access that a completed plan does not need; `remove_document` already ships and already clears both relation directions (`@internal/mcp/tools/remove_document.go`).
2. Keep writing `rejected` on completed plans — ruled out because the temporal rule in `@plugins/archcore/skills/_shared/tracks/actualize.md` reads `rejected` plus an active `implements` edge as staleness, which is what produces the current fifteen unresolvable findings.
3. Leave completed plans at `accepted` — ruled out because the three accepted plans stay in every grounding read as canon beside the `spec` they implement, while the content-kind ownership table in `@plugins/archcore/skills/_shared/prd-contract.md` assigns those statements to the spec.
4. One gate carrying its own type menu — `task-type`, `guide`, `rule`, `cpat` — ruled out because it builds a third parallel type menu beside the decision cascade and the experience offer, and it breaks the instrument-layer invariant that a producer owns one type; the decision instrument is already callable from `review` and already produces `adr` plus a `rule` and `guide` cascade.
5. Reverse the `closeout.accept` → experience ordering so the existing offer track extracts residue before disposal — deferred because `@plugins/archcore/skills/_shared/tracks/experience.md` states that the whole track is an offer, and a blocking prerequisite would contradict that contract.

## Consequences

- Removes 15 documents and 15 unresolvable temporal-drift findings at the first gate run (measured 2026-08-17: `list_documents(types=["plan"], status="rejected")` returned 15, each carrying an active edge in `.archcore/.sync-state.json`).
- Restores one meaning to `rejected`, so a `status: rejected` listing reads as a declined-proposal queue — 15 of the 27 rejected documents were completed plans. [expected]
- Ships in the plugin alone: no CLI release, no version probe, no amendment to the plugin/CLI compatibility contract.
- Splitting capture from disposal keeps each failure independent: a declined capture still allows removal, and an unfulfilled plan still blocks it.
- A discharged plan's Declared Delta and unplanned-Δ record survive only in git and stop answering `search_documents`; recovery costs a `git log --diff-filter=D --follow` lookup.
- Deletion carries no undo at the tool boundary — `@internal/mcp/tools/remove_document.go` sets `destructiveHint: true` and unlinks the file — so this gate carries two blocking preconditions that no other track gate carries. [expected]
- Routing residue into the decision instrument adds up to 4 questions to a closeout run against a 5-question per-invocation ceiling, so capture engages only on a residue the plan or the closeout report already named. [expected]
- The kernel keeps instructing "A plan is abandoned → change status to rejected", so an agent acting outside the closeout gate follows the old guidance until that description changes. [expected]

## Superseded when

- The kernel gains an `archived` value that grounding reads exclude — discharge then becomes a status transition and the two preconditions collapse to one confirmation.
- More than 2 recovery lookups of a deleted plan occur within one quarter, showing that the capture step under-extracts the residue (current: 0).
