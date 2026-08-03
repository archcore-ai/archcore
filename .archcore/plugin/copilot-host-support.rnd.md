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

Two findings below were corrected on 2026-07-27 against the same pages, and two more on 2026-08-03 against a live CLI. See **Re-verification** and the dated updates at the end — read those before treating any hook or MCP claim here as current.

## Findings

**Install path — confirmed.** `copilot plugin install OWNER/REPO:PATH/TO/PLUGIN` is a documented specification form (subdirectory install from a GitHub repo). Declarative install exists via `enabledPlugins` in `~/.copilot/settings.json` or `.github/copilot/settings.json`; Copilot also reads the shared cross-tool subset (incl. `enabledPlugins`, `hooks`) from repo-level `.claude/settings.json`. [cli-plugin-reference; cli-config-dir-reference]

**Manifest — confirmed.** Manifest discovery order: `.plugin/plugin.json`, `plugin.json`, `.github/plugin/plugin.json`, `.claude-plugin/plugin.json`. Our existing manifest is discoverable as-is; a dedicated Copilot manifest at `.plugin/plugin.json` is found *before* the Claude one, allowing per-host component paths without touching the Claude manifest. Manifest fields cover `agents`, `skills`, `commands`, `hooks` (file or inline), `mcpServers` (file or inline); component defaults are `agents/`, `skills/`, `hooks.json` or `hooks/hooks.json`, `.mcp.json`. Changelog v1.0.10: plugins using `.claude-plugin/` manifest dirs load their MCP servers correctly. [cli-plugin-reference; copilot-cli changelog]

Read that discovery order again in the light of the 2026-08-03 update: it is not merely "which manifest we may author". It is a **fallback chain**, per field. A key absent from `.plugin/plugin.json` is looked for in `.claude-plugin/plugin.json`, and the changelog line about `.claude-plugin/` MCP servers loading "correctly" says the same thing from the other side. Both facts sat in this document from day one; nobody drew the consequence for `mcpServers` until a live probe forced it.

**Hooks — confirmed, two authoring formats.** Native events are camelCase (`sessionStart`, `preToolUse`, `postToolUse`, `postToolUseFailure`, `sessionEnd`, `subagentStart`/`subagentStop`, `userPromptSubmitted`, `agentStop`, …) with a fully documented config schema (`version`, per-event entry list, `bash`/`powershell`/`command` variants, `cwd`, `env`, `timeoutSec` default 30, regex `matcher` compiled as `^(?:PATTERN)$`). PascalCase names select the Claude-compatible format: snake_case payload fields (`tool_name`, `tool_input`), Claude matcher semantics (`Edit|Write`, `*`), Claude tool-name mapping (`create`→`Write`, `apply_patch`/`str_replace_editor`→`Edit`, `bash`→`Bash`). Payload arrives as JSON on stdin in both formats.

Two claims this paragraph carried until 2026-07-27 did **not** hold up on re-reading, and both were load-bearing. The env-var list (`CLAUDE_PLUGIN_ROOT`/`COPILOT_PLUGIN_ROOT`/`PLUGIN_ROOT` plus `CLAUDE_PROJECT_DIR`) and the exit-code semantics ("exit 2 is a *warning*; a deny requires stdout JSON") are corrected below. A timeout failing **open** is the one hook claim that survived unchanged.

**MCP — confirmed for CLI.** Plugins ship `.mcp.json` at the plugin root (or `mcpServers` in the manifest); plugin MCP configs load with last-wins precedence over `~/.copilot/mcp-config.json`. Cloud agent and code review support MCP **tools only** (no resources/prompts) and no OAuth remote servers. [about-plugins; configure-mcp-servers]

This paragraph once continued: *"Our `.mcp.json` names `archcore` from PATH with no plugin-root variables, so no expansion question arises."* True about expansion, and beside the point. The question that mattered was not how the command string expands but **whether the file should be discoverable at all** — see the 2026-08-03 update. The plugin no longer ships any filename Copilot discovers.

**Skills and agents — confirmed.** Plugin skills use the Claude layout `skills/NAME/SKILL.md`; recognized frontmatter: `name`, `description`, `license`, `allowed-tools`. Skills dedup by `name`; project/personal skills win over plugin skills. Repo-level `.claude/skills` and `.agents/skills` are also read (home-level `~/.claude` no longer is). Plugin agents are documented as `*.agent.md` files in `agents/`; the general custom-agents reference accepts both `.md` and `.agent.md`, but plugin docs show only `.agent.md` — whether plain `NAME.md` loads from a plugin is the one open compatibility question. `description` is required; Claude tool aliases are accepted case-insensitively and unrecognized tool names are ignored. [add-skills; plugins-creating; custom-agents-configuration]

