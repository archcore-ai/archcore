---
title: "Delta-Routing Model (ΔΠMR) — Conductor, Instruments, and Lifecycle for the Track Layer"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Idea

Replace the sdd track's fixed cascade (frame → require → design → decompose) with a computed route. Grounding derives four objects: Δ — the canon delta (capabilities created, modified, retired; decision delta; intent gap), Π — the gap profile (for each information need, its source: machine, user, world, undecided, empirical), M — the maturity of the touched canon zone, R — risk flags (external contract, data migration, security/compliance, irreversibility, multi-team). A conductor assembles the document set, the acquisition steps, and the strictness from these objects. The track layer decomposes into three kinds of parts: the conductor (routing computation in the `plan` skill), instruments (single-type producers: the four sdd gates, decision, research, describe, the sources set, the iso chain links), and lifecycle sequences that stay ordered (closeout, actualize, experience). Beneficiary: any team whose work spans zero-document fixes to multi-capability initiatives — today both ends misroute into one four-document cascade.

## Value

- A null route exists: zero-delta work (layout fixes, flaky tests, tooling) exits with no documents — the leading abandonment cause of SDD tooling per the routing-bases research.
- Multi-spec decomposition is native: creates ≥ 2 licenses one spec per capability under an umbrella prd; today one topic yields one spec regardless of scope.
- Bugfix and refactor become delta cases, not new tracks: a bugfix resolves through the code-wrong/spec-wrong verdict against the covering spec; a refactor is a zero-capability delta whose decision delta still licenses adr and cpat.
- iso mode becomes a strictness protocol: the compliance flag escalates intent production into the brs → strs → syrs → srs chain for the flagged capabilities only — partial formality, closer to ISO 29148 tailored conformance than the current all-or-nothing mode. Sources mode becomes an acquisition instrument for product-scale intent gaps; the sources/iso mode selection disappears.
- Command symmetry: plan declares Δ (future), document records Δ that happened without a plan (present), review reconciles declared Δ against the diff and reads drift as unplanned Δ (past). One vocabulary across three commands.
- The experience layer becomes a routing input: task-type and cpat precedents de-escalate Π; every accepted capture improves future routing.
- The plan document carries the declared Δ and the route rationale, so closeout reconciles a structured object instead of prose.

## Possible Implementation

- Conductor in the plan skill: staleness precondition on the touched zone → Δ declaration → Π profile → M and R → one-line route announcement (route plus reasons) → instrument invocations. An expert invocation naming a route overrides computation; the conductor never asks the user to choose a route.
- Output table: Δcap empty ∧ Δdec empty ∧ no intent gap → null exit; Δdec only → decision instrument; creates = 1 → spec + plan, plus prd on intent gap; creates ≥ 2 → umbrella prd + one spec per capability + plan; modifies → spec amendment through the verdict.
- Π rules: machine → compose without questions, and each machine claim cites its artifact; user → interview under the existing ceiling; world → research instrument; undecided → decision instrument; empirical → spike — a timeboxed rnd holding Goal and Questions only, throwaway code, re-entering scope with a Δ on success.
- High-uncertainty portfolio: Π decomposes uncertainty by kind, and the conductor composes several instruments instead of funneling into one heavy track. Concept: pressure-test the idea until it hardens or dies (forge/PRFAQ pattern). Feasibility: spike, or a walking skeleton — the thinnest end-to-end slice proving the architecture. Solution shape: rfc, or a judged panel of independent alternatives. Market and user at product scale: the sources instruments. Compliance: iso links on the flagged capabilities. Two assembly principles: cheapest kill shot first (conversation, then research, then spike, then formal document), and portfolio over funnel — no single track owns a high-uncertainty initiative.
- Guide activation: a runbook instrument adds a `guide` to the package when a capability's delta introduces a human-performed operational procedure (install, migrate, operate, verify) or the data-migration flag is raised; the experience offer and plan discharge capture human-actor procedures as `guide`, agent-actor procedures as `task-type`.
- Instruments keep the gate-contract record shape; the `Next:` chaining between gates of different types moves to the conductor. describe and actualize gain callable modes with pre-filled scope, following the reverse-direction precedent in track-layer.spec rule 12.
- The code-wrong/spec-wrong verdict moves to a shared contract consumed by the bugfix route, actualize, and closeout.
- closeout gains Δ reconciliation in verify (declared versus actual diff; the path comparison is CLI-checkable) and a discharge step: a document becomes archivable when its unique information is absorbed elsewhere — spec stays canon; plan discharges into task-type capture and history; prd holds until its success metrics verify; idea discharges after an L-cycle; a spike rnd keeps only its conclusion. Every status transition stays per-document confirmed.
- Rollout phases [assumption — split by dependency, not measured]: (1) Δ routing inside sdd.md — null exit, multi-spec, bugfix verdict; (2) Π profile replacing budget-from-vagueness; (3) M strictness and R escalators. Behavioral routing tests derived from the 40-task bench accompany each phase.
- Named follow-ups: a capability-granularity contract under `_shared/` (anchor: behavior an external consumer relies on; split above the 80-line spec cap; [assumption] merge specs that always change together); an rfc for an `archived` status value — draft/accepted/rejected has no state for discharged documents; an MCP-boundary status-transition guard — a status change without a recorded confirmation is refused server-side — the one hardening item pulled ahead of the redesign, because drift detection, closeout, and global-source inheritance read statuses; [assumption] a soft cap creates > 5 → re-scope; the path-index rfc rises in priority as the staleness precondition's enabler; a guide content contract under `_shared/` — the reader-and-task line, actor-explicit steps, and the actor boundary against `task-type`.

## Risks and Constraints

- Four LLM-computed classifiers extend the prompt-enforced surface, and the routing-bases research shows prompt-only rules leak. Mitigations: Δ recorded structurally in the state block and shape-checked by doctor; closeout reconciliation catches understated Δ after the fact; a capability discovered at decompose outside Δ forces a Δ revision before work continues; an adversarial auditor pass re-derives Δ independently for accepted-core zones.
- Capability granularity is undetermined until the granularity contract lands; identical work can yield one or four specs. The contract is a phase-1 exit condition, not a follow-up.
- Δ computed against a stale canon misroutes; the scoped drift precondition adds entry cost. Fallback: proceed with [assumption]-marked Δ entries against staleness-flagged documents.
- Π self-assessment inherits agent overconfidence in both directions; the citation duty (machine claims name their artifact) and a recorded negative-search protocol bound it.
- Computed routes are less predictable for the user than named shapes; the route announcement line and the S/M/L/XL label vocabulary compensate.
- Route computation costs one grounding pass even for null work; two-tier grounding (session recap plus one negative search short-circuits to null) bounds it.
- No surveyed tool runs this four-axis composition; the phased rollout and bench-derived regression scenarios contain the first-mover risk.
- Constraints: no user-facing track menu (unchanged); gate-contract remains the instrument record format; the archived-status kernel change routes through its own rfc before any discharge behavior lands.