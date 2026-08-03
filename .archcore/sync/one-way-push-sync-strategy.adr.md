---
title: "Use One-Way Push Sync from Local .archcore/ to Server"
status: accepted
tags:
  - "sync"
---

## Context

The `.archcore/` directory is the source of truth for project-level documentation. WHEN a team runs a cloud or on-prem sync server, documents have to reach that server for indexing, GraphRAG-based retrieval, and cross-project search through MCP.

## Decision

One-way push, local to server. The `.archcore/` directory stays the only source of truth. The server receives documents, indexes them, and serves enhanced search. It never writes back.

## Rationale

- Simplicity: no conflict resolution, no merge logic, no divergence state. A push is idempotent.
- Git-native: documents live in the repository and are versioned by git, so the server does not replicate git history or branching.
- Offline-first: authoring and reading work without server connectivity. Sync is an enhancement, not a dependency.
- CI/CD fit: a push from the main branch in CI is deterministic and reproducible, with no side effect on the local repository.
- Large repositories: an incremental sync driven by the SHA-256 manifest (`.archcore/.sync-state.json`) transmits only changed files, so the cost tracks the change set rather than the document count.

## Alternatives Considered

### Two-way sync

Bidirectional merge between local and server. Rejected: it introduces merge conflicts, needs a conflict-resolution interface, and leaves the authoritative version ambiguous.

### Server as source of truth

The server owns the documents and the local directory is a cache. Rejected: it breaks the git-native workflow and turns the server into a hard dependency for authoring.

## Implementation plan at decision time

This section records what the decision assumed. Some items were never implemented; the next section states what the code does today.

- Manifest: `.archcore/.sync-state.json`, gitignored, tracking per-file SHA-256 hashes and the last sync timestamp.
- Diff: walk the document directories, hash the files, compare against the manifest, and produce created, modified, and deleted sets.
- Protocol: `POST /api/v1/projects/{id}/sync` with an incremental payload, chunked for a large repository.
- Auth: a bearer token from the `ARCHCORE_TOKEN` environment variable, or a credential stored by an `archcore login` command.
- Triggers: manual (`archcore sync`), CI/CD (`archcore sync --ci`), or a commit hook.
- Server role: index the documents, build the GraphRAG, and expose a cross-project MCP search tool.

## Current behavior

- The manifest and the diff work as planned; `@internal/sync/` implements hashing, diffing, and payload construction.
- The endpoint is `POST /api/v1/sync`. `Client.Sync` in `@internal/api/client.go` sends the payload; the per-project path from the plan was never used.
- `Client` sets `Authorization: Bearer <token>` from its `Token` field. No `archcore login` command exists.
- Chunking is not implemented. One request carries the payload.
- No commit hook triggers a sync. `SessionStart` is the only active hook event.
- The `sync` command is gated: it is hidden and returns "sync is not available yet — this feature is coming soon". The related ADR on temporarily disabling sync records that decision.

## Consequences

### Positive

- No conflict-resolution complexity at any point.
- Work continues fully offline, and a sync failure is non-destructive because the manifest updates only after a confirmed response.
- The server can be rebuilt from scratch by re-syncing every project with `--force`.
- Several projects can sync to one server, which is what makes cross-project GraphRAG search possible.

### Negative

- Server-side editing is unsupported by design; authoring happens in git.
- IF `.sync-state.json` is lost, THEN a full re-sync is needed. It is safe but slower.
- Deletions have to be derived by comparing the manifest against the current files, which is more work than handling additions alone.

### Neutral

- The local MCP server (`archcore mcp`) keeps serving local-only search while the server MCP layer adds cross-project search. Both coexist.
