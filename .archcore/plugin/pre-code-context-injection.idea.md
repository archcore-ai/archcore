---
title: "Pre-Code Context Injection — PreToolUse Hook for Source-File Edits"
status: draft
tags:
  - "architecture"
  - "hooks"
  - "plugin"
  - "validation"
---

## Idea

Add a `PreToolUse Write|Edit` hook entry that fires on source-file paths (outside `.archcore/`) and injects a compact list of relevant documents — ADRs, rules, specs, cpats — into the agent's context before the write executes. The hook uses a pre-built path index so lookup is O(1) per file path. Output is injected as `additionalContext` with one-line excerpts, not full document content.

This closes the biggest gap identified in `jtbd-alignment-analysis.idea.md` — the absence of any mechanism that activates when the agent is about to modify code, rather than documentation. Without this hook, "Archcore makes the agent code with your project's architecture, rules, and decisions" is an aspirational claim. With this hook, it becomes an engineered guarantee the plugin can demonstrate on first install.

### Concrete shape

```
Agent calls Write/Edit on src/api/handlers/users.ts
  ↓
[PreToolUse Write|Edit] bin/check-archcore-write   → allow (path not .archcore/)
[PreToolUse Write|Edit] bin/check-code-alignment   → query path index
  ↓
  Index lookup: documents referencing src/api/handlers/ or src/api/
  ↓
  additionalContext injected:
    "[Archcore Context] Before editing src/api/handlers/users.ts, these apply:
     - rule:api-handlers-layout — 'Handlers live in src/api/handlers/, one per resource'
     - adr:rest-conventions — 'We use REST, not RPC-over-HTTP'
     - cpat:handler-error-wrapping — 'Wrap errors with withErrorBoundary(), not try/catch'"
  ↓
Write proceeds, but agent now has the right constraints in context
```

### Path index

A pre-computed map from source path prefixes to document paths, maintained by `archcore` CLI:

```
src/api/              → [rule:api-handlers-layout, adr:rest-conventions]
src/api/handlers/     → [rule:api-handlers-layout, cpat:handler-error-wrapping]
src/payments/         → [adr:use-stripe, spec:payment-flow, rule:money-arithmetic]
```

Built by scanning document content for path-like tokens (`src/...`, `lib/...`, module names from `package.json`/`go.mod`/etc.), and updated whenever `create_document` / `update_document` fires (tie into existing PostToolUse validation path, or regenerate lazily on sync manifest write).

### hooks.json addition

Sits alongside the existing `check-archcore-write` entry in the same `PreToolUse Write|Edit` matcher block — both run, each handles its own domain:

```json
{
  "matcher": "Write|Edit",
  "hooks": [
    { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-archcore-write", "timeout": 1 },
    { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-code-alignment", "timeout": 1 }
  ]
}
```

`check-archcore-write` blocks `.archcore/*.md` and passes through otherwise. `check-code-alignment` ignores `.archcore/*.md` and injects context on source paths. They do not conflict because `check-archcore-write` returns exit 2 only on `.archcore/*.md`, and at that point Claude Code has already rejected the call — `check-code-alignment`'s output is irrelevant for blocked calls.

### bin/check-code-alignment

New POSIX-shell script. Responsibilities:

1. Read JSON from stdin, extract `tool_input.file_path`
2. Skip if path is inside `.archcore/` (sister hook handles it)
3. Skip if path does not match any project source root (configurable in `.archcore/settings.json`)
4. Query the path index via `archcore` CLI — new subcommand `archcore align <path>` that returns a ranked list of documents referencing the path, up to N (default 3)
5. If no matches: exit 0 with empty output
6. If matches: emit `hookSpecificOutput.additionalContext` with compact one-liner per document

Output cap: 2 KB. Ranking: specificity first (deeper path match > root path match), then type priority (`rule` > `adr` > `spec` > `cpat` > `guide`), then most recently updated.

## Value

### Closes the primary JTBD gap

Without this hook, the agent reads rules and ADRs only if it spontaneously decides to. With this hook, every source-file edit carries its applicable constraints. That is the difference between a knowledge base the agent *can* read and one it *must* see.

### Scales with the knowledge base

The more rules and cpats a team captures, the more value the hook delivers — exactly the growth dynamic Archcore wants. Teams that record a decision and a rule for a single module now get automatic enforcement for every future agent edit in that module, across sessions and subagents (when combined with the subagent knowledge preload).

