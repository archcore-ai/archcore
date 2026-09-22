---
title: "Component Structure"
status: accepted
tags: [frontend]
---

## Rule

A component folder holds the component, its hook, its test, and its styles:

```
UserMenu/
  UserMenu.tsx
  useUserMenu.ts
  UserMenu.test.tsx
```

Data fetching lives in the hook, not the JSX. Presentational components take
props only — no direct API calls.

## Rationale

Co-location keeps a feature self-contained; the hook boundary keeps rendering
testable.
