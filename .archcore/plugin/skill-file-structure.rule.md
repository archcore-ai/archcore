---
title: "Skill File Structure Standard"
status: accepted
tags:
  - "plugin"
  - "rule"
  - "skills"
---

## Rule

1. Each skill MUST live at `skills/<name>/SKILL.md` where `<name>` is the intent name (for Layer 1), track name (for Layer 2), or Archcore document type identifier (for Layer 3).
2. Each SKILL.md MUST contain frontmatter with `name` and `description`. Intent and track skills MUST include `disable-model-invocation: true`. Track skill descriptions MUST be prefixed with "Advanced —". Non-high-frequency type skill descriptions MUST be prefixed with "Expert —".
3. Section structure varies by skill group:
   - **Intent skills (Layer 1):** Title+one-liner, When to Use, Routing Table, Execution, Result (5 sections).
   - **Track skills (Layer 2):** Sequential steps — Check existing → Scope → Create doc 1 → Create doc 2 → ... → Cross-relate → Result.
   - **Type skills (Layer 3):** When to Use, Quick Create, Relations (3 sections).
4. Creation flows (intent inline recipes, track steps, type Quick Create) MUST show `create_document` MCP tool usage — never Write/Edit.
5. Skills MUST NOT embed full document templates — reference the template system instead.
6. Line limits differ by group: Intent skills ≤ 300 lines, Track skills ≤ 200 lines, Type skills ≤ 100 lines.

## Rationale

Consistent structure within each skill group ensures:

- Predictable content — developers know where to find each type of guidance based on the skill's layer
- Maintainability — skills within a group follow the same pattern, making batch updates feasible
- Quality — required sections per group prevent incomplete skills that miss key guidance
- No drift — referencing templates instead of embedding them prevents staleness when CLI templates change
- MCP-only enforcement — creation flows model the correct behavior
- Tier signaling — description prefixes help users identify skill complexity in the flat skill picker

## Examples

### Good — Intent Skill (Layer 1)

```markdown
---
name: capture
argument-hint: "[topic or description]"
description: Capture documentation for a module, component, or topic.
disable-model-invocation: true
---

# /archcore:capture

...

## When to Use
...
## Routing Table
| Signal | Route |
|---|---|
## Execution
Step 0: Verify MCP...
## Result
...
```

### Good — Track Skill (Layer 2)

```markdown
---
name: product-track
argument-hint: "[topic]"
description: "Advanced — Create idea, PRD, and plan with full traceability."
disable-model-invocation: true
---

# /archcore:product-track
## Step 0: Verify MCP...
## Step 1: Check existing...
## Step 2: Create idea...
## Step 3: Create PRD...
## Step 4: Create plan...
## Result
```

### Good — Type Skill (Layer 3)

```markdown
---
name: adr
argument-hint: "[topic]"
description: >
  Record an architectural decision with context and alternatives.
---

## When to Use
Use ADR when a technical decision has been made...

## Quick Create
create_document(type="adr", content="## Context\n...", ...)
add_relation(source=..., target=..., type="implements")

## Relations
- Incoming: `implements` from plan or spec...
- Outgoing: `related` to rule...
```

### Bad

```markdown
# Missing frontmatter name/description
# Missing disable-model-invocation for intent/track skills
# Template content embedded verbatim (will drift from CLI)
# Example uses Write instead of create_document
# Intent skill missing Routing Table section
# Type skill exceeds 100 lines
```

## Enforcement

- Code review during skill development
- Skills System Specification defines the normative contract
- Plugin Architecture Specification defines the 4-layer hierarchy
- Future: automated lint script in `bin/` to check skill structure