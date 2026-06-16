---
title: "Auto-Detect repo_url from Git Origin During First Sync"
status: accepted
tags:
  - "sync"
---

## Context

When syncing for the first time (no `project_id` in settings), the server auto-creates a project. Previously only `project_name` (derived from directory name) was sent. The server needs to know which repository the project belongs to.

## Decision

Auto-detect `repo_url` from `git remote get-url origin` at sync time and include it in the payload during project auto-creation only.

- Detection lives in `internal/git/git.go` (`DetectRepoURL` function)
- Only sent when `project_id` is nil (first sync / auto-creation)
- Returns empty string (omitted from payload) if not a git repo or no origin remote
- No changes to `settings.json` schema — keeps it simple

## Alternatives Considered

- **Store repo_url in settings.json** — rejected; adds config complexity for something that can be auto-detected
- **Always send repo_url** — rejected; only relevant during project creation, not on every sync
- **Prompt user for repo URL** — rejected; git origin is the obvious source, no need for manual input

## Consequences

- Server receives repo context automatically without user configuration
- Non-git directories work fine — `repo_url` is simply omitted
- If origin remote changes after project creation, the server won't be updated (acceptable for now)