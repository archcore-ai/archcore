---
title: "Host Probe Protocol — Live-Session Verification and Dated Records"
status: accepted
tags:
  - "hooks"
  - "multi-host"
  - "plugin"
  - "testing"
  - "validation"
---

## Purpose & Scope

Defines the live-session verification a host adapter MUST pass before it counts as supported, and holds the dated records of those runs. Covers every implemented host (Claude Code, Cursor, Codex CLI, GitHub Copilot CLI) and every future one. This document satisfies item 6 of `host-adapter-contract.spec`.

Static tests prove the wiring is *shaped* right. Only a live session proves the host *runs* it: that it loads the config at all, that its matcher fires on the tool name the model actually chose, and that a deny is honored rather than merely displayed. Those questions are what this protocol asks, and nothing else — everything mechanically checkable belongs in bats.

## Surface

- **Probe ids** — `P0` (gate), `A`, `A-d`, `B`, `C`, `D`. One probe per shipped guard, not one guard observed three ways.
- **`test/probe/mkprobe`** — builds a disposable probe tree; never writes inside `plugins/archcore/`.
- **Records table** — append-only, between the HTML `PROBE-RECORDS` markers below.
- **Structure tests** — `probe-hygiene.bats` (no probe residue under `plugins/`), `probe-records.bats` (every enrolled host has a well-formed row), `probe-wrapper.bats` (the harness wrapper is transparent).

| Probe | Guard exercised | Action in a live session |
|---|---|---|
| P0 | `bin/session-start` | start a session; its output must prove the **probe tree** was loaded |
| A | `bin/check-code-alignment` | main session writes `src/probe/alpha.ts` |
| A-d | same, via delegation | a sub-agent writes `src/probe/beta.ts` |
| B | `bin/check-archcore-write` | write `.archcore/probe/p.adr.md`; the host must refuse and show the reason |
| C | `bin/validate-archcore` | `update_document` through MCP |
| D | host timeout path | repeat B with a slowed guard |

## Normative Behavior

1. WHEN a probe run begins, the operator MUST establish P0 before issuing any other prompt.
2. IF P0 does not prove the probe tree was loaded, THEN the operator MUST NOT record a row for that run.
3. WHEN probe A runs, the operator MUST observe injected context naming the edited path.
4. WHEN probe A-d runs, the operator MUST observe the same guard firing on the delegated call.
5. WHEN probe B runs, the host MUST refuse the write and surface the guard's reason text.
6. WHEN probe C runs, the operator MUST observe validation output for the MCP call.
7. WHEN a probe run completes, the operator MUST append one row to the records table and MUST paste the captured evidence verbatim into the commit body.
8. WHEN a probe run captures host stdin, the operator MUST add it to `test/fixtures/stdin/<host>/` so the payload shape becomes a CI assertion.
9. The operator MUST NOT record `pass` for a probe by analogy with another host.

## Constraints & Invariants

- The harness MUST live under `test/`, which `release.yml` strips from `main`.
- `mkprobe` MUST copy the working tree to a temporary directory and wrap scripts **in the copy**; no probe line ever reaches a shipped file (`hooks-validation-system.spec` conformance 13).
- `mkprobe` MUST stamp a sentinel version into the copied manifests: a version equal to a cached install lets the host serve the cache instead of the probe tree, which is exactly how the first attempt failed (`jtbd1-phase2-hardening-delegated.plan`).
- The probe project MUST be a **sibling** of the plugin copy, never nested inside it — `bin/session-start` walks upward for plugin manifests and would silence P0.
- A table cell MUST NOT contain `|`; a pipe shifts every column after it, and the outcome checks would then read the wrong fields and pass on garbage. Write the method as `<how>+<grade>`, e.g. `install+log`.
- A row, once written, MUST NOT be edited; a re-run appends.

## Failure Behavior

- IF the host offers no delegation surface, THEN A-d MUST be recorded `n/a:no-delegation-surface (<host> <ver>)`.
- IF a probe cannot be run yet, THEN it MUST be recorded `deferred:<reason>` rather than omitted.
- IF probe D shows the guard bypassed on timeout, THEN it MUST be recorded `fail-open-confirmed`; if the host still denies, `fail-closed-observed`.
- IF the host's self-report is the only evidence for a probe, THEN the grade MUST be `report`, not `log` — on Cursor `additional_context` may be dropped, so an agent's claim that it saw context proves nothing.

Outcome vocabulary: `pass` · `fail` · `n/a:<reason>` · `deferred:<reason>` · D only: `fail-open-confirmed` | `fail-closed-observed`. Method notation: how the plugin reached the session (`install` or `wired`) `+` evidence grade (`log`, `report` or `ui`).

## Records

Evidence pointer is `<commit-sha>:<probe-id>`; the log itself lives in that commit's body.

<!-- PROBE-RECORDS:BEGIN -->

| Date | Host | Host ver | Plugin ver | Method | P0 | A | A-d | B | C | D | Evidence |
|---|---|---|---|---|---|---|---|---|---|---|---|
| — | claude-code | — | — | — | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | — |
| — | cursor | — | — | — | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | — |
| — | codex | — | — | — | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | — |
| — | copilot | — | — | — | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | deferred:not-yet-run | — |

<!-- PROBE-RECORDS:END -->

Every row starts `deferred:not-yet-run` on purpose. Before this document existed, the contract demanded dated probe results for every host and none were recorded anywhere — a silent gap. Rows make it visible and testable, and they are replaced as runs happen.

## Conformance

1. `test/structure/probe-records.bats` parses the table and fails if a host enrolled in `host-coverage-matrix.bats` has no row, if an outcome falls outside the vocabulary, or if a cell smuggles a pipe.
2. `test/structure/probe-hygiene.bats` fails if any probe marker or harness artifact appears under `plugins/`.
3. `test/unit/probe-wrapper.bats` fails if the wrapper alters stdout, stderr, or exit status for any fixture.
4. A host counts as supported only when its row records `pass` or a justified `n/a` for P0, A, B and C.
