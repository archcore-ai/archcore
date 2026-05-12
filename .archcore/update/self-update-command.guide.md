---
title: "How archcore update Works"
status: accepted
---

## Prerequisites

- An installed `archcore` binary (via install script, `go install`, or built from source)
- Internet access to `api.github.com` and `github.com`

## Steps

1. **Check latest version**

   The CLI calls `GET https://api.github.com/repos/archcore-ai/cli/releases/latest` and parses the `tag_name` field from the response.

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
| `cmd/update.go` | Cobra command wiring and styled output |
| `cmd/update_test.go` | Command-level integration tests |

## Verification

```bash
archcore --version
```

Should show the newly installed version.

## Common Issues

- **"Could not check for updates"** — Network issue or GitHub API rate limit (60 req/hr per IP without token). Retry later.
- **"Update failed" with permission error** — The binary is installed in a directory without write access. Reinstall to a writable location or use `sudo`.
- **"Checksum mismatch"** — Download was corrupted. Retry `archcore update`.
- **Dev builds always update** — If running a dev build (`archcore vdev`), the command always downloads the latest release.
