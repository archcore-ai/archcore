---
title: "CLI Owns Layers 4–5 — MCP Track Prompts Removed, Hook Parity in Go"
status: accepted
tags:
  - "architecture"
  - "hooks"
  - "multi-host"
  - "plugin"
---

## Context

Track logic leaked into the CLI twice: five MCP track prompts (@cli `internal/mcp/prompts/` — iso, sources, product, standard, architecture) whose per-phase gate is a single fixed sentence, and REQUIREMENTS TRACKS / WORKFLOW PROMPTS sections in the server instructions — so layer 2 currently has two owners and neither is canonical. Precision policy is also encoded twice (the `create_document` content-parameter description versus the plugin's `check-precision` hook), and guardrail hooks exist only as plugin shell scripts on 4 of 8 registry hosts.

## Decision

The CLI owns layers 4–5 only: delete `internal/mcp/prompts/`, strip the track sections from server instructions while keeping the type catalog and type-selection rules, port the guardrail hooks to Go for all 8 registry hosts (hook parity), and defer the type-schema canon move (`get_type_schema`) to phase 2 after the plugin palette swap.

## Alternatives Considered

1. CLI as the canonical track engine (tracks as MCP prompts or a new `archcore track` command) — rejected because elicitation quality in fixed Go prompt text is capped at the current one-sentence confirmation gate, and the intent-architecture decision already refused orchestration logic in the CRUD server.
2. Storage-only CLI (no hooks, all guardrails in the plugin) — rejected because it strands the 4 hosts without plugin hook wiring and preserves the shell/Go duplication.
3. Ship `get_type_schema` before the palette swap — deferred because it puts CLI schema work on the swap's critical path; a bounded dual-canon window with contract tests in the CLI repo is the accepted trade.

## Consequences

- Layer 2 has one owner (plugin track files); no drift between MCP prompts and skill flows.
- Guardrails (write-guard, injection, validation, staleness) reach all 8 registry hosts; shell/Go hook duplication collapses.
- CLI-only users lose the five track prompts; the bare-host flow-guidance floor is the type-selection rules in server instructions.
- [expected] Dual-canon window for type templates stays within one release until phase 2 lands.

## Superseded when

- The elicitation contract becomes expressible in a host-portable MCP primitive available to subagents.
- Go hook handlers cannot meet the 1s (PreToolUse) / 3s (PostToolUse) budgets on any registry host.