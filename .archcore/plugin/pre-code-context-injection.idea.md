---
title: "Pre-Code Context Injection — PreToolUse Hook for Source-File Edits"
status: accepted
tags:
  - "architecture"
  - "hooks"
  - "plugin"
  - "validation"
---

## Idea

Add a `PreToolUse Write|Edit` hook entry that fires on source-file paths (outside `.archcore/`) and injects a compact list of relevant documents — ADRs, rules, specs, cpats — into the agent's context before the write executes. Output is injected as `additionalContext` with one-line excerpts, not full document content.

This closes the biggest gap identified in `jtbd-alignment-analysis.idea.md` — the absence of any mechanism that activates when the agent is about to modify code, rather than documentation. Without this hook, "Archcore makes the agent code with your project's architecture, rules, and decisions" is an aspirational claim. With this hook, it becomes an engineered guarantee the plugin can demonstrate on first install.

## Status — Realized (Phase 1, MVP)

Shipped in plugin 0.3.0 via `bin/check-code-alignment`. Registered in both `hooks/hooks.json` (Claude Code) and `hooks/cursor.hooks.json` (Cursor) as a second entry on the `PreToolUse Write|Edit` matcher, coexisting with the existing `check-archcore-write` blocking hook. See `pre-code-hook-implementation.plan.md` for the execution plan and acceptance criteria. See `hooks-validation-system.spec.md` (Hook 3) for the formal contract.

Phase 1 is **grep-based, no path index** — scans `.archcore/**/*.md` via `grep -rlF` per token on each edit. Ranking: specificity (longest matching directory prefix) → type priority (`rule > cpat > adr > spec > guide`). Eligible types: rule, cpat, adr, spec, guide (prd/idea/plan/rfc/etc. are filtered out as not line-of-code enforceable). Top-3 cap, 2 KB output cap.

Settings shape implemented:

```json
{ "codeAlignment": { "sourceRoots": ["src", "lib", ...] } }
```

Default roots when `sourceRoots` is absent: `src lib app pkg cmd internal apps packages modules components`. Escape hatch: `ARCHCORE_DISABLE_INJECTION=1` environment variable.

Unit tests: 13 cases in `test/unit/check-code-alignment.bats` — silent-pass paths, injection correctness, specificity ranking, top-3 truncation, type allowlist, settings override, Cursor JSON shape, non-blocking safety.

### Concrete shape

```
Agent calls Write/Edit on src/api/handlers/users.ts
  ↓
[PreToolUse Write|Edit] bin/check-archcore-write   → allow (path not .archcore/)
[PreToolUse Write|Edit] bin/check-code-alignment   → scan .archcore/
  ↓
  Token set: "src/api/handlers/", "src/api/", "src/" (longest first, cap 5)
  ↓
  additionalContext injected:
    "[Archcore Context] Before editing src/api/handlers/users.ts:
     - rule: API Handlers Layout [plugin/api-handlers.rule.md]
     - adr: REST Conventions [plugin/rest-conventions.adr.md]
     - cpat: Handler Error Wrapping [plugin/handler-error.cpat.md]"
  ↓
Write proceeds, agent now has the right constraints in context
```

### Path index — deferred to Phase 2

The originally proposed pre-built path index in `.sync-state.json` remains a Phase 2 item. The MVP scans each edit with `grep -rlF` per token. This is acceptable for corpora ≤50 documents within the 1-second hook timeout; larger corpora benefit from the index.

A CLI subcommand `archcore align <path>` (or equivalent backing for `search_documents`) is a natural home for the index and unifies the hook and `/archcore:context` skill on one primitive.

### hooks.json addition (as implemented)

```json
{
  "matcher": "Write|Edit",
  "hooks": [
    { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-archcore-write", "timeout": 1 },
    { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-code-alignment", "timeout": 1 }
  ]
}
```

Cursor parity via `hooks/cursor.hooks.json` `preToolUse` `Write` matcher, same second-entry placement.

`check-archcore-write` blocks `.archcore/*.md` and passes through otherwise. `check-code-alignment` short-circuits silently inside `.archcore/` and only does real work on source paths. They act on disjoint path sets by construction.

### bin/check-code-alignment (as implemented)

POSIX-shell script. Responsibilities:

