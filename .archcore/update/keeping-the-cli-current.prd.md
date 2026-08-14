---
title: "Keeping the CLI Current"
status: draft
tags:
  - "cli"
  - "integrations"
  - "telemetry"
  - "update"
---

## Vision

An Archcore user runs a current CLI without ever thinking about it. The binary keeps itself up to date from inside the one process that can afford the work, and it reports enough about that path for the project to know whether it functions — on every platform, in both the plugin and the non-plugin population.

## Problem Statement

Nothing updates the CLI unless a user types `archcore update`. The single nudge toward it is the plugin's SessionStart advisory, which reaches only plugin users and only asks. There is no measurement at all: the binary emits no telemetry today, so a stale install base and a healthy one look identical from the outside.

Three consequences follow. Users run old binaries against a plugin that assumes newer ones — the compatibility gate in `bin/cli-gte` exists because that already happens. Update failures are invisible: `@internal/update/update.go` verifies a checksum, extracts an archive under two candidate binary names, and performs a rename dance on Windows, three failure surfaces with no observability. And version adoption is unanswerable, so no release can be judged by whether anyone received it.

The install path is already measured. `cli_installed` covers a fresh `install.sh` run and, with `is_reinstall: true`, a re-run. Everything after the first install is dark.

## Goals and Success Metrics

1. A machine that starts one agent session longer than 60 s picks up a new release within ~48 h, with no user action.
2. Version adoption is readable within two releases of shipping. [assumption] the release that first carries the code emits nothing; the first data point is an upgrade away from it.
3. Every outcome of an unattended attempt is attributable: it replaced, it failed at one of five named stages, or it declined for one of three named reasons. A silent series then means the mechanism never ran, and that is itself the finding.
4. No agent session is slowed by the update path, and no JSON-RPC stream is corrupted by it.
5. A user who wants none of this sets one variable, and a user who wants no telemetry sets a different one. Neither switch turns off the other.
6. The published privacy policy states what the binary sends before any binary that sends it is downloadable.

Out of scope: package-manager channels (Homebrew, Scoop, winget) and the provenance guard that depends on them; `cli_command`, `mcp_tool_call`, `sync_*`, and every other event in `telemetry-provider-selection.rfc.md`, which stays `draft`.

## Requirements

**Updating**

1. An agent session that outlives one minute leaves the machine on the current release, without anyone doing anything.
2. The MCP server's startup, its request handling, and its shutdown are unaffected by the update path.
3. The JSON-RPC stream carries nothing the update path produced.
4. A replacement never interrupts a running process. The new version takes effect at the next launch of `archcore`.
5. Development builds, CI environments, and binaries another process has already claimed are left alone.
6. One machine attempts at most one unattended update per 24 h, however many Archcore processes run on it.
7. `ARCHCORE_NO_AUTO_UPDATE` is the single switch that disables unattended updates. A per-project `settings.json` has no say over a machine-global action.
8. `archcore update` typed by a user behaves as it does today.
9. `archcore doctor` tells a terminal-only user that a newer version exists.

**Reporting**

10. Each replacement is countable, carrying the version it came from, the version it reached, and whether a human or the mechanism started it.
11. Each failure is attributable to one of five stages: `check`, `download`, `checksum`, `extract`, `replace`.
12. Each declined attempt is attributable to one of three reasons, so an empty series means the mechanism never ran rather than "every machine was already current".
13. No error text, file path, host name, user name, or repository data leaves the machine.
14. The `--check` call the SessionStart advisory makes on every session stays silent.
15. A binary built without an injected key is inert: no request, and no identifier file created.
16. `DO_NOT_TRACK` and `ARCHCORE_TELEMETRY_OPTOUT` each suppress every event, and each leaves no trace on disk.
17. Update events join to install events with no second identifier introduced, so retention is answerable in one query.
18. A telemetry failure changes no exit code and no command output.
19. The PostHog key reaches the binary only through the release build, never through this repository.
20. `/privacy` describes both the events and the update behavior before the release that emits them is downloadable.
