---
title: "Skill File Structure Standard"
status: accepted
tags:
  - "plugin"
  - "rule"
  - "skills"
---

## Rule

1. Each skill MUST live at `skills/<type-name>/SKILL.md` where `<type-name>` matches the Archcore document type identifier.
2. Each SKILL.md MUST contain frontmatter with `name` (prefixed `archcore-`) and `description` (trigger conditions).
3. Each SKILL.md MUST contain exactly 7 sections in this order: Overview, When to Use, Required Sections, Best Practices, Common Mistakes, Relation Guidance, Example Workflow.
4. The Example Workflow section MUST show `create_document` MCP tool usage — never Write/Edit.
5. Skills MUST NOT embed full document templates — reference the template system instead.
6. Skills MUST NOT exceed 500 lines.

## Rationale

Consistent structure across 18 skill files ensures:

- Predictable content — developers know where to find each type of guidance
- Maintainability — all skills follow the same pattern, making batch updates feasible
- Quality — required sections prevent incomplete skills that miss key guidance
- No drift — referencing templates instead of embedding them prevents staleness when CLI templates change
- MCP-only enforcement — Example Workflow sections model the correct behavior

## Examples

### Good

```markdown
---
name: archcore-adr
description: >
  Use when the user discusses architectural decisions, technology choices,
  or trade-offs that need to be recorded. Activate when context includes
  phrases like "we decided", "the alternative was", or "trade-off".
---

## Overview

An Architecture Decision Record (ADR) captures a decision...

## When to Use

Use ADR when a technical decision has been made...
Unlike RFC (proposal stage), ADR records a final decision...

## Required Sections

The ADR template includes: Context, Decision, Alternatives Considered, Consequences...

## Best Practices

- State the decision clearly in one sentence...

## Common Mistakes

- Writing an ADR for a decision still under discussion (use RFC instead)...

## Relation Guidance

- Incoming: `implements` from plan or spec documents...
- Outgoing: `related` to rule documents that codify the decision...

## Example Workflow

create_document(type="adr", filename="use-postgres", title="Use PostgreSQL for Primary Persistence", tags=["database"])
add_relation(source="plugin/migration-rules.rule.md", target="plugin/use-postgres.adr.md", type="implements")
```

### Bad

```markdown
# Missing frontmatter name/description

# Missing "When to Use" section

# Template content embedded verbatim (will drift from CLI)

# Example uses Write instead of create_document

## Overview

...

## Template

---

title: ...
status: draft

---

## Context

...

## Example

Write(".archcore/use-postgres.adr.md", "---\ntitle:...")
```

## Enforcement

- Code review during skill development
- Skills System Specification defines the normative contract
- Future: automated lint script in `bin/` to check skill structure
