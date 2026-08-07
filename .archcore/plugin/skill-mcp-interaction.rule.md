---
title: "Skill and MCP Tool Interaction Pattern"
status: accepted
tags:
  - "plugin"
  - "rule"
  - "skills"
---

## Rule

1. A skill MUST improve the quality of the content that it passes to an MCP tool.
2. A skill MUST NOT duplicate behavior that an MCP tool already performs.
3. WHEN a creation flow prepares a document, the flow MUST gather context from the user before it composes content.
4. WHEN a creation flow calls `create_document`, the flow MUST pass the composed content in the `content` parameter.
5. A creation flow MUST NOT omit `content` in order to receive the generated template.
6. A skill MUST compose content that follows the sections the MCP tool description requires for the document type.
7. A skill MUST NOT restate a validation rule that an MCP tool description already states.
8. A skill MUST NOT restate the slug format that an MCP tool description already states.
9. A skill MUST NOT restate a frontmatter rule that an MCP tool description already states.
10. A skill MUST NOT contradict an MCP tool description.
11. IF a skill's guidance contradicts an MCP tool parameter description, THEN the author MUST correct the skill.

## Rationale

The agent sees two instruction layers at the same time. MCP tool descriptions define parameters, validation, and structural rules for document operations. Skills provide type selection, intent routing, and content composition. Without the division above the layers compete: the agent may follow the MCP description's `RECOMMENDED: omit content` instead of the skill's instruction to compose content from user input, or a skill may repeat structural rules that MCP already states and spend context tokens on them. The division is fixed — skills own quality, MCP owns structure — and the composed content is the skill's contribution.

## Examples

### Good

An intent skill asks focused questions, composes content, and passes it to MCP:

```
Intent skill activates for /archcore:document →
  Routing table identifies: single ADR →
  Asks: "What was the decision? What alternatives?"
  User answers with context →
  Agent composes content with all required sections →
  Calls create_document(type="adr", content="## Context\n...", ...)
  MCP validates and saves
```

### Good

A flow-style skill creates documents in sequence, each with content (`plan` running the product flow):

```
Skill activates for /archcore:plan (gated track flow per `track-layer.spec`) →
  Loads skills/plan/references/product-flow.md →
  Step 1: Ask idea questions → create_document(type="idea", content="...") →
  Step 2: Ask PRD questions → create_document(type="prd", content="...") →
  Step 3: Ask plan questions → create_document(type="plan", content="...") →
  add_relation for each step in the chain
```

### Bad

A skill omits `content` and relies on the template — violates item 5:

```
Skill activates for ADR →
  Asks questions →
  Calls create_document(type="adr") WITHOUT content →
  MCP generates template with placeholders →
  Agent then calls update_document to fill it in
```

The result is two MCP calls, template placeholders that may reach the file, and context lost between the calls.

### Bad

A skill restates structural rules that MCP already states — violates items 8 and 9:

```
Skill says: "filename must be lowercase with hyphens"
Skill says: "status must be draft, accepted, or rejected"
```

Both statements are already in the MCP tool description, so they consume context tokens without adding guidance.

## Enforcement

- Skill review: each creation flow shows the `content` parameter being passed to `create_document`.
- Skill review: no skill carries a parameter validation rule that an MCP tool description already states.
