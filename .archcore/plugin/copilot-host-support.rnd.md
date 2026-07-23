---
title: "GitHub Copilot Host Support — Verified Plugin Contract and Install Path"
status: accepted
tags:
  - "copilot"
  - "hooks"
  - "multi-host"
  - "plugin"
  - "roadmap"
---

## Goal

Verify, against official GitHub documentation, that the Archcore plugin can ship on GitHub Copilot as a thin adapter over the portable core — and pin down the install path, hook contract, and required `bin/` adaptations before implementation. Follows the host-integration research (deep-rnd, 2026-07) that ranked Copilot the top plugin target.

## Questions

1. Can Copilot install the plugin straight from our repo subdirectory (`plugins/archcore/`)?
2. Does Copilot read our existing Claude-layout manifest and hooks, and what are the exact hook events and payloads?
3. What must change in `bin/` for the guard semantics to hold?
4. What are the surface limitations (CLI vs IDE vs cloud agent)?

## Approach

Documentation sweep of docs.github.com (cli-plugin-reference, hooks-reference, about-plugins, plugins-creating, add-skills, custom-agents-configuration, cli-config-dir-reference, configure-mcp-servers) and the `github/copilot-cli` changelog, performed 2026-07-05. Copilot's plugin-authoring path is itself fully documented: the plugins-creating how-to, the hooks tutorial, and the CLI plugin reference cover manifest, hooks config schema, payloads, and install specs for the **native** format; only the Claude-compatibility layer is partially documented.

## Findings

**Install path — confirmed.** `copilot plugin install OWNER/REPO:PATH/TO/PLUGIN` is a documented specification form (subdirectory install from a GitHub repo). Declarative install exists via `enabledPlugins` in `~/.copilot/settings.json` or `.github/copilot/settings.json`; Copilot also reads the shared cross-tool subset (incl. `enabledPlugins`, `hooks`) from repo-level `.claude/settings.json`. [cli-plugin-reference; cli-config-dir-reference]

**Manifest — confirmed.** Manifest discovery order: `.plugin/plugin.json`, `plugin.json`, `.github/plugin/plugin.json`, `.claude-plugin/plugin.json`. Our existing manifest is discoverable as-is; a dedicated Copilot manifest at `.plugin/plugin.json` is found *before* the Claude one, allowing per-host component paths without touching the Claude manifest. Manifest fields cover `agents`, `skills`, `commands`, `hooks` (file or inline), `mcpServers` (file or inline); component defaults are `agents/`, `skills/`, `hooks.json` or `hooks/hooks.json`, `.mcp.json`. Changelog v1.0.10: plugins using `.claude-plugin/` manifest dirs load their MCP servers correctly. [cli-plugin-reference; copilot-cli changelog]

**Hooks — confirmed, two authoring formats.** Native events are camelCase (`sessionStart`, `preToolUse`, `postToolUse`, `postToolUseFailure`, `sessionEnd`, `subagentStart`/`subagentStop`, `userPromptSubmitted`, `agentStop`, …) with a fully documented config schema (`version`, per-event entry list, `bash`/`powershell`/`command` variants, `cwd`, `env`, `timeoutSec` default 30, regex `matcher` compiled as `^(?:PATTERN)$`). PascalCase names select the Claude-compatible format: snake_case payload fields (`tool_name`, `tool_input`), Claude matcher semantics (`Edit|Write`, `*`), Claude tool-name mapping (`create`→`Write`, `apply_patch`/`str_replace_editor`→`Edit`, `bash`→`Bash`). Payload arrives as JSON on stdin in both formats. Plugin hooks receive `CLAUDE_PLUGIN_ROOT`/`COPILOT_PLUGIN_ROOT`/`PLUGIN_ROOT` and `CLAUDE_PROJECT_DIR` env vars, plus `{{project_dir}}`/`{{plugin_data_dir}}` template variables in hook configs. **Critical:** a `preToolUse` deny requires stdout JSON `{"permissionDecision":"deny","permissionDecisionReason":…}`. Exit code 2 is a *warning* in Copilot (stderr shown to user, run continues) — the Claude "exit 2 = block" idiom does not block. Any other non-zero exit denies (fail-closed); a timeout fails **open**. [hooks-reference; hooks tutorial]

