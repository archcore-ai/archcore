---
title: "Format Prices With a Space Thousands Separator"
status: accepted
tags:
  - "billing"
---

## Context

Invoices printed by @src/utils/formatPrice.ts went to customers in three countries, and support received complaints about commas read as decimal points.

## Decision

Adopted a space as the thousands separator and a comma as the decimal separator in `formatPrice`.

## Alternatives Considered

1. `Intl.NumberFormat` with the customer locale — rejected because invoices must look identical in every country.
2. A dot decimal separator — rejected because the finance team reconciles with comma decimals.

## Consequences

- Prices read the same on every invoice.
- [expected] A locale-specific format needs a new decision.
