---
title: "Unified Release Cutover: One Tag, Plugin Tree Plus CLI Archives, Legacy Bridge"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "release"
  - "update"
---

## Goal

Make one `vX.Y.Z` tag on this repository publish the plugin tree to `main` and the CLI archives to the same GitHub Release, move archcore.ai and its install statistics to this repository, then archive `archcore-ai/cli`. The first unified version is v0.10.1 (owner confirmation 2026-09-22). Version state before the cutover: plugin tag v0.10.0 with manifests at 0.9.4, CLI release v0.8.7.

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
8. Run `go test ./...` in `cli/` and `make -C plugin all`; the test fixtures that name the legacy repository stay as fixtures.

### Phase 2 — Release workflow

9. Rewrite @.github/workflows/release.yml: `verify-version`, `test-plugin`, `test-cli`, `publish-plugin`, `publish-cli`. Done 2026-09-22.
10. Add `workflow_call` to @.github/workflows/cli-test.yml so the release calls it. Done 2026-09-22.
11. Gate `ARCHCORE_OFFICIAL_BUILD` and `POSTHOG_KEY` in `publish-cli` on `github.repository_id == 1201781375`. Done 2026-09-22.
12. Commit the changes on `dev`, push, and push tag v0.10.1.
13. Verify the release carries 9 assets and `main` matches `scripts/export-plugin.sh` output for the tag.
14. Re-run @.github/workflows/cli-install-smoke.yml after that release and confirm every job green.

### Phase 3 — Landing and the installers

15. In the landing `deploy.yml`, set the installer fetch base to `raw.githubusercontent.com/archcore-ai/plugin/refs/heads/dev/cli`.
16. Keep the landing placeholder check: each fetched installer carries exactly one `__POSTHOG_KEY__` and one `POSTHOG_HOST="..."` line.
17. In the landing `install-stats.yml`, set `CLI_REPO` to `archcore-ai/plugin`; leave the historical backfill on the archived releases.
18. Update the landing `docs/deployment.md` and the two landing ADRs that name `archcore-ai/cli` as the installer source.
19. Store `LANDING_DISPATCH_TOKEN` in this repository and add a root `notify-landing.yml` chained to a successful `CLI Install Smoke` run on `dev`.
20. After v0.10.1 is live, deploy landing and run `curl -fsSL https://archcore.ai/install.sh | bash`; expect `archcore 0.10.1`.
21. On Windows, run `irm https://archcore.ai/install.ps1 | iex`; expect `archcore 0.10.1`.

### Phase 4 — Archive and pins

22. Archive `archcore-ai/cli` after task 20 and task 21 pass.
23. Switch the pinned CLI download in @.github/workflows/test.yml to this repository's v0.10.1 assets and update the `0.8.7-dev` build label.
24. Delete the reference copies under `cli/.github/workflows/` after task 13 succeeds.
25. Update `how-to-release.guide.md`, `release-infrastructure.doc.md`, and the monorepo ADR consequence about CLI publication through `update_document`.

## Acceptance Criteria

- Pushing tag v0.10.1 produces an orphan `main` commit identical to `scripts/export-plugin.sh` output and one GitHub Release with 6 archives, `checksums.txt`, `install.sh`, and `install.ps1`.
- A tag whose manifests differ from it fails in `verify-version` before the tests, the export, the `main` push, and the GoReleaser job.
- `curl -sI https://github.com/archcore-ai/plugin/releases/latest` returns a `location:` header ending in `/releases/tag/v0.10.1`.
- A binary extracted from `archcore_darwin_arm64.tar.gz` prints `archcore 0.10.1` for `archcore --version`.
- A GoReleaser run without `POSTHOG_KEY` or `ARCHCORE_OFFICIAL_BUILD` fails at @cli/scripts/assert-not-inert.sh.
- `curl -fsSL https://archcore.ai/install.sh | bash` installs 0.10.1 after the landing redeploy, and the installed binary's `archcore update` reports "up to date" against this repository.
- The landing install statistics count downloads of this repository's releases.

## Dependencies

- Task 12 blocks task 13, task 14, task 20, and task 21: the installers fail against a `latest` release without CLI assets.
- Task 15 and task 19 live in `archcore-ai/landing` and in this repository's secrets; both need the owner's token.
- Task 22 waits for task 20 and task 21, because the archived repository is the only place older installers still resolve.
- The later rename to `archcore-ai/archcore` ships a release carrying the new name before the rename, because the redirect resolver in @cli/internal/update/update.go stops at the first 3xx.