---
title: "Unified Release Cutover: One Tag, Plugin Tree Plus CLI Archives, Landing, and the Rename"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "release"
  - "update"
---

## Goal

Finish the cutover to one `vX.Y.Z` tag on `archcore-ai/archcore` that publishes the plugin tree to `main` and the CLI archives to the same GitHub Release. The release workflow, the installers, the landing fetch base, the docs, and the rename are done; v0.10.2 is the first release under the canonical name. Only the tasks below are open. Task numbers keep their original values, because other documents cite them.

## Tasks

19. Store `LANDING_DISPATCH_TOKEN` in this repository and add a root `notify-landing.yml` chained to a successful `CLI Install Smoke` run on `dev`. Until then, run the landing deploy by hand after an installer change.
21. On Windows, run `irm https://archcore.ai/install.ps1 | iex`; expect the latest release. The dev copy of the script passed the smoke run on PowerShell 5.1 and 7; a run against archcore.ai is still open.
23. Archive `archcore-ai/cli` after task 21 passes. The archived repository is the only place older installers still resolve.
27. Decide the landing `fetch-github-stars.mts` sources: it still sums stars of `archcore-ai/cli` and the redirected `archcore-ai/plugin`; changing it changes the displayed count.
29. Teach `CheckLatest` in @cli/internal/update/update.go and both installers to follow one same-path 301 before reading the tag redirect, with tests.
30. Release the resolver change from task 29 so binaries survive a future rename. The rename happened first, so v0.10.1 and older binaries cannot self-update and move through the installer.
31. Verify the plugin update from the legacy source to the canonical one on Codex, Cursor, and Copilot, per the remaining gate in `plugin-source-migration.spec`. Claude Code is verified: `archcore update` from the v0.10.2 binary rewrote the marketplace source to `archcore-ai/archcore`, kept `autoUpdate` and the enabled state, and left a mode-600 backup.

## Acceptance Criteria

- The landing install statistics count downloads of this repository's releases. [expected] First nightly run after the merge.
- After v0.10.2, `archcore update` on a machine with the legacy Codex marketplace source rewrites only the repository locator and keeps the plugin enabled as `archcore@archcore-plugins`.

## Dependencies

- Task 19 needs the owner's token in this repository's secrets.
- Task 23 waits for task 21.
- Task 30 depends on task 29. The resolver stops at the first 3xx (redirect ADR), so a rename without it breaks self-update for older binaries.
