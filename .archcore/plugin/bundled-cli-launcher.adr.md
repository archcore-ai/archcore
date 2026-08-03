---
title: "Bundled CLI Launcher with Auto-Install and Plugin-Owned MCP"
status: rejected
tags:
  - "architecture"
  - "multi-host"
  - "plugin"
---

## Context

The plugin proposed shipping a download-on-first-use launcher — `bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`, and a `bin/CLI_VERSION` pin — that resolved the Archcore CLI on demand, plus host-specific MCP configs (`.mcp.json`, `.codex.mcp.json`) pointing at that launcher. The goal was a zero-setup install: a user who cloned the plugin would get a working CLI without a separate install step. The launcher shipped, and in production it produced eight distinct bug classes: offline failures, security-patch lag, uneven host support, cache pollution, first-run latency, enterprise friction, version coupling through `bin/CLI_VERSION`, and plugin bloat.

## Decision

Rejected and reversed. The launcher and `bin/CLI_VERSION` were removed entirely in plugin v0.4.0 on 2026-05-12, commit `2f99997`; the plugin now assumes the Archcore CLI is on PATH, ships no plugin-side binary, downloader, or cache, and every MCP config names `"command": "archcore"`.

## Alternatives Considered

1. **Keep the launcher and fix the eight bug classes individually** — rejected because each fix added plugin-side lifecycle code for a problem the official installer at https://docs.archcore.ai/cli/install/ already solves without coupling CLI releases to plugin releases.
2. **Keep the launcher but drop the `bin/CLI_VERSION` pin** — ruled out because the pin was only one of the eight classes; removing it left offline failure, cache pollution, and patch lag untouched.

## Consequences

- The CLI lifecycle decoupled from plugin releases: a CLI security patch reaches users through the installer without a plugin version bump.
- Tradeoff: a first-time user now performs one install step before the plugin's document tools work. `bin/session-start` prints the install command when `archcore` is off PATH, so the failure is self-explaining rather than silent.
- The shared-core and per-host-adapter split from `multi-host-plugin-architecture.adr` is unaffected; only the CLI-install and MCP-registration sub-decision was reversed.
- The original ADR body and decision matrix were removed from this document. Git history at commit `2f99997` and earlier is the source of truth for what was attempted and why it was rolled back.

## Superseded when

- A host ships a documented, cross-platform mechanism for a plugin to declare a binary dependency that the host resolves and updates, which would remove the eight bug classes rather than relocate them.
- The official installer stops covering a supported platform, leaving users with no path to the CLI outside the plugin.
