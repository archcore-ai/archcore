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

Per the Inverted Invocation Policy ADR, skills are classified into four invocation classes: intent/track (auto-invocable by model + user), mainstream type (user-only via `/`), niche type (model-only, hidden from `/`), and utility (user-only).

## Content

### Skills — Intent (10, auto-invocable by model + user)

Intent skills translate user intent into the correct document types, tracks, or analysis modes. They are the primary user entry points (Layer 1) and are auto-invocable — the model picks them up from user phrasing ("record a decision" → `decide`, "plan this feature" → `plan`, "show the graph" → `graph`). No invocation-restricting flags.

| Skill     | Directory           | User Intent                                                |
| --------- | ------------------- | ---------------------------------------------------------- |
| capture   | `skills/capture/`   | Document a module/component → routes to adr/spec/doc/guide |
| plan      | `skills/plan/`      | Plan a feature → routes to product-track or single plan    |
| decide    | `skills/decide/`    | Record a decision → creates adr, offers rule+guide         |
| standard  | `skills/standard/`  | Establish a standard → routes to standard-track            |
| review    | `skills/review/`    | Check documentation health → analysis + recommendations    |
| status    | `skills/status/`    | Show dashboard → counts, relations, issues                 |
| actualize | `skills/actualize/` | Detect stale docs → code drift, cascade, temporal analysis |
| graph     | `skills/graph/`     | Render the relation graph as a Mermaid flowchart           |
| help      | `skills/help/`      | Navigate the system → layer guide, onboarding              |
| context   | `skills/context/`   | Surface rules/decisions for a code area or pickup          |

### Skills — Tracks (6, auto-invocable by model + user)

Track skills orchestrate complete multi-document flows, creating documents in sequence with proper relations. Descriptions prefixed "Advanced —" (Layer 2). Auto-invocable so the model can route multi-document requests through them; `sources-track` and `iso-track` also programmatically invoke niche type skills.

| Skill              | Directory                    | Flow                          |
| ------------------ | ---------------------------- | ----------------------------- |
| product-track      | `skills/product-track/`      | idea → prd → plan             |
| sources-track      | `skills/sources-track/`      | mrd → brd → urd               |
| iso-track          | `skills/iso-track/`          | brs → strs → syrs → srs       |
| architecture-track | `skills/architecture-track/` | adr → spec → plan             |
| standard-track     | `skills/standard-track/`     | adr → rule → guide            |
| feature-track      | `skills/feature-track/`      | prd → spec → plan → task-type |

### Skills — Document Types, Mainstream (10, user-only via `/`, `disable-model-invocation: true`)

Expert-level shortcuts for power users who know the exact type they want. Description NOT in model context — the model reaches these types only through intent-skill routing. Non-high-frequency types are prefixed "Expert —" (Layer 3).

| Skill               | Type                          | Category   |
| ------------------- | ----------------------------- | ---------- |
| `skills/adr/`       | Architecture Decision Record  | knowledge  |
| `skills/prd/`       | Product Requirements          | vision     |
| `skills/rfc/`       | Request for Comments          | knowledge  |
| `skills/rule/`      | Team Standard                 | knowledge  |
| `skills/guide/`     | How-To Instructions           | knowledge  |
| `skills/doc/`       | Reference Material            | knowledge  |
| `skills/spec/`      | Technical Specification       | knowledge  |
| `skills/idea/`      | Product/Technical Concept     | vision     |
| `skills/task-type/` | Recurring Task Pattern        | experience |
| `skills/cpat/`      | Code Pattern Change           | experience |

### Skills — Document Types, Niche (7, model-only, `user-invocable: false`, hidden from `/`)

Discovery and ISO 29148 types that are rarely invoked directly by users. Hidden from the `/` autocomplete menu to reduce cognitive load. The model still sees their descriptions so `sources-track` and `iso-track` can orchestrate them. Users reach these types by invoking the appropriate track or calling MCP tools directly.

| Skill          | Type                          | Category | Typical access path          |
| -------------- | ----------------------------- | -------- | ---------------------------- |
| `skills/mrd/`  | Market Requirements           | vision   | via `/archcore:sources-track` |
| `skills/brd/`  | Business Requirements         | vision   | via `/archcore:sources-track` |
| `skills/urd/`  | User Requirements             | vision   | via `/archcore:sources-track` |
| `skills/brs/`  | Business Requirements Spec    | vision   | via `/archcore:iso-track`     |
| `skills/strs/` | Stakeholder Requirements Spec | vision   | via `/archcore:iso-track`     |
| `skills/syrs/` | System Requirements Spec      | vision   | via `/archcore:iso-track`     |
| `skills/srs/`  | Software Requirements Spec    | vision   | via `/archcore:iso-track`     |

