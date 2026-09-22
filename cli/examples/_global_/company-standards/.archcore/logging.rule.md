---
title: "Structured JSON Logging"
status: accepted
tags: [backend]
---

## Rule

Logs are structured JSON, one event per line, with at least `level`, `msg`,
`service`, and `request_id`. Never log secrets, tokens, or full PII.

```json
{"level":"error","msg":"charge failed","service":"billing","request_id":"r-91a","code":"card_declined"}
```

## Rationale

Structured logs are queryable in aggregation. A stable `request_id` ties one
request across services.
