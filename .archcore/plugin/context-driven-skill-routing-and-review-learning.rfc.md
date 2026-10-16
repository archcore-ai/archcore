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

This RFC is a proposal. The accepted command and routing specs in the plugin still define the current behavior. Acceptance requires local amendments before runtime changes.

Compatibility (recorded 2026-10-15): the mode words are retired without an alias or a deprecation window. A former mode word at the head of a request is topic text that context resolution reads as wording evidence, and the release notes list the retired forms, as they did for the 2026-09-16 grammar change. A leading type slug and a no-argument call keep their result class.

## Motivation

The current hints expose four plan modes, three document modes, and four review modes. Yet the accepted global gated-track model says the user does not choose a track. The plugin's command grammar, command-surface spec, conductor spec, and track-layer spec make those modes binding. A new entry grammar must amend them explicitly.

The plan conductor currently derives canon delta Δ, gap profile Π, maturity M, and risk R. Its package table makes `creates = 1` produce a spec and plan, `creates ≥ 2` produce a PRD, specs, and plan, and a security-compliance flag engage ISO links. The four-field model was not validated against actual artifact quality: the recorded 42-of-42 bench result scored route names, while three announcements contradicted their own fields. A new project, an amendment, and a repair can have the same capability count yet need different records.

Review already has one durable-context selection for planned and planless branches. It can create a missing spec after code-first work. The current experience track then detects patterns the selection missed and offers capture separately. Its two-file edit-shape signal and type offer do not test whether the resulting lesson helps a later task in another area. The closeout gate retains an unfinished plan, but its report can leave the user without a clear path to removal.

## Current gate map

The map below is based on the plugin skill and track files in the 2026-10-15 working tree. It describes current prompt behavior, not measured runtime frequency.

| Entry | Current selection | Gates and main exit |
|---|---|---|
| `plan` computed | Ground graph, Git, and code; derive Δ, Π, M, R; select `null`, `decision`, `amendment`, `capability`, or `umbrella`; compose instruments. | `sdd.frame → require → design → illustrate → decompose → runbook` are callable in conductor order, not one mandatory chain. A blocked contract choice stops before decomposition. |
| `plan sdd` | Bypass route computation. | Full intent → contract → decompose package. |
| `plan sources` | Bypass computation. | `requirements-cascade.mrd → brd → urd`; each existing owner can skip its gate. |
| `plan iso` | Bypass computation. | `requirements-cascade.brs → strs → syrs → srs`; the computed route can also engage these links from a security-compliance flag. |
| `plan research` or a world/empirical gap | Enter investigation or spike. | `research.frame → gather → conclude` yields `research` for scope coverage or `rnd` for a recommendation; `research.spike` answers one empirical question and reroutes. |
| `document` | Classify resolution, decision, code description, supplied research, or named type. | `decision.classify → adr/rfc/resolve → cascade`; `describe.read → draft → clarify`; or `research.frame → gather → conclude`. Named types outside these tracks take a direct composition path. |
| `review` with branch diff | Compare changed code and documents both ways, then match plans. | `actualize.scope → verdict → fix` for drift; each matched plan runs `closeout.verify → merge → accept → capture → discharge`; planless work runs durable-context selection; `experience.detect → offer` runs last. |
| `review drift/deep/closeout/experience` | The first word selects a specialized path; `deep` widens actualize to the corpus. | The same track gates run with the selected scope. |
| `review` on default branch or empty diff | Fall back to project health. | Counts, relation inventory, and reference checks; no branch closeout. |

There are 35 named gates across these eight tracks: `sdd` 6, `requirements-cascade` 7, `research` 4, `decision` 5, `describe` 3, `actualize` 3, `closeout` 5, and `experience` 2. The shared gate contract evaluates `skip_when` first, checks blocking and advisory exits, and stores resume state in a draft's `archcore:track` block. Auto entry has one five-question ceiling; a named mode raises per-gate limits. With no arguments, `plan` resumes a unique open draft or asks what to plan after grounding; `document` inspects Git then asks one classification question if needed; `review` uses the branch boundary or reports project health.

