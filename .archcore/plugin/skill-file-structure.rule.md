---
title: "Skill File Structure Standard"
status: accepted
tags:
  - "plugin"
  - "rule"
  - "skills"
---

## Rule

1. Each skill MUST live at `skills/<name>/SKILL.md` where `<name>` is the intent name (for Layer 1), the track name (for Layer 2), or the utility name.
2. Each SKILL.md MUST contain frontmatter with `name` and `description`. Invocation flags follow the Inverted Invocation Policy (see `inverted-invocation-policy.adr.md`), as amended by `remove-document-type-skills.adr.md`:
   - **Intent and track skills:** MUST NOT carry invocation-restricting flags. The model auto-invokes them from user phrasing.
   - **Utility skills** (`verify`): MUST include `disable-model-invocation: true`. Maintenance-only, user-invoked.
   - Track skill descriptions MUST be prefixed with "Advanced —". Intent and utility descriptions use clean (unprefixed) text.
3. Section structure varies by skill group:
   - **Intent skills (Layer 1):** Title+one-liner, When to Use, Routing Table, Execution, Result (5 sections). Creation-oriented intents inline per-type elicitation (question + sections + create_document + add_relation) within the Execution section.
   - **Track skills (Layer 2):** Sequential steps — Check existing → Scope → Create doc 1 → Create doc 2 → ... → Cross-relate → Result. Each creation step inlines per-type elicitation.
4. Creation flows (intent inline recipes, track steps) MUST show `create_document` MCP tool usage — never Write/Edit.
5. Skills MUST NOT embed full document templates — reference the template system (MCP server templates) instead.
6. Line limits differ by group: Intent skills ≤ 300 lines, Track skills ≤ 200 lines.
7. Intent and track skill descriptions MUST explicitly enumerate trigger phrases and anti-triggers using the "Activate when X. Do NOT activate for Y (use /archcore:other)." format, so model routing is deterministic.

## Rationale

Consistent structure within each skill group ensures:

- Predictable content — developers know where to find each type of guidance based on the skill's layer
- Maintainability — skills within a group follow the same pattern, making batch updates feasible
- Quality — required sections per group prevent incomplete skills that miss key guidance
- No drift — referencing templates instead of embedding them prevents staleness when CLI templates change
- MCP-only enforcement — creation flows model the correct behavior
- Tier signaling — description prefixes help users identify skill complexity in the flat skill picker
- Routing correctness — under the Inverted Invocation Policy, intent skills are the primary auto-invocation entry point. Precise trigger/anti-trigger language in descriptions prevents the model from mis-routing into a neighboring intent.
- Single home for per-type elicitation — intent and track skills inline the per-type recipes; there is no separate per-type skill layer.

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
Step 3 (per-type creation inlines: ask question → compose sections → create_document → add_relation)
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
## Step 2: Determine scope...
## Step 3: Create idea (question + sections + create_document + add_relation)
## Step 4: Create PRD (same pattern)
## Step 5: Create plan (same pattern)
## Step 6: Relate to existing
## Result
```

Each creation step inlines per-type elicitation.

### Good — Utility Skill

```markdown
---
name: verify
argument-hint: "[options]"
description: "Run plugin integrity checks — tests, lint, config validation, cross-references."
disable-model-invocation: true
---
```

User-only; `disable-model-invocation: true` ensures the model does not auto-run maintenance checks.

### Bad

```markdown
# Missing frontmatter name/description
# Intent/track skill with disable-model-invocation: true (inverted policy violation — blocks routing)
# Template content embedded verbatim (will drift from CLI)
# Example uses Write instead of create_document
# Intent skill missing Routing Table section
# Intent skill exceeds 300 lines
# Track skill exceeds 200 lines
```

## Enforcement

- Code review during skill development
- Skills System Specification defines the normative contract
- Plugin Architecture Specification defines the layer hierarchy
- Inverted Invocation Policy ADR (as amended by `remove-document-type-skills.adr.md`) defines per-layer flag requirements
- Future: automated lint script in `bin/` to check skill structure and invocation flags
