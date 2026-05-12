---
title: "Multi-Host Compatibility Layer Specification"
status: rejected
tags:
  - "architecture"
  - "hooks"
  - "multi-host"
  - "plugin"
---

## Status: Rejected (Superseded by `remove-bundled-launcher-global-cli.idea`)

This specification described the bundled CLI launcher architecture — POSIX, Windows CMD, and PowerShell launchers; resolution order `$ARCHCORE_BIN` → PATH → plugin cache → GitHub download; checksum verification; `bin/CLI_VERSION` pin; host-specific MCP wiring via `.mcp.json` and `.codex.mcp.json` pointing at the launcher; CWD rebase for Codex; `ARCHCORE_BIN` / `ARCHCORE_SKIP_DOWNLOAD` / `ARCHCORE_HIDE_EMPTY_NUDGE` environment contract.

All launcher mechanics were removed in plugin v0.4.0. The current shape is described in:

- `multi-host-plugin-architecture.adr` — shared-core / per-host-adapter split (still governing architecture).
- `remove-bundled-launcher-global-cli.idea` — the global-CLI-on-PATH decision that superseded this spec.
- `component-registry.doc` — current `bin/`, hooks, MCP, and manifest layout.
- `plugin-development.guide` — current prerequisites (`curl -fsSL https://archcore.ai/install.sh | bash` or `irm https://archcore.ai/install.ps1 | iex` per https://docs.archcore.ai/cli/install/) and MCP wiring (`command: "archcore"` everywhere).
- `codex-local-plugin-testing.guide` — current Codex packaging contract.

Original spec body removed to keep the knowledge base clean — git history is the source of truth for the launcher-era contract.
