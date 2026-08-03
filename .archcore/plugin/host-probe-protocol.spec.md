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

This spec defines the live-session verification a host adapter MUST pass before it counts as supported, and holds the dated records of those runs. Normative for every implemented host (Claude Code, Cursor, Codex CLI, GitHub Copilot CLI) and every future one. Depended on by `host-adapter-contract.spec`, whose item 17 this document satisfies. Out of scope: everything mechanically checkable, which belongs in bats.

Static tests prove the wiring is *shaped* right. Only a live session proves the host *runs* it: that the host loads the config at all, that its matcher fires on the tool name the model actually chose, and that a deny is honored rather than merely displayed. The protocol asks those questions and no others.

## Surface

- **Probe ids** — `P0` (gate), `A`, `A-d`, `B`, `C`, `D`. One probe per shipped guard, not one guard observed three ways.
- **`@test/probe/mkprobe`** — builds a disposable probe tree; never writes inside `plugins/archcore/`.
- **Records table** — append-only, between the HTML `PROBE-RECORDS` markers below.
- **Structure tests** — `probe-hygiene.bats` (no probe residue under `plugins/`), `probe-records.bats` (every enrolled host has a well-formed row), `probe-wrapper.bats` (the harness wrapper is transparent).
- **Outcome vocabulary** — `pass` · `fail` · `n/a:<reason>` · `deferred:<reason>`; for probe D only, `fail-open-confirmed` or `fail-closed-observed`.
- **Method notation** — how the plugin reached the session (`install` or `wired`) `+` evidence grade (`log`, `report`, or `ui`).

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
7. WHEN a probe run completes, the operator MUST append one row to the records table.
8. WHEN a probe run completes, the operator MUST paste the captured evidence verbatim into the commit body.
9. WHEN a probe run captures host stdin, the operator MUST add it to `@test/fixtures/stdin/` under the host's directory, so the payload shape becomes a CI assertion.
10. The operator MUST NOT record `pass` for a probe by analogy with another host.

## Constraints & Invariants

- Constraint: the harness MUST live under `test/`, which `release.yml` strips from `main`.
- Constraint: `mkprobe` MUST copy the working tree to a temporary directory and wrap scripts **in the copy**, so no probe line ever reaches a shipped file (`hooks-validation-system.spec` conformance 13).
- Constraint: `mkprobe` MUST stamp a sentinel version into the copied manifests. A version equal to a cached install lets the host serve the cache instead of the probe tree, which is exactly how the first attempt failed (`jtbd1-phase2-hardening-delegated.plan`).
- Constraint: the probe project MUST be a **sibling** of the plugin copy, never nested inside it. `bin/session-start` walks upward for plugin manifests and would otherwise silence P0.
- Constraint: a table cell MUST NOT contain `|`. A pipe shifts every column after it, and the outcome checks would then read the wrong fields and pass on garbage. Write the method as `<how>+<grade>`, for example `install+log`.
- Invariant: a row, once written, is never edited; a re-run appends.

## Failure Behavior

1. IF the host offers no delegation surface, THEN the operator MUST record A-d as `n/a:no-delegation-surface (<host> <ver>)`.
2. IF a probe cannot be run yet, THEN the operator MUST record it as `deferred:<reason>` rather than omit it.
3. IF probe D shows the guard bypassed on timeout, THEN the operator MUST record `fail-open-confirmed`.
4. IF probe D shows the host still denying, THEN the operator MUST record `fail-closed-observed`.
5. IF the host's self-report is the only evidence for a probe, THEN the operator MUST grade it `report` rather than `log`. On Cursor `additional_context` may be dropped, so an agent's claim that it saw context proves nothing.

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

Every row starts at `deferred:not-yet-run` deliberately. Before this document existed, the contract demanded dated probe results for every host and none were recorded anywhere — a silent gap. The rows make that gap visible and testable, and each is replaced as a run happens.

## Conformance

1. `@test/structure/probe-records.bats` parses the table and fails if a host enrolled in `host-coverage-matrix.bats` has no row, if an outcome falls outside the vocabulary, or if a cell smuggles a pipe.
2. `@test/structure/probe-hygiene.bats` fails if any probe marker or harness artifact appears under `plugins/`.
3. `@test/unit/probe-wrapper.bats` fails if the wrapper alters stdout, stderr, or exit status for any fixture.
4. A host counts as supported only when its row records `pass` or a justified `n/a` for P0, A, B, and C.
