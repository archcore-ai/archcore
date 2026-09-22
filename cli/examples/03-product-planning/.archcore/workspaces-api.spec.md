---
title: "Workspaces API Contract"
status: draft
tags: [product, backend]
---

## Scope

Endpoints for creating workspaces and managing membership. Implements
@.archcore/team-workspaces.prd.md.

## Contract

- `POST /api/v1/workspaces` → create; caller becomes owner.
- `POST /api/v1/workspaces/{id}/invites` → owner/admin only.
- `GET /api/v1/workspaces` → workspaces the caller belongs to.

## Open questions

Per-workspace billing or account-level? Deferred until after beta.