**Surface limitations.** Plugins target Copilot CLI and cloud agent; self-serve plugin install in VS Code agent mode is not documented (enterprise-managed plugins only, public preview). In cloud-agent sandboxes the only default hook source is repo-level `.github/hooks/*.json` — user-level files and installed plugins are absent by default, and whether `enabledPlugins` materializes plugin MCP/skills there is contradictory across docs. [about-plugins; hooks-reference; github.blog changelogs]

## Re-verification (2026-07-27)

Triggered by a field report, not a scheduled sweep: a user session emitted `/bin/sh: /bin/session-start: No such file or directory`, which is what `"${COPILOT_PLUGIN_ROOT}"/bin/session-start` collapses to when the variable is empty. Re-read cli-plugin-reference, hooks-reference and the hooks concepts page.

**`COPILOT_PLUGIN_ROOT` is undocumented.** Neither `COPILOT_PLUGIN_ROOT` nor `CLAUDE_PLUGIN_ROOT` appears anywhere on the plugin reference or the hooks reference. The only documented plugin-path variable is `${PLUGIN_ROOT}` — *"Use `${PLUGIN_ROOT}` to reference paths within the plugin directory"* — and on the plugin reference it appears in the **LSP server** section, plus `pluginRoot?` as an optional `marketplace.json` field. For writable state the documented pair is `${COPILOT_PLUGIN_DATA}` / `${CLAUDE_PLUGIN_DATA}`. The original three-name list has no source we can point at today; it may have been read off a changelog fragment or inferred from the Claude-compat layer.

**Exit codes are the reverse of what was recorded.** Verbatim: *"exit 2 is treated as a deny: any stdout JSON is merged with the deny decision and the tool call is denied even if that JSON reports `permissionDecision: allow`"*, and *"Other non-zero exits denies the tool call with `Denied by preToolUse hook (hook errored)`"*, and *"Timeouts are fail-open for every event, including `preToolUse` and admin-deployed policy hooks"*. So on Copilot **every** non-zero exit denies. Our stdout-JSON arm remains correct — exit 0 has its stdout parsed as hook output — but the reason it exists changes: it is how a deny carries `permissionDecisionReason`, not the only way a deny lands.

The corollary is the actual severity of the field report. A guard that cannot start is an "other non-zero exit", so an unresolved plugin root denies every `create|edit|str_replace_editor|apply_patch` call rather than merely printing noise at session start. `hooks/copilot.hooks.json` now probes three candidate roots with `-x` and exits 0 with a warning when none resolves; `copilot-plugin.bats` runs those commands rather than string-matching them.

**Hooks run in two surfaces, and VS Code is not one of them.** *"Hooks are supported in two Copilot surfaces: Copilot CLI and Copilot cloud agent."* The concepts page agrees — cloud agent on GitHub, and Copilot CLI in the terminal. No documentation covers hooks in VS Code agent mode. The field report came from a VS Code Copilot Chat session, which ships its own Copilot CLI binary under the extension's `globalStorage`; hook machinery evidently fires there, in a configuration GitHub does not document, so nothing about variable injection is guaranteed on that surface. Treat the CLI-only release scope below as covering hooks too, not just plugin install.

**CLI hook config locations** (unchanged, recorded here because the re-read enumerated them): policy files under `/etc/github-copilot/policy.d/*.json`, repo `.github/hooks/*.json`, user `~/.copilot/hooks/` or `$COPILOT_HOME/hooks/`, inline hooks in `.github/copilot/settings.json` / `.github/copilot/settings.local.json` / `~/.copilot/settings.json`, and plugin-provided `hooks.json`. Cloud agent reads only `.github/hooks/*.json` from the cloned repo.

## Recommendation

All three design decisions are made (2026-07-05) and recorded in `copilot-adapter-design.adr`:

- **Install path: subdir spec `archcore-ai/plugin:plugins/archcore`**, plus a marketplace listing later. Consistent with `subdirectory-plugin-layout.adr`.
- **Release scope: Copilot CLI only.** VS Code (no self-serve plugin install, and no documented hook support at all) and cloud agent (no plugin hooks in the sandbox; MCP tools-only) are documented as limitations, not supported surfaces. A repo-level `.github/hooks/*.json` template for cloud-agent users is a separable future item.
- **Hook wiring: design B — native adapter files.** A dedicated `.plugin/plugin.json` manifest + `hooks/copilot.hooks.json` in the documented native camelCase format, aligned with the Cursor/Codex per-host precedent. Design A (Claude-compat reuse of `hooks/hooks.json`) was rejected: it depends on partially documented compat behavior and would require live probes to pin.
- `bin/` guard scripts gain a Copilot output branch — stdout `permissionDecision` JSON keyed on `ARCHCORE_HOST=copilot` — per `host-adapter-contract.spec`; `normalize-stdin.sh` already detects `copilot`.

