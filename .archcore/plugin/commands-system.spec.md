---
title: "Skills System — User-Invoked Workflows Specification"
status: draft
tags:
  - "commands"
  - "plugin"
---

## Purpose

Define the contract for user-invoked workflow skills the Archcore Claude Plugin exposes. These are skills with `disable-model-invocation: true` that users trigger explicitly via slash commands.

Note: Claude Code has merged commands into skills. The `commands/` directory is legacy. All user-invoked workflows use `skills/<name>/SKILL.md` with `disable-model-invocation: true`.

## Scope

This specification covers the 3 workflow skills (create, review, status) and the 18 type skills that also serve as quick-create shortcuts when invoked directly. It does not cover model-invoked behavior (covered by the Skills System Specification).

## Authority

This specification is the authoritative reference for user-invoked skill behavior in the plugin.

## Subject

The user-invoked workflow system consists of:

1. **Workflow skills** — `skills/create/`, `skills/review/`, `skills/status/` with `disable-model-invocation: true`
2. **Type skills with Quick Create** — all 18 type skills in `skills/<type>/` support both model invocation (knowledge) and user invocation (`/archcore:<type> <topic>` for quick creation)

### Workflow Skills

| Skill | Command | Purpose |
|-------|---------|---------|
| `skills/create/SKILL.md` | `/archcore:create` | Interactive document creation wizard |
| `skills/review/SKILL.md` | `/archcore:review` | Documentation health review |
| `skills/status/SKILL.md` | `/archcore:status` | Documentation dashboard |

### Type Quick Create

All 18 type skills support `/archcore:<type> <topic>` for quick creation:
- `/archcore:adr use PostgreSQL` — creates an ADR
- `/archcore:prd notification system` — creates a PRD
- `/archcore:rule API error format` — creates a rule
- etc.

## Contract Surface

### Workflow Skill File Format

```yaml
---
name: <skill-name>
description: <What this workflow does>
disable-model-invocation: true
---

<Workflow instructions>
```

### Type Skill Quick Create Section

Each type skill includes a "Quick Create" section at the top:

```markdown
## Quick Create

When invoked directly with `/archcore:<type> <topic>`, create immediately:
1. Call `list_documents(types=[...])` to check for duplicates
2. Ask 1-2 type-specific questions
3. Call `create_document(type=..., ...)` with gathered content
4. Suggest `add_relation` calls based on existing documents
```

This section is activated when the user invokes the skill directly. When Claude activates the skill via model invocation, it uses the knowledge sections instead.

## Normative Behavior

- All workflow skills MUST use MCP tools for document operations. No direct Write/Edit.
- Workflow skills MUST call `list_documents` before `create_document` to prevent duplicates.
- The create workflow should suggest document type based on user intent.
- The review workflow should produce actionable findings, not verbose analysis.
- The status workflow should produce compact, scannable output.
- Type quick-create should ask at most 2-3 clarifying questions before creating.
- All creation workflows should suggest `add_relation` calls after document creation.

## Constraints

- Workflow skill files must not exceed 200 lines.
- Type skill Quick Create sections must not exceed 10 lines.

## Invariants

- Every workflow skill has `disable-model-invocation: true`.
- Every type skill has a Quick Create section.
- Every creation workflow checks for duplicates first and suggests relations after.

## Error Handling

- If MCP server is not running, inform user and suggest checking plugin installation.
- If `create_document` fails due to duplicate filename, suggest an alternative slug.
- If `list_documents` returns empty, inform user that `archcore init` may be needed.

## Conformance

A workflow skill conforms to this specification if:
1. It resides at `skills/<name>/SKILL.md`
2. It has `disable-model-invocation: true` in frontmatter
3. It uses MCP tools exclusively for document operations
4. It checks for duplicates before creation
5. It suggests relations after creation