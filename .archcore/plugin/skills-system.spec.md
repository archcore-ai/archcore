---
title: "Skills System Specification"
status: draft
tags:
  - "plugin"
  - "skills"
---

## Purpose

Define the contract for how skills are structured, discovered, and used within the Archcore Claude Plugin. Skills are the primary mechanism for teaching Claude about Archcore's document types and orchestrating documentation workflows.

## Scope

This specification covers all skill files in the `skills/` directory: 18 document-type skills, 6 track skills, and 3 workflow skills. It defines their naming convention, content structure, invocation triggers, and relationship to MCP tools. It does not cover the agent (subagent).

## Authority

This specification is the authoritative reference for all skill files in the plugin. The Skill File Structure Standard (rule) derives from this specification.

## Subject

The skills system consists of directories under `skills/`, each containing a `SKILL.md` file. Skills fall into three groups:

### Document-Type Skills (18)

Each teaches Claude about one Archcore document type. Model-invoked (Claude activates automatically) and user-invokable via `/archcore:<type> <topic>`.

| Directory           | Document Type                          | Category   |
| ------------------- | -------------------------------------- | ---------- |
| `skills/adr/`       | Architecture Decision Record           | knowledge  |
| `skills/rfc/`       | Request for Comments                   | knowledge  |
| `skills/rule/`      | Team Standard                          | knowledge  |
| `skills/guide/`     | How-To Instructions                    | knowledge  |
| `skills/doc/`       | Reference Material                     | knowledge  |
| `skills/spec/`      | Technical Specification                | knowledge  |
| `skills/prd/`       | Product Requirements                   | vision     |
| `skills/idea/`      | Product/Technical Concept              | vision     |
| `skills/plan/`      | Implementation Plan                    | vision     |
| `skills/mrd/`       | Market Requirements                    | vision     |
| `skills/brd/`       | Business Requirements                  | vision     |
| `skills/urd/`       | User Requirements                      | vision     |
| `skills/brs/`       | Business Requirements Specification    | vision     |
| `skills/strs/`      | Stakeholder Requirements Specification | vision     |
| `skills/syrs/`      | System Requirements Specification      | vision     |
| `skills/srs/`       | Software Requirements Specification    | vision     |
| `skills/task-type/` | Recurring Task Pattern                 | experience |
| `skills/cpat/`      | Code Pattern Change                    | experience |

### Track Skills (6)

Each orchestrates a complete multi-document flow, creating documents in sequence with proper relations. User-only (`disable-model-invocation: true`).

| Directory                    | Track                         | Flow                          |
| ---------------------------- | ----------------------------- | ----------------------------- |
| `skills/product-track/`      | Product Track (simple)        | idea → prd → plan             |
| `skills/sources-track/`      | Sources Track (discovery)     | mrd → brd → urd               |
| `skills/iso-track/`          | ISO 29148 Track (formal)      | brs → strs → syrs → srs      |
| `skills/architecture-track/` | Architecture Track (design)   | adr → spec → plan             |
| `skills/standard-track/`     | Standard Track (codify)       | adr → rule → guide            |
| `skills/feature-track/`      | Feature Track (lifecycle)     | prd → spec → plan → task-type |

Track skills do NOT duplicate document-type skill content. They define the flow — sequence of steps, relation chain, and scope detection (pick up where existing documents left off).

### Workflow Skills (3)

Each provides a utility workflow. User-only (`disable-model-invocation: true`).

| Directory         | Purpose                          |
| ----------------- | -------------------------------- |
| `skills/create/`  | Interactive creation wizard      |
| `skills/review/`  | Documentation health review      |
| `skills/status/`  | Documentation dashboard          |

## Contract Surface

### File Location

Each skill resides at `skills/<name>/SKILL.md` where `<name>` is the Archcore document type identifier (for type skills), track name (for track skills), or workflow name (for workflow skills).

### SKILL.md Frontmatter

**Document-type skills:**
```yaml
---
name: <type-name>
argument-hint: "[topic]"
description: <When Claude should activate this skill — specific triggers and context>
---
```

**Track and workflow skills:**
```yaml
---
name: <skill-name>
argument-hint: "[topic]"
description: <What this workflow does>
disable-model-invocation: true
---
```

