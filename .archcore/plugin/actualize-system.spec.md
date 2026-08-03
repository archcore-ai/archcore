---
title: "Actualize System Specification (now /archcore:audit --drift)"
status: accepted
tags:
  - "hooks"
  - "plugin"
  - "skills"
  - "validation"
---

## Purpose & Scope

**Outcome (2026-05-15):** the Actualize system shipped, but as the `--drift` mode of the unified `/archcore:audit` skill rather than as a standalone `/archcore:actualize` skill. Layer 1 and Layer 2 — the `bin/check-staleness` SessionStart hook and the `bin/check-cascade` PostToolUse hook — shipped as designed. Layer 3 was folded into `audit` by `skill-surface-collapse.adr`, and the detection protocol now lives at `skills/audit/lib/drift-detection.md`. This document remains normative for the three detection layers themselves; the user-facing command contract lives in `commands-system.spec` and `plugin-architecture.spec`.

This spec defines the contract for documentation freshness detection: the SessionStart staleness check (Layer 1), the PostToolUse cascade detection (Layer 2), and the drift analysis mode of `/archcore:audit` (Layer 3) — their triggers, detection logic, output formats, and interaction with the existing hooks and MCP tools. Normative for `@plugins/archcore/bin/check-staleness`, `@plugins/archcore/bin/check-cascade`, and the four host hook configs. `actualize-system.adr` records the architectural rationale, and `hooks-validation-system.spec` defines the hook execution model this system extends. Out of scope: structural validation (`archcore doctor`), the dashboard and `--deep` modes of `/archcore:audit`, and the `archcore-auditor` agent.

## Surface

The system detects three kinds of staleness: **code→doc drift** (source changes that invalidate documentation), **doc→doc cascade** (a document update that leaves related documents behind), and **temporal staleness** (a document left in one status longer than expected). Detection runs at three depths.

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

**Layer 1 — `bin/check-staleness`.** Called from `bin/session-start` after the CLI availability check and context loading succeed. It resolves the last `.archcore/` commit, diffs non-`.archcore/` paths from there to HEAD, matches each document's directory references against the changed set, and emits plain text as SessionStart additional context, rate-limited to once per 24 hours through a timestamp file at `$CLAUDE_PLUGIN_DATA/archcore/last-staleness`, with XDG and HOME fallbacks.

```
[Archcore Staleness] {N} source files changed since last documentation update.
Potentially affected documents:
  - {doc-path} — references {dir/} ({M} files changed)
Run /archcore:audit --drift for detailed analysis.
```

**Layer 2 — `bin/check-cascade`.** Registered as a PostToolUse entry in all four host hook configs (`hooks/hooks.json`, `hooks/cursor.hooks.json`, `hooks/codex.hooks.json`, `hooks/copilot.hooks.json`). It parses the updated document path from stdin, queries `.archcore/.sync-state.json` for relations whose target is that path and whose type is `implements`, `depends_on`, or `extends`, and reports the source documents of those relations.

| Relation in graph | Updated doc role | Potentially stale doc | Why |
|---|---|---|---|
| B `implements` A | A (target) | B (source) | B implements changed specification |
| B `depends_on` A | A (target) | B (source) | B depends on changed dependency |
| B `extends` A | A (target) | B (source) | B extends changed base |

`related` relations are excluded to reduce noise. The envelope differs per host — Claude Code and Codex take `hookSpecificOutput.additionalContext`, Cursor takes `additional_context`, Copilot a bare top-level `additionalContext`, and OpenCode the plain message. The script never builds these by hand; the output helpers in `@plugins/archcore/bin/lib/normalize-stdin.sh` select the shape from `ARCHCORE_HOST`.

**Host hook wiring.** Claude Code, Cursor, and Codex share one shape, differing only in the plugin-root variable:

```json
{
  "matcher": "mcp__archcore__update_document",
  "hooks": [{"type": "command", "command": "${PLUGIN_ROOT}/bin/check-cascade", "timeout": 3}]
}
```

Copilot's entry is structurally different and cannot be produced by substituting that variable. It is a flat object carrying `bash` instead of `command`, `timeoutSec` instead of `timeout`, `cwd: "."`, `env.ARCHCORE_HOST=copilot`, and **no matcher**, because Copilot's `postToolUse` accepts none — the script self-filters there on the normalized tool name. Its `bash` value probes `$COPILOT_PLUGIN_ROOT`, `$PLUGIN_ROOT`, and `$CLAUDE_PLUGIN_ROOT` in turn with `-x`, execs the first that holds `bin/check-cascade`, and otherwise warns on stderr and exits 0. Until 2026-07-27 it was the one-liner `"${COPILOT_PLUGIN_ROOT}"/bin/check-cascade`, which resolved to the literal path `/bin/check-cascade` whenever that undocumented variable was unset. The live config is `@plugins/archcore/hooks/copilot.hooks.json`; the reasoning is in `copilot-adapter-design.adr`.