### Skills — Utility (1, user-only, `disable-model-invocation: true`)

| Skill  | Directory        | Purpose                                                                        |
| ------ | ---------------- | ------------------------------------------------------------------------------ |
| verify | `skills/verify/` | Run plugin integrity checks — tests, lint, config validation, cross-references |

### Visible `/` menu surface

Intent (10) + Tracks (6) + Mainstream types (10) + Utility (1) = **27 visible commands**. The 7 niche type skills exist as directories and are model-invocable but do not appear in `/` autocomplete. Total on disk: 34 skills.

### Agents (2)

| Agent                | File                           | Role                            | Model  | Tools                       |
| -------------------- | ------------------------------ | ------------------------------- | ------ | --------------------------- |
| `archcore-assistant` | `agents/archcore-assistant.md` | Read/write documentation agent  | sonnet | All 8 MCP + Read/Grep/Glob  |
| `archcore-auditor`   | `agents/archcore-auditor.md`   | Read-only documentation auditor | sonnet | 3 read MCP + Read/Grep/Glob |

**archcore-assistant** — complex multi-document tasks: creation, requirements engineering, relation management. Foreground, blue, max 20 turns.

**archcore-auditor** — documentation health checks: coverage gaps, orphaned docs, stale statuses, code-document correlation (cross-references document path mentions with git history to flag drift). Background, yellow, max 15 turns.

### Hooks (4 entries across 3 events)

| #   | Event        | Matcher                                                                                           | Handler                    | Timeout |
| --- | ------------ | ------------------------------------------------------------------------------------------------- | -------------------------- | ------- |
| 1   | SessionStart | (all)                                                                                             | `bin/session-start`        | —       |
| 2   | PreToolUse   | `Write\|Edit`                                                                                     | `bin/check-archcore-write` | 1s      |
| 3   | PostToolUse  | `mcp__archcore__create_document\|update_document\|remove_document\|add_relation\|remove_relation` | `bin/validate-archcore`    | 3s      |
| 4   | PostToolUse  | `mcp__archcore__update_document`                                                                  | `bin/check-cascade`        | 3s      |

Hook configs: `hooks/hooks.json` (Claude Code, PascalCase events), `hooks/cursor.hooks.json` (Cursor, camelCase events + `afterMCPExecution`).

Historical note: a prior revision had a 5th entry — `PostToolUse` with matcher `Write|Edit` invoking `validate-archcore`. It was removed because PreToolUse already blocks all Write/Edit to `.archcore/*.md` (PostToolUse fires only on success), so the matcher was dead weight forking a shell on every Write/Edit anywhere in the repo. See `hooks-validation-system.spec.md` for the rationale. Structure tests guard against its re-introduction.

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

#### Hook Scripts (5) and Library (1)

| Script                       | Hook Event                                   | Purpose                                                                                                                                                                                                                                                                                                    |
| ---------------------------- | -------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `bin/lib/normalize-stdin.sh` | (library)                                    | Multi-host stdin normalization. Detects host (Claude Code/Cursor/Copilot), extracts fields (tool_name, file_path, path), normalizes MCP tool names, provides output helpers (archcore_hook_block, archcore_hook_info, archcore_hook_allow). Sourced by all hook scripts except check-staleness.           |
| `bin/session-start`          | SessionStart                                 | Sources the normalizer, detects missing `.archcore/` and emits init guidance (instructs the agent to call `mcp__archcore__init_project`), otherwise invokes the local launcher with `ARCHCORE_SKIP_DOWNLOAD=1` to run `archcore hooks <host> session-start`, then calls `bin/check-staleness`. Always exits 0. |
| `bin/check-archcore-write`   | PreToolUse                                   | Blocks direct Write/Edit to `.archcore/**/*.md` with exit 2 + stderr message redirecting to MCP tools. Allows `.archcore/settings.json` and `.archcore/.sync-state.json`. Allows all paths outside `.archcore/`.                                                                                         |
| `bin/validate-archcore`      | PostToolUse                                  | Runs `archcore validate` via the launcher after MCP document operations (by tool_name prefix). The legacy Write/Edit branch in the script is retained as defensive code but is never reached from the current hooks config. Outputs JSON `hookSpecificOutput` when issues found, empty otherwise. Silently exits 0 if the launcher/CLI is unavailable. Always exits 0. |
| `bin/check-staleness`        | SessionStart (called by `bin/session-start`) | Detects code-document drift via git: finds source files changed since the last `.archcore/` commit, cross-references with documents that mention affected directories. Rate-limited to once per 24h via a timestamp file (`$CLAUDE_PLUGIN_DATA/archcore/last-staleness`, with XDG/HOME fallbacks). Emits only when matching documents exist — no generic "N files changed" fallback. Outputs plain text warning (max 2KB) or empty. Always exits 0. |
| `bin/check-cascade`          | PostToolUse                                  | After `update_document`, queries `.sync-state.json` relation graph for documents connected via `implements`, `depends_on`, or `extends` to the updated document. Outputs JSON `hookSpecificOutput` listing potentially stale dependents, or empty if no cascade. Always exits 0.                          |

