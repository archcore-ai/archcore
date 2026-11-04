---
title: "SessionStart Context Output Contract"
status: accepted
tags:
  - "cli"
  - "component:cli"
  - "integrations"
  - "performance"
---

## Purpose & Scope

This spec defines what `archcore hooks <host> session-start` emits: the payload
text and the invariants that keep it bounded and host-safe. Normative for
`buildSessionContext` (@cli/cmd/hooks_common.go) and every per-host writer that
wraps its result.

Out of scope: the per-host JSON envelope shapes, the dedup stamp, the
document model, and the content of the `GLOBALS` block, which
@.archcore/globals/session-globals-disclosure.spec.md owns.

## Surface

One context string, embedded by the caller in a host-specific JSON envelope.

Sections, in order: the header; the context address; the MCP tool line; the
invalid-settings warning when present; `CORPUS`; `BRANCH` when resolvable;
`GLOBALS` when sources are declared; `IN PROGRESS`; `RECENTLY ACCEPTED`; the
staleness advisory when due; the managed-block advisory when due; `EXISTING
TAGS`; `DOCUMENT RELATIONS`.

## Normative Behavior

1. The context string MUST hold at most 24 document lines across `IN PROGRESS`
   and `RECENTLY ACCEPTED` combined.
2. WHEN a block is truncated, the builder MUST state how many documents were
   omitted and name the `list_documents` filter that returns them.
3. The builder MUST exclude a document with status `rejected` from both blocks.
4. WHEN the corpus holds at least one `rejected` document, the builder MUST
   report their count in `CORPUS`.
5. The builder MUST limit `RECENTLY ACCEPTED` to documents modified within the
   last 30 days.
6. WHEN the staleness advisory has fired for a project within 24 hours, the
   builder MUST omit it.
7. The staleness advisory MUST NOT name a command the CLI does not own.
8. The builder MUST reserve up to 6 budget lines for `RECENTLY ACCEPTED`.
9. The builder MUST give `IN PROGRESS` every line that reserve leaves.
10. WHEN `settings.json` declares at least one global source, the builder MUST
    render the `GLOBALS` block per
    @.archcore/globals/session-globals-disclosure.spec.md.
11. WHEN the `GLOBALS` block renders, the builder MUST label the `CORPUS` count
    "local documents".
12. The builder MUST emit the context address directly after the header line.
13. The builder MUST NOT close the context with a pointer to the MCP server
    instructions.
14. WHEN a managed block in the project differs from the block the CLI writes,
    the builder MUST emit one `[Archcore Instructions]` line per finding.
15. WHEN the managed-block advisory fired for a project within 24 hours, the
    builder MUST omit it.
16. WHEN the session folder holds no `.archcore/`, the builder MUST NOT carry
    `init_project` or `create_document` advice.
17. WHEN the session folder holds no `.archcore/`, the builder MUST print
    `NO PROJECT HERE`, then `PROJECT ABOVE` with the nearest ancestor that
    holds `.archcore/`, then `PROJECTS BELOW` with the roots a bounded scan
    finds.
18. WHEN a bound cuts the scan, the builder MUST add "(the search stopped at its
    bound)" to the recap.

## Constraints & Invariants

- Invariant: output length is a function of the line budget, not of corpus size.
  Between a 300-document and a 3000-document corpus the length MAY differ only
  by the width of the `CORPUS` counters. The `GLOBALS` block carries its own
  ceilings for the same reason.
- Invariant: this function owns the text inside the envelope and never the
  envelope itself. The plugin splices its own advisories into the same JSON
  document on Copilot, where a failed parse discards the entire payload — a
  change to the wrapper would break that splice with no error anywhere.
- Constraint: `CORPUS` omits any status counted zero, not only `rejected`.
- Constraint: every dedup scope keeps its own stamp directory. A sweep expires
  everything older than its own window, so a shared directory would let the
  10-minute session scope erase the 24-hour staleness budget.
- Constraint: no global document line appears in this context. The `GLOBALS`
  block reports per-source metadata only; global content stays behind the MCP
  read tools (@.archcore/globals/session-globals-disclosure.spec.md).
- Constraint: the project scan of clause 17 descends at most 3 levels below the
  session folder, reads at most 2000 directories, and shows at most 10 roots.
  It has no time bound of its own. The values are chosen, not measured
  [assumption]. `inventoryDepth` and `inventoryDirs` in
  @cli/cmd/hook_project_root.go hold them.
- Constraint: the plugin's `bin/session-start` hands the no-project case to the
  CLI only for archcore CLI 0.15.0 or newer. An older CLI receives a static
  line that tells the agent to ask which project to use.
- Constraint: the project-touch advisory (a source-file edit under another
  project root) fires once per session, root, and host. It claims its stamp in
  the directory `project-touch-stamps` with a 24-hour window; the window is a
  chosen value [assumption].
- Constraint: the OpenCode bridge hands the arguments of `write`, `edit`,
  `apply_patch`, and `patch` to the pre-tool-use leaf as `tool_input`
  (`PRE_MUTATION` in @plugin/plugins/opencode/src/bridge.ts). The host probe run
  of 2026-10-30 on OpenCode 2.1.4 passed the write-guard and project-touch
  probes (@.archcore/plugin/host-probe-protocol.spec.md).

## Failure Behavior

1. IF a declared global source is unreadable, THEN the builder MUST degrade to a
   local-only scan, render the source's warning inside the `GLOBALS` block, and
   keep every local document.
2. IF `settings.json` is present but invalid, THEN the builder MUST emit a
   warning naming the failure, MUST NOT render a `GLOBALS` block, and MUST
   continue.
3. IF the directory is not a git working tree, or git is absent, or HEAD is
   detached, THEN the builder MUST omit the `BRANCH` line rather than render a
   placeholder.
4. IF the corpus is empty, THEN the builder MUST say so in `CORPUS` and MUST
   omit both recap blocks.
5. IF the staleness correlation finds no affected document, THEN the builder
   MUST omit the advisory and MUST NOT consume the 24-hour budget.

## Conformance

The builder is conformant when it satisfies behaviors 1–18, holds every
invariant, and degrades per the failure rules. The budget invariant is verified
against synthetic corpora of 300 and 3000 documents; the `GLOBALS` behaviors
are verified by @cli/cmd/hooks_globals_block_test.go.