**MCP — confirmed for CLI.** Plugins ship `.mcp.json` at the plugin root (or `mcpServers` in the manifest); plugin MCP configs load with last-wins precedence over `~/.copilot/mcp-config.json`. Our `.mcp.json` names `archcore` from PATH with no plugin-root variables, so no expansion question arises. Cloud agent and code review support MCP **tools only** (no resources/prompts) and no OAuth remote servers. [about-plugins; configure-mcp-servers]

**Skills and agents — confirmed.** Plugin skills use the Claude layout `skills/NAME/SKILL.md`; recognized frontmatter: `name`, `description`, `license`, `allowed-tools`. Skills dedup by `name`; project/personal skills win over plugin skills. Repo-level `.claude/skills` and `.agents/skills` are also read (home-level `~/.claude` no longer is). Plugin agents are documented as `*.agent.md` files in `agents/`; the general custom-agents reference accepts both `.md` and `.agent.md`, but plugin docs show only `.agent.md` — whether plain `NAME.md` loads from a plugin is the one open compatibility question. `description` is required; Claude tool aliases are accepted case-insensitively and unrecognized tool names are ignored. [add-skills; plugins-creating; custom-agents-configuration]

**Surface limitations.** Plugins target Copilot CLI and cloud agent; self-serve plugin install in VS Code agent mode is not documented (enterprise-managed plugins only, public preview). In cloud-agent sandboxes the only default hook source is repo-level `.github/hooks/*.json` — user-level files and installed plugins are absent by default, and whether `enabledPlugins` materializes plugin MCP/skills there is contradictory across docs. [about-plugins; hooks-reference; github.blog changelogs]

## Recommendation

All three design decisions are made (2026-07-05) and recorded in `copilot-adapter-design.adr`:

- **Install path: subdir spec `archcore-ai/plugin:plugins/archcore`**, plus a marketplace listing later. Consistent with `subdirectory-plugin-layout.adr`.
- **Release scope: Copilot CLI only.** VS Code (no self-serve plugin install) and cloud agent (no plugin hooks in the sandbox; MCP tools-only) are documented as limitations, not supported surfaces. A repo-level `.github/hooks/*.json` template for cloud-agent users is a separable future item.
- **Hook wiring: design B — native adapter files.** A dedicated `.plugin/plugin.json` manifest + `hooks/copilot.hooks.json` in the documented native camelCase format, aligned with the Cursor/Codex per-host precedent. Design A (Claude-compat reuse of `hooks/hooks.json`) was rejected: it depends on partially documented compat behavior and would require live probes to pin.
- `bin/` guard scripts gain a Copilot output branch — stdout `permissionDecision` JSON keyed on `ARCHCORE_HOST=copilot` — per `host-adapter-contract.spec`; `normalize-stdin.sh` already detects `copilot`.

## Next Action

Implementation per `copilot-adapter-design.adr` and `host-adapter-contract.spec`: (1) `.plugin/plugin.json` manifest + `hooks/copilot.hooks.json` + MCP wiring; (2) Copilot deny-output branch in `bin/` guard scripts with bats fixtures; (3) `copilot` row in the host-coverage-matrix structure test. Release verification: install smoke test on Copilot CLI (subdir install; sessionStart/preToolUse/postToolUse fire and route to `bin/`; MCP loads; deny JSON blocks a `.archcore/` write), one agent-naming check (plain `NAME.md` from a plugin `agents/` dir; fallback: ship `*.agent.md` copies), and the standard three-probe protocol recorded in `hooks-validation-system.spec`.