### Differentiates from memory tools

claude-mem, Memory Bank, Mem0 all solve "recall past context". None of them inject *typed, project-specific constraints* at the moment of code change. This hook is specifically about constraints at the boundary, not recall, and it is the clearest wedge against generic memory products.

### Cheap to demonstrate

The README hero reel immediately becomes compelling: user asks for a feature, agent sees rule+ADR appear in its context before it writes, produces code that respects both. No narrative required.

## Possible Implementation

### Phase 1 — Minimum viable injection (2–3 days)

- CLI: `archcore align <path> [--limit N]` — returns JSON array of `{path, type, title, summary}`. Initial implementation: grep document bodies for path tokens on each call. Slow but correct; acceptable for projects under ~50 docs.
- Plugin: new `bin/check-code-alignment` script, new hooks.json entry in both Claude Code and Cursor manifests.
- Settings: add `codeAlignment: { enabled: bool, sourceRoots: ["src", "lib"], maxMatches: 3 }` to `.archcore/settings.json`.
- Spec update: extend `hooks-validation-system.spec.md` with a fifth hook, and add the invariant "every `Write|Edit` to a source path that references documented constraints produces an `additionalContext` injection".

### Phase 2 — Path index (3–5 days)

- CLI: persist a path index in the sync manifest, updated on every `create_document` / `update_document` / `remove_document`. Lookup becomes O(1).
- Performance budget: hook must complete in under 500ms on a 500-document repo.

### Phase 3 — Ranking and de-duplication (2–3 days)

- Rank by specificity (longest matching path wins), then type priority, then recency.
- De-duplicate when multiple documents reference the same rule.
- Session-level de-duplication: do not re-inject the same document within the same session unless the document has changed.

### Phase 4 — Measurement (1 day)

- CLI telemetry (opt-in): count of injections per session, top-cited documents. Feeds back into `/archcore:review` as "most-applied rules".

## Risks and Constraints

- **Performance.** 1-second `PreToolUse` budget is tight. Phase 1 grep-based lookup is risky for large repos — Phase 2 path index is a hard prerequisite for repos over ~50 docs. Budget must hold at the 99th percentile, not the median.
- **False positives.** A document referencing `src/` generically is not necessarily applicable to a specific edit in `src/payments/utils/`. Ranking must penalize generic path prefixes and reward specific ones. If precision is low, users learn to ignore injections — the same failure mode as overzealous linters.
- **Hook noise vs value.** Two `PreToolUse` entries on `Write|Edit` double the per-edit shell fork. On a large repo with many edits, the cumulative overhead is non-trivial even if each call is fast. Mitigation: short-circuit in shell before invoking the CLI when the path is outside configured source roots.
- **Trigger surface too narrow.** `Write|Edit` catches inline edits, but not agent-generated code reviewed in a planning tool and pasted later. Acceptable for v1 — the vast majority of coding-agent edits go through `Write|Edit`.
- **Coupling to path conventions.** A repo without a clean `src/` layout (e.g., monorepos with many roots) requires configuration. The `sourceRoots` setting is the escape hatch. Default conservative: if `sourceRoots` is not configured, do not inject.
- **Subagent compatibility.** Hooks fire for subagent tool calls too, so the mechanism works for delegated work as long as the subagent has already received the knowledge tree (see `subagent-knowledge-tree-preload.idea.md`). Without the tree preload, the subagent sees the injection but not the overall structure — still useful, but weaker.
- **Cursor parity.** Cursor's `preToolUse` hook event is similar but not identical to Claude Code's. The multi-host compatibility layer must normalize both.
- **Churn on document renames.** When a document is renamed or removed, the path index must be invalidated. Tie this to the existing sync manifest write path.
- **User control.** Users need a way to mute injections per-path or globally for a session (e.g., `ARCHCORE_SKIP_ALIGNMENT=1`). Default-on is correct for the value prop; an escape hatch is correct for UX when users know better than the index.

## Related work in this repo

- `jtbd-alignment-analysis.idea.md` — names this proposal as the single highest-impact addition to close the JTBD #1 gap
- `hooks-validation-system.spec.md` — target for extension; this becomes the fifth hook
- `subagent-knowledge-tree-preload.idea.md` — complementary; subagent coverage requires both the preload and this hook
- `code-alignment-intent-skill.idea.md` — user-facing pull counterpart to this push mechanism
- `multi-host-compatibility-layer.spec.md` — path for Cursor parity
