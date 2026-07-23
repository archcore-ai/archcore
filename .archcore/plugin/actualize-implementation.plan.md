---
title: "Actualize System Implementation Plan"
status: accepted
tags:
  - "hooks"
  - "plugin"
  - "roadmap"
  - "skills"
---

> **Outcome (2026-05-15):** Plan executed. Layer 1 (`bin/check-staleness`) and Layer 2 (`bin/check-cascade`) shipped as designed. Layer 3 (deep analysis) shipped as the `--drift` mode of `/archcore:audit` rather than as a standalone `/archcore:actualize` intent skill, per `skill-surface-collapse.adr.md`. The drift-mode protocol lives at `skills/audit/lib/drift-detection.md`. All acceptance criteria below are met under the new naming.

## Goal

Implement the 3-layer Actualize system for documentation freshness detection as specified in the Actualize System ADR and Specification. Deliver all components: two new bin scripts (check-staleness, check-cascade), updated session-start script, updated hook configs, deep-analysis mode in the appropriate audit/inspection skill, and updated `/archcore:help` skill.

## Tasks

### Phase 1: Layer 1 — Passive Detection (SessionStart)

**1.1 Create `bin/check-staleness`**

New POSIX shell script that detects code-document drift via git.

Logic:

1. Check if in a git repo (`git rev-parse --git-dir`)
2. Find last `.archcore/` commit: `git log -1 --format=%H -- .archcore/`
3. If no commit → exit 0 (docs never committed)
4. Find changed code files: `git diff --name-only $COMMIT..HEAD -- ':(exclude).archcore/'`
5. If no changes → exit 0
6. Count changed files
7. For each `.archcore/*.md` document: grep for directory references from changed files
8. Output formatted warning (max 2KB), rate-limited to once per 24 hours

Files: `bin/check-staleness` (new, ~50 lines)

**1.2 Extend `bin/session-start`**

Add call to `bin/check-staleness` after the successful `archcore hooks <host> session-start` line. The staleness output is appended to the session context.

Files: `bin/session-start` (edit, ~5 lines added)

### Phase 2: Layer 2 — Reactive Cascade Detection (PostToolUse)

**2.1 Create `bin/check-cascade`**

New POSIX shell script that detects cascade staleness after `update_document`.

Logic:

1. Read JSON from stdin
2. Extract `tool_input.path` (the updated document path)
3. If extraction fails → exit 0
4. Query relation graph via `.archcore/.sync-state.json`: find relations where target matches updated path and type is `implements`, `depends_on`, or `extends`
5. If no matching relations → exit 0
6. Extract document title from tool result or path
7. Output JSON with `hookSpecificOutput.additionalContext` listing affected documents

Files: `bin/check-cascade` (new, ~60 lines)

**2.2 Update hook configs**

Add new PostToolUse entry for cascade detection across all hosts:

```json
{
  "matcher": "mcp__archcore__update_document",
  "hooks": [{"type": "command", "command": "${PLUGIN_ROOT}/bin/check-cascade", "timeout": 3}]
}
```

Files: `hooks/hooks.json`, `hooks/cursor.hooks.json`, `hooks/codex.hooks.json` (edit)

### Phase 3: Layer 3 — Deep Analysis (drift mode of audit)

**3.1 Move drift protocol into the audit skill**

Per `skill-surface-collapse.adr.md`, Layer 3 ships as the `--drift` mode of `/archcore:audit`. Originally this phase was scoped to a new `skills/actualize/SKILL.md`; the same content (routing table, 3-dimension analysis, assisted-fix flow) now lives at `skills/audit/lib/drift-detection.md` and is loaded by `skills/audit/SKILL.md` on `--drift`.

Frontmatter for `audit`:

- `name: audit`
- `argument-hint: "[--deep] [--drift] [category, tag, or scope]"`
- `description: "Audit Archcore docs: dashboard (counts, status, relations, orphans), deep coverage audit, or drift detection (code/cascade/temporal staleness)."`

Content structure for the `audit` skill (drift portion):

1. Title + one-liner
2. When to Use (with explicit anti-trigger for capture / decide)
3. Routing Table (default short / `--deep` / `--drift`)
4. Execution (per mode) — drift mode steps loaded from `lib/drift-detection.md`:
   - Step 1: Gather (list_documents + list_relations + git log)
   - Step 2: Apply scope filter from $ARGUMENTS
   - Step 3: Analyze Code→Doc drift
   - Step 4: Analyze Doc→Doc cascade
   - Step 5: Analyze Temporal
   - Step 6: Report (grouped by severity: critical, cascade, temporal)
   - Step 7: Assisted fix (offer update_document per finding, one at a time)
5. Result

Files: `skills/audit/SKILL.md`, `skills/audit/lib/drift-detection.md` (the latter holds what would have been Phase 3.1's standalone SKILL.md).

### Phase 4: Integration Updates

**4.1 Update `skills/help/SKILL.md`**

Document the `audit` skill's three modes (default short, `--deep`, `--drift`).

**4.2 Update `agents/archcore-auditor.md`**

Add a 6th audit dimension: "Code-Document Correlation" — check if documents reference code paths that have changed. This enhances the background auditor to include drift detection when spawned.

### Phase 5: Validation

**5.1 Structural validation**

- Verify `bin/check-staleness` and `bin/check-cascade` are executable
- Verify every host hook config has the cascade matcher
- Verify `skills/audit/lib/drift-detection.md` exists and is loaded by `skills/audit/SKILL.md`
- Count at plan completion (under `skill-surface-collapse.adr.md`): 7 skills total (`init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`)

**5.2 Content validation**

- `bin/check-staleness`: exits 0 in all cases, output < 2KB, works without git
- `bin/check-cascade`: exits 0 in all cases, reads sync-state.json correctly, outputs valid JSON
- `skills/audit/SKILL.md`: has all 5 sections, three modes routed deterministically
- `skills/help/SKILL.md`: lists all 7 commands

**5.3 Integration validation**

- `bin/session-start` calls `bin/check-staleness` after context loading
- Every host hook config's cascade matcher fires only on `update_document`
- No existing hook behavior is broken

## Acceptance Criteria

- [x] `bin/check-staleness` produces code-drift warnings when `.archcore/` is behind code changes
- [x] `bin/check-staleness` exits cleanly with no output when no drift or git unavailable
- [x] `bin/session-start` includes staleness check output in session context
- [x] `bin/check-cascade` produces cascade warnings after `update_document` when dependents exist
- [x] `bin/check-cascade` exits cleanly with no output when no cascade
- [x] Every host hook config registers `check-cascade` on `update_document`
- [x] `/archcore:audit --drift` exists with routing, 3-dimension analysis, and assisted fix
- [x] `/archcore:help` lists all 7 primary commands including `audit`
- [x] `archcore-auditor` includes code-doc correlation dimension
- [x] All bin scripts are POSIX shell compatible and exit 0
- [x] All bin scripts degrade gracefully when git or CLI is unavailable
- [x] Total skill directory count at plan completion: 7 (per `skill-surface-collapse.adr.md`)

## Dependencies

- Actualize System ADR (accepted) — architectural decision
- Actualize System Specification (accepted) — detailed contract
- Hooks and Validation System Specification — extended hook contract
- Plugin Architecture Specification — intent skills, hooks
- Skills System Specification — intent skills
- Commands System Specification — `audit` in the 7-command surface
- `skill-surface-collapse.adr.md` — the decision that folded Layer 3 into the `audit` skill as `--drift`
