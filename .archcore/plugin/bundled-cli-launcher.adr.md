---
title: "Bundled CLI Launcher with Auto-Install and Plugin-Owned MCP"
status: rejected
tags:
  - "architecture"
  - "multi-host"
  - "plugin"
---

## Status: Rejected (Superseded by `remove-bundled-launcher-global-cli.idea`)

This ADR proposed shipping a download-on-first-use shell/PowerShell launcher (`bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`, `bin/CLI_VERSION`) that resolved the Archcore CLI on demand, plus host-specific MCP configs (`.mcp.json`, `.codex.mcp.json`) pointing at that launcher.

The launcher shipped briefly and was removed entirely in plugin v0.4.0 (2026-05-12, commit `2f99997`). Eight bug classes — offline failures, security patch lag, uneven host support, cache pollution, first-run latency, enterprise friction, version coupling via `bin/CLI_VERSION`, and plugin bloat — made the "zero-setup install" framing a net loss. The official installer at https://docs.archcore.ai/cli/install/ covers the same first-touch UX without coupling CLI lifecycle to plugin releases.

The plugin now assumes the Archcore CLI is on PATH and never invokes any plugin-side binary, downloader, or cache. The shared-core / per-host-adapter split from `multi-host-plugin-architecture.adr` remains; only the CLI-install / MCP-registration sub-decision was reversed.

**Current state:** see `remove-bundled-launcher-global-cli.idea`. The three MCP configs (`.mcp.json`, `.codex.mcp.json`, `cursor.mcp.json`) all use `"command": "archcore"`. Original ADR body and decision matrix removed — git history (commit `2f99997` and earlier) is the source of truth for what was attempted and why it was rolled back.
