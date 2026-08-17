---
title: "How to Write a spec Document"
status: accepted
tags:
  - "document-types"
---

## Overview

A `spec` document is the durable, normative contract for **behavior others rely on right now** — one API, interface, schema, protocol, feature, or subsystem. One subject per spec, and **one form**: the same six sections regardless of subject. A spec may be written after code (capture the behavior of what exists) or before it (specify what to build). If implementation diverges from the spec, the spec takes precedence.

**Canonical source**: this guide mirrors the Archcore plugin's spec contract (`skills/_shared/spec-contract.md` in the plugin repo) and its ADR "Spec — Single Narrative, Generalized Sections, EARS + BCP 14 Notation". When that contract changes, update this guide and the template in @templates/templates.go together.

**Routing gate**: if the document answers *"what should we build and why"* (user stories, priorities, success metrics), it is a `prd` (or ISO `syrs`/`srs`) — not a spec. If it answers *"what behavior can consumers rely on right now"*, it is a spec.

### Target Audience

Engineers and AI agents authoring or reviewing specs in this repository.

## Prerequisites

- Confirm the subject warrants a spec — see the spec-type-usage rule (not a `doc`, `rule`, `adr`, `guide`, or `prd`)
- If an `adr` records the decision behind the subject, link it via `add_relation` after creating

## Steps

### Step 1: Purpose & Scope

Name the one subject this spec is normative for, who depends on it (external code, teams, UI surfaces, or sibling modules), and what is out of scope (with pointers).

### Step 2: Surface

What dependents see of the subject: the externally observable interface (inputs, outputs, signatures) and/or the parts, states, and data fields that drive behavior. Reference source definitions by canonical identifier plus `@path/to/file` — never copy interface, type, or struct bodies; copies go stale. Use a code block only where the exact textual format is itself normative (HTTP endpoint shape, CLI flag grammar, wire format).

### Step 3: Normative Behavior

Numbered requirements, each in EARS clause order with a BCP 14 keyword (MUST / SHOULD / MAY, uppercase only — RFC 2119 + RFC 8174) as the modal:

- Ubiquitous: `The <subject> MUST <response>.`
- Event-driven: `WHEN <trigger>, the <subject> MUST <response>.`
- State-driven: `WHILE <state>, the <subject> MUST <response>.`
- Unwanted behavior: `IF <undesired condition>, THEN the <subject> MUST <response>.`

Grade with intent: MUST sparingly — interoperation or harm prevention only (RFC 2119 §6); SHOULD where deviation needs a weighed reason; MAY for true options. No rationale tails ("…so that X") — rationale lives in a linked `adr`.

Three rules keep each line strict-EARS conformant:

1. **Active voice with an obligated subject** — never a subjectless passive. "Tokens MUST be rotated" names no component that bears the obligation; write `the <component> MUST rotate the token`.
2. **One requirement per numbered line = one modal keyword** (MUST NOT counts as one). Split `MUST X and MUST NOT Y` into two numbered lines.
3. **Event responses open with the trigger.** When behavior answers a command, request, or state change, use `WHEN <trigger>, the <subject> MUST …` — the event is the trigger, never the grammatical subject. Never leave the trigger implicit inside the subject.

### Step 4: Constraints & Invariants

Hard limits (each with a rationale) and invariants (conditions that MUST always hold), listed separately. Plain BCP 14 statements — EARS clauses are not required here.

### Step 5: Failure Behavior

Error and edge conditions with the observable outcome of each: response and recovery semantics (retriable? idempotent? timeout behavior?) and degradation on bad, empty, or missing input or on dependency failure. Same notation and rules as Normative Behavior; error paths use `IF …, THEN …`, not `WHEN`.

### Step 6: Conformance

What makes an implementation correct: satisfies all MUST requirements, all invariants, and all failure rules. Point to the executable conformance suite (co-located tests). MAY close with ONE non-normative example block (≤ 5 lines, Given/When/Then) anchoring the most load-bearing behavior.

## Body Cap

**≤ 120 lines**, measured over the whole body — headings and blank lines counted, and on the six-section form those alone take about 19 lines. The "reference, don't reproduce" rule is what keeps a spec inside the cap even for a complex subject.

