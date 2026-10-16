---
title: "An Open Choice Offers Three Exits, and a Lone Question Without Evidence Goes to Investigation"
status: draft
tags:
  - "commands"
  - "component:plugin"
  - "plugin"
  - "skills"
---

## Context

`decision.classify` in @plugin/plugins/archcore/skills/_shared/tracks/decision.md read "should we" wording as an open proposal and asked the user to confirm the RFC branch, a yes/no question that leans to `rfc`. The same gate named two defaults: `adr` "when no answer marks it open" and `rfc` for an open choice under `/archcore:document`. On 2026-10-16 the fresh held-out case F12 of @plugin/test/behavioral/fixtures/matrix-holdout-fresh.tsv (`should we pre-generate image thumbnail variants instead of resizing on the fly?`, no facts, no reviewer) produced an `rfc` in 2 of 3 `claude-haiku-5-5` calls and 3 of 3 `claude-sonnet-5-5` calls, while verification case 12 of `context-driven-skill-routing-rollout.plan` writes nothing for a request without material. Some users skip the `rfc` and record an `adr`; others want an `rfc` first and an `adr` after it.

## Decision

`decision.classify` asks one question with three exits for an open choice — record an `adr` now, propose an `rfc` first (a later "we accepted the proposal" turns it into an `adr` that extends the `rfc` through `decision.resolve`), or investigate first through `/archcore:plan` — and recommends one from the evidence: a named reviewer, a disagreement, or proposal wording recommends `rfc`; no recorded facts and no named reviewer recommends investigation; "you decide" takes the recommendation. A request that states the chosen option creates exactly one `adr` and no `rfc`, with no exit question.

## Alternatives Considered

1. Keep `rfc` for every "should we" proposal — rejected because an `rfc` written from a bare question with no facts carries a motivation the agent infers, and some users want to skip the `rfc`.
2. Route a "should we" request without material to `/archcore:plan` in every case — rejected because a user who wants a proposal for review, or an `adr` now, loses that path.
3. Ask the question with no recommendation — rejected because "you decide" still needs a defined result, and the bench harness answers every question that way.
4. A fourth exit that creates an accepted `rfc` and its `adr` in one invocation — rejected because on `claude-haiku-5-5` the settled-decision case D05 then created an `adr` and an `rfc` in 2 to 3 of 6 calls, against 6 of 6 correct calls on the HEAD contracts; the `rfc` → `adr` path through `decision.resolve` already gives both documents.

## Consequences

Positive:

- The user chooses the `adr` path or the `rfc` path in one question; the `rfc` → `adr` path through `decision.resolve` stays.
- On `claude-haiku-5-5`, 6 repetitions on 2026-10-16: F12 6 of 6, D05 6 of 6, D06 5 of 6 calls correct; on the HEAD contracts D06 was 0 of 6.

Negative:

- matrix own case D06 and document-bench row 2 change their expected result from `rfc` to no document.
- Every open-choice request in `/archcore:document` costs one question, inside the 5-question ceiling.
- The recommendation rule depends on the agent spotting a named reviewer or recorded facts in the request; a missed reviewer sends a proposal to investigation. [assumption]
