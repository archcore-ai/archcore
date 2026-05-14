---
title: "Plugin Component Registry"
status: accepted
tags:
  - "plugin"
  - "reference"
---

## Overview

Reference document listing all components of the Archcore Plugin (multi-host: Claude Code, Cursor, Codex CLI).

Note: Claude Code and Cursor surface user-invoked workflows directly from skills (`skills/<name>/SKILL.md`). Codex CLI discovers slash commands from root-level `commands/*.md` wrappers — thin host-adapter shims that delegate to the matching skill. The skill remains the single behavioral source of truth across all three hosts; no workflow logic lives in `commands/`.

Per `skill-surface-collapse.adr.md`, the visible `/` palette is exactly **7 auto-invocable intent skills**. There are no per-document-type skills, no track skills, and no utility skills. Track flows live as references under `skills/plan/references/`; drift detection lives at `skills/audit/lib/drift-detection.md`.

## Content

### Skills (7, auto-invocable by model + user)

Every skill carries no `disable-model-invocation` flag. Routing is governed by per-skill "Activate when X. Do NOT activate for Y." descriptions.

| Skill | Directory | User Intent |
| --- | --- | --- |
| init | `skills/init/` | First-time onboarding — seed `.archcore/` with stack rule, run guide, scale-appropriate extras |
| capture | `skills/capture/` | Document a module/component — routes to adr / spec / doc / guide |
| decide | `skills/decide/` | Record a decision (adr) or draft a proposal (rfc); optional standard cascade (cpat → rule → guide) |
| plan | `skills/plan/` | Plan a feature or initiative — routes to single plan or one of four flows (product / sources / iso / feature) |
| audit | `skills/audit/` | Documentation health — dashboard (default), `--deep` audit, or `--drift` detection |
| context | `skills/context/` | Surface rules/decisions for a code area or pickup |
| help | `skills/help/` | Navigate the system — command catalogue, onboarding |

### Per-Flow References

Flows that previously lived as standalone track skills now live as references loaded on demand by `plan`:

| Reference | Path | Flow |
| --- | --- | --- |
| product-flow | `skills/plan/references/product-flow.md` | idea → prd → plan |
| sources-flow | `skills/plan/references/sources-flow.md` | mrd → brd → urd |
| iso-flow | `skills/plan/references/iso-flow.md` | brs → strs → syrs → srs |
| feature-flow | `skills/plan/references/feature-flow.md` | prd → spec → plan → task-type |

`decide` similarly carries continuation logic under `skills/decide/references/continuations.md` (ADR → CPAT → rule → guide cascade).

`audit` carries its drift-mode protocol at `skills/audit/lib/drift-detection.md` (code-drift, cascade, and temporal staleness checks).

### Shared Runtime Assets (`skills/_shared/`)

