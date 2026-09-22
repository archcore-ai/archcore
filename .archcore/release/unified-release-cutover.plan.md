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

Make one `vX.Y.Z` tag on this repository publish the plugin tree to `main` and the CLI archives to the same GitHub Release, move archcore.ai and its install statistics to this repository, archive `archcore-ai/cli`, and complete the rename to `archcore-ai/archcore` without breaking installed plugins. The first unified version, v0.10.1, shipped on 2026-09-22 from commit 599332a (Release run 35766167490). The owner renamed the repository from `archcore-ai/plugin` to `archcore-ai/archcore` later the same day; v0.10.2 carries the new name in the CLI. Version state before the cutover: plugin tag v0.10.0 with manifests at 0.9.4, CLI release v0.8.7.

## Tasks

### Phase 0 — Inputs

1. Confirm the first unified version. Done 2026-09-22: v0.10.1.
2. Copy repository variables `POSTHOG_KEY` and `POSTHOG_HOST` from `archcore-ai/cli` to this repository. Done 2026-09-22.

### Phase 1 — Source changes on dev

3. Set `GITHUB_REPO` to this repository in @cli/install.sh and @cli/install.ps1. Done 2026-09-22.
4. Set the update repository to this repository in @cli/cmd/update.go. Done 2026-09-22.
5. Point the changelog commit link in @cli/.goreleaser.yaml at this repository and add both installers as release files. Done 2026-09-22.
6. Replace `archcore-ai/cli` links in @cli/README.md and @plugin/README.md. Done 2026-09-22.
7. Set the four plugin manifests to 0.10.1. Done 2026-09-22.
8. Run `go test ./...` in `cli/` and `make -C plugin all`. Done 2026-09-22: 22 Go packages and 639 bats tests pass.

### Phase 2 — Release workflow

9. Rewrite @.github/workflows/release.yml: `verify-version`, `test-plugin`, an inlined `test-cli`, `publish-plugin`, `publish-cli`. Done 2026-09-22.
10. Gate `ARCHCORE_OFFICIAL_BUILD` and `POSTHOG_KEY` in `publish-cli` on `github.repository_id == 1201781375`. Done 2026-09-22.
11. Set `release.replace_existing_artifacts: true` in @cli/.goreleaser.yaml so a re-run converges. Done 2026-09-22.
12. Commit on `dev`, push, and push tag v0.10.1. Done 2026-09-22: commit 599332a.
13. Verify the release: 9 assets, `main` equal to the export, `releases/latest` at v0.10.1, `archcore --version` from the darwin_arm64 archive. Done 2026-09-22.
14. Re-run @.github/workflows/cli-install-smoke.yml on `dev` after the release. Done 2026-09-22: run 35767139007, 11 jobs green.

### Phase 3 — Landing and the installers

15. In the landing `deploy.yml`, set the installer fetch base to `raw.githubusercontent.com/archcore-ai/archcore/refs/heads/dev/cli`.
16. Keep the landing placeholder check: each fetched installer carries exactly one `__POSTHOG_KEY__` and one `POSTHOG_HOST="..."` line.
17. In the landing `install-stats.yml`, set `CLI_REPO` to `archcore-ai/archcore`; leave the historical backfill on the archived releases.
18. Update the landing `docs/deployment.md`, its install commands and links, and its tests to the new repository name.
19. Store `LANDING_DISPATCH_TOKEN` in this repository and add a root `notify-landing.yml` chained to a successful `CLI Install Smoke` run on `dev`.
20. Deploy landing and run `curl -fsSL https://archcore.ai/install.sh | bash`; expect `v0.10.2`.
21. On Windows, run `irm https://archcore.ai/install.ps1 | iex`; expect `v0.10.2`.

### Phase 4 — Archive and pins

22. Archive `archcore-ai/cli` after task 20 and task 21 pass.
23. Switch the pinned CLI download in @.github/workflows/test.yml to this repository's v0.10.1 assets and the `0.10.1-dev` build label. Done 2026-09-22.
24. Delete the reference copies under `cli/.github/workflows/`. Done 2026-09-22.
25. Update the release, installer, self-update, plugin stack, and plugin development documents and the monorepo ADR. Done 2026-09-22.

