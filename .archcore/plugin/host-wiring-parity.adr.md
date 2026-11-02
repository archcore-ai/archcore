---
title: "Host-Wiring Parity — /archcore:init Writes the Same Host Configs as archcore init"
status: accepted
tags:
  - "architecture"
  - "component:plugin"
  - "multi-host"
  - "onboarding"
  - "plugin"
---

## Context

`/archcore:init` in the plugin initialized only `.archcore/`, while `archcore init` in the CLI also wrote per-host wiring: the MCP config, the SessionStart hook, and the usage nudge. A repository initialized from the plugin was not self-contained. A teammate cloning it without the plugin got no MCP registration, no hook, and no nudge.

The gap was widest on Cursor, where the plugin cannot ship an MCP server (`cursor-mcp-architecture.adr`), and on Codex, where plugin hooks sit behind a per-hook trust review that makes `AGENTS.md` the only reliable nudge channel. GitHub Copilot CLI made wiring mandatory, because `copilot-mcp-architecture.adr` removes plugin-shipped MCP there, so `archcore init --agent copilot` is the only route to MCP tools on that host.

## Verified facts that shaped the design

- Claude Code dedupes a plugin-shipped and a project `.mcp.json` archcore server by endpoint, giving one connection and one toolset. With a project `.mcp.json` the tools are named `mcp__archcore__*`, the naming the PostToolUse matchers were written against. Parity therefore repairs validation hooks that never fired in plugin-only Claude sessions, because matchers are exact-match and plugin naming is `mcp__plugin_archcore_archcore__*`.
- Codex reads a project `.codex/config.toml` only in trusted projects. Its `archcore` entry shadows the plugin-provided server with identical behavior (verified live on 0.144.5).
- Cursor guarantees the working directory neither for agent shell commands nor for stdio MCP spawns. Any wiring path must carry the project root explicitly.
- The CLI hook installers matched exact command strings in their idempotency probes, so a changed hook command appended a duplicate instead of updating one. This was fixed as a precondition.

## Decision

One Go core with two thin frontends and a deterministic cascade in the skill, so that `/archcore:init` writes the same host configs as `archcore init`, always behind an explicit user confirmation.

**Core.** The existing installers in the CLI (`installAgents`, the instructions writers, the hook installers) with two hardening fixes. Marker-based hook probes match the `archcore hooks` substring, update a stale entry in place, and heal past duplicates. A plugin-cache guard in `resolveProjectRoot` rejects roots containing `.cursor/plugins/`, `.claude/plugins/`, `.codex/plugins/`, `plugins/cache/`, or `~/.copilot/installed-plugins/`. A plugin developer repository with manifests at its root stays valid.

**Instruction-nudge targets.** For claude-code, one managed block (`<!-- archcore:start -->` to `<!-- archcore:end -->`) is upserted into both `CLAUDE.md` and `AGENTS.md`. `CLAUDE.md` is what Claude Code reads natively, and `AGENTS.md` is the shared block the other hosts converge on. For cursor, codex-cli, and copilot the target is the `AGENTS.md` managed block. The `install_host_config` report names the primary file in `instructions_path` and extra files in `instructions_extra_paths`. The skill's preview names both claude-code files itself.

**Path B, primary.** The MCP tool `install_host_config(host, all_detected)` runs in the MCP server process, whose base directory is correct by construction, and returns a JSON report.

**Path A, fallback.** `archcore init --agent <id>... --project <root>` runs non-interactively, validates ids before any write, and writes under the resolved root regardless of the process working directory. It serves Cursor on day one when no archcore MCP is connected yet, Copilot always, plus CI and scripts.

**Path C, last resort.** A ready-to-run terminal command for the user, `archcore update && archcore init --agent <host> --project "<root>"`, when the CLI is too old for `--agent` at execution time.

**Skill.** `/archcore:init` detects the host through `bin/detect-host`, which reads the environment only. `CLAUDECODE` or `CLAUDE_SKILL_DIR` selects claude-code, `CURSOR_TRACE_ID` selects cursor, and `CODEX_THREAD_ID` (or `CODEX_HOME` as a second condition) selects codex-cli. Otherwise the result is `__UNKNOWN__` and the skill asks. Precedence is claude, then cursor, then codex, because sessions inherit stray companion variables (`codex-adapter-conformance.adr` records why Codex is keyed on `CODEX_THREAD_ID`). Copilot has no detection branch, because every documented `COPILOT_*` variable is read from the user, not set by the host. A Copilot session reaches wiring through the ask path, which offers all four hosts. The skill shows the wiring files and the resolved project root in the preview and runs the cascade only after `confirm`. Scope is the current host by default, with `edit → hosts: all` for a team repository. `--refresh` retrofits wiring onto repositories seeded earlier.

