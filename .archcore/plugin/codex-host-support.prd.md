---
title: "PRD: Codex CLI Host Support"
status: accepted
tags:
  - "codex"
  - "multi-host"
  - "plugin"
---

## Vision

Archcore plugin runs natively in OpenAI Codex CLI as a third first-class host alongside Claude Code (production) and Cursor (implemented), installable via the plugin marketplace, with Codex-native packaging for slash commands, skills, plugin-managed MCP, hooks config, and a read-only auditor subagent TOML. Hook execution uses Codex's current hooks runtime (`[features].hooks`; `codex_hooks` is a deprecated alias) and still requires user trust for plugin-bundled hooks. Zero regression for existing Claude Code and Cursor users.

## Problem Statement

Users of OpenAI Codex CLI need the same Archcore surfaces Claude Code users get: skills, MCP tools, hook guardrails, and documentation agents. Codex CLI v0.117.0+ (March 2026) introduced a plugin system with a similar surface to Claude Code, making the port technically feasible at low marginal cost — the existing shared core (skills, agents, bin/, normalize-stdin.sh) is reusable as-is, and the per-host adapter pattern from the Multi-Host Plugin Architecture ADR was designed for exactly this moment.

## Goals and Success Metrics

| Goal | Metric |
|------|--------|
| Single-command install | `codex plugin marketplace add archcore-ai/plugin` registers the marketplace; enabled installs load skills and plugin-managed MCP without manual `codex mcp add` |
| Skill parity | All 16 skills discoverable and invokable in Codex without modifications to existing SKILL.md files |
| Slash command parity | All 16 user-facing Archcore workflows available in Codex as `/archcore:*` via `commands/*.md` wrappers |
| MCP parity with Claude Code | Plugin-shipped MCP works in Codex (no external `codex mcp add` needed) |
| Hook packaging | `hooks/codex.hooks.json` ships the same guardrails as Claude Code with Codex matchers (incl. `apply_patch`); live execution uses the current Codex hooks runtime and plugin-hook trust flow |
| Auditor subagent | `archcore-auditor` runs in `sandbox_mode = "read-only"` with no file-write and (where supported) `disabled_tools[]` blocking mutating MCP tools |
| Zero regression | All existing tests pass unchanged; Claude Code and Cursor flows verified manually |
| Shared bin/ invariant | No host-specific logic added to bin scripts beyond an explicit `codex` branch in `normalize-stdin.sh` |

## Requirements

### Functional

**F1 — Plugin Manifest.** Create `.codex-plugin/plugin.json` with `name`, `version`, `description` synchronized to `.claude-plugin/plugin.json` and `.cursor-plugin/plugin.json`. Component pointers (Codex relative paths, `./...`): `skills`, `hooks`, `mcpServers`. `interface{}` block for marketplace UI metadata.

**F2 — Marketplace Listing.** Create `.agents/plugins/marketplace.json` with the Codex marketplace schema. Entry uses `INSTALLED_BY_DEFAULT` and points the `archcore` plugin at the repo root. Do not create legacy `.codex-plugin/marketplace.json`.

**F2a — Slash Commands.** Create root-level Codex command wrappers under `commands/*.md` for every user-facing Archcore workflow. Wrappers are host-adapter shims: `description:` frontmatter plus a one-line delegate instruction pointing at `skills/<name>/SKILL.md`. No workflow logic.

**F3 — Hooks Config.** Create `hooks/codex.hooks.json` mapping the active hook functions:

- SessionStart → `./bin/session-start`
- PreToolUse (matcher `Write|Edit|apply_patch`) → `./bin/check-archcore-write` + `./bin/check-code-alignment` (timeout 1s)
- PostToolUse (MCP mutation matchers) → `./bin/validate-archcore` (3s)
- PostToolUse (`mcp__archcore__update_document`) → `./bin/check-cascade` (3s)
- PostToolUse (`mcp__archcore__create_document|update_document`) → `./bin/check-precision` (3s)

