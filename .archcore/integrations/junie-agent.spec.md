---
title: "Junie Agent — Detection, .junie/mcp/mcp.json Wiring, Instruction Placement, and the Extension Note"
status: draft
tags:
  - "component:cli"
  - "integrations"
  - "jetbrains"
  - "multi-host"
---

## Purpose & Scope

This spec is normative for the `junie` agent of the CLI: how `archcore init` detects Junie, writes its MCP entry and managed instruction block, and how `archcore plugin` answers for Junie. Depended on by Junie in JetBrains IDEs and Junie CLI, and by the Go suites under `@cli/internal/agents/` and `@cli/internal/plugin/`. Out of scope: Claude Agent in JetBrains over ACP (covered by the JetBrains integration guide) and Junie hooks (`junie-delivery.adr`).

Host facts come from junie.jetbrains.com/docs pages dated 2026-10-30: `junie-cli-mcp-configuration.html`, `guidelines-and-memory.html`, `junie-ide-plugin.html`, `agent-skills.html`, `junie-cli-extensions.html`, `junie-cli-hooks.html`.

## Surface

- Agent `@cli/internal/agents/junie.go`, ID `junie`, registered in `@cli/internal/agents/agents.go`.
- Detection marker: a `.junie/` directory in the project root.
- MCP file: `.junie/mcp/mcp.json`, shape `{"mcpServers":{"archcore":{"command":"archcore","args":["mcp"]}}}`, written with `WriteStandardMCPJSON` from `@cli/internal/agents/mcp_helpers.go`. The IDE plugin and Junie CLI share this file.
- Instruction file: `.junie/AGENTS.md` when it exists, else the root `AGENTS.md`; fenced upsert through `@cli/internal/agents/instructions.go`.
- Plugin row: Junie in `@cli/internal/plugin/hosts.go` with no CLI command and a UI note naming `/extensions` and the `archcore-plugins` marketplace at `archcore-ai/archcore`.

## Normative Behavior

1. WHEN the project root holds a `.junie/` directory, the `junie` agent MUST report itself detected.
2. WHEN `archcore init` wires `junie`, the agent MUST merge the `archcore` entry into `.junie/mcp/mcp.json`.
3. IF `.junie/mcp/mcp.json` already holds an `archcore` entry, THEN the agent MUST leave that entry unchanged.
4. The agent MUST preserve every other `mcpServers` entry in `.junie/mcp/mcp.json`.
5. WHEN `.junie/AGENTS.md` exists, the agent MUST write the managed block into `.junie/AGENTS.md`.
6. WHEN `.junie/AGENTS.md` does not exist, the agent MUST write the managed block into the root `AGENTS.md`.
7. WHEN `archcore instructions remove --agent junie` runs, the agent MUST strip the block from the file behavior 5 or 6 selected.
8. The agent MUST NOT write a hook config.
9. The agent MUST NOT write files under `.junie/skills/`.
10. WHEN `archcore plugin install --agent junie` runs, the CLI MUST print the UI note and run no command.

## Constraints & Invariants

- Constraint: Junie reads `.junie/AGENTS.md` exclusively when it exists; a block in the root file is then ignored.
- Constraint: Junie hooks are Early Access and do not run under ACP; AI Chat runs Junie over ACP.
- Constraint: skills arrive through a Junie extension from the marketplace, not from the CLI (`junie-delivery.adr`).
- Constraint: the JetBrains AI Assistant 2026.2 help names only the root `AGENTS.md`; the newer Junie docs are authoritative here.
- Invariant: a second `archcore init --agent junie` run leaves `.junie/mcp/mcp.json` and the instruction file byte-identical.

## Failure Behavior

1. IF `.junie/mcp/mcp.json` is not valid JSON, THEN the agent MUST back it up and write a new file with only the `archcore` entry.
2. IF `.junie/mcp/` does not exist, THEN the agent MUST create it.

## Conformance

An implementation conforms when it satisfies behaviors 1–10, holds the invariant, degrades per Failure Behavior, and passes: `@cli/internal/agents/junie_test.go` for behaviors 1–9; `TestAllAgents_RequiredFields` in `@cli/internal/agents/agents_test.go`; row tests in `@cli/internal/plugin/hosts_test.go` for behavior 10; and a recorded check that a Junie session lists the Archcore MCP tools after `archcore init --agent junie`.

Given a project with `.junie/AGENTS.md`, when `archcore init --agent junie` runs, then the block lands in `.junie/AGENTS.md` and the root `AGENTS.md` is unchanged.
