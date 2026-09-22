---
title: "Hook Commands Are Recognized by the \"archcore hooks \" Prefix Marker"
status: accepted
tags:
  - "cli"
  - "integrations"
---

## Context

`archcore hooks install` (and every path that shares its installers: `archcore init --agent`, `doctor --fix`, the `install_host_config` MCP tool) writes hook entries into host config files it does not own — `.claude/settings.json`, `.cursor/hooks.json`, `.gemini/settings.json`, `.github/hooks/archcore.json`. On a re-run, a future CLI version must recognize entries a past version installed — even when the exact command string has changed between versions — or it appends a duplicate and the host fires the hook twice per session. At the same time, entries the user wrote or customized must never be touched.

## Decision

Every hook command archcore installs MUST start with the literal prefix `archcore hooks ` (trailing space included). Installers recognize archcore-owned entries with `strings.HasPrefix` against this marker (`isArchcoreHookCommand` in @internal/wiring/hooks_install.go) and classify entries three ways:

- **current** — carries today's exact command; nothing to do.
- **stale archcore** — starts with the marker but the command is outdated; updated in place, duplicates dropped.
- **foreign** — everything else (including undecodable entries); never touched.

This is a **forever contract** with two directions:

1. Every command any CLI version ever installs must start with the marker — otherwise later versions will not recognize it and will duplicate the hook.
2. The marker string itself can never change. If recognition ever needs to widen, add recognition of legacy strings — do not alter the prefix new installs write.

Matching is prefix, not substring, deliberately: a user-wrapped command such as `sh -c 'archcore hooks cursor session-start 2>/dev/null'` contains the marker but does not start with it, so it classifies foreign and survives untouched — the wrapper is a deliberate user customization.

A matcher-shaped entry (Claude Code / Gemini) counts as archcore-owned only when EVERY inner hook carries the marker; a hand-merged entry mixing archcore and foreign hooks is classified foreign so an update never drops someone else's hook.

## Alternatives

- **Exact-command matching** (pre-marker behavior): any command change between CLI versions appended a second entry → double session-start firing. Rejected; this defect motivated the marker.
- **Substring matching** (`strings.Contains`): recognizes user-wrapped commands as archcore's and clobbers the wrapper on update. Rejected.
- **A dedicated JSON ownership field** (e.g. `"archcore": true`): hosts do not guarantee unknown fields survive their own rewrites, and old entries would not carry it. Rejected.

## Consequences

- Re-running any installer over configs written by any other CLI version converges to exactly one archcore entry per event (spec tests in @internal/wiring/hooks_install_dedup_spec_test.go).
- User wrappers around archcore commands are permanently safe, at the cost of not auto-updating them — the user owns what they wrapped.
- Any new hook-installing surface (new agent, new event) must route its command strings through `archcore hooks <agent> <event>` shape so the prefix holds.