### Test Suite

| Component       | Location                     | Tests    | Description                                                                                             |
| --------------- | ---------------------------- | -------- | ------------------------------------------------------------------------------------------------------- |
| Unit tests      | `test/unit/`                 | 81+      | Test each bin script: stdin parsing, host detection, exit codes, output format, edge cases. Includes `launcher.bats` (CLI launcher resolution order) and `check-staleness.bats` (24h rate limit, corrupt-stamp recovery). |
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

Cursor users still register MCP externally (via Cursor's MCP settings or a project `mcp.json`) — the launcher works identically for them, just isn't wired in via a plugin-shipped MCP config.

Rationale: see the Bundled CLI Launcher ADR. The prior "plugin does not own MCP" stance (documented in the Multi-Host Plugin Architecture ADR) is superseded for Claude Code; duplicate-suppression concerns are resolved because the launcher defers to an existing global install when present, making the effective command identical to a user-registered one.

### Plugin Configs

| File                              | Host        | Purpose                                                                |
| --------------------------------- | ----------- | ---------------------------------------------------------------------- |
| `.claude-plugin/plugin.json`      | Claude Code | Plugin manifest                                                        |
| `.cursor-plugin/plugin.json`      | Cursor      | Plugin manifest (with explicit component paths; no `mcpServers` field) |
| `.claude-plugin/marketplace.json` | Claude Code | Marketplace metadata                                                   |
| `.cursor-plugin/marketplace.json` | Cursor      | Marketplace metadata                                                   |
| `.mcp.json`                       | Claude Code | Plugin-provided MCP server registration (launcher-backed)              |
| `hooks/hooks.json`                | Claude Code | Hook event config (PascalCase)                                         |
| `hooks/cursor.hooks.json`         | Cursor      | Hook event config (camelCase + afterMCPExecution)                      |
| `rules/archcore-context.mdc`      | Cursor      | Always-apply context rule                                              |
| `rules/archcore-files.mdc`        | Cursor      | .archcore/ glob-triggered MCP-only rule                                |

## Examples

### All skills available as slash commands (visible `/` surface)

```
## Primary (intent skills — auto-invocable)
/archcore:capture          — document a module or component
/archcore:plan             — plan a feature end-to-end
/archcore:decide           — record a decision
/archcore:standard         — establish a team standard
/archcore:review           — documentation health check
/archcore:status           — dashboard
/archcore:actualize        — detect stale docs, suggest updates
/archcore:graph            — render the relation graph (Mermaid)
/archcore:help             — system guide
/archcore:context          — rules/decisions for a code area or pickup

## Utility
/archcore:verify           — run plugin integrity checks

## Advanced (track skills — auto-invocable)
/archcore:product-track    — idea → prd → plan
/archcore:sources-track    — mrd → brd → urd
/archcore:iso-track        — brs → strs → syrs → srs
/archcore:architecture-track — adr → spec → plan
/archcore:standard-track   — adr → rule → guide
/archcore:feature-track    — prd → spec → plan → task-type

## Expert (mainstream type skills — user-only via /)
/archcore:adr <topic>      — quick ADR creation
/archcore:prd <topic>      — quick PRD creation
/archcore:rfc <topic>      — quick RFC creation
/archcore:rule <topic>     — quick rule creation
/archcore:guide <topic>    — quick guide creation
/archcore:doc <topic>      — quick doc creation
/archcore:spec <topic>     — quick spec creation
/archcore:idea <topic>     — quick idea creation
/archcore:task-type <topic> — quick task-type creation
/archcore:cpat <topic>     — quick cpat creation

## Hidden (niche type skills — model-only, not in autocomplete)
(mrd, brd, urd, brs, strs, syrs, srs — reach via sources-track or iso-track)
```

Total visible in `/` menu: 27 commands.
