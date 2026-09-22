---
title: "Billing-Specific Conventions"
status: accepted
---

## Rule

On top of the company standards, billing adds:

- Money is integer minor units (cents), never floats.
- Every external payment call is idempotent (idempotency key per attempt).
- Webhook handlers verify the signature before any state change.

## Rationale

These are billing-domain invariants. The company-wide rules (error handling,
logging, API versioning) come from the mounted `company-standards` global.
