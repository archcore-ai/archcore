---
title: "Invoice Totals"
status: accepted
tags:
  - "billing"
---

## Purpose & Scope

This spec defines how @src/billing/invoice.ts computes invoice totals. The API handler in @src/api/createInvoice.ts depends on it.

## Surface

- `lineTotal(line)` and `invoiceTotal(lines)` in @src/billing/invoice.ts.

## Normative Behavior

1. The module MUST compute a line total as quantity times unit price.
2. The module MUST compute an invoice total as the sum of its line totals.

## Constraints & Invariants

- Invariant: an empty invoice totals 0.

## Failure Behavior

1. IF a line carries a negative quantity, THEN the module MUST still return the arithmetic product.

## Conformance

An implementation is conformant when it satisfies behaviors 1–2.
