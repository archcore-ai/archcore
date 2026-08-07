---
title: "Track Layer — Gated Flows Beneath the Command Surface"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec defines the track layer: gated flows that layer-1 commands route into without a user-visible track menu. Normative for track files under @plugins/archcore/skills/_shared/tracks/ and for the skills that execute them (`plan`, `document`, `review`). Out of scope: interview mechanics and per-type content contracts.

## Surface

- Track files: `skills/_shared/tracks/<track-id>.md`; gates as `### gate: <track>.<stage>` sections.
- Catalog: `sdd` (frame → require → design → decompose), `requirements-cascade` (`mode: sources` = mrd → brd → urd; `mode: iso` = brs → strs → syrs → srs), `decision` (classify → adr | rfc → cascade), `describe` (read code → draft spec/doc/guide → clarify gaps), `actualize` (scope diff → per-finding verdict → confirmed fixes), `experience` (detect repeated pattern → cpat | task-type offer).
- Primary executors: `plan` → sdd, requirements-cascade; `document` → describe, decision; `review` → actualize, experience; `decision` is callable from all three.
- Gate record fields, fixed order: Purpose; Entry conditions with `skip_when`; Elicitation knobs (trigger, taxonomy, budget); Produces (type, status, relations); Exit checks tagged `blocking` or `advisory`; Next.
- Track state block inside the draft artifact: `<!-- archcore:track -->` with fields `track`, `gate`, `taxonomy`, `asked`, `budget`, `deferred`.

## Normative Behavior

1. WHEN routing resolves, the executing skill MUST evaluate signals in this order: explicit expert invocation, document-graph state, branch state, request wording.
2. The executing skill MUST NOT ask the user to choose a track.
3. The executing skill MUST derive the question budget, not the track choice, from input vagueness.
4. WHEN a gate opens, the executing skill MUST evaluate `skip_when` before any other gate step.
5. WHEN existing documents or the request text satisfy a gate's entry conditions, the executing skill MUST ask zero questions at that gate.
6. WHEN a gate produces a document, the executing skill MUST create it with `status: draft` via `create_document`.
7. WHEN all `blocking` exit checks pass, the executing skill MUST advance the state block's `gate` field to the next stage.
8. IF an `advisory` exit check fails, THEN the executing skill MUST proceed to the next gate and report the finding.
9. WHEN a track exits, the executing skill MUST remove the state block from the artifact.
10. WHEN a skill opens a draft carrying a state block, the skill MUST resume at the earliest gate whose exit checks have not passed.
11. WHILE resuming, the executing skill MUST NOT re-ask questions recorded in `taxonomy` or `## Clarifications`.
12. The review skill MAY run a gate in the reverse direction (code → document) with entry evidence pre-filled from git.
13. WHEN a gate closes, the executing skill MUST persist answers and the state block in one `update_document` call.

## Constraints & Invariants

- Constraint: a track MUST NOT appear as a palette command.
- Constraint: every gate MUST declare `skip_when`.
- Constraint: every exit check MUST carry the tag `blocking` or `advisory`.
- Constraint: a gate MUST reference shared contracts by path.
- Constraint: a gate MUST NOT restate a shared contract's rules.
- Invariant: adding a track changes one new track file plus one routing-table row per calling skill, and no other file.
- Invariant: the draft artifact is the only carrier of track state; no session memory or side file holds it.

## Failure Behavior

1. IF a `blocking` exit check fails, THEN the executing skill MUST stop at the current gate and report the failed check.
2. IF an upstream document required by an entry condition is missing, THEN the executing skill MUST route to the earliest gate that produces it.
3. IF the state block names a stage absent from the track file, THEN the executing skill MUST resume at the first gate whose entry conditions fail and preserve recorded clarifications.

## Conformance

A track file and its executing skills are conformant when they satisfy behaviors 1–13, hold all invariants, and degrade per the failure rules.