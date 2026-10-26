# @archcore/opencode-plugin

Archcore for [OpenCode](https://opencode.ai). The plugin gives OpenCode the Archcore MCP server, the `/archcore-*` commands, two subagents, and the hooks that guard `.archcore/` and check document writes.

You install the plugin once, in the global OpenCode config. It then works in every project that has `.archcore/`.

## Requirements

- OpenCode 1.18.34 or later, 1.x or 2.x (`opencode --version`).
- Archcore CLI 0.10.0 or later on the `PATH` of the shell that starts OpenCode (`archcore --version`). To install the CLI, see [Install Archcore](https://docs.archcore.ai/start/install/).

## Install

Run the command for your OpenCode version:

```sh
opencode plugin add @archcore/opencode-plugin   # OpenCode 2.x
opencode plugin -g @archcore/opencode-plugin    # OpenCode 1.x
```

Or, in a project root, run `archcore init --agent opencode`. That command creates `.archcore/` if it is missing, writes the MCP entry to the project `opencode.json`, and adds the plugin to the global config.

All three ways add the package to `~/.config/opencode/opencode.json`. OpenCode 2.x writes it under the `plugins` key, and OpenCode 1.x under `plugin`. OpenCode 2.x reads both keys.

Restart OpenCode after the install.

## Check the install

1. Open OpenCode in a project that has `.archcore/`.
2. Type `/archcore-`.

   Expected result: the list shows `/archcore-init`, `/archcore-plan`, `/archcore-document`, and `/archcore-review`.

3. Ask the agent to list the Archcore documents.

   Expected result: the agent calls `list_documents`. An empty list is valid for a new project.

The commands start with `archcore-` because OpenCode has its own `/init` and `/review`. The subagents are `@archcore-assistant` and `@archcore-auditor`.

## MCP server

The plugin adds the MCP server `archcore` with the command `archcore mcp`. If your OpenCode config already has an `mcp.archcore` entry, the plugin keeps that entry and adds nothing.

If an old `mcp.archcore` entry points at another command, document tools go to that command. Remove the old entry from `~/.config/opencode/opencode.json`, or run `archcore init --agent opencode` in the project to write the standard entry. Then restart OpenCode.

## Update and remove

| Task | OpenCode 2.x | OpenCode 1.x |
|---|---|---|
| Update | `opencode plugin update @archcore/opencode-plugin` | Delete `~/.cache/opencode/node_modules/@archcore/opencode-plugin`, then restart OpenCode |
| Remove | `opencode plugin remove @archcore/opencode-plugin` | Delete the package from the `plugin` array of `~/.config/opencode/opencode.json` |

## Troubleshooting

- No session recap and no write guard: `archcore` is not on the `PATH` of the OpenCode process, or the CLI is older than 0.10.0. Run `archcore --version` in the shell that starts OpenCode.
- `/init` or `/review` runs the OpenCode command: use `/archcore-init` and `/archcore-review`.
- OpenCode skips the plugin with a version error: upgrade OpenCode to 1.18.34 or later.

## Links

- [Connect Archcore to OpenCode](https://docs.archcore.ai/guides/connect-your-agent/#opencode)
- [Archcore documentation](https://docs.archcore.ai)
- [Source](https://github.com/archcore-ai/archcore/tree/dev/plugin/plugins/opencode)