### Phase 5 — Rename to `archcore-ai/archcore`

26. Keep every plugin identifier unchanged: marketplace `archcore-plugins`, plugin `archcore`, id `archcore@archcore-plugins`, path `plugins/archcore`, branch `main`. Done by construction; verified in the v0.10.2 manifests and catalogs.
27. Teach `CheckLatest` in @cli/internal/update/update.go and both installers to follow one same-path 301 before reading the tag redirect, with tests.
28. Release the resolver change from task 27 so binaries survive a future rename; this rename happened first, so v0.10.1 and older binaries cannot self-update and move through the installer.
29. Verify the plugin update from the legacy source to the canonical one on Claude Code, Codex, Cursor, and Copilot, per the remaining gate in `plugin-source-migration.spec`.
30. Rename the repository on GitHub and never create a new `archcore-ai/plugin`. Done by the owner 2026-09-22.
31. Confirm `github.repository_id` still reads 1201781375 after the rename. Done 2026-09-22.
32. Set `RepoID` in @cli/internal/plugin/plugin.go to the canonical id, and `GITHUB_REPO`, the update repository, the changelog link, and the four manifests' `repository` to `archcore-ai/archcore`. Done 2026-09-22 on `dev`; ships as v0.10.2.
33. Update the Archcore documents that name the repository: the delivery, update, and source-migration specs, the compatibility rule, the agents and component docs, the rollout plan, and the development guide. Done 2026-09-22.

## Acceptance Criteria

- Pushing tag v0.10.1 produced an orphan `main` commit identical to `scripts/export-plugin.sh` output and one GitHub Release with 6 archives, `checksums.txt`, `install.sh`, and `install.ps1`. Met 2026-09-22.
- A tag whose manifests differ from it fails in `verify-version` before the tests, the export, the `main` push, and the GoReleaser job.
- `curl -sI https://github.com/archcore-ai/archcore/releases/latest` returns a `location:` header ending in `/releases/tag/<latest tag>`. Met 2026-09-22 for v0.10.1 after the rename.
- A binary extracted from the v0.10.2 `archcore_darwin_arm64.tar.gz` prints `v0.10.2` for `archcore --version` and reports "up to date" on `archcore update --check` against `archcore-ai/archcore`.
- A GoReleaser run without `POSTHOG_KEY` or `ARCHCORE_OFFICIAL_BUILD` fails at @cli/scripts/assert-not-inert.sh.
- `curl -fsSL https://archcore.ai/install.sh | bash` installs the latest release after the landing redeploy.
- The landing install statistics count downloads of this repository's releases.
- After v0.10.2, `archcore update` on a machine with the legacy Claude Code or Codex marketplace source rewrites only the repository locator and keeps the plugin enabled as `archcore@archcore-plugins`.

## Dependencies

- Task 14, task 20, and task 21 depend on task 13: the installers fail against a `latest` release without CLI assets.
- Task 15 and task 19 live in `archcore-ai/landing` and in this repository's secrets; both need the owner's token. The landing change is prepared as a pull request, and its merge deploys the site.
- Task 22 waits for task 20 and task 21, because the archived repository is the only place older installers still resolve.
- Task 28 would have preceded task 30, because the redirect resolver in @cli/internal/update/update.go stops at the first 3xx (redirect ADR). The owner renamed first; the cost is the self-update path of binaries v0.10.1 and older, which now answer `unexpected redirect` and reinstall through archcore.ai. GitHub redirects git and web traffic after a rename, and a new repository under the old name disables those redirects (docs.github.com, renaming a repository, read 2026-09-22).
- Task 26 follows the Claude Code plugin documentation (code.claude.com/docs/en/plugin-marketplaces, read 2026-09-22): plugins are keyed `name@marketplace-name`, and a changed marketplace name orphans that reference.