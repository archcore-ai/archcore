---
title: "PRD: Codex CLI Host Support"
status: draft
tags:
  - "codex"
  - "multi-host"
  - "plugin"
---

## Vision

Archcore plugin runs natively in OpenAI Codex CLI as a third first-class host alongside Claude Code (production) and Cursor (implemented), installable via `codex plugin marketplace add archcore-ai/plugin`, with Codex-native packaging for skills, plugin-managed MCP, hooks config, and read-only auditor subagent TOML. Hook execution depends on Codex's `codex_hooks` feature/runtime support, so the plugin ships the documented hook surface while treating live hook execution as a runtime smoke-test item. Zero regression for existing Claude Code and Cursor users. The Multi-Host Compatibility Layer Specification's "Codex CLI" row is promoted from `TBD / Future` to actual implementation values.

## Problem Statement

Users of OpenAI Codex CLI need the same Archcore surfaces Claude Code users get: skills, MCP tools, hook guardrails, and documentation agents. Codex CLI v0.117.0+ (March 2026) introduced a plugin system with a similar surface to Claude Code, making the port technically feasible at low marginal cost — the existing shared core (skills, agents, bin/, launcher, normalize-stdin.sh) is reusable as-is, and the per-host adapter pattern from the Multi-Host Plugin Architecture ADR was designed for exactly this moment. Without Codex support, Archcore's value proposition as a host-agnostic context tool is incomplete and the multi-host architecture investment is under-realized.

## Goals and Success Metrics

| Goal | Metric |
|------|--------|
| Single-command install | `codex plugin marketplace add archcore-ai/plugin` registers the marketplace; enabled installs load skills and plugin-managed MCP without manual `codex mcp add` |
| Skill parity | All 17 skills (`skills/<name>/SKILL.md`) discoverable and invokable in Codex without modifications to existing SKILL.md files |
| MCP parity with Claude Code | Plugin-shipped MCP wiring works in Codex (no external `claude mcp add`-equivalent needed); first MCP tool call triggers launcher resolution exactly as in Claude Code |
| Hook packaging | `hooks/codex.hooks.json` ships the same guardrails as Claude Code with Codex matchers; live execution is verified when `codex_hooks` runtime support is active |
| Auditor subagent | `archcore-auditor` runs in `sandbox_mode = "read-only"` with no file-write capability and no access to mutating MCP tools |
| Zero regression | All existing `test/unit/`, `test/structure/` suites pass unchanged; Claude Code and Cursor flows verified manually |
| Shared bin/ invariant | No host-specific logic added to bin scripts beyond an explicit `codex` branch in `normalize-stdin.sh` |
| Spec promotion | `multi-host-compatibility-layer.spec.md` Supported Hosts table updated: Codex CLI row contains real values, not TBD |

## Requirements

### Functional

**F1 — Plugin Manifest.** Create `.codex-plugin/plugin.json` with required `name`, `version`, `description` synchronized to `.claude-plugin/plugin.json` and `.cursor-plugin/plugin.json`. Component pointers as Codex relative paths (`./...`): `skills`, `hooks`, `mcpServers`. Optional `interface{}` block for marketplace UI metadata (displayName, category, keywords, capabilities, defaultPrompt, brandColor, screenshots). Identical metadata across hosts is mandated by the Compatibility Layer spec invariant.

**F2 — Marketplace Listing.** Create `.agents/plugins/marketplace.json` with the Codex marketplace schema. The entry uses `INSTALLED_BY_DEFAULT` and points the `archcore` plugin at the repo root. Do not create legacy `.codex-plugin/marketplace.json`.

**F3 — Hooks Config.** Create `hooks/codex.hooks.json` mapping the five active hook functions:
- SessionStart → `bin/session-start`
- PreToolUse (matcher `Write|Edit|apply_patch`) → `bin/check-archcore-write` + `bin/check-code-alignment` (both with timeout 1s)
- PostToolUse (matcher MCP create/update/remove + relations) → `bin/validate-archcore` (timeout 3s)
- PostToolUse (matcher `mcp__archcore__update_document`) → `bin/check-cascade` (timeout 3s)
- PostToolUse (matcher `mcp__archcore__create_document|mcp__archcore__update_document`) → `bin/check-precision` (timeout 3s)

Use plugin-relative commands (`./bin/...`) rather than `${CODEX_PLUGIN_ROOT}`; current Codex docs/examples do not require a root-substitution variable. PascalCase event names (same as Claude Code). No `validate-archcore` on Write/Edit PostToolUse path (Compatibility Layer invariant). Runtime execution requires Codex hooks support and `[features].codex_hooks = true`.

**F4 — MCP Wiring.** Plugin-shipped MCP registration through `.codex-plugin/plugin.json` `mcpServers: "./.codex.mcp.json"`. The Codex-specific plugin-root file uses the public Codex examples' `{"mcpServers": {...}}` wrapper and points at the plugin-relative launcher (`./bin/archcore`) with `args: ["mcp"]`.

**F5 — Stdin Normalization.** Add an explicit `codex` branch to `bin/lib/normalize-stdin.sh` host detection. Update the heuristic: presence of `turn_id` without `conversation_id` and without `hookEventName` → `codex`. The existing claude-code field-extraction logic (`tool_name`, `file_path`, `path`) is reused for codex (Codex uses identical snake_case stdin schema). Both PreToolUse and PostToolUse output helpers (`archcore_hook_pretool_info`, `archcore_hook_info`) must emit the `hookSpecificOutput.additionalContext` JSON shape for the `codex` branch (identical to claude-code branch).

