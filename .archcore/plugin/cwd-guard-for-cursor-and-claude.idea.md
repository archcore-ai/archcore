---
title: "Cross-Host CWD Sanity Guard for Cursor and Claude Code MCP"
status: rejected
tags:
  - "claude-code"
  - "cursor"
  - "multi-host"
  - "plugin"
---

## Status: Rejected (Superseded)

This idea was based on the bundled CLI launcher architecture. With the transition to a global CLI on PATH, CWD guards in the launcher are no longer needed.

**New architecture (as of 2026-05-12):**
- Users install the Archcore CLI globally: `brew install archcore-ai/cli` or `go install github.com/archcore-ai/cli@latest`
- All MCP configs point directly to `archcore` on PATH (no launcher indirection)
- The `cwd` field in MCP configs is standard MCP practice (Cursor uses `${workspaceFolder}`, etc.)
- No custom `ARCHCORE_CWD`, `ARCHCORE_ALLOW_ANY_CWD`, or Step 0b/0c guards needed

See: `remove-bundled-launcher-global-cli.idea.md` for the rationale and benefits of removing the launcher entirely.

---

## Original Idea (Historical, Superseded)

[Original content preserved below for reference only]

### Idea

Extend the existing Codex cache-cwd guard in `bin/archcore` (Step 0b) with a cross-host sanity check (Step 0c) that refuses to start the archcore MCP server when cwd does not look like a user project root, plus ship a `cursor.mcp.json` template and README guidance so users register the server with `cwd: "${workspaceFolder}"` from the start.

[Full original content removed for brevity — see git history if needed]

---

## Why Rejected

- **Launcher removed.** The bundled shell launcher (`bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`) and all its resolution logic have been deleted.
- **CWD handling simplified.** The host's native `cwd` field (e.g., Cursor's `${workspaceFolder}`) is sufficient and more standard.
- **No custom env vars.** `ARCHCORE_CWD`, `ARCHCORE_ALLOW_ANY_CWD`, `ARCHCORE_ALLOW_PLUGIN_CWD` are no longer needed.
- **Tests removed.** `test/unit/launcher.bats` and the associated Step 0b/0c guard tests were deleted.

The global CLI approach is simpler, more maintainable, and solves the CWD problem through standard MCP mechanisms rather than custom launcher logic.
