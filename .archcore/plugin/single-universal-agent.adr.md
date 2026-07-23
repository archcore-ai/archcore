---
title: "Single Universal Agent Design"
status: accepted
tags:
  - "agents"
  - "architecture"
  - "plugin"
---

## Context

The plugin needs subagent capabilities for complex documentation tasks that go beyond what skills and commands can handle — multi-document creation, requirements engineering cascades, documentation audits, and relation graph management.

Claude Code agents support custom system prompts, tool restrictions, and model settings. The question is whether to create multiple specialized agents (requirements-engineer, decision-recorder, documentation-reviewer) or a single universal agent.

## Decision

The plugin provides one universal agent: `archcore-assistant`.

This agent has:

- **Full knowledge of all 18 document types** — templates, required sections, when to use each type
- **Requirements engineering expertise** — product track (prd, idea, plan), sources track (mrd, brd, urd), ISO 29148 cascade (brs, strs, syrs, srs)
- **Relation pattern knowledge** — when to use implements, extends, depends_on, related; common flows (idea → prd → plan, mrd → brs → strs → syrs → srs)
- **Documentation review capability** — identify gaps, staleness, missing relations, orphaned documents, inconsistent statuses
- **Tool restrictions**: only archcore MCP tools (`list_documents`, `get_document`, `create_document`, `update_document`, `remove_document`, `add_relation`, `remove_relation`, `list_relations`) plus read-only file tools (Read, Grep, Glob). No Write, Edit, or Bash access to `.archcore/` files.

The agent is defined in `agents/archcore-assistant.md` with frontmatter specifying name, description, tools, and disallowedTools.

## Alternatives Considered

### Multiple specialized agents (3+)

Separate agents for requirements engineering, decision recording, and documentation review. Rejected because:

- Higher maintenance burden — each agent's system prompt must be kept in sync with Archcore's evolving type system
- Users must know which agent to pick or Claude must route correctly
- Overlap between agents (e.g., requirements-engineer and decision-recorder both need to understand relations)
- Simpler to maintain one comprehensive agent

### No agents, skills only

Rely entirely on skills for guidance and commands for actions. Rejected because:

- Complex multi-step workflows (create a PRD, then decompose into BRS → StRS → SyRS → SRS with relations) benefit from agentic orchestration
- Skills provide knowledge but not sustained focus on a multi-document task
- Documentation audits require iterating over all documents — better suited for an agent loop

### Multiple agents with shared base prompt

A base prompt included in all agents, with specialization layers. Rejected because:

- Added complexity without clear benefit over one agent
- Claude Code doesn't natively support prompt composition for agents

## Consequences

### Positive

- Single agent to maintain — one system prompt, one set of tool restrictions
- Agent covers the full spectrum of documentation tasks
- Users don't need to choose between agents
- Claude can invoke the agent whenever a documentation task exceeds skill-level complexity
- Tool restrictions ensure the MCP-only principle is enforced even in agentic mode

### Negative

- System prompt may become long as it covers all 18 types plus engineering patterns
- No domain specialization — a dedicated requirements-engineer might produce slightly better output for ISO 29148 cascades
- If the agent's scope grows too large, may need to revisit this decision and split
