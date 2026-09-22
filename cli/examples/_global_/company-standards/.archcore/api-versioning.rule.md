---
title: "API Versioning via URL Prefix"
status: accepted
tags: [backend]
---

## Rule

Public HTTP APIs are versioned with a URL prefix: `/api/v1/...`.
Breaking changes ship under a new prefix (`/api/v2`); the previous version stays
until all clients migrate. Never break a published `vN` contract in place.

## Rationale

Clients pin a version explicitly. Additive changes are safe within a version;
removals and renames require a new one.
