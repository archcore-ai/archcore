---
title: "Add Read-Only Auditor Agent"
status: accepted
tags:
  - "agents"
  - "architecture"
  - "plugin"
---

## Context

The plugin originally adopted a single universal agent, recorded in `single-universal-agent.adr`, in which `archcore-assistant` handled every complex documentation task: creation, updating, relation management, and requirements engineering. In practice two usage patterns separated cleanly — mutation tasks, which need write access to MCP tools and benefit from multi-step planning, and audit tasks, which are purely read-only and can run in the background. Combining both in one agent meant every audit ran with full write permissions, and no background audit could run alongside other work without risk of an unwanted change.

## Decision

Add a second agent, **`archcore-auditor`**, alongside `archcore-assistant` — read-only tools (`list_documents`, `get_document`, `list_relations`, `Read`, `Grep`, `Glob`), `background: true`, `model: sonnet`, and a structured audit report categorized as critical, warning, and info — extending rather than replacing the single-agent design.

`archcore-assistant` keeps its role as the read/write agent for complex multi-document tasks.

## Alternatives Considered

1. **Keep one agent and add an audit mode** — rejected because instructing the assistant to "only read" carries no enforcement: it still holds write tools, and it cannot run in the background by default.
2. **Create one specialized agent per task type — creation, audit, requirements, relations** — rejected because it multiplies maintenance and overlapping domain knowledge while making discovery harder; two agents split by read and write is the granularity that matches the observed usage patterns.

## Consequences

- An audit is safe by construction: with no write tool in the allow-list, an accidental mutation is impossible rather than merely unlikely.
- A background audit does not block the user's workflow.
- Responsibilities separate cleanly: the assistant mutates, the auditor observes.
- Audits cost less, running on Sonnet with read-only operations.
- Tradeoff: two agents to maintain instead of one, though they share the same domain-knowledge patterns.
- Tradeoff: `agent-system.spec` had to grow to cover the auditor.
- Tradeoff: users must learn when to use which, mitigated by the trigger phrasing in each `description`.
- Migration performed: `archcore-assistant` gained `model: sonnet`, `maxTurns: 20`, and `color: blue`; `archcore-auditor` was created with the read-only tool set and `background: true`.

## Superseded when

- A host ships per-invocation tool scoping that lets one agent run with a read-only tool set on demand, which would remove the reason the split exists.
- Audit findings require mutation to be useful — for example an auto-fix mode — which would put the auditor's read-only constraint in conflict with its purpose.
