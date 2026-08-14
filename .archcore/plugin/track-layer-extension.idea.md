---
title: "Track-Layer Extension — Research, Closeout, and RFC Resolution"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Idea

Extend the track layer with the three flow-coverage gaps the 2026-08-07 landscape audit surfaced: a `research` track producing `rnd`, a `closeout` track closing the implementation cycle under `/archcore:review`, and a resolution entry for `rfc` inside the decision track. Beneficiary: any team whose `.archcore/` accumulates types no command path produces or consumes — today `rnd` has no producer and no grounding reader, `rfc` has no resolution path, and no track transitions `status: draft` to `accepted`.

## Value

- `rnd` becomes producible through a command path — closing a gap in the plugin-v2 "every document type is producible through a command path" metric. Evidence in this corpus: three hand-created `rnd` documents are the evidence base of the v2-redesign ADRs, all created through direct MCP outside any flow.
- Closes the post-implementation loop the 2026 market converges on (OpenSpec `/opsx:archive`, Spec Kit `/speckit.converge`, claude-shapeup `/shapeup:ship`): plan-versus-diff verification, canon updates, and the draft → accepted status ceremony — today fully manual.
- An `rfc` stops dead-ending after `decision.rfc`: an accepted proposal becomes an `adr` plus the existing cascade; a rejected one gets `status: rejected`.

## Possible Implementation

- `research` track: frame the questions → gather sources and facts → conclude with findings and a recommendation; produces `rnd` with the template sections Goal, Questions, Approach, Findings, Recommendation, Next Action. Executor: `/archcore:plan` — settled at the require gate; boundary against sources mode: technical research versus market discovery. An existing `rnd` closes `sdd.frame`'s skip_when the way an `idea` does; `rnd` joins the plan and document grounding type filters.
- `closeout` track under `/archcore:review`: verify (plan tasks and acceptance criteria versus the branch diff) → merge (update the spec and doc documents the diff touched) → accept (per-document confirmed status transitions draft → accepted) → exit into the existing experience offer.
- `decision.resolve`: a resume gate on an `rfc` draft — on acceptance create the `adr` with `extends` → rfc and continue into `decision.cascade`; on rejection set the rfc `status: rejected`.
- Per the track-layer.spec invariant, each new track costs one track file under `skills/_shared/tracks/` plus one routing-table row per calling skill.

## Risks and Constraints

- The track-layer.spec catalog line and the component registry enumerate tracks; both corpus documents must be updated in step — the "one file plus routing rows" invariant covers plugin files, not corpus documents.
- `research` borders `requirements-cascade` sources mode: the routing boundary between technical research and market discovery must be explicit, or `plan` mis-routes ("market research" → mrd today).
- `closeout` writes statuses across many documents; it needs the same per-document confirmation discipline as `actualize.fix` to stay non-destructive.
- The 5-question auto-mode ceiling must hold across the new gates; `closeout`'s per-document confirmations follow the `actualize.fix` model (confirmations, not budget questions) [assumption].
