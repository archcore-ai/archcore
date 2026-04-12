---
title: "Keep Document-Type Skills as Domain Knowledge Layer"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Context

The Archcore Claude Plugin has two layers that interact with the `.archcore/` knowledge base:

1. **MCP tools (8)** — atomic CRUD operations: `create_document`, `update_document`, `list_documents`, `get_document`, `remove_document`, `add_relation`, `remove_relation`, `list_relations`
2. **Document-type skills (18)** — one SKILL.md per Archcore document type (adr, prd, spec, rule, guide, etc.)

During the skills redesign discussion, the question arose: are document-type skills redundant now that MCP tools exist? The MCP server already includes type descriptions and template generation in its instructions. Should skills be removed, leaving only MCP primitives?

### What MCP tools provide

- Document CRUD with validation, templates, and sync manifest updates
- Type selection hints embedded in MCP server instructions (~20 lines of guidance)
- Template generation when `content` parameter is omitted
- Relation management between documents

### What MCP tools do NOT provide

- **Disambiguation logic**: when to use ADR vs RFC vs Rule for a given context
- **Elicitation**: what questions to ask the user before creating each type
- **Content composition**: how to structure the document body from user answers, covering all required sections with appropriate depth
- **Relation guidance**: which relation types and targets are typical for each document type
- **Workflow context**: how this type fits into broader flows (e.g., ADR often leads to Rule + Guide)

## Decision

**Keep all 18 document-type skills as the domain knowledge layer over MCP primitives.**

The architecture is explicitly two-layered:

```
Document-type skills (18)  — domain knowledge: when, what, how for each type
         ↓
MCP tools (8)              — data operations: CRUD + relations over .archcore/
```

Skills are NOT wrappers around MCP calls. They are **teaching materials** that give Claude the domain expertise to use MCP tools effectively. Without skills, Claude can create documents (MCP works), but it lacks the judgment to create the _right_ document with the _right_ content at the _right_ time.

### Dual invocation model

Each document-type skill serves two use cases:

- **Model-invoked (automatic)**: Claude recognizes conversation context and activates the skill. User says "let's record the decision to use Redis" → Claude activates `adr` skill → follows its guidance → creates a well-structured ADR. The user doesn't need to know skills exist.
- **User-invoked (direct)**: `/archcore:adr use Redis` → Quick Create flow. Power users who know exactly what type they need.

Both modes are valuable. The first serves the majority of users. The second serves experts.

### What each skill provides (that MCP alone does not)

| Skill section   | Purpose                         | Example (ADR)                                                                                            |
| --------------- | ------------------------------- | -------------------------------------------------------------------------------------------------------- |
| When to Use     | Disambiguation vs similar types | "Decision finalized → adr. Still discussing → rfc. Mandatory standard → rule."                           |
| Quick Create    | Elicitation + composition flow  | "Ask: What was the decision? What alternatives? → Compose Context, Decision, Alternatives, Consequences" |
| Relations       | Type-specific relation patterns | "Incoming `implements` from plan. Outgoing `related` to rule."                                           |
| Best Practices  | Quality guidance                | "Record the decision, not the discussion. Keep alternatives concrete."                                   |
| Common Mistakes | Anti-patterns                   | "Don't mix decision recording with proposal (that's an RFC)."                                            |

## Alternatives Considered

### Remove skills, rely on MCP server instructions only

MCP server instructions include ~20 lines of type guidance. This is enough for Claude to pick a type and generate a template, but not enough to:

- Ask the right questions before creation
- Compose content that covers required sections with depth
- Suggest appropriate relations
- Distinguish between similar types in ambiguous contexts

Result: functional but shallow documents, poor type selection in edge cases.

### Merge skill content into MCP server instructions

Would make MCP server instructions ~1000+ lines. This bloats every MCP tool call context. Skills load on-demand — only the relevant skill's content enters context when activated. MCP instructions are always present.

### Replace skills with a single "smart create" skill

One skill that handles all 18 types. Would be 500+ lines and lose the specificity that makes each skill valuable. Claude works better with focused, type-specific guidance than with a monolithic decision tree.

## Consequences

### Positive

- Clear separation of concerns: MCP = data layer, Skills = knowledge layer
- On-demand context loading: only the relevant type's guidance enters Claude's context
- Dual invocation gives both automatic (model) and explicit (user) entry points
- Each skill is independently maintainable and testable
- New document types only require a new skill file — no changes to MCP layer

### Negative

- 18 skill files to maintain (though each is 50-80 lines)
- Skill content can drift from MCP template changes — mitigated by the rule that skills reference templates, not embed them
- Model invocation depends on description quality — poor descriptions lead to wrong skill activation

### Constraints

- Skills MUST NOT embed template content (templates live in MCP/CLI and can change between versions)
- Skills MUST reference MCP tools by exact name for document operations
- Skills MUST NOT instruct direct Write/Edit to `.archcore/` files
- Each skill MUST follow the 7-section structure defined in skills-system.spec
