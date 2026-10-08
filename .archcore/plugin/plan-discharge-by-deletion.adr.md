---
title: "Plan Discharge by Deletion — a Completed Plan Leaves the Corpus, Not Its Status"
status: accepted
tags:
  - "architecture"
  - "component:plugin"
  - "plugin"
  - "skills"
---

## Context

Eighteen `plan` documents live in `.archcore/plugin/`; fifteen carry `status: rejected`, all written in one sweep on 2026-08-07 between 13:21 and 13:22, and every one of them still holds an `implements` or `depends_on` edge — the exact pair that `@plugin/plugins/archcore/skills/_shared/tracks/actualize.md` flags as temporal staleness, so drift detection returns fifteen findings that never resolve. The kernel supplies no other word: `@internal/mcp/server.go` defines `rejected` as "superseded, abandoned, or declined", and `@internal/mcp/tools/remove_document.go` instructs "A plan is abandoned → change status to rejected", leaving a completed plan with no status that describes it. `@plugin/plugins/archcore/skills/_shared/tracks/closeout.md` already specifies plan discharge into `task-type` or `guide` capture, but its Discharge report section blocks every transition on an `archived` value the kernel does not carry.

## Decision

A completed `plan` discharges by deletion across two gates. `closeout.capture` routes the plan's residue to the instrument that owns that type. `closeout.discharge` then calls `remove_document` under a per-document confirmation, after every plan task and acceptance criterion carries a `fulfilled` verdict.

On 2026-10-01, the user approved the plain-review entry described in this paragraph. A plain branch review also enters closeout for each local plan that covers the branch work, after checking changed code against documents and applying any confirmed drift fixes. The review checks every task and acceptance criterion; branch readiness alone does not establish completion. A plan with unfinished work or insufficient evidence stays in the corpus with the reason in the report. An explicit `review closeout` remains available.

On 2026-09-28, the user approved removing the committed-file prerequisite. A plan created or edited during implementation can now leave the corpus before the final commit. Completion verification uses the current scoped working tree, the branch diff, and available verification reports. Staged, unstaged, and untracked plans follow the same completion and confirmation checks. Closeout neither stages files nor creates commits.

Before requesting removal confirmation, closeout states whether git history preserves the plan's current content. The final report names each removed plan and each retained plan's remaining work or blocking reason.

## Alternatives Considered

1. The `archived` status value added to the kernel enum — rejected because it costs a CLI release, a version probe in the plugin, and skew handling across four MCP tool schemas plus `@internal/mcp/server.go` and the hook counters, to buy residual read access that a completed plan does not need; `remove_document` already ships and already clears both relation directions (`@internal/mcp/tools/remove_document.go`).
2. Keep writing `rejected` on completed plans — ruled out because the temporal rule in `@plugin/plugins/archcore/skills/_shared/tracks/actualize.md` reads `rejected` plus an active `implements` edge as staleness, which is what produces the current fifteen unresolvable findings.
3. Leave completed plans at `accepted` — ruled out because the three accepted plans stay in every grounding read as canon beside the `spec` they implement, while the content-kind ownership table in `@plugin/plugins/archcore/skills/_shared/prd-contract.md` assigns those statements to the spec.
4. One gate carrying its own type menu — `task-type`, `guide`, `rule`, `cpat` — ruled out because it builds a third parallel type menu beside the decision cascade and the experience offer, and it breaks the instrument-layer invariant that a producer owns one type; the decision instrument is already callable from `review` and already produces `adr` plus a `rule` and `guide` cascade.
5. Reverse the `closeout.accept` → experience ordering so the existing offer track extracts residue before disposal — deferred because `@plugin/plugins/archcore/skills/_shared/tracks/experience.md` states that the whole track is an offer, and a blocking prerequisite would contradict that contract.

## Consequences

- Removes 15 documents and 15 unresolvable temporal-drift findings at the first gate run (measured 2026-08-17: `list_documents(types=["plan"], status="rejected")` returned 15, each carrying an active edge in `.archcore/.sync-state.json`).
- Restores one meaning to `rejected`, so a `status: rejected` listing reads as a declined-proposal queue — 15 of the 27 rejected documents were completed plans. [expected]
- Ships in the plugin alone: no CLI release, no version probe, no amendment to the plugin/CLI compatibility contract.
- Splitting capture from disposal keeps each failure independent: a declined capture still allows removal, and an unfulfilled plan still blocks it.
- A plain branch review with a matching plan now runs the closeout checks and may require document and removal confirmations. A review with no matching plan keeps its branch-review result.
- A discharged plan stops answering `search_documents`. A committed version remains recoverable from git. Uncommitted edits and a never-committed plan have no recovery copy in git; removal loses that version.
- Deletion carries no undo at the tool boundary: `@cli/internal/mcp/tools/remove_document.go` sets `destructiveHint: true` and unlinks the file. The completion checks and per-document confirmation remain; the committed-file prerequisite was removed on 2026-09-28.
- Routing residue into the decision instrument added up to 4 questions to a closeout run against a 5-question per-invocation ceiling. Since 2026-10-09, `closeout.capture` runs the durable-context selection and asks one batched preview question for all its writes (`durable-context-selection-by-reader-task.adr`). [expected]
- The kernel keeps instructing "A plan is abandoned → change status to rejected", so an agent acting outside the closeout gate follows the old guidance until that description changes. [expected]

## Superseded when

- The kernel gains an `archived` value that grounding reads exclude, providing a retained copy instead of deletion.
- More than 2 recovery lookups of a deleted plan occur within one quarter, showing that the capture step under-extracts the residue (current: 0).
