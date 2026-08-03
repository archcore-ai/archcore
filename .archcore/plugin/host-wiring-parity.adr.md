---
title: "Host-Wiring Parity — /archcore:init Writes the Same Host Configs as archcore init"
status: accepted
tags:
  - "architecture"
  - "multi-host"
  - "onboarding"
  - "plugin"
---

## Context

`/archcore:init` in the plugin initialized only `.archcore/`, while `archcore init` in the CLI additionally wrote per-host wiring — the MCP config, the SessionStart hook, and the usage nudge. A repository initialized from the plugin was therefore not self-contained: a teammate cloning it without the plugin got no MCP registration, no hook, and no nudge. The gap was widest on Cursor, where the plugin cannot ship an MCP server at all per `cursor-mcp-architecture.adr`, leaving day-one users a manual `~/.cursor/mcp.json` step, and on Codex, where plugin hooks sit behind a per-hook trust review that makes `AGENTS.md` the only reliable nudge channel. GitHub Copilot CLI joined later and made wiring mandatory rather than merely useful, because `copilot-mcp-architecture.adr` removes plugin-shipped MCP there too, leaving `archcore init --agent copilot` as the only route to MCP tools on that host.

## Verified facts that shaped the design

- Claude Code dedupes a plugin-shipped and a project `.mcp.json` archcore server by endpoint, yielding one connection and one toolset, verified live. With a project `.mcp.json` the tools are named `mcp__archcore__*`, which is the naming the plugin's PostToolUse validation matchers were written against — so parity *repairs* validation hooks that never fired in plugin-only Claude sessions, since matchers are exact-match and plugin naming is `mcp__plugin_archcore_archcore__*`.
- Codex reads a project `.codex/config.toml` only in trusted projects, and its `archcore` entry shadows the plugin-provided server with identical behavior, verified live on 0.144.5.
- Cursor guarantees the working directory neither for agent shell commands nor for stdio MCP spawns — the docs are silent, the behavior is community-confirmed, and a historical dev-docs leak agrees. Any wiring path must therefore carry the project root explicitly.
- The CLI hook installers' idempotency probes matched exact command strings, so a changed hook command appended a duplicate entry instead of updating one. This was fixed as a precondition of the work.

## Decision

Adopt one Go core with two thin frontends and a deterministic cascade in the skill, so that `/archcore:init` writes the same host configs as `archcore init`, always behind an explicit user confirmation.

**Core.** The existing installers in `cli` — `installAgents`, the instructions writers, and the hook installers — with two hardening fixes: marker-based hook probes matching the `archcore hooks` substring, which update a stale archcore entry in place and heal past duplicates; and a plugin-cache guard in `resolveProjectRoot` that rejects roots containing `.cursor/plugins/`, `.claude/plugins/`, `.codex/plugins/`, or `plugins/cache/`, while a plugin *developer* repository with manifests at its root stays valid. The fragment list gained `~/.copilot/installed-plugins/` in CLI v0.6.7, after a live probe on 2026-08-03 settled that Copilot auto-discovers a plugin-root `.mcp.json` regardless of the manifest, so that host no longer rests on the manifest omission alone.

**Instruction-nudge targets, CLI v0.6.1 and later.** For claude-code, one fenced managed block — `<!-- archcore:start -->` to `<!-- archcore:end -->` — is upserted into both `CLAUDE.md` and `AGENTS.md`, because `CLAUDE.md` is what Claude Code natively reads while `AGENTS.md` is the shared standard block the other hosts converge on; one write refreshes both, and the CLI deletes the legacy owned `.claude/rules/archcore.md` whenever claude-code is rewired. For cursor, codex-cli, and copilot, the target is the `AGENTS.md` managed block. The `install_host_config` JSON report names the primary file in `instructions_path` and any additional files in the additive `instructions_extra_paths` field, from CLI v0.6.2; the skill's preview and closing message name both claude-code files themselves, independent of the report shape.

**Path B, primary.** The MCP tool `install_host_config(host, all_detected)` executes in the MCP server process, whose baseDir is correct by construction, and returns a JSON report. It is injected from `cmd` into the server to avoid an import cycle, and the server shields the JSON-RPC stream by pointing `os.Stdout` at stderr for tool-executor prints.

**Path A, fallback.** `archcore init --agent <id>... --project <root>` runs non-interactively, with no picker and no confirms, validates ids before any write, and writes under the resolved root regardless of process working directory. It serves Cursor on day one, when no archcore MCP is connected yet, Copilot always, plus CI and scripts. `--project` is also threaded into `hooks install` and `instructions install`.

