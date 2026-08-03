---
title: "Single Universal Agent Design"
status: accepted
tags:
  - "agents"
  - "architecture"
  - "plugin"
---

## Context

The plugin needs subagent capability for documentation tasks that exceed what a skill or command can carry — multi-document creation, requirements engineering cascades, documentation audits, and relation graph management. Claude Code agents support a custom system prompt, tool restrictions, and model settings, so the open question was whether to ship several specialized agents (requirements-engineer, decision-recorder, documentation-reviewer) or one universal agent.

## Decision

Ship one universal agent, **`archcore-assistant`**, defined in `@plugins/archcore/agents/archcore-assistant.md`, carrying knowledge of all 18 document types, the three requirements-engineering tracks, and the four relation types, and restricted to the archcore MCP tools plus the read-only file tools Read, Grep, and Glob, with no Write, Edit, or Bash access to `.archcore/` files.

Its knowledge covers each document type's template, required sections, and selection criteria; the product track (`prd`, `idea`, `plan`), the sources track (`mrd`, `brd`, `urd`), and the ISO 29148 cascade (`brs`, `strs`, `syrs`, `srs`); when to use `implements`, `extends`, `depends_on`, and `related`, along with the common flows; and documentation review — gaps, staleness, missing relations, orphaned documents, and inconsistent statuses.

## Alternatives Considered

1. **Three or more specialized agents, split by requirements engineering, decision recording, and documentation review** — rejected because each system prompt would have to stay in sync with the evolving type system, because the user or the model would have to route correctly between them, and because the agents overlap heavily, since both a requirements engineer and a decision recorder need the same relation knowledge.
2. **No agents, relying on skills and commands alone** — rejected because a complex multi-step workflow, such as creating a PRD and decomposing it into a `brs → strs → syrs → srs` cascade with relations, benefits from agentic orchestration that skills cannot sustain, and because a documentation audit iterates over every document, which suits an agent loop.
3. **Several agents sharing a base prompt with specialization layers** — ruled out because it adds composition complexity without a clear benefit over one agent, and Claude Code supports no native prompt composition for agents.

## Consequences

- One system prompt and one set of tool restrictions to maintain, covering the full spectrum of documentation tasks.
- The user never chooses between agents, and the host can invoke this one whenever a documentation task exceeds skill-level complexity.
- Tool restrictions enforce the MCP-only principle inside agentic mode as well as inside skills.
- Tradeoff: the system prompt grows as it covers all 18 types plus the engineering patterns.
- Tradeoff: no domain specialization. [assumption] A dedicated requirements engineer might produce better ISO 29148 cascades; this has not been measured.
- Extended rather than replaced by `add-read-only-auditor-agent.adr`, which added `archcore-auditor` for read-only audit work while leaving this agent's role unchanged.

## Superseded when

- The system prompt exceeds the 2000-line constraint recorded in `agent-system.spec`, which would force a split by domain.
- A measured comparison shows a specialized requirements agent producing materially better ISO 29148 cascades than the universal agent.
