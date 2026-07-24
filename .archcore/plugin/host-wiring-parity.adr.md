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

`/archcore:init` (plugin) initialized only `.archcore/`; `archcore init` (CLI) additionally wrote per-host wiring — MCP config, SessionStart hook, usage-nudge. A repo initialized from the plugin was therefore not self-contained: a teammate cloning it without the plugin got no MCP registration, no hook, no nudge. The gap was widest on Cursor (the plugin cannot ship an MCP server there at all — `cursor-mcp-architecture.adr` — so day-one users had a manual `~/.cursor/mcp.json` step) and on Codex (plugin hooks sit behind a per-hook trust review, making `AGENTS.md` the only reliable nudge channel).

Verified facts that shaped the design:

- Claude Code dedupes a plugin-shipped and a project `.mcp.json` archcore server by endpoint — one connection, one toolset (verified live). With a project `.mcp.json`, tools are named `mcp__archcore__*`, which is also the naming the plugin's PostToolUse validation matchers were written against — so parity *repairs* validation hooks that never fired in plugin-only Claude sessions (matchers are exact-match; plugin naming is `mcp__plugin_archcore_archcore__*`).
- Codex reads a project `.codex/config.toml` only in trusted projects; its `archcore` entry shadows the plugin-provided server with identical behavior (verified live on 0.144.5).
- Cursor guarantees cwd neither for agent shell commands nor for stdio MCP spawns (docs silent, community-confirmed; historical dev-docs leak). Any wiring path must carry the project root explicitly.
- The CLI hook installers' idempotency probes matched exact command strings — a changed hook command would append a duplicate entry instead of updating (fixed as a precondition).

## Decision

One Go core, two thin frontends, a deterministic cascade in the skill:

- **Core** — the existing installers in `cli` (`installAgents`, instructions writers, hook installers) with two hardening fixes: marker-based hook probes (`archcore hooks` substring) that update a stale archcore entry in place and heal past duplicates, and a plugin-cache guard in `resolveProjectRoot` (rejects roots containing `.cursor/plugins/`, `.claude/plugins/`, `.codex/plugins/`, `plugins/cache/`; a plugin *developer* repo with manifests at its root stays valid).
- **Instruction-nudge targets (CLI ≥ v0.6.1)** — claude-code: one fenced managed block (`<!-- archcore:start -->` … `<!-- archcore:end -->`) upserted into BOTH `CLAUDE.md` and `AGENTS.md` — CLAUDE.md is what Claude Code natively reads, AGENTS.md is the shared standard block the other hosts converge on; one write refreshes both, and the CLI deletes the legacy owned `.claude/rules/archcore.md` whenever claude-code is (re)wired. cursor / codex-cli: the `AGENTS.md` managed block. The `install_host_config` JSON report names the primary file in `instructions_path` and any additional files in the additive `instructions_extra_paths` field (CLI ≥ v0.6.2); the skill's preview and closing message name both claude-code files themselves, independent of the report shape.
- **B, primary** — MCP tool `install_host_config(host, all_detected)`: executes in the MCP server process whose baseDir is correct by construction, returns a JSON report. Injected from `cmd` into the server (no import cycle); the server shields the JSON-RPC stream by pointing `os.Stdout` at stderr for tool-executor prints.
- **A, fallback** — `archcore init --agent <id>... --project <root>`: non-interactive (no picker, no confirms), validates ids before any write, writes under the resolved root regardless of process cwd. Serves Cursor day-one (no archcore MCP connected yet), CI, and scripts. `--project` also threaded into `hooks install` / `instructions install`.
- **C, last resort** — a ready-to-run terminal command for the user (`archcore update && archcore init --agent <host> --project "<root>"`) when the CLI turns out too old for `--agent` at execution time.
- **Skill** — `/archcore:init` detects the host via `bin/detect-host` (env-only: `CLAUDECODE`/`CLAUDE_SKILL_DIR` → claude-code, `CURSOR_TRACE_ID` → cursor, `CODEX_HOME` → codex-cli, else `__UNKNOWN__` → ask; precedence claude > cursor > codex because sessions inherit stray companion vars), shows the wiring files AND the resolved project root in the preview, and executes the cascade only after `confirm`. Scope: current host by default, `edit → hosts: all` for team repos. The empty route (no source yet) offers wiring behind its own mini-confirm — an empty repo is exactly where a CLI-only teammate needs the configs. `--refresh` retrofits wiring on repos seeded before this ADR. A user who declines the version-gate update never enters the cascade — their channel is the closing-message note carrying the same manual command.
- **Deterministic version gate** — the skill's pre-flight compares the installed CLI against v0.6.1 via `bin/cli-gte` (numeric field-by-field semver compare, prints `yes|no|__NO_CLI__`, always exit 0), never in prose: an LLM comparing "0.10.0" against "0.6.0" lexically gets it wrong. Same right-sized-helper mold as `detect-host`; contract in `test/unit/cli-gte.bats`. The gate was bumped from v0.6.0 when CLI v0.6.1 moved the claude-code nudge to the CLAUDE.md + AGENTS.md managed blocks: a v0.6.0 CLI still writes the legacy `.claude/rules/archcore.md` layout, which the skill's preview no longer promises — the gate keeps preview and reality in lockstep.
- **Dual naming everywhere a tool list gates behavior** — PostToolUse matchers (both hooks configs), agent allow-lists (`agents/*.md` `tools:`), and the Codex auditor's deny-list (`agents/archcore-auditor.toml` `disabled_tools`) each carry both namings. Deny-lists fail OPEN under naming drift (an unlisted twin would let the read-only auditor mutate), so the twins are mandatory there even if Codex ultimately yields only one naming. Twin pairing is guarded by structure tests in `test/structure/hooks.bats` (matchers) and `test/structure/agents.bats` (agent lists).
- **Cursor MCP config** gets a dedicated writer emitting `args: ["mcp", "--project", "${workspaceFolder}"]` (project-level interpolation is documented), aligning the installed config with `docs/cursor.mcp.example.json` and the cwd-independence invariant. The shared standard writer keeps plain `["mcp"]` for Claude.
- **SessionStart dedup in the binary** — both the plugin hook script and a repo-committed hook delegate to `archcore hooks <host> session-start`; the handler now stamps `session_id`+`source` (Cursor: `conversation_id`) in XDG state with a short window, so a double-registered hook emits context once. Living in the binary, it protects every plugin/CLI version combination. Fail-open on missing id or unwritable state.
- **Freshness loop** — `archcore update --check` (24h-cached, ~500ms-bounded, always exit 0, silent on failure) + a rate-limited session-start advisory; the skill's pre-flight gates wiring on CLI ≥ v0.6.1 with a consent-based `archcore update` offer. Auto-update from hooks was rejected (offline/latency/enterprise/pinning failure modes — the bundled-launcher lesson).
- **doctor --fix convergence** — `--agent`/`--project` flags; re-runs installers (now update-capable) and converges drifted MCP entries (e.g. a pre-`--project` Cursor config) via semantic-diff rewrite that never touches foreign servers.

