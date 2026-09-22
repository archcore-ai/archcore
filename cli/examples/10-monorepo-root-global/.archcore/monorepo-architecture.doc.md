---
title: "Monorepo Architecture"
status: accepted
---

## Overview

A single repository holding multiple deployable apps and shared libraries:

```
apps/      web (React SPA), api (Go service)
packages/  ui (shared component library)
```

Cross-cutting standards for the whole monorepo live in **this root
`.archcore/`**. Every app and package mounts it as a read-only global, so the
rules are defined once at the root and inherited everywhere.
