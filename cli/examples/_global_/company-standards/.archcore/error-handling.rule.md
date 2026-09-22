---
title: "Wrap Errors with Context"
status: accepted
tags: [backend]
---

## Rule

Every error crossing a function boundary is wrapped with a context message.
Never return a bare upstream error and never swallow one silently.

```ts
// good
throw new AppError("loadInvoice: fetch failed", { cause: err });

// bad
throw err;   // no context
return null; // swallowed
```

## Rationale

A wrapped chain makes a failure traceable from the log line back to the call
site without a debugger. This is the company-wide baseline — every service
follows it.