Source files: @plugin/plugins/archcore/skills/plan/SKILL.md, @plugin/plugins/archcore/skills/document/SKILL.md, @plugin/plugins/archcore/skills/review/SKILL.md, @plugin/plugins/archcore/skills/_shared/delta-routing.md, @plugin/plugins/archcore/skills/_shared/gate-contract.md, @plugin/plugins/archcore/skills/_shared/elicitation-contract.md, and @plugin/plugins/archcore/skills/_shared/tracks/.

## Critical assessment of Δ, Π, M, and R

This assessment compares the current conductor, gate files, and bench fixtures. It is an engineering inference from their contracts, not a measured claim about production success.

| Field | Useful part | Failure in the current gates | Proposed treatment |
|---|---|---|---|
| Δ | A declared change can be checked against the implementation at closeout. | Capability count selects documents before the need for each document is known. `intent_gap` is a Boolean inside a change record although it describes missing intent. A `null` route can still need a multi-step execution plan. | Keep a claim-level declared change ledger for plan and review; stop using capability count as an artifact switch. |
| Π | Listing unresolved questions before writing prevents unsupported claims. | Its values mix locations (`machine`, `world`), a person (`user`), decision state (`undecided`), and a verification method (`empirical`). One question may require several sources. An unresolved Δ field is assigned to `user` by default even when a machine or world source could answer it. The `undecided → decision` mapping enters a gate whose default is the settled ADR branch unless an open proposal is confirmed. | Record question kind, evidence state, candidate sources, and next verification action separately. |
| M | Accepted claims and external dependents constrain a change. | `stone` merges acceptance, dependency, and freshness into one bit. Any stone zone raises a size label. In the current runtime assets that label is reported and persisted, but does not select a stronger gate check. | Read authority, coverage, freshness, and dependents as separate facts; retire `pencil/stone` and the size escalator. |
| R | External contracts, migration, security, reversibility, and coordination deserve explicit checks. | One flag and several flags have the same size effect. `security-compliance` combines distinct concerns and automatically engages ISO links; `data-migration` can add a guide without testing whether a reusable human procedure exists. | For each evidenced hazard, name its consequence, required check, and owner. Engage an artifact only if that check needs a durable owner. |

Counterexamples already appear in the existing route fixtures. Rows 23 and 37 label a React major upgrade and a Kubernetes upgrade `null` despite multi-step migration work; the bench document marks both as legacy plan-expectation gaps. Rows 19 and 20 distinguish an SSR hydration bug (`amendment`) from a device-rotation crash (`null`) by whether the agent judges the behavior consumer-relied. That boundary needs a concrete owner and reader test, not an unreviewed label. Row 39 engages ISO links from a compliance flag, although the fixture does not establish which formal requirement levels the work needs.

The current benchmark supplies the routing contract and an already established grounding summary to the model. Its pass condition checks the one-line response and route name only. The bench document reports three field-level inconsistencies despite 42 matching route names; the fixture file now contains 44 rows. These results show the model can reproduce the route labels under supplied grounding. They do not establish correct grounding, artifact selection, question choice, risk treatment, or end-to-end completion.

### Replacement decision procedure

