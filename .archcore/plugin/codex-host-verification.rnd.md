---
title: "Codex Host Contract — Source-Verified Facts for the CLI and the Desktop App"
status: accepted
tags:
  - "codex"
  - "hooks"
  - "multi-host"
  - "plugin"
  - "validation"
---

## Goal

Establish, from OpenAI's published documentation and the `openai/codex` source, which Codex behaviors the Archcore adapter depends on — and settle the three field defects that rest on them: `bin/detect-host` returning `__UNKNOWN__` inside Codex, a SessionStart hook that Codex reports as failed, and the missing host id for the Codex desktop app. Verified against Codex CLI 0.147.0, Codex desktop app 26.608.12217, and plugin 0.7.2.

## Questions

1. Which environment variables does Codex expose to a shell command the model runs, and is `CODEX_HOME` among them?
2. How does Codex parse hook stdout, and what happens when a hook prints one JSON document followed by plain text?
3. Where does Codex load hooks and project config from, and what does the user have to consent to first?
4. Which tool name does a hook matcher see for an MCP tool call?
5. Does the desktop app need a host id of its own?
6. Which limit applies to hook context, and how close is the current payload to it?
7. How does a `bin/` script learn that it is running under Codex?

## Approach

Three source tiers, in this order. Official pages — `developers.openai.com/codex/hooks`, `developers.openai.com/plugins/build/plugins`, and the Codex manual at `developers.openai.com/codex/codex-manual.md` — fetched 2026-08-14. The `openai/codex` source at `main` for the exact predicates behind the prose. Live probes on one macOS 26.5.2 machine: five `codex exec` runs and two `codex doctor` runs, with and without project trust. Each finding names the tier that carries it.

## Findings

**Detection — `CODEX_HOME` is not a host marker.** Codex builds the shell environment in `create_env` (`codex-rs/protocol/src/shell_environment.rs`) and injects `CODEX_THREAD_ID` as its final step, after the `include_only` filter, so the variable survives every inheritance policy. `codex-rs/core/src/tools/runtimes/mod.rs` carries `CODEX_SESSION_ID` and `CODEX_PERMISSION_PROFILE` through the same override path. `CODEX_HOME` appears in none of them: it is read *from* the user to locate the config directory, and the manual's own examples spell it `${CODEX_HOME:-$HOME/.codex}`. A live `printenv` inside `codex exec` returned `CODEX_THREAD_ID`, `CODEX_SANDBOX`, `CODEX_SANDBOX_NETWORK_DISABLED`, `CODEX_CI`, and no `CODEX_HOME`. The plugin's Codex branch therefore never fired, in the CLI and in the desktop app alike.

**Inheritance default is `all`, and it leaks both ways.** `impl Default for ShellEnvironmentPolicy` (`codex-rs/protocol/src/config_types.rs`) sets `inherit: All`. Under `inherit = "core"` the allowlist is `PATH`, `SHELL`, `TMPDIR`, `TEMP`, `TMP`, `HOME`, `LANG`, `LC_ALL`, `LC_CTYPE`, `LOGNAME`, `USER` — which drops a user-exported `CODEX_HOME` as well. The default also means a Codex session launched from a Claude Code shell inherits `CLAUDECODE=1`, observed live; a Claude Code session launched from a Codex shell inherits `CODEX_THREAD_ID` symmetrically. Environment alone cannot separate the two nestings.

**A hook process learns its host from nothing at all.** The hooks page lists the handler keys — `type`, `command`, `commandWindows`, `timeout`, `statusMessage`, `additionalContextLimit`, `async` — and there is no `env` among them, unlike Copilot's config. `hooks/codex.hooks.json` therefore passed no host id, leaving the stdin heuristic in `@plugins/archcore/bin/lib/normalize-stdin.sh`, which keys Codex on `turn_id`. The captured payload at `@test/fixtures/stdin/codex/session-start.json` shows why that fails for the one event that matters most: a `SessionStart` payload carries `session_id`, `transcript_path`, `cwd`, `hook_event_name`, `model`, `permission_mode`, and `source` — and no `turn_id`, which the documentation lists for turn-scoped events only. Every Codex session start was therefore processed as Claude Code, including the dialect passed to `archcore hooks <host> session-start`.

**Hook stdout is one document or none — and `[` counts as JSON.** `looks_like_json` (`codex-rs/hooks/src/engine/output_parser.rs`) tests whether the trimmed stdout starts with `{` **or** `[`. `session_start.rs` parses that stdout against the event shape; if the parse fails while `looks_like_json` holds, it sets `HookRunStatus::Failed` with `hook returned invalid session start JSON output` and the whole payload is dropped. `pre_tool_use.rs`, `post_tool_use.rs`, `user_prompt_submit.rs`, and `stop.rs` carry the same predicate with their own message. Stdout that starts with neither character becomes model context verbatim, which the hooks page states as "Plain text on `stdout` is added as extra developer context". The bracket half of the predicate is load-bearing here: every plain message the plugin writes opens with `[Archcore]`, so the plain-text arm is unsafe on this host even without a JSON payload in front of it.

