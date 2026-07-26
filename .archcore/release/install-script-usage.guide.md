---
title: "Installing Archcore CLI via install.sh and install.ps1"
status: accepted
tags:
  - "release"
---

## Prerequisites

### macOS / Linux

- `curl` — for downloading release artifacts
- `tar` — for extracting the archive
- `sha256sum` or `shasum` — for checksum verification (optional but recommended)
- `bash` — the install script requires bash (not POSIX sh). On minimal distros (Alpine, distroless), install it first (e.g. `apk add bash ca-certificates`).
- A GitHub release must exist with `archcore_<os>_<arch>.tar.gz` and `checksums.txt` assets

### Windows

- PowerShell 5.1+ (ships by default on Windows 10/11) or PowerShell 7+
- A GitHub release must exist with `archcore_windows_<arch>.zip` and `checksums.txt` assets

### Windows (WSL)

- [WSL](https://learn.microsoft.com/en-us/windows/wsl/install) installed
- Same prerequisites as macOS / Linux (curl, tar, bash, sha256sum)

## Installation Methods

### macOS / Linux (recommended)

1. **Basic install (latest version)**

   ```bash
   curl -fsSL https://archcore.ai/install.sh | bash
   ```

2. **Pin a specific version**

   ```bash
   ARCHCORE_VERSION=v1.0.0 curl -fsSL https://archcore.ai/install.sh | bash
   ```

3. **Custom install directory** (default: `~/.local/bin`)

   ```bash
   ARCHCORE_INSTALL_DIR=/usr/local/bin curl -fsSL https://archcore.ai/install.sh | bash
   ```

4. **Authenticate for private repos**

   ```bash
   GITHUB_TOKEN=ghp_xxx curl -fsSL https://archcore.ai/install.sh | bash
   ```

   The token authenticates asset downloads only. Version resolution never uses it — see Environment Variables below.

### Windows

1. **Basic install (latest version)**

   ```powershell
   irm https://archcore.ai/install.ps1 | iex
   ```

2. **Pin a specific version**

   ```powershell
   $env:ARCHCORE_VERSION = 'v1.0.0'; irm https://archcore.ai/install.ps1 | iex
   ```

3. **Custom install directory** (default: `%LOCALAPPDATA%\Programs\archcore`)

   ```powershell
   $env:ARCHCORE_INSTALL_DIR = 'C:\tools\archcore'; irm https://archcore.ai/install.ps1 | iex
   ```

4. **Authenticate for private repos**

   ```powershell
   $env:GITHUB_TOKEN = 'ghp_xxx'; irm https://archcore.ai/install.ps1 | iex
   ```

The script installs `archcore.exe` under `%LOCALAPPDATA%\Programs\archcore` and adds that directory to your user `PATH`. Open a new PowerShell window after install so the updated `PATH` is picked up.

### Windows (WSL)

Install [WSL](https://learn.microsoft.com/en-us/windows/wsl/install), then run inside the WSL terminal:

```bash
curl -fsSL https://archcore.ai/install.sh | bash
```

This uses the same install script as macOS / Linux — WSL provides a full Linux environment.

## What the Scripts Do

1. Detect OS (`darwin`/`linux`/`windows`) and architecture (`amd64`/`arm64`)
2. Resolve the latest version by reading the `Location` header of `https://github.com/archcore-ai/cli/releases/latest` (or skip the lookup entirely with `ARCHCORE_VERSION`). The GitHub REST API is deliberately not used — see `resolve-latest-via-github-redirect.adr.md`.
3. Download the platform-specific archive (`.tar.gz` on Unix, `.zip` on Windows) and `checksums.txt`
4. Verify SHA-256 checksum
5. Extract the binary and install it atomically to the install directory
6. On Unix: check if the install directory is in `$PATH` and print shell-specific guidance if not. On Windows: append the install directory to the user `PATH` via `HKCU\Environment` and prompt for a new terminal session.

> **Note:** `install.sh` requires bash (`set -euo pipefail`) and covers macOS + Linux. `install.ps1` requires PowerShell 5.1+ and covers Windows. They share the same env-var surface and UX.

## Environment Variables

| Variable | Default (Unix) | Default (Windows) | Description |
|---|---|---|---|
| `ARCHCORE_VERSION` | (latest) | (latest) | Pin to a specific release tag (e.g. `v1.0.0`). Skips the version lookup. |
| `ARCHCORE_INSTALL_DIR` | `~/.local/bin` | `%LOCALAPPDATA%\Programs\archcore` | Override the install directory |
| `GITHUB_TOKEN` | (none) | (none) | GitHub token for authenticated asset downloads (private repos). Not used for version resolution — that reads a public redirect with no rate limit. |

On Windows, set env vars with PowerShell syntax: `$env:ARCHCORE_VERSION = 'v1.0.0'` before the `irm | iex` pipeline.

## Verification

```bash
archcore --version
```

Expected output: `archcore <version> (commit: <sha>)`

## Common Issues

- **"command not found" after install** — The install directory is not in your `$PATH`. The script prints instructions for your shell (bash/zsh/fish).
- **"Could not reach https://github.com/…/releases/latest"** — Network, proxy, or DNS issue. There is no API rate limit involved, so `GITHUB_TOKEN` will not help; pin a version instead: `ARCHCORE_VERSION=x.y.z`.
- **"Could not resolve the latest version … (unexpected response)"** — `github.com` answered without the expected `/releases/tag/` redirect. Usually a captive portal or proxy interstitial intercepting the request; it can also mean the repo has no published release yet. Pin a version to bypass the lookup.
- **"Checksum verification failed"** — The download was corrupted. Retry the install.
- **"Unsupported operating system/architecture"** — Only `darwin`/`linux`/`windows` on `amd64`/`arm64` are supported. If you're on a different target (armv7, ppc64le, s390x, riscv64), no binary ships for your platform — build from source via `go install github.com/archcore-ai/cli@latest`.
- **"this installer requires bash"** — You piped the install script into `sh` (or another POSIX shell). Use `| bash` instead. On Debian/Ubuntu `/bin/sh` is dash, which does not support `set -o pipefail`.
- **Windows SmartScreen blocks the binary** — The installer calls `Unblock-File` on the downloaded exe so this should not happen, but if it does, right-click `archcore.exe` → Properties → Unblock, or re-run the installer.
- **Windows antivirus false-positive** — Go static binaries occasionally trip Defender heuristics. Whitelist `%LOCALAPPDATA%\Programs\archcore` or report the detection to the AV vendor. Code-signed builds are on the roadmap.
