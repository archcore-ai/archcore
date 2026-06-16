---
title: "Project Deletion Only Via Removing .archcore Directory"
status: draft
tags:
  - "cli-ui"
---

## Problem

When a project is deleted on the platform (cloud/on-prem) but the CLI still has `project_id` in `settings.json`, sync fails with a 404:

```
✗ Sync failed
  → sync request failed: server returned status 404:
    {"code":"PROJECT_NOT_FOUND","detail":"Project with identifier '32' was not found"}
```

This creates a broken state where the local `.archcore/` directory exists with documents but has no server-side project to sync to. There's no recovery path — the user must manually fix settings or re-init.

## Proposal

**Project removal should only happen by removing the `.archcore/` directory from the repository.** The server should use soft-delete semantics.

### Rules

1. **No delete button in the platform UI** — projects cannot be hard-deleted from the web interface
2. **Remove project = remove `.archcore/` from repo** — this is the single source of truth
3. **Server uses soft-delete** — when a project is "deleted", it's marked inactive but data is preserved
4. **Sync detects orphaned project_id** — if the server returns `PROJECT_NOT_FOUND`, the CLI should offer to create a new project (re-use the existing documents) rather than just failing

### Flow

```
User removes .archcore/ from repo
  → No more syncs happen (no directory = no CLI operations)
  → Server-side project becomes stale (no new syncs)
  → Platform can auto-archive after N days of inactivity (soft-delete)

User wants to restore:
  → Re-run `archcore init` → fresh project created on next sync
  → Or restore .archcore/ from git history
```

### CLI Recovery for Orphaned project_id

When sync gets a 404 `PROJECT_NOT_FOUND`, instead of failing:

1. Warn the user: "Project #32 not found on server"
2. Offer: "Create a new project and re-sync all documents? (Y/n)"
3. If yes: clear `project_id` from settings, re-run sync with `project_name` (auto-create flow)
4. If no: exit with hint to check server or run `archcore config set project_id <new_id>`

## Benefits

- Single source of truth: `.archcore/` directory in the repo
- No accidental data loss from UI clicks
- Git history preserves everything — easy to restore
- Eliminates the orphaned project_id problem
- Simpler mental model for users