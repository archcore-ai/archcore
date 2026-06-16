---
title: "Temporarily Disable Sync in CLI"
status: accepted
tags:
  - "sync"
---

## Context

The Archcore CLI includes full sync functionality (cloud and on-prem modes) for pushing local `.archcore/` documents to a remote server. However, for the initial public release, sync is not ready for end users — the server-side infrastructure and onboarding flow are still being finalized.

We need to ship the CLI now with local-only functionality while preserving all sync code for future activation.

## Decision

Temporarily disable all sync-related user-facing features in the CLI:

1. **Init** always sets `sync: "none"` — no sync type selector or URL prompt is shown
2. **Sync command** is hidden from help and returns "not available yet" when invoked directly
3. **Config** blocks setting or getting sync-related keys (`sync`, `project_id`, `archcore_url`)

All sync code (commands, internal packages, tests) is preserved in the codebase unchanged. Re-enabling sync requires removing the guards and restoring the init prompts.

## Alternatives Considered

- **Remove sync code entirely**: Rejected — would require significant rework to re-add later and lose test coverage.
- **Feature flag via environment variable**: Over-engineered for a temporary measure. The code changes are small and easily reversible.
- **Separate branch for sync-less release**: Adds merge complexity. A simple hide-in-place approach is cleaner.

## Consequences

- Users cannot configure or use sync until we explicitly re-enable it
- All sync-related unit tests continue to pass (they test internal functions directly)
- `settings.json` will always contain `{"sync": "none"}` for new installations
- Re-enabling requires changes to 3 files: `cmd/init.go`, `cmd/sync.go`, `cmd/config.go`
