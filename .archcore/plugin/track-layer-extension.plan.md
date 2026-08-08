---
title: "Track-Layer Extension Implementation — research, closeout, decision.resolve, Gate Edits"
status: draft
tags:
  - "architecture"
  - "plugin"
  - "roadmap"
  - "skills"
---

## Goal

Ship the track-layer extension per `track-layer-extension.prd`: a `research` track under `/archcore:plan`, a `closeout` track under `/archcore:review`, a `decision.resolve` gate, and the gate edits that wire orphaned types into existing flows — within the track-layer.spec invariant (one track file plus one routing row per calling skill, per track).

## Tasks

### Phase 1 — research track (closes the rnd orphan)

1. Author the track file `research.md` — gates `research.frame` (questions and scope), `research.gather` (sources and facts; the read-only auditor MAY collect, the main thread interviews), `research.conclude` (findings, recommendation, next action) — per the gate-contract template, producing `rnd` with `status: draft` → new file under @plugins/archcore/skills/_shared/tracks/
2. Add the routing row to the plan skill — signal: technical research, alternatives comparison, "investigate X before planning or deciding"; keep market discovery routing to sources mode; add `rnd` to the grounding type filter → @plugins/archcore/skills/plan/SKILL.md
3. Accept an existing `rnd` in `sdd.frame`'s skip_when alongside `idea`/`prd`, and name `rnd` as evidence input at `decision.adr` → @plugins/archcore/skills/_shared/tracks/sdd.md, @plugins/archcore/skills/_shared/tracks/decision.md

### Phase 2 — decision.resolve (rfc lifecycle)

4. Add the gate `decision.resolve` — entry: an `rfc` draft on the topic exists; accepted → create `adr` with `extends` → rfc, continue into `decision.cascade`; rejected → rfc `status: rejected`; dropped → record and exit → @plugins/archcore/skills/_shared/tracks/decision.md
5. Add the resume signal to the document skill routing table ("resolve the RFC", "we accepted the proposal") and `rfc` to the plan grounding filter → @plugins/archcore/skills/document/SKILL.md, @plugins/archcore/skills/plan/SKILL.md

### Phase 3 — closeout track (status ceremony and canon merge)

6. Record the status-transition model as an `adr` through the decision track before authoring the gates: which types transition, who confirms, how rejection is recorded — no code target; prerequisite for task 7.
7. Author the track file `closeout.md` — gates `closeout.verify` (plan tasks and acceptance criteria versus the branch-state diff), `closeout.merge` (update the spec and doc documents the diff touched, per-document confirmation as in `actualize.fix`), `closeout.accept` (confirmed draft → accepted transitions), exit into the experience offer → new file under @plugins/archcore/skills/_shared/tracks/
8. Add the routing row to the review skill — signal: "close out the feature", "ship", a finished branch after review — and pre-fill scope from `branch-state` → @plugins/archcore/skills/review/SKILL.md

### Phase 4 — gate edits (wiring orphans into existing flows)

9. Add `task-type` and `cpat` precedent to `sdd.decompose` composition inputs → @plugins/archcore/skills/_shared/tracks/sdd.md
10. Let existing `urd`/`srs` documents satisfy `sdd.require` entry conditions as recorded requirement sources → @plugins/archcore/skills/_shared/tracks/sdd.md
11. Make the `requirements-cascade.srs` exit create its `spec`/`plan` relations via `add_relation` instead of advising → @plugins/archcore/skills/_shared/tracks/requirements-cascade.md

### Phase 5 — corpus and verification

12. Update the track-layer.spec catalog and executors lines (8 gates picture: +research, +closeout, +decision.resolve) and the component registry via MCP `update_document` — corpus documents, no plugin files.
13. Run `make verify` from the repository root; extend structure tests if they enumerate tracks → @Makefile, @test/

## Acceptance Criteria

- Each new track file passes the gate-contract authoring rules: template field order, `skip_when` in every gate, every exit check tagged `blocking` or `advisory`, shared contracts referenced by path.
- "Investigate X" through `/archcore:plan` produces an `rnd` draft; a fully-specified request asks zero questions; a "market research" request still routes to sources mode.
- An `rfc` draft resumes into `decision.resolve`; the accepted path yields an `adr` with `extends` → rfc; the rejected path sets `status: rejected`.
- A closeout run transitions statuses only after per-document confirmation, modifies no code file, and ends with the experience offer.
- The plan and document grounding filters include `rnd` and `rfc`; `sdd.decompose` names `task-type` precedent among composition inputs.
- `make verify` passes; the auto-mode 5-question ceiling holds across all new gates.

## Dependencies

- `gate-contract.md` (authoring template) and `elicitation-contract.md` (budgets) govern all new gates.
- track-layer.spec invariant: one track file plus one routing row per calling skill — corpus doc updates ride the same change but touch no plugin file.
- `branch-state.md` supplies the closeout.verify scope; the experience track supplies the closeout exit.
- Task 6 (status-transition `adr`) blocks task 7; Phases 1, 2, 4 are mutually independent; Phase 5 lands last.
