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

Verify against the official GitHub documentation that the Archcore plugin can ship on GitHub Copilot as a thin adapter over the portable core, and pin down the install path, the hook contract, and the required `bin/` adaptations before implementation. This follows the host-integration research of 2026-07 that ranked Copilot the top plugin target.

## Questions

1. Can Copilot install the plugin straight from the repository subdirectory `plugins/archcore/`?
2. Does Copilot read the existing Claude-layout manifest and hooks, and what are the exact hook events and payloads?
3. What must change in `bin/` for the guard semantics to hold?
4. What are the surface limitations across CLI, IDE, and cloud agent?

## Approach

A documentation sweep of docs.github.com — cli-plugin-reference, hooks-reference, about-plugins, plugins-creating, add-skills, custom-agents-configuration, cli-config-dir-reference, and configure-mcp-servers — plus the `github/copilot-cli` changelog, performed on 2026-07-05. Copilot's plugin-authoring path is itself fully documented for the native format, covering the manifest, the hooks config schema, the payloads, and the install specs; only the Claude-compatibility layer is partially documented.

Two findings below were corrected on 2026-07-27 against the same pages, and two more on 2026-08-03 against a live CLI. Read the re-verification and the dated update before treating any hook or MCP claim here as current.

## Findings

**Install path — confirmed.** `copilot plugin install OWNER/REPO:PATH/TO/PLUGIN` is a documented specification form for a subdirectory install from a GitHub repository. Declarative install exists through `enabledPlugins` in `~/.copilot/settings.json` or `.github/copilot/settings.json`, and Copilot also reads the shared cross-tool subset — including `enabledPlugins` and `hooks` — from a repo-level `.claude/settings.json`.

**Manifest — confirmed.** Discovery order is `.plugin/plugin.json`, then `plugin.json`, then `.github/plugin/plugin.json`, then `.claude-plugin/plugin.json`, so the existing manifest is discoverable as-is and a dedicated Copilot manifest is found before the Claude one, allowing per-host component paths without touching the Claude manifest. Manifest fields cover `agents`, `skills`, `commands`, `hooks` as file or inline, and `mcpServers` as file or inline, with component defaults of `agents/`, `skills/`, `hooks.json` or `hooks/hooks.json`, and `.mcp.json`. Changelog v1.0.10 records that a plugin using a `.claude-plugin/` manifest directory loads its MCP servers correctly.

Read that discovery order again in light of the 2026-08-03 update. It is not merely which manifest may be authored; it is a **fallback chain, per field**. A key absent from `.plugin/plugin.json` is looked for in `.claude-plugin/plugin.json`, and the changelog line says the same thing from the other side. Both facts sat in this document from day one, and nobody drew the consequence for `mcpServers` until a live probe forced it.

**Hooks — confirmed, in two authoring formats.** Native events are camelCase — `sessionStart`, `preToolUse`, `postToolUse`, `postToolUseFailure`, `sessionEnd`, `subagentStart` and `subagentStop`, `userPromptSubmitted`, `agentStop` — with a fully documented config schema covering `version`, the per-event entry list, the `bash`, `powershell`, and `command` variants, `cwd`, `env`, a `timeoutSec` defaulting to 30, and a regex `matcher` compiled as `^(?:PATTERN)$`. PascalCase names select the Claude-compatible format, with snake_case payload fields, Claude matcher semantics, and Claude tool-name mapping where `create` becomes `Write`, `apply_patch` and `str_replace_editor` become `Edit`, and `bash` becomes `Bash`. The payload arrives as JSON on stdin in both formats.

Two claims this paragraph carried until 2026-07-27 did not hold up on re-reading, and both were load-bearing: the environment-variable list, and the exit-code semantics that recorded exit 2 as a warning requiring stdout JSON for a deny. Both are corrected below. A timeout failing **open** is the one hook claim that survived unchanged.

**MCP — confirmed for the CLI.** A plugin ships `.mcp.json` at its root, or `mcpServers` in the manifest, and a plugin MCP config loads with last-wins precedence over `~/.copilot/mcp-config.json`. The cloud agent and code review support MCP tools only, with no resources or prompts, and no OAuth remote servers.

This paragraph once continued that the MCP config names `archcore` from PATH with no plugin-root variables, so no expansion question arises. That was true about expansion and beside the point: the question that mattered was not how the command string expands but whether the file should be discoverable at all. The plugin now ships no filename Copilot discovers.

