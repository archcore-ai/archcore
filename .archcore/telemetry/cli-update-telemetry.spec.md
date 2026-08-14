---
title: "CLI Update Telemetry Contract"
status: draft
tags:
  - "cli"
  - "telemetry"
  - "update"
---

## Purpose & Scope

This spec is normative for the telemetry emitted by the CLI's update path — `@cmd/update.go` and the unattended policy in `internal/update`. Dependents: `src/lib/analytics/events.ts` in `archcore-ai/landing`, which declares the event names under `ExternalAnalyticsEventMap`; PostHog queries that join CLI events to installer events; and `@install.sh` and `@install.ps1`, which share the identifier file this contract reads and writes.

Out of scope: the installer beacon's own events and payload, and every CLI event other than the three named here.

## Surface

- Command surface: `archcore update` and `archcore update --check` — `@cmd/update.go`. The unattended path carries no flag in phase 1; its only caller is the MCP trigger.
- Sender: `internal/telemetry` [planned] — a package-level key variable, populated by `-X` ldflags in `@.goreleaser.yaml` at release, empty in every other build.
- Endpoint: `POST https://ph.archcore.ai/i/v0/e/`, PostHog capture payload — the endpoint `send_event()` in `@install.sh` already uses.
- Events: `cli_updated`, `cli_update_failed`, `cli_update_skipped`.
- Identifier: `${XDG_STATE_HOME:-$HOME/.local/state}/archcore/install-id`, the path `install_id_path()` in `@install.sh` writes and `updateCheckCachePath()` in `@cmd/update.go` mirrors.
- Stage categories on `cli_update_failed`, derived from the failure points of `@internal/update/update.go`: `check`, `download`, `checksum`, `extract`, `replace`.
- Reason categories on `cli_update_skipped`: `optout`, `current`, `not_writable`.
- Common properties: `$lib` (`archcore-cli`), `$lib_version` (the running version), `source` (`cli`), `os`, `arch`, `ci`, `trigger` (`manual` or `auto`).

## Normative Behavior

1. WHEN `Apply` returns without error, the CLI MUST send exactly one `cli_updated` event.
2. WHEN the CLI sends `cli_updated`, the CLI MUST set `from_version` to the running binary's version and `to_version` to the resolved latest tag.
3. WHEN `CheckLatest` or `Apply` returns an error, the CLI MUST send exactly one `cli_update_failed` event carrying `stage` set to one of the five stage categories.
4. WHEN an unattended attempt stops at a condition the update policy marks reportable, the CLI MUST send exactly one `cli_update_skipped` event.
5. WHEN the CLI sends `cli_update_skipped`, the CLI MUST set `reason` to `optout`, `current`, or `not_writable`.
6. WHEN a user typed `archcore update` and the running version equals the resolved latest tag, the CLI MUST NOT send an event.
7. WHEN the user runs `archcore update --check`, the CLI MUST NOT send an event.
8. The CLI MUST resolve `distinct_id` from the `install-id` file.
9. IF the `install-id` file is absent or unreadable, THEN the CLI MUST write a new identifier of 32 lowercase hexadecimal characters to that path and use it.
10. The CLI MUST set `trigger` to `manual` on an invocation a user typed.
11. The CLI MUST set `trigger` to `auto` on an invocation an orchestrator made.
12. The CLI MUST NOT send `cli_update_skipped` on a `manual` invocation.
13. WHEN an event is accepted by the endpoint on a `manual` invocation, the CLI MUST print one disclosure line naming an opt-out variable and `https://archcore.ai/privacy`.
14. WHILE the endpoint has not accepted a payload, the CLI MUST NOT print the disclosure line.
15. WHILE the invocation is `auto`, the CLI MUST NOT print the disclosure line.
16. WHILE the CLI emits `auto` events, the published privacy page MUST carry the disclosure the runtime cannot print.

## Constraints & Invariants

- Constraint: the CLI MUST NOT send when the key variable lacks the `phc_` prefix. Rationale: a `go build`, a `go install`, a fork, and a CI build are inert by construction, which is the property the installer beacon obtains from deploy-time substitution.
- Constraint: the CLI MUST NOT send when `DO_NOT_TRACK` or `ARCHCORE_TELEMETRY_OPTOUT` holds any value other than empty or `0`.
- Constraint: the CLI MUST evaluate all three guards before it reads or creates the identifier file. Rationale: an opt-out leaves no trace on disk.
- Constraint: `cli_update_skipped` MUST fall inside the update policy's claim window. Rationale: unbounded, the series would count MCP server starts rather than machines, and a host that restarts its servers often would outweigh every other machine.
- Constraint: the CLI MUST NOT send `cli_update_skipped` for a development build, for a CI environment, or for a claim another process holds. Rationale: a development build carries no key, CI runners are ephemeral and mint a fresh identifier per run, and a held claim fires once per concurrent process rather than once per machine.
- Constraint: the CLI MUST bound the request by a connect timeout and a total timeout. [assumption] 2 s and 3 s, matching the `curl` flags in `@install.sh`.
- Constraint: the CLI MUST NOT transmit an error message, a file path, a directory name, a user name, a host name, or repository data.
- Constraint: `settings.json` MUST NOT act as an opt-out surface for these events. Rationale: `archcore update` runs outside any project.
- Invariant: one `archcore update` invocation produces at most one event.
- Invariant: one unattended attempt produces at most one event.
- Invariant: `trigger` separates evidence of user intent from evidence that a mechanism ran. A query that measures adoption filters on `manual`.
- Invariant: the three events partition every outcome of an unattended attempt — it replaced, it tried and failed, or it declined for a named reason. A silent series means the mechanism never ran.
- Invariant: the identifier file format is identical across `@install.sh`, `@install.ps1`, and the CLI, so one machine resolves to one `distinct_id`.

## Failure Behavior

1. IF the request times out, fails DNS resolution, or returns a non-2xx status, THEN the CLI MUST discard the outcome and continue.
2. IF telemetry fails for any reason, THEN the CLI MUST NOT change the exit code of `archcore update`.
3. IF telemetry fails for any reason, THEN the CLI MUST NOT print an error, a warning, or a stack trace.
4. IF the identifier cannot be read and cannot be created, THEN the CLI MUST skip the event.
5. IF the update fails before the latest tag is resolved, THEN the CLI MUST omit `to_version` rather than send a placeholder.
6. IF an unattended attempt declines to update, THEN the CLI MUST NOT send `cli_update_failed`. A refusal is not a failure; it is `cli_update_skipped` or it is nothing.

## Conformance

An implementation is conformant when it satisfies behaviors 1–16, holds the three guards in the stated order, preserves the one-event-per-invocation and shared-identifier invariants, grades every event with `trigger`, bounds `cli_update_skipped` by the claim window, and degrades per the failure rules.

Given a binary built with `go build` and no ldflags key, when the user runs `archcore update` and the update succeeds, then no network request is made, no identifier file is created, and no disclosure line is printed.