**Path C, last resort.** A ready-to-run terminal command for the user — `archcore update && archcore init --agent <host> --project "<root>"` — when the CLI turns out too old for `--agent` at execution time.

**Skill.** `/archcore:init` detects the host through `bin/detect-host`, which reads the environment only: `CLAUDECODE` or `CLAUDE_SKILL_DIR` for claude-code, `CURSOR_TRACE_ID` for cursor, `CODEX_HOME` for codex-cli, otherwise `__UNKNOWN__` and an ask, with precedence claude over cursor over codex because sessions inherit stray companion variables. Copilot has no detection branch by design, because every documented `COPILOT_*` variable is read *from* the user rather than set by the host, so a Copilot session lands on `__UNKNOWN__` and reaches wiring through the ask path — which is why that path offers all four hosts. The skill shows the wiring files and the resolved project root in the preview and executes the cascade only after `confirm`. Scope is the current host by default, with `edit → hosts: all` for a team repository. The empty route, where no source exists yet, offers wiring behind its own mini-confirm, because an empty repository is exactly where a CLI-only teammate needs the configs. `--refresh` retrofits wiring onto repositories seeded before this decision. A user who declines the version-gate update never enters the cascade, and reaches the same manual command through the closing-message note.

**Deterministic version gate.** The skill's pre-flight compares the installed CLI against the pinned minimum through `bin/cli-gte`, which does a numeric field-by-field semver compare, prints `yes`, `no`, or `__NO_CLI__`, and always exits 0 — never in prose, because an LLM comparing `0.10.0` against `0.6.0` lexically gets it wrong. The gate moved from v0.6.0 to v0.6.1 when the claude-code nudge became the `CLAUDE.md` plus `AGENTS.md` managed blocks, and to v0.6.4 when the Copilot writer stopped targeting `.vscode/mcp.json`, a surface Copilot CLI dropped in v1.0.37, in favor of the workspace-root `.mcp.json` it actually reads. Each bump keeps the preview and the writes in lockstep. The contract lives in `@test/unit/cli-gte.bats`.

**Dual naming wherever a tool list gates behavior.** PostToolUse matchers in the Claude and Codex configs — Cursor has no postToolUse event, and Copilot's carries no matcher because the scripts self-filter — the agent allow-lists in `agents/*.md`, which also carry Copilot's flat `archcore-<tool>` form, and the Codex auditor's `disabled_tools` deny-list each carry every naming the host can produce. A deny-list fails open under naming drift, because an unlisted twin would let the read-only auditor mutate, so the twins are mandatory there even if Codex ultimately yields only one naming. `bin/lib/normalize-stdin.sh` additionally folds all three namings to the canonical one before any guard inspects a tool name. Structure tests guard the twin pairing in `@test/structure/hooks.bats` and `@test/structure/agents.bats`.

**Cursor MCP config.** A dedicated writer emits `args: ["mcp", "--project", "${workspaceFolder}"]`, since project-level interpolation is documented, aligning the installed config with `docs/cursor.mcp.example.json` and the cwd-independence invariant. The shared standard writer keeps plain `["mcp"]` for Claude and Copilot, which both read a workspace-root `.mcp.json` from the project process.

**SessionStart dedup in the binary.** Both the plugin hook script and a repo-committed hook delegate to `archcore hooks <host> session-start`, and the handler stamps `session_id` plus `source` — `conversation_id` on Cursor — into XDG state with a short window, so a double-registered hook emits context once. Living in the binary protects every plugin and CLI version combination, and it fails open on a missing id or unwritable state. The response shape is per host: Copilot's native `sessionStart` takes a bare `{"additionalContext":…}` rather than Claude's `hookSpecificOutput` wrapper, from CLI v0.6.4.

**Freshness loop.** `archcore update --check` is 24-hour cached, bounded to roughly 500 ms, always exits 0, and stays silent on failure, paired with a rate-limited session-start advisory; the skill's pre-flight gates wiring on the pinned CLI minimum with a consent-based `archcore update` offer.

**`doctor --fix` convergence.** The `--agent` and `--project` flags re-run the now update-capable installers and converge a drifted MCP entry — for example a pre-`--project` Cursor config — through a semantic-diff rewrite that never touches a foreign server.

Rollout order is part of the decision: CLI first, so dedup protects old plugins immediately; the plugin second, behind the version-gated skill; advisories last.

## Alternatives Considered