**Skills and agents — confirmed.** A plugin skill uses the Claude layout at `skills/NAME/SKILL.md`, with `name`, `description`, `license`, and `allowed-tools` recognized in frontmatter. Skills deduplicate by `name`, and a project or personal skill wins over a plugin skill. Repo-level `.claude/skills` and `.agents/skills` are also read, while home-level `~/.claude` no longer is. A plugin agent is documented as an `*.agent.md` file in `agents/`; the general custom-agents reference accepts both `.md` and `.agent.md`, but the plugin documentation shows only `.agent.md`, so whether a plain `NAME.md` loads from a plugin was the one open compatibility question. `description` is required, Claude tool aliases are accepted case-insensitively, and an unrecognized tool name is ignored.

**Surface limitations.** Plugins target Copilot CLI and the cloud agent. Self-serve plugin install in VS Code agent mode is not documented — enterprise-managed plugins only, in public preview. In a cloud-agent sandbox the only default hook source is a repo-level `.github/hooks/*.json`, user-level files and installed plugins are absent by default, and whether `enabledPlugins` materializes plugin MCP or skills there is contradictory across the documentation.

## Re-verification (2026-07-27)

Triggered by a field report rather than a scheduled sweep: a user session emitted `/bin/sh: /bin/session-start: No such file or directory`, which is what `"${COPILOT_PLUGIN_ROOT}"/bin/session-start` collapses to when the variable is empty.

**`COPILOT_PLUGIN_ROOT` is undocumented.** Neither it nor `CLAUDE_PLUGIN_ROOT` appears anywhere on the plugin reference or the hooks reference. The only documented plugin-path variable is `${PLUGIN_ROOT}` — "Use `${PLUGIN_ROOT}` to reference paths within the plugin directory" — which appears on the plugin reference in the LSP server section, alongside an optional `pluginRoot?` field in `marketplace.json`. For writable state the documented pair is `${COPILOT_PLUGIN_DATA}` and `${CLAUDE_PLUGIN_DATA}`. The original three-name list has no source that can be pointed at today; it may have been read off a changelog fragment or inferred from the Claude-compatibility layer.

**Exit codes are the reverse of what was recorded.** Verbatim: "exit 2 is treated as a deny: any stdout JSON is merged with the deny decision and the tool call is denied even if that JSON reports `permissionDecision: allow`"; "Other non-zero exits denies the tool call with `Denied by preToolUse hook (hook errored)`"; and "Timeouts are fail-open for every event, including `preToolUse` and admin-deployed policy hooks". So on Copilot every non-zero exit denies. The stdout-JSON arm remains correct, since exit 0 has its stdout parsed as hook output, but its reason for existing changes: it is how a deny carries its reason text, not the only way a deny lands.

The corollary is the actual severity of the field report. A guard that cannot start is an "other non-zero exit", so an unresolved plugin root denies every matched mutation call rather than merely printing noise at session start. `hooks/copilot.hooks.json` now probes three candidate roots with `-x` and exits 0 with a warning when none resolves, and `copilot-plugin.bats` runs those commands rather than string-matching them.

**Hooks run in two surfaces, and VS Code is not one of them.** Verbatim: "Hooks are supported in two Copilot surfaces: Copilot CLI and Copilot cloud agent." The concepts page agrees. No documentation covers hooks in VS Code agent mode, and the field report came from a VS Code Copilot Chat session, which ships its own Copilot CLI binary under the extension's `globalStorage`. Hook machinery evidently fires there in a configuration GitHub does not document, so nothing about variable injection is guaranteed on that surface. The CLI-only release scope therefore covers hooks as well as plugin install.

**CLI hook config locations**, unchanged and recorded because the re-read enumerated them: policy files under `/etc/github-copilot/policy.d/*.json`; repo `.github/hooks/*.json`; user `~/.copilot/hooks/` or `$COPILOT_HOME/hooks/`; inline hooks in `.github/copilot/settings.json`, `.github/copilot/settings.local.json`, or `~/.copilot/settings.json`; and a plugin-provided `hooks.json`. The cloud agent reads only `.github/hooks/*.json` from the cloned repository.

## Recommendation

All three design decisions were made on 2026-07-05 and are recorded in `copilot-adapter-design.adr`.

