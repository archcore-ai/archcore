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

When a config file exists but cannot be parsed as valid JSON, archcore creates a backup at `{path}.bak` before proceeding with a fresh config.

### Behavior

1. Read the existing file
2. Attempt to parse as JSON
3. **If parsing fails:**
   - Write the original content to `{path}.bak` (mode `0644`)
   - Log a warning: `"Corrupted {path} backed up, starting fresh"`
   - Continue with an empty config structure
4. **If parsing succeeds:** merge archcore entries into the existing config
5. **If file doesn't exist:** start with an empty config (no backup needed)

### Affected Files

| File | Agent | Backup Location | Implementation |
|------|-------|-----------------|----------------|
| `.claude/settings.json` | Claude Code | `.claude/settings.json.bak` | `cmd/hooks.go:155` |
| `.cursor/hooks.json` | Cursor | `.cursor/hooks.json.bak` | `cmd/hooks_cursor.go:70` |
| `.gemini/settings.json` | Gemini CLI | `.gemini/settings.json.bak` | `cmd/hooks_gemini_cli.go:56` |
| Standard MCP JSON files | Multiple | `{path}.bak` | `internal/agents/mcp_helpers.go:30` |

## Alternatives Considered

### Fail with error
Reject: Blocks the user from installing until they manually fix the file. Poor UX, especially in CI.

### Silent overwrite
Reject: Data loss. The user may have had valid (non-archcore) configuration in the file that gets destroyed.

### Interactive prompt ("File is invalid, overwrite?")
Reject: Doesn't work in non-interactive contexts (CI, hooks, scripts). Adds complexity for a rare edge case.

### Versioned backups (`.bak.1`, `.bak.2`, ...)
Reject: Over-engineered. The scenario (invalid JSON) should be rare. A single `.bak` is sufficient to recover.

## Consequences

### Positive

- No data loss — the original content is always preserved in `.bak`
- Works in CI and non-interactive environments without prompts
- Installation proceeds automatically — no manual intervention needed
- Consistent behavior across all agent config files

### Negative

- `.bak` files may accumulate if the issue recurs (mitigated: single overwrite, not versioned)
- Users must know to check `.bak` files to recover original content
- `.bak` files should be added to `.gitignore` to avoid accidental commits