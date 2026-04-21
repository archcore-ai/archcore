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
2. Each SKILL.md MUST contain frontmatter with `name` and `description`. Invocation flags follow the Inverted Invocation Policy (see `inverted-invocation-policy.adr.md`):
   - **Intent and track skills:** MUST NOT carry invocation-restricting flags. The model auto-invokes them from user phrasing.
   - **Mainstream type skills** (adr, prd, rfc, rule, guide, doc, spec, idea, task-type, cpat): MUST include `disable-model-invocation: true`. User-invokable via `/` only; descriptions not in model context.
   - **Niche type skills** (mrd, brd, urd, brs, strs, syrs, srs): MUST include `user-invocable: false`. Hidden from `/` menu; model reaches them via track orchestration.
   - **Utility skills** (`verify`): MUST include `disable-model-invocation: true`. Maintenance-only, user-invoked.
   - Track skill descriptions MUST be prefixed with "Advanced —". Non-high-frequency type skill descriptions MUST be prefixed with "Expert —".
3. Section structure varies by skill group:
   - **Intent skills (Layer 1):** Title+one-liner, When to Use, Routing Table, Execution, Result (5 sections).
   - **Track skills (Layer 2):** Sequential steps — Check existing → Scope → Create doc 1 → Create doc 2 → ... → Cross-relate → Result.
   - **Type skills (Layer 3):** When to Use, Quick Create, Relations (3 sections).
4. Creation flows (intent inline recipes, track steps, type Quick Create) MUST show `create_document` MCP tool usage — never Write/Edit.
5. Skills MUST NOT embed full document templates — reference the template system instead.
6. Line limits differ by group: Intent skills ≤ 300 lines, Track skills ≤ 200 lines, Type skills ≤ 100 lines.
7. Intent and track skill descriptions MUST explicitly enumerate trigger phrases and anti-triggers using the "Activate when X. Do NOT activate for Y (use /archcore:other)." format, so model routing is deterministic.

## Rationale

Consistent structure within each skill group ensures:

- Predictable content — developers know where to find each type of guidance based on the skill's layer
- Maintainability — skills within a group follow the same pattern, making batch updates feasible
- Quality — required sections per group prevent incomplete skills that miss key guidance
- No drift — referencing templates instead of embedding them prevents staleness when CLI templates change
- MCP-only enforcement — creation flows model the correct behavior
- Tier signaling — description prefixes help users identify skill complexity in the flat skill picker
- Routing correctness — under the Inverted Invocation Policy, intent skills are the sole auto-invocation entry point. Precise trigger/anti-trigger language in descriptions prevents the model from mis-routing into a neighboring intent.

## Examples

### Good — Intent Skill (Layer 1)

```markdown
---
name: capture
argument-hint: "[topic or description]"
description: "Document a module, component, or system — automatically picks the right type (ADR, spec, doc, or guide). Activate when user says 'document this module', 'capture how X works', 'write reference docs'. Do NOT activate for recording a decision (use /archcore:decide) or planning a feature (use /archcore:plan)."
---

# /archcore:capture

...

## When to Use
...
## Routing Table
| Signal | Route |
|---|---|
## Execution
...
## Result
...
```

No invocation-restricting flag — the model auto-invokes from user phrasing.

### Good — Track Skill (Layer 2)

```markdown
---
name: product-track
argument-hint: "[topic]"
description: "Advanced — Lightweight product requirements flow: idea → PRD → plan. Activate when user explicitly requests a full product cascade for a small-scope feature. For architectural design, use /archcore:architecture-track; for ISO cascade, use /archcore:iso-track."
---

# /archcore:product-track
## Step 1: Check existing...
## Step 2: Create idea...
## Step 3: Create PRD...
## Step 4: Create plan...
## Result
```

### Good — Mainstream Type Skill (Layer 3)

```markdown
---
name: adr
argument-hint: "[topic]"
description: Records architectural decisions with context, alternatives, and consequences. Activates for finalized technical decisions, technology choices, or trade-off discussions.
disable-model-invocation: true
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

User-invoked only via `/archcore:adr`; the model reaches ADR creation through `/archcore:decide` or `/archcore:capture` routing.

### Good — Niche Type Skill (Layer 3)

```markdown
---
name: brs
argument-hint: "[topic]"
description: "Expert — Formalizes business requirements into a traceable specification per ISO 29148..."
user-invocable: false
---

## When to Use
Use BRS within the ISO 29148 cascade (brs → strs → syrs → srs)...
```

Hidden from `/` autocomplete; the model invokes via `/archcore:iso-track` orchestration.

### Bad

```markdown
# Missing frontmatter name/description
# Intent/track skill with disable-model-invocation: true (inverted policy violation — blocks routing)
# Mainstream type skill without disable-model-invocation: true (pollutes model context)
# Niche type skill without user-invocable: false (noise in / menu)
# Template content embedded verbatim (will drift from CLI)
# Example uses Write instead of create_document
# Intent skill missing Routing Table section
# Type skill exceeds 100 lines
```

## Enforcement

- Code review during skill development
- Skills System Specification defines the normative contract
- Plugin Architecture Specification defines the 4-layer hierarchy
- Inverted Invocation Policy ADR defines per-layer flag requirements
- Future: automated lint script in `bin/` to check skill structure and invocation flags
