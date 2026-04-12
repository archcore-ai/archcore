---
title: "Skill and MCP Tool Interaction Pattern"
status: accepted
tags:
  - "plugin"
  - "rule"
  - "skills"
---

## Rule

1. Skills ENHANCE the quality of content passed to MCP tools — they do not replace or duplicate MCP behavior.
2. When creating a document, all creation flows (intent inline recipes, track steps, type Quick Create) MUST gather context from the user, compose rich content, and pass it as the `content` parameter to `create_document`. Do not omit `content` to get the template — the skill's job is to produce better content than a template.
3. The composed content MUST follow the required sections for the document type (as defined by MCP tool description). Skills and MCP agree on structure — skills add depth and quality.
4. Skills MUST NOT re-explain what MCP already enforces (validation, slug format, frontmatter). Skills focus on what MCP cannot do: asking the right questions, guiding type selection, producing contextually rich content, and routing intent to the correct types/tracks.
5. Skills MUST NOT conflict with MCP tool descriptions. If a skill's guidance contradicts an MCP tool parameter description, the skill is wrong and must be fixed.

## Rationale

The plugin has two instruction layers that the agent sees simultaneously:
- **MCP tool descriptions** — define parameters, validation, and structural rules for document operations
- **Skills** — provide domain expertise, type selection guidance, intent routing, and content quality enhancement

Without clear separation, these layers compete for the agent's attention. The agent may follow MCP's "RECOMMENDED: omit content" instead of the skill's guidance to compose content from user input. Or the skill may repeat structural rules already in MCP, wasting context tokens.

The resolution: skills own QUALITY (asking the right questions, composing rich content, routing intent), MCP owns STRUCTURE (validation, templates, frontmatter, manifest). Skills always pass content to MCP — the composed content is the skill's value-add.

## Examples

### Good

Skill asks focused questions, composes content, passes to MCP:

```
Intent skill activates for /archcore:decide →
  Routing table identifies: single ADR →
  Asks: "What was the decision? What alternatives?"
  User answers with context →
  Agent composes rich content with all required sections →
  Calls create_document(type="adr", content="## Context\n...", ...)
  MCP validates and saves
```

### Good

Track skill creates sequentially with content:

```
Track skill activates for /archcore:product-track →
  Step 2: Ask idea questions → create_document(type="idea", content="...") →
  Step 3: Ask PRD questions → create_document(type="prd", content="...") →
  Step 4: Ask plan questions → create_document(type="plan", content="...") →
  add_relation for each step in the chain
```

### Bad

Skill omits content, relies on template:

```
Skill activates for ADR →
  Asks questions →
  Calls create_document(type="adr") WITHOUT content →
  MCP generates template with placeholders →
  Agent then calls update_document to fill in
```

Result: two MCP calls, template placeholders may leak, agent may lose context between calls.

### Bad

Skill duplicates MCP's structural rules:

```
Skill says: "filename must be lowercase with hyphens"
Skill says: "status must be draft, accepted, or rejected"
```

This is already in MCP tool description. Wasted tokens.

## Enforcement

- Skill review: every creation flow (intent inline recipe, track step, type Quick Create) must show `content` parameter being passed
- Skills must not contain parameter validation rules already in MCP tool descriptions
- The skill-file-structure rule references this rule for content guidance