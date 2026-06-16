---
title: "Use One-Way Push Sync from Local .archcore/ to Server"
status: accepted
tags:
  - "sync"
---

## Context

The `.archcore/` directory is the source of truth for project-level documentation. When teams use a cloud or on-prem sync server, documents need to reach the server for indexing, GraphRAG-based retrieval, and cross-project search via MCP.

We needed to decide between:

1. **One-way push** — local `.archcore/` pushes to server, server is read-only consumer
2. **Two-way sync** — bidirectional merge between local and server
3. **Server as source of truth** — server owns documents, local is a cache

## Decision

**One-way push (local → server).** The `.archcore/` directory remains the sole source of truth. The server receives documents, indexes them, and provides enhanced search — but never writes back.

## Rationale

- **Simplicity:** No conflict resolution, no merge logic, no divergence states. Push is idempotent.
- **Git-native:** Documents live in the repo, versioned by git. The server doesn't need to replicate git history or branching.
- **Offline-first:** All authoring and reading works without server connectivity. Sync is an enhancement, not a dependency.
- **CI/CD friendly:** A push from the main branch in CI is a natural fit — deterministic, reproducible, no side effects on the local repo.
- **Large repos:** Incremental sync via SHA-256 manifest (`.archcore/.sync-state.json`) ensures only changed files are transmitted, keeping sync fast regardless of total document count.

Two-way sync was rejected because it introduces merge conflicts, requires conflict resolution UI, and creates ambiguity about which version is authoritative. Server-as-source-of-truth was rejected because it breaks the git-native workflow and makes the server a hard dependency.

## Implementation

- **Manifest:** `.archcore/.sync-state.json` (gitignored) tracks per-file SHA-256 hashes and last sync timestamps
- **Diff:** Walk document directories, hash files, compare to manifest → produce created/modified/deleted sets
- **Protocol:** `POST /api/v1/projects/{id}/sync` with incremental payload; chunked for large repos
- **Auth:** Bearer token via `ARCHCORE_TOKEN` env var or `archcore login` stored credential
- **Triggers:** Manual (`archcore sync`), CI/CD (`archcore sync --ci`), or hook-based (`on-commit`)
- **Server role:** Index documents, build GraphRAG, expose cross-project MCP search tool

## Consequences

### Positive

- No conflict resolution complexity — ever
- Works fully offline; sync failures are non-destructive (manifest only updates on success)
- Server can be rebuilt from scratch by re-syncing all projects (`--force`)
- Multiple projects can sync to the same server, enabling cross-project GraphRAG search

### Negative

- Server-side edits are not possible (by design — use git for authoring)
- If `.sync-state.json` is lost, a full re-sync is needed (safe but slower)
- Deletions must be tracked by comparing manifest against current files (slightly more complex than pure additions)

### Neutral

- The local MCP tool (`archcore mcp`) continues to serve local-only search; server MCP provides the cross-project layer — both coexist