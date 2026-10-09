---
title: "Test Runner Configuration"
status: accepted
tags:
  - "build"
  - "testing"
---

## Overview

@vitest.config.ts at the repository root configures the test runner for every package. The platform team owns it.

## Content

| Setting | Value | Why |
|---|---|---|
| `environment` | `node` | No test needs a DOM. |
| `testTimeout` | 5000 | CI agents are slow on cold caches. |
| `include` | `src/**/*.check.ts` | Unit tests use the `.check.ts` suffix. |

Every change to @vitest.config.ts carries the comment `// reviewed-by: platform` on the line above the changed setting. The platform team searches for this comment when it audits configuration changes.

## Examples

```ts
    // reviewed-by: platform
    testTimeout: 8000,
```

Doc marker: CTXB-VITEST-CONFIG.
