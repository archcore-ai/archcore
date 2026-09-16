---
title: "Track Handoffs — Cross-Track Transitions and Command Lifecycles"
status: draft
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec defines the edges between tracks and between commands: where one gate or command hands control to another, under which condition, and what carries state across the hop. Normative for the `plan`, `document`, and `review` skills, for track-file authors under `@plugins/archcore/skills/_shared/tracks/`, and for the behavioral routing tests. It complements, and does not restate, the gate mechanics of `track-layer.spec`, the route computation of `delta-routing-conductor.spec`, and the instrument sequencing of `delta-routing-instruments.spec`: an edge those specs already obligate appears in the register below with its owner and receives no clause here. Out of scope: gate internals, interview mechanics, and hook behavior.

## Surface

Transition register. Each row cites the file that owns the edge; a row marked `draft` comes from a draft spec and is not shipped.

| # | From | To | Condition | Owner |
|---|---|---|---|---|
| 1 | `plan` grounding | conductor Derivation | always, before any question | `@plugins/archcore/skills/_shared/delta-routing.md` |
| 2 | conductor | `sdd.require` → `sdd.design` ×N → `sdd.decompose` | `creates` ≥ 1; runbook on an operational procedure | same |
| 3 | conductor | `decision.classify` | Δ `decision` non-empty, or Π `undecided` | same |
| 4 | conductor | `research.frame` / `research.spike` | Π `world` / `empirical`; `plan research` | same |
| 5 | conductor | `requirements-cascade.mrd` | product-scale `intent_gap`, or `plan sources` | same |
| 6 | conductor | `requirements-cascade.brs` | R `security-compliance` per capability, or `plan iso` | same |
| 7 | conductor | `describe.read` (callable) | `modifies` names a capability no `spec` covers | same, rule 11 |
| 8 | `research.spike` exit | conductor Derivation | question resolved, revised Δ | `@plugins/archcore/skills/_shared/tracks/research.md` |
| 9 | `research.conclude` exit on `plan` | conductor, package resumes | investigation closed | this spec, behavior 11 |
| 10 | `requirements-cascade.urd` exit | `sdd.require`, or `requirements-cascade.brs` | follow-up choice; `prd` exists → `related` edges | this spec, behaviors 1–2 |
| 11 | `requirements-cascade.srs` exit | existing `spec` / `plan`, or `sdd.require` | covering document exists or not | this spec, behavior 3 |
| 12 | `brd` / `urd` on topic | `sdd.require` composition | recorded requirement sources | this spec, behavior 4 |
| 13 | any gate | `decision.classify` → back to the gate | a settled choice surfaces | this spec, behaviors 5–6 |
| 14 | `describe.draft` | decision track | the type answer is "a decision" | this spec, behavior 7 |
| 15 | `decision.resolve` | `decision.cascade`, or exit | accepted; rejected or open | this spec, behaviors 8–9 |
| 16 | `decision.rfc` | exit | the track produced an `rfc` | this spec, behavior 10 |
| 17 | `document` unclear | one classifying question → `document` | git evidence supports both readings | `plugin-architecture.spec`, failure 3–4 |
| 18 | `review` branch review | `actualize.scope` | a `spec-wrong` or `code-wrong` finding | this spec, behavior 13 |
| 19 | `review` completion signal | `closeout.verify` | scope from branch state | `@plugins/archcore/skills/review/SKILL.md` |
| 20 | `closeout.capture` | decision standard cascade, or experience types | named residue | `delta-routing-instruments.spec`, 21–23 |
| 21 | closeout exit | `experience.detect` | always | this spec, behavior 12 |
| 22 | `plan` implement fork | later `/archcore:plan` resume | a draft carries a state block | `track-layer.spec`, 10 |
| 23 | `plan` Declared Delta | `closeout.verify` | plan in branch scope | `delta-routing-instruments.spec`, 18 |
| 24 | compatibility probe ≠ `yes` | legacy `rnd`; evidence exits without a write | engine below 0.8.3 | `@plugins/archcore/skills/_shared/research-compatibility.md` |
| 25 | `document journey` | callable `sdd.require`, journey only | `draft` | `actor-subject-compatibility.spec` |
| 26 | `sdd.design` | `sdd.illustrate` | illustrate condition; `draft` | `illustrate-instrument.spec` |

