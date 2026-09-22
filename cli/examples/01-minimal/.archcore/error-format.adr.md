---
title: "JSON API Error Envelope"
status: accepted
---

## Context

The frontend needs a predictable shape for every failed request.

## Decision

All API errors return the same JSON envelope with an HTTP status:

```json
{ "error": { "code": "invalid_argument", "message": "email is required" } }
```

`code` is a stable machine string; `message` is human-readable and may change.

## Consequences

The frontend switches on `error.code`, never on `message`. New error kinds add
a new `code`.
