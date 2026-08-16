---
title: "Delta-Routing Bench — 40 Task Traces Through ΔΠMR"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "testing"
---

## Overview

Reference corpus for the delta-routing model: 40 real tasks — 14 from archcore-ai/plugin issues and commit history, 26 from other stacks and domains — each traced to the route the model computes. Source material for the behavioral routing tests that accompany each rollout phase. Traces recorded 2026-08-15 from the design session that produced the model; each route is the model's predicted output, not an executed run [assumption]. Verified 2026-08-15 against the implemented conductor contract by a 12-trace dry-run; the divergences it surfaced were resolved by rule amendments in the contract — a standalone `plan` contribution for `decision` and `amendment` routes with multi-task implementation, intent-gap scaling (a gap that fits one capability's purpose records in that capability's `spec`, no `prd`), and one additional size-label step when iso links engage.

First harness run 2026-08-15, four parallel conductor sessions over the 42 fixtures in @test/behavioral/fixtures/routing-bench.tsv (runner: @test/behavioral/route-bench.sh, `make test-routing-bench`): 42 of 42 route names match the fixtures. Three announcements carried internal inconsistencies at the field level — a `null` route declaring `intent_gap: yes` with a contract instrument (N+1 trace), and two settled decisions profiled as `user`-source Π — so announcement-field consistency is the runner's next scoring dimension beyond the route name.

Legend: Δ — canon delta (`∅` none, `mod` modifies accepted spec, `new×N` creates N capabilities, `dec` decision delta, `+int` intent gap). Π — dominant information source (`machine`, `user`, `world`, `undecided`, `empirical`). M — touched-zone maturity (`pencil` exploratory, `stone` accepted core, `—` no canon zone). R — risk flags, recorded in trace shorthand; the implemented vocabulary maps: security, compliance → `security-compliance`; contract, multi-host → `external-contract`; migration → `data-migration`; irreversible → `irreversibility`; multi-team → `multi-team`. Route — documents licensed.

## Content

Repository tasks (archcore-ai/plugin):

| Task | Δ | Π | M / R | Route |
|---|---|---|---|---|
| #18 README before/after diagram | ∅ | machine | — | null |
| #20 Remove stale test references | ∅ | machine | — | null |
| #19 CI size guard for demo.gif | ∅ | machine | — | null |
| #12 Security test: no env values in config doc | ∅ | machine | R: security | do + rule capture on recurrence |
| #21 Run probe protocol on four hosts | ∅ | machine | stone | execute existing spec + refresh dated records |
| #2 Codex marketplace install failure | mod? | machine | stone | verdict: no covering spec → fix + spec capture |
| #4 Block shell writes to .archcore/ | mod | machine | stone, R: contract | strict spec amendment + plan |
| #15 Empty-state detection plugin→CLI | mod + dec | machine | stone | adr + two spec amendments + plan |
| #13 Re-sync imported instruction docs | new×1 | machine | pencil | light spec + plan |
| #25 OpenCode host adapter | new×1 | machine + world | stone, R: multi-host | rnd + spec + plan |
| #24 Copilot host adapter (shipped) | new×1 + dec | machine + world | stone | adr + spec + plan (matches recorded history) |
| #7 Hard enforcement via fs immutability | new×1 + dec (contested) | world + undecided | stone, R: irreversible | rnd → rfc → spec + plan after verdict |
| #22 Import-graph domain analysis | new×1 | empirical | pencil | spike first, spec on success |
| #10 Context-injection telemetry | new×1 | empirical | pencil | rnd/experiment first |

Cross-domain tasks:

| Task | Δ | Π | M / R | Route |
|---|---|---|---|---|
| Button misaligned in Safari | ∅ | machine | — | null |
| UI copy typo | ∅ | machine | — | null |
| Stabilize flaky test | ∅ | machine | — | null |
| N+1 in order listing | ∅ (mod if NFR line recorded) | machine | — | null / NFR amendment |
| SSR hydration bug | mod?: no covering spec | machine | pencil | fix; offer spec capture |
| Crash on device rotation | mod?: no spec | machine | — | fix, null |
| CRA → Vite migration | ∅ + dec | machine | — | adr + plan |
| ORM swap | ∅ + dec | machine | — | adr + cpat + plan |
| React major upgrade | ∅ | machine + world | — | plan + cpat for new patterns |
| Dark mode | new×1 +int | user | pencil | 1–2 questions → light spec + plan |
| i18n | new×1 +int | user | — | interview → spec + plan |
| Rate limiting on public API | mod | machine | stone, R: contract | strict spec amendment + plan |
| Pagination on endpoint | mod | machine | stone, R: contract | strict spec amendment + plan |
| Split name column | mod | machine | R: data migration | strictness +1: amendment + migration plan + runbook guide |
| Webhook subsystem | new×2–3 +int | user + machine | stone, R: contract | umbrella prd + 2–3 specs + plan |
| Push notifications | new×1 | world | — | rnd (APNs/FCM) + spec + plan |
| Design system extraction | new×N +int | machine + user | pencil→stone | prd + per-capability specs + plan |
| ML pipeline feature | new×1 | empirical | pencil | spike → spec on success |
| Training data leakage | ∅ + dec | machine | R: security | fix + rule capture |
| LLM prompt change | mod (if evals recorded) | machine / empirical | pencil | light experiment; evals as executable canon |
| CI provider migration | ∅ + dec | machine | R: multi-team | adr (+rfc on dispute) + plan |
| Canary deploys | new×1 | machine + world | — | rnd + spec + plan |
| Kubernetes upgrade | ∅ | machine + world | R: irreversible + migration | strict plan + runbook guide |
| Secret rotation after leak | ∅ | machine | R: security | not sdd: procedure → guide via describe |
| GDPR data deletion | new×1 | user + world | R: compliance | XL: formal chain on flagged capability + spec + plan |
| Cross-team SAP integration | new×1 | undecided + user | R: multi-team + contract | rfc → spec + plan |
| New product MVP | new×N +int | user + world | pencil | full L: idea/prd + specs + plan, maximum interview |
| Vector-search POC | — | empirical | pencil | spike: rnd note only |

## Examples

Non-normative worked trace, #25 OpenCode adapter: Δ = one new capability (host adapter with contract obligations); Π = machine (adapter obligations recorded in the host-adapter contract) plus world (OpenCode host facts unverified) → research instrument before contract production; M = stone (adapter contract is accepted canon); R = multi-host (→ `external-contract`). Route: rnd → spec → plan — the same route the Codex adapter followed in recorded history.

Non-normative worked trace, Safari button fix: negative canon search, no risk word, no decision → tier-1 grounding short-circuits to null; no document created; no closeout obligation.