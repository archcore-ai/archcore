---
title: "Bundled CLI Launcher with Auto-Install and Plugin-Owned MCP"
status: rejected
tags:
  - "architecture"
  - "multi-host"
  - "plugin"
---

## Status: Rejected (Superseded)

This decision has been superseded by the Global CLI + Thin Plugin architecture. See `remove-bundled-launcher-global-cli.idea.md` for context and decision rationale.

The bundled launcher, auto-download caching, and plugin-owned MCP registration have been removed as of [date]. The plugin now assumes users have the Archcore CLI installed globally (via `brew install archcore-ai/cli` or equivalent), and the three MCP config files (`.mcp.json`, `.codex.mcp.json`, `cursor.mcp.json`) point directly to `archcore` on PATH.

---

## Original Context (Historical)

[Original content preserved below for reference]

The plugin previously required users to install the Archcore CLI out-of-band (via `curl | bash`, `go install`, or package managers) and register the MCP server themselves — either per-user (`claude mcp add archcore archcore mcp -s user`) or per-repo (`.mcp.json` at the project root). This was captured in the Multi-Host Plugin Architecture ADR under the \"MCP ownership boundary\" section, which justified the choice on the grounds of avoiding Claude Code's duplicate-MCP suppression (v2.1.71+).

In practice this produced real friction:

- First-run onboarding required three separate, correctly-sequenced steps (install CLI → register MCP → reload plugin). Users routinely stopped after `/plugin install`.
- Install scripts (`curl | bash`) are a non-starter in many enterprise environments.
- The `claude mcp add ...` step is discoverable only by reading the README — `/plugin install` gives no hint that MCP registration is still required.
- Error messages from `bin/session-start` when MCP was unreachable (\"install the CLI and register the MCP server\") were ignored or misread as install failures.

The Claude Code plugin runtime now supports `${CLAUDE_PLUGIN_ROOT}` substitution in `.mcp.json` shipped at the plugin root, and treats plugin-provided MCP servers as first-class. Duplicate suppression only kicks in when the `command`/`args` exactly match a user- or project-registered server — and if a user has installed `archcore` globally, the PATH resolution inside the launcher picks it up, so the effective command is identical to the user's global registration and deduping is benign. Codex CLI v0.117.0+ (March 2026) gained plugin-shipped MCP via the Codex manifest's `mcpServers` pointer, but its plugin examples use plugin-relative command paths rather than Claude's root-variable substitution. Codex therefore ships a separate plugin-root MCP config at `.codex.mcp.json` that points at the same launcher with `./bin/archcore`.

### Original Drivers

- Zero-setup install is the single largest adoption lever for the plugin.
- The Go CLI ships single-file binaries per platform via GitHub Releases, making platform-targeted auto-download tractable.
- Enterprise/offline environments can still pin their own binary via `ARCHCORE_BIN` or `ARCHCORE_SKIP_DOWNLOAD=1`.

## Original Decision (Now Rejected)

**The plugin bundles a shell/PowerShell launcher (`bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`) that resolves the Archcore CLI on demand, and ships host-specific MCP registration files pointing at that launcher.** Claude Code consumes the plugin-root `.mcp.json`; Codex consumes plugin-root `.codex.mcp.json` via `.codex-plugin/plugin.json`.

[Rest of original ADR content removed for brevity — see git history if needed.]
