---
title: "Scenario Evidence in Describe and Closeout — Feature Files as Evidence, Readiness and Coverage Checks"
status: draft
tags:
  - "document-types"
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec defines how the document and review skills consume scenarios and feature files: `describe.read` reads `features/*.feature` as evidence and `describe.draft` may produce a `scenario` for existing behavior; `closeout.verify` reports readiness and coverage in Smart's two report kinds; `closeout.accept` names the readiness result in its offer. Normative for the `document` and `review` skills and for the authors of `skills/_shared/tracks/describe.md` and `skills/_shared/tracks/closeout.md`. Plan-side production is the illustrate-instrument spec; document content is the content-contracts spec. Out of scope: execution of any example — the runtime executes nothing.

## Surface

- Describe track in `@plugins/archcore/skills/_shared/tracks/describe.md`: `describe.read` evidence base gains feature files; the type heuristics table gains a `scenario` row; `describe.draft` Produces gains `scenario`.
- Closeout track in `@plugins/archcore/skills/_shared/tracks/closeout.md`: `closeout.verify` gains two advisory exit checks — readiness and coverage; `closeout.accept` offer text names the readiness result for a `scenario`.
- Review grounding in `@plugins/archcore/skills/review/SKILL.md`: the closeout scope filter adds `scenario` and `journey` when the compatibility probe returned `yes`.
- Verdict vocabulary: `@plugins/archcore/skills/_shared/verdict-contract.md` — `spec-wrong`, `code-wrong`, `ok`.
- Report kinds: readiness — every example of each scoped `scenario` was run or confirmed; coverage — no scoped `spec` clause lacks an example, and no cited feature file is unnamed by a `spec`.

## Normative Behavior

1. WHEN `describe.read` finds `features/*.feature` files for the subject, the executing skill MUST record them as evidence for Failure Behavior and Conformance.
2. WHEN the evidence shows an actor-subject flow of existing behavior, the executing skill MAY produce a `scenario` at `describe.draft` beside the `spec`.
3. WHEN `describe.draft` produces a scenario, the executing skill MUST add `depends_on` → the covering `spec`.
4. WHEN a scenario is requested and no covering `spec` exists, the executing skill MUST produce the `spec` first.
5. WHEN `describe.draft` produces a scenario, the executing skill MUST cite the feature files it read in the Flows `Anchors:` lines.
6. WHEN `closeout.verify` scopes a `scenario`, the executing skill MUST report readiness per example: run, confirmed, or unconfirmed.
7. WHEN `closeout.verify` scopes a `spec`, the executing skill MUST report each Normative Behavior clause that no example illustrates.
8. WHEN `closeout.verify` scopes a `spec` that cites a feature file, the executing skill MUST report a cited file absent from the branch.
9. WHEN a feature file on the branch is cited by no scoped `spec`, the executing skill MUST report the file.
10. The executing skill MUST label readiness and coverage findings advisory.
11. WHEN a scenario's `Anchors:` paths changed in the branch diff, the executing skill MUST label the scenario per the verdict contract.
12. WHEN `closeout.accept` offers a `scenario` transition, the offer MUST name that scenario's readiness result.
13. WHEN `closeout.merge` proposes a scenario update, the executing skill MUST obtain the per-document confirmation before the write.
14. The executing skill MUST NOT execute a feature file or an example.
15. WHEN readiness rests on the user's word, the executing skill MUST record the confirmation in the running report.

## Constraints & Invariants

- Constraint: coverage per clause reads clause numbers as written; [assumption] stable clause identifiers, the open item of `spec-single-narrative-ears-bcp14.adr`, are not required for this release.
- Constraint: an example counts for coverage when it sits in the `spec` Conformance block, in a `scenario` that `depends_on` the `spec`, or in a feature file the `spec` cites by `@path`.
- Constraint: readiness evidence is a test-run report in the branch, a scenario body that records the confirmation, or the user's confirmation at the gate; the runtime infers none.
- Invariant: the review skill modifies no code file on the closeout track.
- Invariant: readiness and coverage close no gate and block no status transition.
- Invariant: `describe.read` reads a feature file as evidence; it copies no feature file into `.archcore/` (a copy is a second canon).

## Failure Behavior

1. IF a scoped `spec` has no numbered Normative Behavior clause, THEN the executing skill MUST report coverage as not computable for that spec.
2. IF no runner and no confirmation exist for a scenario, THEN the executing skill MUST report every example as unconfirmed.
3. IF a cited feature file cannot be read, THEN the executing skill MUST report the path and continue.
4. IF the compatibility probe returns other than `yes`, THEN the executing skill MUST keep the legacy scope filter and skip both checks.

## Conformance

An implementation is conformant when the describe and closeout gate records carry the surfaces above, behaviors 1–15 hold, and the failure rules produce the stated outcomes. Regression coverage: `@test/structure/track-goldens.bats` (gate record shape) and the `review` closeout golden in `@test/structure/` [planned]; live sessions are not exercised.

Given a branch with one `scenario` and its `spec`, When `closeout.verify` runs, Then the report lists each example's readiness and each uncovered clause number, and both findings are advisory.
