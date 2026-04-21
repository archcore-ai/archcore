---
title: "Creating a Batch of Related Skills"
status: draft
tags:
  - "experience"
  - "plugin"
  - "skills"
---

## What

A pattern for efficiently creating multiple SKILL.md files when building out coverage for a category of Archcore document types (e.g., all 6 knowledge types, all 10 vision types, or all 2 experience types).

## When to Use

- Adding skill coverage for an entire Archcore category at once
- Building the initial 17 type skills during Phase 2 of plugin development
- Updating all skills in a category after a structural change to the Skill File Structure Standard or the Inverted Invocation Policy

## Steps

### 1. Pick the batch scope

Choose a category to batch: knowledge (6 types), vision (10 types), or experience (2 types). Start with the category you understand best.

### 2. Identify the skill layer and constraints

All document-type skills are Layer 3 (Type skills). Per the Skill File Structure Standard and the Inverted Invocation Policy ADR:

- **Section structure**: 3 sections — When to Use, Quick Create, Relations
- **Line limit**: ≤ 100 lines
- **Description prefix**: Non-high-frequency types MUST use "Expert —" prefix; high-frequency mainstream types (adr, prd, rule, guide, idea) keep clean descriptions
- **Invocation flag depends on policy class**:
  - **Mainstream types** (adr, prd, rfc, rule, guide, doc, spec, idea, task-type, cpat): MUST include `disable-model-invocation: true`. User-only via `/`; the model reaches them through intent-skill routing.
  - **Niche types** (mrd, brd, urd, brs, strs, syrs, srs): MUST include `user-invocable: false`. Hidden from `/` autocomplete; the model reaches them programmatically via `sources-track` or `iso-track` orchestration.

### 3. Create one exemplar skill

Pick the most representative type in the category (e.g., ADR for knowledge, PRD for vision mainstream, BRS for vision niche, task-type for experience). Write its SKILL.md thoroughly — this becomes the quality bar and structural template for the batch.

Get the exemplar reviewed and validated:

- Follows the 3-section structure (When to Use, Quick Create, Relations)
- Quick Create uses MCP tools
- Description field has precise trigger conditions with "Expert —" prefix where applicable
- Invocation flag matches the policy class (mainstream vs niche)
- Contrast with similar types is clear
- Under 100 lines

### 4. Extract the pattern

Identify which parts of the exemplar are:

- **Fixed**: section headings (When to Use, Quick Create, Relations), frontmatter structure, MCP tool references — these stay the same
- **Variable**: type name, description triggers, use-case contrast, relation flows, example parameters, invocation flag (per class) — these change per type

### 5. Create remaining skills in the batch

For each remaining type in the category:

1. Copy the exemplar's 3-section structure (not content)
2. Set the correct invocation flag for the type's policy class (mainstream → `disable-model-invocation: true`; niche → `user-invocable: false`)
3. Study that type's template via `create_document` (no content) + `remove_document`
4. Fill in the variable sections with type-specific content
5. Pay special attention to "When to Use" — the contrast with other types must be accurate
6. Verify the Quick Create uses realistic parameters for this type
7. Confirm the skill stays under 100 lines

### 6. Cross-validate the batch

After all skills in the batch are created:

- Read all "When to Use" sections together — ensure no ambiguity between types
- Check all "Relations" sections — ensure flows are consistent across the batch
- Verify all Quick Create examples use correct MCP tool names
- Verify invocation flags match policy class for every skill in the batch
- Run `/reload-plugins` and test each skill's user invocation (mainstream) or track-mediated invocation (niche)

## Example

Creating knowledge category skills (6 mainstream types):

1. **Exemplar**: `skills/adr/SKILL.md` — ADR is well-understood, has clear contrasts (vs RFC, vs rule)
2. **Extract pattern**: 3 sections are fixed, `disable-model-invocation: true` is fixed (all knowledge types are mainstream), but triggers differ (ADR: "decision made"; RFC: "proposal to discuss"; rule: "team standard")
3. **Batch create**: rfc, rule, guide, doc, spec — each with unique triggers, Quick Create examples, and relation guidance, all carrying `disable-model-invocation: true`
4. **Cross-validate**: ensure "When to Use" for ADR clearly separates from RFC, rule from guide, doc from spec

Creating vision-niche batch (4 ISO 29148 types):

1. **Exemplar**: `skills/brs/SKILL.md` — BRS opens the ISO cascade
2. **Extract pattern**: All 4 niche types use `user-invocable: false` (hidden from `/`), reached only via `/archcore:iso-track`
3. **Batch create**: strs, syrs, srs — each scoped to its layer of the ISO cascade with implements relations to the prior layer
4. **Cross-validate**: ensure each step's Quick Create includes the implements relation back to the prior cascade document

## Things to Watch Out For

- **Copy-paste drift**: After copying the exemplar structure, ensure all content is customized. Leftover ADR-specific content in an RFC skill is confusing.
- **Wrong invocation flag**: Mainstream types missing `disable-model-invocation: true` will pollute model context with type descriptions and bypass intent routing. Niche types missing `user-invocable: false` will appear in `/` autocomplete and add cognitive load.
- **Trigger overlap**: Two skills with similar descriptions will compete for activation. The "When to Use" section must clearly differentiate, even though most types are user-only — the differentiation still matters for `/` autocomplete and for human readability.
- **Relation guidance inconsistency**: If the ADR skill says "rules implement ADRs" but the rule skill doesn't mention ADRs in its incoming relations, agents get confused.
- **Template embedding**: Don't copy template sections verbatim into skills. The templates change with CLI versions. Reference them instead.
- **Batch fatigue**: Quality drops on the last few skills in a batch. Review the final skills in the batch with the same rigor as the first.
- **Line limit violations**: Type skills must be ≤ 100 lines. If a skill is getting long, move detailed guidance to the Adding a New Document Type Skill guide instead.
