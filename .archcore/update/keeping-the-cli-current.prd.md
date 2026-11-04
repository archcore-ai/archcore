---
title: "Keeping the CLI Current"
status: accepted
tags:
  - "cli"
  - "component:cli"
  - "integrations"
  - "telemetry"
  - "update"
---

## Vision

An Archcore user runs a current CLI without ever thinking about it. The binary keeps itself up to date from inside the one process that can afford the work, and it reports enough about that path for the project to know whether it functions — on every platform, in both the plugin and the non-plugin population. The plugin rides the same rails: selecting a host delivers everything that host needs, and one command keeps both entry points current.

## Problem Statement

Nothing updates the CLI unless a user types `archcore update`. The single nudge toward it is the plugin's SessionStart advisory, which reaches only plugin users and only asks. There is no measurement at all: the binary emits no telemetry today, so a stale install base and a healthy one look identical from the outside.

Three consequences follow. Users run old binaries against a plugin that assumes newer ones — the compatibility gate in `bin/cli-gte` exists because that already happens. The mismatch also runs the other way: nothing updates an installed plugin, so an aging plugin meets every newer CLI, and the duplicated-hook warning in `plugin-cli-compatibility.rule.md` exists because that side already happens too. Update failures are invisible: `@cli/internal/update/update.go` verifies a checksum, extracts an archive under two candidate binary names, and performs a rename dance on Windows, three failure surfaces with no observability. And version adoption is unanswerable, so no release can be judged by whether anyone received it.

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

Out of scope: package-manager channels (Homebrew, Scoop, winget) and the install provenance receipt that depends on them — the receipt activates with the first package-manager channel; release signing — decided against 2026-08-15, the trust anchor stays GitHub account and pipeline security with the health probe, the official-build marker, and the non-inertness gate as the compensating controls; `cli_command`, `mcp_tool_call`, `sync_*`, and every other event the telemetry provider RFC proposed and the CLI never built.

## Requirements

The normative behavior lives in the specs: `unattended-update.spec` and `mcp-background-update.spec` for the update path, `cli-update-telemetry.spec` for reporting, `updating-the-plugin.spec` for the plugin step, and `plugin-delivery.spec` for delivery. This section keeps the product decisions behind them.

1. Unattended update has no opt-out surface: no environment variable, flag, or setting disables it. A per-project `settings.json` has no say over a machine-global action. A root-owned install directory is the supported operator answer for machines that must not self-update.
2. A replacement lands only after the downloaded binary passed a health probe; a fleet cannot be bricked by a release that cannot start.
3. Each declined attempt is attributable to `current` or `not_writable`, so an empty series means the mechanism never ran rather than "every machine was already current".
4. Update events join to install events with no second identifier, so retention is answerable in one query.
5. The PostHog key reaches the binary only through the release build, never through this repository, and the release pipeline proves the published artifact is not inert.
6. `/privacy` describes both the events and the update behavior before the release that emits them is downloadable.
7. The plugin step runs only inside a user-typed `archcore update`. The unattended path and the MCP trigger never reach it, and it emits no telemetry event.
8. Selecting a host in `archcore init` delivers everything that host needs — hooks, MCP config, and the plugin. Deselecting the host is the opt-out, and no second prompt exists.
9. Non-interactive init installs a plugin only for a host named with `--agent`, which carries the consent. `--yes` alone and CI environments print the per-host commands instead of running them.
10. Claude Code delivery defaults to user scope with marketplace auto-update, so the host keeps the plugin current on its own. `--scope project` is the opt-in that delivers the declaration to the whole team.
11. A delivery or plugin-step failure prints the exact command and never changes the exit code of `archcore init` or `archcore update`. A directly typed `archcore plugin` command reports its failures with a nonzero exit.