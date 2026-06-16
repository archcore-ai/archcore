---
title: "Add a Feature Flag"
status: accepted
---

## When

Shipping a change that must be dark-launched or rolled out gradually.

## Steps

1. Register the flag with a default of `off`.
2. Gate the new path; keep the old path intact behind the flag.
3. Roll out by cohort; watch metrics between steps.
4. At 100% and stable, remove the flag and the old path in a cleanup PR.

## Done when

The flag is fully ramped and a dated cleanup task exists to delete it.
