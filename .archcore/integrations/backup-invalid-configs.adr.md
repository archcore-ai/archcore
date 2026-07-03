---
title: "Backup Invalid Config Files Before Overwriting"
status: accepted
tags:
  - "integrations"
---

## Context

When `archcore hooks install` or `archcore mcp install` runs, it reads and modifies config files belonging to AI agents (e.g., `.claude/settings.json`, `.cursor/hooks.json`, `.gemini/settings.json`, MCP JSON files). These files may already exist with content that is not valid JSON — due to manual edits, editor crashes, merge conflicts, or other tools writing malformed data.

If archcore fails on invalid JSON, the user is blocked from installing. If archcore silently overwrites, the user loses whatever was in the file.

## Decision

When a config file exists but cannot be parsed as valid JSON, archcore creates a backup at `{path}.bak` before proceeding with a fresh config. The shared implementation is `jsonfile.ReadOrBackup` (@internal/jsonfile/jsonfile.go), used by the generic hook installer (@cmd/hooks_install.go) and the MCP config writer (@internal/agents/mcp_helpers.go).

### Behavior

1. Read the existing file
2. Attempt to parse as JSON
3. **If parsing fails:**
   - Write the original content atomically to `{path}.bak` (mode `0644`)
   - **If the backup write itself fails: abort the install with an error.** The original file is never overwritten without a confirmed backup — proceeding would destroy the only (possibly hand-recoverable) copy of the user's data.
   - Log a warning: `"Corrupted {path} backed up, starting fresh"`
   - Continue with an empty config structure
4. **If parsing succeeds:** merge archcore entries into the existing config, round-tripping everything archcore does not own (unknown keys, foreign entries, key order) as opaque `json.RawMessage`; skip the write entirely when nothing changed
5. **If file doesn't exist:** start with an empty config (no backup needed)

All writes are atomic (temp file + rename), so a crash mid-write can never truncate a live user config.

### Exception: JSONC-capable targets (`.vscode/mcp.json`)

VS Code config files legitimately contain JSONC (comments, trailing commas). For `.vscode/mcp.json` (Copilot), "invalid strict JSON" usually means a perfectly valid JSONC config whose other MCP servers must not be silently replaced. This target therefore uses the `corruptSkipInstall` policy instead of backup-and-reset: the file is **left untouched**, a warning plus manual-install instructions are printed, and the install returns success (mirroring the `ManualMCPInstallHint` UX for agents without automatic MCP config).

### Affected Files

| File | Agent | Policy | Implementation |
|------|-------|--------|----------------|
| `.claude/settings.json` | Claude Code | backup-and-reset | @cmd/hooks.go via @cmd/hooks_install.go |
| `.cursor/hooks.json` | Cursor | backup-and-reset | @cmd/hooks_cursor.go via @cmd/hooks_install.go |
| `.gemini/settings.json` | Gemini CLI | backup-and-reset | @cmd/hooks_gemini_cli.go via @cmd/hooks_install.go |
| `.github/hooks/archcore.json` | Copilot (hooks) | backup-and-reset | @cmd/hooks_copilot.go via @cmd/hooks_install.go |
| Standard MCP JSON files (`.mcp.json`, `.cursor/mcp.json`, `.gemini/settings.json`, …) | Multiple | backup-and-reset | @internal/agents/mcp_helpers.go (`WriteStandardMCPJSON`) |
| `opencode.json` | OpenCode | backup-and-reset | @internal/agents/opencode.go (delegates to `writeMCPConfig`) |
| `.vscode/mcp.json` | Copilot (MCP) | **skip-install (JSONC exception)** | @internal/agents/mcp_helpers.go (`WriteVSCodeMCPJSON`) |

## Alternatives Considered

### Fail with error
Reject: Blocks the user from installing until they manually fix the file. Poor UX, especially in CI. (Exception: failing IS correct when the backup itself cannot be written — see Behavior 3.)

### Silent overwrite
Reject: Data loss. The user may have had valid (non-archcore) configuration in the file that gets destroyed.

### Interactive prompt ("File is invalid, overwrite?")
Reject: Doesn't work in non-interactive contexts (CI, hooks, scripts). Adds complexity for a rare edge case.

### Versioned backups (`.bak.1`, `.bak.2`, ...)
Reject: Over-engineered. The scenario (invalid JSON) should be rare. A single `.bak` is sufficient to recover.

### Tolerating JSONC everywhere
Reject: a JSONC parser is a new dependency for a case that only demonstrably occurs in VS Code files; the skip-install exception covers the real-world case without one.

## Consequences

### Positive

- No data loss — the original content is always preserved in `.bak`, and the install aborts if the backup cannot be written
- Unknown fields, foreign entries, and key order in user configs survive every install (RawMessage round-trip)
- Works in CI and non-interactive environments without prompts
- Installation proceeds automatically — no manual intervention needed
- Consistent behavior across all agent config files, with a single documented exception for JSONC targets

### Negative

- `.bak` files may accumulate if the issue recurs (mitigated: single overwrite, not versioned)
- Users must know to check `.bak` files to recover original content
- `.bak` files should be added to `.gitignore` to avoid accidental commits
- For `.vscode/mcp.json` the user must add the archcore entry manually when their file is JSONC (instructions are printed)