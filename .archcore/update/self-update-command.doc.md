---
title: "How archcore update Works"
status: accepted
tags:
  - "update"
---

## Overview

Explain what `archcore update` does, how to run it, and how to read its failures.

## Prerequisites

- An installed `archcore` binary, from the install script, from `go install`, or built from source.
- Network access to `github.com` for the version check, the archive download, and `checksums.txt`. The CLI does not contact the GitHub REST API (`api.github.com`).

## Usage

```bash
archcore update
```

Output when an update is available:

```
Archcore — Git-native context for AI coding agents

  Checking for updates...
  ✓ Current: v1.0.0
  ✓ Latest:  v1.1.0

  Downloading archcore_darwin_arm64.tar.gz...
  ✓ Checksum verified
  ✓ Updated to v1.1.0
```

Output when the binary is current:

```
Archcore — Git-native context for AI coding agents

  Checking for updates...
  ✓ Current: v1.1.0
  ✓ Latest:  v1.1.0

  ✓ Already up to date (v1.1.0)
```

## How the update runs

This section describes current CLI behavior, not steps for the reader.

1. Check the latest version. The CLI sends `GET https://github.com/archcore-ai/cli/releases/latest` and reads the tag from the redirect's `Location` without following it. Any `3xx` is accepted: GitHub answers `302` today, and the tag-page check, not the status code, is the real gate. The tag comes from the resolved URL's path (`resp.Location()`), so a query string, a fragment, or a `..` segment in the `Location` cannot smuggle a false version through.

   The REST API is avoided deliberately: it is capped at 60 unauthenticated requests per hour per IP, which teams sharing one egress address exhaust. The related ADR records that decision.

2. Compare versions. The current version, injected via ldflags at build time, is compared with the latest by semver order (major, then minor, then patch). A development build, where `version = "dev"`, always triggers an update.

3. Download the archive. WHEN a newer version exists, the CLI downloads the platform-specific archive from `https://github.com/archcore-ai/cli/releases/download/<version>/archcore_<os>_<arch>.<ext>`. The extension is `.zip` on Windows and `.tar.gz` on every other platform, matching `format_overrides` in `@.goreleaser.yaml`. The platform comes from `runtime.GOOS` and `runtime.GOARCH`.

4. Verify the checksum. The CLI downloads `checksums.txt` from the same release, computes the SHA-256 of the downloaded archive, and compares it against the expected hash. A mismatch fails the update immediately.

5. Extract the binary. The archive format, `tar.gz` or `zip`, is detected from the magic bytes. The CLI tries the name `archcore` first and falls back to the repository basename `cli`, because GoReleaser may use either. On Windows both candidates carry an `.exe` suffix.

6. Replace the binary atomically. The CLI resolves the current binary path via `os.Executable()` and `filepath.EvalSymlinks()`, writes the new binary to `<binary>.tmp.<pid>`, sets permissions to `0755`, and renames it over the current binary. It removes the temporary file on failure.

   On Windows a running `.exe` cannot be overwritten in place, so the current binary is renamed to `<binary>.old` first and the new file moves in. The `.old` file is removed on a best-effort basis; the next `archcore update` sweeps it when the running process still holds the lock.

## Code structure

| File | Purpose |
|------|---------|
| `@internal/update/update.go` | The `Updater` struct: `CheckLatest`, `NeedsUpdate`, `Apply`, `VerifyChecksum`, `ExtractBinary` |
| `@internal/update/update_test.go` | Unit tests with `httptest` servers |
| `@cmd/update.go` | Cobra command wiring, styled output, and the cached background version check |
| `@cmd/update_test.go` | Command-level integration tests |
| `@cmd/update_check_test.go` | Tests for the background check: cache TTL, negative caching, silence on failure |

## Verification

```bash
archcore --version
```

Expected result: the newly installed version.

## Troubleshooting

- `Could not check for updates` — a network problem, or `github.com` answered with something other than a redirect, such as a captive portal, a proxy interstitial, or an outage. Retry later. No API rate limit applies: the check reads the `github.com` web redirect, which carries no rate-limit budget and needs no token.
- `unexpected redirect resolving latest release` — the redirect landed somewhere that is not a `/releases/tag/` page. Two causes occur in practice: a proxy or captive portal intercepting the request, or the repository having no published release, which GitHub answers with a `302` to the bare `/releases` page.
- `no Location header` or `parsing redirect location` — a `3xx` arrived without a usable `Location`. This indicates an intercepting proxy rather than GitHub.
- `Update failed` with a permission error — the binary sits in a directory without write access. Reinstall it to a writable location, or run the command with `sudo`.
- `Checksum mismatch` — the download was corrupted. Run `archcore update` again.
- A development build always updates. WHILE the binary reports `archcore vdev`, the command downloads the latest release on every run.
