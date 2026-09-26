---
title: "A Missing Global Source Degrades to Local Documents Instead of Stopping the MCP Server"
status: draft
tags:
  - "component:cli"
  - "config"
  - "globals"
  - "mcp"
---

## Context

This repository commits `.archcore/settings.json` with the global source `archcore` at `../global/.archcore`. On a clone without that sibling repository, `checkGlobals` in @cli/cmd/mcp.go aborted `archcore mcp`, so the host lost all ten MCP tools, including `create_document` and `init_project` for purely local work. When a session root changed to such a project, `docs.Scan` in @cli/internal/docs/scan.go failed `list_documents` and `search_documents` for the whole corpus. The SessionStart hook degraded to local documents with a warning, the code-alignment advisory in @cli/internal/advisory/code_alignment.go degraded without one, and `archcore status` ran its local checks but counted the source as an issue and exited non-zero.

## Decision

A declared global source whose resolved directory does not exist (`GlobalMissing`) is a warning state like `GlobalEmpty`: `archcore mcp` starts, prints `… — starting without it` to stderr, and names the source under `GLOBAL SOURCES NOT ON DISK` in its `initialize` instructions instead of as mounted, the scan mounts no documents from it, the SessionStart `GLOBALS` block prints `… — skipped until it is cloned; local documents are unaffected`, and `archcore status` and `archcore doctor` print a warning and exit zero, while a source that is not a directory, unreadable, self-overlapping, or a duplicate path, and an invalid `settings.json`, stay fatal.

## Alternatives Considered

1. Keep a missing source fatal, as the mandatory-globals decision set it — rejected because one uncloned repository removes every local tool, and the user requires local work to continue unchanged.
2. Add a per-entry `optional: true` flag — rejected because it restores the removed `required` flag and leaves every existing declaration fatal by default.
3. Report the missing source inside every `list_documents` and `search_documents` response — deferred because it changes two tool contracts for a signal that stderr, `archcore status`, the SessionStart `GLOBALS` block, and the `initialize` instructions already carry; the `coverage` map of `search_documents` already omits the absent source.
4. Add an `ARCHCORE_NO_GLOBALS` environment override — rejected because it is a hidden manual switch that each user has to discover.

## Consequences

### Positive

- A clone without the global source serves all MCP tools. Measured on 2026-09-25 with a binary built from this change on a scratch project that declares `../global/.archcore`: the server started, `list_documents` returned `by_source: {"local": 1}`, and `archcore status` exited 0.
- The fatal-or-warning verdict lives in one place, `GlobalState.Fatal` in @cli/internal/docs/inspect.go. Startup, the scan, session-root acceptance in @cli/internal/mcp/root_provider.go, `archcore status`, and SessionStart consult it and reach the same verdict for a missing source.

### Negative

- A typo in `path` now produces a warning instead of a startup failure. The agent sees the warning in the SessionStart `GLOBALS` block and in the `initialize` instructions. The instructions are built once for the start-time root, so a session root that changes later to a project with a missing source carries no in-band signal. [expected]
- The rejection of this option in the worktree-anchoring decision no longer holds. A worktree looks for an escaping source in the main checkout. IF that anchor lookup fails, for example without git, past the 500 ms bound, or in a submodule, THEN resolution falls back to the worktree's own root, and the worktree serves local documents only, with a warning, while the source exists in the main checkout. [expected]
- A regression in worktree anchoring now surfaces as a warning, not as a failed startup. `TestCheckGlobals_WorktreeResolvesFromMainCheckout` in @cli/cmd/mcp_worktree_test.go asserts `GlobalOK` so that the regression still fails a test.

## Superseded when

- Archcore ships a distribution step, for example `archcore globals pull` or a lockfile, that puts every declared source on disk during setup.
- One reported case shows a mistyped `path` that no one noticed for a full session, because the warning did not reach the user.