- **Install through the subdirectory spec `archcore-ai/plugin:plugins/archcore`**, with a marketplace listing later, consistent with `subdirectory-plugin-layout.adr`.
- **Scope the release to Copilot CLI.** VS Code has no self-serve plugin install and no documented hook support, and the cloud agent has no plugin hooks in the sandbox and MCP tools only; both are documented as limitations rather than supported surfaces. A repo-level `.github/hooks/*.json` template for cloud-agent users is a separable future item.
- **Wire hooks as native adapter files**: a dedicated `.plugin/plugin.json` manifest plus `hooks/copilot.hooks.json` in the documented camelCase format, aligned with the Cursor and Codex precedent. Claude-compat reuse was rejected, because it depends on partially documented behavior that only live probes could pin.
- **Give the `bin/` guard scripts a Copilot output branch** emitting stdout `permissionDecision` JSON keyed on `ARCHCORE_HOST=copilot`, per `host-adapter-contract.spec`. `normalize-stdin.sh` already detects the host.

## Next Action

Implement per `copilot-adapter-design.adr` and `host-adapter-contract.spec`: the manifest, the hooks config, and the MCP wiring; the Copilot deny-output branch in the guard scripts with bats fixtures; and the `copilot` row in the host-coverage-matrix structure test. Release verification is an install smoke test on Copilot CLI covering the subdirectory install, the three hook events routing to `bin/`, MCP loading, and a deny blocking a `.archcore/` write; one agent-naming check for a plain `NAME.md` from a plugin `agents/` directory, with `*.agent.md` copies as the fallback; and the probe protocol.

**Two items only a live CLI session can close.** Which plugin-root variable Copilot actually injects, and whether it injects one at all outside `copilot plugin install`, is unresolved from documentation — the candidate chain is a hedge rather than an answer. The second, added 2026-08-03: whether Copilot also auto-loads the Claude-format `hooks/hooks.json` alongside the manifest-declared Copilot config. Conventional discovery is known to run independently of the manifest for MCP, so the same may hold for hooks, and if it does then every guard fires twice with an unset variable. `host-probe-protocol.spec` records both as deferred.

## Update 2026-08-03 — plugin MCP discovery settled, in two passes

**First pass — the manifest key was never the only source.** `copilot mcp list` on Copilot CLI 1.0.76 with plugin 0.6.1 installed lists `Plugin servers: archcore (local)` from any working directory, with `mcpServers` absent from `.plugin/plugin.json` and no project or user-level config present. The concepts page's discovery rule is real, and `@test/integration/copilot-plugin-smoke.bats` reproduces it in an isolated `COPILOT_HOME`.

**Second pass — the damage was a name collision, and the first remedy did not work.** The plugin server and the project server that `archcore init --agent copilot` registers share the key `archcore`, and Copilot merges user, then workspace, then plugins with last-wins, so the plugin entry replaced the user's. Measured with a sentinel command in the workspace config and a fresh `COPILOT_HOME` per arm:

| plugin layout | surviving `archcore` |
| --- | --- |
| no plugin installed (control) | workspace |
| `.mcp.json` at plugin root | plugin (`cwd=${PLUGIN_ROOT}`) |
| renamed, no manifest key | workspace |
| renamed + key in `.claude-plugin/plugin.json` | **plugin** |
| `.mcp.json` kept + empty key in `.plugin/plugin.json` | plugin |
| renamed + both keys (shipped in 0.6.2) | workspace |

Row four is the finding that cost a release: renaming the file and pointing Claude Code at the new name through `.claude-plugin/plugin.json` re-armed Copilot, through the manifest fallback chain this document recorded in July. The shipped configuration therefore needs all three parts — a filename off the discovery list, the Claude manifest key, and an empty `mcpServers` in `.plugin/plugin.json` to stop the fallback.

**Discovery is per directory and takes two filenames.** Probing a git repository with configs planted at four levels shows Copilot reading a config from every directory between the working directory and the git root, preferring `.mcp.json` and falling back to `.github/mcp.json` only where the first is absent in that same directory. The wiring advisory in `bin/session-start` mirrors this exactly.

**Method note.** Two measurement traps cost time and are worth carrying forward. `copilot --plugin-dir` mounts a plugin but does not load its MCP servers, so a smoke test built on that flag passes whether or not the bug exists; the real install layout plus a registered entry in `config.json` is required. And a fresh `COPILOT_HOME` is mandatory per arm, because an already-installed copy in the default home silently shadows every variant under test.

Countermeasures and full consequences live in `copilot-mcp-architecture.adr`. The same rename also removed a latent plugin-shipped MCP on Cursor, whose loader accepts both `.mcp.json` and `mcp.json`.
