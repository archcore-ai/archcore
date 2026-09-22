---
title: "Component & File Naming"
status: accepted
tags: [frontend]
---

## Rule

React components are PascalCase, one per file, file named after the component
(`UserMenu.tsx`). Hooks are camelCase with a `use` prefix (`useUserMenu.ts`).
Co-locate styles and tests next to the component.

## Rationale

Predictable names make a component greppable and its tests/styles obvious.
