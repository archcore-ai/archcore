---
title: "Shared Conventions (monorepo-wide)"
status: accepted
---

## Rule

Applies to every app and package in this repo:

- TypeScript everywhere, from one shared `tsconfig` base.
- Conventional Commits for history.
- Each project owns its tests; CI runs them per affected project.

## Rationale

Defining these once at the root keeps every project consistent without copying
rules into each one.