**Layer 3 — `/archcore:audit --drift`.** A mode of the `audit` intent skill, activated by the `--drift` flag or by drift phrasing such as "are any docs stale?". Drift mode loads `skills/audit/lib/drift-detection.md` for the protocol, then gathers (`list_documents`, `list_relations`, `git log`), analyses code→doc drift, doc→doc cascade, and temporal staleness, reports findings grouped by severity, and offers an assisted fix one document at a time.

| Signal | Mode | Scope |
|---|---|---|
| No arguments | → short dashboard | All documents |
| `--deep` | → deep audit | All documents |
| `--deep <filter>` or `<filter>` (non-flag arg) | → deep audit, filtered | Filter applied |
| `--drift` | → drift detection | All documents |
| `--drift <filter>` | → drift detection, filtered | Filter applied |

## Normative Behavior

1. WHEN a session starts and git is available and `.archcore/` has commits, Layer 1 MUST run.
2. Layer 1 MUST NOT block session start.
3. Layer 1 MUST keep its output at or below 2 KB.
4. WHEN `update_document` succeeds, Layer 2 MUST fire.
5. Layer 2 MUST NOT fire after `create_document` or `remove_document`.
6. Layer 2 MUST flag only documents connected by `implements`, `depends_on`, or `extends`.
7. Layer 2 MUST NOT block the update operation.
8. IF a host's PostToolUse event accepts no matcher, THEN `bin/check-cascade` MUST reach the same decision by filtering on the normalized tool name.
9. The set of updates that trigger a cascade warning MUST NOT differ by host.
10. IF a host's hook command cannot resolve `bin/check-cascade`, THEN the command MUST exit 0 rather than fail.
11. A post-mutation hook MUST NOT turn a resolution failure into a verdict, because it carries no verdict to deliver.
12. WHEN Layer 3 starts, it MUST verify MCP availability before any analysis.
13. Layer 3 MUST NOT modify a document without explicit user confirmation for that document.
14. Layer 3 MUST group its findings by severity: critical, cascade, temporal.
15. Each layer MUST degrade to a skip when git is unavailable.
16. Each hook MUST be POSIX shell compatible.
17. Each hook MUST exit 0.

## Constraints & Invariants

- Constraint: Layer 1 MUST complete within 3 seconds, because it runs inside session start.
- Constraint: Layer 2 MUST complete within 3 seconds, which is the PostToolUse timeout.
- Constraint: the drift-mode reference `skills/audit/lib/drift-detection.md` MUST NOT exceed 200 lines.
- Constraint: `bin/check-staleness` and `bin/check-cascade` MUST make no network access.
- Constraint: `bin/check-staleness` and `bin/check-cascade` MUST modify no file.
- Constraint: Layer 3 keeps its report to findings and actions rather than narrated analysis, because the report is read inside a session turn.
- Invariant: SessionStart loads context even when the staleness check fails or is skipped.
- Invariant: PostToolUse validation (`archcore doctor`) and cascade detection run independently; both fire, neither depends on the other.
- Invariant: drift mode reads `.archcore/` content through MCP tools, never through direct file reads.
- Invariant: cascade detection never fires on `create_document`.
- Invariant: no layer modifies a document autonomously; Layer 3 requires user confirmation.

## Failure Behavior

1. IF git is unavailable, THEN Layer 1 MUST skip silently.
2. IF git is unavailable, THEN Layer 3 MUST skip code-drift analysis and still run the cascade and temporal checks.
3. IF `.archcore/` has no commits, THEN Layer 1 MUST skip.
4. IF `.archcore/` has no commits, THEN Layer 3 MUST fall back to file modification times.
5. IF the archcore CLI is unavailable, THEN Layer 2 MUST skip.
6. IF the relation graph is empty, THEN Layer 2 MUST produce no output.
7. IF the relation graph is empty, THEN Layer 3 MUST skip cascade analysis.
8. IF a hook command cannot locate its script, THEN the command MUST exit 0.
9. IF a hook command cannot locate its script, THEN the command MUST write a warning to stderr. The layer is inert for that session, and the warning is the only signal.
10. IF the project holds more than 100 documents, THEN Layer 3 SHOULD scope the analysis and ask the user for a tag or category filter.

## Conformance

The system is conformant when:

1. `bin/check-staleness` runs at SessionStart and produces a code-drift warning where one applies.
2. `bin/check-cascade` runs after `update_document` and produces a cascade warning where one applies.
3. Every host hook config registers `check-cascade` on `update_document` — by matcher where the host has one, and by the script's own filtering on Copilot, which has none.
4. `/archcore:audit --drift` exists as a mode of the `audit` skill, with routing-table support and all three analyses.
5. The drift protocol lives at `skills/audit/lib/drift-detection.md`.
6. Every hook completes inside its timeout budget.
7. No layer blocks an operation, and no layer modifies a document without user confirmation.
8. Every layer degrades to a skip when git or the CLI is unavailable.
