---
title: "Plugin Component Registry"
status: accepted
tags:
  - "plugin"
  - "reference"
---

## Overview

Reference document listing all components of the Archcore Plugin (multi-host: Claude Code, Cursor).

Note: Claude Code has merged commands into skills. All slash commands use `skills/<name>/SKILL.md`. The `commands/` directory is legacy and not used.

Per the Inverted Invocation Policy ADR (as amended by `remove-document-type-skills.adr.md` and `merge-review-status-remove-graph.adr.md`), skills are classified into three invocation classes: intent/track (auto-invocable by model + user) and utility (user-only). There are no per-document-type skills.

## Content

### Skills — Intent (9, auto-invocable by model + user)

Intent skills translate user intent into the correct document types, tracks, or analysis modes. They are the primary user entry points (Layer 1) and are auto-invocable — the model picks them up from user phrasing ("record a decision" → `decide`, "plan this feature" → `plan`, "review the docs" → `review`). No invocation-restricting flags. Creation-oriented intents inline per-type elicitation.

| Skill     | Directory           | User Intent                                                                          |
| --------- | ------------------- | ------------------------------------------------------------------------------------ |
| bootstrap | `skills/bootstrap/` | Seed an empty `.archcore/` on first install                                          |
| capture   | `skills/capture/`   | Document a module/component → routes to adr/spec/doc/guide                           |
| plan      | `skills/plan/`      | Plan a feature → routes to product-track or single plan                              |
| decide    | `skills/decide/`    | Record a decision (adr) or draft a proposal (rfc); offer rule+guide after ADR        |
| standard  | `skills/standard/`  | Establish a standard → routes to standard-track (adr → optional cpat → rule → guide) |
| review    | `skills/review/`    | Dashboard (default) or full health audit (`--deep`) — counts, gaps, staleness, recommendations |
| actualize | `skills/actualize/` | Detect stale docs → code drift, cascade, temporal analysis                           |
| help      | `skills/help/`      | Navigate the system → layer guide, onboarding                                        |
| context   | `skills/context/`   | Surface rules/decisions for a code area or pickup                                    |

### Skills — Tracks (6, auto-invocable by model + user)

Track skills orchestrate complete multi-document flows, creating documents in sequence with proper relations. Descriptions prefixed "Advanced —" (Layer 2). Each track step inlines per-type elicitation.

| Skill              | Directory                    | Flow                                      |
| ------------------ | ---------------------------- | ----------------------------------------- |
| product-track      | `skills/product-track/`      | idea → prd → plan                         |
| sources-track      | `skills/sources-track/`      | mrd → brd → urd                           |
| iso-track          | `skills/iso-track/`          | brs → strs → syrs → srs                   |
| architecture-track | `skills/architecture-track/` | adr → spec → plan                         |
| standard-track     | `skills/standard-track/`     | adr → (optional cpat) → rule → guide      |
| feature-track      | `skills/feature-track/`      | prd → spec → plan → task-type             |

### Skills — Utility (1, user-only, `disable-model-invocation: true`)

| Skill  | Directory        | Purpose                                                                        |
| ------ | ---------------- | ------------------------------------------------------------------------------ |
| verify | `skills/verify/` | Run plugin integrity checks — tests, lint, config validation, cross-references |

### Skills — Shared Runtime Assets (`skills/_shared/`)