1. Run a cheap relevance check first: find applicable accepted constraints and a matching open draft. Expand code and graph reads only for a material candidate or conflict.
2. Record intended or observed change as claim-level units: subject and consumer; baseline owner; proposed or observed claim; evidence; verification; and `create`, `amend`, `retire`, or `no durable change`.
3. Record every material unknown with separate fields: question kind (`fact`, `choice`, `feasibility`, or `acceptance`); evidence state (`supported`, `missing`, or `conflicted`); candidate sources; and next action. A source can be plural.
4. Resolve authority before writing: accepted local and global claims, exact coverage, freshness evidence, and affected dependents. A code/doc conflict does not become `spec-wrong` solely because the code is newer.
5. Record each hazard as trigger → consequence → required verification → responsible owner. A security concern can require a security check without any ISO document; an explicit formal traceability obligation can engage the needed ISO links.
6. Run the document-need test for each claim: future reader task, missing content kind, owning type or document, evidence, and non-duplication. The result can be zero documents.
7. Produce an execution `plan` when tasks, dependencies, coordination, or resumption warrant it, even when no capability contract changes. Preserve a structured `Declared Delta` section for closeout, but store claim-level outcomes instead of using its count as a route.
8. Let each selected instrument run its content gates. An existing document skips creation only when it covers the actual claim; existence on the same topic is insufficient.

This procedure is a proposal to replace ΔΠMR as a four-field routing API. It retains the valuable operations—change reconciliation, unknown tracking, constraint reading, and hazard checks—without requiring four classifications on every invocation or printing `S/M/L/XL` labels that do not change behavior.

### Failure modes of the replacement

The replacement can fail in the opposite direction: an agent can invent a "future reader task" to justify any document, split one change into many artificial claims, or expand repository reads without improving the decision. A document-need result therefore requires a named consumer or an observable use in the request, code, or existing graph. Merge claim units when they share one owner and verification. Stop grounding when another read cannot change scope, authority, artifact choice, or a required check; record the unresolved uncertainty instead of searching indefinitely.

A read-only context resolver must not lose an answered question on resume. Material answers, source references, and rejected alternatives belong in the planning or documentation draft's existing track state or document body. The resolver can recompute candidate scope, but cannot silently discard prior user input. A conflict between code and accepted canon requires an authority decision before a `spec-wrong` or `code-wrong` verdict. Recency of code alone is not such a decision. Planning may describe a repair and its verification; code edits remain an implementation action outside the plan command's routing gate.

Hazard checks need an auditable mapping, not a free-form risk adjective: each hazard must have an observed trigger, plausible failure, chosen check, and disposition. If no hazard is evidenced, the gate records none; it does not infer compliance obligations from a keyword. If formal traceability is asserted, the record must identify the source obligation and which requirement levels it actually requires.

### Validation criteria

Use a held-out set whose expected outputs specify claim ownership, minimum sufficient artifacts, questions, required checks, and closeout outcomes. Evaluate both false positives and false negatives; a route name is insufficient. Include paired cases that differ in one decisive fact: an internal library upgrade with and without a coordinated rollout; a security defect with and without a formal traceability obligation; a public behavior change with and without an existing covering spec; an accepted contract with code that diverges intentionally versus accidentally. Record tool reads, latency, tokens, and any unsupported claim so a quality gain is not purchased through unbounded inspection.

## Detailed Design

### Entry and context resolution

1. Keep the four visible commands `init`, `plan`, `document`, and `review`; change only the latter three hints to optional free-text subject or scope.
2. Treat explicit user intent and an explicit document-type request as evidence with first precedence. No first word is reserved as a mode. An explicit type request is a leading type slug or a slug in backticks, per @.archcore/plugin/explicit-type-request-wins.adr.md; a bare type slug elsewhere in a topic is not an explicit type request.
3. Run one shared, read-only context resolution layer before selecting a track. It is not a document-producing track or a persisted gate. A later invocation recomputes it from the repository and saved draft state.
4. Collect the request and supplied materials; open `archcore:track` drafts; accepted local and global context; related documents; branch boundary and working tree; relevant code, tests, and available verification reports.
5. Emit an internal context record: command intent; scope and source; work time (future, present, or past); claim-level change units; affected consumers; owner coverage and authority; unresolved questions with evidence state and candidate sources; evidenced hazards and required checks; candidate documents; and open conflicts.
6. Apply precedence: an explicit current request and named scope; a matching open draft; accepted local canon; accepted global defaults; observed code and tests; branch signals; wording-based inference. A request conflicting with accepted canon becomes a visible conflict, not silent authority to change it.
7. Ask one focused question only when competing interpretations change the next write or review scope. Report the inferred scope before any write. Do not fabricate a topic from a dirty branch.

