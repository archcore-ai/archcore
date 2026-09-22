---
title: "Monorepo Conventions"
status: accepted
---

## Rule

This monorepo holds multiple apps under `apps/*` and shared packages under
`packages/*`. Cross-cutting standards live once in
`packages/shared-standards/.archcore` and are mounted by every app as a global.
Never copy a standard into an app — mount it.

## Rationale

One source of truth for shared rules; each app stays focused on its own context.
