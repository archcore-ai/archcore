---
title: "Codex Adapter — Detection, Hook Stdout, and Project Wiring Contract"
status: accepted
tags:
  - "codex"
  - "hooks"
  - "multi-host"
  - "plugin"
  - "validation"
---

## Purpose & Scope

This spec is normative for the Codex adapter: the manifest `@plugins/archcore/.codex-plugin/plugin.json`, the hook config `@plugins/archcore/hooks/codex.hooks.json`, the `codex` branches of `@plugins/archcore/bin/`, and the host question in `@plugins/archcore/skills/init/SKILL.md`. One adapter covers both Codex surfaces — the CLI and the desktop app — because they share one binary, one `~/.codex/config.toml`, and one plugin install. Depended on by users of both surfaces and by the `codex-cli` wiring the archcore CLI writes into a repository. Out of scope: the archcore CLI's own hook leaves (`archcore hooks codex-cli …`), the portable core the adapter calls, and every other host.

## Surface

- Host id printed by `@plugins/archcore/bin/detect-host`: `codex-cli`. Shell host id inside `bin/`: `codex`, mapped to the CLI agent id `codex-cli` by each launcher.
- Environment markers Codex injects into model-run shell commands: `CODEX_THREAD_ID` (`codex-rs/protocol/src/shell_environment.rs`, injected after filtering), `CODEX_SANDBOX`, `CODEX_SANDBOX_NETWORK_DISABLED`.
- Host id inside a hook process: pinned by the `ARCHCORE_HOST=codex` assignment prefix on every command in `@plugins/archcore/hooks/codex.hooks.json`; Codex offers no `env` field on a hook handler.
- Hook events registered by that config: `SessionStart`, `PreToolUse`, `PostToolUse`.
- Hook stdin: snake_case payload carrying `hook_event_name`, `tool_name`, `tool_input`, `cwd`, `model`, `permission_mode`, and — on turn-scoped events only — `turn_id`. Captured payloads: `@test/fixtures/stdin/codex/session-start.json`, `@test/fixtures/stdin/codex/mcp-list-documents.json`.
- Project wiring artifacts written for `codex-cli`: `.codex/config.toml` (`[mcp_servers.archcore]`) and `.codex/hooks.json`.

## Normative Behavior

1. WHEN `CODEX_THREAD_ID` is set and no Claude Code or Cursor marker is present, `bin/detect-host` MUST print `codex-cli`.
2. WHEN `CODEX_HOME` is set and no other host marker is present, `bin/detect-host` MUST print `codex-cli`.
3. IF `CLAUDECODE` and `CODEX_THREAD_ID` are both set, THEN `bin/detect-host` MUST print `claude-code`.
4. The Codex adapter MUST prefix every hook command in `hooks/codex.hooks.json` with `ARCHCORE_HOST=codex`.
5. WHEN a `bin/` hook script writes stdout in a Codex session, the script MUST emit either exactly one JSON document valid for the event or text whose first non-whitespace character is neither `{` nor `[`.
6. WHILE `ARCHCORE_HOST` is `codex`, `bin/session-start` MUST emit every informational message as a `SessionStart` document carrying `hookSpecificOutput`.
7. WHILE `ARCHCORE_HOST` is `codex`, `bin/session-start` MUST buffer every advisory instead of printing it.
8. WHEN buffered advisories exist and the CLI payload matches the Claude-shaped session-start document, `bin/session-start` MUST splice them into that document's `additionalContext` value.
9. WHEN no buffered advisory exists, `bin/session-start` MUST pass the CLI payload through byte for byte.
10. The Codex adapter MUST keep session-start `additionalContext` below the host's default threshold of 2,500 tokens, or set `additionalContextLimit` on the handler.
11. The Codex adapter MUST write every hook matcher for an MCP tool in the `mcp__<server>__<tool>` form.
12. WHEN the init skill asks which host to wire, the skill MUST offer one Codex option that names the CLI and the desktop app.
13. WHEN the user selects that option, the init skill MUST map it to the agent id `codex-cli`.
14. WHEN host wiring for `codex-cli` completes, the init skill MUST report that project trust and hook trust are both required before the wiring takes effect.
15. The Codex adapter MUST NOT introduce a host id for the desktop app.
16. The Codex adapter MUST NOT write hooks into `.codex/config.toml` while `.codex/hooks.json` carries them.

## Constraints & Invariants

- Constraint: Codex fails a hook whose stdout starts with `{` **or** `[` and does not parse against the event shape, and discards that hook's entire output (`looks_like_json` in `codex-rs/hooks/src/engine/output_parser.rs`). Items 5–9 exist for this rule alone, and item 6 exists because every plain message the plugin writes opens with `[Archcore]`.
- Constraint: a Codex `SessionStart` payload carries no `turn_id`, so the stdin heuristic in `@plugins/archcore/bin/lib/normalize-stdin.sh` cannot resolve that event; item 4 is the only thing that does.
- Constraint: Codex loads project `.codex/` layers only for a project marked `trust_level = "trusted"`, so project wiring is inert until the user trusts the repository.
- Constraint: Codex runs a non-managed command hook only after the user reviews and trusts its exact hash through `/hooks`; `codex exec` without `--dangerously-bypass-hook-trust` skips untrusted hooks silently.
- Constraint: Codex merges `hooks.json` and inline `[hooks]` found in one layer and warns at startup, which is why item 16 keeps one representation.
- Invariant: the host id `codex-cli` covers both Codex surfaces; no plugin artifact distinguishes them.
- Invariant: a hook matcher sees `mcp__<server>__<tool>` even when the model-visible namespace omits that prefix.

## Failure Behavior

1. IF the buffered advisories cannot be spliced because the CLI payload does not match the expected shape, THEN `bin/session-start` MUST print the payload unchanged and write the advisories to stderr.
2. IF `.archcore/` is absent, THEN `bin/session-start` MUST emit the init guidance as one `SessionStart` document.
3. IF the archcore CLI is absent from PATH, THEN `bin/session-start` MUST emit the install advisory as one `SessionStart` document.
4. IF the archcore CLI is older than v0.7.0, THEN `bin/pre-tool-use` and `bin/post-tool-use` MUST exit 0 without writing stdout.
5. IF the hook's working directory resolves inside a plugin cache, THEN the script MUST exit 0 and write nothing.
6. IF a hook exits non-zero on Codex, THEN Codex reports a hook failure and continues the turn; no `bin/` script MUST rely on a non-zero exit for anything other than the documented PreToolUse deny.

## Conformance

An implementation is conformant when it satisfies behaviors 1–16, holds both invariants, follows the failure rules, and passes: `@test/unit/detect-host.bats` (items 1–3), `@test/structure/hooks.bats` (items 4, 11 and 16), `@test/unit/session-start-codex-stdout.bats` (items 5–9 and failure rules 1–3), `@test/unit/normalize-stdin.bats` (the captured-payload cases behind item 4), `@test/structure/init-skill.bats` (items 12–14), and the `codex` row of the probe records in `host-probe-protocol.spec`.

Given a repository whose `.archcore/` is empty, when a Codex session starts, then stdout holds one JSON document whose `additionalContext` ends with the empty-state advisory.
