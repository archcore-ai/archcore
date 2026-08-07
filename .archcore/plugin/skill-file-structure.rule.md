---
title: "Skill File Structure Standard"
status: accepted
tags:
  - "plugin"
  - "rule"
  - "skills"
---

## Rule

1. Each skill MUST live at `skills/<name>/SKILL.md`.
2. `<name>` MUST be one of the four canonical command names: `init`, `plan`, `document`, `review`, per `four-command-palette.adr`.
3. Each `SKILL.md` MUST carry a `name` field in its frontmatter.
4. Each `SKILL.md` MUST carry a `description` field in its frontmatter.
5. A `SKILL.md` MUST NOT carry `disable-model-invocation`.
6. Each `description` MUST enumerate its trigger phrases in the form `Activate when X.`
7. Each `description` MUST enumerate its anti-triggers in the form `Do NOT activate for Y (use /archcore:other).`
8. Each `SKILL.md` MUST contain these five sections: a title with a one-line summary, `When to Use`, a routing table or a numbered step sequence with deterministic branches, `Execution`, and `Result`.
9. A creation-oriented skill MUST inline its per-type elicitation — question, sections, `create_document`, `add_relation` — inside the `Execution` section.
10. A flow-style skill MAY load a per-flow reference from `skills/<name>/references/<flow>.md` or `skills/<name>/lib/<mode>.md` on demand.
11. A creation flow MUST call `create_document`.
12. A creation flow MUST NOT call Write or Edit.
13. A `SKILL.md` MUST NOT embed a full document template.
14. A `SKILL.md` MUST reference the MCP server template system in place of an embedded template.
15. A `SKILL.md` MUST NOT exceed 300 lines.
16. A per-flow reference file under `references/` or `lib/` MUST NOT exceed 200 lines.
17. The `plan` skill MUST hold its per-flow logic in `skills/plan/references/<flow>.md`.
18. WHEN a new flow is added to the `plan` skill, the author MUST add a reference file instead of a top-level skill.

## Rationale

A fixed file location and a fixed section set let a contributor find each kind of guidance without reading the whole skill, and let a batch update touch every skill the same way. `four-command-palette.adr` fixes the four-command surface and the auto-invocation invariant, so a `disable-model-invocation` flag or an eighth top-level skill breaks routing instead of extending it. `remove-document-type-skills.adr` removed the per-type skill layer, which is why item 9 places per-type elicitation inside `Execution`. Items 13 and 14 exist because an embedded template drifts as soon as the CLI templates change. The trigger and anti-trigger forms in items 6 and 7 are what make model routing between neighboring intents deterministic.

## Examples

### Good — Intent Skill

```markdown
---
name: document
argument-hint: "[topic or description]"
description: "Document a module, component, or system, or record a decision — automatically picks the right type (ADR, spec, doc, guide, or rule). Activate when user says 'document this module', 'capture how X works', 'record this decision', 'write reference docs'. Do NOT activate for planning a feature (use /archcore:plan) or auditing existing documentation (use /archcore:review)."
---

# /archcore:document

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

The frontmatter carries no invocation-restricting flag, so the skill auto-invokes from user phrasing.

### Good — Flow-Style Skill (plan)

```markdown
---
name: plan
argument-hint: "[topic] [--product|--sources|--iso|--feature]"
description: "Plan a feature or initiative end-to-end. Activate when user says 'let's plan', 'create a roadmap for X', 'I need to plan Y'. Do NOT activate for recording a decision (use /archcore:document) or documenting an existing module (use /archcore:document)."
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

### Bad

```markdown
# Frontmatter without name or description        → violates items 3 and 4
# disable-model-invocation: true                 → violates item 5
# Template content embedded verbatim             → violates item 13
# Creation flow calling Write instead of create_document → violates items 11 and 12
# No routing table and no numbered step sequence → violates item 8
# SKILL.md at 340 lines                          → violates item 15
# Per-flow reference file at 260 lines           → violates item 16
# New top-level skill added for a new plan flow  → violates item 18
```

## Enforcement

- Code review during skill development.
- `skills-system.spec` defines the normative contract for skill behavior.
- `plugin-architecture.spec` defines the cross-component invariants.
- `four-command-palette.adr` fixes the four-command surface and the auto-invocation invariant that item 5 depends on.
- No lint script checks items 1–18 today. A `bin/` lint script is the intended verifier. [assumption] No implementation date is set.
