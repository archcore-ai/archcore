---
title: "JetBrains: junie Agent in the CLI With .junie/mcp/mcp.json and AGENTS.md, Skills Through a Junie Extension, and a Verified Claude Agent ACP Guide"
status: draft
tags:
  - "component:cli"
  - "integrations"
  - "jetbrains"
  - "multi-host"
---

## Goal

Reach JetBrains IDE users through two routes. First, `archcore init` detects Junie and writes `.junie/mcp/mcp.json` and the managed block in the file Junie reads; skills arrive as a Junie extension from the existing `archcore-plugins` marketplace. Junie gets no write guard. Second, a verified guide shows whether Claude Agent in a JetBrains IDE loads the Archcore Claude Code plugin. The contract is `junie-agent.spec`; the decision is `junie-delivery.adr` (accepted 2026-11-04).

Host facts: Junie docs dated 2026-10-30; JetBrains AI reached 12.8% in the Stack Overflow 2026 AI coding agents table.

## Tasks

### Phase 1 — junie agent in the CLI

1. Add `AgentID` `junie` and its constructor to the `all` slice in `@cli/internal/agents/agents.go`.
2. Add `@cli/internal/agents/junie.go` with detection by the `.junie/` directory.
3. Set `MCPConfigPath` to `.junie/mcp/mcp.json` and write it with `WriteStandardMCPJSON` from `@cli/internal/agents/mcp_helpers.go`.
4. Select the instruction file per spec behaviors 5–6 with the helpers in `@cli/internal/agents/instructions.go`.
5. Add `@cli/internal/agents/junie_test.go` for spec behaviors 1–9 and Failure Behavior 1–2.
6. Add a Junie row with a UI note to `@cli/internal/plugin/hosts.go` and map `junie` in `@cli/internal/plugin/plugin.go`.
7. Add row tests to `@cli/internal/plugin/hosts_test.go` and `plan_test.go`.
8. Update the agent-enumerating tests in `@cli/cmd/` that list every agent.
9. Amend `plugin-delivery.spec` with the Junie UI-note row.

### Phase 2 — Live verification of Junie

10. Install a JetBrains IDE with the Junie plugin and Junie CLI on the probe machine.
11. Run `archcore init --agent junie` and check that a Junie session lists the Archcore MCP tools.
12. Add the `archcore-ai/archcore` marketplace in Junie CLI `/extensions` and install `archcore`.
13. Check that the Archcore skills appear in a Junie session after task 12.
14. If task 12 fails, record the failure in `junie-delivery.adr` and stop skill delivery work.

### Phase 3 — Claude Agent over ACP

15. Install the Archcore Claude Code plugin at user scope on the probe machine.
16. Start a Claude Agent session in a JetBrains IDE on a repository with `.archcore/`.
17. Record whether the recap arrives and whether a direct `.archcore/` write is refused.
18. If task 17 fails, repeat it with the plugin at project scope.
19. Write the guide `jetbrains-claude-agent` with the setup that passed.
20. Record the result in `wrapper-hosts-defer-to-inner-agent-adapter.adr`.

### Phase 4 — Documentation

21. Add Junie to `supported-ai-agents.doc`: registry row, integration level, instruction-nudge row.
22. Add Junie to `@README.md`.
23. Report the roster change to the `archcore` global `supported-ai-hosts.doc` through its project.

## Acceptance Criteria

- `make verify` passes at the repository root.
- `archcore init --agent junie` writes `.junie/mcp/mcp.json` and the block; a second run changes no byte.
- A Junie session lists the Archcore MCP tools, and after the extension install, the Archcore skills.
- The guide states, with a probe date, whether Claude Agent in JetBrains enforces the `.archcore/` write guard.

## Dependencies

- A JetBrains IDE with the Junie plugin, Junie CLI, and AI Assistant on the probe machine.
- The Archcore Claude Code plugin for Phase 3.

## Clarifications

- 2026-11-04: the maintainer chose a `junie` agent in the CLI plus the Claude Agent ACP check, and accepted the decisions this plan needs.
- 2026-11-04: skills arrive through a Junie extension, not through CLI file copies (`junie-delivery.adr`).
- The ACP probe here and Phase 2 of `openclaw-skill-and-acp.plan` use different clients; each records its own result.

## Declared Delta

- create spec `junie-agent` (draft, 2026-11-04).
- create adr `junie-delivery` (accepted, 2026-11-04).
- create guide `jetbrains-claude-agent` (Phase 3, task 19).
- amend spec `plugin-delivery` — Junie UI-note row.
- amend adr `wrapper-hosts-defer-to-inner-agent-adapter` — JetBrains ACP result.
- amend doc `supported-ai-agents` — Junie rows.
