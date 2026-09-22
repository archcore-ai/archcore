---
title: "Add a REST Endpoint"
status: accepted
tags: [backend]
---

## When

A new resource or action is needed under the versioned API.

## Steps

1. Route + types (plural noun, versioned prefix).
2. Handler returning the shared error envelope.
3. Migration if it needs storage.
4. Handler test: happy path + one validation failure.
5. Client hook for the frontend.

## Done when

Tested, documented, and consistent with existing API conventions.
