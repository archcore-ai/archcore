---
title: "PRD: Codex CLI Host Support"
status: accepted
tags:
  - "codex"
  - "multi-host"
  - "plugin"
---

**Count note.** The skill and command counts below were 16 when this document was written. `skill-surface-collapse.adr` later took the palette to 7. Read every "16" as the surface of that period; the parity requirements themselves are unchanged.

## Vision

The Archcore plugin runs natively in OpenAI Codex CLI as a third first-class host alongside Claude Code and Cursor, installable through the plugin marketplace, with Codex-native packaging for slash commands, skills, plugin-managed MCP, a hooks config, and a read-only auditor subagent in TOML. Hook execution uses Codex's current hooks runtime under `[features].hooks`, where `codex_hooks` is a deprecated alias, and a plugin-bundled hook still requires user trust. Existing Claude Code and Cursor users see zero regression.

## Problem Statement

Users of OpenAI Codex CLI need the same Archcore surfaces Claude Code users get: skills, MCP tools, hook guardrails, and documentation agents. Codex CLI v0.117.0, released in March 2026, introduced a plugin system with a surface similar to Claude Code, which makes the port technically feasible at low marginal cost: the existing shared core — skills, agents, `bin/`, and `normalize-stdin.sh` — is reusable as it stands, and the per-host adapter pattern of `multi-host-plugin-architecture.adr` was designed for exactly this moment.

## Goals and Success Metrics

| Goal | Metric |
|------|--------|
| Single-command install | `codex plugin marketplace add archcore-ai/plugin` registers the marketplace, and an enabled install loads skills and plugin-managed MCP with no manual `codex mcp add` |
| Skill parity | Every skill is discoverable and invokable in Codex with no modification to an existing `SKILL.md` |
| Slash command parity | Every user-facing workflow is available in Codex as `/archcore:*` through a `commands/*.md` wrapper |
| MCP parity with Claude Code | Plugin-shipped MCP works in Codex, needing no external `codex mcp add` |
| Hook packaging | `hooks/codex.hooks.json` ships the same guardrails as Claude Code with Codex matchers including `apply_patch`; live execution uses the current hooks runtime and the plugin-hook trust flow |
| Auditor subagent | `archcore-auditor` runs with `sandbox_mode = "read-only"`, no file write, and — where supported — `disabled_tools[]` blocking the mutating MCP tools |
| Zero regression | Every existing test passes unchanged, and the Claude Code and Cursor flows are verified manually |
| Shared `bin/` invariant | No host-specific logic is added to a bin script beyond an explicit `codex` branch in `normalize-stdin.sh` |

## Requirements

### Functional

**F1 — plugin manifest.** Create `.codex-plugin/plugin.json` with `name`, `version`, and `description` synchronized to the Claude Code and Cursor manifests, component pointers for `skills`, `hooks`, and `mcpServers` as Codex-relative paths, and an `interface{}` block for marketplace UI metadata.

**F2 — marketplace listing.** Create `.agents/plugins/marketplace.json` at the repo root using the Codex marketplace schema. The entry uses `INSTALLED_BY_DEFAULT` and points the plugin's `source.path` at the `./plugins/archcore` subdirectory, because Codex does not discover a plugin whose manifest sits at the marketplace root — see `subdirectory-plugin-layout.adr` and issue #2. Do not create a legacy `.codex-plugin/marketplace.json`.

**F2a — slash commands.** Create Codex command wrappers under `commands/*.md`, one per user-facing workflow. Each wrapper is a host-adapter shim carrying `description:` frontmatter plus a one-line delegate instruction pointing at its `skills/<name>/SKILL.md`, and no workflow logic.

**F3 — hooks config.** Create `hooks/codex.hooks.json` mapping the active hook functions: SessionStart to `./bin/session-start`; PreToolUse on the matcher `Write|Edit|apply_patch` to `./bin/check-archcore-write` and `./bin/check-code-alignment` at a 1-second timeout; PostToolUse on the MCP mutation matchers to `./bin/validate-archcore` at 3 seconds; PostToolUse on `update_document` to `./bin/check-cascade` at 3 seconds; and PostToolUse on `create_document` or `update_document` to `./bin/check-precision` at 3 seconds. Commands use `${PLUGIN_ROOT}/bin/...`, which is Codex's plugin-hook variable for the installed plugin root, with PascalCase event names.

**F4 — MCP wiring.** Register plugin-shipped MCP through the manifest's `mcpServers` pointer at `./.codex.mcp.json`, whose file uses Codex's documented direct server map — `{ "archcore": { "command": "archcore", "args": ["mcp"] } }` — resolving `archcore` from the host process's PATH, with no wrapper object, no `cwd`, no `env_vars`, and no plugin-relative path.

**F5 — stdin normalization.** Add an explicit `codex` branch to host detection in `bin/lib/normalize-stdin.sh`, keyed on `turn_id` present without `conversation_id` or `hookEventName`. Codex uses snake_case stdin identical to Claude Code, so field extraction mirrors that branch, and the output helpers emit the same context envelope shape.

**F6 — launcher cache for Codex. Obsolete.** This requirement originally extended the bundled launcher to check `$CODEX_PLUGIN_DATA/archcore/cli` before the XDG fallback. The launcher was removed in plugin v0.4.0, and the plugin now ships, caches, and downloads no CLI binary; users install `archcore` globally per https://docs.archcore.ai/cli/install/.

**F7 — skills compatibility.** Every `SKILL.md` works unchanged in Codex. Non-standard frontmatter such as `argument-hint` is tolerated by the Codex loader, and no Codex-specific skill validation is required.

**F8 — subagent TOML conversion.** Convert `agents/archcore-auditor.md` to `agents/archcore-auditor.toml` carrying `name`, `description`, `developer_instructions` ported from the MD body, `sandbox_mode = "read-only"`, and a `disabled_tools` list naming the five mutating MCP tools. Apply the same conversion to the assistant with `sandbox_mode = "workspace-write"` and no `disabled_tools`. Keep both MD originals for Claude Code and Cursor, keep the `developer_instructions` bodies identical between formats, and let `@test/structure/agents.bats` enforce that parity.

**F9 — marketplace install.** `codex plugin marketplace add archcore-ai/plugin` resolves to the GitHub repository and installs without error, with the README updated to carry the command in a Codex CLI install section.

**F10 — documentation.** Record the Codex packaging in `codex-local-plugin-testing.guide` and `component-registry.doc`.

### Non-functional

**NF1 — zero regression.** No change to a skill beyond frontmatter cleanup where needed, no change to existing `bin/` script logic outside the explicit `codex` branch, and every existing test passing unchanged.

**NF2 — single repository.** Codex support lives in the same repository.

**NF3 — shared core invariant.** The Codex addition introduces no host-specific business logic into a skill, an agent, or a hook script, apart from `normalize-stdin.sh`.

## Out of Scope

- Codex Web UI, Codex IDE extensions, and any other client on the Codex API not covered by the CLI's plugin runtime.
- Modifying `SKILL.md` frontmatter beyond the minimum needed to pass the Codex loader.
- New hook events beyond the active set.
- Cross-host MCP wiring uniformity for Cursor, which still requires a user-registered server.

## Dependencies

- `multi-host-plugin-architecture.adr` as the architectural authority for the shared-core and per-host-adapter split.
- `multi-host-implementation.plan` as the predecessor this document continues.
- Codex CLI v0.117.0 or later, available for testing.
- The Archcore CLI installed globally on PATH per https://docs.archcore.ai/cli/install/. The plugin neither bundles nor fetches it, a coupling that `remove-bundled-launcher-global-cli.idea` removed.
