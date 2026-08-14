---
title: "Actualize System Specification (now /archcore:review --drift)"
status: accepted
tags:
  - "hooks"
  - "plugin"
  - "skills"
  - "validation"
---

## Purpose & Scope

**Outcome (2026-05-15):** the Actualize system shipped, but as the `--drift` mode of the unified `/archcore:audit` skill rather than as a standalone `/archcore:actualize` skill. `/archcore:audit` was itself absorbed into `/archcore:review` by `four-command-palette.adr`, so the drift mode is now `/archcore:review --drift`.

**Correction (v0.7.0, `ca6dfb4`):** Layers 1 and 2 no longer ship as plugin scripts. `cli-owns-layers-4-5.adr` moved their policy into the `archcore` binary, and `ca6dfb4` deleted `bin/check-staleness` and `bin/check-cascade`. Layer 3's protocol moved from `skills/audit/lib/drift-detection.md` to the actualize track at `skills/_shared/tracks/actualize.md`.

This spec defines documentation-freshness detection: the SessionStart staleness check (Layer 1), the PostToolUse cascade detection (Layer 2), and `/archcore:review --drift` (Layer 3) — their triggers, detection logic, and output. Normative for `@plugins/archcore/bin/session-start`, `@plugins/archcore/bin/post-tool-use`, and the CLI hook leaves they delegate to. `actualize-system.adr` records the rationale; `hooks-validation-system.spec` owns the hook execution model, the per-host output envelope, and the launcher's script-resolution behavior; `host-adapter-contract.spec` owns the adapter's routing obligations. Out of scope: structural validation (`archcore doctor`), the dashboard and `--deep` modes of `/archcore:review`, and the `archcore-auditor` agent.

## Surface

Three kinds of staleness: **code→doc drift** (source changes that invalidate documentation), **doc→doc cascade** (an update that leaves related documents behind), and **temporal staleness** (a document left in one status longer than expected). Detection runs at three depths.

| Layer | Trigger | Depth | Output | Runs in |
|---|---|---|---|---|
| 1 — passive | SessionStart | git diff heuristic | staleness line in the session recap | CLI, via `bin/session-start` |
| 2 — reactive | PostToolUse on a document mutation | relation-graph traversal | `[Archcore Cascade]` notice as `additionalContext` | CLI, via `bin/post-tool-use` |
| 3 — deep | `/archcore:review --drift` | code↔doc cross-reference plus relation graph | findings report and confirmed fixes | skill, via the actualize track |

**Layer 1.** The CLI resolves the last `.archcore/` commit, diffs non-`.archcore/` paths from there to HEAD, and matches each document's directory references against the changed set. The launcher emits the recap verbatim and adds its own advisories — install nudge, Copilot wiring, CLI-update notice — each rate-limited to once per 24 hours through a per-repository stamp file under XDG state.

**Layer 2.** `archcore hooks <host> post-tool-use` reports three finding classes: post-write validation, the cascade notice, and the precision lexicon. The cascade notice queries `.archcore/.sync-state.json` for relations whose target is the mutated document, then names the source documents of those relations.

The direction is fixed: A is the mutated target, B holds the relation into it, and B is the document reported as potentially stale. `related` relations are excluded to reduce noise. The host matcher enumerates ten tool names — create, update, and remove document, plus add and remove relation, each under the bare `mcp__archcore__` and the plugin-scoped `mcp__plugin_archcore_archcore__` prefix — except on Copilot and Cursor, whose post-mutation events accept no matcher and where the CLI self-filters on the normalized tool name.

**Layer 3.** A mode of the `review` command, activated by `--drift` or by drift phrasing such as "are any docs stale?". It loads `skills/_shared/tracks/actualize.md`, gathers (`list_documents`, `list_relations`, `git log`), analyses all three kinds, labels each finding `spec-wrong`, `code-wrong`, or `ok`, and offers a confirmed fix one document at a time.

## Normative Behavior

