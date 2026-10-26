---
title: "Installing the Archcore OpenCode Plugin and Verifying a Session"
status: draft
tags:
  - "component:plugin"
  - "multi-host"
  - "opencode"
  - "plugin"
---

Reader: a developer wiring an OpenCode project that carries `.archcore/`; task: install `@archcore/opencode-plugin` and confirm that the hooks, commands, and agents work; step actor: that developer.

## Prerequisites

- Archcore CLI 0.10.0 or newer on the `PATH` of the shell that starts OpenCode (`archcore --version`).
- OpenCode 1.18.34 or newer, 1.x or 2.x (`opencode --version`).
- A project with `.archcore/`.
- No `mcp.archcore` entry in `~/.config/opencode/opencode.json` that points at another server; the bridge keeps an existing entry.

## Steps

1. If `archcore init` offers OpenCode as a plugin host, run `archcore init --agent opencode` and skip to step 3.
2. Run the command of your OpenCode line to add the plugin to the global config:

   ```sh
   opencode plugin add @archcore/opencode-plugin   # OpenCode 2.x
   opencode plugin -g @archcore/opencode-plugin    # OpenCode 1.x
   ```

   Expected result: `~/.config/opencode/opencode.json` lists the package, under `plugins` on 2.x and under `plugin` on 1.x.

3. Restart OpenCode in the project directory.

   [assumption] The config is not hot-reloaded, so a restart is required (opencode.ai/docs/plugins, read 2026-10-09).

4. Start a session and send any prompt.

   Expected result: the reply reflects the `[Archcore]` recap — document counts and the context address.

5. Ask the agent to write a file at `.archcore/probe.rule.md` with its file-writing tool.

   Expected result: the tool call fails with the redirect-to-MCP reason; the file does not exist.

6. Ask the agent to call `list_documents`.

   Expected result: the agent lists document paths from `.archcore/`.

7. Type `/archcore-` in the prompt.

   Expected result: the command list shows `/archcore-init`, `/archcore-plan`, `/archcore-document`, and `/archcore-review`.

8. Type `@archcore-` in the prompt.

   Expected result: the subagent list shows `archcore-assistant` and `archcore-auditor`.

## Verification

- On 2.x, `opencode plugin list` in the project shows `@archcore/opencode-plugin`. On 1.x, `ls ~/.cache/opencode/node_modules/@archcore/opencode-plugin` lists `dist`, `bin`, `skills`, and `agents`.
- Steps 4–6 are probes P0, B, and C of `host-probe-protocol.spec`; when running a dated probe, append the row there.

## Common Issues

- No recap and no refusal: `archcore` is absent from the `PATH` of the OpenCode process, or older than 0.10.0 — the bridge then spawns nothing. Run `archcore --version` from the shell that starts OpenCode.
- Document tools fail to start: an existing `mcp.archcore` entry in a project or global `opencode.json` points at another command, and the bridge keeps it. Remove that entry, or run `archcore init --agent opencode` to write the standard one, then restart.
- Recap arrives but document tools are missing: the `config` hook ran after MCP start [assumption]. Run `archcore init --agent opencode` to write the entry into `opencode.json`, then restart.
- `/init` or `/review` runs OpenCode's own command: that is expected; the Archcore commands are `/archcore-init` and `/archcore-review`.
- The plugin is skipped with a version error: the running OpenCode is older than the `engines.opencode` range of the package. Upgrade OpenCode.
- `opencode plugin` prints subcommands (`add`, `list`, `remove`) and rejects the package argument: this is OpenCode 2.x. Use `opencode plugin add` in step 2.
- The package is in both the global and a project `opencode.json`: OpenCode loads it once, and the project entry wins (`deduplicatePluginOrigins`, `packages/opencode/src/config/plugin.ts`, 1.18.34). No action is needed.
- To remove the plugin on 2.x, run `opencode plugin remove @archcore/opencode-plugin`. On 1.x, delete the entry from the `plugin` array.
- A stale package after a release: delete `~/.cache/opencode/node_modules/@archcore/opencode-plugin` and restart; pin with `@archcore/opencode-plugin@<version>` to hold a version.
