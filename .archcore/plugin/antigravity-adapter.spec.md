---
title: "Antigravity Adapter — plugins/antigravity Root, antigravity Hook Dialect, and Payload Reading"
status: draft
tags:
  - "antigravity"
  - "component:cli"
  - "component:plugin"
  - "hooks"
  - "multi-host"
---

## Purpose & Scope

This spec is normative for the Antigravity adapter: the plugin root `plugin/plugins/antigravity/`, its `hooks.json` and `mcp_config.json`, the launcher behavior under `ARCHCORE_HOST=antigravity`, and the `antigravity` row of the CLI hook dialects with its payload reading. Depended on by Antigravity sessions (`agy` CLI v1.4.12 and app v2.29.0 at the time of writing) in a project with `.archcore/`, by the Antigravity row of `plugin-delivery.spec`, and by the bats and Go suites that pin `@plugin/test/fixtures/stdin/antigravity/`. Out of scope: guard and validation policy (`hook-runtime.spec`) and the Marketplace listing.

Every host fact below comes from antigravity.google/docs (`/build-plugins`, `/hooks`, `/mcp`, `/plugins`, `/customization-discovery`), fetched 2026-11-04, and is unconfirmed against a running host until probes P0–C pass.

## Surface

- Plugin root `plugin/plugins/antigravity/`: tracked `plugin.json`, `hooks.json`, `mcp_config.json`, `agents/archcore-assistant.md`, `agents/archcore-auditor.md`; packed `skills/` and `bin/` from `@plugin/plugins/archcore/` (`antigravity-plugin-layout.adr`).
- `plugin.json`: schema `https://antigravity.google/schemas/v1/plugin.json`, `additionalProperties: false`; fields `name: "archcore"`, `displayName`, `description`, `version`, `logo`, `author`, `homepage`, `repository`, `license`.
- `hooks.json`: one named group `archcore` with `SessionStart` (flat handler array) and `PreToolUse` (matcher `write_to_file|replace_file_content|multi_replace_file_content`). Commands run with the plugin root as cwd; timeout unit is seconds.
- Hook stdin (camelCase): `conversationId`, `workspacePaths[]`, `toolCall.name`, `toolCall.args`, `stepIdx`. Write tools carry the path in `toolCall.args.TargetFile`.
- Hook stdout: deny is `{"decision":"deny","reason":<text>}` with exit 0; session context is `{"injectSteps":[{"systemMessage":{"systemMessage":<text>}}]}`. Any non-zero exit is logged and ignored (fail open).
- `mcp_config.json`: `{"mcpServers":{"archcore":{"command":"archcore","args":["mcp"]}}}`; Antigravity names the server `archcore_archcore`.
- CLI: `antigravity` row in `@cli/cmd/hook_dialect.go`; payload keys in `@cli/cmd/hook_payload.go`; launchers `@plugin/plugins/archcore/bin/session-start`, `@plugin/plugins/archcore/bin/pre-tool-use`, `@plugin/plugins/archcore/bin/lib/normalize-stdin.sh`.

## Normative Behavior

1. The `plugin.json` MUST pass `agy plugin validate --strict` at the release tag.
2. The `hooks.json` MUST route `SessionStart` to `./bin/session-start`.
3. The `hooks.json` MUST route `PreToolUse` to `./bin/pre-tool-use` with the matcher in Surface.
4. The `hooks.json` MUST NOT route `PostToolUse`, because Antigravity documents no context slot in its output.
5. WHEN stdin carries `conversationId` and `workspacePaths`, `normalize-stdin.sh` MUST set `ARCHCORE_HOST=antigravity`.
6. WHILE `ARCHCORE_HOST` is `antigravity`, the launchers MUST skip the cwd-based plugin-cache guard.
7. WHEN the `antigravity` leaf reads a tool event, the CLI MUST take the tool name from `toolCall.name`.
8. WHEN the `antigravity` leaf reads a tool event, the CLI MUST take the target path from `toolCall.args.TargetFile`.
9. WHEN `workspacePaths` holds one path that is an ancestor of the target, the CLI MUST use that path as the project root.
10. WHEN no workspace path is an ancestor of the target, the CLI MUST use `workspacePaths[0]` as the project root.
11. The CLI MUST use `conversationId` as the session key of the `antigravity` leaf.
12. WHEN the guard denies, the `antigravity` leaf MUST print the deny object of Surface and exit 0.
13. WHEN the guard allows, the `antigravity` leaf MUST print nothing and exit 0.
14. The `antigravity` leaf MUST NOT print a `decision` value other than `deny`.
15. WHEN the session leaf has context, the `antigravity` leaf MUST print the `injectSteps` object of Surface.
16. The `antigravity` dialect MUST set `preToolContext` to false.
17. The pack step MUST copy `skills/` and `bin/` byte-identical from `@plugin/plugins/archcore/`.
18. Each Antigravity agent variant MUST carry a prompt body byte-identical to its source in `@plugin/plugins/archcore/agents/`.
19. Each Antigravity agent variant MUST list only tools that Antigravity documents.

## Constraints & Invariants

- Constraint: a non-zero exit fails open on Antigravity, so `denyExitTwo` would allow every write; behavior 12 is the only working deny.
- Constraint: an `allow` decision may skip the user's permission prompt; behavior 14 keeps Archcore out of permission policy.
- Constraint: hook cwd is the plugin root under `~/.gemini/config/plugins/archcore/`; the cwd guard would exit silently there, which behavior 6 prevents.
- Constraint: Antigravity reads `.agents/` and `AGENTS.md` up to the `.git` root; Codex also uses `.agents/plugins`, so the adapter writes nothing under `.agents/`.
- Constraint: the MCP tool name inside `toolCall.name` is undocumented; no behavior here depends on it.
- Invariant: guard decision and reason text come only from the CLI leaf (`host-adapter-contract.spec`, Failure Behavior 1).
- Invariant: skill content is byte-identical to `@plugin/plugins/archcore/skills/`.

## Failure Behavior

1. IF `archcore` is absent or older than the version that carries the `antigravity` leaf, THEN the launcher MUST exit 0 without output.
2. IF stdin is not valid JSON, THEN the `antigravity` leaf MUST print nothing and exit 0.
3. IF `workspacePaths` is absent or empty, THEN the `antigravity` leaf MUST print nothing and exit 0.
4. IF `toolCall.args.TargetFile` is absent, THEN the `antigravity` leaf MUST print nothing and exit 0.
5. IF the `archcore` MCP server starts with a root inside `~/.gemini/config/plugins/`, THEN `archcore mcp` MUST refuse that root as a plugin cache.
6. IF probe C shows the MCP server without a project root, THEN the adapter MUST document a project-level `.agents/mcp_config.json` fallback.

## Conformance

An implementation conforms when it satisfies behaviors 1–19, holds the invariants, degrades per Failure Behavior, and passes: Go table tests for the `antigravity` dialect and payload keys in `@cli/cmd/`; the `antigravity` cases of `@plugin/test/unit/hook-launchers.bats`; a structure test for `plugin.json`, `hooks.json`, `mcp_config.json`, and the pack parity of behaviors 17–18; `agy plugin validate --strict`; and the `antigravity` row of `host-probe-protocol.spec` with `pass` on P0, A, B, and C.

Given a workspace with `.archcore/`, when the agent calls `write_to_file` with `TargetFile` `<ws>/.archcore/x.rule.md`, then the hook prints `{"decision":"deny",...}`, exits 0, and the file does not exist.
