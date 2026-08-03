---
title: "Installing Archcore CLI via install.sh and install.ps1"
status: accepted
tags:
  - "release"
---

## Purpose

Install the Archcore CLI on macOS, Linux, Windows, or WSL with the published install scripts.

## Prerequisites

### macOS and Linux

- `curl`, to download the release artifacts.
- `tar`, to extract the archive.
- `sha256sum` or `shasum`, to verify the checksum.
- `bash`. The install script needs bash, not POSIX `sh`. On a minimal distribution such as Alpine or distroless, install it first, for example with `apk add bash ca-certificates`.
- A GitHub release that carries the `archcore_<os>_<arch>.tar.gz` and `checksums.txt` assets.

### Windows

- PowerShell 5.1 or later, which ships with Windows 10 and 11, or PowerShell 7 or later.
- A GitHub release that carries the `archcore_windows_<arch>.zip` and `checksums.txt` assets.

### Windows with WSL

- [WSL](https://learn.microsoft.com/en-us/windows/wsl/install) installed.
- The macOS and Linux prerequisites above: `curl`, `tar`, `bash`, `sha256sum`.

`install.sh` covers macOS and Linux and runs under bash with `set -euo pipefail`. `install.ps1` covers Windows and needs PowerShell 5.1 or later. Both scripts expose the same environment variables and the same user experience.

## Procedure — macOS and Linux

Install the latest version:

```bash
curl -fsSL https://archcore.ai/install.sh | bash
```

Pin a specific version:

```bash
ARCHCORE_VERSION=v1.0.0 curl -fsSL https://archcore.ai/install.sh | bash
```

Choose a different install directory; the default is `~/.local/bin`:

```bash
ARCHCORE_INSTALL_DIR=/usr/local/bin curl -fsSL https://archcore.ai/install.sh | bash
```

Authenticate for a private repository:

```bash
GITHUB_TOKEN=ghp_xxx curl -fsSL https://archcore.ai/install.sh | bash
```

The token authenticates asset downloads only. Version resolution never uses it.

## Procedure — Windows

Install the latest version:

```powershell
irm https://archcore.ai/install.ps1 | iex
```

Pin a specific version:

```powershell
$env:ARCHCORE_VERSION = 'v1.0.0'; irm https://archcore.ai/install.ps1 | iex
```

Choose a different install directory; the default is `%LOCALAPPDATA%\Programs\archcore`:

```powershell
$env:ARCHCORE_INSTALL_DIR = 'C:\tools\archcore'; irm https://archcore.ai/install.ps1 | iex
```

Authenticate for a private repository:

```powershell
$env:GITHUB_TOKEN = 'ghp_xxx'; irm https://archcore.ai/install.ps1 | iex
```

The script installs `archcore.exe` under `%LOCALAPPDATA%\Programs\archcore` and adds that directory to the user `PATH`. Open a new PowerShell window after the install, so the session picks up the updated `PATH`.

## Procedure — Windows with WSL

Install [WSL](https://learn.microsoft.com/en-us/windows/wsl/install), then run inside the WSL terminal:

```bash
curl -fsSL https://archcore.ai/install.sh | bash
```

WSL provides a full Linux environment, so this path uses the macOS and Linux script.

## What the scripts do

1. Detect the operating system (`darwin`, `linux`, `windows`) and the architecture (`amd64`, `arm64`).
2. Resolve the latest version by reading the `Location` header of `https://github.com/archcore-ai/cli/releases/latest`, or skip the lookup entirely when `ARCHCORE_VERSION` is set. The GitHub REST API is avoided deliberately; the related ADR records that decision.
3. Download the platform-specific archive — `.tar.gz` on Unix, `.zip` on Windows — and `checksums.txt`.
4. Verify the SHA-256 checksum.
5. Extract the binary and install it atomically into the install directory.
6. On Unix, check whether the install directory is in `$PATH` and print shell-specific guidance when it is not. On Windows, append the install directory to the user `PATH` through `HKCU\Environment` and prompt for a new terminal session.

## Environment variables

| Variable | Default (Unix) | Default (Windows) | Description |
|---|---|---|---|
| `ARCHCORE_VERSION` | (latest) | (latest) | Pin a specific release tag, for example `v1.0.0`. Skips the version lookup. |
| `ARCHCORE_INSTALL_DIR` | `~/.local/bin` | `%LOCALAPPDATA%\Programs\archcore` | Override the install directory. |
| `GITHUB_TOKEN` | (none) | (none) | GitHub token for authenticated asset downloads from a private repository. Version resolution does not use it; that reads a public redirect with no rate limit. |

On Windows, set an environment variable with PowerShell syntax before the `irm | iex` pipeline: `$env:ARCHCORE_VERSION = 'v1.0.0'`.

## Verification

```bash
archcore --version
```

Expected result: `archcore <version> (commit: <sha>)`.

## Troubleshooting

- `command not found` after the install — the install directory is not in `$PATH`. The script prints instructions for bash, zsh, and fish.
- `Could not reach https://github.com/…/releases/latest` — a network, proxy, or DNS problem. No API rate limit is involved, so `GITHUB_TOKEN` does not help. Pin a version instead: `ARCHCORE_VERSION=x.y.z`.
- `Could not resolve the latest version … (unexpected response)` — `github.com` answered without the expected `/releases/tag/` redirect. A captive portal or a proxy interstitial usually intercepts the request; the repository may also have no published release yet. Pin a version to bypass the lookup.
- `Checksum verification failed` — the download was corrupted. Run the install again.
- `Unsupported operating system/architecture` — only `darwin`, `linux`, and `windows` on `amd64` and `arm64` are supported. On another target such as armv7, ppc64le, s390x, or riscv64, no binary ships; build from source with `go install github.com/archcore-ai/cli@latest`.
- `this installer requires bash` — the script was piped into `sh` or another POSIX shell. Use `| bash`. On Debian and Ubuntu, `/bin/sh` is dash, which does not support `set -o pipefail`.
- Windows SmartScreen blocks the binary — the installer calls `Unblock-File` on the downloaded executable, so this is not expected. IF it happens, THEN right-click `archcore.exe`, open Properties, select Unblock, or run the installer again.
- Windows antivirus false positive — a Go static binary occasionally trips Defender heuristics. Add `%LOCALAPPDATA%\Programs\archcore` to the allowlist, or report the detection to the antivirus vendor. Code-signed builds are planned, not implemented.
