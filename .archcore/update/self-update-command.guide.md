---
title: "How archcore update Works"
status: accepted
tags:
  - "update"
---

## Prerequisites

- An installed `archcore` binary (via install script, `go install`, or built from source)
- Internet access to `github.com` — used for the version check, the archive download, and `checksums.txt`. The GitHub REST API (`api.github.com`) is not contacted.

## Steps

1. **Check latest version**

   The CLI sends `GET https://github.com/archcore-ai/cli/releases/latest` and reads the tag out of the redirect's `Location` without following it. Any `3xx` is accepted — GitHub answers `302` today, but the tag-page check is the real gate, not the status code. The tag is taken from the *resolved* URL's path (`resp.Location()`), so a query string, fragment, or `..` segment in the `Location` cannot smuggle a bogus version through.

   The REST API is deliberately not used — it is capped at 60 unauthenticated requests per hour per IP, which teams sharing one egress address exhaust. See `resolve-latest-via-github-redirect.adr.md`.

2. **Compare versions**

   Current version (injected via ldflags at build time) is compared with the latest using semver comparison (major → minor → patch). Dev builds (`version = "dev"`) always trigger an update.

3. **Download archive**

   If a newer version exists, the CLI downloads the platform-specific archive from GitHub Releases:
   `https://github.com/archcore-ai/cli/releases/download/<version>/archcore_<os>_<arch>.<ext>`

   The extension is `.zip` on Windows and `.tar.gz` on all other platforms — matching the `format_overrides` in `.goreleaser.yaml`. Platform is detected via `runtime.GOOS` and `runtime.GOARCH`.

4. **Download and verify checksum**

   Downloads `checksums.txt` from the same release. Computes SHA-256 of the downloaded archive and compares it against the expected hash. Fails immediately on mismatch.

5. **Extract binary**

   Extracts the binary from the archive — `tar.gz` or `zip`, auto-detected from the magic bytes. Tries the name `archcore` first, falls back to repo basename `cli` (GoReleaser may use either). On Windows both candidates carry an `.exe` suffix.

6. **Atomic replace**

   Resolves the current binary path via `os.Executable()` + `filepath.EvalSymlinks()`. Writes the new binary to a temp file (`<binary>.tmp.<pid>`), sets permissions (`0755`), and atomically renames it over the current binary. Cleans up the temp file on failure.

   On Windows the running `.exe` cannot be overwritten in place, so the current binary is first renamed to `<binary>.old` and the new file is moved in; the `.old` file is removed on a best-effort basis (the next `archcore update` sweeps it if the running process still holds the lock).

## Usage

```bash
# Update to latest
archcore update
```

**Output when update is available:**
```
Archcore — Git-native context for AI coding agents

  Checking for updates...
  ✓ Current: v1.0.0
  ✓ Latest:  v1.1.0

  Downloading archcore_darwin_arm64.tar.gz...
  ✓ Checksum verified
  ✓ Updated to v1.1.0
```

**Output when already up to date:**
```
Archcore — Git-native context for AI coding agents

  Checking for updates...
  ✓ Current: v1.1.0
  ✓ Latest:  v1.1.0

  ✓ Already up to date (v1.1.0)
```

## Code Structure

| File | Purpose |
|------|---------|
| `internal/update/update.go` | `Updater` struct: `CheckLatest`, `NeedsUpdate`, `Apply`, `VerifyChecksum`, `ExtractBinary` |
| `internal/update/update_test.go` | Unit tests with httptest servers |
| `cmd/update.go` | Cobra command wiring, styled output, and the cached background version check |
| `cmd/update_test.go` | Command-level integration tests |
| `cmd/update_check_test.go` | Tests for the background check: cache TTL, negative caching, silence on failure |

## Verification

```bash
archcore --version
```

Should show the newly installed version.

## Common Issues

- **"Could not check for updates"** — Network issue, or `github.com` answered with something other than a redirect (a captive portal, proxy interstitial, or an outage). Retry later. There is no API rate limit to wait out — the check reads the `github.com` web redirect, which carries no rate-limit budget and needs no token.
- **"unexpected redirect resolving latest release"** — the redirect landed somewhere that is not a `/releases/tag/` page. Two real causes: a proxy or captive portal intercepting the request, or the repo genuinely having no published release — GitHub answers that case with a `302` to the bare `/releases` page.
- **"no Location header" / "parsing redirect location"** — a `3xx` arrived without a usable `Location`. Almost always an intercepting proxy rather than GitHub.
- **"Update failed" with permission error** — The binary is installed in a directory without write access. Reinstall to a writable location or use `sudo`.
- **"Checksum mismatch"** — Download was corrupted. Retry `archcore update`.
- **Dev builds always update** — If running a dev build (`archcore vdev`), the command always downloads the latest release.