Lifecycle sequences, each crossing at least two commands: build — `plan` → implementation → `review` closeout → discharge → experience offer; proposal — `document` rfc → `document` resolve → cascade; investigation — `plan research` → Derivation → package; discovery — `plan sources` → `sdd.require` → contract → decompose; amendment — `plan` verdict → describe callable → decompose → closeout; first day — `init` → SessionStart recap → `plan` or `document`.

## Normative Behavior

1. WHEN `requirements-cascade.urd` exits and a `prd` on the topic exists, the plan skill MUST add `related` from each source document to that `prd`.
2. WHEN `requirements-cascade.urd` exits and no `prd` exists, the plan skill MUST name the intent instrument and the iso entry as follow-ups.
3. WHEN `requirements-cascade.srs` exits and a `spec` or `plan` covers the topic, the plan skill MUST add `related` from the `srs` to it.
4. WHEN `sdd.require` opens with a `brd` or `urd` on the topic, the plan skill MUST compose from their metrics and criteria without re-asking them.
5. WHEN a settled choice surfaces at a gate, the executing skill MUST enter `decision.classify` before that gate closes.
6. WHEN the decision instrument exits, the executing skill MUST return to the invoking gate with its state block intact.
7. WHEN the `describe.draft` type answer is "a decision", the document skill MUST continue on the decision track instead of `describe.clarify`.
8. WHEN `decision.resolve` records an accepted verdict, the document skill MUST continue at `decision.cascade`.
9. WHEN `decision.resolve` records a rejected or open verdict, the document skill MUST exit without creating a document.
10. WHEN the decision track produced an `rfc`, the document skill MUST exit at `decision.cascade` without a cascade.
11. WHEN the research instrument exits on the `plan` command, the conductor MUST resume the package with the revised Δ.
12. WHEN the closeout track exits, the review skill MUST run the experience offer.
13. WHEN branch review surfaces a `spec-wrong` or `code-wrong` finding, the review skill MUST enter `actualize.scope` with the branch state pre-filled.
14. WHEN a track hands off to another command, the executing skill MUST carry state only through documents: status, state block, Declared Delta.
15. WHEN a register row cites a track file, that file's `Next` field MUST name the same target as the row.

## Constraints & Invariants

- Invariant: command tenses — `plan` declares a future Δ, `document` records the present state, `review` reconciles a past Δ; a lifecycle crosses commands only in that order or by re-entering `plan`.
- Invariant: the register adds no edge the owning files lack; a new edge changes its owning track file or skill first and this register second.
- Invariant: `init` has no track edge; its only return path is the SessionStart empty-state nudge (`hooks-validation-system.spec`).
- Constraint: rows marked `draft` ship with the actor-subject vocabulary release and bind only after their specs are accepted.
- Constraint: a callable entry (rows 7, 25) runs scope-question-free per the instruments spec; the caller pre-fills the scope.
- Constraint: the register lists edges, not gate chains; the chain inside one track stays in its track file and its golden.

## Failure Behavior

1. IF the invoking gate's state block is absent when the decision instrument exits, THEN the executing skill MUST resume per the track-layer resume rules.
2. IF the research instrument exits without a closed investigation on `plan`, THEN the conductor MUST keep Δ unchanged and record a `user`-source Π need.
3. IF a handoff target names a type the engine lacks, THEN the executing skill MUST apply the matching compatibility file before the hop.
4. IF a register row and its owning file disagree, THEN the owning file is right and the executing skill MUST follow it.

## Conformance

An implementation is conformant when every register row is backed by its owner's `Next` field or clause, behaviors 1–15 hold, and the failure rules produce the stated outcomes. Regression coverage: `@test/structure/track-goldens.bats` pins each track's `Next` fields; `@test/structure/trigger-routing.bats` pins the command-level rows; the six lifecycle sequences become routing traces under `@test/behavioral/fixtures/` [planned].

Given a `urd` closed at `requirements-cascade.urd` and a `prd` on the topic, When the track exits, Then `mrd`, `brd`, and `urd` each carry `related` → that `prd`.
