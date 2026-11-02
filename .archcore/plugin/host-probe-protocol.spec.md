---
title: "Host Probe Protocol — Live-Session Verification and Dated Records"
status: accepted
tags:
  - "component:plugin"
  - "hooks"
  - "multi-host"
  - "plugin"
  - "testing"
  - "validation"
---

## Purpose & Scope

This spec defines the live-session verification a host adapter MUST pass before it counts as supported, and holds the dated records of those runs. Normative for every implemented host (Claude Code, Cursor, Codex CLI, GitHub Copilot CLI, OpenCode) and every future one. Depended on by `host-adapter-contract.spec`, whose item 17 this document satisfies. Out of scope: everything mechanically checkable, which belongs in bats.

Static tests prove the wiring is *shaped* right. Only a live session proves the host *runs* it: that the host loads the config at all, that its matcher fires on the tool name the model actually chose, and that a deny is honored rather than merely displayed. The protocol asks those questions and no others.

## Surface

- **Probe ids** — `P0` (gate), `A`, `A-d`, `B`, `C`, `D`. One probe per shipped guard behavior, not one launcher observed three ways.
- **`@plugin/test/probe/mkprobe`** — builds a disposable probe tree; never writes inside `plugins/archcore/`.
- **Records table** — append-only, between the HTML `PROBE-RECORDS` markers below.
- **Structure tests** — `probe-hygiene.bats` (no probe residue under `plugins/`), `probe-records.bats` (every enrolled host has a well-formed row), `probe-wrapper.bats` (the harness wrapper is transparent).
- **Outcome vocabulary** — `pass` · `fail` · `n/a:<reason>` · `deferred:<reason>`; for probe D only, `fail-open-confirmed` or `fail-closed-observed`.
- **Method notation** — how the plugin reached the session (`install` or `wired`) `+` evidence grade (`log`, `report`, or `ui`).

| Probe | Guard exercised | Action in a live session |
|---|---|---|
| P0 | `bin/session-start` | start a session; its output must prove the **probe tree** was loaded |
| A | `bin/pre-tool-use` — code-alignment advisory | main session writes `src/probe/alpha.ts` |
| A-d | same, via delegation | a sub-agent writes `src/probe/beta.ts` |
| B | `bin/pre-tool-use` — write guard | write a probe document under `.archcore/probe/`; the host must refuse and show the reason |
| C | `bin/post-tool-use` | `update_document` through MCP |
| D | host timeout path | repeat B with a slowed guard |

Since v0.7.0 (`cli-owns-layers-4-5.adr`) each launcher delegates to `archcore hooks <host> <leaf>`, so a probe exercises the launcher, the host wiring, and the CLI leaf as one path. Probes A and B share `bin/pre-tool-use` and stay distinct probes because they exercise two decisions: advise on a source edit, refuse an `.archcore/` write. Item 11 exists because both launchers exit 0 silently against a pre-0.7 CLI, which makes every probe on such a session indistinguishable from a passing one. Failure Behavior item 4 grades a self-report down because Cursor may drop `additional_context`, so an agent's claim that it saw context proves nothing.

## Normative Behavior

1. WHEN a probe run begins, the operator MUST establish P0 before issuing any other prompt.
2. IF P0 does not prove the probe tree was loaded, THEN the operator MUST NOT record a row for that run.
3. WHEN probe A runs, the operator MUST observe injected context naming the edited path.
4. WHEN probe A-d runs, the operator MUST observe the same guard firing on the delegated call.
5. WHEN probe B runs, the host MUST refuse the write and surface the guard's reason text.
6. WHEN probe C runs, the operator MUST observe validation output for the MCP call.
7. WHEN a probe run completes, the operator MUST append one row to the records table.
8. WHEN a probe run completes, the operator MUST paste the captured evidence verbatim into the commit body.
9. WHEN a probe run captures host stdin, the operator MUST add it to `@plugin/test/fixtures/stdin/` under the host's directory.
10. The operator MUST NOT record `pass` for a probe by analogy with another host.
11. IF the `archcore` CLI on PATH is older than 0.7.0, THEN the operator MUST NOT record a row.

## Constraints & Invariants