**Deterministic version gate.** The skill's pre-flight compares the installed CLI against the pinned minimum through `bin/cli-gte`. It does a numeric field-by-field semver compare, prints `yes`, `no`, or `__NO_CLI__`, and always exits 0. The comparison never happens in prose, because an LLM comparing `0.10.0` against `0.6.0` lexically gets it wrong. The pinned minimum is v0.7.0, the first CLI with the hook leaves the plugin launchers delegate to (`cli-owns-layers-4-5.adr`). It rises whenever the wiring contract changes, so the preview and the writes stay in lockstep. `@plugin/test/unit/cli-gte.bats` holds the contract.

**Dual naming wherever a tool list gates behavior.** PostToolUse matchers, agent allow-lists, and the Codex auditor's `disabled_tools` deny-list each carry every naming the host can produce. A deny-list fails open under naming drift, because an unlisted twin would let the read-only auditor mutate. `bin/lib/normalize-stdin.sh` also folds all three namings to the canonical one before any guard inspects a tool name. `@plugin/test/structure/hooks.bats` and `@plugin/test/structure/agents.bats` guard the twin pairing.

**Cursor MCP config.** A dedicated writer emits `args: ["mcp", "--project", "${workspaceFolder}"]`, because project-level interpolation is documented. The shared writer keeps plain `["mcp"]` for Claude and Copilot, which read a workspace-root `.mcp.json` from the project process.

**SessionStart dedup in the binary.** The plugin hook script and a repo-committed hook both delegate to `archcore hooks <host> session-start`. The handler stamps `session_id` plus `source` (`conversation_id` on Cursor) into XDG state with a short window, so a double-registered hook emits context once. It fails open on a missing id or unwritable state.

**Freshness loop.** `archcore update --check` is 24-hour cached, bounded to roughly 500 ms, always exits 0, and stays silent on failure. A rate-limited session-start advisory and a consent-based `archcore update` offer in the pre-flight complete it.

**`doctor --fix` convergence.** The `--agent` and `--project` flags re-run the update-capable installers and converge a drifted MCP entry through a semantic-diff rewrite that never touches a foreign server.

Rollout order is part of the decision: CLI first, so dedup protects old plugins immediately; the plugin second, behind the version-gated skill; advisories last.

## Alternatives Considered

1. **Call plain `archcore init` from the skill** — rejected because there is no TTY. The picker is skipped, auto-detect is empty in a fresh repository, and the output degenerates to `init_project`.
2. **Extend `init_project`** — rejected because its documented boundary, that it does not install agent hooks, is load-bearing. A separate consent-framed tool is clearer.
3. **Have the agent write the configs through the Write tool** — rejected because it reimplements the CLI's merge logic in prose, which is the drift class this repository fights.
4. **Make the SessionStart hook the writer** — rejected because it writes user files without consent on every session start. Only the advisory nudge survives from that idea.
5. **Auto-update from hooks** — rejected because it re-imports the bundled-launcher bug classes: offline failure, latency, enterprise friction, version pinning, and Windows file locks. Check, advisory, and consent replaced it.

## Consequences

- Repositories become self-contained for CLI-only teammates from either init surface, and Cursor loses its manual day-one MCP step.
- Claude validation hooks start firing, because the matchers cover both tool namings.
- Wiring is idempotent and self-healing: duplicate hooks from the old probe bug are cleaned on the next install or converge.
- Tradeoff: a teammate without the CLI sees a failing hook and a failing MCP spawn after pulling a wired repository. Claude gates a project `.mcp.json` behind approval, Codex fails silently, and the hook command stays bare because a POSIX `command -v` wrapper is not portable to native-Windows teammates. Wiring is always behind an explicit confirm.
- Tradeoff: on Copilot, wiring is a dependency, not an improvement. Without it the plugin has no MCP tools, and the version gate makes that visible instead of silent.
- Tradeoff: the plugin's `bin/session-start` guard rejects a plugin developer repository that `resolveProjectRoot` accepts, because its bounded upward walk silences the hook under a manifest-carrying ancestor. A false silent exit costs one session's context. A false emit would surface the plugin's bundled files as the user's knowledge base.
- Tradeoff: a user project whose absolute path contains a literal `plugins/cache/` segment is silenced by the guard. Accepted, because the fragment set mirrors the CLI guard.
- Tradeoff: the update advisory does not reach users on a CLI older than v0.6.0, the transition's whole audience. The pre-flight gate covers them. Do not close the silence with version-sniffing in the hook, which re-imports the prose-version-logic drift class.
- Tradeoff: Codex Desktop ignores project-level MCP config (upstream issue #13025). The CLI is unaffected.

## Superseded when

- Every supported host documents and honors a plugin-shipped MCP server launched in the user's project, which removes the reason project-level wiring exists.
- The pinned CLI minimum reaches a version where `--agent` and `--project` are universally present, which lets paths A and C collapse into path B.