**F6 — Launcher Cache for Codex.** Extend `bin/archcore` (POSIX) and `bin/archcore.ps1` (Windows) cache resolution step 3 to check Codex-specific data dirs before XDG fallback:
- POSIX: `$CODEX_PLUGIN_DATA/archcore/cli` → `$CLAUDE_PLUGIN_DATA/archcore/cli` → `$XDG_DATA_HOME/archcore-plugin/cli` → `$HOME/.local/share/archcore-plugin/cli`
- Windows: `$env:CODEX_PLUGIN_DATA\archcore\cli` → `$env:CLAUDE_PLUGIN_DATA\archcore\cli` → `$env:LOCALAPPDATA\archcore-plugin\cli`

If Codex doesn't expose `$CODEX_PLUGIN_DATA`, the existing XDG/LOCALAPPDATA fallbacks already work — no breakage, just slightly less locality.

**F7 — Skills Compatibility.** All 17 `skills/<name>/SKILL.md` files must work unchanged in Codex. The `argument-hint` frontmatter field is non-standard for Codex but is expected to be ignored gracefully (frontmatter spec does not require strict field validation). Verify in spike; if it causes failures, propose minimal frontmatter cleanup that doesn't break Claude Code/Cursor.

**F8 — Subagent TOML Conversion.** Convert `agents/archcore-auditor.md` to `agents/archcore-auditor.toml`:
- `name = "archcore-auditor"`
- `description = "..."` (current frontmatter description)
- `developer_instructions = """..."""` (current MD body)
- `model = "..."` (Codex-equivalent of "sonnet" — to be confirmed)
- `model_reasoning_effort = "high"` (optional, opt-in for thoroughness)
- `sandbox_mode = "read-only"`
- `[mcp_servers.archcore]` if needed for inheritance override
- `disabled_tools = [...]` listing mutating MCP tools (`mcp__archcore__create_document`, `update_document`, `remove_document`, `add_relation`, `remove_relation`) — IF Codex supports per-subagent tool disabling (spike-confirmed)

Keep the original `archcore-auditor.md` for Claude Code/Cursor — do not delete. Do the same conversion for `archcore-assistant.md` if plugin-bundled subagents are supported.

**F9 — Marketplace Install.** `codex plugin marketplace add archcore-ai/plugin` resolves to the GitHub repo and installs without errors. README updated with this command in a "Codex CLI" install section.

**F10 — Spec Update.** Update `multi-host-compatibility-layer.spec.md` Supported Hosts table: Codex CLI row populated with `.codex-plugin/plugin.json`, `hooks/codex.hooks.json`, plugin-shipped `.codex.mcp.json`, status "Implemented". Update relevant Normative Behavior, Constraints, and Conformance items to reference Codex.

### Non-Functional

**NF1 — Zero Regression.** No changes to skills (apart from frontmatter cleanup if F7 spike requires it), no changes to existing `bin/` script logic outside the explicit `codex` branch in `normalize-stdin.sh`, no changes to launcher resolution order (only the cache directory list is extended). All existing tests pass unchanged.

**NF2 — Single Repository.** Codex support lives in the same `archcore-ai/plugin` repo. No forks, no separate branches for distribution.

**NF3 — Shared Core Invariant.** Codex addition must not introduce host-specific business logic to skills, agents, hook scripts (apart from `normalize-stdin.sh`), or the launcher. Per the Multi-Host Plugin Architecture ADR Positive consequence: "Adding a new host requires only a manifest (~10 lines) and hooks config (~30 lines)."

**NF4 — Spike-First Discipline.** Phase 0 spike must resolve all open questions about Codex's plugin runtime BEFORE any production code is written. Unresolved questions become explicit risks with workaround plans, not silent assumptions.

## Out of Scope

- Support for Codex Web UI, Codex IDE extensions, or other clients on the Codex API not covered by Codex CLI's plugin runtime.
- Modifying SKILL.md frontmatter beyond the minimum required to pass Codex's frontmatter validator (if any). Architectural changes to the skill format are a separate initiative.
- Adding new hook events beyond the five already in production. Codex-only events (PermissionRequest, UserPromptSubmit, Stop) remain unused for now.
- Cross-host MCP wiring uniformity for Cursor — Cursor still requires user-registered MCP per the Bundled CLI Launcher ADR Negative consequence. Codex parity does not unblock Cursor parity.
- Marketplace publishing automation (e.g., release scripts that update plugin marketplaces on tag push). Manual coordination acceptable for the first release.

## Dependencies

- Multi-Host Plugin Architecture ADR — architectural authority for the shared-core / per-host-adapter split.
- Multi-Host Compatibility Layer Specification — current spec; will be amended by F10.
- Bundled CLI Launcher ADR — launcher resolution order; F6 extends step 3 cache directory list.
- Multi-Host Implementation Plan — predecessor plan; Codex work is a continuation, not a replacement.
- Phase 0 spike resolution: Codex uses plugin-relative commands for bundled MCP/hooks packaging; plugin-managed MCP loads from the manifest's `mcpServers` pointer; plugin-local hook execution remains gated by Codex hook feature/runtime support; skills load from installed plugin cache; subagent TOML files are packaged side-by-side with MD agents.
- Codex CLI v0.117.0+ available for testing (the version that introduced the plugin system).
