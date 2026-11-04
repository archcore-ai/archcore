---
title: "Antigravity Adapter: Plugin Manifests, antigravity Hook Dialect, and CLI Delivery Through agy"
status: draft
tags:
  - "antigravity"
  - "component:cli"
  - "component:plugin"
  - "multi-host"
  - "plugin"
---

## Goal

Ship Archcore to Google Antigravity as the plugin root `plugin/plugins/antigravity/`, installed by `agy plugin install`. An Antigravity session gets the session recap, the `.archcore/` write guard, the `archcore` MCP server, the skills, and both agents. `archcore init`, `archcore plugin`, and `archcore update` deliver and refresh it. The contract is `antigravity-adapter.spec`; the layout decision is `antigravity-plugin-layout.adr` (accepted 2026-11-04).

Why this host: Antigravity replaced Gemini CLI for unpaid and Google One users on 2026-06-18, and Alphabet reported about 2.4 million weekly users on its Q2 2026 earnings call (CNBC, 2026-07-22).

## Tasks

### Phase 1 — CLI dialect and payload reading

1. Add a deny style `denyDecisionJSON` in `@cli/cmd/hook_dialect.go`: stdout `{"decision":"deny","reason":…}`, exit 0.
2. Add a session envelope `envelopeInjectSteps` in `@cli/cmd/hook_dialect.go` per the spec's Surface.
3. Add the `antigravity` row: `envelopeInjectSteps` session, `denyDecisionJSON`, `preToolContext: false`.
4. Add `AgentID` `antigravity` to `@cli/internal/agents/agents.go`.
5. Read `toolCall.name`, `toolCall.args`, and `toolCall.args.TargetFile` in `@cli/cmd/hook_payload.go`.
6. Resolve the project root from `workspacePaths` per spec behaviors 9–10 in `@cli/cmd/hook_payload.go`.
7. Read `conversationId` as the session key in `@cli/cmd/hook_payload.go`.
8. Add `~/.gemini/config/plugins/` to the plugin-cache roots `resolveProjectRoot` refuses; no target found until the refusal list is located.
9. Add Go table tests for behaviors 7–16 and Failure Behavior 2–4 in `@cli/cmd/`.

### Phase 2 — Launchers

10. Detect `antigravity` from `conversationId` and `workspacePaths` in `@plugin/plugins/archcore/bin/lib/normalize-stdin.sh`.
11. Skip the cwd guard for `antigravity` in `@plugin/plugins/archcore/bin/lib/plugin-cache-guard.sh` and `@plugin/plugins/archcore/bin/session-start`.
12. Add synthetic fixtures under `@plugin/test/fixtures/stdin/antigravity/` from the documented payload.
13. Add `antigravity` cases to `@plugin/test/unit/hook-launchers.bats`.

### Phase 3 — Plugin root

14. Create `@plugin/plugins/antigravity/plugin.json` with the spec's fields.
15. Create `@plugin/plugins/antigravity/hooks.json` with the `archcore` group of the spec.
16. Create `@plugin/plugins/antigravity/mcp_config.json` with the `archcore` server.
17. Write `agents/archcore-assistant.md` and `agents/archcore-auditor.md` in Antigravity frontmatter.
18. Map each source `tools:` entry to a documented Antigravity tool name in those agents.
19. Write a pack script copying `skills/` and `bin/` from `@plugin/plugins/archcore/`.
20. Gitignore the packed `skills/` and `bin/` under `plugin/plugins/antigravity/` in `@.gitignore`.
21. Extend `@scripts/export-plugin.sh` to run the pack step and export the root.
22. Add a structure test for the manifest files and pack parity under `@plugin/test/structure/`.
23. Add `agy plugin validate --strict` to `@plugin/Makefile`, skipping with a message when `agy` is absent.
24. Add the `antigravity` row to `@plugin/test/structure/host-coverage-matrix.bats` and `@plugin/test/structure/probe-records.bats` in one commit.
25. Add the `antigravity` row to `host-probe-protocol.spec` as deferred until Phase 5.
26. Amend `stack-and-tooling.rule` item 14 with the three manifest shapes of the ADR.

