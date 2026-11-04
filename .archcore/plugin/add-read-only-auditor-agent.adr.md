---
title: "Two Agents: A Read/Write Assistant and a Read-Only Auditor"
status: accepted
tags:
  - "agents"
  - "architecture"
  - "component:plugin"
  - "plugin"
---

## Context

The plugin needs subagents for documentation tasks that exceed what a skill or command can carry: multi-document creation, requirements engineering cascades, relation graph management, and documentation audits. Claude Code agents support a custom system prompt, tool restrictions, and model settings, so the open question was how many agents to ship.

Two usage patterns separate cleanly. Mutation tasks need write access to MCP tools and benefit from multi-step planning. Audit tasks are read-only and can run in the background. One agent for both means every audit runs with write permissions, and no background audit is safe next to other work.

## Decision

Ship two agents.

- **`archcore-assistant`** (`@plugin/plugins/archcore/agents/archcore-assistant.md`) is the read/write agent for complex multi-document tasks. It carries the engine's document types, the three requirements tracks, and the relation vocabulary. Its tools are the archcore MCP tools plus Read, Grep, and Glob. It has no Write, Edit, or Bash access to `.archcore/` files.
- **`archcore-auditor`** (`@plugin/plugins/archcore/agents/archcore-auditor.md`) is read-only. Its tools are `list_documents`, `search_documents`, `get_document`, `list_relations`, `Read`, `Grep`, and `Glob`. It runs with `background: true` and `model: sonnet`, and returns a structured report with critical, warning, and info findings.

The auditor has no write tool, so a mutation during an audit is impossible, not only unlikely.

## Alternatives Considered

1. **Three or more specialized agents (requirements engineer, decision recorder, documentation reviewer)** — rejected. Each system prompt must follow the evolving type system. The user or the model must route between them. The agents overlap, because both a requirements engineer and a decision recorder need the same relation knowledge.
2. **No agents, skills and commands only** — rejected. A multi-step workflow such as a PRD decomposed into a `brs → strs → syrs → srs` cascade with relations needs agentic orchestration. A documentation audit iterates over every document, which suits an agent loop.
3. **One agent with an audit mode** — rejected. Telling the assistant to "only read" has no enforcement, because it still holds write tools. It also cannot run in the background by default.
4. **Several agents sharing a base prompt with specialization layers** — rejected. It adds composition complexity, and Claude Code has no native prompt composition for agents.
5. **One specialized agent per task type (creation, audit, requirements, relations)** — rejected. It multiplies maintenance and makes discovery harder. Two agents split by read and write match the observed usage.

## Consequences

- A background audit does not block the user's workflow, and it costs less on Sonnet with read-only operations.
- The assistant mutates and the auditor observes. Tool restrictions enforce the MCP-only principle in agentic mode as well as in skills.
- Tradeoff: two agents to maintain, and `agent-system.spec` covers both.
- Tradeoff: users must learn which agent to use. The trigger phrasing in each `description` mitigates this.
- Tradeoff: no domain specialization. [assumption] A dedicated requirements engineer might produce better ISO 29148 cascades. This has not been measured.

## Superseded when

- A host ships per-invocation tool scoping that lets one agent run read-only on demand, which removes the reason for the split.
- Audit findings need mutation to be useful, for example an auto-fix mode, which conflicts with the auditor's read-only constraint.
- The assistant system prompt exceeds the 2000-line constraint in `agent-system.spec`, which forces a split by domain.
- A measured comparison shows a specialized requirements agent producing materially better ISO 29148 cascades.
