---
title: "Multi-Host Compatibility Layer Specification"
status: rejected
tags:
  - "architecture"
  - "hooks"
  - "multi-host"
  - "plugin"
---

## Status: Rejected (Superseded)

This specification describes the bundled CLI launcher architecture, which has been superseded. See `remove-bundled-launcher-global-cli.idea.md` for the new global CLI approach.

The bundled launcher (`bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`, `bin/CLI_VERSION`), launcher resolution logic, and related environment variables (`ARCHCORE_SKIP_DOWNLOAD`, `ARCHCORE_BIN` cache management) have been removed.

A new global CLI-based architecture is now in place:
- Users install the Archcore CLI globally: `brew install archcore-ai/cli` or `go install github.com/archcore-ai/cli@latest`
- MCP configs point directly to `archcore` on PATH (no launcher indirection)
- Plugin remains thin: skills, agents, hooks only
- bin/session-start checks CLI availability and warns if missing
- bin/validate-archcore calls `archcore doctor` directly

---

## Original Specification (Historical, Superseded)

[Historical content below for reference]

The original specification detailed:
- Bundled POSIX, Windows CMD, and PowerShell launchers
- Launcher resolution order: ARCHCORE_BIN → PATH → plugin cache → GitHub download
- Plugin-managed cache with version pinning via `bin/CLI_VERSION`
- Checksum verification on downloads
- Host-specific MCP wiring via `.mcp.json` and `.codex.mcp.json` pointing at the launcher
- CWD rebase mechanism for Codex
- Stdout/stderr passthrough
- Environment variable contract (ARCHCORE_BIN, ARCHCORE_SKIP_DOWNLOAD, ARCHCORE_HIDE_EMPTY_NUDGE)

[Full original content removed for brevity — see git history if needed]

---

## Migration to Global CLI

Teams should:
1. Update `.mcp.json` from `"${CLAUDE_PLUGIN_ROOT}/bin/archcore"` to `"archcore"`
2. Update `.codex.mcp.json` from `"./bin/archcore"` + `cwd: "."` to `"archcore"` only
3. Keep `cursor.mcp.json` with `"command": "archcore"` and `"cwd": "${workspaceFolder}"`
4. Remove all launcher files (bin/archcore*, bin/CLI_VERSION)
5. Ensure `archcore` CLI is installed globally in CI/CD base images and developer machines
6. Update README with prerequisites

See: `.archcore/plugin/remove-bundled-launcher-global-cli.idea.md` for full rationale and benefits.
