---
title: "Plugin v2 Implementation Plan — Context Migration First, Then Palette Swap"
status: rejected
tags:
  - "plugin"
  - "roadmap"
  - "skills"
---

## Goal

Implement the four-command redesign in the plugin. Hard precondition (Stage 0): migrate the `.archcore/` base itself from the 7-command canon to the v2 canon, so that development sessions, sub-agents, and hook injection never receive contradictory context from the old system. CLI work is tracked in its own plan and referenced only through release ordering.

## Tasks

Stage 0 — Context migration (before any code change):

1. Inventory every document asserting the 7-command surface: search the base for `/archcore:capture`, `/archcore:decide`, `/archcore:context`, `/archcore:audit`, `/archcore:help`, and "exactly 7"; classify each hit as rewrite-to-v2 / reject-as-superseded / keep-unchanged.
2. Flip the v2 design set to `accepted` after maintainer review: the command-surface, track-layer, and elicitation specs.
3. Reject superseded documents so they leave agent steering: the 7-command surface spec, the 7-skill skills spec, the context filtering-pipeline doc, and completed or overtaken implementation plans (context skill, intent-skill migration, scenario-track skills). Record each supersession as a relation to the replacing v2 document.
4. Patch surviving documents whose prose references removed commands (hooks spec examples, agent prompts, guides): replace stale command names with v2 names.
5. Verify that injection sources exclude `rejected` documents (SessionStart listing, `check-code-alignment` ranking). IF a source lacks a status filter, THEN add the status filter to the CLI release A ranking work.

Stage 1 — Shared contracts (additive, old skills untouched):

6. Write `skills/_shared/elicitation-contract.md` per the elicitation spec.
7. Write `skills/_shared/gate-contract.md`: the gate record template with fixed field order.
8. Move init's 13 detect/extract catalogs from `skills/init/lib/` to `skills/_shared/grounding/`; update init references.
9. Write `skills/_shared/branch-state.md`: plain-git instructions for the branch boundary (merge-base against the default branch, changed-file listing) with sentinel handling (no branch, detached HEAD, on default branch).
10. Write `skills/_shared/coverage-taxonomy.md`: per-family coverage categories (Spec Kit's nine, mapped to vision/knowledge/experience).

Stage 2 — Track files under `skills/_shared/tracks/`:

11. `decision.md`: ports decide's classify/compose steps and `continuations.md` into gates (classify → adr | rfc → cascade); question budget 4.
12. `sdd.md`: frame (3 questions) → require (5) → design (3) → decompose (0); product/feature flow content becomes gate bodies.
13. `requirements-cascade.md`: `mode: sources | iso`; per-stage bodies ported from the existing sources/iso flow references; budget 2 per stage.
14. `describe.md`: read code → draft spec/doc/guide → clarify only evidence gaps.
15. `actualize.md`: relocated drift detection + per-finding verdict (`spec-wrong` / `code-wrong` / `ok`) + confirmed fixes one document at a time.
16. `experience.md`: repeated-pattern detection → `cpat` | `task-type` offer.

Stage 3 — New skills (built alongside old ones):

17. Rewrite `skills/plan/SKILL.md`: grounding → routing → budget → sdd or cascade → task-to-file mapping → implement fork.
18. Write `skills/document/SKILL.md`: three-way classification (decision / code-doc / unclear with git investigation) → decision or describe track; trigger description merges capture's and decide's phrases.
19. Rewrite `skills/review/SKILL.md`: branch scope via the branch-state contract (plain git) → bidirectional check → actualize gate on drift signals or `--deep` → experience offer; the auditor agent collects findings read-only, the main thread confirms and applies fixes.
20. Update `skills/init/SKILL.md` edges only: closing summary names four commands; lib paths point to `skills/_shared/grounding/`.

Stage 4 — Cutover (one release):

21. Delete `skills/{context,capture,decide,audit,help}/`, their command wrappers, `bin/git-scope` (context-skill helper), and per-host mirrors.
22. Sweep stale command strings: init follow-ups, `bin/check-staleness` and `bin/check-cascade` messages, `skills/_shared/spec-contract.md`, `agents/archcore-assistant.md`, `rules/*.mdc`, `copilot-agents/*`, and the CLI instructions managed block.
23. Update 4 host manifests and hooks configs; bump the version per the bump-plugin-version pattern.
24. Rewrite the governing docs in the same change set: commands spec, skills spec, plugin-architecture spec, skill-file-structure rule.

Stage 5 — Test gates (built before Stage 4 ships):

25. Trigger-phrase regression suite: fixture prompts → expected command, track, and question count.
26. Golden-transcript checks for gate prose (routing, skip paths, budget adherence).
27. Per-host probe records per the host-probe protocol; CI grep asserting zero stale command strings.

## Acceptance Criteria

- After Stage 0, no `accepted` document in the base describes the 7-command surface.
- All four commands auto-invoke on their trigger fixtures with zero cross-routing failures.
- A fully-specified request completes `plan` and `document` with zero questions; a vague request stays within the 5-question ceiling across all gates.
- An interrupted cascade resumes in a new session without re-asking a recorded answer.
- CI grep finds no `/archcore:{context,capture,decide,audit,help}` string in shipped files.
- Probe records are green for all supported hosts.

## Dependencies

- Stage 0 precedes all other stages.
- CLI release A (Go hook repatriation, status filter in injection, ranked recap) precedes Stage 4.
- CLI release B (MCP prompts removed, instruction track sections stripped) precedes the Stage 4 release.
- The four v2 ADRs are accepted (2026-08-04); the governing-doc rewrites (task 24) land atomically with Stage 4.