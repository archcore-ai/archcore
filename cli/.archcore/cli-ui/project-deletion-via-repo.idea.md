---
title: "Project Deletion Only Via Removing .archcore Directory"
status: draft
tags:
  - "cli-ui"
---

## Idea

Make removal of the `.archcore/` directory from the repository the only way to remove a project,
and have the server use soft-delete semantics.

The problem this addresses: WHEN a project is deleted on the platform (cloud or on-prem) while the
CLI still holds `project_id` in `settings.json`, sync fails with a 404.

```
✗ Sync failed
  → sync request failed: server returned status 404:
    {"code":"PROJECT_NOT_FOUND","detail":"Project with identifier '32' was not found"}
```

That leaves a broken state: the local `.archcore/` directory holds documents but has no
server-side project to sync to, and there is no recovery path. The user has to edit settings by
hand or re-initialize.

Proposed status: not implemented. This document records a proposal, not current behavior.

## Value

- One source of truth: the `.archcore/` directory in the repository.
- No accidental data loss from a click in the web interface.
- Git history preserves everything, so restoring is a checkout.
- The orphaned `project_id` state disappears.
- A simpler mental model for users.

## Possible Implementation

### Proposed rules

1. The platform interface offers no delete button, so a project cannot be hard-deleted from the web.
2. Removing a project means removing `.archcore/` from the repository.
3. The server soft-deletes: a "deleted" project is marked inactive and its data is preserved.
4. WHEN the server answers `PROJECT_NOT_FOUND`, the CLI offers to create a new project and reuse
   the existing documents instead of failing.

### Proposed flow

```
User removes .archcore/ from the repo
  → No further syncs happen (no directory means no CLI operations)
  → The server-side project goes stale (no new syncs)
  → The platform can auto-archive it after N days of inactivity (soft delete)

User wants to restore:
  → Re-run `archcore init` → a fresh project is created on the next sync
  → Or restore .archcore/ from git history
```

### Proposed CLI recovery for an orphaned project_id

WHEN sync receives a 404 `PROJECT_NOT_FOUND`:

1. The CLI warns: "Project #32 not found on server".
2. The CLI asks: "Create a new project and re-sync all documents? (Y/n)".
3. IF the user accepts, THEN the CLI clears `project_id` from settings and re-runs sync with
   `project_name`, which triggers the auto-create flow.
4. IF the user declines, THEN the CLI exits with a hint to check the server or run
   `archcore config set project_id <new_id>`.

## Risks and Constraints

- Rules 1 and 3 need server-side work; the CLI alone cannot deliver them.
- `N days of inactivity` for auto-archiving is unset. [LIMIT REQUIRED]
- The `sync` command is currently gated, so any CLI-side recovery lands only after sync is enabled again.
