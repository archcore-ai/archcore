---
title: "Server State in React Query, UI State Local"
status: accepted
tags: [frontend]
---

## Context

Mixing server data and UI state in one global store caused stale data and
hard-to-trace updates.

## Decision

Server state lives in React Query. UI-only state (modals, form drafts) stays in
component state or a small store. The two never mix.

## Consequences

Caching and refetching are handled by React Query. See
@.archcore/frontend/component-structure.rule.md for how components consume it.
