---
title: "Actualize System for Documentation Freshness Detection"
status: accepted
tags:
  - "architecture"
  - "component:plugin"
  - "hooks"
  - "plugin"
  - "skills"
---

## Context

The plugin validated the structure of `.archcore/` documents (PreToolUse blocking, PostToolUse validation through `archcore doctor`) but nothing detected when document content became stale. Three kinds of staleness went unnoticed:

- **Cascade.** A is updated while the documents that implement, depend on, or extend A stay unchanged.
- **Code drift.** Source changes invalidate the assumptions, APIs, or patterns a document describes.
- **Temporal.** A document sits in one status far longer than expected: a long-lived draft, a plan past its deadline, or a rejected document still referenced in an active chain.

Staleness surfaced only in a manual audit or when someone read an outdated document.

## Decision

Add a three-layer detection system that surfaces staleness at increasing depth, built on the existing hook, skill, and relation-graph infrastructure. All detection is informational. It never blocks and never modifies a document without confirmation.

- **Layer 1, passive, at SessionStart.** The CLI session-start recap carries a git-based check that compares code changes since the last `.archcore/` modification against document references (`cli-owns-layers-4-5.adr`). It runs inside the SessionStart budget and degrades to a skip when git is unavailable.
- **Layer 2, reactive, at PostToolUse.** After `update_document` succeeds, the CLI post-tool-use handler warns about documents that point at the updated one through `implements`, `depends_on`, or `extends`. `related` is excluded because it is too loose and would add noise.
- **Layer 3, deep, on demand.** `/archcore:review` with staleness wording (for example `/archcore:review are docs out of date`) runs three dimensions together: code-to-doc drift, doc-to-doc cascade, and temporal staleness (drafts older than 30 days, accepted plans past their deadline, rejected documents in active chains). It offers assisted fixes through `update_document`, each behind user confirmation.

The layers follow `four-command-palette.adr`: Layer 3 ships as a mode of `review`, not as a separate `/archcore:actualize` skill.

**Naming.** The system is called Actualize. `refresh` implies reloading data rather than analyzing content. `sync` implies bidirectional synchronization and would mislead. `update` collides with the `update_document` tool name. `actualize` means to make current and relevant, which is the intent.

## Alternatives Considered

1. **Only enhance the existing audit skill's dashboard and `--deep` modes** — rejected because the audit is on-demand, with no passive or reactive detection. Layer 3 was later folded into the review skill as staleness wording, while Layers 1 and 2 stayed separate.
2. **Background polling through a `/loop` command** — rejected because it needs a persistent open terminal session, wastes resources, and stops when the user closes the terminal.
3. **A dedicated `check_staleness` MCP command in the CLI** — deferred, not rejected. It would help cross-tool portability. The hook can delegate to it if the CLI adds it.
4. **A git pre-commit hook outside the host** — rejected on timing. By commit time the user has already worked from potentially stale documents, and session start is when they can act.

## Consequences

- Staleness surfaces at session start and right after a document update, so drift is not silent. Deep analysis is available on demand.
- Tradeoff: Layer 1 and the code-drift dimension of Layer 3 depend on git history, so both skip when git is unavailable or `.archcore/` is uncommitted.
- Tradeoff: file-path matching in Layer 1 gives heuristic false positives, for example a document that references a directory where unrelated files changed. Accepted, because a false positive is an informational nudge.

## Superseded when

- Measured false-positive rates from Layer 1 path matching make its nudges unread, which calls for a more precise reference-extraction mechanism.