**The plugin hit that rule in production.** `@plugins/archcore/bin/session-start` prints the CLI hook's JSON, then `_archcore_note` appends `""` and an advisory line on every host except Copilot. Two of those advisories are reachable on Codex: the empty-`.archcore/` nudge, and the once-per-24h CLI-update advisory, which fires in any repository. Reproduced live on 2026-08-14: an empty `.archcore/` produced `hook: SessionStart Failed` in `codex exec`, with the Archcore context absent from the session; the same repository with documents present produced `hook: SessionStart Completed` and the full corpus listing. A later run put both versions in one session — the installed 0.7.2 plugin failed while the corrected script completed and delivered its context.

**Hook and config discovery, and the two consents.** The hooks page lists four locations — `~/.codex/hooks.json`, `~/.codex/config.toml`, `<repo>/.codex/hooks.json`, `<repo>/.codex/config.toml` — and states "Project-local hooks load only when the project `.codex/` layer is trusted"; the manual repeats it for config (`projects.<path>.trust_level`, "Untrusted projects skip project-scoped `.codex/` layers, including project-local config, hooks, and rules"). A second, independent consent covers execution: "Before a non-managed command hook can run, Codex requires you to review and trust the exact hook definition", recorded against the hook's hash, reviewed through `/hooks`, bypassable per invocation with `--dangerously-bypass-hook-trust`. Both were reproduced: `codex doctor` reported one MCP server in an untrusted project and two with `trust_level = "trusted"`; a project hook wrote its marker file only under the bypass flag. Plugin-bundled hooks use the same trust flow. A layer that carries both `hooks.json` and inline `[hooks]` merges and warns — observed verbatim as "prefer a single representation for this layer".

**MCP tool names reach matchers with the `mcp__` prefix.** `hook_tool_name()` in `codex-rs/core/src/tools/handlers/mcp.rs` returns `ensure_mcp_prefix(join_tool_name(...))`, so a matcher sees `mcp__<server>__<tool>` even where the model-visible namespace drops the prefix behind the feature flag described in `codex-rs/features/src/lib.rs`. A captured PostToolUse payload on 2026-08-14 carried `"tool_name":"mcp__archcore__list_documents"` while the same session's rollout recorded the model calling bare `list_documents`. The matchers in `@plugins/archcore/hooks/codex.hooks.json` and the field extraction in the `codex` branch of `normalize-stdin.sh` are correct as written.

**The desktop app is the same core.** `codex app` launches it from the CLI; the bundle ships the same binary at `/Applications/Codex.app/Contents/Resources/codex`; both surfaces read `~/.codex/config.toml`, where the desktop keeps only a `[desktop]` section of its own. Plugin installs, marketplaces, hooks, and MCP registrations are shared state. No behavior in this document differs between the two, so the adapter needs no second host id — only wording that lets a desktop user recognize the option. One open item is inherited rather than verified here: `host-wiring-parity.adr` records upstream issue #13025, that Codex Desktop ignores project-level MCP config. The project-config path was re-verified on the CLI only.

**Plugin packaging.** The official manifest fields are `name`, `version`, `description`, publisher metadata, `skills`, `mcpServers`, `hooks`, the compatibility `apps`, and `interface`; `hooks` is documented, and a plugin storing hooks at `./hooks/hooks.json` needs no manifest entry at all. Codex reads `.codex-plugin/plugin.json` and falls back to `.claude-plugin/plugin.json`. There is no `agents` or `subagents` field, so the Codex TOML subagents this repository ships have no plugin delivery path today.

**Context limit.** The hooks page caps model-visible hook output at "roughly 2,500 tokens" by default, spilling the remainder to `<temp_dir>/hook_outputs/<session_id>/<uuid>.txt` and giving the model a head-and-tail preview; `additionalContextLimit` per handler overrides the threshold, with `0` meaning no cap. Measured on this repository (101 documents, 2026-08-14): `archcore hooks codex-cli session-start` emits 3,725 characters of `additionalContext`, about 930 tokens — 37% of the default threshold.

## Recommendation

Key the Codex branch of `bin/detect-host` on `CODEX_THREAD_ID`, keeping `CODEX_HOME` only as a user-set fallback, and leave the `claude-code > cursor > codex-cli` precedence unchanged, since no environment variable resolves the nesting ambiguity in either direction. Pin `ARCHCORE_HOST=codex` as an assignment prefix in `hooks/codex.hooks.json`, because no handler field carries environment and no stdin field identifies a Codex `SessionStart`. Extend the single-document stdout discipline that `bin/session-start` already implements for Copilot to Codex, treat "exactly one JSON document, or text starting with neither `{` nor `[`" as the adapter's stdout invariant on every Codex hook event, and route the standalone messages through the JSON arm rather than plain text. Keep the shipped host id `codex-cli` for both surfaces and change only the label the init skill offers. Record the two consents in the wiring documentation instead of adding retry logic for them. Leave `additionalContextLimit` at its default while the measured payload stays under the threshold, and pin the measurement so a growing corpus does not cross it unnoticed.

## Next Action

Re-verify the project-level MCP claim for the desktop app (upstream issue #13025) before treating `.codex/config.toml` wiring as equivalent across both surfaces.
