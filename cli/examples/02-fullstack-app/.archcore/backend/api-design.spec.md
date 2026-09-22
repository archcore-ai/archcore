---
title: "REST API Contract"
status: accepted
tags: [backend]
---

## Scope

Conventions every endpoint under `/api/v1` follows.

## Contract

- Resources are plural nouns: `/api/v1/projects`, `/api/v1/projects/{id}`.
- Status codes: 200/201 success, 400 validation, 401/403 auth, 404 missing, 409 conflict.
- Errors use the envelope from @.archcore/backend/error-handling.rule.md.
- List endpoints use `?limit=` + `?cursor=` pagination; no offset paging.

## Invariants

A published `v1` endpoint never changes its response shape; additions only.
