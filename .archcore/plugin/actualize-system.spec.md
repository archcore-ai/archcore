---
title: "Actualize System Specification"
status: accepted
tags:
  - "hooks"
  - "plugin"
  - "skills"
  - "validation"
---

## Purpose

Define the contract for the Actualize system — a 3-layer documentation freshness detection mechanism that identifies stale `.archcore/` documents through passive session-start checks, reactive cascade detection after document updates, and deep on-demand analysis via an intent skill.

## Scope

This specification covers: the SessionStart staleness check (Layer 1), the PostToolUse cascade detection (Layer 2), and the `/archcore:actualize` intent skill (Layer 3). It defines their triggers, detection logic, output formats, and interaction with existing hooks and MCP tools.

It does not cover: structural validation (`archcore validate`), the `/archcore:review` skill (coverage/relation health), or the archcore-auditor agent (read-only background audits). Those are complementary but separate.

## Authority

This specification is the authoritative reference for all staleness detection behavior in the plugin. The Actualize System ADR provides the architectural rationale. The Hooks Validation System Specification defines the hook execution model this system extends.

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
│  Trigger: /archcore:actualize (user-invoked)             │
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

New script `bin/check-staleness`, called from `bin/session-start` after the normal context loading succeeds.

#### Detection Logic

```
1. LAST_DOC_COMMIT = git log -1 --format=%H -- .archcore/
   (most recent commit touching .archcore/ files)

2. If no commit found → skip (docs never committed, can't compare)

3. CHANGED_FILES = git diff --name-only $LAST_DOC_COMMIT..HEAD -- ':(exclude).archcore/'
   (source files changed since last doc update)

4. If CHANGED_FILES is empty → skip (no code drift)

5. CHANGED_COUNT = count of CHANGED_FILES

6. For each .archcore/**/*.md document:
   - Extract directory references from content (paths like src/, lib/, etc.)
   - Match against CHANGED_FILES
   - If match → add to AFFECTED list with matched paths

7. Output warning with AFFECTED documents and CHANGED_COUNT
```

#### Output Format

When drift is detected:

```
[Archcore Staleness] {N} source files changed since last documentation update.
Potentially affected documents:
  - {doc-path} — references {dir/} ({M} files changed)
  - {doc-path} — references {dir/} ({M} files changed)
Run /archcore:actualize for detailed analysis.
```

When many code changes but no specific document matches:

```
[Archcore Staleness] {N} source files changed since last documentation update.
No specific document references matched, but consider running /archcore:actualize.
```

Output is plain text, injected as SessionStart additional context.

#### Constraints

- Must complete within 3 seconds (shared with session-start context loading)
- Output must not exceed 2KB
- Must degrade gracefully: skip if git is unavailable, if `.archcore/` has no commits, or if project is not a git repo
- Must not block session start — always exit 0
- POSIX shell compatible (no bash-specific features)

### Layer 2: Reactive Cascade Detection (PostToolUse Enhancement)

#### Trigger

