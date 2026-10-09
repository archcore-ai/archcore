---
title: "Cross-Spec Scenario — One Realized Flow May Illustrate Clauses of Several Specs"
status: accepted
tags:
  - "component:plugin"
  - "document-types"
  - "plugin"
  - "precision"
---

## Context

`@plugin/plugins/archcore/skills/_shared/scenario-contract.md` binds a `scenario` to "the clauses of one `spec`", with one `depends_on` edge to it, and `@plugin/plugins/archcore/skills/_shared/journey-contract.md` limits a `journey` to an intended path before any `spec` exists. A realized end-to-end flow that crosses several contracts fits neither: the plugin's own build cycle `/archcore:plan` → implementation → `/archcore:review` illustrates clauses of `command-surface-v2.spec`, `delta-routing-conductor.spec`, `delta-routing-instruments.spec`, and `review-durable-context-selection.spec`. On 2026-10-09 that gap forced the plugin usage catalog into a `doc`, which loses the actor-subject form and receives no cascade notice when one of those specs changes.

## Decision

A `scenario` may illustrate clauses of two or more `spec` documents — a cross-spec scenario — when its Subject names each `spec` with its clause numbers, each example's `Illustrates:` line names the `spec` and clause, and the scenario carries one `depends_on` edge to each named `spec`.

## Alternatives Considered

1. A `doc` catalog of flows — rejected because it drops the actor-subject steps and the Given/When/Then examples, and a `spec` edit reaches it only through a `related` edge the cascade notice does not read as a dependency.
2. An as-is `journey` — rejected because `journey` is a vision-category intent record; realized behavior in it would read as unbuilt intent, and the journey-to-scenario pairing in the illustrate instrument would lose its meaning.
3. One `scenario` per `spec` plus a `doc` that chains them — rejected because the continuity of the flow across contracts is the content a reader needs, and the chain would live in the one document that no cascade reaches.

## Consequences

Positive:

- An end-to-end flow keeps its actor-subject form, and every `spec` it crosses reaches it through the cascade notice. [expected]
- `closeout.verify` readiness and coverage checks apply per scenario unchanged; a clause of any linked `spec` can be marked illustrated.

Negative:

- `scenario-contract.md` and `actor-subject-content-contracts.spec` change. The CLI needs no change: on 2026-10-09 the post-write hook reported no finding for three cross-spec scenarios carrying 2, 3, and 4 `depends_on` edges to specs.
- A cross-spec scenario changes whenever any of its specs changes, so it drifts more often than a single-spec one. [expected]
- The 120-line cap still binds; a long flow splits by actor, not by `spec`.

## Superseded when

- Cross-spec scenarios exceed 30% of all scenarios and drift findings on them exceed 2 per review, showing that the flows belong in smaller units.
- The kernel adds a dedicated end-to-end flow type.
