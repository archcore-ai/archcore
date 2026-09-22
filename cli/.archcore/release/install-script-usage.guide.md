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
7. Send one anonymous install event, then print a one-line notice that the event was sent. See "Install analytics" below. An opt-out or a script copy without an injected key skips this step silently.

## After the install: the CLI keeps itself current

The installed binary updates itself. A user runs `archcore update` for the manual path. The MCP server runs an unattended attempt in the background of a session, 60 s after it begins serving. `self-update-command.doc` describes both paths and the conditions the unattended one requires.

No environment variable disables unattended update. To stop a machine from replacing its own binary, make the install directory writable only by root: the CLI checks write access before it downloads anything, and reports a skip.

An installer with a pinned `ARCHCORE_VERSION` does not pin the installed binary. The pin selects what to download; the installed binary still updates itself afterwards.

## Environment variables

| Variable | Default (Unix) | Default (Windows) | Description |
|---|---|---|---|
| `ARCHCORE_VERSION` | (latest) | (latest) | Pin a specific release tag, for example `v1.0.0`. Skips the version lookup. |
| `ARCHCORE_INSTALL_DIR` | `~/.local/bin` | `%LOCALAPPDATA%\Programs\archcore` | Override the install directory. |
| `GITHUB_TOKEN` | (none) | (none) | GitHub token for authenticated asset downloads from a private repository. Version resolution does not use it; that reads a public redirect with no rate limit. |
| `DO_NOT_TRACK` | (none) | (none) | Set to any value other than `0` to disable analytics in the installer and in the installed CLI. The [consoledonottrack.com](https://consoledonottrack.com) convention. |
| `ARCHCORE_TELEMETRY_OPTOUT` | (none) | (none) | Set to any value other than `0` to disable analytics in the installer and in the installed CLI. Tool-specific equivalent of `DO_NOT_TRACK`. |

On Windows, set an environment variable with PowerShell syntax before the `irm | iex` pipeline: `$env:ARCHCORE_VERSION = 'v1.0.0'`.

Neither opt-out variable affects updating. Both stop analytics and leave the update paths working.

## Install analytics

The installers published at `https://archcore.ai/install.sh` and `https://archcore.ai/install.ps1` send one anonymous event when an install finishes or fails. The user-facing policy is at `https://archcore.ai/privacy`.

The installed CLI reports its own update outcomes under the same two opt-out variables and the same install identifier. `cli-update-telemetry.spec` carries that contract; this guide covers the installer.

### What the event contains

The event name is `cli_installed` on success and `cli_install_failed` on failure.

| Property | Example | Notes |
|---|---|---|
| `archcore_version` | `1.0.0` | Absent when the run failed before version resolution. |
| `os` | `darwin` | `unknown` when the run failed before platform detection. |
| `arch` | `arm64` | `unknown` when the run failed before platform detection. |
| `installer` | `install.sh` | Or `install.ps1`. |
| `is_reinstall` | `true` | An install identifier already existed on this machine. |
| `ci` | `false` | True when `CI`, `GITHUB_ACTIONS`, `GITLAB_CI`, `BUILDKITE`, `JENKINS_URL`, or `TEAMCITY_VERSION` is set. |
| `pinned_version` | `false` | True when `ARCHCORE_VERSION` is set. |
| `install_dir_default` | `true` | False when `ARCHCORE_INSTALL_DIR` is set. |
| `stage` | `download` | Failure events only. One of `start`, `prereq`, `platform`, `version`, `download`, `checksum`, `extract`, `install`, `done`. |

The event does not contain an error message, a file path, a directory name, a user name, a hostname, or repository data. A failure reports the `stage` category only.

### The install identifier

The `distinct_id` is a random 32-character hexadecimal value. The installers store it at:

- Unix: `${XDG_STATE_HOME:-$HOME/.local/state}/archcore/install-id`
- Windows: `%USERPROFILE%\.local\state\archcore\install-id`

The value is not derived from the hostname, the user name, or any hardware identifier. Delete the file to get a new identifier.

The CLI resolves the same path through `StateDir` in `@internal/xdg/xdg.go`, which `@internal/telemetry/telemetry.go` and the freshness cache in `@internal/update/cache.go` both call. The `.local/state` layout applies on every platform including Windows. Do not move the identifier to a platform-idiomatic location such as `%LOCALAPPDATA%`. The CLI's own telemetry reads the same identifier and joins "installed" to "used" without a second identifier.

`install.sh` writes 32 lowercase hexadecimal characters from `/dev/urandom`. `install.ps1` writes `[guid]::NewGuid().ToString('N')`, which produces the same format. The CLI accepts an existing identifier verbatim and generates one in the same format when the file is absent. A machine that has used both installers therefore counts once.

### Procedure — opt out

Set either variable before you run the installer.

macOS and Linux:

```bash
DO_NOT_TRACK=1 curl -fsSL https://archcore.ai/install.sh | bash
```

Windows:

```powershell
$env:DO_NOT_TRACK = '1'; irm https://archcore.ai/install.ps1 | iex
```

Expected result: the installer sends no event, writes no identifier file, and prints no analytics notice. An opt-out leaves no trace on disk.

Set the same variable in the environment the CLI runs in to keep the installed binary silent too. The CLI reads both variables before any filesystem access, so an opted-out machine also creates no identifier file.

### Why a copy from the repository sends nothing

`@install.sh` and `@install.ps1` in this repository carry a `__POSTHOG_KEY__` placeholder in place of a PostHog project key. The `archcore-ai/landing` deploy workflow substitutes the real key while it copies the scripts into `public/`. Both installers send an event only when the key has the `phc_` prefix.

Therefore:

- A script that runs from a clone, a fork, or `@.github/workflows/install-smoke.yml` reports nothing.
- Only the copies served from `https://archcore.ai/` report.

Rules:

- Do not commit a PostHog key to this repository.
- Keep exactly one occurrence of `__POSTHOG_KEY__` per script. The landing deploy fails when the count is not 1, because a renamed placeholder produces an installer that reports nothing — a state indistinguishable from "nobody is installing".
- Do not compare against the placeholder text to detect the unsubstituted state. Test for the `phc_` prefix instead, so the substitution cannot rewrite its own off-switch.

The CLI applies the same rule to its own key, which `@.goreleaser.yaml` injects at link time. A build without an injected key sends nothing, so `go build` and a fork's pipeline both produce a silent binary. `@scripts/assert-not-inert.sh` fails the release when the official pipeline produces one by accident.

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
- The install succeeds but no analytics notice appears — the script has no injected key, or an opt-out variable is set. This never affects the install.
- The install analytics show no events after a release — check that the `POSTHOG_KEY` repository variable is still set on `archcore-ai/landing`. Its deploy fails loudly when the variable is missing or when the placeholder count is wrong.
- A binary installed with `go install` never updates itself — that build carries no official-build marker. Reinstall with the install script, or run `archcore update` by hand.