Plain-markdown assets loaded at runtime by skills before composing documents. They ship with the plugin; skill instructions reference plugin-internal paths only (never the consumer's `.archcore/`).

| Asset | Path | Loaded by | Purpose |
| --- | --- | --- | --- |
| `precision-rules.md` | `skills/_shared/precision-rules.md` | `decide`, `capture` | Forbidden vagueness lexicon, imperative phrasing, `[assumption]` marker conventions |
| `adr-contract.md` | `skills/_shared/adr-contract.md` | `decide`, `capture` (ADR) | Mandatory sections + bad/good examples for ADR content per MADR 4.0 |

Companion ADR: `precision-over-coverage.adr.md` documents the design rationale.

### Codex Slash Command Wrappers (`commands/`)

Codex CLI requires a thin wrapper per user-facing skill so that `/archcore:<name>` appears in its `/` menu. Each wrapper is a host-adapter shim — `description:` frontmatter plus a one-line delegate instruction pointing at `skills/<name>/SKILL.md`. No workflow logic lives here.

| Wrapper Set | Count | Purpose |
| --- | --- | --- |
| `commands/<name>.md` (Codex CLI) | 7 | One per skill (`init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`) — surfaces `/archcore:<name>` in Codex |

Conformance is enforced by `test/structure/codex-plugin.bats`: every entry must exist, carry `description:`, and reference the matching `skills/<name>/SKILL.md`. Claude Code and Cursor do not need wrappers — they surface skills directly.

### Document-type coverage

Every Archcore document type is reachable via an intent skill or direct MCP (`mcp__archcore__create_document(type=<any>)`). See `skills-system.spec.md` → "Document-type coverage" for the full mapping.

### Visible `/` menu surface

**7 visible commands.** All 7 skills on disk are visible — no hidden or flagged-out skills. Codex CLI exposes the same 7 entries via the matching `commands/*.md` wrappers.

### Agents (2)

| Agent | File | Role | Model | Tools |
| --- | --- | --- | --- | --- |
| `archcore-assistant` | `agents/archcore-assistant.md` | Read/write documentation agent | sonnet | All MCP + Read/Grep/Glob |
| `archcore-auditor` | `agents/archcore-auditor.md` | Read-only documentation auditor | sonnet | Read MCP + Read/Grep/Glob |

**archcore-assistant** — complex multi-document tasks: creation, requirements engineering, relation management. Foreground, blue, max 20 turns.

**archcore-auditor** — documentation health checks: coverage gaps, orphaned docs, stale statuses, code-document correlation (cross-references document path mentions with git history to flag drift). Background, yellow, max 15 turns.

For Codex CLI, both subagents also ship as TOML variants (`agents/archcore-assistant.toml`, `agents/archcore-auditor.toml`) — same `developer_instructions` content, plus Codex-specific `sandbox_mode` and `disabled_tools[]` fields.

### Hooks (6 entries across 3 events)

| # | Event | Matcher | Handler | Timeout |
| --- | --- | --- | --- | --- |
| 1 | SessionStart | (all) | `bin/session-start` | — |
| 2 | PreToolUse | `Write\|Edit` | `bin/check-archcore-write` | 1s |
| 3 | PreToolUse | `Write\|Edit` | `bin/check-code-alignment` | 1s |
| 4 | PostToolUse | `mcp__archcore__create_document\|update_document\|remove_document\|add_relation\|remove_relation` | `bin/validate-archcore` | 3s |
| 5 | PostToolUse | `mcp__archcore__update_document` | `bin/check-cascade` | 3s |
| 6 | PostToolUse | `mcp__archcore__create_document\|mcp__archcore__update_document` | `bin/check-precision` | 3s |

Hook configs: `hooks/hooks.json` (Claude Code, PascalCase events), `hooks/cursor.hooks.json` (Cursor, camelCase events + `afterMCPExecution`), `hooks/codex.hooks.json` (Codex CLI, PascalCase events with `apply_patch` matcher addition for Codex's native edit primitive).

Hook 2 and Hook 3 share the `Write|Edit` matcher. Hook 2 (`check-archcore-write`) blocks direct writes to `.archcore/*.md`. Hook 3 (`check-code-alignment`) injects relevant `.archcore/` context for source-file edits via `hookSpecificOutput.additionalContext`. They act on disjoint path sets by construction — no conflict.

Hook 6 (`check-precision`) is the Phase 1 implementation of the Precision Initiative (see `precision-over-coverage.adr.md`). It emits soft warnings via `additionalContext` for forbidden vague words, missing mandatory sections, frontmatter gaps, and stub-length bodies. It never blocks (always exits 0).

Historical note: a prior revision had a `PostToolUse` entry with matcher `Write|Edit` invoking `validate-archcore`. It was removed because PreToolUse already blocks all Write/Edit to `.archcore/*.md` (PostToolUse fires only on success), so the matcher was dead weight forking a shell on every Write/Edit anywhere in the repo. See `hooks-validation-system.spec.md` for the rationale. Structure tests guard against its re-introduction.

`bin/session-start` carries a plugin-install-dir guard (see `cursor-mcp-architecture.adr.md`): when cwd contains a sibling `.cursor-plugin/`, `.claude-plugin/`, or `.codex-plugin/` manifest, the hook exits silently. This prevents the hook from emitting the plugin's bundled `.archcore/` (if any future regression reintroduces it) as the user's knowledge base.

### Bin Scripts

The `bin/` tree contains hook scripts and the shared stdin-normalization library. The plugin **does not bundle the Archcore CLI binary or any launcher wrapper** — it invokes `archcore` directly from PATH. Users install the CLI globally via the official installer at https://docs.archcore.ai/cli/install/. See `remove-bundled-launcher-global-cli.idea.md` for the rationale.

| Script | Hook Event | Purpose |
| --- | --- | --- |
| `bin/lib/normalize-stdin.sh` | (library) | Multi-host stdin normalization. Detects host (Claude Code/Cursor/Copilot/Codex), extracts fields (tool_name, file_path, path), normalizes MCP tool names, provides output helpers (archcore_hook_block, archcore_hook_info, archcore_hook_pretool_info, archcore_hook_allow). Sourced by all hook scripts except check-staleness. |
| `bin/session-start` | SessionStart | Sources the normalizer; if `archcore` is not on PATH, prints an install message pointing at https://docs.archcore.ai/cli/install/ and exits 0. Refuses to run when cwd contains a sibling plugin manifest. Otherwise detects missing `.archcore/` and emits init guidance (instructs the agent to call `mcp__archcore__init_project`), or invokes `archcore hooks <host> session-start` directly, then calls `bin/check-staleness`. Always exits 0. |
| `bin/check-archcore-write` | PreToolUse | Blocks direct Write/Edit to `.archcore/**/*.md` with exit 2 + stderr message redirecting to MCP tools. Allows `.archcore/settings.json` and `.archcore/.sync-state.json`. Allows all paths outside `.archcore/`. |
| `bin/check-code-alignment` | PreToolUse | Injects relevant `.archcore/` context for source-file Write/Edit. Greps `.archcore/**/*.md` for documents referencing the edit path (directory prefixes, longest-first). Ranks by specificity → type priority (rule > cpat > adr > spec > guide). Emits top-3 via `hookSpecificOutput.additionalContext` (Claude Code/Codex/Copilot) or `additional_context` (Cursor). Never blocks; always exits 0. Honors `ARCHCORE_DISABLE_INJECTION=1` escape hatch and `.archcore/settings.json → codeAlignment.sourceRoots` override. |
| `bin/validate-archcore` | PostToolUse | Runs `archcore doctor` directly (timeout 2s) after MCP document operations (by tool_name prefix). The legacy Write/Edit branch in the script is retained as defensive code but is never reached from the current hooks config. Outputs JSON `hookSpecificOutput` when issues found, empty otherwise. Silently exits 0 if the CLI is unavailable. Always exits 0. |
| `bin/check-staleness` | SessionStart (called by `bin/session-start`) | Detects code-document drift via git: finds source files changed since the last `.archcore/` commit, cross-references with documents that mention affected directories. Rate-limited to once per 24h via a timestamp file. Emits only when matching documents exist. Outputs plain text warning (max 2KB) or empty. Always exits 0. |
| `bin/check-cascade` | PostToolUse | After `update_document`, queries `.sync-state.json` relation graph for documents connected via `implements`, `depends_on`, or `extends` to the updated document. Outputs JSON `hookSpecificOutput` listing potentially stale dependents, or empty if no cascade. Always exits 0. |
| `bin/check-precision` | PostToolUse | Phase 1 of the Precision Initiative. After `create_document` and `update_document`, reads the resulting file from disk and runs four checks: forbidden vagueness lexicon, mandatory sections by type, frontmatter title+status presence, body length ≥200 chars. Emits soft warnings via `additionalContext`. Always exits 0 (never blocks). See `precision-over-coverage.adr.md`. |

### Test Suite

| Component | Location | Tests | Description |
| --- | --- | --- | --- |
| Unit tests | `test/unit/` | — | Test each bin script: stdin parsing, host detection, exit codes, output format, edge cases. |
| Structure tests | `test/structure/` | — | Validate JSON configs, skill frontmatter, agent frontmatter, hook references, script permissions, rules. `hooks.bats` includes anti-regression invariants. `cursor-plugin.bats` locks `docs/cursor.mcp.example.json` shape. `codex-plugin.bats` enforces Codex manifest, marketplace schema, hooks shape, MCP wiring, TOML agents, and parity between `commands/*.md` wrappers (7) and `skills/<name>/SKILL.md`. |
| Fixtures | `test/fixtures/stdin/` | — | Mock stdin JSON for Claude Code, Cursor, Copilot, Codex CLI, and malformed inputs |
| Helpers | `test/helpers/` | — | common.bash (setup, mocks, timeout shim), bats-support, bats-assert (git submodules) |
| Makefile | `Makefile` | — | Targets: `test`, `test-unit`, `test-structure`, `lint`, `check-json`, `check-perms`, `verify`. Dev-only — stripped from `main` distribution. |
| CI | `.github/workflows/test.yml` | — | GitHub Actions on push/PR to `dev`: macOS + Linux matrix, bats + shellcheck |
| Release | `.github/workflows/release.yml` | — | GitHub Actions on tag push: strips dev-only artifacts, force-pushes the clean tree to `main`. See `docs/release.md` for the blocklist. |

Run `make verify` for full check. Run `make test` for tests only. See `plugin-testing.guide.md` for details.

### MCP Server

The plugin **ships MCP registration** for Claude Code via `.mcp.json` at the plugin root:

```json
{
  "mcpServers": {
    "archcore": {
      "command": "archcore",
      "args": ["mcp"]
    }
  }
}
```

The `command` resolves through PATH — users must have the Archcore CLI installed globally (see https://docs.archcore.ai/cli/install/). If the CLI is missing at session start, the MCP server fails to register and `bin/session-start` prints the install instructions.

Codex CLI uses `.codex-plugin/plugin.json` to point at plugin-root `.codex.mcp.json`, which uses the same shape — `command: "archcore"`, `args: ["mcp"]`.

Cursor is the exception. The plugin **does not** ship a plugin-MCP for Cursor — no `mcp.json` at the plugin root. Cursor users instead copy `docs/cursor.mcp.example.json` into `~/.cursor/mcp.json` (user-scoped) or `.cursor/mcp.json` (project-scoped); the template passes `--project ${workspaceFolder}` in `args` to make the workspace explicit. Full rationale and three-layer defense in `cursor-mcp-architecture.adr.md`.

Rationale: see `remove-bundled-launcher-global-cli.idea.md`. The previous bundled launcher (download-on-first-use, checksum-verified, cached per host) is removed; CLI lifecycle now decouples cleanly from plugin releases.

### Plugin Configs

| File | Host | Purpose |
| --- | --- | --- |
| `.claude-plugin/plugin.json` | Claude Code | Plugin manifest |
| `.cursor-plugin/plugin.json` | Cursor | Plugin manifest (with explicit component paths; **no `mcpServers` field** — deliberately disabled, see `cursor-mcp-architecture.adr.md`) |
| `.codex-plugin/plugin.json` | Codex CLI | Plugin manifest with `skills`, `hooks`, and `mcpServers` pointers |
| `.claude-plugin/marketplace.json` | Claude Code | Marketplace metadata |
| `.cursor-plugin/marketplace.json` | Cursor | Marketplace metadata |
| `.agents/plugins/marketplace.json` | Codex CLI | Marketplace metadata and default-install policy |
| `.mcp.json` | Claude Code | Plugin-provided MCP registration (`command: "archcore"` on PATH) |
| `.codex.mcp.json` | Codex CLI | Plugin-provided MCP registration (`command: "archcore"` on PATH) |
| `docs/cursor.mcp.example.json` | Cursor | Reference MCP config for users to copy into `~/.cursor/mcp.json` or `.cursor/mcp.json`. |
| `hooks/hooks.json` | Claude Code | Hook event config (PascalCase) |
| `hooks/cursor.hooks.json` | Cursor | Hook event config (camelCase + afterMCPExecution) |
| `hooks/codex.hooks.json` | Codex CLI | Hook event config (PascalCase + apply_patch matcher) |
| `commands/*.md` | Codex CLI | Slash command wrappers (7) — thin shims delegating to `skills/<name>/SKILL.md` |
| `agents/archcore-assistant.toml` | Codex CLI | Codex TOML subagent (`sandbox_mode = "workspace-write"`); MD original used by Claude Code/Cursor |
| `agents/archcore-auditor.toml` | Codex CLI | Codex TOML subagent (`sandbox_mode = "read-only"` + `disabled_tools[]`); MD original used by Claude Code/Cursor |
| `rules/archcore-context.mdc` | Cursor | Always-apply context rule |
| `rules/archcore-files.mdc` | Cursor | .archcore/ glob-triggered MCP-only rule |

## Examples

### All skills available as slash commands (visible `/` surface)

```
/archcore:init       — seed an empty .archcore/ on first install
/archcore:capture    — document a module or component
/archcore:decide     — record a decision (ADR) or draft a proposal (RFC)
/archcore:plan       — plan a feature end-to-end (single plan or full flow)
/archcore:audit      — dashboard (default), `--deep` audit, or `--drift` detection
/archcore:context    — rules/decisions for a code area or pickup
/archcore:help       — system guide
```

Total visible in `/` menu: **7 commands**. Every Archcore document type is reachable via these skills or directly through `mcp__archcore__create_document(type=<any>)`. Codex CLI surfaces the same 7 entries via the matching `commands/*.md` wrappers.
