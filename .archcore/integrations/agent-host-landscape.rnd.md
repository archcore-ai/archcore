---
title: "Agent Host Landscape — Next Hosts, Wrapper Editors, and Distribution Channels for Archcore"
status: draft
tags:
  - "integrations"
  - "multi-host"
  - "plugin"
  - "release"
  - "roadmap"
---

## Goal

Decide where Archcore ships next as a host plugin or adapter after the four shipped hosts, how the OpenCode adapter stands today, and how a wrapper editor that runs other agents (bb, Conductor, Superset) receives Archcore without a second copy of what the inner agent's plugin already delivers. Reader: the maintainer who orders host work and release channels.

## Questions

1. Which state has the OpenCode adapter reached, and what remains open after its release?
2. Which agent hosts with their own agent loop can carry a full Archcore adapter, and at what cost?
3. Which wrapper editors run Claude Code, Codex, or other agents, and does the inner agent's plugin still load under them?
4. Which cross-host formats and distribution channels let one artifact reach several hosts?
5. In which order does host work proceed?

## Approach

- Repository read on 2026-10-18: the supported-agents registry, the OpenCode adapter records, the hook dialect table, and every local branch for a `plugin/plugins/opencode/` tree.
- Machine read on 2026-10-18: installed host versions and the user-level OpenCode config.
- bb read on 2026-10-18 from the running install (`bb guide plugins`, `bb provider list`, `bb plugin list`). The session that wrote this record ran inside bb with the Claude Code provider.
- Web sweep on 2026-10-18 over official documentation and repositories (Pi, Hermes Agent, bb, Conductor, Superset, Nimbalyst, the cross-host standards agentskills.io, agents.md, agent-plugins.org, and the MCP Registry, plus each other host's own documentation) [assumption: the sweep names the last group without full URLs]. Star counts and releases come from the GitHub API on that date.
- Every external host fact below comes from the web sweep and was not confirmed against a running host, except where a line names a local check.

## Findings

### 1. OpenCode adapter state

The CLI side and the shell side are done. `archcore hooks opencode <event>` exists with the `opencode` row in `@cli/cmd/hook_dialect.go` (commit `ea474ad`); `@cli/cmd/hook_payload.go` folds `archcore_<tool>`; `@plugin/plugins/archcore/bin/lib/normalize-stdin.sh` accepts `ARCHCORE_HOST=opencode`; synthetic payloads sit in `@plugin/test/fixtures/stdin/opencode/`, and two of them use spellings OpenCode does not send (`mcp__archcore__*`, `file_path`).

The package side shipped. `@archcore/opencode-plugin` has been on npm since v0.14.0 (2026-10-26): The bridge package in `@plugin/plugins/opencode/`, its `bun test` suites, the npm publish job, and the `HostOpenCode` row of `@cli/internal/plugin/hosts.go` shipped (@.archcore/plugin/opencode-adapter.spec.md). OpenCode 2.x support and the install into the global OpenCode config followed in v0.14.2. A live load check on OpenCode 2.1.4 on 2026-10-26 showed the package loaded with both agents, four `/archcore-*` commands, and four skills registered. The live probes P0, A, B, and C have not run and stay open.

One machine preparation step remains open (local check, 2026-10-18). Bun 1.4.3 is installed, so the `bun test` phase runs locally.

- `~/.config/opencode/opencode.json` already holds `mcp.archcore` pointing at `npx mcp-remote http://localhost:8080/mcp/message`. The bridge yields to an existing entry per the draft entry-ownership decision, so probe C on this machine reports a dead MCP server unless that entry is removed first.

Kilo Code's CLI and v7 extension are OpenCode forks whose plugin behavior the Kilo documentation calls identical to OpenCode, with types from `@kilocode/plugin`. Kilo CLI 7.8.14 is a Bun-compiled binary with the same plugin loader, and it compares `engines.opencode` against its own 7.x version (`packages/opencode/src/plugin/shared.ts` in Kilo-Org/kilocode, read 2026-10-18), so a `^1.x` range rejects the package and a `>=1.18.34` range admits it. [assumption] `@archcore/opencode-plugin` loads under Kilo with no code change; a live check is still open.

### 2. Hosts with their own agent loop

Adapter cost by host, from the sweep:

- Pi 1.3.0 (pre-tool event `tool_call`), Amp (`tool.call`), and Kilo Code (the OpenCode plugin API) take a TypeScript bridge of the OpenCode shape; Kilo reuses the OpenCode package.
- Hermes Agent 0.22.2 (`pre_tool_call`) takes skills and MCP cheaply; hooks need the shell route [assumption] or Python.
- Antigravity, Gemini CLI 0.64.0, Factory Droid, Auggie, and Qwen Code take manifests or only verification, because they read the Claude Code layout or a similar manifest format.
- Kiro accepts Agent Plugins 1.0 but has no documented submission path. Trae and Crush have no plugin system: documentation only.

Gemini CLI now serves paid tiers only; Antigravity replaced it for unpaid and Google One users on 2026-06-18. Roo Code shut down on 2026-05-15 with its repository archived, and Continue is frozen at 2.0.0 [assumption: secondary sources only]. The supported-agents registry still lists Roo Code as an MCP-only host.

### 3. Wrapper editors

| Wrapper | Inner agents | Own plugin system | Inner agent's plugins load |
|---|---|---|---|
| bb desktop 0.46.1 | Claude Code, Codex, Pi, ACP (Cursor, opencode) | TypeScript SDK; `bb-community` marketplace | yes — local check: this session loaded the Archcore Claude Code plugin (its MCP tools and SessionStart hook) beside bb's own skills and `bb-bridge` MCP server |
| Conductor 0.91.0 (Melty Labs) | bundled Claude Code and Codex; Cursor; OpenCode | none | yes, per its FAQ and the 0.25.0 changelog |
| Superset 1.38.0 | real Claude Code, Codex, others | none; installs its own skills into `~/.claude` | yes |
| Warp | Oz, Claude Code, Codex, Gemini CLI, OpenCode | none; ships its own Claude Code plugin | yes |
| Zed, JetBrains, Devin Desktop | ACP agents through `claude-agent-acp` | Zed WASM extensions | [assumption] unknown — whether `claude-agent-acp` loads `~/.claude` plugins is unverified |
| Nimbalyst, Sculptor, Claude Squad | real CLIs | UI only or none | [assumption] yes |

"Conductor" in the `archcore` global source names Google's Gemini CLI extension, a different product. Conductor by Melty Labs appears locally only as the `~/conductor/workspaces/` worktree path in the relative-globals decision.

bb specifics (local check): bb stages each enabled plugin's `skills/` into every provider session; `bb.agents.configure(context)` receives `context.provider.id` and selects the plugin's tools and skills per provider; the only bb hook is `message.dispatch`, which runs before a message is sent, so bb has no pre-tool event and cannot run the `.archcore/` write guard. The bb store already lists an official provider-independent Memory plugin in the "Memory & Context" category.

### 4. Cross-host formats and channels

| Format or channel | Reach | Cost to Archcore |
|---|---|---|
| Claude Code plugin layout (`.claude-plugin/`) | Claude Code, Copilot CLI, VS Code, Factory, Auggie, Qwen Code, OpenHands; Conductor for commands | shipped; verification on new readers |
| Agent Plugins 1.0 (root `plugin.json`) | Copilot CLI, VS Code, Cursor, OpenHands, Kiro, Hermes catalog | one more manifest |
| Agent Skills (`SKILL.md`) | about 45 clients in the agentskills.io showcase, shared path `.agents/skills` | shipped content |
| `AGENTS.md` | 23 tools on agents.md plus Pi, Hermes, Crush, Trae, Antigravity, Kiro | the managed block already reaches them |
| MCP Registry | every MCP host | `mcp-publisher` and namespace proof |
| Host channels | npm (OpenCode, Pi), Hermes catalog, Gemini gallery, `agy`, `bb-community`, Kilo marketplace, skills.sh | one listing per channel |

ACP Registry lists agents, not plugins, and does not apply to Archcore.

## Recommendation

Proceed, in this order:

1. Run the OpenCode live probes, then verify the same package under Kilo Code.
2. Take the near-zero-cost reach: publish the MCP server to the MCP Registry, add an Agent Plugins 1.0 manifest, verify the Claude Code layout on Factory, Auggie, and Qwen Code, and verify the Roo Code and Continue shutdowns before re-tiering them.
3. Build a Pi package on the OpenCode bridge shape (Finding 2). Extract a shared spawn module only when Pi is the second TypeScript host.
4. Add Antigravity and Gemini extension manifests (Finding 2).
5. Ship Hermes skills and MCP first; add hooks only through Hermes shell hooks calling `archcore hooks hermes <event>`, or through a new language decision.
6. Ship a thin bb plugin under the wrapper-host decision recorded beside this investigation.
7. Give Conductor, Superset, and Warp a verification run and an integration recipe page; no plugin.
8. Probe `claude-agent-acp` for plugin loading before any Zed or JetBrains work.

The order follows three supporting findings: a host with a blocking pre-tool event supports the full guard (Finding 2), every checked wrapper already loads the inner agent's plugin (Finding 3), and shared formats reach several hosts per artifact (Finding 4).

Resulting host list: 4 shipped (Claude Code, Codex CLI, GitHub Copilot, Cursor); 2 CLI-wired without a plugin (Gemini CLI, Cline); 1 shipped without live probes (OpenCode); 10 adapter candidates (Kilo Code, Factory Droid, Auggie, Qwen Code, Antigravity, Gemini CLI with Goose, Kiro, Pi, Hermes, Amp); 8 wrapper entries needing verification or a thin plugin (bb, Conductor, Superset, Warp, Zed, JetBrains, Devin Desktop, Nimbalyst with Sculptor and Claude Squad); 2 documentation-only (Trae, Crush); retire or skip Roo Code, Continue, Vibe Kanban, Terragon, Omnara; Mosaic and Cube are not hosts.

## Next Action

Run the OpenCode live probes; before probe C, remove the user-level `mcp.archcore` entry on the probe machine. After the Roo Code and Continue checks, update the host roster in the `archcore` global source and `supported-ai-agents.doc` in one change.
