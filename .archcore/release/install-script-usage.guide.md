---
title: "Installing Archcore CLI via install.sh and install.ps1"
status: accepted
tags:
  - "component:cli"
  - "release"
---

## Purpose

Install the Archcore CLI on macOS, Linux, Windows, or WSL with the published install scripts. The scripts also install the Archcore plugin when Claude Code, Codex CLI, GitHub Copilot CLI, or OpenCode is already on `PATH`.

## Prerequisites

- macOS and Linux: `curl`, `tar`, `sha256sum` or `shasum`, and `bash`. The script needs bash, not POSIX `sh`. On a minimal distribution such as Alpine or distroless, install it first, for example with `apk add bash ca-certificates`.
- Windows: PowerShell 5.1 or later.
- WSL: use the macOS and Linux path inside the WSL terminal.

## Procedure

Run `curl -fsSL https://archcore.ai/install.sh | bash` on macOS, Linux, and WSL. Run `irm https://archcore.ai/install.ps1 | iex` on Windows.

Set an environment variable before the pipeline: `ARCHCORE_VERSION` pins a release tag and skips the version lookup, `ARCHCORE_INSTALL_DIR` overrides the install directory, `ARCHCORE_SKIP_PLUGIN_INSTALL` set to a nonzero value installs only the CLI, and `GITHUB_TOKEN` authenticates asset downloads from a private repository. Version resolution never uses the token. On Windows, use PowerShell syntax, for example `$env:ARCHCORE_VERSION = 'v1.0.0'`. The installer adds its directory to the user `PATH`; open a new PowerShell window afterwards.

## What the scripts do

1. Detect the operating system (`darwin`, `linux`, `windows`) and the architecture (`amd64`, `arm64`).
2. Resolve the latest version by reading the `Location` header of `https://github.com/archcore-ai/archcore/releases/latest`, or skip the lookup when `ARCHCORE_VERSION` is set. The GitHub REST API is avoided deliberately; the related ADR records that decision. Every tag releases the plugin and the CLI together, so the redirect always lands on a release that carries the CLI assets.
3. Download the platform archive and `checksums.txt`, and verify the SHA-256 checksum.
4. Extract the binary and install it atomically into the install directory.
5. Send one anonymous `cli_installed` event, then print a one-line notice. An opt-out or a script copy without an injected key skips this step silently.
6. For each Claude Code, Codex CLI, GitHub Copilot CLI, or OpenCode command already on `PATH`, call the new binary's `archcore plugin install --agent <id>` action. The action uses user scope, checks for an existing installation, and leaves project files untouched. OpenCode gets `@archcore/opencode-plugin` in its global config. The scripts skip this step in CI or when `ARCHCORE_SKIP_PLUGIN_INSTALL` is set to a nonzero value. Cursor still requires its UI.
   - A pinned CLI of v0.14.3 or older cannot install OpenCode from the installer. The scripts skip OpenCode with a notice, and the run does not fail.
7. If a host plugin action fails, print its retry command and exit nonzero after trying the other detected hosts. The CLI remains installed, and the scripts send no `cli_install_failed` event. A CLI without installer mode, such as a pinned v0.10.10, skips the plugin step with a warning.

Run `archcore init` later in each repository where you want `.archcore/`, MCP configuration, and lifecycle hooks. The machine-level plugin install does not initialize a project. GitHub Copilot's plugin cannot provide a project MCP server by itself, so Copilot needs this project step before document tools work.

## After the install: the CLI keeps itself current

The installed binary updates itself. A user runs `archcore update` for the manual path. The MCP server runs an unattended attempt in the background of a session, 60 s after it begins serving. `self-update-command.doc` describes both paths and the conditions the unattended one requires.

No environment variable disables unattended update. To stop a machine from replacing its own binary, make the install directory writable only by root: the CLI checks write access before it downloads anything, and reports a skip.

An installer with a pinned `ARCHCORE_VERSION` does not pin the installed binary. The pin selects what to download; the installed binary still updates itself afterwards.

## Install analytics