PostToolUse hook, fires after `mcp__archcore__update_document` succeeds. Does NOT fire on `create_document` (new documents can't cause cascade) or `remove_document` (removal is intentional).

#### Handler

New script `bin/check-cascade`, registered as an additional PostToolUse hook entry in `hooks/hooks.json`.

#### Detection Logic

```
1. Parse tool_input from stdin JSON
2. Extract updated document path from tool_input.path

3. RELATIONS = archcore relations list --json (or equivalent CLI command)
   Filter to relations where:
   - target = updated document path
   - type ∈ {implements, depends_on, extends}

4. AFFECTED = source documents from filtered relations

5. If AFFECTED is empty → exit 0 with empty output (no cascade)

6. Output cascade warning with AFFECTED documents and relation types
```

#### Output Format

When cascade is detected:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "[Archcore Cascade] Updated \"{document-title}\".\nDocuments that may need review:\n  → {path} ({relation-type} this document)\n  → {path} ({relation-type} this document)\nRun /archcore:actualize {tag} for detailed analysis."
  }
}
```

When no cascade detected: empty output (exit 0).

#### Relation Direction Table

| Relation in graph | Updated doc role | Potentially stale doc | Why |
|---|---|---|---|
| B `implements` A | A (target) | B (source) | B implements changed specification |
| B `depends_on` A | A (target) | B (source) | B depends on changed dependency |
| B `extends` A | A (target) | B (source) | B extends changed base |

`related` relations are excluded to reduce noise.

#### Constraints

- Must complete within 3 seconds (PostToolUse timeout)
- Fires only on `update_document`, not on `create_document` or `remove_document`
- Must not block the operation — always exit 0
- Must degrade gracefully if `archcore` CLI is unavailable
- POSIX shell compatible

### Layer 3: Deep Analysis — /archcore:actualize Skill

#### Classification

Layer 1 intent skill. 8th primary command. User-invoked only (`disable-model-invocation: true`).

#### Frontmatter

```yaml
---
name: actualize
argument-hint: "[scope: tag, directory, or 'all']"
description: Detect stale documentation and suggest updates based on code changes and relation graph.
disable-model-invocation: true
---
```

#### Routing Table

| Signal | Route | Scope |
|---|---|---|
| No arguments | → full analysis | All documents |
| Tag name ("auth", "payments") | → tag-scoped | Filter by tag |
| Category ("knowledge", "vision") | → category-scoped | Filter by category |
| Specific type ("adr", "spec") | → type-scoped | Filter by type |

Default: full analysis of all documents.

#### Execution Flow

**Step 0: Verify MCP**
Call `list_documents`. If unavailable, stop with install/init instructions.

**Step 1: Gather**
- `list_documents` (with optional filters from `$ARGUMENTS`)
- `list_relations`
- `git log --stat` — recent code changes
- `git log -1 --format=%H -- .archcore/` — last doc commit

**Step 2: Analyze — Code→Doc Drift**
For each document in scope:
- Read document content via `get_document`
- Extract file/directory references from content (paths, module names)
- Check `git log` for changes to referenced paths since last doc modification
- If changes found → flag as potential code drift with specific file list

**Step 3: Analyze — Doc→Doc Cascade**
For each document in scope:
- Check relation graph: find documents where this doc is the TARGET of `implements`, `depends_on`, `extends`
- Compare modification dates (git log): if target was modified more recently than source → source may be stale
- Flag cascade findings with relation type and date comparison

**Step 4: Analyze — Temporal Staleness**
- Documents in `draft` status with last git modification > 30 days ago
- Documents in `accepted` status with TODO/FIXME/TBD markers in content
- Plans with phases that reference past dates
- `rejected` documents that are still targets of active `implements` or `depends_on` relations

**Step 5: Report**
Present findings grouped by severity:

```
## Actualize Report

### Critical (code drift with evidence)
- {doc-path}: references {src/path} — {N} files changed since doc was last updated
  Changed files: {file1}, {file2}, ...

### Cascade (relation graph indicates staleness)
- {doc-path}: implements "{target-title}" which was updated on {date}
  This document was last modified on {date} — {N} days before the target changed

### Temporal
- {doc-path}: draft for {N} days — consider accepting or removing
- {doc-path}: accepted but contains {N} TODO markers

