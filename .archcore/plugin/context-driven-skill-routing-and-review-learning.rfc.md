---
title: "Context-Driven Routing for Plan, Document, and Review"
status: accepted
tags:
  - "architecture"
  - "concepts"
  - "document-types"
  - "skills"
---

## Summary

Propose context-based entry for `/archcore:plan [topic]`, `/archcore:document [subject]`, and `/archcore:review [scope]`. All three accept no arguments. A shared read-only context resolution layer selects scope and evidence before command-specific routing. Document types and tracks stay internal. Review checks implementation, completes durable contracts after code-first work, and extracts reusable experience without copying incident details.

Compatibility (recorded 2026-10-15): the mode words are retired without an alias or a deprecation window. A former mode word at the head of a request is topic text that context resolution reads as wording evidence, and the release notes list the retired forms. A leading type slug and a no-argument call keep their result class.

The accepted result lives in `context-driven-routing-beneath-command-grammar.adr`, `explicit-type-request-wins.adr`, `command-surface-v2.spec`, and `skills/_shared/delta-routing.md`. This RFC keeps the critique and the rejected alternatives.

## Motivation

The old hints exposed four plan modes, three document modes, and four review modes, yet the gated-track model says the user does not choose a track. A new entry grammar had to amend the binding command specs explicitly.

The plan conductor derived canon delta Δ, gap profile Π, maturity M, and risk R. Its package table made `creates = 1` produce a spec and plan, `creates ≥ 2` produce a PRD, specs, and plan, and a security-compliance flag engage ISO links. The four-field model was not validated against artifact quality: the recorded 42-of-42 bench result scored route names, while three announcements contradicted their own fields. A new project, an amendment, and a repair can have the same capability count yet need different records.

Review had one durable-context selection for planned and planless branches, then a separate experience track that detected patterns the selection missed. Its two-file edit-shape signal did not test whether the resulting lesson helps a later task in another area. The closeout gate retained an unfinished plan, but its report could leave the user without a clear path to removal.

## Critical assessment of Δ, Π, M, and R

This assessment compares the then-current conductor, gate files, and bench fixtures. It is an engineering inference from their contracts, not a measured claim about production success.

| Field | Useful part | Failure in the gates | Proposed treatment |
|---|---|---|---|
| Δ | A declared change can be checked against the implementation at closeout. | Capability count selects documents before the need for each document is known. `intent_gap` is a Boolean inside a change record although it describes missing intent. A `null` route can still need a multi-step execution plan. | Keep a claim-level declared change ledger for plan and review; stop using capability count as an artifact switch. |
| Π | Listing unresolved questions before writing prevents unsupported claims. | Its values mix locations (`machine`, `world`), a person (`user`), decision state (`undecided`), and a verification method (`empirical`). One question may require several sources. An unresolved Δ field is assigned to `user` by default even when a machine or world source could answer it. The `undecided → decision` mapping enters a gate whose default is the settled ADR branch unless an open proposal is confirmed. | Record question kind, evidence state, candidate sources, and next verification action separately. |
| M | Accepted claims and external dependents constrain a change. | `stone` merges acceptance, dependency, and freshness into one bit. Any stone zone raises a size label. The label was reported and persisted, but did not select a stronger gate check. | Read authority, coverage, freshness, and dependents as separate facts; retire `pencil/stone` and the size escalator. |
| R | External contracts, migration, security, reversibility, and coordination deserve explicit checks. | One flag and several flags have the same size effect. `security-compliance` combines distinct concerns and automatically engages ISO links; `data-migration` can add a guide without testing whether a reusable human procedure exists. | For each evidenced hazard, name its consequence, required check, and owner. Engage an artifact only if that check needs a durable owner. |

Counterexamples already appeared in the route fixtures. A React major upgrade and a Kubernetes upgrade were labeled `null` despite multi-step migration work. An SSR hydration bug (`amendment`) and a device-rotation crash (`null`) differed only by whether the agent judged the behavior consumer-relied, a boundary that needs a concrete owner and reader test. One fixture engaged ISO links from a compliance flag, although it did not establish which formal requirement levels the work needs.

The bench supplied the routing contract and an already established grounding summary to the model, and checked the one-line response and route name only. It showed the model can reproduce route labels under supplied grounding. It did not establish correct grounding, artifact selection, question choice, risk treatment, or end-to-end completion.

### Replacement decision procedure