1. WHEN a session starts and git is available and `.archcore/` has commits, Layer 1 MUST run.
2. Layer 1 MUST NOT block session start.
3. Layer 1 MUST keep its output at or below 2 KB.
4. WHEN a document mutation named in the matcher succeeds, Layer 2 MUST fire.
5. Layer 2 MUST name only documents holding a relation into the mutated document.
6. Layer 2 MUST flag only documents connected by `implements`, `depends_on`, or `extends`.
7. Layer 2 MUST NOT block the mutation.
8. IF a host's PostToolUse event accepts no matcher, THEN the CLI MUST filter on the normalized tool name.
9. The set of mutations that trigger a cascade warning MUST NOT differ by host.
10. IF a host's hook command cannot resolve `bin/post-tool-use`, THEN the command MUST exit 0 rather than fail.
11. A post-mutation hook MUST NOT turn a resolution failure into a verdict, because it carries no verdict to deliver.
12. WHEN Layer 3 starts, it MUST verify MCP availability before any analysis.
13. Layer 3 MUST NOT modify a document without explicit user confirmation for that document.
14. Layer 3 MUST label each finding `spec-wrong`, `code-wrong`, or `ok`.
15. WHEN git is unavailable, each layer MUST degrade to a skip.
16. Each launcher MUST be POSIX shell compatible.
17. Each launcher MUST exit 0 except where it propagates the CLI's deny.

## Constraints & Invariants

- Constraint: Layer 1 MUST complete within 3 seconds, because it runs inside session start. [assumption] No host config sets a SessionStart timeout, so this is a design target rather than a host-enforced cutoff.
- Constraint: Layer 2 MUST complete within 4 seconds, which is the PostToolUse timeout in every host config.
- Constraint: neither launcher may make network access or modify a file on the hook path.
- Constraint: Layer 3 keeps its report to findings and actions rather than narrated analysis, because the report is read inside a session turn.
- Invariant: SessionStart loads context even when the staleness check fails or is skipped.
- Invariant: validation, cascade detection, and the precision lexicon are independent finding classes of one CLI leaf; none depends on another.
- Invariant: drift mode reads `.archcore/` content through MCP tools, never through direct file reads.
- Invariant: no layer modifies a document autonomously; Layer 3 requires user confirmation.

## Failure Behavior

1. IF git is unavailable, THEN Layer 1 MUST skip silently.
2. IF git is unavailable, THEN Layer 3 MUST skip code drift and still run cascade and temporal.
3. IF `.archcore/` has no commits, THEN Layer 1 MUST skip.
4. IF `.archcore/` has no commits, THEN Layer 3 MUST fall back to file modification times.
5. IF the archcore CLI is absent or older than 0.7.0, THEN Layers 1 and 2 MUST skip silently.
6. IF the relation graph is empty, THEN Layer 2 MUST produce no output.
7. IF the relation graph is empty, THEN Layer 3 MUST skip cascade analysis.
8. IF the project holds more than 100 documents, THEN Layer 3 SHOULD ask for a tag or category filter.

Item 5 is the widest silent gap in the system: on a machine whose `archcore` predates 0.7.0, both hook layers exit 0 without output and nothing distinguishes that session from a clean one.

## Conformance

1. `bin/session-start` reaches `archcore hooks <host> session-start` and its recap carries a code-drift line where one applies.
2. `bin/post-tool-use` reaches `archcore hooks <host> post-tool-use` and produces a cascade notice where one applies.
3. Every host config registers `bin/post-tool-use` on the document-mutation tools — by matcher where the host has one, by the CLI's own filtering on Copilot and Cursor.
4. `/archcore:review --drift` exists as a mode of `review`, with routing-table support and all three analyses.
5. The drift protocol lives at `skills/_shared/tracks/actualize.md`, and `@test/structure/track-goldens.bats` pins its gate records.
6. Every hook completes inside its timeout budget, bounded by `@test/unit/hook-latency.bats`.
7. No layer blocks an operation, and no layer modifies a document without user confirmation.
8. Every layer degrades to a skip when git or the CLI is unavailable.
