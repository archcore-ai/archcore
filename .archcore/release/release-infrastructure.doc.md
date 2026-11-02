---
title: "Release Infrastructure Overview"
status: accepted
tags:
  - "component:cli"
  - "release"
---

## Overview

The Archcore CLI and the Archcore plugin share one tag-driven release pipeline in `archcore-ai/archcore`. Pushing a `v*` tag triggers @.github/workflows/release.yml, which checks that the tag is the next release version (@scripts/check-release-tag.sh), runs both test suites, regenerates the plugin distribution on `main` with the tag's version in the four plugin manifests (@scripts/export-plugin.sh), publishes `@archcore/opencode-plugin` to npm with the same version, and invokes GoReleaser from `cli/` to build cross-platform binaries and publish the GitHub Release. The tag is the only version source; the manifests on `dev` carry the placeholder `0.0.0`. The jobs run in the order `verify-version`, `test-plugin` and `test-cli`, `publish-plugin`, `publish-opencode`, `publish-cli`. `publish-opencode` skips a version npm already has, so recovery is a re-run of the tag's run.

Two former names remain in circulation. `archcore-ai/cli` published CLI releases up to v0.8.7 and receives no further releases. `archcore-ai/plugin` was this repository's name until 2026-09-22; GitHub redirects git and web traffic from it, and a new repository under that name would disable the redirect.

## Content

### Release shape

Builds target darwin, linux, and windows on amd64 and arm64, with `CGO_ENABLED=0` and the `-s -w` ldflags. Archives are `archcore_<os>_<arch>.tar.gz` for darwin and linux and `.zip` for windows. Every release also carries `checksums.txt`, `install.sh`, and `install.ps1`. The pipeline needs no signing keys and no notarization credentials.

Both installers and `archcore update` download `checksums.txt` on every run. Its download count is therefore a denoised proxy for real installer runs, while the archive count is the raw figure; see the install analytics ADR. `archcore --version` prints the version with a `v` prefix. A build without an injected version starts from `dev` and takes the module version from Go build info when one is recorded (`resolveVersion` in `@cli/main.go`).

### Version resolution

Both install scripts and `archcore update` resolve "latest" by reading the `Location` header of `https://github.com/archcore-ai/archcore/releases/latest`, a `302` that already carries the tag. The GitHub REST API is avoided deliberately: its 60 requests per hour unauthenticated limit is per IP and breaks teams behind a shared egress address. Because every tag releases both components, the redirect always lands on a release that carries CLI assets. The resolver halts at the first 3xx, so a repository rename, which answers with a 301 to the new name first, breaks the check for binaries built before the rename; the 2026-09-22 rename did exactly that to the v0.10.1 binaries.

`go install` is not a channel: the module path in `@cli/go.mod` is `archcore-cli`, not a GitHub path.

### Install analytics

The installers published on archcore.ai send one anonymous event per run. The PostHog key is **not** baked into the scripts in this repository: both carry a `__POSTHOG_KEY__` placeholder, and the landing deploy workflow substitutes the real key while it syncs them into `public/`.

- A script run from a clone, a fork, or `cli-install-smoke.yml` reports nothing, because the guard requires a `phc_` prefix.
- The landing deploy fails when a synced script does not carry exactly one placeholder.
- The landing deploy fetches both installers from this repository's `dev` branch (`raw.githubusercontent.com/archcore-ai/archcore/refs/heads/dev/cli`). The copies from `archcore-ai/cli` are no longer served.

The contract, the event properties, and the opt-out procedure are in install-script-usage.guide.md. The decision and its trade-offs are in the install analytics ADR.

### Secrets and variables

| Name | Location | Required | Purpose |
|---|---|---|---|
| `GITHUB_TOKEN` | this repo (automatic) | yes | Pushes `main` and publishes the release. GitHub Actions provides it; nothing to configure. |
| `NPM_TOKEN` | this repo (secret) | yes | Publishes `@archcore/opencode-plugin` in `publish-opencode`. An npm granular token with read-and-write (publish) permission on the `@archcore` scope and 2FA bypass. |
| `POSTHOG_KEY` | this repo (variable) | yes, for telemetry | Public PostHog project key, injected as `-X archcore-cli/internal/telemetry.apiKey` when `github.repository_id` is `1201781375`. `@cli/scripts/assert-not-inert.sh` fails the release when the built binary does not carry it. |
| `POSTHOG_HOST` | this repo (variable) | no | Ingestion host. Present for parity with the landing deploy. |
| `ARCHCORE_OFFICIAL_BUILD` | workflow expression | yes | The official-build marker, `-X archcore-cli/internal/update.officialBuild`, injected only when `github.repository_id` is `1201781375`; the id survived the rename, and a fork builds without it and never self-replaces. |
| `LANDING_DISPATCH_TOKEN` | this repo (secret) | no | Pending with the root landing dispatcher (open task in `unified-release-cutover.plan`); a PAT with `contents: write` on the landing repository. |
| `POSTHOG_KEY` | landing repository (variable) | yes, for analytics | Substituted into the installers at landing deploy time. A missing or non-`phc_` value fails the landing deploy. |
| `POSTHOG_HOST` | landing repository (variable) | no | Ingestion host, `https://edge.archcore.ai`. Falls back to the same value when unset. |
