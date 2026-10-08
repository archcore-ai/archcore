---
title: "An Explicitly Named Document Type Wins Over Routing — Every Kernel Type Reachable on Request"
status: draft
tags:
  - "commands"
  - "component:plugin"
  - "document-types"
  - "plugin"
  - "skills"
---

## Context

The kernel accepts all 23 document types through `create_document` (`@cli/internal/mcp/tools/create_document.go`), and the post-write hook only reports findings. The skills reach fewer: `/archcore:document` produces no `journey` (`journey-produced-only-on-plan.adr`, `command-surface-v2.spec` behavior 29), and the describe track produces only `spec`, `doc`, `guide`, and `scenario` (`@plugin/plugins/archcore/skills/_shared/tracks/describe.md`). On 2026-10-09 a user asked for use cases and journeys of the plugin itself; the agent, following the repository rule to load the type contract first, declined the types and produced a `doc` instead. The accepted ADR names its own cost — "a `journey` with no `prd` has no path" — and `AGENTS.md` ranks an explicit user requirement above any type contract.

## Decision

WHEN a user's request names one of the 23 kernel document types in the subject text, the executing skill composes that type under its content contract, reports a failed routing condition (for example a `journey` where a covering `spec` exists) as one line in the result without asking, and `/archcore:document` gains this path for every type, including a `journey` with no `prd`.

## Alternatives Considered

1. Keep the routing as the only path (status quo) — rejected because a kernel type becomes unreachable by direct request, and the agent overrides an explicit requirement that `AGENTS.md` ranks first.
2. A `document journey` entry only — rejected because it fixes one type and leaves `idea`, `prd`, `cpat`, and `task-type` unreachable from `document` by the same mechanism.
3. A new mode word such as `document type <name>` — rejected because `command-entry-grammar.adr` removed type-name entries from the argument hint; the subject text already carries a named type under `command-surface-v2.spec` behavior 28.
4. Ask a confirmation question when the routing condition fails — rejected because it spends 1 of the 5-question ceiling on a choice the user already made; a report line keeps the information without the cost.

## Consequences

Positive:

- Every kernel type is reachable by naming it; the user's explicit choice is honored. [expected]
- The grammar stays unchanged: the mode remains the first word, and the type sits in the subject.

Negative:

- More documents of a poorly fitting type, for example an as-is `journey`; the report line and the next review's type-fitness check are the only guards. [expected]
- `journey-produced-only-on-plan.adr` is superseded; `command-surface-v2.spec` behaviors 28–29 and its constraint "a `journey` is produced only at `sdd.require` on `plan`" change.
- The `command-entry-grammar.scenario` example "Asking for a journey through document" inverts its expected result.
- The write affinity `document` → knowledge types gains an exception for an explicitly named type of another category.

## Superseded when

- The review bench records more than 1 explicitly requested document per 10 that the next review reclassifies as the wrong type.
- The kernel removes or merges document types so that the routing again reaches all of them.