### Phase 4 — CLI delivery

27. Add `@cli/internal/agents/antigravity.go` and its test: instructions `AGENTS.md`, no MCP writer, detection by `agy` on `PATH`.
28. Add `HostAntigravity` to `hostOrder` and `agentToHost` in `@cli/internal/plugin/plugin.go`.
29. Add the row to `@cli/internal/plugin/hosts.go`: listing `agy plugin list --json`, registry `~/.gemini/config/plugins/archcore/`.
30. Set its install to `agy plugin install <subpath URL>` and its remove to `agy plugin uninstall archcore`.
31. Set its update to the same install command, because `agy` has no update verb.
32. Read Antigravity evidence from the JSON listing or the registry directory in `@cli/internal/plugin/evidence.go`.
33. Add row tests to `@cli/internal/plugin/hosts_test.go`, `plan_test.go`, `execute_test.go`, and `evidence_test.go`.
34. Correct the host-count comments in `@cli/internal/plugin/plugin.go`, `@cli/cmd/plugin_run.go`, and `@cli/cmd/init.go`.
35. Amend `plugin-delivery.spec` and `updating-the-plugin.spec` per the Declared Delta.
36. Add Antigravity to `supported-ai-agents.doc`, `cli-hooks-reference.doc`, and `@README.md`.
37. Report the roster change to the `archcore` global `supported-ai-hosts.doc` through its project.

### Phase 5 — Live verification and listing

38. Install Antigravity on the probe machine and record the app and `agy` versions.
39. Verify one GitHub-subpath URL form for `agy plugin install` and set it in task 30.
40. Run probes P0, A, B, and C per `host-probe-protocol.spec`.
41. If probe C shows no project root for the MCP server, write the `.agents/mcp_config.json` fallback guide.
42. Capture live payloads and replace the synthetic fixtures; record the MCP tool-name spelling.
43. Confirm that empty stdout on PreToolUse logs no hook error.
44. Submit the plugin through the Antigravity Marketplace partner intake form.

## Acceptance Criteria

- `make verify` passes at the repository root.
- `agy plugin validate --strict plugin/plugins/antigravity` passes after the pack step.
- In an Antigravity session, a `write_to_file` under `.archcore/` is refused with the CLI reason, and the file does not exist.
- `archcore plugin install --agent antigravity` installs the plugin; a second run runs no install command.
- The `antigravity` probe row records `pass` on P0, A, B, and C with an evidence pointer.

## Dependencies

- Antigravity app and `agy` CLI v1.4.12 or newer on the probe machine.
- A CLI release carrying the `antigravity` leaf before the plugin release that calls it (`plugin-cli-compatibility.rule`).

## Clarifications

- 2026-11-04: the maintainer chose CLI delivery as for OpenCode, and accepted the decisions this plan needs.
- 2026-11-04: host facts verified from antigravity.google/docs; no public `agy` source exists.

## Declared Delta

- create spec `antigravity-adapter` (draft, 2026-11-04).
- create adr `antigravity-plugin-layout` (accepted, 2026-11-04).
- amend spec `hook-runtime` — the `denyDecisionJSON` style and `envelopeInjectSteps` envelope.
- amend spec `hook-payload-reading` — `toolCall.*`, `workspacePaths`, `conversationId`.
- amend spec `plugin-delivery` — Antigravity row; `antigravity` among the `--agent` hosts.
- amend spec `updating-the-plugin` — Antigravity update by reinstall.
- amend rule `stack-and-tooling` — item 14 shapes for `plugins/antigravity/`.
- amend doc `supported-ai-agents` and doc `cli-hooks-reference` — Antigravity rows.