1. Run a cheap relevance check first: find applicable accepted constraints and a matching open draft. Expand code and graph reads only for a material candidate or conflict.
2. Record intended or observed change as claim-level units: subject and consumer; baseline owner; proposed or observed claim; evidence; verification; and `create`, `amend`, `retire`, or `no durable change`.
3. Record every material unknown with separate fields: question kind (`fact`, `choice`, `feasibility`, or `acceptance`); evidence state (`supported`, `missing`, or `conflicted`); candidate sources; and next action. A source can be plural.
4. Resolve authority before writing: accepted local and global claims, exact coverage, freshness evidence, and affected dependents. A code/doc conflict does not become `spec-wrong` solely because the code is newer.
5. Record each hazard as trigger → consequence → required verification → responsible owner. A security concern can require a security check without any ISO document. An explicit formal traceability obligation can engage the needed ISO links.
6. Run the document-need test for each claim: future reader task, missing content kind, owning type or document, evidence, and non-duplication. The result can be zero documents.
7. Produce an execution `plan` when tasks, dependencies, coordination, or resumption warrant it, even when no capability contract changes. Preserve a structured `Declared Delta` section for closeout, but store claim-level outcomes instead of using its count as a route.
8. Let each selected instrument run its content gates. An existing document skips creation only when it covers the actual claim; existence on the same topic is insufficient.

The procedure replaces ΔΠMR as a four-field routing API. It retains change reconciliation, unknown tracking, constraint reading, and hazard checks, without four classifications on every invocation or `S/M/L/XL` labels that change no behavior.

### Failure modes of the replacement

The replacement can fail in the opposite direction. An agent can invent a "future reader task" to justify any document, split one change into many artificial claims, or expand repository reads without improving the decision. A document-need result therefore requires a named consumer or an observable use in the request, code, or existing graph. Merge claim units when they share one owner and verification. Stop grounding when another read cannot change scope, authority, artifact choice, or a required check, and record the unresolved uncertainty.

A read-only context resolver must not lose an answered question on resume. Material answers, source references, and rejected alternatives belong in the draft's existing track state or document body. A conflict between code and accepted canon requires an authority decision before a `spec-wrong` or `code-wrong` verdict. Hazard checks need an auditable mapping, not a free-form risk adjective. If no hazard is evidenced, the gate records none and does not infer compliance obligations from a keyword.

### Validation criteria

Use a held-out set whose expected outputs specify claim ownership, minimum sufficient artifacts, questions, required checks, and closeout outcomes. Evaluate both false positives and false negatives. Include paired cases that differ in one decisive fact: an internal library upgrade with and without a coordinated rollout; a security defect with and without a formal traceability obligation; a public behavior change with and without an existing covering spec; an accepted contract with code that diverges intentionally versus accidentally. Record tool reads, latency, tokens, and any unsupported claim, so a quality gain is not purchased through unbounded inspection. `delta-routing-bench.doc` records the validity limits of the runs that followed.

## Detailed Design

Entry, context resolution, no-argument continuations, and the per-command routing are shipped and specified in `context-driven-routing-beneath-command-grammar.adr`, `explicit-type-request-wins.adr`, `command-surface-v2.spec`, and `skills/_shared/delta-routing.md`. Two design points do not live elsewhere in this form.

- Experience is a transferable decision aid, not a compressed diff. Selection asks which future task will use it, what trigger and mechanism recur, what evidence supports the scope, when it does not apply, and whether an existing document already owns the lesson. An incident-specific failure belongs in the owning contract's Failure Behavior or a procedure's Pitfalls. Two similar edits alone do not prove transfer.
- Closeout reports one outcome per matched plan. For every retained plan it names the exact task or criterion, the evidence checked, the missing proof or work, and the next action. The capture-then-removal sequence and per-plan confirmation stay unchanged.

## Drawbacks

Context resolution can add reads and can be ambiguous on a dirty branch. The relevance check limits expansion, while the evidence record and one-question rule bound its effect. Removing modes reduces exact expert shortcuts; natural-language type and scope requests preserve explicit control. A former mode word at the head of a request is read as wording, so an ambiguous one (`sources`) can cost one question. Claim-level document selection is less mechanically predictable than a capability-count table, so independent artifact-level benchmarks are required.

## Alternatives

1. Keep modes but hide them from hints. Rejected: it leaves two entry grammars and makes the visible hint incomplete.
2. Remove modes only, keeping the fixed plan package table. Rejected: the same artifact mismatch remains behind a simpler command.
3. Add a new experience document type. Deferred: existing types may own transferable lessons when their scope and evidence are enforced.
4. Make every review create a spec and experience record. Rejected: private refactors and local failures can be fully explained by code, tests, or an existing owner.
5. Replace plan deletion with a new completed status. Deferred: the current kernel has no such status; the review report can first make retained-plan reasons and recovery clear.
6. Keep ΔΠMR as mandatory diagnostics while replacing only its package table. Rejected: every run still pays for four classifications, and M's size label still changes no gate check.
7. Keep the mode words and the hints and change only the mode-less path. Rejected: it keeps the second grammar this proposal exists to remove.
8. Retire the mode words through a deprecation window of two releases with a retired notice. Rejected: it keeps two entry grammars for two releases; the retired words stay readable as wording, and the release notes carry the change.