- Constraint: the harness MUST live under `test/`, which `release.yml` strips from `main`.
- Constraint: `mkprobe` MUST copy the working tree to a temporary directory and wrap scripts **in the copy**, so no probe line ever reaches a shipped file (`hooks-validation-system.spec` conformance 13).
- Constraint: `mkprobe` MUST stamp a sentinel version into the copied manifests. A version equal to a cached install lets the host serve the cache instead of the probe tree, which is exactly how the first attempt failed (`jtbd1-phase2-hardening-delegated.plan`).
- Constraint: `mkprobe` MUST write a settings file the current CLI parser accepts. A file missing `sync` opens every recap with `invalid .archcore/settings.json`, which turns P0 into a judgement call about whether the tree loaded or the project is broken.
- Constraint: the probe project MUST be a **sibling** of the plugin copy, never nested inside it. `bin/session-start` walks upward for plugin manifests and would otherwise silence P0.
- Constraint: a table cell MUST NOT contain `|`. A pipe shifts every column after it, and the outcome checks would then read the wrong fields and pass on garbage. Write the method as `<how>+<grade>`, for example `install+log`.
- Invariant: a row, once written, is never edited; a re-run appends.

## Failure Behavior

1. IF the host offers no delegation surface, THEN the operator MUST record A-d as `n/a:no-delegation-surface (<host> <ver>)`.
2. IF a probe cannot be run yet, THEN the operator MUST record it as `deferred:<reason>` rather than omit it.
3. WHEN probe D runs, the operator MUST record `fail-open-confirmed` if the guard was bypassed, `fail-closed-observed` otherwise.
4. IF the host's self-report is the only evidence for a probe, THEN the operator MUST grade it `report` rather than `log`.

## Records

Evidence pointer is `<commit-sha>:<probe-id>`; the log itself lives in that commit's body.

<!-- PROBE-RECORDS:BEGIN -->

| Date | Host | Host ver | Plugin ver | Method | P0 | A | A-d | B | C | D | Evidence |
|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-08-14 | claude-code | 2.1.232 | 0.0.0-probe | wired+log | pass | pass | pass | pass | pass | fail-open-confirmed | pending-commit |
| 2026-08-14 | cursor | — | — | — | deferred:not-installed-on-probe-machine | deferred:not-installed-on-probe-machine | deferred:not-installed-on-probe-machine | deferred:not-installed-on-probe-machine | deferred:not-installed-on-probe-machine | deferred:not-installed-on-probe-machine | — |
| 2026-08-14 | codex | 0.147.0 | 0.0.0-probe | install+log | pass | deferred:isolated-home-unauthenticated | deferred:isolated-home-unauthenticated | deferred:isolated-home-unauthenticated | deferred:isolated-home-unauthenticated | deferred:isolated-home-unauthenticated | pending-commit |
| — | copilot | 1.0.76 | — | — | deferred:needs-authenticated-home | deferred:needs-authenticated-home | deferred:needs-authenticated-home | deferred:needs-authenticated-home | deferred:needs-authenticated-home | deferred:needs-authenticated-home | — |
| — | opencode | 1.18.34 | — | — | deferred:bridge-not-released | deferred:bridge-not-released | deferred:bridge-not-released | deferred:bridge-not-released | deferred:bridge-not-released | deferred:bridge-not-released | — |
| 2026-10-30 | claude-code | 2.1.314 | 0.0.0-probe | wired+log | pass | pass | deferred:not-run-this-pass | pass | pass | deferred:not-run-this-pass | pending-commit |
| 2026-10-30 | codex | 0.166.0 | 0.0.0-probe | install+log | pass | fail | deferred:not-run-this-pass | pass | pass | deferred:not-run-this-pass | pending-commit |
| 2026-10-30 | opencode | 2.1.4 | local-build | wired+log | pass | pass | deferred:not-run-this-pass | pass | pass | deferred:not-run-this-pass | pending-commit |
| 2026-11-02 | claude-code | 2.1.316 | 0.0.0-probe | wired+log | pass | pass | pass | pass | pass | fail-open-confirmed | pending-commit |
| 2026-11-02 | codex | 0.166.0 | 0.0.0-probe | install+log | pass | pass | pass | pass | pass | fail-open-confirmed | pending-commit |
| 2026-11-02 | opencode | 2.1.4 | 0.0.0-probe | wired+log | pass | pass | pass | pass | pass | fail-open-confirmed | pending-commit |
<!-- PROBE-RECORDS:END -->

**Run of 2026-08-14.** Claude Code reached the tree through `--plugin-dir`, so its method is `wired`, not `install`; probe C additionally needed `--mcp-config` and `--allowedTools`, because headless `-p` exposes no plugin MCP server and gates MCP tools behind a prompt no non-interactive session can answer — the accommodation makes the tools reachable and changes nothing about the hook under test. **Probe D is the finding of this run: with a 5-second delay injected into the wrapper against a 2-second `timeout`, the write into `.archcore/` landed with no error, no hook output, and no timeout message** — Claude Code fails open on a PreToolUse timeout exactly as Copilot does, and silently, so the latency budget is a correctness concern on this host too. Codex reached P0 through a real marketplace install of the sentinel version and logged host `codex`; its remaining probes stopped at an unauthenticated isolated `CODEX_HOME`, a property of the probe machine rather than of the adapter. Cursor is not installed here, and Copilot has no isolated-home path that authenticates.