The installers published at `archcore.ai` send one anonymous event when an install finishes or fails; the policy is at `https://archcore.ai/privacy`, and the decision is `install-analytics-via-installer-beacon.adr`. The installed CLI reports its own update outcomes under the same identifier — `cli-update-telemetry.spec`.

- The event is `cli_installed` or `cli_install_failed`. A failure carries a `stage` category only. No error message, path, directory name, user name, hostname, or repository data is sent.
- The `distinct_id` is a random 32-character hexadecimal value stored at `${XDG_STATE_HOME:-$HOME/.local/state}/archcore/install-id` on Unix and `%USERPROFILE%\.local\state\archcore\install-id` on Windows. It is not derived from the hostname, user name, or hardware. Delete the file to get a new one. Do not move it to a platform-idiomatic location such as `%LOCALAPPDATA%`: the CLI reads the same path through `StateDir` in `@cli/internal/xdg/xdg.go`, so one machine counts once.
- To opt out, set `DO_NOT_TRACK=1` or `ARCHCORE_TELEMETRY_OPTOUT=1` before the installer. The installer sends no event, writes no identifier file, and prints no notice. Set the variable in the CLI's environment too to keep the installed binary silent. Neither variable affects updating.
- A copy from the repository sends nothing. `@cli/install.sh` and `@cli/install.ps1` carry a `__POSTHOG_KEY__` placeholder, and the `archcore-ai/landing` deploy substitutes the real key. Both installers send only when the key has the `phc_` prefix.
- Do not commit a PostHog key to this repository. Keep exactly one `__POSTHOG_KEY__` per script, because the landing deploy fails on any other count. Test for the `phc_` prefix, never for the placeholder text, so the substitution cannot rewrite its own off-switch.
- The CLI applies the same rule to its own key, which `@cli/.goreleaser.yaml` injects at link time. `@cli/scripts/assert-not-inert.sh` fails the release when the official pipeline produces a silent binary.

## Verification

Run `archcore --version`. Expected result: the installed tag with its `v` prefix, for example `v0.10.2`.

## Troubleshooting

- `command not found` after the install — the install directory is not in `$PATH`. The script prints instructions for bash, zsh, and fish.
- `Could not reach https://github.com/…/releases/latest` — a network, proxy, or DNS problem. No API rate limit is involved, so `GITHUB_TOKEN` does not help. Pin a version instead: `ARCHCORE_VERSION=x.y.z`.
- `Could not resolve the latest version … (unexpected response)` — `github.com` answered without the expected `/releases/tag/` redirect. A captive portal or a proxy interstitial usually intercepts the request; the repository may also have no published release yet. Pin a version to bypass the lookup.
- `Checksum verification failed` — the download was corrupted. Run the install again.
- `Unsupported operating system/architecture` — only `darwin`, `linux`, and `windows` on `amd64` and `arm64` are supported. On another target, no binary ships; build from source with `go build -o archcore .` inside `cli/`.
- `this installer requires bash` — the script was piped into `sh` or another POSIX shell. Use `| bash`. On Debian and Ubuntu, `/bin/sh` is dash, which does not support `set -o pipefail`.
- Windows SmartScreen blocks the binary — the installer calls `Unblock-File` on the downloaded executable, so this is not expected. IF it happens, THEN right-click `archcore.exe`, open Properties, select Unblock, or run the installer again.
- Windows antivirus false positive — a Go static binary occasionally trips Defender heuristics. Add `%LOCALAPPDATA%\Programs\archcore` to the allowlist, or report the detection to the antivirus vendor. Code-signed builds are planned, not implemented.
- The install succeeds but no analytics notice appears — the script has no injected key, or an opt-out variable is set. This never affects the install.
- The install analytics show no events after a release — check that the `POSTHOG_KEY` repository variable is still set on `archcore-ai/landing`. Its deploy fails loudly when the variable is missing or when the placeholder count is wrong.
- A Dockerfile or devcontainer build fails at the plugin step — the build image has a host CLI on `PATH` and sets none of the CI variables, so the installer tries to install the plugin over the network. Set `ARCHCORE_SKIP_PLUGIN_INSTALL=1` in automation.
- A binary built from source never updates itself — that build carries no official-build marker. Reinstall with the install script, or run `archcore update` by hand.
