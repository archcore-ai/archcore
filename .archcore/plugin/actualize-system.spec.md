---
title: "Actualize System Specification (now /archcore:audit --drift)"
status: accepted
tags:
  - "hooks"
  - "plugin"
  - "skills"
  - "validation"
---

> **Outcome (2026-05-15):** The Actualize system shipped, but as the `--drift` mode of the unified `/archcore:audit` skill rather than as a standalone `/archcore:actualize` skill. Layer 1 and Layer 2 (the `bin/check-staleness` SessionStart hook and the `bin/check-cascade` PostToolUse hook) shipped as designed. Layer 3 was folded into `audit` per `skill-surface-collapse.adr.md`. The detection protocol now lives at `skills/audit/lib/drift-detection.md`. This spec is preserved for historical context; the active contract is the `--drift` mode in `commands-system.spec.md` and `plugin-architecture.spec.md`.

## Purpose

Define the contract for the Actualize system — a 3-layer documentation freshness detection mechanism that identifies stale `.archcore/` documents through passive session-start checks, reactive cascade detection after document updates, and deep on-demand analysis. Layer 3, originally planned as a standalone `/archcore:actualize` skill, ships as the `--drift` mode of `/archcore:audit`.

## Scope

This specification covers: the SessionStart staleness check (Layer 1), the PostToolUse cascade detection (Layer 2), and the deep analysis mode of `/archcore:audit` (Layer 3, originally Layer 3 of the now-merged Actualize skill). It defines their triggers, detection logic, output formats, and interaction with existing hooks and MCP tools.

It does not cover: structural validation (`archcore doctor`), the dashboard or `--deep` modes of `/archcore:audit`, or the archcore-auditor agent. Those are complementary but separate.

## Authority

This specification is the authoritative reference for staleness detection behavior in the plugin. The Actualize System ADR provides the architectural rationale. `skill-surface-collapse.adr.md` superseded the standalone Layer 3 skill in favor of an `--drift` mode on the merged `audit` intent. The Hooks Validation System Specification defines the hook execution model this system extends.

## Subject

The Actualize system detects three types of documentation staleness:

1. **Code→Doc drift** — source code changes that invalidate documentation content
2. **Doc→Doc cascade** — document updates that make related documents stale
3. **Temporal staleness** — documents stuck in inappropriate statuses over time

Detection operates at three depths:

```
┌─────────────────────────────────────────────────────────┐
│  Layer 1: Passive Detection                              │
│  Trigger: SessionStart                                   │
│  Depth: git diff heuristic                               │
│  Output: Brief warning in session context                │
│  Cost: ~1-2s at session start                            │
├─────────────────────────────────────────────────────────┤
│  Layer 2: Reactive Cascade                               │
│  Trigger: PostToolUse (update_document)                  │
│  Depth: Relation graph traversal                         │
│  Output: Cascade warning in additionalContext            │
│  Cost: <1s after each update                             │
├─────────────────────────────────────────────────────────┤
│  Layer 3: Deep Analysis                                  │
│  Trigger: /archcore:audit --drift (user-invoked)         │
│  Depth: Full code↔doc cross-reference + relation graph   │
│  Output: Actionable report + interactive fixes           │
│  Cost: 10-30s depending on project size                  │
└─────────────────────────────────────────────────────────┘
```

## Contract Surface

### Layer 1: Passive Detection (SessionStart Enhancement)

#### Trigger

SessionStart hook, executed as part of the existing `bin/session-start` pipeline. Runs after the CLI availability check and project context loading.

#### Handler

`bin/check-staleness`, called from `bin/session-start` after the normal context loading succeeds.

#### Detection Logic

```
1. LAST_DOC_COMMIT = git log -1 --format=%H -- .archcore/
2. If no commit found → skip
3. CHANGED_FILES = git diff --name-only $LAST_DOC_COMMIT..HEAD -- ':(exclude).archcore/'
4. If CHANGED_FILES is empty → skip
5. For each .archcore/**/*.md document, match directory references against CHANGED_FILES
6. Output warning with AFFECTED documents
```

