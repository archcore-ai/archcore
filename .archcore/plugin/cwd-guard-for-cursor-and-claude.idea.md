---
title: "Cross-Host CWD Sanity Guard for Cursor and Claude Code MCP"
status: accepted
tags:
  - "cursor"
  - "claude-code"
  - "multi-host"
  - "plugin"
---

## Idea

Extend the existing Codex cache-cwd guard in `bin/archcore` (Step 0b) with a cross-host sanity check (Step 0c) that refuses to start the archcore MCP server when cwd does not look like a user project root, plus ship a `cursor.mcp.json` template and README guidance so users register the server with `cwd: "${workspaceFolder}"` from the start.

Three cooperating pieces:

1. **`bin/archcore` Step 0c — cross-host sanity guard.** When invoked as `archcore mcp` and the bypass env vars are unset, refuse if cwd is `/`, `$HOME`, `$CLAUDE_PLUGIN_ROOT`, `$CURSOR_PLUGIN_ROOT`, or a directory without any of these project markers: `.git`, `.archcore`, `package.json`, `go.mod`, `pyproject.toml`, `Cargo.toml`, `pom.xml`, `build.gradle`, `build.gradle.kts`. The refusal prints per-host fix instructions (Cursor: add `cwd: "${workspaceFolder}"`; Claude: launch from project dir; Codex: install shell wrapper). $HOME and the plugin-root env vars are compared via `cd "$VAR" && pwd -P` so symlinked install dirs match too.

2. **`cursor.mcp.json` template at the plugin root.** Cursor plugin manifests do not register MCP servers — users configure them in `~/.cursor/mcp.json` or `.cursor/mcp.json`. We ship a canonical snippet with `cwd: "${workspaceFolder}"` *and* `env.ARCHCORE_CWD: "${workspaceFolder}"` (belt-and-braces; the launcher honors `ARCHCORE_CWD` in Step 0 even on hosts that ignore `cwd`).

3. **`bin/archcore` stderr diagnostic.** On every `archcore mcp` start, one line on stderr: `[archcore mcp] cwd=<X> archcore_dir=<X>/.archcore (exists|missing)`. Surfaces in Cursor's MCP server panel and Claude Code's `/mcp` output, so users can verify which project the server actually attached to.

## Value

**Before.** A single global `~/.cursor/mcp.json` entry without `cwd` made archcore stick to whichever workspace Cursor opened first, leaking that project's `.archcore/` into every other project the user opened later. Diagnosis time: hours — the MCP responded successfully with plausible-looking but wrong documents. Same failure class hit Claude Code in multi-repo workspaces and when launched not from a project root.

**After.** Wrong cwd is converted from a silent successful read of the wrong project into a loud refusal with a per-host one-line fix. The diagnostic line on every start gives users a positive confirmation of which project the server is attached to. The template snippet makes the correct setup the obvious copy-paste.

## Possible Implementation

Shipped:

- `bin/archcore`:
  - Resolves `ARCHCORE_ALLOW_ANY_CWD` (canonical) with `ARCHCORE_ALLOW_PLUGIN_CWD` (back-compat alias) into `_archcore_allow_any_cwd`. Both Step 0b and Step 0c honor it.
  - **Step 0c**: cross-host sanity guard with five refuse conditions (filesystem root, `$HOME`, `$CLAUDE_PLUGIN_ROOT`, `$CURSOR_PLUGIN_ROOT`, no project markers). Refusal message lists the per-host fix and the escape hatch. Step 0b (cache-cwd refusal) remains the first line of defense for Codex and prints the wrapper recipe inline.
  - **Diagnostic**: one stderr line per `archcore mcp` start with the resolved cwd and `.archcore/` presence.
- `cursor.mcp.json` at the plugin root: snippet that users copy into their `.cursor/mcp.json` (or `~/.cursor/mcp.json` if pinning with `cwd: "${workspaceFolder}"`).
- `README.md`: new "Cursor: project-scoped MCP setup" block under the Cursor install steps with the snippet, the rationale, and a pointer to the refusal log line.
- `Makefile`: `cursor.mcp.json` added to `JSON_FILES` so `make check-json` validates it.
- `test/unit/launcher.bats`: new cases for Step 0c (HOME refuse, plugin-root refuse, no-marker refuse, `.git`-marker pass, `.archcore`-marker pass, `package.json`-marker pass, `ARCHCORE_CWD`-rebase pass, `ARCHCORE_ALLOW_ANY_CWD` bypass, legacy `ARCHCORE_ALLOW_PLUGIN_CWD` alias still works, diagnostic-log presence/absence). Existing cases adapted by adding `.git` to fake plugin install / user project fixtures and `2>/dev/null` to assertions that pin exact stdout.
- `test/structure/cursor-plugin.bats` (new): pins the `cursor.mcp.json` contract — `cwd: "${workspaceFolder}"` and `env.ARCHCORE_CWD: "${workspaceFolder}"` must both be present; `command` must invoke `archcore` with `args[0] == "mcp"`; README must reference the template.

