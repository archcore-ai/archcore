---
title: "Local .archcore/ is the Only Source of Truth for Sync"
status: accepted
tags:
  - "sync"
---

## Rule

The local `.archcore/` directory is the **sole source of truth** for all project documentation. The sync server is a **read-only consumer** that indexes documents for search — it MUST NEVER write back to the local directory.

## Implications

- Sync is always **one-way push** (local → server). There is no pull, no merge, no bidirectional sync.
- All authoring and editing happens locally, versioned by git. The server does not replicate git history.
- If the server's data diverges from local, **local wins**. Run `archcore sync --force` to re-push everything.
- The server can be rebuilt from scratch at any time by re-syncing all projects.
- Server-side edits are not possible by design. Use git for authoring.

## Rationale

This eliminates an entire class of problems: merge conflicts, conflict resolution UI, divergence states, and ambiguity about which version is authoritative. The sync operation becomes idempotent and safe to retry.