### Summary
{N} documents analyzed, {M} findings ({X} critical, {Y} cascade, {Z} temporal)
```

**Step 6: Assisted Fix (interactive)**
For each finding, offer specific action:
- Code drift → "Want me to update this document to reflect the current code?"
- Cascade → "Want me to review and update this document based on the changed {target}?"
- Temporal → "Change status to accepted/rejected?" or "Remove TODO markers?"

Use `update_document` MCP tool for all modifications. One document at a time, with user confirmation.

#### Constraints

- Must verify MCP availability before analysis
- Must not modify documents without explicit user confirmation
- Report must be concise — findings and actions, not verbose analysis
- Must handle projects with no git history (skip code-drift analysis, still do cascade and temporal)

### hooks/hooks.json Updates

Two additions to the existing hooks configuration:

**New PostToolUse entry for cascade detection:**

```json
{
  "matcher": "mcp__archcore__update_document",
  "hooks": [
    {
      "type": "command",
      "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-cascade",
      "timeout": 3
    }
  ]
}
```

**SessionStart hook unchanged** — `bin/session-start` is extended internally to call `bin/check-staleness`.

### bin/ Scripts

Two new executable scripts:

#### bin/check-staleness

Called from `bin/session-start` after normal context loading. Performs git-based code-doc drift detection.

Requirements:
- Must be executable (`chmod +x`)
- Must exit 0 in all cases
- Must complete within 3 seconds
- Must output plain text warning or empty output
- Must skip gracefully if git is unavailable or no `.archcore/` commits exist
- POSIX shell compatible

#### bin/check-cascade

PostToolUse handler for cascade detection after `update_document`.

Requirements:
- Must be executable (`chmod +x`)
- Must read JSON from stdin (same format as `validate-archcore`)
- Must exit 0 in all cases
- Must output JSON with `hookSpecificOutput` when cascade detected, empty otherwise
- Must complete within 3 seconds
- Must skip gracefully if `archcore` CLI is unavailable
- POSIX shell compatible

## Normative Behavior

- Layer 1 MUST run at every session start when git is available and `.archcore/` has commits.
- Layer 1 MUST NOT block session start regardless of findings.
- Layer 1 output MUST NOT exceed 2KB.
- Layer 2 MUST fire only after `update_document`, not after `create_document` or `remove_document`.
- Layer 2 MUST only flag documents connected via `implements`, `depends_on`, or `extends` (not `related`).
- Layer 2 MUST NOT block the update operation.
- Layer 3 MUST verify MCP availability before analysis.
- Layer 3 MUST NOT modify documents without explicit user confirmation per document.
- Layer 3 MUST present findings grouped by severity (critical, cascade, temporal).
- All three layers MUST degrade gracefully when git is unavailable.
- All hooks MUST be POSIX shell compatible.
- All hooks MUST exit 0 (never block).

## Constraints

- Layer 1: max 3 seconds execution, max 2KB output.
- Layer 2: max 3 seconds execution (PostToolUse timeout).
- Layer 3 skill: max 300 lines (intent skill limit).
- `bin/check-staleness` and `bin/check-cascade`: POSIX shell, no network access, no file modifications.
- The `/archcore:actualize` skill is user-only (`disable-model-invocation: true`).

## Invariants

- SessionStart always loads context even if staleness check fails or is skipped.
- PostToolUse validation (`archcore validate`) runs independently of cascade detection — both fire, neither depends on the other.
- The actualize skill reads documents via MCP tools, never via direct file reads for `.archcore/` content.
- Cascade detection never fires on `create_document` — only `update_document`.
- No layer ever modifies documents autonomously — Layer 3 requires user confirmation.

## Error Handling

- **Git unavailable**: Layer 1 skips silently. Layer 3 skips code-drift analysis but still performs cascade and temporal checks.
- **No `.archcore/` commits**: Layer 1 skips (can't determine baseline). Layer 3 falls back to file modification times.
- **archcore CLI unavailable**: Layer 2 skips. Layer 3 uses MCP tools directly (no CLI needed for skill execution).
- **Relation graph empty**: Layer 2 produces no output. Layer 3 skips cascade analysis, reports remaining dimensions.
- **Large project (>100 documents)**: Layer 3 should scope analysis when possible. Suggest user provides tag/category filter.

## Conformance

The Actualize system conforms to this specification if:

1. `bin/check-staleness` runs at SessionStart and produces code-drift warnings when applicable
2. `bin/check-cascade` runs after `update_document` and produces cascade warnings when applicable
3. `hooks/hooks.json` contains the cascade PostToolUse entry
4. `skills/actualize/SKILL.md` exists as a Layer 1 intent skill with routing table and 3-dimension analysis
5. All hooks complete within their timeout budgets
6. No layer blocks operations or modifies documents without user confirmation
7. All layers degrade gracefully when git or CLI is unavailable