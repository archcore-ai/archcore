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
- Catalog: `sdd` (frame → require → design → decompose), `requirements-cascade` (`mode: sources` = mrd → brd → urd; `mode: iso` = brs → strs → syrs → srs), `decision` (classify → adr | rfc → cascade; resolution entry `decision.resolve` on an existing rfc draft), `describe` (read code → draft spec/doc/guide → clarify gaps), `actualize` (scope diff → per-finding verdict → confirmed fixes), `experience` (detect repeated pattern → cpat | task-type offer), `research` (frame → gather → coverage synthesis or recommendation; standalone evidence exits at gather), `closeout` (verify plan against branch diff → confirmed canon merge → confirmed draft → accepted status transitions).
- Primary executors: `plan` → sdd, requirements-cascade, research; `document` → describe, decision, research; `review` → actualize, experience, closeout; `decision` is callable from all three.
- Gate record fields, fixed order: Purpose; Entry conditions with `skip_when`; Elicitation knobs (trigger, taxonomy, budget); Produces (type, status, relations); Exit checks tagged `blocking` or `advisory`; Next.
- Track state block inside the draft artifact: `<!-- archcore:track -->` with fields `track`, `gate`, `route`, `delta`, `taxonomy`, `asked`, `budget`, `deferred`; research may append `artifact_type`.

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
14. WHEN the user confirms a status transition, the executing skill MAY apply that transition through the owning gate or `update_document`.
15. A hook or subagent MUST NOT change a document status.

16. WHEN an expert invocation names `research`, the research instrument MUST fix its product to `research`, subject to the engine gate.
17. WHEN an expert invocation names `rnd`, the research instrument MUST fix its product to `rnd`.
18. WHEN no type is fixed, the research instrument MUST choose the product by its closing test.
19. The research instrument MUST close `research` on declared scope coverage.
20. The research instrument MUST close `rnd` on an evidenced recommendation.
21. WHEN resuming an artifact, the research instrument MUST preserve its filename type.
22. WHEN an explicit request names `evidence`, the research instrument MUST enter gather without a parent investigation.
23. WHEN standalone evidence has no identified consumer, the research instrument MAY create its draft without a relation.
24. WHEN gather creates dependent evidence, the research instrument MUST persist pending paths and edges before the evidence write.
25. WHEN gather creates dependent evidence, the research instrument MUST add its first evidential edge before gate close.
26. WHEN an evidence operation fails, the research instrument MUST retain successful writes for retry.
27. WHEN an evidence operation fails, the research instrument MUST leave gather open.
28. WHEN resuming a pending evidence operation, the research instrument MUST reconcile existing documents and edges before retrying.
29. WHEN using new vocabulary, the executing skill MUST apply @plugins/archcore/skills/_shared/research-compatibility.md before the first affected MCP call.
30. WHEN standalone evidence completes gather, the research instrument MUST exit the track.
31. WHEN a research state field contradicts the filename type, the research instrument MUST report a blocking state error.

## Constraints & Invariants

- Constraint: a track MUST NOT appear as a palette command.
- Constraint: every gate MUST declare `skip_when`.
- Constraint: every exit check MUST carry the tag `blocking` or `advisory`.
- Constraint: a gate MUST reference shared contracts by path.
- Constraint: a gate MUST NOT restate a shared contract's rules.
- Invariant: adding a track changes one new track file plus one routing-table row per calling skill, and no other file.
- Exception: research gather may checkpoint pending evidence operations before its single gate-close update; the shared gate contract owns the exception.
- Constraint: research and rnd belong to vision; evidence belongs to knowledge.
- Constraint: explicit standalone evidence satisfies frame through the request; no upstream investigation is required.
- Constraint: required research and evidence sections follow the CLI templates; the track records method without prescribing one.
- Invariant: the draft artifact is the only carrier of track state; no session memory or side file holds it.

## Failure Behavior

1. IF a `blocking` exit check fails, THEN the executing skill MUST stop at the current gate and report the failed check.
2. IF an upstream document required by an entry condition is missing, THEN the executing skill MUST route to the earliest gate that produces it.
3. IF a recorded stage is absent, THEN the executing skill MUST resume at the first gate with unmet entry conditions.
4. WHEN recovering from an absent stage, the executing skill MUST preserve recorded clarifications.

## Conformance

A track file and its executing skills are conformant when they satisfy behaviors 1–31, hold all invariants, and degrade per the failure rules.