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

An Archcore user runs a current CLI without ever thinking about it. The binary keeps itself up to date from inside the one process that can afford the work, and it reports enough about that path for the project to know whether it functions — on every platform, in both the plugin and the non-plugin population. The plugin rides the same rails: selecting a host delivers everything that host needs, and one command keeps both entry points current.

## Problem Statement

Nothing updates the CLI unless a user types `archcore update`. The single nudge toward it is the plugin's SessionStart advisory, which reaches only plugin users and only asks. There is no measurement at all: the binary emits no telemetry today, so a stale install base and a healthy one look identical from the outside.

Three consequences follow. Users run old binaries against a plugin that assumes newer ones — the compatibility gate in `bin/cli-gte` exists because that already happens. The mismatch also runs the other way: nothing updates an installed plugin, so an aging plugin meets every newer CLI, and the duplicated-hook warning in `plugin-cli-compatibility.rule.md` exists because that side already happens too. Update failures are invisible: `@internal/update/update.go` verifies a checksum, extracts an archive under two candidate binary names, and performs a rename dance on Windows, three failure surfaces with no observability. And version adoption is unanswerable, so no release can be judged by whether anyone received it.

The install side is split the same way. `install.sh` delivers the CLI and reports `cli_installed`; the plugin is a separate, per-host manual install that a CLI-first user has no reason to know exists. Everything after the first install is dark.

## Goals and Success Metrics

1. A machine that starts one agent session longer than 60 s picks up a new release within ~48 h, with no user action.
2. Version adoption is readable within two releases of shipping. [assumption] the release that first carries the code emits nothing; the first data point is an upgrade away from it.
3. Every outcome of an unattended attempt is attributable: it replaced, it failed at one of five named stages, or it declined for one of two named reasons. A silent series then means the mechanism never ran, and that is itself the finding.
4. No agent session is slowed by the update path, and no JSON-RPC stream is corrupted by it.
5. Unattended update has no opt-out: no variable, flag, or setting turns it off. `DO_NOT_TRACK` and `ARCHCORE_TELEMETRY_OPTOUT` govern telemetry only, and neither one disables an update.
6. The published privacy policy states what the binary sends — and that unattended update cannot be switched off — before any binary that behaves that way is downloadable.
7. One `archcore update` leaves the plugin as current as the binary on every host where it is installed, so the plugin population ages no faster than the binary population.
8. A user who selects a plugin-capable host in `archcore init` ends up with the plugin installed without knowing it was a separate artifact — the install story is one selection, not one install per surface.

Out of scope: package-manager channels (Homebrew, Scoop, winget) and the install provenance receipt that depends on them — the receipt activates with the first package-manager channel; release signing — decided against 2026-08-15, the trust anchor stays GitHub account and pipeline security with the health probe, the official-build marker, and the non-inertness gate as the compensating controls; `cli_command`, `mcp_tool_call`, `sync_*`, and every other event in `telemetry-provider-selection.rfc.md`, which stays `draft`.

## Requirements

**Updating**

1. An agent session that outlives one minute leaves the machine on the current release, without anyone doing anything.
2. The MCP server's startup, its request handling, and its shutdown are unaffected by the update path.
3. The JSON-RPC stream carries nothing the update path produced.
4. A replacement never interrupts a running process. The new version takes effect at the next launch of `archcore`.
5. Marker-less builds — forks and repackaged binaries — development builds, CI environments, and binaries another process has already claimed are left alone.
6. One machine attempts at most one unattended update per 24 h, however many Archcore processes run on it.
7. Unattended update has no opt-out surface: no environment variable, no flag, and no setting disables it. A per-project `settings.json` has no say over a machine-global action. A root-owned install directory is the supported operator answer for machines that must not self-update.
8. `archcore update` typed by a user updates the binary exactly as it does today, and then updates the plugin per host tier. The plugin step never changes the command's exit code.
9. `archcore doctor` tells a terminal-only user that a newer version exists.
10. A replacement lands only after the downloaded binary passed a health probe; a fleet cannot be bricked by a release that cannot start.

**Reporting**

11. Each replacement is countable, carrying the version it came from, the version it reached, and whether a human or the mechanism started it.
12. Each failure is attributable to one of five stages: `check`, `download`, `checksum`, `extract`, `replace`.
13. Each declined attempt is attributable to one of two reasons, `current` or `not_writable`, so an empty series means the mechanism never ran rather than "every machine was already current".
14. No error text, file path, host name, user name, or repository data leaves the machine.
15. The `--check` call the SessionStart advisory makes on every session stays silent.
16. A binary built without an injected key is inert: no request, and no identifier file created. The release pipeline proves the published artifact is not inert.
17. `DO_NOT_TRACK` and `ARCHCORE_TELEMETRY_OPTOUT` each suppress every event, and each leaves no trace on disk.
18. Update events join to install events with no second identifier introduced, so retention is answerable in one query.
19. A telemetry failure changes no exit code and no command output.
20. The PostHog key reaches the binary only through the release build, never through this repository.
21. `/privacy` describes both the events and the update behavior before the release that emits them is downloadable.

**Plugin update**

22. The plugin step runs only inside a user-typed `archcore update`: after the binary phase, whether that phase replaced the binary or found it current — and never when the binary phase failed or `--check` was passed. It covers the hosts a plugin ships for: Claude Code, GitHub Copilot, Codex CLI, and Cursor.
23. The step reads host evidence and, inside the plugin surface, the host's own plugin state; nothing outside that surface changes on what it finds (`plugin-cli-compatibility.rule.md`, clause 3).
24. With the host CLI on `PATH`, the step asks the host's own plugin listing and runs the update command only for a listed plugin. With only the host's registry listing the plugin, it prints the exact command — for Cursor, the UI instruction. Otherwise it skips silently.
25. A nonzero exit or a timeout prints the exact command for the user to run and changes no exit code of `archcore update`.
26. A user who never installed the plugin sees no plugin output from `archcore update` and pays no mutating host command.
27. The plugin step emits no telemetry event; an `archcore update` invocation reports the binary phase only.
28. The unattended update path and the MCP background trigger never reach the plugin step.

**Plugin delivery**

29. Selecting a host in `archcore init` delivers everything that host needs — hooks, MCP config, and the plugin. The selection screen says the plugin comes with the selection; deselecting the host is the opt-out, and no second prompt exists.
30. A rerun of `archcore init` over an installed plugin reports it and changes nothing.
31. Non-interactive init installs a plugin only for a host named with `--agent`, which carries the consent the way it already does for wiring; `--yes` alone and CI environments print the per-host commands instead of running them.
32. The selection screen names Codex CLI and GitHub Copilot installs as machine-level, not project-level.
33. Claude Code delivery defaults to user scope and enables marketplace auto-update (`autoUpdate: true`), so the host keeps the plugin current on its own; `--scope project` is the opt-in that delivers the declaration to the whole team, said at write time.
34. A delivery failure prints the exact command, never fails `archcore init`, and never changes its exit code; a directly typed `archcore plugin` command reports its failures with a nonzero exit.
35. `archcore plugin install|update|remove|status` exposes the same engine the init step and the update step run — one engine, three entry points.