## Next Action

Implementation per `copilot-adapter-design.adr` and `host-adapter-contract.spec`: (1) `.plugin/plugin.json` manifest + `hooks/copilot.hooks.json` + MCP wiring; (2) Copilot deny-output branch in `bin/` guard scripts with bats fixtures; (3) `copilot` row in the host-coverage-matrix structure test. Release verification: install smoke test on Copilot CLI (subdir install; sessionStart/preToolUse/postToolUse fire and route to `bin/`; MCP loads; deny JSON blocks a `.archcore/` write), one agent-naming check (plain `NAME.md` from a plugin `agents/` dir; fallback: ship `*.agent.md` copies), and the standard three-probe protocol recorded in `hooks-validation-system.spec`.

**Open, and only a live CLI session can close it.** Which plugin-root variable Copilot actually injects, and whether it injects one at all outside `copilot plugin install`, is unresolved from documentation — the candidate chain is a hedge, not an answer. The second open item, added 2026-08-03: whether Copilot also auto-loads the Claude-format `hooks/hooks.json` alongside the manifest-declared `hooks/copilot.hooks.json`. Conventional discovery is known to run independently of the manifest for MCP (see below), so the same may hold for hooks — and if it does, every guard fires twice with an unset `${CLAUDE_PLUGIN_ROOT}`. `host-probe-protocol.spec` records `deferred:not-yet-run` for both.

## Update 2026-08-03 — plugin MCP discovery settled, in two passes

**First pass — the manifest key was never the only source.** `copilot mcp list` on Copilot CLI 1.0.76 with plugin 0.6.1 installed lists `Plugin servers: archcore (local)` from any cwd, with `mcpServers` absent from `.plugin/plugin.json` and no project or user-level config present. The concepts page's discovery rule is real. Reproduced in an isolated `COPILOT_HOME` by `test/integration/copilot-plugin-smoke.bats`.

**Second pass — the damage was a name collision, and the first remedy did not work.** The plugin server and the project server that `archcore init --agent copilot` registers share the key `archcore`; Copilot merges user → workspace → plugins **last-wins**, so the plugin entry replaced the user's. Measured with a sentinel command in the workspace config, fresh `COPILOT_HOME` per arm:

| plugin layout | surviving `archcore` |
| --- | --- |
| no plugin installed (control) | workspace |
| `.mcp.json` at plugin root | plugin (`cwd=${PLUGIN_ROOT}`) |
| renamed, no manifest key | workspace |
| renamed + key in `.claude-plugin/plugin.json` | **plugin** |
| `.mcp.json` kept + empty key in `.plugin/plugin.json` | plugin |
| renamed + both keys (shipped in 0.6.2) | workspace |

Row four is the finding that cost a release: renaming and pointing Claude Code at the new name via `.claude-plugin/plugin.json` re-armed **Copilot**, through the manifest fallback chain this document recorded in July. The shipped configuration therefore needs all three parts — a filename off the discovery list, the Claude manifest key, and an empty `mcpServers` in `.plugin/plugin.json` to stop the fallback.

**Discovery is per directory, and takes two filenames.** Probing a git repo with configs planted at four levels: Copilot reads a config from **every** directory between cwd and the git root, preferring `.mcp.json` and falling back to `.github/mcp.json` only where the first is absent in that same directory. `bin/session-start`'s wiring advisory mirrors this exactly.

**Method note.** Two measurement traps cost time and are worth carrying forward. `copilot --plugin-dir` mounts a plugin but does **not** load its MCP servers, so any smoke test built on that flag passes whether or not the bug exists — the real install layout plus a registered entry in `config.json` is required. And a fresh `COPILOT_HOME` is mandatory per arm: an already-installed copy in the default home silently shadows every variant under test.

Countermeasures and full consequences: `copilot-mcp-architecture.adr.md`. The same rename also removed a latent plugin-shipped MCP on Cursor, whose loader accepts both `.mcp.json` and `mcp.json` — see `cursor-mcp-architecture.adr.md`.
