---
title: "Hermes Agent Adapter: Skills and MCP First, Then Shell Hooks Through archcore hooks hermes"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "hermes"
  - "multi-host"
  - "plugin"
---

## Goal

Give Hermes Agent users the `archcore` MCP server, the session recap, and the `.archcore/` write guard through `archcore init --agent hermes`, which writes the Hermes profile. The guard runs in shell hooks that call `archcore hooks hermes <leaf>`; there is no Python and no post-write validation text. The contract is `hermes-integration.spec`; the decision is `hermes-hooks-through-shell-leaves.adr` (accepted 2026-11-04).

Host facts: `NousResearch/hermes-agent` commit `cad0aaa`, release `v2026.9.24`; about 260,000 GitHub stars, no public user count on 2026-11-04.

## Tasks

### Phase 1 — CLI dialect and root resolution

1. Add `AgentID` `hermes` to `@cli/internal/agents/agents.go`.
2. Add an envelope `envelopeHermesContext` (`{"context": …}`) in `@cli/cmd/hook_dialect.go`.
3. Add the `hermes` row: that envelope for session and tool events, `denyExitTwo`, `preToolContext: false`.
4. Resolve the project from an absolute target path for `hermes`, ignoring payload `cwd`, in `@cli/cmd/hook_payload.go`.
5. Deny a relative target with a `.archcore` segment for `hermes` (spec behavior 2).
6. Check that the V4A markers of `patch` are read by the existing patch-body scan in `@cli/cmd/hook_payload.go`.
7. Gate the `hermes` session leaf on `extra.is_first_turn` (spec behaviors 5–6).
8. Add fixtures under `@plugin/test/fixtures/stdin/hermes/` for `write_file`, `patch` replace, `patch` V4A, and `pre_llm_call`.
9. Add Go table tests for spec behaviors 1–6 in `@cli/cmd/`.

### Phase 2 — Profile wiring

10. Add `@cli/internal/agents/hermes.go`: detection `.hermes/` or `HERMES.md`, instruction file per spec behavior 14.
11. Write a YAML node merge for `hooks:` and `mcp_servers:` with `gopkg.in/yaml.v3`, preserving unknown keys.
12. Resolve the profile path from `HERMES_HOME`, else `~/.hermes/config.yaml`.
13. Merge the consent entries into `~/.hermes/shell-hooks-allowlist.json`.
14. Back up an invalid profile per `backup-invalid-configs.adr` before stopping.
15. Add `InstallHermesHooks` to `@cli/internal/wiring/hooks_agents.go` and register it in `hooksInstallers`.
16. Print each changed file path from `archcore init` for Hermes.
17. Add round-trip tests for spec behaviors 7–12 and the idempotence invariant.
18. Add the `hermes` row to `@plugin/test/structure/host-coverage-matrix.bats` and `@plugin/test/structure/probe-records.bats` in one commit.
19. Add the `hermes` row to `host-probe-protocol.spec` as deferred until Phase 3.

### Phase 3 — Documentation and live verification

20. Add Hermes to `supported-ai-agents.doc` with the fail-open timeout and the missing post-write text.
21. Add Hermes to `@README.md`.
22. Report the roster change to the `archcore` global `supported-ai-hosts.doc` through its project.
23. Install Hermes on the probe machine and record its version.
24. Run probes P0, A, and B per `host-probe-protocol.spec` in a gateway session.
25. Check that a relative `path` reaches the hook unchanged; update fixtures from live payloads.
26. Check whether `hermes skills tap add` accepts `archcore-ai/archcore` with path `plugins/archcore/skills`.
27. If task 26 succeeds, add the tap command to the Hermes section of `supported-ai-agents.doc`.

## Acceptance Criteria

- `make verify` passes at the repository root.
- After `archcore init --agent hermes`, a Hermes session lists the Archcore MCP tools.
- In a gateway session, `write_file` on `<repo>/.archcore/x.rule.md` returns a tool error with the CLI reason.
- A second `archcore init --agent hermes` run leaves the profile and consent files byte-identical.
- The `hermes` probe row records `pass` on P0, A, and B with an evidence pointer.

## Dependencies

- Hermes release `v2026.9.24` or newer on the probe machine.
- A CLI release carrying the `hermes` leaf.

## Clarifications

- 2026-11-04: the maintainer chose shell hooks over a Python plugin, and accepted the decisions this plan needs.
- 2026-11-04: CLI plugin delivery through `archcore plugin` covers Antigravity and Pi only; Hermes is wired by `archcore init`.
- 2026-11-04: a separate config-shape ADR is not needed; the CLI writes the user's profile, not a file in the plugin tree.

## Declared Delta

- create spec `hermes-integration` (draft, 2026-11-04).
- create adr `hermes-hooks-through-shell-leaves` (accepted, 2026-11-04).
- amend spec `hook-runtime` — `envelopeHermesContext` and the first-turn session gate.
- amend spec `hook-payload-reading` — path-based root resolution for `hermes`.
- amend doc `supported-ai-agents` — Hermes rows.
- none spec `plugin-delivery`: no Hermes row.
