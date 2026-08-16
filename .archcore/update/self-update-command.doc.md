---
title: "How archcore update Works"
status: accepted
tags:
  - "update"
---

## Overview

Explain how the Archcore CLI keeps itself current, how to run `archcore update`, and how to read its failures.

The CLI updates itself on two paths. A user types `archcore update`, which is the manual path. The MCP server runs the unattended path in the background of a session. Both replace the same binary through the same code; they differ in what must hold before the replacement starts and in what each reports.

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

Both transcripts show a machine without the Archcore plugin. A machine that carries the plugin adds the lines described under "Plugin update step".

## How the update runs

This section describes current CLI behavior, not steps for the reader.

1. Check the latest version. The CLI sends `GET https://github.com/archcore-ai/cli/releases/latest` and reads the tag from the redirect's `Location` without following it. Any `3xx` is accepted: GitHub answers `302` today, and the tag-page check, not the status code, is the real gate. The tag comes from the resolved URL's path (`resp.Location()`), so a query string, a fragment, or a `..` segment in the `Location` cannot smuggle a false version through.

   The REST API is avoided deliberately: it is capped at 60 unauthenticated requests per hour per IP, which teams sharing one egress address exhaust. The related ADR records that decision.

2. Compare versions. The current version, injected via ldflags at build time, is compared with the latest by semver order (major, then minor, then patch). A development build, where `version = "dev"`, always triggers an update on the manual path. A version that neither side can parse reports as current rather than as newer, so an unexpected tag never becomes a downgrade.

3. Download the archive. WHEN a newer version exists, the CLI downloads the platform-specific archive from `https://github.com/archcore-ai/cli/releases/download/<version>/archcore_<os>_<arch>.<ext>`. The extension is `.zip` on Windows and `.tar.gz` on every other platform, matching `format_overrides` in `@.goreleaser.yaml`. The platform comes from `runtime.GOOS` and `runtime.GOARCH`.

4. Verify the checksum. The CLI downloads `checksums.txt` from the same release, computes the SHA-256 of the downloaded archive, and compares it against the expected hash. A mismatch fails the update immediately.

5. Extract the binary. The archive format, `tar.gz` or `zip`, is detected from the magic bytes. The CLI tries the name `archcore` first and falls back to the repository basename `cli`, because GoReleaser may use either. On Windows both candidates carry an `.exe` suffix.

6. Stage the new binary. The CLI resolves the current binary path via `os.Executable()` and `filepath.EvalSymlinks()`, then writes the new binary to `<binary>.tmp.<pid>` with permissions `0755`. The temporary name carries the process id so two attempts on one machine cannot truncate each other's staged file. Leftovers from a killed attempt are swept before the next write.

7. Probe the staged binary, on the unattended path only. The CLI runs the staged file with `--version` under a 3 s bound. A failure abandons the replacement and reports the failure as stage `replace`, so a corrupt download never reaches the target path. The manual path runs no probe: a user watching the command can retry it.

8. Replace the binary atomically. The staged file is renamed over the current binary.

   On Windows a running `.exe` cannot be overwritten in place, so the current binary is renamed to `<binary>.old.<pid>` first and the new file moves in. The `.old.<pid>` files are removed on a best-effort basis and swept by the next attempt, so a second update while an older server still holds its image does not collide.

## Unattended update from the MCP server

`archcore mcp` starts one background attempt 60 s after it begins serving. The attempt runs on its own goroutine, writes nothing to stdout, and never delays a JSON-RPC response. `unattended-update.spec` and `mcp-background-update.spec` carry the normative behavior.

Every condition below must hold, in this order, before the CLI replaces anything:

1. The binary carries the official-build marker, injected only by this repository's release workflow.
2. The running version is not `dev`.
3. No CI environment variable is set.
4. No other process holds the 24 h claim for this binary path.
5. The install directory is writable by this process.
6. The latest version parses and is strictly newer.

A refusal at conditions 1 to 3 is silent and takes no claim. A machine whose install directory is root-owned reports a skip and pays no bandwidth; making the directory root-owned is the supported way to stop a machine from updating itself.

The running process keeps executing the image it started with. The new version takes effect at the next launch — the next session, the next hook invocation, or the next command a user types. WHEN a replacement completes, the server writes one line to stderr, which lands in the host's server log.

There is no environment variable that disables unattended update. `DO_NOT_TRACK` and `ARCHCORE_TELEMETRY_OPTOUT` stop telemetry and leave updates working.

## Plugin update step

After the binary phase, manual `archcore update` refreshes the Archcore plugin on each host that already carries it. `updating-the-plugin.spec` carries the normative behavior.

- The step runs after a replacement and after an already-current result. A failed binary phase skips it.
- `archcore update --check` never runs it.
- The step queries each host's own read-only plugin listing before any mutating command, and runs an update command only for a listed plugin. A host without the plugin produces no output.
- The step never changes the exit code of `archcore update` and never sends a telemetry event.
- The unattended path and the MCP trigger never reach this step.

## Telemetry

The update paths report three events to the Archcore endpoint: `cli_updated`, `cli_update_failed` with the failed `stage`, and `cli_update_skipped` with the skip `reason`. A `trigger` property separates a typed `manual` invocation from an `auto` one. `cli-update-telemetry.spec` carries the contract.

- No event carries an error message, a path, a directory name, a user name, a host name, or repository data.
- Set `DO_NOT_TRACK` or `ARCHCORE_TELEMETRY_OPTOUT` to a value other than `0` to stop every event. Both are read before any filesystem access, so an opted-out machine also creates no identifier file.
- A build without an injected key sends nothing.
- WHEN a manual event is delivered, `archcore update` prints one disclosure line naming the opt-out variable and the privacy page.

## Cached advisory

`archcore update --check` is the hook-facing probe. It caches the answer for 24 h, holds a short network timeout, always exits 0, and sends no event. A failed lookup writes a negative stamp that suppresses retries for 1 h.

`archcore doctor` reads that cache and prints an advisory when it holds a newer version. The advisory makes no network call and does not count as an issue.

## Code structure

| File | Purpose |
|------|---------|
| `@internal/update/update.go` | The `Updater` struct: `CheckLatest`, `NeedsUpdate`, `NewerSemver`, `Apply`, `VerifyChecksum`, `ExtractBinary`, `HealthProbe` |
| `@internal/update/policy.go` | `RunUnattended` — the unattended conditions in their fixed order |
| `@internal/update/cache.go` | The freshness cache: path, 24 h TTL, 1 h negative TTL |
| `@internal/update/buildinfo.go` | The official-build marker |
| `@internal/update/stage.go` | `StageError` — the failed step, carried as a typed value |
| `@internal/telemetry/telemetry.go` | The event sender and its three guards |
| `@internal/plugin/` | The plugin engine: evidence, planner, executor |
| `@cmd/update.go` | Cobra command wiring, styled output, and the plugin step |
| `@cmd/mcp.go` | The background trigger: the 60 s delay and the stderr line |

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
- A development build always updates on the manual path. WHILE the binary reports `archcore vdev`, the command downloads the latest release on every run. The unattended path refuses a `dev` build instead.
- A machine that never self-updates in the background — check the conditions under "Unattended update from the MCP server" in order. A binary built with `go build` carries no official-build marker and stops at the first condition, which is the intended behavior for a local build.
