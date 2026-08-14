---
title: "Track-Layer Extension — Flow Coverage for rnd, rfc Resolution, and Closeout"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
  - "vision"
---

## Vision

One initiative closes the flow-coverage gaps the 2026-08-07 landscape audit surfaced: every document type is producible and consumed through a command path, an implemented feature closes its own loop, and proposals resolve instead of dead-ending.

## Problem Statement

`rnd` has no producer and no grounding reader; `rfc` dead-ends after `decision.rfc`; `task-type`/`cpat` never reach `sdd.decompose`, which composes "only from upstream drafts"; the mrd…srs cascade hands off outward only as advisory; no track transitions `status: draft` to `accepted` — acceptance is fully manual.

## Goals and Success Metrics

- "Investigate X" through `/archcore:plan` produces an `rnd` draft; a fully-specified request asks zero questions.
- An existing `rnd` on the topic closes `sdd.frame`'s skip_when the way an `idea` does.
- An `rfc` draft resumes into a resolution gate: accepted → `adr` + cascade; rejected → `status: rejected`.
- A closeout run transitions confirmed documents draft → accepted, one per-document confirmation each, and modifies no code.
- `rnd` and `rfc` appear in the plan and document grounding type filters; `sdd.decompose` lists `task-type` precedent among its composition inputs.
- The track-layer.spec invariant holds: each new track costs one track file plus one routing-table row per calling skill.

## Requirements

1. `research` track at `/archcore:plan`: routing signal — technical research, alternatives comparison, "investigate before planning or deciding"; market discovery keeps routing to `requirements-cascade` sources mode.
2. `research` gates: frame → gather → conclude; produces `rnd` (Goal, Questions, Approach, Findings, Recommendation, Next Action), `status: draft`.
3. `closeout` track at `/archcore:review`: verify (plan versus branch diff) → merge (update the spec and doc documents the diff touched) → accept (per-document confirmed status transitions) → exit into the experience offer.
4. `decision.resolve`: resume gate on an `rfc` draft; accepted → `adr` `extends` rfc, then `decision.cascade`; rejected → rfc `status: rejected`.
5. Gate-edit phase: `rnd`/`rfc` join the grounding type filters; `task-type` becomes a precedent input at `sdd.decompose`; existing `urd`/`srs` documents satisfy `sdd.require` entry conditions.
6. The track-layer.spec catalog and the component registry are updated in the same change.

## Clarifications

- research executor: a dedicated track under `/archcore:plan`; boundary — technical research versus market discovery, sources mode keeps `mrd` (user, 2026-08-07).
- scope: the non-track gate edits ship as a separate phase of this same plan (user, 2026-08-07).