Commands use `${PLUGIN_ROOT}/bin/...`, the Codex plugin-hook environment variable for the installed plugin root. PascalCase event names. Runtime execution uses `[features].hooks`; `codex_hooks` is only a deprecated alias.

**F4 — MCP Wiring.** Plugin-shipped MCP registration through `.codex-plugin/plugin.json` `mcpServers: "./.codex.mcp.json"`. The `.codex.mcp.json` file at the plugin root uses Codex's documented direct server map shape `{ "archcore": { "command": "archcore", "args": ["mcp"] } }` — `command: "archcore"` resolved via PATH from the host process. No wrapper object, no `cwd`, no `env_vars`, no plugin-relative paths.

**F5 — Stdin Normalization.** Add an explicit `codex` branch to `bin/lib/normalize-stdin.sh` host detection (heuristic: `turn_id` without `conversation_id`/`hookEventName`). Codex uses snake_case stdin identical to Claude Code, so field extraction mirrors the `claude-code` branch. Output helpers emit `hookSpecificOutput.additionalContext` for `codex` (same shape as Claude Code).

**F6 — ~~Launcher Cache for Codex~~.** **Obsolete.** F6 originally extended the bundled launcher to check `$CODEX_PLUGIN_DATA/archcore/cli` before XDG fallback. The launcher was removed in plugin v0.4.0 (see `remove-bundled-launcher-global-cli.idea`); the plugin no longer ships, caches, or downloads any CLI binary. Users install `archcore` globally per https://docs.archcore.ai/cli/install/.

**F7 — Skills Compatibility.** All 16 SKILL.md files work unchanged in Codex. Non-standard frontmatter fields (`argument-hint`) are tolerated by Codex's loader. No Codex-specific skill validation required.

**F8 — Subagent TOML Conversion.** Convert `agents/archcore-auditor.md` to `agents/archcore-auditor.toml`:

- `name`, `description`, `developer_instructions` (port from MD body)
- `sandbox_mode = "read-only"`
- `disabled_tools = [...]` listing the five mutating MCP tools (`mcp__archcore__create_document`, `update_document`, `remove_document`, `add_relation`, `remove_relation`)

Same conversion for `archcore-assistant.md` → `archcore-assistant.toml` with `sandbox_mode = "workspace-write"` (no `disabled_tools`). Keep both MD originals for Claude Code/Cursor — TOML and MD must keep identical `developer_instructions` bodies; parity enforced by `test/structure/agents.bats`.

**F9 — Marketplace Install.** `codex plugin marketplace add archcore-ai/plugin` resolves to the GitHub repo and installs without errors. README updated with this command in a "Codex CLI" install section.

**F10 — Docs.** Codex packaging documented in `codex-local-plugin-testing.guide` and `component-registry.doc`.

### Non-Functional

**NF1 — Zero Regression.** No changes to skills (apart from frontmatter cleanup if needed), no changes to existing `bin/` script logic outside the explicit `codex` branch in `normalize-stdin.sh`. All existing tests pass unchanged.

**NF2 — Single Repository.** Codex support lives in the same `archcore-ai/plugin` repo.

**NF3 — Shared Core Invariant.** Codex addition must not introduce host-specific business logic to skills, agents, or hook scripts (apart from `normalize-stdin.sh`).

## Out of Scope

- Support for Codex Web UI, Codex IDE extensions, or other clients on the Codex API not covered by Codex CLI's plugin runtime.
- Modifying SKILL.md frontmatter beyond minimum needed to pass Codex's loader.
- New hook events beyond the active set.
- Cross-host MCP wiring uniformity for Cursor (Cursor still requires user-registered MCP).

## Dependencies

- Multi-Host Plugin Architecture ADR — architectural authority for the shared-core / per-host-adapter split.
- Multi-Host Implementation Plan — predecessor; this PRD continues that work.
- Codex CLI v0.117.0+ available for testing.
- Archcore CLI installed globally on PATH per https://docs.archcore.ai/cli/install/ (the plugin does not bundle or fetch the CLI; previously coupled to the bundled-launcher ADR, now decoupled per `remove-bundled-launcher-global-cli.idea`).