Track and workflow skills use `disable-model-invocation: true` because they are explicit workflows the user initiates via `/archcore:<name>`, not guidance that Claude should auto-activate.

### Document-Type Skill Content Structure

Every document-type skill file MUST contain these sections in order:

1. **Overview** — What this document type is, its purpose in the Archcore knowledge base, and which virtual category it belongs to (vision/knowledge/experience).

2. **When to Use** — Specific scenarios, conversation cues, and context signals that indicate this type should be used. Include contrast with similar types (e.g., ADR vs RFC, PRD vs MRD).

3. **Required Sections** — List the sections the document template includes, with a brief description of what goes in each. Reference the template system — do NOT embed template content verbatim.

4. **Best Practices** — Domain-specific guidance for writing high-quality documents of this type. Include what makes a good vs. poor example.

5. **Common Mistakes** — Pitfalls to avoid (e.g., mixing decision recording with proposal, creating a PRD when an idea would suffice).

6. **Relation Guidance** — Which relation types and target document types are commonly used with this type. Include typical flows (e.g., "An ADR often has incoming `implements` from a plan and outgoing `related` to rules").

7. **Example Workflow** — A concrete example showing the MCP tool calls to create a document of this type, including the `create_document` call with appropriate parameters and a follow-up `add_relation` call.

### Track Skill Content Structure

Every track skill file MUST contain:

1. **Title and summary** — Track name, flow diagram, when to use this track.
2. **Step 1: Check existing** — `list_documents` call to detect existing documents and prevent duplicates.
3. **Step 2: Determine scope** — Logic for picking up where existing documents left off.
4. **Steps 3-N: One step per document** — For each document in the flow: focused questions, content sections to compose, `create_document` call, `add_relation` calls.
5. **Final step: Relate to existing** — Suggest links to other documents outside the track.
6. **Result** — Summary of what was created and the relation chain.

## Normative Behavior

- Document-type skills are model-invoked: Claude activates them automatically based on conversation context matching the `description` field.
- Track and workflow skills are user-only: invoked via `/archcore:<name>` slash command.
- All skills MUST instruct the agent to use `create_document` MCP tool for document creation. They MUST NOT instruct direct file writes via Write/Edit.
- Skills MUST reference MCP tools by exact name: `create_document`, `update_document`, `list_documents`, `get_document`, `add_relation`, `remove_relation`, `list_relations`, `remove_document`.
- Skills provide guidance around the template, not the template itself. The `create_document` MCP tool generates the template when `content` is omitted.
- When multiple document-type skills could apply, Claude should prefer the most specific type. Skills should include enough contrast in "When to Use" to help Claude disambiguate.
- Track skills create documents sequentially, asking focused questions before each creation step. They do not batch-create all documents at once.

## Constraints

- Skill files must not exceed 500 lines to keep context injection manageable.
- Skill files must not include code blocks longer than 20 lines.
- Skills must not reference internal CLI implementation details — only the MCP tool interface.
- Skills must not embed full document templates (these change with CLI versions and would drift).
- Track skills must not duplicate content from document-type skills — they define the flow, not the type guidance.

## Invariants

- There is exactly one skill per Archcore document type (18 total).
- There is exactly one skill per track (6 total).
- Every document-type skill follows the same 7-section structure.
- Every track skill follows the sequential step structure.
- Every skill references `create_document` in its workflow.
- No skill instructs direct Write/Edit to `.archcore/` files.

## Error Handling

- If a skill references an MCP tool that is not available (e.g., MCP server not running), Claude should inform the user that the Archcore MCP server needs to be active and suggest checking the plugin installation.
- If `create_document` fails (e.g., duplicate filename), the skill guidance should help Claude recover by suggesting an alternative filename.
- If a track skill detects existing documents mid-flow, it should skip already-created documents and resume from the next step.

## Conformance

A skill file conforms to this specification if:

1. It resides at the correct path (`skills/<name>/SKILL.md`)
2. It has valid frontmatter with `name` and `description` fields
3. Document-type skills contain all 7 required sections in order
4. Track skills contain the sequential step structure
5. It references `create_document` MCP tool (not Write/Edit) in its workflow
6. It stays within the 500-line limit
7. It does not embed full template content