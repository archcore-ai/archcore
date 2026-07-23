---
title: "Skill File Structure Standard"
status: accepted
tags:
  - "plugin"
  - "rule"
  - "skills"
---

## Rule

1. Each skill MUST live at `skills/<name>/SKILL.md` where `<name>` is one of the 7 canonical skill names: `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`.
2. Each SKILL.md MUST contain frontmatter with `name` and `description`. Per `skill-surface-collapse.adr.md`:
   - All skills are auto-invocable. No skill carries `disable-model-invocation`.
   - Skill descriptions MUST enumerate trigger phrases and anti-triggers using the "Activate when X. Do NOT activate for Y (use /archcore:other)." format, so model routing is deterministic.
3. Section structure: Title+one-liner, When to Use, Routing Table (or numbered step sequence with deterministic branches), Execution, Result (5 sections). Creation-oriented skills inline per-type elicitation (question + sections + create_document + add_relation) within the Execution section. Flow-style skills (e.g., `init`, `plan`) may load per-flow references from `skills/<name>/references/<flow>.md` or `skills/<name>/lib/<mode>.md` on demand.
4. Creation flows MUST show `create_document` MCP tool usage — never Write/Edit.
5. Skills MUST NOT embed full document templates — reference the template system (MCP server templates) instead.
6. Line limits: SKILL.md ≤ 300 lines; per-flow reference files (under `references/` or `lib/`) ≤ 200 lines each.
7. The plan skill MUST hold per-flow logic in `skills/plan/references/<flow>.md` rather than spawning new top-level skills. Adding a new flow is a new reference file, not a new skill.

## Rationale

Consistent structure ensures:

- Predictable content — developers know where to find each type of guidance based on the skill's role.
- Maintainability — skills follow the same pattern, making batch updates feasible.
- Quality — required sections prevent incomplete skills that miss key guidance.
- No drift — referencing templates instead of embedding them prevents staleness when CLI templates change.
- MCP-only enforcement — creation flows model the correct behavior.
- Routing correctness — every skill is auto-invocable; precise trigger/anti-trigger language in descriptions prevents the model from mis-routing into a neighboring intent.
- Single home for per-type elicitation — creation skills inline the per-type recipes; there is no separate per-type skill layer (removed by `remove-document-type-skills.adr.md`).
- Per-flow reuse without proliferation — per-flow references absorb what would otherwise become new top-level skills (track tier removed by `skill-surface-collapse.adr.md`).

## Examples

### Good — Intent Skill

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

No invocation-restricting flag — every skill auto-invokes from user phrasing.

### Good — Flow-Style Skill (plan)

```markdown
---
name: plan
argument-hint: "[topic] [--product|--sources|--iso|--feature]"
description: "Plan a feature or initiative end-to-end. Activate when user says 'let's plan', 'create a roadmap for X', 'I need to plan Y'. Do NOT activate for recording a decision (use /archcore:decide) or documenting an existing module (use /archcore:capture)."
---

# /archcore:plan

## Routing Table
| Flag / signal | Reference loaded |
|---|---|
| `--product` or product-flow phrasing | skills/plan/references/product-flow.md |
| `--sources` or sources phrasing | skills/plan/references/sources-flow.md |
| `--iso` or ISO-cascade phrasing | skills/plan/references/iso-flow.md |
| `--feature` or feature phrasing | skills/plan/references/feature-flow.md |
| (none) | inline single-plan recipe |

## Execution
- Step 1: Check existing documents via list_documents
- Step 2: Scope confirmation (one AskUserQuestion if ambiguous)
- Step 3: Load the matching reference and run its step sequence
- Step 4: Cross-relate to existing documents
```

Per-flow content lives in references, keeping the SKILL.md under the 300-line budget.

### Bad

```markdown
# Missing frontmatter name/description
# Skill with disable-model-invocation: true (every skill must auto-invoke)
# Template content embedded verbatim (will drift from CLI)
# Example uses Write instead of create_document
# Skill missing Routing Table section (and not a flow-style skill with numbered steps)
# SKILL.md exceeds 300 lines
# Per-flow reference file exceeds 200 lines
# New top-level skill added for a new flow (should be a reference under skills/plan/references/)
```

## Enforcement

- Code review during skill development.
- Skills System Specification defines the normative contract.
- Plugin Architecture Specification defines the cross-component invariants.
- `skill-surface-collapse.adr.md` defines the 7-skill surface and auto-invocation invariant.
- Future: automated lint script in `bin/` to check skill structure.
