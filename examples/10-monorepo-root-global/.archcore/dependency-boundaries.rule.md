---
title: "Dependency Boundaries"
status: accepted
tags: [backend, frontend]
---

## Rule

Dependencies flow one way: `apps/*` may depend on `packages/*`, never the
reverse, and packages never depend on apps. Two apps never import each other —
shared code moves into a package.

## Rationale

A directed dependency graph keeps builds fast and prevents cyclic coupling
between deployables.
