---
title: "Add a REST Endpoint"
status: accepted
tags: [backend]
---

## When

Adding a new resource or action under `/api/v1`.

## Steps

1. Define route + request/response types per @.archcore/backend/api-design.spec.md.
2. Add the handler; wrap errors per @.archcore/backend/error-handling.rule.md.
3. Add a migration if it needs storage (@.archcore/backend/db-migrations.guide.md).
4. Write a handler test (happy path + one validation failure).
5. Add the frontend hook (@.archcore/frontend/component-structure.rule.md).

## Done when

Endpoint is documented, tested, and returns the standard error envelope.