### No-argument entry

| Command | Unique continuation | Branch or repository signal | No reliable signal |
|---|---|---|---|
| `plan` | Resume the matching open planning draft at its earliest failed gate. | Recommend a future target from a clear request in session or a single relevant unfinished plan. A diff alone does not turn completed code into a future plan. | Ask what outcome to plan, after searching drafts and Git. |
| `document` | Resume the matching documentation draft. | Inspect changed code and supplied material, then propose the smallest durable owner for a clear present fact. | Ask what present fact or subject to record. |
| `review` | Review the named scope or current branch boundary. | Run the full branch check when a diff exists; on a clean default branch, report health and any explicit staleness signal. | Ask for a path, topic, or ref if Git cannot resolve scope. |

If several open drafts qualify, show a short list with a recommended draft and ask once. A command never resumes another command's track by writing to it.

### Plan: select missing owners, then sequence them

Replace fixed document packages with the replacement decision procedure above. For each proposed artifact, name the future reader task, missing content kind, owner, evidence, and reason an existing document cannot hold it. Do not require Δ, Π, M, and R as four fields, a route name from the package table, or a size label. Report only decisions that change an instrument, a check, or a write.

| Observed need | Candidate owner |
|---|---|
| Unsettled concept with a distinct value question | `idea` or bounded `rnd` |
| Product outcome, beneficiary, or measurable goal beyond one contract | `prd` |
| Depended-on behavior, interface, invariant, or failure response | `spec` |
| User path before a covering contract | `journey`; a confirmed example after the contract uses `scenario` |
| Settled choice with evidenced alternatives | `adr` |
| Work with meaningful phases, dependencies, or a handoff | `plan` |
| Evidence the team must gather from market, business, or users | `mrd`, `brd`, or `urd` only for the respective source question |
| Required formal traceability across requirement levels | The applicable `brs`/`strs`/`syrs`/`srs` links, only when an obligation or explicit request establishes that need |
| Repeatable human operation introduced by the work | `guide` |

A new project first establishes its scope, users, boundaries, and initial delivery steps; an existing capability amendment checks the accepted contract and changes its owner; a repair may need only a task plan or no Archcore document. These are context profiles, not user modes or closed lists of task kinds. A specific security hazard engages its required verification. It does not itself prove that an ISO cascade or extra interview is required.

The SDD label describes the relation between durable contracts and implementation. It is not a mandatory entry path. The Sources and ISO document tracks stay available as vocabulary and can be assembled when the evidence requires their content.

### Document: record the present owner

Use the same context record to classify the supplied fact: settled choice or proposal, observed contract, reference, procedure, investigation, external material, or reusable practice. Search for an existing owner before creation. An explicit type request remains possible in ordinary language, subject to its content contract. If the user supplies a finished report, file its findings; if the user asks an unanswered question, route to planning or investigation without inventing a report. If code conflicts with accepted canon, report the conflict before updating a document.

### Review: one coherent branch pass

A branch review checks both code against accepted claims and changed documents against code. It then reconciles matched plans and selects durable context from every delivered change, including changes without a plan and delivered parts of a plan that remains open. A relied-on boundary with no owner is a candidate for a `spec` through the describe instrument. Review does not create a retrospective plan.

The selection uses one batched preview for proposed writes. It can update an existing owner, create a missing contract, record an evidenced decision, or capture experience. Declined and deferred units appear with a reason. An accepted document that conflicts with code remains a conflict until the user resolves the authority question.