#### Output Format

```
[Archcore Staleness] {N} source files changed since last documentation update.
Potentially affected documents:
  - {doc-path} — references {dir/} ({M} files changed)
Run /archcore:audit --drift for detailed analysis.
```

Output is plain text, injected as SessionStart additional context, rate-limited to once per 24 hours.

#### Constraints

- Must complete within 3 seconds
- Output must not exceed 2KB
- Must degrade gracefully when git is unavailable or `.archcore/` has no commits
- Must not block session start — always exit 0
- POSIX shell compatible

### Layer 2: Reactive Cascade Detection (PostToolUse Enhancement)

#### Trigger

PostToolUse hook, fires after `mcp__archcore__update_document` succeeds. Does NOT fire on `create_document` or `remove_document`.

#### Handler

`bin/check-cascade`, registered as a PostToolUse hook entry across all hosts (`hooks/hooks.json`, `hooks/cursor.hooks.json`, `hooks/codex.hooks.json`).

#### Detection Logic

```
1. Parse tool_input from stdin JSON
2. Extract updated document path
3. Query .archcore/.sync-state.json for relations where target = updated path
   and type ∈ {implements, depends_on, extends}
4. AFFECTED = source documents from filtered relations
5. If empty → exit 0; otherwise output cascade warning
```

