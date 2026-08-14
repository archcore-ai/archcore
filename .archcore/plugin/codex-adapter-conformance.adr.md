---
title: "Codex Adapter Conformance — Runtime Env Marker for Detection and Single-Document Hook Stdout"
status: draft
tags:
  - "architecture"
  - "codex"
  - "hooks"
  - "multi-host"
  - "plugin"
---

## Context

`@plugins/archcore/bin/detect-host` resolves Codex from `CODEX_HOME`, which Codex never injects into a model-run shell command — the variable is read *from* the user to locate the config directory — so every Codex session, CLI and desktop app alike, resolves to `__UNKNOWN__` and reaches host wiring through the init skill's ask, whose four options name no desktop surface. Independently, `@plugins/archcore/hooks/codex.hooks.json` pinned no host id and a Codex `SessionStart` payload carries no `turn_id`, so the stdin heuristic read every Codex session start as Claude Code — which is why `@plugins/archcore/bin/session-start` streamed the CLI hook's JSON and then appended plain-text advisories, and Codex fails any hook whose stdout starts with `{` or `[` and does not parse (`looks_like_json` in `codex-rs/hooks/src/engine/output_parser.rs`), dropping the whole payload: measured on 2026-08-14, an empty `.archcore/` produced `hook: SessionStart Failed` and no Archcore context in the session. The same class of failure was already solved for Copilot in `_archcore_copilot_flush`, one host too narrowly.

## Decision

Correct the Codex adapter in four places without adding a host id: key the Codex branch of `bin/detect-host` on `CODEX_THREAD_ID` with `CODEX_HOME` retained as a user-set fallback; pin `ARCHCORE_HOST=codex` as an assignment prefix on every command in `hooks/codex.hooks.json`, since Codex offers no `env` field on a hook handler; generalize the Copilot single-document stdout path in `bin/session-start` into a host set that includes `codex` — splicing buffered advisories into the CLI payload's `additionalContext`, emitting every standalone message as a `SessionStart` document because `looks_like_json` fires on `[` and every plain message opens with `[Archcore]`, and falling back to stderr when the payload shape does not match; and relabel the init skill's host question so the Codex option reads as covering both the CLI and the desktop app while still mapping to the shipped agent id `codex-cli`.

This supersedes the Codex sentence of `host-wiring-parity.adr`, which records `CODEX_HOME` as the detection signal.

## Alternatives Considered

- **Add a `codex-desktop` host id across the plugin and the CLI `--agent` set** — rejected because the desktop app and the CLI share one binary, one `~/.codex/config.toml`, and one plugin install; a second id would duplicate every wiring path to produce byte-identical output, and `archcore init --agent codex-desktop` would write the files `codex-cli` already writes.
- **Detect Codex from `CODEX_SANDBOX`** — ruled out because the variable is absent under `danger-full-access`, so the marker would disappear exactly in the permission mode a first-run wiring session is most likely to use.
- **Leave the host id to the stdin heuristic and add a `SessionStart` discriminator such as `model`** — rejected because the field is a Codex-specific extension whose absence on other hosts is not contractual, and a wrong guess routes the CLI call to the wrong dialect; the assignment prefix is deterministic and testable.
- **Emit the Codex session-start payload as plain text instead of JSON** — ruled out because plain stdout carries no `systemMessage`, which is the channel that reports the CLI version and MCP connectivity, and because a message opening with `[Archcore]` is itself parsed as JSON by this host.
- **Drop the advisories on Codex** — deferred as the fallback arm only: it is what the implementation does when the CLI payload does not match the expected shape, but making it the primary behavior would silence the CLI-update nudge on a whole host.
- **Invert the detection precedence so `CODEX_THREAD_ID` outranks `CLAUDECODE`** — rejected because both hosts inject their marker per spawned command and inherit the other's under the default `inherit = "all"` policy, so the inversion trades one nesting misdetection for its mirror image without net gain.

## Consequences

- Positive: a Codex session resolves to `codex-cli` from the environment, so `/archcore:init` stops asking a question whose options do not name the user's surface. [expected] The ask remains reachable for GitHub Copilot CLI, which still has no marker.
- Positive: Codex SessionStart output stays one parseable document. Verified live on 2026-08-14 against Codex CLI 0.147.0 in a repository with an empty `.archcore/`: the fixed script produced `hook: SessionStart Completed` and the model quoted back both the corpus context and the empty-state advisory, while the installed 0.7.2 plugin in the same session produced `hook: SessionStart Failed`.
- Positive: the CLI now receives the `codex-cli` dialect on Codex sessions. Until this change the launchers passed `claude-code`, because the host id never resolved.
- Positive: the stdout invariant becomes testable rather than incidental — the plain-text arm of `_archcore_emit_info` passed on Codex only because the host id never selected it.
- Negative: `bin/session-start` gains a second buffering host, so the advisory path has two shapes to keep in sync — Copilot's flat `{"additionalContext":…}` and the Claude-shaped `hookSpecificOutput` wrapper the Codex CLI leaf emits.
- Negative: the splice depends on the CLI's session-start payload shape; when the CLI changes key order the advisory is routed to stderr and disappears from the model's context until the pattern is updated. [expected] The payload itself is never corrupted, because the fallback prints it unchanged.
- Negative: a Codex session started from a Claude Code shell still resolves to `claude-code`, which is unchanged from today's behavior and not fixed by this decision.

## Superseded when

- Codex injects a documented, stable host marker other than `CODEX_THREAD_ID` into model-run shell commands, or removes that variable from `codex-rs/protocol/src/shell_environment.rs`.
- Codex accepts mixed stdout — a JSON document followed by plain text — on any hook event, making the buffering path unnecessary on this host.
- Codex adds an `env` field to a hook handler, at which point the assignment prefix becomes redundant.
- The Codex desktop app stops sharing `~/.codex/config.toml` with the CLI, at which point a separate host id becomes the smaller change.
