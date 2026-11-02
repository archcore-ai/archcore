---
title: "Backup Invalid Config Files Before Overwriting"
status: accepted
tags:
  - "component:cli"
  - "integrations"
---

## Context

When `archcore hooks install` or `archcore mcp install` runs, it reads and modifies config files belonging to AI agents (e.g., `.claude/settings.json`, `.cursor/hooks.json`, `.gemini/settings.json`, `.codex/hooks.json`, MCP JSON files). These files may already exist with content that is not valid JSON — due to manual edits, editor crashes, merge conflicts, or other tools writing malformed data.

If archcore fails on invalid JSON, the user is blocked from installing. If archcore silently overwrites, the user loses whatever was in the file.

## Decision

When a config file exists but cannot be parsed as valid JSON, archcore creates a backup at `{path}.bak` before proceeding with a fresh config. The shared implementation is `jsonfile.ReadOrBackup` (@cli/internal/jsonfile/jsonfile.go), used by the generic hook installer (@cli/internal/wiring/hooks_install.go) and the MCP config writer (@cli/internal/agents/mcp_helpers.go).

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

### Ownership is per field, not per entry

An entry archcore wrote is not archcore's to rewrite wholesale. The hook installer classifies an entry as stale whenever it differs from what archcore would write now, and "differs" includes a field archcore never writes — a `timeout` the user raised on our hook, a key a newer archcore added. Replacing the entry then deleted that field, silently, on every `init`, `hooks install`, and `doctor --fix`.

A stale entry is therefore updated by **overlay**: the fields archcore owns (`matcher`, the inner `type` and `command`, and any field a given host's entry shape declares) take the value archcore would write, and every other key keeps its value and its position. `overlayEntry` (@cli/internal/wiring/hooks_install.go) merges objects by key and arrays element-wise; a shorter desired array truncates, which is how a duplicated inner hook is still dropped. When the overlay changes nothing, the file is not written at all.

The trade this accepts: a field archcore used to write and no longer does is preserved rather than cleaned up. Keeping a stray key is the cheaper mistake.

The same principle governs MCP entries; the converge-ownership decision records it for that surface.

### Scope

Every host hook installer hands its event table to `installHookEvents`, which owns the read-parse-backup step (@cli/internal/wiring/hooks_install.go). The MCP JSON writers share it through @cli/internal/agents/mcp_helpers.go. `.codex/config.toml` is not covered: the Codex MCP entry is TOML, and hook wiring writes a separate `.codex/hooks.json` so that one writer owns each file.

No target archcore writes is JSONC, so every one takes backup-and-reset. A JSONC exception for `.vscode/mcp.json` was dropped when Copilot CLI stopped reading that file (v1.0.37). Its reasoning would apply again to any VS Code-owned target archcore starts writing: a valid JSONC config must not be replaced because strict JSON rejects it.

## Alternatives Considered

### Fail with error
Reject: Blocks the user from installing until they manually fix the file. Poor UX, especially in CI. (Exception: failing IS correct when the backup itself cannot be written — see Behavior 3.)

### Silent overwrite
Reject: Data loss. The user may have had valid (non-archcore) configuration in the file that gets destroyed.

### Replace a stale archcore entry wholesale
Reject: it is the silent-overwrite failure at entry scope. The entry carries the user's fields as well as ours, and no version of "this entry is ours" makes their `timeout` ours to delete.

### Interactive prompt ("File is invalid, overwrite?")
Reject: Doesn't work in non-interactive contexts (CI, hooks, scripts). Adds complexity for a rare edge case.

### Versioned backups (`.bak.1`, `.bak.2`, ...)
Reject: Over-engineered. The scenario (invalid JSON) should be rare. A single `.bak` is sufficient to recover.

### Tolerating JSONC everywhere
Reject: a JSONC parser is a new dependency for a case that only ever occurred in VS Code files, and archcore no longer writes one.

## Consequences

### Positive

- No data loss — the original content is always preserved in `.bak`, and the install aborts if the backup cannot be written
- Unknown fields, foreign entries, and key order in user configs survive every install (RawMessage round-trip) — including unknown fields **inside** archcore's own entries, which the field-level overlay preserves
- An install that only needs to preserve, not change, writes nothing: the file keeps its bytes and its mtime
- Works in CI and non-interactive environments without prompts
- Installation proceeds automatically — no manual intervention needed
- One policy across every target archcore writes, with no live exception to remember

### Negative

- `.bak` files may accumulate if the issue recurs (mitigated: single overwrite, not versioned)
- Users must know to check `.bak` files to recover original content
- `.bak` files should be added to `.gitignore` to avoid accidental commits
- A field archcore wrote in an earlier version and no longer writes stays in the config until the user removes it
- A future VS Code-owned target would reopen the JSONC question, which is why the superseded exception is recorded above rather than deleted
