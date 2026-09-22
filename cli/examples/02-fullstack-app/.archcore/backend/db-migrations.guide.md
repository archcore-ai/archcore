---
title: "Writing a Database Migration"
status: accepted
tags: [backend]
---

## Steps

1. Add a numbered pair `NNNN_name.up.sql` / `NNNN_name.down.sql`.
2. Keep it additive: new columns nullable or defaulted; never drop a column in
   the same release that stops writing it.
3. Regenerate query code (`sqlc generate`) and update callers.
4. Test up + down on a copy of staging data before merging.

## Why

Additive, reversible migrations let a deploy roll back without data loss.
