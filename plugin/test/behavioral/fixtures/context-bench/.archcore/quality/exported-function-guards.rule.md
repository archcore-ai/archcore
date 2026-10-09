---
title: "Exported Functions Open With an Argument Guard"
status: accepted
tags:
  - "code-quality"
---

## Rule

1. WHEN an author adds an exported function, the function MUST open with an `assertArgs` call that checks every argument.
2. The author MUST import `assertArgs` from the project's `assert` module in the `lib` folder.
3. The guard MUST name the failing argument in its message.

## Rationale

Billing functions receive amounts from untyped request bodies. A guard at the first line turns a silent `NaN` total into a `TypeError` with the argument name. Functions written before this rule are left as they are until someone changes them.

## Examples

### Good

```ts
export function scale(amount: number, factor: number): number {
  assertArgs(Number.isFinite(amount), 'amount must be a finite number');
  assertArgs(Number.isFinite(factor), 'factor must be a finite number');
  return amount * factor;
}
```

### Bad

```ts
export function scale(amount: number, factor: number): number {
  return amount * factor;
}
```

## Enforcement

Code review. Rule marker: CTXB-GUARDS.
