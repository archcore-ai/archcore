---
title: "Add Read-Only Auditor Agent"
status: accepted
tags:
  - "agents"
  - "architecture"
  - "plugin"
---

## Context

The plugin originally adopted a single universal agent design (see `single-universal-agent.adr`). The `archcore-assistant` handles all complex documentation tasks: creation, updating, relation management, requirements engineering.

In practice, there are two distinct usage patterns:

1. **Mutation tasks** — creating/updating documents, managing relations, requirements decomposition. These need write access to MCP tools and benefit from careful, multi-step planning.
2. **Audit tasks** — reviewing documentation health, finding gaps, checking consistency. These are purely read-only and can safely run in the background without risk of unwanted changes.

Combining both in one agent means audits run with full write permissions, and there's no way to run a background audit while working on other tasks without risk.

## Decision

Add a second agent `archcore-auditor` alongside the existing `archcore-assistant`. This extends — not replaces — the single agent design:

- **`archcore-assistant`** — read/write agent for complex multi-document tasks (unchanged role)
- **`archcore-auditor`** — read-only agent for documentation health checks and audits

Key properties of the auditor:
- **Read-only tools only**: `list_documents`, `get_document`, `list_relations`, `Read`, `Grep`, `Glob`
- **Background execution**: `background: true` — runs without blocking the user
- **Lighter model**: `model: sonnet` — audit doesn't need Opus-level reasoning
- **Structured output**: produces a categorized audit report (critical/warning/info)

## Alternatives Considered

### Keep single agent, add audit mode
The assistant could be instructed to "only read" for audit tasks. But there's no enforcement — it still has write tools available, and it can't run as background by default.

### Create many specialized agents
One agent per task type (creation, audit, requirements, relations). Rejected: too much maintenance, overlapping knowledge, harder to discover. Two agents with clear read/write split is the right granularity.

## Consequences

**Positive:**
- Audits are safe by design — no write tools means no accidental mutations
- Background audits don't block the user's workflow
- Clear separation of concerns: assistant mutates, auditor observes
- Lower cost for audits (Sonnet model, read-only operations)

**Negative:**
- Two agents to maintain instead of one (but they share domain knowledge patterns)
- Agent-system spec needs updating to cover the auditor
- Users need to understand when to use which (mitigated by clear descriptions)

**Migration:**
- `archcore-assistant` updated with `model: sonnet`, `maxTurns: 20`, `color: blue`
- `archcore-auditor` created with read-only tool set and `background: true`
- This ADR extends `single-universal-agent.adr` — the original design was sound, this is an incremental refinement
