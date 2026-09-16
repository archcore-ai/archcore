---
title: "Scenario and Journey in the Precision Canon and the Advisory Engines"
status: draft
tags:
  - "docs-style"
  - "document-types"
  - "integrations"
---

## Purpose & Scope

This contract defines how the precision canon, the post-write checker, the pre-write injector, the search ranker, and the relation engines treat `scenario` and `journey`. It depends on the type contract for the section names. Consumers are the hook path, the plugin's review and closeout gates, and any project that installs the CLI without the runtime. The templates and the MCP surface are outside scope.

## Surface

The canon data lives in @templates/precision.go: `RequiredSections`, `ProseProfiles`, `StepSections`, `ObservationSections`, `ActorStepSections`, `ForeignSections`, and the per-type body-cap table `MaxBodyLines`. The checker is @internal/advisory/precision.go. The injector ranking is `alignmentTypePriority` in @internal/advisory/code_alignment.go. The search ranking is `typePriority` in @internal/mcp/tools/search_documents.go. The cascade set is `cascadeRelations` in @cmd/hook_post_tool_use.go; the restatement set is `contentRelations` in @internal/advisory/restatement.go.

Line format F6, actor-subject step: `<Actor> <action>; <system> <observable response>.` or `Given|When|Then|And|But <observation>.` One step carries one action or one observation, holds 20 words or fewer, and carries no modal. The five opening words of the second form are the observation keywords.

## Normative Behavior

1. The precision canon MUST require the sections the type contract's Surface table assigns to each type.
2. The precision canon MUST carry both types under the ISO profile.
3. The precision canon MUST register Flows for `scenario` and Journeys for `journey` as step sections.
4. WHEN a step in a step section exceeds 20 words, the checker MUST report the step.
5. WHEN a step in a step section carries a BCP 14 modal, the checker MUST report the modal as a modal in a step.
6. WHEN a line in a step section or a `scenario` Examples section opens with an observation keyword, the checker MUST treat it as a step.
7. WHEN a `###` subsection under Flows carries no line opening with `Anchors:`, the checker MUST report the subsection title.
8. WHEN a step under Flows or Journeys opens with neither an Actors-table actor nor an observation keyword, the checker MUST report the step.
9. The precision canon MUST hold the body caps in one table keyed by type, with `spec`, `scenario`, and `journey` at 120 lines each.
10. WHEN a body exceeds its type's cap, the checker MUST report the line count and the cap, counted as the `spec` cap is counted.
11. WHEN a `spec` or `prd` carries a `Flows` or `Examples` section, the checker MUST report the heading with owner `scenario`.
12. WHEN a `spec` or `prd` carries a `Journeys` section, the checker MUST report the heading with owner `journey`.
13. WHEN a `scenario` or `journey` carries a `Surface`, `Normative Behavior`, or `Failure Behavior` section, the checker MUST report the heading with owner `spec`.
14. WHEN a `scenario` or `journey` carries a `Requirements` section, the checker MUST report the heading with owner `prd`.
15. The CodeAlignment selector MUST include `scenario`, ranked below `spec` and above `guide`.
16. The CodeAlignment selector MUST exclude `journey`.
17. The search ranker MUST use its default type priority for both types.

## Constraints & Invariants

The `Examples` heading of `rule` and `doc` templates is not foreign; clause 11 names `spec` and `prd` only. The actor list for clause 8 is the first column of the Actors table, matched case-insensitively against the opening words of a step; a row counts with or without its outer pipes, as GFM allows. A line under clause 6 counts whether or not it carries a number. A fenced block under Actors or Flows contributes no actor, no subsection, and no `Anchors:` line, as a fenced block contributes no clause elsewhere in the checker. A Flows section with numbered steps and no `###` subsection counts as one flow named after the section. The identifier `MaxSpecBodyLines` stays readable at 120 and feeds the three table entries, so existing citations hold.

1. Every check above MUST report and MUST NOT block a write.
2. The cascade set MUST stay `implements`, `depends_on`, `extends`, so `scenario depends_on spec` and `scenario implements journey` carry the notice without a code change.
3. The restatement set MUST keep reading `implements`, so a `scenario` restating its `journey` is reported without a code change.
4. The ISO-profile claim check MUST keep applying to numbered lines outside step sections, so a modal in Subject or Intent is reported as a claim defect.

## Failure Behavior

1. IF the Actors table is absent, THEN the checker MUST skip clause 8 and report only the missing section.
2. IF a required section is absent, THEN the checker MUST emit an advisory naming the section and the write proceeds.
3. IF a body is over its cap, THEN the checker MUST report the excess and the write proceeds.

## Conformance

Conformance requires every numbered obligation above. Regression coverage: @internal/advisory/precision_actor_subject_test.go pins clauses 1 to 14, the fence and outer-pipe constraints, and the failure rules; @internal/advisory/code_alignment_test.go pins clauses 15 and 16; @internal/mcp/tools/search_documents_test.go pins clause 17; @cmd/hook_post_tool_use_test.go and @internal/advisory/restatement_test.go pin constraints 2 and 3.