Closeout reports one outcome per matched plan: removable, unfinished, not verifiable, open draft, accepted-canon conflict, or removal declined/failed. For every retained plan, report the exact task or criterion, evidence checked, missing proof or work, and next action. A completed plan retains the accepted capture-then-removal sequence and its per-plan confirmation; this proposal does not silently change the deletion policy.

### Experience quality

Treat experience as a transferable decision aid, not a compressed diff. Selection asks: which future task will use it; what general trigger and mechanism recur; what evidence supports the scope; when it does not apply; and whether a current `spec`, `adr`, `rule`, `guide`, `task-type`, or `cpat` already owns the lesson.

An incident-specific failure belongs in the owning contract's Failure Behavior or a procedure's Pitfalls. A change of code practice with evidenced reason and reusable scope may become a `cpat`; a recurring agent task may become a `task-type`; human steps become a `guide`. One case can justify a lesson if the transfer mechanism and limits are evidenced. Two similar edits alone do not prove transfer. Keep case paths, commits, and narrow details as evidence references; write the reusable condition, action, and boundary in the main body. Mark inferred scope and unresolved rationale instead of generalizing them as fact. Prefer a focused update to an existing experience document.

The review's durable-context preview includes experience candidates. The separate experience offer remains only for a clearly new candidate discovered after that preview and must apply the same transfer test.

### Migration and verification

1. Add the shared context-resolution contract and amend the accepted local command-surface, track-layer, conductor, instrument, durable-context, and command-grammar records after this RFC is accepted.
2. Change the three skill and command hints, their descriptions, and the routing text together; list the retired mode words in the release notes. Remove expert-invocation budget behavior; budget follows unresolved needs and task scope.
3. Replace the ΔΠMR route table and size escalators with claim-level change, evidence, constraint, hazard, and document-need checks. Keep each instrument's content gate and draft state lifecycle.
4. Update review closeout reports and experience selection. Preserve the current explicit status and deletion confirmations.
5. Replace route-name-only scoring with independent artifact and gate outcomes. Include no-argument continuations, a new project, amendment, repair, non-capability migration, planless public contract, partial plan, unsupported generalization, and a completed plan with missing proof.
6. Measure false extra documents, missed contracts, wrong owner choices, unjustified questions, unsafe canon changes, weak retained-plan explanations, over-specific experience, MCP reads, tokens, and latency. Record model and host limits with the results.

## Drawbacks

Context resolution can add reads and can be ambiguous on a dirty branch. The relevance check limits expansion, while the evidence record and one-question rule bound its effect. Removing modes reduces exact expert shortcuts; natural-language type and scope requests preserve explicit control. A former mode word at the head of a request is read as wording, so an ambiguous one (`sources`) can cost one question. Claim-level document selection is less mechanically predictable than a capability-count table, so independent artifact-level benchmarks are required. The review may inspect more code around relied-on boundaries.

## Alternatives

1. Keep modes but hide them from hints. Rejected: it leaves two entry grammars and makes the visible hint incomplete.
2. Remove modes only, keeping the fixed plan package table. Rejected: the same artifact mismatch remains behind a simpler command.
3. Add a new experience document type. Deferred: existing types may own transferable lessons when their scope and evidence are enforced.
4. Make every review create a spec and experience record. Rejected: private refactors and local failures can be fully explained by code, tests, or an existing owner.
5. Replace plan deletion with a new completed status. Deferred: the current kernel has no such status; the review report can first make retained-plan reasons and recovery clear.
6. Keep ΔΠMR as mandatory diagnostics while replacing only its package table. Rejected: every run still pays for four classifications, and M's size label still changes no gate check.
7. Keep the mode words and the hints and change only the mode-less path. Rejected: it keeps the second grammar this proposal exists to remove.
8. Retire the mode words through a deprecation window of two releases with a retired notice. Rejected: it keeps two entry grammars for two releases; the retired words stay readable as wording, and the release notes carry the change.
