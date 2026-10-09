---
title: "API Handlers Validate Request Bodies"
status: accepted
tags:
  - "api"
---

## Rule

1. WHEN a handler in @src/api/ receives a request body, the handler MUST validate it before use.

## Rationale

Untrusted input reaches billing totals through the handlers.

## Examples

### Good

A handler that checks `body.lines` is an array before it calls `invoiceTotal`.

### Bad

A handler that passes `body.lines` straight to `invoiceTotal`.

## Enforcement

Code review.