1. **Call plain `archcore init` from the skill** — rejected because there is no TTY: the picker is skipped, auto-detect is empty in a fresh repository, and the output degenerates to today's `init_project`.
2. **Extend `init_project`** — rejected because its documented boundary, that it does not install agent hooks, is load-bearing; a separate consent-framed tool is clearer.
3. **Have the agent write the configs through the Write tool** — rejected because it reimplements the CLI's merge logic in prose, which is exactly the drift class this repository fights.
4. **Make the SessionStart hook the writer** — rejected because it writes to user files without consent on every session start; only the advisory nudge survives from that idea.
5. **Auto-update from hooks** — rejected because it re-imports the bundled-launcher bug classes: offline failure, latency, enterprise friction, version pinning, and Windows file locks. Check plus advisory plus consent replaced it.

## Consequences

- Repositories become self-contained for CLI-only teammates from either init surface.
- Cursor loses its manual day-one MCP step.
- Claude validation hooks start firing, because the matchers now cover both tool namings.
- Wiring is idempotent and self-healing: duplicate hooks from the old probe bug are cleaned on the next install or converge.
- Neutral: `install_host_config` adds one tool to every session's listing. Its description restricts use to explicit init and setup flows, and hosts gate the writes behind permissions.
- Tradeoff: a teammate without the CLI sees a failing hook and a failing MCP spawn after pulling a wired repository. This is mitigated rather than eliminated — Claude gates a project `.mcp.json` behind approval and reports spawn failures to the model, Codex fails silently, and the hook command stays bare, because a POSIX `command -v` soft-fail wrapper is not portable to native-Windows teammates sharing the same repository file. Wiring is always behind an explicit confirm.
- Tradeoff: Codex Desktop ignores project-level MCP config, upstream issue #13025. The CLI is unaffected.
- Tradeoff: the combination of an old CLI, an old plugin, and a teammate pulling wiring still double-fires SessionStart until that teammate updates the CLI. The version gate narrows the window and the rollout order narrows it further.
- Tradeoff: runtime `--project` in the Cursor config requires a teammate CLI of v0.3.6 or later, the flag's introduction. Every other committed command exists since v0.0.1.
- Tradeoff: on Copilot, wiring is a dependency rather than an improvement. Without it the plugin has no MCP tools at all, so a Copilot user on a CLI older than v0.6.4 gets skills, commands, agents, and hooks with nothing to write documents with. The version gate makes that visible instead of silent.
- Tradeoff: a shell-guard asymmetry with the CLI. `resolveProjectRoot` accepts a plugin developer repository whose root carries the manifests; the plugin's `bin/session-start` guard does not, because its bounded upward walk silences the hook in any directory nested up to 12 levels under a manifest-carrying ancestor, developer repositories included. This extends pre-existing behavior and is accepted: a false silent exit costs one session's context, while a false emit surfaces the plugin's bundled files as the user's knowledge base. A legitimate project nested deeper than 12 levels below such an ancestor runs normally, and the bound is tested in both directions.
- Tradeoff: a `plugins/cache/` path-fragment false positive. A user project whose absolute path contains a literal `plugins/cache/` segment is silenced by guard layer 1. Accepted deliberately, because the fragment set mirrors the CLI guard and the collision odds are negligible against the misrouted-cwd risk it closes. Tests pin the behavior either way.
- Tradeoff: the update advisory is silent for the population it targets during this rollout, because `archcore update --check` exists only in CLI v0.6.0 and later, so users on older CLIs — the transition's whole audience — get nothing from that channel. The pre-flight gate and the config-rejection advisory cover them instead, and the advisory starts paying off one CLI generation out. Do not close the silence with version-sniffing in the hook; that re-imports the prose-version-logic drift class.

## Superseded when

- Every supported host documents and honors a plugin-shipped MCP server launched in the user's project, which would remove the reason project-level wiring exists.
- The pinned CLI minimum reaches a version where `--agent` and `--project` are universally present in the installed base, which would let paths A and C collapse into path B.

## Key files

- CLI: @cmd/init.go, @cmd/host_wiring.go, @cmd/hooks_install.go, @cmd/hooks_copilot.go, @cmd/mcp_root.go, @cmd/hooks_claude_code.go, @cmd/doctor.go, @cmd/update.go, @internal/agents/mcp_helpers.go, @internal/agents/copilot.go, @internal/agents/instructions.go, @internal/wiring/hooks_agents.go, @internal/mcp/tools/install_host_config.go, @internal/mcp/server.go
- Plugin: @plugins/archcore/bin/detect-host, @plugins/archcore/bin/cli-gte, @plugins/archcore/bin/session-start, @plugins/archcore/bin/lib/normalize-stdin.sh, @plugins/archcore/hooks/hooks.json, @plugins/archcore/hooks/codex.hooks.json, @plugins/archcore/hooks/copilot.hooks.json, @plugins/archcore/agents/archcore-auditor.toml, @plugins/archcore/skills/init/SKILL.md