1. Read JSON from stdin via `bin/lib/normalize-stdin.sh` — host-normalized extraction.
2. Short-circuit (exit 0, empty output) when: no `file_path`, no `.archcore/`, path inside `.archcore/`, absolute path outside `$CWD`, `ARCHCORE_DISABLE_INJECTION=1`.
3. Source-root filter against `.archcore/settings.json → codeAlignment.sourceRoots` (fallback to built-in defaults).
4. Generate candidate tokens — directory prefixes, longest-first, cap 5 levels.
5. Scan `.archcore/**/*.md` via `grep -rlF` per token; de-duplicate documents (longest matching token wins); filter to eligible types; rank and cap at top-3.
6. Render compact block and emit via `archcore_hook_pretool_info` (host-aware JSON shape). Cap 2 KB.

## Value

### Closes the primary JTBD gap

Without this hook, the agent reads rules and ADRs only if it spontaneously decides to. With this hook, every source-file edit carries its applicable constraints. That is the difference between a knowledge base the agent *can* read and one it *must* see.

### Scales with the knowledge base

The more rules and cpats a team captures, the more value the hook delivers — exactly the growth dynamic Archcore wants. Teams that record a decision and a rule for a single module now get automatic enforcement for every future agent edit in that module, across sessions and subagents (when combined with the subagent knowledge preload).

### Differentiates from memory tools

claude-mem, Memory Bank, Mem0 all solve "recall past context". None of them inject *typed, project-specific constraints* at the moment of code change. This hook is specifically about constraints at the boundary, not recall, and it is the clearest wedge against generic memory products.

### Cheap to demonstrate

The README hero reel immediately becomes compelling: user asks for a feature, agent sees rule+ADR appear in its context before it writes, produces code that respects both. No narrative required.

## Remaining Phases (Phase 2–4)

### Phase 2 — Path index (deferred)

- CLI: persist a path index in the sync manifest, updated on every `create_document` / `update_document` / `remove_document`. Lookup becomes O(1).
- Performance budget: hook must complete in under 500 ms on a 500-document repo.
- Prerequisite for corpora > ~50 documents.

### Phase 3 — Ranking improvements and session dedup (deferred)

- Session-level de-duplication: do not re-inject the same document within the same session unless the document has changed. Reduces repetition fatigue.
- More nuanced specificity scoring (e.g., penalize documents that mention many paths as generic reference).

### Phase 4 — Measurement (deferred)

- CLI telemetry (opt-in): count of injections per session, top-cited documents. Feeds back into `/archcore:review` as "most-applied rules".

## Risks and Constraints

- **Performance.** Phase 1 grep-per-token is O(tokens × docs). Acceptable for ≤50 docs; degrades for larger corpora. Phase 2 path index is the fix.
- **False positives.** A document referencing `src/` generically matches every source edit. Specificity ranking mitigates but does not eliminate. Monitor precision via user feedback.
- **Hook noise vs value.** Two `PreToolUse` entries on `Write|Edit` double the per-edit shell fork. Short-circuit paths keep the overhead minimal for non-source files, but watch cumulative overhead on hot repos.
- **Trigger surface too narrow.** `Write|Edit` catches inline edits, not code reviewed in a planning tool and pasted later. Acceptable for v1.
- **Coupling to path conventions.** Monorepos with non-standard roots need `codeAlignment.sourceRoots` configured. Default conservative set covers common layouts.
- **Subagent compatibility.** Hooks fire for subagent tool calls too — combined with `subagent-knowledge-tree-bootstrap.adr`, delegated work is covered.
- **Cursor parity.** Whether Cursor's `preToolUse` respects `additional_context` is host-version-dependent. Graceful degradation: if ignored, the hook becomes a no-op on Cursor; SessionStart context + `/archcore:context` skill still carry the pull path.
- **User control.** `ARCHCORE_DISABLE_INJECTION=1` gives a global off-switch. Per-path muting is not yet implemented.

## Related work in this repo

- `jtbd-alignment-analysis.idea.md` — names this proposal as the single highest-impact addition to close the JTBD #1 gap
- `pre-code-hook-implementation.plan.md` — execution plan and acceptance criteria (Phase 1 delivered)
- `hooks-validation-system.spec.md` — formal contract for the 5-hook system
- `subagent-knowledge-tree-bootstrap.adr` — complementary; subagent coverage requires both
- `code-alignment-intent-skill.idea.md` — pull counterpart (now realized as `/archcore:context`)
- `context-skill-implementation.plan.md` — pull-mode implementation plan
- `multi-host-compatibility-layer.spec.md` — Cursor parity path