Plain-markdown assets loaded at runtime by skills before composing documents. They ship with the plugin; skill instructions reference plugin-internal paths only (never the consumer's `.archcore/`).

| Asset                | Path                                  | Loaded by                            | Purpose                                                                              |
| -------------------- | ------------------------------------- | ------------------------------------ | ------------------------------------------------------------------------------------ |
| `precision-rules.md` | `skills/_shared/precision-rules.md`   | `decide`, `standard`, `capture`      | Forbidden vagueness lexicon, imperative phrasing, `[assumption]` marker conventions  |
| `adr-contract.md`    | `skills/_shared/adr-contract.md`      | `decide`, `standard`, `capture` (ADR) | Mandatory sections + bad/good examples for ADR content per MADR 4.0                  |

Companion ADR: `precision-over-coverage.adr.md` documents the design rationale; the runtime assets above are the canonical content.

### Document-type coverage

There are no per-document-type skills. Every Archcore document type is reachable via an intent skill, a track skill, or direct MCP (`mcp__archcore__create_document(type=<any>)`). See `skills-system.spec.md` → "Document-type coverage without type skills" for the full mapping.

### Visible `/` menu surface

Intent (9) + Tracks (6) + Utility (1) = **16 visible commands**. All 16 skills on disk are visible in `/` autocomplete — no hidden or flagged-out skills.

### Agents (2)

| Agent                | File                           | Role                            | Model  | Tools                       |
| -------------------- | ------------------------------ | ------------------------------- | ------ | --------------------------- |
| `archcore-assistant` | `agents/archcore-assistant.md` | Read/write documentation agent  | sonnet | All MCP + Read/Grep/Glob    |
| `archcore-auditor`   | `agents/archcore-auditor.md`   | Read-only documentation auditor | sonnet | Read MCP + Read/Grep/Glob   |

**archcore-assistant** — complex multi-document tasks: creation, requirements engineering, relation management. Foreground, blue, max 20 turns.

**archcore-auditor** — documentation health checks: coverage gaps, orphaned docs, stale statuses, code-document correlation (cross-references document path mentions with git history to flag drift). Background, yellow, max 15 turns.

### Hooks (6 entries across 3 events)

| #   | Event        | Matcher                                                                                           | Handler                     | Timeout |
| --- | ------------ | ------------------------------------------------------------------------------------------------- | --------------------------- | ------- |
| 1   | SessionStart | (all)                                                                                             | `bin/session-start`         | —       |
| 2   | PreToolUse   | `Write\|Edit`                                                                                     | `bin/check-archcore-write`  | 1s      |
| 3   | PreToolUse   | `Write\|Edit`                                                                                     | `bin/check-code-alignment`  | 1s      |
| 4   | PostToolUse  | `mcp__archcore__create_document\|update_document\|remove_document\|add_relation\|remove_relation` | `bin/validate-archcore`     | 3s      |
| 5   | PostToolUse  | `mcp__archcore__update_document`                                                                  | `bin/check-cascade`         | 3s      |
| 6   | PostToolUse  | `mcp__archcore__create_document\|mcp__archcore__update_document`                                  | `bin/check-precision`       | 3s      |

Hook configs: `hooks/hooks.json` (Claude Code, PascalCase events), `hooks/cursor.hooks.json` (Cursor, camelCase events + `afterMCPExecution`).

Hook 2 and Hook 3 share the `Write|Edit` matcher. Hook 2 (`check-archcore-write`) blocks direct writes to `.archcore/*.md`. Hook 3 (`check-code-alignment`) injects relevant `.archcore/` context for source-file edits via `hookSpecificOutput.additionalContext`. They act on disjoint path sets by construction — no conflict.

Hook 6 (`check-precision`) is the Phase 1 implementation of the Precision Initiative (see `precision-over-coverage.adr.md`). It emits soft warnings via `additionalContext` for forbidden vague words, missing mandatory sections, frontmatter gaps, and stub-length bodies. It never blocks (always exits 0).

Historical note: a prior revision had a `PostToolUse` entry with matcher `Write|Edit` invoking `validate-archcore`. It was removed because PreToolUse already blocks all Write/Edit to `.archcore/*.md` (PostToolUse fires only on success), so the matcher was dead weight forking a shell on every Write/Edit anywhere in the repo. See `hooks-validation-system.spec.md` for the rationale. Structure tests guard against its re-introduction.

### Bin Scripts

The `bin/` tree contains four distinct kinds of files: the CLI launcher, the CLI version pin, hook scripts, and the stdin-normalization library.

#### CLI Launcher (3 files + 1 version pin)

| File                    | Purpose                                                                                                                                                                                                       |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `bin/archcore`          | POSIX shell launcher. Resolves and execs the Archcore CLI in order: `$ARCHCORE_BIN` → `archcore` on `PATH` → plugin-managed cache → download from GitHub Releases (checksum-verified). Exit code passes through. |
| `bin/archcore.cmd`      | Windows cmd shim that delegates to `archcore.ps1` with `-NoProfile -NonInteractive -ExecutionPolicy Bypass`.                                                                                                  |
| `bin/archcore.ps1`      | PowerShell launcher. Same resolution order as the POSIX launcher; uses `Invoke-WebRequest` + `Get-FileHash` for download/verify; calls `Unblock-File` to strip MOTW so SmartScreen doesn't prompt.            |
| `bin/CLI_VERSION`       | Single-line file with the pinned semver of the CLI release the plugin is tested against. Launchers read this for cache key (`archcore-v${VERSION}`) and download URL.                                         |

Cache directory (first existing): `$CLAUDE_PLUGIN_DATA/archcore/cli` → `$XDG_DATA_HOME/archcore-plugin/cli` → `$HOME/.local/share/archcore-plugin/cli` (POSIX), or `$env:LOCALAPPDATA\archcore-plugin\cli` (Windows).

Env overrides: `ARCHCORE_BIN` pins an explicit binary; `ARCHCORE_SKIP_DOWNLOAD=1` disables step 4 (used by `bin/session-start` to keep SessionStart non-blocking).

See the Bundled CLI Launcher ADR for rationale.

#### Hook Scripts (7) and Library (1)

| Script                       | Hook Event                                   | Purpose                                                                                                                                                                                                                                                                                                    |
| ---------------------------- | -------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `bin/lib/normalize-stdin.sh` | (library)                                    | Multi-host stdin normalization. Detects host (Claude Code/Cursor/Copilot), extracts fields (tool_name, file_path, path), normalizes MCP tool names, provides output helpers (archcore_hook_block, archcore_hook_info, archcore_hook_pretool_info, archcore_hook_allow). Sourced by all hook scripts except check-staleness. |
| `bin/session-start`          | SessionStart                                 | Sources the normalizer, detects missing `.archcore/` and emits init guidance (instructs the agent to call `mcp__archcore__init_project`), otherwise invokes the local launcher with `ARCHCORE_SKIP_DOWNLOAD=1` to run `archcore hooks <host> session-start`, then calls `bin/check-staleness`. Always exits 0. |
| `bin/check-archcore-write`   | PreToolUse                                   | Blocks direct Write/Edit to `.archcore/**/*.md` with exit 2 + stderr message redirecting to MCP tools. Allows `.archcore/settings.json` and `.archcore/.sync-state.json`. Allows all paths outside `.archcore/`.                                                                                         |
| `bin/check-code-alignment`   | PreToolUse                                   | Injects relevant `.archcore/` context for source-file Write/Edit. Greps `.archcore/**/*.md` for documents referencing the edit path (directory prefixes, longest-first). Ranks by specificity → type priority (rule > cpat > adr > spec > guide). Emits top-3 via `hookSpecificOutput.additionalContext` (Claude Code/Copilot) or `additional_context` (Cursor). Never blocks; always exits 0. Honors `ARCHCORE_DISABLE_INJECTION=1` escape hatch and `.archcore/settings.json → codeAlignment.sourceRoots` override. |
| `bin/validate-archcore`      | PostToolUse                                  | Runs `archcore validate` via the launcher after MCP document operations (by tool_name prefix). The legacy Write/Edit branch in the script is retained as defensive code but is never reached from the current hooks config. Outputs JSON `hookSpecificOutput` when issues found, empty otherwise. Silently exits 0 if the launcher/CLI is unavailable. Always exits 0. |
| `bin/check-staleness`        | SessionStart (called by `bin/session-start`) | Detects code-document drift via git: finds source files changed since the last `.archcore/` commit, cross-references with documents that mention affected directories. Rate-limited to once per 24h via a timestamp file (`$CLAUDE_PLUGIN_DATA/archcore/last-staleness`, with XDG/HOME fallbacks). Emits only when matching documents exist — no generic "N files changed" fallback. Outputs plain text warning (max 2KB) or empty. Always exits 0. |
| `bin/check-cascade`          | PostToolUse                                  | After `update_document`, queries `.sync-state.json` relation graph for documents connected via `implements`, `depends_on`, or `extends` to the updated document. Outputs JSON `hookSpecificOutput` listing potentially stale dependents, or empty if no cascade. Always exits 0.                          |
| `bin/check-precision`        | PostToolUse                                  | Phase 1 of the Precision Initiative. After `create_document` and `update_document`, reads the resulting file from disk and runs four checks: forbidden vagueness lexicon (hardcoded list mirroring `skills/_shared/precision-rules.md`), mandatory sections by type (adr/rule/spec/guide/rfc), frontmatter title+status presence, body length ≥200 chars. Emits soft warnings via `additionalContext`. Always exits 0 (never blocks). See `precision-over-coverage.adr.md`. |

### Test Suite

| Component       | Location                     | Tests    | Description                                                                                             |
| --------------- | ---------------------------- | -------- | ------------------------------------------------------------------------------------------------------- |
| Unit tests      | `test/unit/`                 | 94+      | Test each bin script: stdin parsing, host detection, exit codes, output format, edge cases. Includes `launcher.bats` (CLI launcher resolution order), `check-staleness.bats` (24h rate limit, corrupt-stamp recovery), and `check-code-alignment.bats` (source-root filter, specificity ranking, top-3 cap, settings override, Cursor JSON shape, non-blocking safety). |
| Structure tests | `test/structure/`            | 50+      | Validate JSON configs, skill frontmatter, agent frontmatter, hook references, script permissions, rules. `hooks.bats` includes Phase 2.1 anti-regression invariants: no Write/Edit matcher on PostToolUse, no postToolUse event on Cursor, exact event-set invariants per host. |
| Fixtures        | `test/fixtures/stdin/`       | 12 files | Mock stdin JSON for Claude Code, Cursor, Copilot, and malformed inputs                                  |
| Helpers         | `test/helpers/`              | —        | common.bash (setup, mocks, timeout shim), bats-support, bats-assert (git submodules)                    |
| Makefile        | `Makefile`                   | —        | Targets: `test`, `test-unit`, `test-structure`, `lint`, `check-json`, `check-perms`, `verify`           |
| CI              | `.github/workflows/test.yml` | —        | GitHub Actions: macOS + Linux matrix, bats + shellcheck                                                 |

Run `make verify` for full check. Run `make test` for tests only. See `plugin-testing.guide.md` for details.

### MCP Server

The plugin **ships MCP registration** for Claude Code via `.mcp.json` at the plugin root:

```json
{
  "mcpServers": {
    "archcore": {
      "command": "${CLAUDE_PLUGIN_ROOT}/bin/archcore",
      "args": ["mcp"]
    }
  }
}
```

The `command` points at the bundled launcher, which resolves the actual CLI binary at invocation time (`$ARCHCORE_BIN` → `PATH` → cache → download). Users with a global `archcore` on `PATH` hit their existing install; users without one get a one-time auto-download on first MCP call. No manual `claude mcp add` or project-level `.mcp.json` required.

Codex CLI uses `.codex-plugin/plugin.json` to point at plugin-root `.codex.mcp.json`, which registers the same launcher with a plugin-relative command (`./bin/archcore`). Cursor users still register MCP externally (via Cursor's MCP settings or a project `mcp.json`) — the launcher works identically for them, just isn't wired in via a plugin-shipped MCP config.

Rationale: see the Bundled CLI Launcher ADR. The prior "plugin does not own MCP" stance (documented in the Multi-Host Plugin Architecture ADR) is superseded for Claude Code and Codex CLI; duplicate-suppression concerns are resolved because the launcher defers to an existing global install when present, making the effective command identical to a user-registered one.

### Plugin Configs

| File                              | Host        | Purpose                                                                |
| --------------------------------- | ----------- | ---------------------------------------------------------------------- |
| `.claude-plugin/plugin.json`      | Claude Code | Plugin manifest                                                        |
| `.cursor-plugin/plugin.json`      | Cursor      | Plugin manifest (with explicit component paths; no `mcpServers` field) |
| `.codex-plugin/plugin.json`       | Codex CLI   | Plugin manifest with `skills`, `hooks`, and `mcpServers` pointers      |
| `.claude-plugin/marketplace.json` | Claude Code | Marketplace metadata                                                   |
| `.cursor-plugin/marketplace.json` | Cursor      | Marketplace metadata                                                   |
| `.agents/plugins/marketplace.json` | Codex CLI  | Marketplace metadata and default-install policy                        |
| `.mcp.json`                       | Claude Code | Plugin-provided MCP server registration (launcher-backed)              |
| `.codex.mcp.json`                 | Codex CLI   | Plugin-provided MCP server registration (launcher-backed)              |
| `hooks/hooks.json`                | Claude Code | Hook event config (PascalCase)                                         |
| `hooks/cursor.hooks.json`         | Cursor      | Hook event config (camelCase + afterMCPExecution)                      |
| `hooks/codex.hooks.json`          | Codex CLI   | Hook event config (PascalCase + apply_patch matcher)                   |
| `rules/archcore-context.mdc`      | Cursor      | Always-apply context rule                                              |
| `rules/archcore-files.mdc`        | Cursor      | .archcore/ glob-triggered MCP-only rule                                |

## Examples

### All skills available as slash commands (visible `/` surface)

```
## Primary (intent skills — auto-invocable)
/archcore:bootstrap        — seed an empty .archcore/ on first install
/archcore:capture          — document a module or component
/archcore:plan             — plan a feature end-to-end
/archcore:decide           — record a decision (ADR) or draft a proposal (RFC)
/archcore:standard         — establish a team standard
/archcore:review           — dashboard (default) or full health audit (`--deep`)
/archcore:actualize        — detect stale docs, suggest updates
/archcore:help             — system guide
/archcore:context          — rules/decisions for a code area or pickup

## Advanced (track skills — auto-invocable)
/archcore:product-track      — idea → prd → plan
/archcore:sources-track      — mrd → brd → urd
/archcore:iso-track          — brs → strs → syrs → srs
/archcore:architecture-track — adr → spec → plan
/archcore:standard-track     — adr → (optional cpat) → rule → guide
/archcore:feature-track      — prd → spec → plan → task-type

## Utility
/archcore:verify           — run plugin integrity checks
```

Total visible in `/` menu: 16 commands. Every Archcore document type is reachable via these skills or directly through `mcp__archcore__create_document(type=<any>)`.
