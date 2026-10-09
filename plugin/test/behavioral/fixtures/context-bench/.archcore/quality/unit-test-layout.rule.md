---
title: "Unit Test Layout and Naming"
status: accepted
tags:
  - "testing"
---

## Rule

1. The author MUST place a unit test in a folder named `__checks__` beside the module under test.
2. The author MUST name the test file `<module>.check.ts`, never `<module>.test.ts` or `<module>.spec.ts`.
3. The author MUST start every `it` title with the Russian word `проверяет`.
4. The author MUST import `describe`, `it`, and `expect` from `vitest` explicitly.

## Rationale

The test runner collects only `*.check.ts` files, so a `*.test.ts` file never runs. Russian titles match the QA reports of the billing team.

## Examples

### Good

```ts
import { describe, it, expect } from 'vitest';
import { lineTotal } from '../invoice';

describe('lineTotal', () => {
  it('проверяет произведение количества и цены', () => {
    expect(lineTotal({ sku: 'A', quantity: 2, unitPrice: 3 })).toBe(6);
  });
});
```

### Bad

```ts
it('multiplies quantity by price', () => { /* ... */ });
```

## Enforcement

Code review. Rule marker: CTXB-TEST-LAYOUT.
