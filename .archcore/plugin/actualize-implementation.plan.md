---
title: "Actualize System Implementation Plan"
status: accepted
tags:
  - "hooks"
  - "plugin"
  - "roadmap"
  - "skills"
---

## Goal

Implement the 3-layer Actualize system for documentation freshness detection as specified in the Actualize System ADR and Specification. Deliver all components: two new bin scripts (check-staleness, check-cascade), updated session-start script, updated hooks.json, new `/archcore:actualize` intent skill, and updated `/archcore:help` skill.

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
8. Output formatted warning (max 2KB)

Files: `bin/check-staleness` (new, ~50 lines)

**1.2 Extend `bin/session-start`**

Add call to `bin/check-staleness` after the successful `archcore hooks claude-code session-start` line. The staleness output is appended to the session context.

```sh
# After existing archcore hooks call:
STALENESS=$("${CLAUDE_PLUGIN_ROOT}/bin/check-staleness" 2>/dev/null) || true
if [ -n "$STALENESS" ]; then
  echo ""
  echo "$STALENESS"
fi
```

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

Note: Reads `.sync-state.json` directly (faster than CLI call, within 3s budget). This file is always present and updated by MCP tools.

Files: `bin/check-cascade` (new, ~60 lines)

**2.2 Update `hooks/hooks.json`**

Add new PostToolUse entry for cascade detection:

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

Append to the existing PostToolUse array (after the MCP validation entry).

Files: `hooks/hooks.json` (edit)

### Phase 3: Layer 3 — Deep Analysis Skill

**3.1 Create `skills/actualize/SKILL.md`**

New Layer 1 intent skill for comprehensive staleness analysis.

Frontmatter:

- `name: actualize`
- `argument-hint: "[scope: tag, category, or 'all']"`
- `description: Detect stale documentation and suggest updates based on code changes and relation graph.`

Note: under the Inverted Invocation Policy (adopted after this plan was drafted), intent skills are auto-invocable and do NOT carry `disable-model-invocation`. The actualize skill follows that policy as shipped.

Content structure (following intent skill pattern):

1. Title + one-liner
2. When to Use (vs review, vs status)
3. Routing Table (full / tag-scoped / category-scoped / type-scoped)
4. Execution:
   - Step 1: Gather (list_documents + list_relations + git log)
   - Step 2: Apply scope filter from $ARGUMENTS
   - Step 3: Analyze Code→Doc drift (for each doc: extract path references, check git changes)
   - Step 4: Analyze Doc→Doc cascade (traverse relation graph, compare modification dates)
   - Step 5: Analyze Temporal (draft age, TODO markers, rejected in chains)
   - Step 6: Report (grouped by severity: critical, cascade, temporal)
   - Step 7: Assisted fix (offer update_document per finding, one at a time)
5. Result

Files: `skills/actualize/SKILL.md` (new, ~250 lines)

### Phase 4: Integration Updates

**4.1 Update `skills/help/SKILL.md`**

Add `/archcore:actualize` to the Quick Start section, between `status` and `help`.

**4.2 Update `agents/archcore-auditor.md`**

Add a 6th audit dimension: "Code-Document Correlation" — check if documents reference code paths that have changed. This enhances the background auditor to include drift detection when spawned.

### Phase 5: Validation

**5.1 Structural validation**

- Verify `bin/check-staleness` and `bin/check-cascade` are executable
- Verify `hooks/hooks.json` has 4 hook entries (1 SessionStart, 1 PreToolUse, 2 PostToolUse — MCP mutations + cascade)
- Verify `skills/actualize/SKILL.md` exists with correct frontmatter
- Count at plan completion: 32 skill directories (8 intent + 6 track + 17 type + 1 utility). Today: 33 (intent grew to 9 with `graph`).

**5.2 Content validation**

- `bin/check-staleness`: exits 0 in all cases, output < 2KB, works without git
- `bin/check-cascade`: exits 0 in all cases, reads sync-state.json correctly, outputs valid JSON
- `skills/actualize/SKILL.md`: has all 5 intent skill sections, routing table, within 300 lines
- `skills/help/SKILL.md`: lists all primary intent commands

**5.3 Integration validation**

- `bin/session-start` calls `bin/check-staleness` after context loading
- `hooks/hooks.json` cascade matcher fires only on `update_document`
- No existing hook behavior is broken

## Acceptance Criteria

- [x] `bin/check-staleness` produces code-drift warnings when `.archcore/` is behind code changes
- [x] `bin/check-staleness` exits cleanly with no output when no drift or git unavailable
- [x] `bin/session-start` includes staleness check output in session context
- [x] `bin/check-cascade` produces cascade warnings after `update_document` when dependents exist
- [x] `bin/check-cascade` exits cleanly with no output when no cascade
- [x] `hooks/hooks.json` has 4 entries: SessionStart, PreToolUse, 2x PostToolUse (MCP mutations + update_document cascade). No PostToolUse Write|Edit entry — PreToolUse already blocks.
- [x] `/archcore:actualize` skill exists with routing table, 3-dimension analysis, and assisted fix
- [x] `/archcore:help` lists all primary commands including actualize
- [x] `archcore-auditor` includes code-doc correlation dimension
- [x] All bin scripts are POSIX shell compatible and exit 0
- [x] All bin scripts degrade gracefully when git or CLI is unavailable
- [x] Total skill directory count at plan completion: 32 (today 33 after `graph` was added)

## Dependencies

- Actualize System ADR (accepted) — architectural decision
- Actualize System Specification (accepted) — detailed contract
- Hooks and Validation System Specification (updated) — extended hook contract
- Plugin Architecture Specification (updated) — intent skills, hooks
- Skills System Specification (updated) — intent skills
- Commands System Specification (updated) — actualize in Tier 1
