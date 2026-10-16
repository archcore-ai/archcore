---
title: "Installing the Archcore OpenCode Plugin and Verifying a Session"
status: draft
tags:
  - "component:plugin"
  - "multi-host"
  - "opencode"
  - "plugin"
---

Reader: a developer wiring an OpenCode project that carries `.archcore/`; task: install `@archcore/opencode-plugin` and confirm the three hook events fire; step actor: that developer.

## Prerequisites

- Archcore CLI 0.10.0 or newer on `PATH` of the shell that starts OpenCode (`archcore --version`).
- OpenCode 1.17.13 or newer (`opencode --version`).
- A project with `.archcore/`; `opencode.json` at its root may not exist yet.
- Until the CLI delivery of `opencode-adapter.plan` ships, a text editor for `opencode.json`.

## Steps

1. If `archcore init` already offers OpenCode as a plugin host, run `archcore init --agent opencode` and skip to step 4.
2. Open `opencode.json` at the project root; create it with `{}` when absent.
3. Add the package to the plugin array:

   ```json
   { "plugin": ["@archcore/opencode-plugin"] }
   ```

   Expected result: `jq .plugin opencode.json` prints a one-element array.

4. Restart OpenCode in the project directory; the host installs the package at startup.

   [assumption] The config is not hot-reloaded, so a restart is required (opencode.ai/docs/plugins, read 2026-10-09).

5. Start a session and send any prompt.

   Expected result: the reply reflects the `[Archcore]` recap — document counts and the context address.

6. Ask the agent to write a file at `.archcore/probe.rule.md` with the `write` tool.

   Expected result: the tool call fails with the redirect-to-MCP reason; the file does not exist.

7. Ask the agent to call `list_documents`.

   Expected result: the agent lists document paths from `.archcore/`.

8. Ask the agent which Archcore skills it can run.

   Expected result: `init`, `plan`, `document`, and `review` are named.

## Verification

- `ls ~/.cache/opencode/node_modules/@archcore/opencode-plugin/bin` lists `session-start`, `pre-tool-use`, `post-tool-use`, `cli-gte`, and `lib`.
- Steps 5–7 are probes P0, B, and C of `host-probe-protocol.spec`; when running a dated probe, append the row there.

## Common Issues

- No recap and no refusal: `archcore` is absent from the `PATH` of the OpenCode process, or older than 0.7.0 — the launchers exit silently. Run `archcore --version` from the shell that starts OpenCode.
- Recap arrives but document tools are missing: the `config` hook ran after MCP start [assumption]. Run `archcore init --agent opencode` to write the entry into `opencode.json`, then restart.
- Duplicate skill warning at startup: the project also carries copies of the Archcore skills under `.claude/skills/` or `.agents/skills/`; remove one copy — the later load wins nondeterministically (`opencode-host-support.rnd`).
- A stale package after a release: delete `~/.cache/opencode/node_modules/@archcore/opencode-plugin` and restart; pin with `@archcore/opencode-plugin@<version>` to hold a version.
