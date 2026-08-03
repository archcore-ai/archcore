---
title: "Actualize System for Documentation Freshness Detection"
status: accepted
tags:
  - "architecture"
  - "hooks"
  - "plugin"
  - "skills"
---

**Outcome (2026-05-15).** The decision was accepted as recorded, but Layer 3 shipped as the `--drift` mode of `/archcore:audit` rather than as a standalone `/archcore:actualize` intent skill, per `skill-surface-collapse.adr`. The three-layer architecture, the detection dimensions, and the naming rationale below remain authoritative; read `/archcore:audit --drift` wherever the original text said `/archcore:actualize`.

## Context

The plugin already had a validation system — PreToolUse blocking and PostToolUse validation through `archcore doctor` — that ensured the **structural integrity** of `.archcore/` documents, and an audit skill that provided on-demand health checks for coverage gaps, relation health, and status issues. Nothing detected when documentation **content** became stale, so three kinds of staleness went unnoticed: a document cascade, where A is updated while the documents that implement, depend on, or extend it stay unchanged; code-document drift, where source changes invalidate the assumptions, APIs, or patterns a document describes; and temporal staleness, where a document sits in one status far longer than expected — a long-lived draft, a plan past its deadline, or a rejected document still referenced in an active chain. Staleness was discovered only during a manual audit or when someone happened to read an outdated document, with no proactive or reactive detection at all.

## Decision

Add a three-layer detection system that surfaces documentation staleness at increasing depths, built entirely on the existing hook, skill, and relation-graph infrastructure.

**Layer 1 — passive detection, at SessionStart.** `bin/session-start` runs a lightweight git-based check at every session start, comparing code changes since the last `.archcore/` modification and injecting a brief warning into session context, capped at 2 KB inside the 10 KB additional-context budget. The mechanism is `git log` for the last `.archcore/` commit, `git diff --name-only` for the code changes since, and `grep -rl` inside `.archcore/` for documents referencing a changed path. It is informational only: it never blocks session start and never modifies a document.

**Layer 2 — reactive cascade detection, at PostToolUse.** After `update_document` succeeds, the hook queries the relation graph for documents connected to the updated one through `implements`, `depends_on`, or `extends`, where the updated document is the target, and injects a warning naming the potentially affected documents. The mechanism parses the updated path from the tool input and filters `.sync-state.json` by directional relation type. It is informational only and never blocks.

| Relation direction | Relation type | Stale document | Reason |
|---|---|---|---|
| B `implements` A | implements | B | B's implementation of A may be outdated |
| B `depends_on` A | depends_on | B | B's assumptions about A may be invalid |
| B `extends` A | extends | B | B's extensions of A may be incompatible |

`related` relations are excluded, because they are too loose and would create noise.

**Layer 3 — deep analysis, on demand.** Three detection dimensions run together: code-to-doc drift, finding referenced code paths in each document's content and checking `git log` for changes since that document was last modified; doc-to-doc cascade, traversing the relation graph from recently updated documents to find stale dependents; and temporal, flagging drafts older than 30 days, accepted plans past their deadlines, and rejected documents still in active relation chains. It offers interactive assisted fixes through `update_document`.

**Naming.** The system is called Actualize because `refresh` implies reloading data rather than analyzing content, `sync` implies bidirectional synchronization and would mislead, `update` collides with the `update_document` tool name, and `actualize` means to make current and relevant, which is exactly the intent.

**Scope constraints.** Layer 1 is read-only with at most 2 KB of output and 3 seconds of execution. Layer 2 is read-only, fits the 3-second PostToolUse budget, and fires only on `update_document`. Layer 3 is interactive, may modify through MCP with user confirmation, and carries no time limit.

## Alternatives Considered

1. **Only enhance the existing audit skill's dashboard and `--deep` modes** — initially rejected because the audit was on-demand only, with no passive or reactive detection, whereas this system catches staleness automatically through hooks. Partially reconsidered later: Layer 3 was folded into `/archcore:audit --drift` per `skill-surface-collapse.adr`, while Layers 1 and 2 remained separate as designed.
2. **Background polling through a `/loop` command** — rejected because it requires a persistent session open in a terminal, wastes resources, and stops when the user closes the terminal.
3. **An MCP-side staleness check, adding a `check_staleness` command to the CLI** — deferred rather than rejected, because it would be good for cross-tool portability, while the plugin-side implementation delivers value now without CLI changes; if the CLI adds it later, the hook can delegate.
4. **A git pre-commit hook outside the host** — rejected on timing: by the time a user commits, they have already been working from potentially stale documents, whereas session start is when they can act.

## Consequences

- Staleness surfaces at session start, before the user makes decisions on outdated documentation.
- Cascade effects surface immediately after a document update, so drift is not silent.
- Deep analysis is available on demand for a thorough cleanup session.
- The system builds entirely on existing infrastructure — hooks, bin scripts, MCP tools, and the relation graph.
- All detection is transparent and informational, never destructive and never blocking.
- Tradeoff: Layer 1 and the code-drift dimension of Layer 3 depend on git history, so both must degrade to a skip when git is unavailable or `.archcore/` is uncommitted.
- Tradeoff: file-path matching in Layer 1 produces heuristic false positives, flagging a document that references a directory where unrelated files changed. Accepted, because a false positive is an informational nudge rather than an error.
- Tradeoff: Layer 1 adds roughly 1–2 seconds to session startup for its git operations, inside the 3-second budget and inside what a user expects context loading to cost.
- Tradeoff: Layer 2 adds a relation-graph query after every `update_document`, which must complete inside the existing 3-second timeout.
- The skill-surface cost recorded at the time — taking the intent skills from 7 to 8 — was later eliminated by folding Layer 3 into the existing `audit` skill as `--drift`.
- The standalone `/archcore:actualize` skill described here is superseded by `skill-surface-collapse.adr`. Layers 1 and 2 remain as designed.

## Superseded when

- The CLI ships a `check_staleness` command, which would let both hooks delegate detection and make alternative 3 the implementation.
- Measured false-positive rates from Layer 1's path matching make its nudges unread, which would call for a more precise reference-extraction mechanism.
