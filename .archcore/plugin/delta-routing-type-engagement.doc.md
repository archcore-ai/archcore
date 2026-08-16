---
title: "Delta-Routing Type Engagement — 19-Type Producer Verification Matrix"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Overview

This matrix verifies that every shipped document type keeps at least one producer under delta routing (ΔΠMR). Verdict: 19 of 19 types retain a producer; no type is lost. What changes is the engagement condition — today a type fires when its track gate opens and `skip_when` finds no covering document; under delta routing a type fires when the computed Δ, Π, M, R values call for it. "Producer today" cells cite the track files under `@plugins/archcore/skills/_shared/tracks/`.

## Content

| Type | Producer today | Engagement under delta routing | Change |
|---|---|---|---|
| `idea` | `sdd.frame` — default first artifact of every sdd run | concept instrument, engaged by the high-uncertainty portfolio (pressure-test before commitment) | narrowed |
| `prd` | `sdd.require` — every full sdd run | umbrella route (`creates` ≥ 2); capability route with `intent_gap` present | narrowed |
| `spec` | `sdd.design`; `decision.cascade` architecture branch; `describe.draft` | capability and umbrella routes — one per capability; amendment target for `modifies`; describe path unchanged | widened |
| `plan` | `sdd.decompose`; `decision.cascade` architecture branch | every executable route; additionally carries the declared Δ and route rationale | widened |
| `rnd` | `research.frame` | research instrument (Π `world`); new spike form (Π `empirical`, Goal and Questions only) | widened |
| `adr` | `decision.adr`; `decision.resolve` | decision route (`decision` delta); Π `undecided`; a refactor's decision delta | widened |
| `rfc` | `decision.rfc` | decision instrument; solution-shape uncertainty (judged panel of alternatives) | unchanged |
| `rule` | `decision.cascade` standard branch | same producer — the `/archcore:document` path is untouched; input role at grounding stays (conductor reads rules as constraints) | unchanged |
| `guide` | `decision.cascade` standard branch; `describe.draft` | same producers, plus: runbook instrument — package member when a capability's delta introduces an operational procedure, or when R carries `data-migration`; capture target for human-actor procedures at the experience offer and at plan discharge | widened |
| `doc` | `describe.draft` | same producer | unchanged |
| `mrd` | `requirements-cascade.mrd`, sources mode | acquisition instrument — product-scale `intent_gap`, or expert invocation | narrowed |
| `brd` | `requirements-cascade.brd`, sources mode | acquisition instrument — same condition | narrowed |
| `urd` | `requirements-cascade.urd`, sources mode | acquisition instrument — same condition | narrowed |
| `brs` | `requirements-cascade.brs`, iso mode | iso chain link on a `security-compliance`-flagged capability; full chain via expert invocation | narrowed |
| `strs` | `requirements-cascade.strs`, iso mode | iso chain link — same condition | narrowed |
| `syrs` | `requirements-cascade.syrs`, iso mode | iso chain link — same condition | narrowed |
| `srs` | `requirements-cascade.srs`, iso mode | iso chain link — same condition | narrowed |
| `task-type` | `experience.offer` | unchanged producer; new consumption — routing input that de-escalates Π; discharge target for a completed `plan` with an agent-actor procedure | widened |
| `cpat` | `experience.offer`; `decision.cascade` opt-in | unchanged producers; new consumption — routing input that de-escalates Π | widened |

Legend: narrowed — the type fires under a stricter computed condition than today's gate order; widened — the type gains a producer, a form, or a consumption role; unchanged — producer and condition survive as they are.

Consumption-side changes the matrix does not show:

- `skip_when` conditioned on document-graph state stops being the routing basis; graph state still closes gates inside an engaged instrument.
- The `brd`/`urd` composition into `sdd.require` (Goals from `brd` metrics, Requirements from `urd` acceptance criteria) survives inside the intent instrument unchanged.
- `task-type` and `cpat` move from output-only to routing inputs — the first types the conductor reads, not only writes at review.
- `guide` gains its first vision-command production path: the runbook instrument adds it to an assembled package, where today no `plan`-command track produces a `guide`.
- The `guide`-versus-`task-type` boundary is the procedure's actor: human → `guide`, agent → `task-type` (instrument-layer spec, behavior 17).

## Examples

- Issue #25 (OpenCode adapter): `creates` = [opencode-adapter], R = [`external-contract`] → capability route raised to `L`: one `spec`, one `plan`, no `prd` — intent is already recorded by the host-expansion documents. The adapter's local-testing procedure is an operational procedure, so the package adds one `guide` — the artifact the corpus today holds as `codex-local-plugin-testing.guide.md` for the sibling adapter. Types engaged: `spec`, `plan`, `guide`.
- "Fix the Safari button overflow": all Δ lists empty, no `intent_gap` → `null` route. Types engaged: none — the case ~30% of the 40 bench traces resolve to.