## Risks and Constraints

- **False positives on minimal projects.** A user opening a brand-new directory before `git init` (no project markers yet) hits the guard. Mitigation: error message includes the escape hatch (`ARCHCORE_ALLOW_ANY_CWD=1`) and lists the markers; the common workflow is `git init` before invoking archcore anyway.
- **Resolved-symlink comparison.** `$HOME`, `$CLAUDE_PLUGIN_ROOT`, `$CURSOR_PLUGIN_ROOT` are all compared via `cd "$VAR" 2>/dev/null && pwd -P` so symlinked install dirs match too. If the env var points at a non-existent path, the check silently skips (no false refusal). Important on macOS where `$HOME=/Users/<u>` but `pwd -P` on `cd $HOME` yields the same — and on Linux distros where `/home/<u>` is symlinked.
- **Stderr noise from the diagnostic.** One line per MCP start. Cursor's MCP server panel and Claude Code's `/mcp` show stderr — that is exactly where users want this. Not visible to chat output.
- **Cross-host neutrality.** Step 0c has no host detection — it refuses based on cwd alone. The fix instructions cover all three hosts. Codex was already protected by Step 0b; Step 0c piggybacks safely (escape hatch + Codex-specific cwd marker `*/plugins/cache/*` already handled in 0b first).
- **`cursor.mcp.json` is a template, not an auto-installed file.** Cursor does not pick it up from the plugin root. Users copy the snippet into their own config. Auto-install would require a host hook (Cursor `sessionStart`) and risk overwriting user configs — rejected for now.

## Verification

A/B reproducer against the launcher (no MCP host required). **Important:** isolate `CLAUDE_PLUGIN_DATA` / `XDG_DATA_HOME` to a scratch dir — otherwise the launcher exec's any locally cached real CLI binary and you end up testing the live server instead of the launcher.

```sh
PLUGIN=/path/to/plugin
TMP=$(mktemp -d)
PETPROJECT="$TMP/pet"
SCRATCH_CACHE="$TMP/scratch-cache"
mkdir -p "$PETPROJECT" "$SCRATCH_CACHE"
(cd "$PETPROJECT" && git init -q && mkdir .archcore)

# Common env: no archcore on PATH, cache directories pointed at scratch (miss),
# ARCHCORE_SKIP_DOWNLOAD=1 so we exit at the download step with a "not cached"
# message instead of fetching from GitHub.
common_env=(env -i HOME="$HOME" PATH=/usr/bin:/bin USER="$USER" LANG=C TERM=xterm
            CLAUDE_PLUGIN_DATA="$SCRATCH_CACHE" XDG_DATA_HOME="$SCRATCH_CACHE"
            ARCHCORE_SKIP_DOWNLOAD=1)

# 1) BAD cwd ($HOME) -> guard refuses
(cd "$HOME" && "${common_env[@]}" "$PLUGIN/bin/archcore" mcp 2>&1) \
  | grep -q "Refusing to start MCP"

# 2) BAD cwd (plain tmpdir, no markers) -> "no project markers"
(cd "$TMP" && "${common_env[@]}" "$PLUGIN/bin/archcore" mcp 2>&1) \
  | grep -q "no project markers"

# 3) GOOD cwd (has .git + .archcore) -> guard passes, hits cache miss
(cd "$PETPROJECT" && "${common_env[@]}" "$PLUGIN/bin/archcore" mcp 2>&1) \
  | grep -q "not cached"

# 4) ARCHCORE_ALLOW_ANY_CWD=1 lets a bad cwd through
(cd "$HOME" && "${common_env[@]}" ARCHCORE_ALLOW_ANY_CWD=1 "$PLUGIN/bin/archcore" mcp 2>&1) \
  | grep -q "not cached"

# 5) Back-compat: legacy ARCHCORE_ALLOW_PLUGIN_CWD=1 alias still works
(cd "$HOME" && "${common_env[@]}" ARCHCORE_ALLOW_PLUGIN_CWD=1 "$PLUGIN/bin/archcore" mcp 2>&1) \
  | grep -q "not cached"

# 6) Diagnostic stderr line emitted on every successful mcp start
(cd "$PETPROJECT" && "${common_env[@]}" "$PLUGIN/bin/archcore" mcp 2>&1) \
  | grep -qE '\[archcore mcp\] cwd=.*\(exists\)'

rm -rf "$TMP"
```

Bats coverage: `bats test/unit/launcher.bats test/structure/cursor-plugin.bats` exercises the same matrix in isolation (mock archcore + scratch tmpdirs). 42 tests across the two files, all green.
