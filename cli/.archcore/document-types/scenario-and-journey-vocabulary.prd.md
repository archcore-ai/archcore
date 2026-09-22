---
title: "Stored Scenarios and Journeys: Engine Outcomes"
status: draft
tags:
  - "document-types"
  - "mcp"
---

## Vision

Archcore stores the user-side flow and its concrete examples as two typed documents, `scenario` in knowledge and `journey` in vision, so a project keeps how a user moves through the system beside the contract that governs it.

## Problem Statement

The Product track stores no user-perspective flow. A `prd` requirement forbids trigger and response, a `spec` clause obligates the component, and a `spec` Conformance admits one example of five lines (@templates/templates.go). A team with a conversational skill and no test runner, or a system whose interaction record follows its `spec`, has no home for that content today; the flow survives only in `urd` and `strs`, on tracks projects rarely run. The gap and the type pair are recorded in `concepts/scenario-and-journey-types` and `concepts/bdd-document-type-evaluation` [global · archcore · read-only], both still `draft`.

## Goals and Success Metrics

- Registered types after release: 23 (13 vision, 8 knowledge, 2 experience); release v0.8.3 shipped 21 (@templates/templates.go).
- A `scenario` or `journey` written from the generated template reports 0 missing-section findings in the post-write hook.
- A `scenario` flow without an `Anchors:` line reports 1 finding naming the flow.
- A `scenario` whose anchors cite a source directory appears in the pre-write injection when that directory is edited, ranked below a matching `spec`. [assumption] Measured on this repository's own `.archcore/` after the first scenario exists.
- The full Go suite, the linter, and the build pass on the release commit.

## Requirements

1. An author creates a `scenario` or a `journey` through `create_document` and receives a template with the actor as the subject of every step.
2. An agent reading the server instructions decides between `journey`, `scenario`, and `spec` from one test per pair.
3. The post-write hook reports the shape defects of both types and rejects nothing.
4. A `scenario` anchored to source reaches the agent before the agent edits that source.
5. A bare `features/login.feature` mention counts as a path reference in search.
6. An older binary reading a repository with the new types skips those files and names the valid types in `status`.

## Out of Scope

Runtime gates, contracts, and the compatibility probe belong to the plugin repository. The docs site, the Cucumber recipe, and per-type caps for the other 21 types (`concepts/size-caps-measured-on-a-real-corpus`) are separate work.

## Dependencies

Release publication waits for the global RFC to reach `accepted` (its adoption step 3). Implementation and verification proceed before that on the user's instruction of 2026-09-15.
