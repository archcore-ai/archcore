---
title: "API Error Envelope & Wrapping"
status: accepted
tags: [backend]
---

## Rule

Handlers return the JSON envelope `{ "error": { "code", "message" } }`.
Internally, errors are wrapped with `fmt.Errorf("...: %w", err)` so the chain
survives to the log. Map known errors to a stable `code`; unknown errors become
`internal` with a 500 and a logged `request_id`.

## Rationale

Consistent shape for clients, traceable chain for operators.