One number now covers every path. The separate 120-line flagship allowance that `/archcore:init` carried for hotspot synthesis is folded into the default, so a synthesized spec and an authored one are measured alike. The engine reads @templates/precision.go `MaxSpecBodyLines`; the finding is advisory and never blocks a write.

Past the cap the answer is decomposition, not a longer document: split by sub-surface and relate the parts (Issue 6 below).

## Forbidden in the Body

- Decision rationale ("we chose X because…") → linked `adr`
- User stories, priorities, success metrics → `prd`
- General reference material (glossaries of everything, inventories) → `doc`
- Sequential how-to steps ("first call X, then Y") → `guide`
- A section enumerating other `.archcore/` documents — cross-document links live in the relation graph via `add_relation`. Citing source code (`@path`), schemas, and external authorities is fine.

## Verification

After writing, check:

- [ ] Exactly six sections, in order: Purpose & Scope, Surface, Normative Behavior, Constraints & Invariants, Failure Behavior, Conformance
- [ ] Every normative line is numbered, follows EARS clause order, and carries a BCP 14 modal — no SHALL
- [ ] One requirement per line — one modal keyword; split multi-MUST clauses
- [ ] Active voice with an obligated subject — no subjectless passives ("MUST be rotated"); the subject of every clause is the specified system, not its caller
- [ ] Event responses (commands, requests, state changes) open with `WHEN <trigger>,` — the event is never the grammatical subject
- [ ] Error paths sit in Failure Behavior as `IF …, THEN …`
- [ ] Interfaces referenced by identifier + `@path`, never reproduced
- [ ] Body ≤ 120 lines
- [ ] No rationale, stories, reference dumps, how-to steps, or related-documents sections

## Common Issues

### Issue 1: SHALL-only pure EARS

**Cause**: Adopting EARS together with its traditional `shall` keyword.

**Solution**: The modal here is BCP 14 — MUST/SHOULD/MAY grading. Shall-only EARS was an explicitly rejected alternative in the plugin ADR (it loses the grading).

### Issue 2: Pattern-taxonomy sections

**Cause**: Grouping requirements into "Ubiquitous / Event-driven / State-driven…" subsections with U1/E1/S1 labels.

**Solution**: Plain sequential numbering — the clause order itself carries the pattern. Taxonomy labels add lines and misclassification surface without adding precision.

### Issue 3: Error paths written as WHEN

**Cause**: Treating failures as ordinary events.

**Solution**: Unwanted behavior is `IF <condition>, THEN the <subject> MUST <response>` — and it belongs in Failure Behavior.

### Issue 4: Compound clauses

**Cause**: Chaining several MUSTs (or a MUST and a SHOULD with different subjects) in one numbered line.

**Solution**: One requirement per line. Split.

### Issue 5: Requirements on the caller

**Cause**: Normative lines like "the caller MUST abort".

**Solution**: The clause subject is the specified component. Caller guidance becomes a recovery note in Failure Behavior, or moves to the consumer's own spec.

### Issue 6: Spec too broad, or a second copy of the code

**Cause**: Specifying a whole subsystem in one document, or pasting interface/type definitions "for precision".

**Solution**: One subject per spec — split by component boundary and link via `add_relation`. Reference `@path`s instead of reproducing source; reserve code blocks for wire-level contracts where the textual form is itself normative.

### Issue 7: Subjectless passive obligation

**Cause**: `<thing> MUST be <verb-ed>` with no actor named — "tokens MUST be rotated", "results MUST be recorded". The reader cannot tell which component owns the obligation.

**Solution**: Name the obligated component as the grammatical subject — `the <component> MUST rotate the token`. If the actor is genuinely responding to an event, use the event-driven form `WHEN <trigger>, the <component> MUST …`.

### Issue 8: Command or event as the subject

**Cause**: `The /rebuild-index command invalidates the cache` — the triggering command written as the clause subject, leaving the trigger implicit and no component obligated.

**Solution**: Make the trigger explicit and the component the subject: `WHEN the user invokes /rebuild-index, the <component> MUST invalidate the cache`.