---
title: "Document Status Transitions — Confirmed Closeout Ceremony, No Autonomous Acceptance"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Context

The status field carries three values — draft, accepted, rejected — and every track gate creates documents with `status: draft`, yet no gate in the six pre-extension tracks transitions a status: acceptance is manual `update_document` editing, and the corpus shows it (62 accepted documents, all promoted by hand). The 2026-08-07 flow-coverage audit recorded this as a finding, and the flow-patterns research maps Kiro's per-phase approval flag onto `status: accepted` while measuring that non-blocking validation degrades to decoration.

## Decision

A document status changes only through an explicitly confirmed action: `closeout.accept` transitions draft → accepted after a per-document user confirmation, `decision.resolve` sets an rfc to accepted or rejected on the recorded verdict, and every other transition stays a direct user-driven `update_document` — hooks, subagents, and unconfirmed gate steps MUST NOT change a status.

## Alternatives Considered

1. Auto-accept on merge (OpenSpec-style archive without confirmation) — rejected because the actualize track's temporal check treats `accepted` as verified canon (30-day draft flag, TODO scan on accepted documents), so unconfirmed promotion would grade unreviewed content as canon, and the flow-patterns research records non-blocking validation degrading to decoration.
2. Hook-driven transitions (PostToolUse promotes a document when `archcore doctor` passes) — ruled out because the plugin architecture holds the invariant that staleness detection never modifies a document autonomously, and every PostToolUse handler exits 0 as an advisory.
3. Richer status vocabulary (proposed / superseded per MADR 4.0) — deferred because the CLI schema pins three values and no consumer — grounding filters, hooks, dashboards — distinguishes more states yet.

## Consequences

- Positive: draft → accepted becomes a gated, auditable ceremony at feature completion; the health dashboard's draft count starts to measure genuinely unfinished work. [expected]
- Positive: the rfc lifecycle closes — an accepted proposal leaves an `adr` trail through `extends`, a rejected one is queryable as `status: rejected`. [expected]
- Negative: closeout adds one confirmation per draft document at feature completion, the same per-document cost `actualize.fix` already carries.
- Negative: bulk acceptance of a large init seed (up to 40 hotspot specs at deep depth) stays manual until a batch confirmation form exists. [expected]

## Superseded when

- The CLI gains a machine-validated acceptance gate — a doctor check promoted to blocking that verifies a document against code without user confirmation.
- The status schema grows beyond draft / accepted / rejected.