Rollout order is part of the decision: CLI v0.6.0 first (dedup protects old plugins immediately), plugin second (version-gated skill), advisories last.

## Alternatives

- **Plain `archcore init` from the skill** — no TTY: picker skipped, auto-detect empty in a fresh repo, output degenerates to today's `init_project`. Rejected.
- **Extending `init_project`** — its documented boundary ("does not install agent hooks") is load-bearing; a separate consent-framed tool is clearer. Rejected.
- **Agent writes configs via Write tool** — reimplements the CLI's merge logic in prose; drift class the repo explicitly fights. Rejected.
- **SessionStart hook as writer** — writes to user files without consent on every session start. Rejected; only the advisory nudge survives from this idea.
- **Auto-update in hooks** — re-imports the bundled-launcher bug classes (offline, latency, enterprise, version pinning, Windows lock). Rejected in favor of check+advisory+consent.

## Consequences

**Positive.** Repos become self-contained for CLI-only teammates from either init surface; Cursor day-one loses its manual MCP step; Claude validation hooks start firing (matchers now cover both tool namings); wiring is idempotent and self-healing (duplicate hooks from the old probe bug are cleaned on the next install/converge).

**Negative / accepted risks.**
- A teammate without the CLI sees a failing hook and a failing MCP spawn after pulling a wired repo. Mitigated, not eliminated: Claude gates project `.mcp.json` behind approval and reports spawn failures to the model; Codex fails silently; the hook command stays bare (a POSIX `command -v` soft-fail wrapper is not portable to native-Windows teammates sharing the same repo file). Wiring is always behind an explicit confirm.
- Codex Desktop ignores project-level MCP config (upstream #13025) — CLI unaffected.
- The "everything old + newly wired repo" combination (old CLI, old plugin, teammate pulls wiring) still double-fires SessionStart until that teammate updates the CLI; the version-gate narrows the window, rollout order narrows it further.
- Runtime `--project` in the Cursor config requires teammates' CLI ≥ v0.3.6 (flag introduction); all other committed commands exist since v0.0.1.
- **Shell-guard asymmetry with the CLI.** `resolveProjectRoot` (CLI) accepts a plugin *developer* repo whose root carries the manifests; the plugin's `bin/session-start` guard does not — its bounded upward walk silences the hook in any directory nested (≤ 12 levels) under a manifest-carrying ancestor, developer repos included. This extends the pre-existing behavior (manifest-at-cwd already silenced the hook) and is accepted: a false silent-exit costs one session's context, a false emit surfaces the plugin's bundled files as the user's knowledge base. Legitimate projects nested deeper than 12 levels below such an ancestor run normally (bound tested both ways).
- **`plugins/cache/` path-fragment false positive.** A user project whose absolute path contains a literal `plugins/cache/` segment is silently silenced by guard layer 1. Accepted deliberately — the fragment set mirrors the CLI guard, and the collision odds are negligible against the misrouted-cwd risk it closes. Pinned by tests either way.
- **The update advisory is silent for the population it targets during this rollout.** `archcore update --check` exists only in CLI ≥ v0.6.0, so users on older CLIs — the transition's whole audience — get nothing from this channel; the pre-flight gate and the config-rejection advisory cover them instead. Accepted: the advisory starts paying off one CLI generation out. Do NOT "fix" the silence with version-sniffing in the hook — that re-imports the prose-version-logic drift class.

**Neutral.** `install_host_config` adds one tool to every session's listing; its description restricts use to explicit init/setup flows and hosts gate the writes behind permissions.

## Key files

- CLI: @cmd/init.go, @cmd/host_wiring.go, @cmd/hooks_install.go, @cmd/mcp_root.go, @cmd/hooks_claude_code.go, @cmd/doctor.go, @cmd/update.go, @internal/agents/mcp_helpers.go, @internal/agents/instructions.go, @internal/mcp/tools/install_host_config.go, @internal/mcp/server.go
- Plugin: @plugins/archcore/bin/detect-host, @plugins/archcore/bin/cli-gte, @plugins/archcore/bin/session-start, @plugins/archcore/hooks/hooks.json, @plugins/archcore/hooks/codex.hooks.json, @plugins/archcore/agents/archcore-auditor.toml, @plugins/archcore/skills/init/SKILL.md