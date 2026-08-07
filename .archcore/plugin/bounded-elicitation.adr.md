---
title: "Bounded Elicitation Budget Replaces the One-Question Invariant"
status: accepted
tags:
  - "plugin"
  - "precision"
  - "skills"
---

## Context

The current command spec caps every command at one scope question, which makes requirement-stage documents (prd, mrd/brd/urd, brs/strs/syrs/srs) impossible to produce credibly — their content lives in the user's head, not in code. Market evidence (2026-08-04 research) shows both failure poles: Spec Kit's fixed 5-question quota is too few for novel design and too many for a rename (https://github.com/github/spec-kit/blob/main/templates/commands/clarify.md), while the grilling skill's 16–50 questions produce no artifact discipline (https://github.com/mattpocock/skills).

## Decision

Adopted a shared elicitation budget — a hard ceiling of 5 user questions per command invocation in auto mode, drawn down across all gates of the invocation, with expert invocation raising per-gate budgets up to the track's declared maximum — shipped as `skills/_shared/elicitation-contract.md`.

## Alternatives Considered

1. Keep the one-scope-question invariant — rejected because the requirements tracks are unusable under it; the interview is the product for sources/iso cascades.
2. Per-gate stacking budgets (3+5+3 per run, 11–14 total) — rejected because stacked totals reproduce measured SDD fatigue: 7–9× doc-to-code line ratio (https://blog.scottlogic.com/2025/11/26/putting-spec-kit-through-its-paces-radical-idea-or-reinvented-waterfall.html) and 181,040 vs 91,729 tokens at lower success (https://marmelab.com/blog/2025/11/12/spec-driven-development-waterfall-strikes-back.html).
3. MCP Elicitation protocol or a hard AskUserQuestion dependency — ruled out because both are host-gated and unavailable to subagents; prose questions are the portable canonical form.

## Consequences

- Requirements tracks become usable by non-experts: guided interview, recommended default on every question, "you decide" escape recorded as `[assumption]`.
- Every accepted answer persists in the artifact's `## Clarifications` at gate close, surviving session loss.
- The auto-mode ceiling of 5 limits depth; reaching a track's full question maximum requires explicit expert invocation.
- The one-question invariant in the command spec is superseded; the spec rewrite lands in the same release.

## Superseded when

- Users abandon interviews before question 3 in observed sessions ([EVIDENCE REQUIRED] — no telemetry exists today).
- All 8 registry hosts ship a portable interactive-question primitive usable by skills and subagents.