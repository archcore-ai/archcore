---
title: "Actualize System for Documentation Freshness Detection"
status: accepted
tags:
  - "architecture"
  - "hooks"
  - "plugin"
  - "skills"
---

## Context

The Archcore Claude Plugin has a comprehensive validation system (PreToolUse blocking, PostToolUse validation via `archcore validate`) that ensures **structural integrity** of `.archcore/` documents. The `/archcore:review` skill provides on-demand health checks for coverage gaps, relation health, and status issues.

However, no mechanism detects when documentation **content becomes stale**. Three types of staleness go undetected:

1. **Document cascade**: Updating document A while documents that `implements`, `depends_on`, or `extends` A remain unchanged. Example: PRD is rewritten but the plan that `implements` it still describes the old scope.
2. **Code-document drift**: Source code changes that invalidate assumptions, APIs, or patterns described in archcore documents. Example: auth module is refactored but the ADR still describes the old JWT strategy.
3. **Temporal staleness**: Documents stuck in inappropriate statuses — long-lived drafts, plans with passed deadlines, rejected documents still referenced in active chains.

Currently, staleness is only discovered during manual `/archcore:review` invocations or when someone happens to read an outdated document. There is no proactive or reactive detection.

## Decision

Add a 3-layer Actualize system that detects documentation staleness at increasing depths, building on the existing hook and skill infrastructure.

### Layer 1 — Passive Detection (SessionStart hook enhancement)

Extend `bin/session-start` to run a lightweight git-based staleness check at every session start. Compare code file changes since the last `.archcore/` modification. If significant drift detected, inject a brief warning into session context (max 2KB, within the 10KB `additionalContext` budget).

**Mechanism**: `git log` to find last `.archcore/` commit, `git diff --name-only` to find code changes since, `grep -rl` in `.archcore/` to find documents referencing changed paths.

**Character**: Informational only. Never blocks session start, never modifies documents.

### Layer 2 — Reactive Cascade Detection (PostToolUse hook)

Add cascade detection logic to the PostToolUse hook pipeline. After `mcp__archcore__update_document`, query the relation graph for documents connected to the updated document via `implements`, `depends_on`, or `extends` relations (where the updated document is the target). Inject a cascade warning naming potentially affected documents.

**Mechanism**: Parse updated document path from tool input, query `archcore` CLI for relations, filter by directional relation types.

**Character**: Informational only. Injected as `additionalContext` after the update succeeds. Never blocks operations.

### Layer 3 — Deep Analysis (/archcore:actualize intent skill)

Add an 8th Layer 1 intent skill that performs comprehensive staleness analysis on demand. Three detection dimensions:

- **Code→Doc drift**: For each document, find referenced code paths in content, check `git log` for changes since document was last modified.
- **Doc→Doc cascade**: For recently updated documents, traverse the relation graph to find stale dependents.
- **Temporal**: Draft documents older than 30 days, accepted plans with past deadlines, rejected documents still in active relation chains.

Offers interactive assisted fixes via MCP `update_document`.

### Naming

The skill is named `actualize` because:
- `refresh` implies reloading data, not content analysis
- `sync` implies bidirectional synchronization (misleading)
- `update` conflicts with `update_document` MCP tool name
- `actualize` means "make current and relevant" — exactly the intent

### Detection direction for cascade (Layer 2)

When document A is updated, which documents may be stale?

| Relation direction | Relation type | Stale document | Reason |
|---|---|---|---|
| B `implements` A | implements | B | B's implementation of A may be outdated |
| B `depends_on` A | depends_on | B | B's assumptions about A may be invalid |
| B `extends` A | extends | B | B's extensions of A may be incompatible |

`related` relations are excluded — too loose, would create noise.

### Scope constraints

- Layer 1 (passive): read-only, max 2KB output, max 3s execution
- Layer 2 (reactive): read-only, within PostToolUse 3s budget, fires only on `update_document`
- Layer 3 (deep): interactive, can modify via MCP with user confirmation, no time limit

## Alternatives Considered

### Only enhance /archcore:review

Add staleness checks to the existing review skill. **Rejected**: review is on-demand only — no passive or reactive detection. Users must remember to run it. The actualize system catches staleness automatically.

### Background polling via /loop

Use Claude Code's `/loop` command to periodically check for staleness. **Rejected**: requires a persistent session open in terminal, wasteful of resources, and stops when the user closes the terminal.

### MCP-side staleness check

Add `check_staleness` command to the archcore CLI itself. **Deferred, not rejected**: good for cross-tool portability (works in any IDE plugin, not just Claude Code). The plugin-side implementation delivers value now without CLI changes. If the CLI adds this later, the hook can delegate to it.

### Git pre-commit hook (outside Claude Code)

Check staleness before git commits. **Rejected**: wrong timing — by the time the user commits, they've already been working with potentially stale docs. Session-start is when they can act.

## Consequences

### Positive

- Staleness detected at session start — before the user makes decisions based on outdated documentation
- Cascade effects surfaced immediately after document updates — no silent drift
- Deep analysis available on demand for thorough cleanup sessions
- Builds entirely on existing infrastructure (hooks, bin scripts, MCP tools, relation graph)
- All detection is transparent and informational — never destructive, never blocking

### Negative

- **Git dependency**: Layer 1 and Layer 3 code-doc drift require git history. Must degrade gracefully if git is unavailable or `.archcore/` is not committed.
- **Heuristic false positives**: File-path matching in Layer 1 may flag documents that reference a directory where unrelated files changed. Acceptable tradeoff — false positives are informational nudges, not errors.
- **SessionStart latency**: Layer 1 adds ~1-2 seconds to session startup for git operations. Acceptable — already within the 3-second budget and user expects startup to load context.
- **PostToolUse overhead**: Layer 2 adds relation graph query after every `update_document`. Must complete within existing 3-second timeout.
- **Skill count increase**: Layer 1 intent skills go from 7 to 8. Requires updating plugin-architecture.spec, skills-system.spec, and commands-system.spec constraints.