#### Output Format

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "[Archcore Cascade] Updated \"{document-title}\".\nDocuments that may need review:\n  → {path} ({relation-type} this document)\nRun /archcore:audit --drift for detailed analysis."
  }
}
```

#### Relation Direction Table

| Relation in graph | Updated doc role | Potentially stale doc | Why |
|---|---|---|---|
| B `implements` A | A (target) | B (source) | B implements changed specification |
| B `depends_on` A | A (target) | B (source) | B depends on changed dependency |
| B `extends` A | A (target) | B (source) | B extends changed base |

`related` relations are excluded to reduce noise.

#### Constraints

- Must complete within 3 seconds (PostToolUse timeout)
- Fires only on `update_document`
- Must not block the operation — always exit 0
- POSIX shell compatible

### Layer 3: Deep Analysis — /archcore:audit --drift

#### Classification

Mode of the `/archcore:audit` intent skill. Activated by the `--drift` flag or by drift-related phrasing ("are any docs stale?", "check if docs match code"). The skill is auto-invocable per `skill-surface-collapse.adr.md`.

#### Routing Table (for the `audit` skill)

| Signal | Mode | Scope |
|---|---|---|
| No arguments | → short dashboard | All documents |
| `--deep` | → deep audit | All documents |
| `--deep <filter>` or `<filter>` (non-flag arg) | → deep audit, filtered | Filter applied |
| `--drift` | → drift detection | All documents |
| `--drift <filter>` | → drift detection, filtered | Filter applied |

Drift mode loads `skills/audit/lib/drift-detection.md` for the detailed protocol.

#### Execution Flow (drift mode)

1. **Gather**: `list_documents` (with optional filters), `list_relations`, `git log`.
2. **Analyze — Code→Doc drift**: cross-reference document content against `git diff --name-only` since the document was last modified.
3. **Analyze — Doc→Doc cascade**: walk the relation graph; flag sources whose targets were modified after them.
4. **Analyze — Temporal**: long-running drafts, accepted docs with TODO markers, rejected docs still referenced as active.
5. **Report**: severity-grouped findings (critical / cascade / temporal).
6. **Assisted fix** (interactive): offer to update findings one at a time via `update_document`; never modify without confirmation.

#### Constraints

- Must verify MCP availability before analysis
- Must not modify documents without explicit user confirmation
- Report must be concise — findings and actions, not verbose analysis
- Must handle projects with no git history (skip code-drift analysis, still do cascade and temporal)

### hooks configuration

The cascade PostToolUse entry is shipped in all three host hook configs:

```json
{
  "matcher": "mcp__archcore__update_document",
  "hooks": [{"type": "command", "command": "${PLUGIN_ROOT}/bin/check-cascade", "timeout": 3}]
}
```

SessionStart hook calls `bin/check-staleness` internally as part of `bin/session-start`.

### bin/ Scripts

#### bin/check-staleness

Called from `bin/session-start` after normal context loading. Performs git-based code-doc drift detection. Rate-limited to once per 24 hours via a timestamp file (`$CLAUDE_PLUGIN_DATA/archcore/last-staleness`, with XDG/HOME fallbacks).

Requirements: executable, exits 0, completes within 3 seconds, POSIX shell, degrades gracefully without git.

#### bin/check-cascade

PostToolUse handler for cascade detection after `update_document`.

Requirements: executable, exits 0, reads JSON from stdin, outputs JSON with `hookSpecificOutput` when cascade detected, POSIX shell.

## Normative Behavior

- Layer 1 MUST run at every session start when git is available and `.archcore/` has commits.
- Layer 1 MUST NOT block session start regardless of findings.
- Layer 1 output MUST NOT exceed 2KB.
- Layer 2 MUST fire only after `update_document`, not after `create_document` or `remove_document`.
- Layer 2 MUST only flag documents connected via `implements`, `depends_on`, or `extends` (not `related`).
- Layer 2 MUST NOT block the update operation.
- Layer 3 (`/archcore:audit --drift`) MUST verify MCP availability before analysis.
- Layer 3 MUST NOT modify documents without explicit user confirmation per document.
- Layer 3 MUST present findings grouped by severity (critical, cascade, temporal).
- All three layers MUST degrade gracefully when git is unavailable.
- All hooks MUST be POSIX shell compatible.
- All hooks MUST exit 0 (never block).

## Constraints

- Layer 1: max 3 seconds execution, max 2KB output.
- Layer 2: max 3 seconds execution (PostToolUse timeout).
- Drift-mode reference (`skills/audit/lib/drift-detection.md`): max 200 lines.
- `bin/check-staleness` and `bin/check-cascade`: POSIX shell, no network access, no file modifications.

## Invariants

- SessionStart always loads context even if staleness check fails or is skipped.
- PostToolUse validation (`archcore doctor`) runs independently of cascade detection — both fire, neither depends on the other.
- The drift mode reads documents via MCP tools, never via direct file reads for `.archcore/` content.
- Cascade detection never fires on `create_document` — only `update_document`.
- No layer ever modifies documents autonomously — Layer 3 requires user confirmation.

## Error Handling

- **Git unavailable**: Layer 1 skips silently. Layer 3 skips code-drift analysis but still performs cascade and temporal checks.
- **No `.archcore/` commits**: Layer 1 skips. Layer 3 falls back to file modification times.
- **archcore CLI unavailable**: Layer 2 skips. Layer 3 uses MCP tools directly.
- **Relation graph empty**: Layer 2 produces no output. Layer 3 skips cascade analysis.
- **Large project (>100 documents)**: Layer 3 should scope analysis when possible. Suggest user provides tag/category filter.

## Conformance

The Actualize system conforms to this specification if:

1. `bin/check-staleness` runs at SessionStart and produces code-drift warnings when applicable.
2. `bin/check-cascade` runs after `update_document` and produces cascade warnings when applicable.
3. Every host hook config (`hooks.json`, `cursor.hooks.json`, `codex.hooks.json`) registers `check-cascade` on `update_document`.
4. `/archcore:audit --drift` exists as a mode of the `audit` intent skill, with routing-table support and the 3-dimension analysis.
5. The drift protocol lives at `skills/audit/lib/drift-detection.md`.
6. All hooks complete within their timeout budgets.
7. No layer blocks operations or modifies documents without user confirmation.
8. All layers degrade gracefully when git or CLI is unavailable.