**Run of 2026-10-30.** The three rows above ran archcore CLI v0.15.0-dev on the PATH of each host, with the project-b sibling next to the probe project for the explicit-target-project probes of `call-root-resolution.spec`. Claude Code loaded the tree through `--plugin-dir` and the init event named `archcore@inline` at `0.0.0-probe`. Codex installed the sentinel version through a local marketplace in an isolated `HOME`; its plugin hooks ran only with `--dangerously-bypass-hook-trust`, which is the hook-trust consent recorded in `codex-host-verification.rnd`, not an adapter defect. OpenCode loaded the bridge from an isolated `XDG_CACHE_HOME` that held the published package layout with this repository's `dist/`, `bin/`, and `skills/`; a `file://` plugin entry did not load on 2.1.4, so the plugin version cell names the build, not a manifest version.

Findings, each with a fix and a test in the same change:

1. Codex 0.166.0 sends an `apply_patch` call as `tool_input.command`. The write guard read no such key, so a patch into `.archcore/` landed on Codex. Probe B was `fail` before the fix and `pass` after it. The captured payload replaced the hand-written fixture at `@plugin/test/fixtures/stdin/codex/apply-patch-archcore.json`.
2. A write into the session's own `.archcore/` spelled through a symlink of the root — `/tmp` against the `/private/tmp` working directory — was allowed on every host, CLI 0.14.4 included. `docs.RelativeToBase` now compares again with symlinks evaluated.
3. The OpenCode bridge sent no session id with tool events, so the project-touch advisory repeated on every edit there. The bridge now passes `session_id`.

Probe A on Codex is `fail`: Codex edits every file with `apply_patch`, and the code-alignment advisory reads only a `file_path` target, so no context reaches a patch edit. The fix is open for an owner decision on which targets of a multi-file patch feed the advisory. Decided 2026-11-02: the first patch target feeds the advisory (hook-runtime.spec behavior 18), pinned by `TestPreToolUse_CodeAlignmentFromThePatchsFirstTarget`; The live re-probe of 2026-11-02 passed: the Codex transcript carried `[Archcore Context]` for the patched file.

Explicit-target-project probes, outside the protocol columns: from a session in the probe project, `create_document` and `update_document` with `project` wrote under project-b; `search_documents` with an absolute `for_path` in project-b returned its rule; a direct write into project-b's `.archcore/` was denied with `project="<root>"`; a source edit in project-b carried one `[Archcore Project]` note per session; `post-tool-use` validated the project-b document. All passed on the three hosts. In a folder without `.archcore/`, Claude Code and Codex asked which project to use and created nothing; OpenCode's non-interactive `run` cancelled the question and told the model to continue, and the model wrote to an existing project through `project` — the session folder stayed without `.archcore/` on all three hosts.

**Run of 2026-11-02.** The three rows above ran archcore CLI v0.15.0-dev with the same setup as the run of 2026-10-30, and completed every probe column on each host.

1. Codex Probe A passed after the first-patch-target change: a developer message in the transcript named `src/probe/gamma.ts` under `[Archcore Context]`.
2. Probe D failed open on all three hosts. A 5-second delay in the `pre-tool-use` wrapper let the write into `.archcore/` land. On OpenCode the bridge stopped the guard at its 2000 ms budget: the guard logged its start, and the write completed 2.25 s after the call began.
3. OpenCode Probe C passed through Code Mode: the `[Archcore Precision]` advice reached the model in the system context of the next request, not in the tool result. This is the designed path of the v2 bridge.
4. Codex `exec` needed per-tool `approval_mode = "approve"` in the isolated `config.toml` to call MCP tools; this is a host setting, not an adapter defect.

The explicit-target-project probes repeated with the same outcomes as on 2026-10-30. In X4, Claude Code's `acceptEdits` mode refused a write outside the working directory after the plugin allowed the write and emitted the `[Archcore Project]` note; the refusal belongs to the host.

## Conformance

1. `@plugin/test/structure/probe-records.bats` parses the table and fails if a host enrolled in `host-coverage-matrix.bats` has no row, if an outcome falls outside the vocabulary, or if a cell smuggles a pipe.
2. `@plugin/test/structure/probe-hygiene.bats` fails if any probe marker or harness artifact appears under `plugins/`.
3. `@plugin/test/unit/probe-wrapper.bats` fails if the wrapper alters stdout, stderr, or exit status for any fixture.
4. A host counts as supported only when its row records `pass` or a justified `n/a` for P0, A, B, and C.
