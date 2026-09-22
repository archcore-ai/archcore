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

Make one `vX.Y.Z` tag on this repository publish the plugin tree to `main` and the CLI archives to the same GitHub Release, move archcore.ai and its install statistics to this repository, archive `archcore-ai/cli`, and prepare the later rename to `archcore-ai/archcore` without breaking installed plugins or binaries. The first unified version, v0.10.1, shipped on 2026-09-22 from commit 599332a (Release run 35766167490). Version state before the cutover: plugin tag v0.10.0 with manifests at 0.9.4, CLI release v0.8.7.

## Tasks

### Phase 0 — Inputs

1. Confirm the first unified version. Done 2026-09-22: v0.10.1.
2. Copy repository variables `POSTHOG_KEY` and `POSTHOG_HOST` from `archcore-ai/cli` to `archcore-ai/plugin`. Done 2026-09-22.

### Phase 1 — Source changes on dev

3. Set `GITHUB_REPO` to `archcore-ai/plugin` in @cli/install.sh and @cli/install.ps1. Done 2026-09-22.
4. Set the update repository to `archcore-ai/plugin` in @cli/cmd/update.go. Done 2026-09-22.
5. Point the changelog commit link in @cli/.goreleaser.yaml at this repository and add both installers as release files. Done 2026-09-22.
6. Replace `archcore-ai/cli` links in @cli/README.md and @plugin/README.md. Done 2026-09-22.
7. Set the four plugin manifests to 0.10.1. Done 2026-09-22.
8. Run `go test ./...` in `cli/` and `make -C plugin all`. Done 2026-09-22: 22 Go packages and 639 bats tests pass; the test fixtures that name the legacy repository stay as fixtures.

### Phase 2 — Release workflow

9. Rewrite @.github/workflows/release.yml: `verify-version`, `test-plugin`, an inlined `test-cli`, `publish-plugin`, `publish-cli`. Done 2026-09-22.
10. Gate `ARCHCORE_OFFICIAL_BUILD` and `POSTHOG_KEY` in `publish-cli` on `github.repository_id == 1201781375`. Done 2026-09-22.
11. Set `release.replace_existing_artifacts: true` in @cli/.goreleaser.yaml so a re-run converges. Done 2026-09-22.
12. Commit on `dev`, push, and push tag v0.10.1. Done 2026-09-22: commit 599332a.
13. Verify the release: 9 assets, `main` equal to the export, `releases/latest` at v0.10.1, `archcore --version` from the darwin_arm64 archive. Done 2026-09-22.
14. Re-run @.github/workflows/cli-install-smoke.yml on `dev` after the release and confirm every job green.

### Phase 3 — Landing and the installers

15. In the landing `deploy.yml`, set the installer fetch base to `raw.githubusercontent.com/archcore-ai/plugin/refs/heads/dev/cli`.
16. Keep the landing placeholder check: each fetched installer carries exactly one `__POSTHOG_KEY__` and one `POSTHOG_HOST="..."` line.
17. In the landing `install-stats.yml`, set `CLI_REPO` to `archcore-ai/plugin`; leave the historical backfill on the archived releases.
18. Update the landing `docs/deployment.md` and the two landing ADRs that name `archcore-ai/cli` as the installer source.
19. Store `LANDING_DISPATCH_TOKEN` in this repository and add a root `notify-landing.yml` chained to a successful `CLI Install Smoke` run on `dev`.
20. Deploy landing and run `curl -fsSL https://archcore.ai/install.sh | bash`; expect `archcore 0.10.1`.
21. On Windows, run `irm https://archcore.ai/install.ps1 | iex`; expect `archcore 0.10.1`.

### Phase 4 — Archive and pins

22. Archive `archcore-ai/cli` after task 20 and task 21 pass.
23. Switch the pinned CLI download in @.github/workflows/test.yml to this repository's v0.10.1 assets and update the `0.8.7-dev` build label.
24. Delete the reference copies under `cli/.github/workflows/`.
25. Update `how-to-release.guide.md`, `release-infrastructure.doc.md`, `install-script-usage.guide.md`, `self-update-command.doc.md`, `stack-and-tooling.rule.md`, `plugin-development.guide.md`, and the monorepo ADR. Done 2026-09-22.

### Phase 5 — Rename to `archcore-ai/archcore`

26. Keep every plugin identifier unchanged: marketplace `archcore-plugins`, plugin `archcore`, id `archcore@archcore-plugins`, path `plugins/archcore`, branch `main`.
27. Teach `CheckLatest` in @cli/internal/update/update.go and both installers to follow one same-path 301 before reading the tag redirect, with tests.
28. Release the resolver change from task 27 before the rename, so binaries in the wild survive the 301.
29. Verify the plugin update from the old source to the new one on Claude Code, Codex, Cursor, and Copilot, per the remaining gate in `plugin-source-migration.spec`.
30. Rename the repository on GitHub and never create a new `archcore-ai/plugin`.
31. Confirm `github.repository_id` still reads 1201781375 after the rename, so the official-build gate holds.
32. Set `RepoID` in @cli/internal/plugin/plugin.go, `GITHUB_REPO` in both installers, the update repository in @cli/cmd/update.go, and the changelog link in @cli/.goreleaser.yaml to `archcore-ai/archcore`; release.
33. Repeat task 15 and task 17 with the new name and redeploy landing.

## Acceptance Criteria

- Pushing tag v0.10.1 produced an orphan `main` commit identical to `scripts/export-plugin.sh` output and one GitHub Release with 6 archives, `checksums.txt`, `install.sh`, and `install.ps1`. Met 2026-09-22.
- A tag whose manifests differ from it fails in `verify-version` before the tests, the export, the `main` push, and the GoReleaser job.
- `curl -sI https://github.com/archcore-ai/plugin/releases/latest` returns a `location:` header ending in `/releases/tag/v0.10.1`. Met 2026-09-22.
- A binary extracted from `archcore_darwin_arm64.tar.gz` prints `archcore 0.10.1` for `archcore --version`. Met 2026-09-22.
- A GoReleaser run without `POSTHOG_KEY` or `ARCHCORE_OFFICIAL_BUILD` fails at @cli/scripts/assert-not-inert.sh.
- `curl -fsSL https://archcore.ai/install.sh | bash` installs 0.10.1 after the landing redeploy, and the installed binary's `archcore update` reports "up to date" against this repository.
- The landing install statistics count downloads of this repository's releases.
- After the rename, a binary released before it resolves `releases/latest` through the 301, and the plugin stays enabled as `archcore@archcore-plugins` on all four hosts.

## Dependencies

- Task 14, task 20, and task 21 depend on task 13: the installers fail against a `latest` release without CLI assets.
- Task 15 and task 19 live in `archcore-ai/landing` and in this repository's secrets; both need the owner's token.
- Task 22 waits for task 20 and task 21, because the archived repository is the only place older installers still resolve.
- Task 28 precedes task 30, because the redirect resolver in @cli/internal/update/update.go stops at the first 3xx (redirect ADR); GitHub redirects git and web traffic after a rename but a new repository under the old name disables those redirects (docs.github.com, renaming a repository, read 2026-09-22).
- Task 26 follows the Claude Code plugin documentation (code.claude.com/docs/en/plugin-marketplaces, read 2026-09-22): plugins are keyed `name@marketplace-name`, and a changed marketplace name orphans that reference.