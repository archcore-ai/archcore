---
title: "Local .archcore/ is the Only Source of Truth for Sync"
status: accepted
tags:
  - "sync"
---

## Rule

1. The local `.archcore/` directory MUST be the only source of truth for project documentation.
2. The sync server MUST NOT write back to the local `.archcore/` directory. It consumes documents and indexes them for search.
3. The CLI MUST push in one direction only, from local to server. No pull, no merge, and no bidirectional sync exists.
4. IF the server's data diverges from the local directory, THEN the operator MUST resolve the difference in favor of local by running `archcore sync --force`.

## Current behavior

- Authoring and editing happen locally, versioned by git. The server does not replicate git history.
- The server can be rebuilt from scratch by re-syncing every project.
- Server-side editing is unsupported by design.

## Rationale

A single authoritative copy removes merge conflicts, conflict-resolution interfaces, divergence states, and any question about which version wins. Sync becomes idempotent and safe to retry.

## Enforcement

- `Client` in `@internal/api/client.go` exposes `CheckHealth` and `Sync` only. No client method downloads documents from the server.
- Code review: the reviewer rejects any change that introduces a server-to-local write path.
