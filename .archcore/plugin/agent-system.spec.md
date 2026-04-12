---
title: "Universal Agent Specification"
status: draft
tags:
  - "agents"
  - "plugin"
---

## Purpose

Define the contract for the Archcore Claude Plugin's subagents — `archcore-assistant` (read/write) and `archcore-auditor` (read-only).

## Scope

This specification covers both agent definitions in `agents/`, their system prompts, tool restrictions, invocation triggers, and domain expertise.

## Authority

This specification is the authoritative reference for both agents. The Single Universal Agent Design ADR provides the original rationale; the Add Read-Only Auditor Agent ADR extends it.

## Subject

### Agent 1: archcore-assistant (Read/Write)

Handles complex, multi-step documentation tasks requiring write access to MCP tools.

#### Definition File

Location: `agents/archcore-assistant.md`

```yaml
---
name: archcore-assistant
description: >
  Archcore documentation expert. Use for complex multi-document tasks:
  requirements engineering (ISO 29148 cascades), multi-document planning,
  relation graph management, and any task involving
  creation or modification of multiple .archcore/ documents.
model: sonnet
maxTurns: 20
color: blue
tools:
  - mcp__archcore__list_documents
  - mcp__archcore__get_document
  - mcp__archcore__create_document
  - mcp__archcore__update_document
  - mcp__archcore__remove_document
  - mcp__archcore__add_relation
  - mcp__archcore__remove_relation
  - mcp__archcore__list_relations
  - Read
  - Grep
  - Glob
---
```

#### Invocation Triggers

Claude should invoke `archcore-assistant` when:

- User requests creation of multiple related documents
- Task involves requirements decomposition (e.g., "break this PRD into specifications")
- Complex refactoring of existing documentation structure
- Task requires understanding the full relation graph to make decisions

### Agent 2: archcore-auditor (Read-Only)

Performs documentation health checks without any mutation capability.

#### Definition File

Location: `agents/archcore-auditor.md`

```yaml
---
name: archcore-auditor
description: >
  Read-only documentation auditor. Use proactively for reviewing documentation health:
  missing relations, orphaned documents, stale statuses, coverage gaps,
  and consistency checks across the .archcore/ knowledge base.
model: sonnet
maxTurns: 15
color: yellow
background: true
tools:
  - mcp__archcore__list_documents
  - mcp__archcore__get_document
  - mcp__archcore__list_relations
  - Read
  - Grep
  - Glob
---
```

#### Invocation Triggers

Claude should invoke `archcore-auditor` when:

- User asks for a documentation audit, health check, or review
- User asks "what's missing?" or "what needs attention?" about documentation
- Proactively after a batch of documents has been created (quality check)
- User wants to verify documentation coverage before a release or milestone
- User wants to check if documentation matches the current code

#### Background Execution

The auditor runs with `background: true` by default. This means:

- User can continue working while the audit runs
- Results are delivered when complete, not blocking the conversation
- Ideal for large knowledge bases where audit takes multiple turns

## Contract Surface

### Shared Domain Knowledge

Both agents MUST understand:

#### 1. Document Type Expertise

All 18 types across 3 categories:

- Knowledge: adr, rfc, rule, guide, doc, spec
- Vision: prd, idea, plan, mrd, brd, urd, brs, strs, syrs, srs
- Experience: task-type, cpat

For each type: purpose, when to use, required sections, differentiation from similar types.

#### 2. Requirements Engineering Patterns

Three tracks that can coexist:

**Product Track (simple):** idea → prd → plan

**Sources Track (discovery):** mrd + brd + urd → prd

**ISO 29148 Track (decomposition):** brs → strs → syrs → srs

#### 3. Relation Types

- `implements` — source fulfills target (plan implements prd)
- `extends` — source builds upon target (rfc extends adr)
- `depends_on` — source requires target (plan depends_on adr)
- `related` — general association

### Tool Access Matrix

| Tool | assistant | auditor |
|------|-----------|---------
| list_documents | Yes | Yes |
| get_document | Yes | Yes |
| create_document | Yes | No |
| update_document | Yes | No |
| remove_document | Yes | No |
| add_relation | Yes | No |
| remove_relation | Yes | No |
| list_relations | Yes | Yes |
| Read | Yes | Yes |
| Grep | Yes | Yes |
| Glob | Yes | Yes |
| Write/Edit/Bash | No | No |

### Output Contracts

**archcore-assistant** outputs: created/updated documents, relation changes, and explanations of choices.

**archcore-auditor** outputs: structured audit report with sections:
- Audit Summary (counts, issue totals)
- Critical Issues (broken references, misleading content)
- Warnings (quality gaps)
- Code-Document Correlation (documents referencing source paths where code has changed since the document was last modified)
- Info (suggestions)
- Recommendations (prioritized actions)

## Normative Behavior

### Both Agents

- MUST use MCP tools for all `.archcore/` operations (no Write/Edit/Bash).
- MUST call `list_documents` before creating any document to prevent duplicates.
- Should explain reasoning when choosing document types or relation types.

### archcore-assistant Only

- MUST create relations between documents it creates when semantic links exist.
- Should present a plan before creating multiple documents, letting the user approve.
- MUST NOT create more than 10 documents in a single invocation without user confirmation.

### archcore-auditor Only

- MUST NOT attempt to create, update, or delete any document.
- MUST produce a structured audit report (not free-form commentary).
- Should cross-reference documentation with actual code via Read/Grep/Glob.
- Should use `Grep` to find path references in document content, then check via `git log` if those paths changed since the document was last modified.
- Should prioritize specs, ADRs, and guides that describe specific code modules for code-document correlation checks.

## Constraints

- System prompts must not exceed 2000 lines.
- Neither agent may modify files outside `.archcore/` via any means.
- Both agents respect existing document statuses.

## Invariants

- `archcore-assistant` never uses Write/Edit/Bash on `.archcore/` files.
- `archcore-auditor` has zero write tools — enforcement by tool whitelist.
- Both agents check for existing documents before suggesting creation.

## Error Handling

- If MCP server is unavailable, inform the user and exit gracefully.
- If a document operation fails, report the error and continue with remaining tasks.
- If a relation target doesn't exist, skip the relation and note it for the user.

## Conformance

An agent conforms to this specification if:

1. It resides at `agents/<name>.md` with the correct frontmatter
2. Its tool list matches the allowed tools exactly (per tool access matrix)
3. Its system prompt covers the shared domain knowledge
4. It follows the normative behavior for its role
5. archcore-auditor produces no mutations; archcore-assistant produces structured output