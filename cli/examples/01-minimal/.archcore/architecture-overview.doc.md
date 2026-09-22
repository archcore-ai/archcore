---
title: "Architecture Overview"
status: accepted
---

## Overview

A small fullstack web app: a React single-page frontend talks to one Go HTTP
API, which owns a single PostgreSQL database. No queue, no microservices — a
deliberately simple monolith.

```
[ React SPA ] --HTTPS--> [ Go API ] --SQL--> [ PostgreSQL ]
```

One container per tier. See `tech-stack.rule.md` for the fixed choices and
`error-format.adr.md` for the API error contract.
