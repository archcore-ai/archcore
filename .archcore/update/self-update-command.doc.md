---
title: "How archcore update Works"
status: accepted
tags:
  - "component:cli"
  - "update"
---

## Overview

Explain how the Archcore CLI keeps itself current, how to run `archcore update`, and how to read its failures.

The CLI updates itself on two paths. A user types `archcore update`, which is the manual path. The MCP server runs the unattended path in the background of a session. Both replace the same binary through the same code; they differ in what must hold before the replacement starts and in what each reports.

## Prerequisites

- An installed `archcore` binary, from the install script or built from source.
- Network access to `github.com` for the version check, the archive download, and `checksums.txt`. The CLI does not contact the GitHub REST API (`api.github.com`).

## How the update runs

This section describes current CLI behavior, not steps for the reader. Run `archcore update` for the manual path. A machine that carries the Archcore plugin adds the lines described under "Plugin update step".

1. Check the latest version. The CLI sends `GET https://github.com/archcore-ai/archcore/releases/latest` and reads the tag from the redirect's `Location` without following it. Any `3xx` is accepted, and the tag-page check, not the status code, is the real gate. The tag comes from the resolved URL's path (`resp.Location()`), so a query string, a fragment, or a `..` segment in the `Location` cannot smuggle a false version through. The REST API is avoided deliberately: it is capped at 60 unauthenticated requests per hour per IP, which teams sharing one egress address exhaust. The related ADR records that decision.
2. Compare versions by semver order. A development build, where `version = "dev"`, always triggers an update on the manual path. A version that neither side can parse reports as current rather than as newer, so an unexpected tag never becomes a downgrade.
3. Download the platform archive and `checksums.txt` from the same release, and verify the SHA-256. A mismatch fails the update immediately. The extension is `.zip` on Windows and `.tar.gz` elsewhere, matching `format_overrides` in `@cli/.goreleaser.yaml`.
4. Extract the binary. The archive format is detected from the magic bytes. The CLI tries the name `archcore` first and falls back to the repository basename, because GoReleaser may use either.
5. Stage the new binary at `<binary>.tmp.<pid>`. The process id keeps two attempts on one machine from truncating each other's staged file. Leftovers from a killed attempt are swept before the next write.
6. Probe the staged binary, on the manual path and on the unattended path. On macOS the CLI first reads the minimum macOS from the binary's Mach-O header (`LC_BUILD_VERSION`) and refuses a binary built for a newer system than the running one. Then the CLI runs the binary with `--version` under a 3 s bound. A failure abandons the replacement and reports stage `replace`, so neither a corrupt download nor a build for a newer system reaches the target path. The manual path probes too, because after a broken binary replaces the working one the user has no binary left to retry with.
7. Replace the binary atomically by renaming the staged file over the current one. On Windows a running `.exe` cannot be overwritten, so the current binary is renamed to `<binary>.old.<pid>` first. The `.old.<pid>` files are removed on a best-effort basis and swept by the next attempt.

Two older generations of binaries resolve a repository that no longer answers with a tag page. Binaries released as v0.8.7 and earlier resolve the former CLI repository `archcore-ai/cli`, which receives no further releases and reports "already up to date". Binaries released as v0.10.1 resolve `archcore-ai/plugin`, the name this repository carried until 2026-09-22; GitHub now answers that address with a `301` to the new name, which the resolver rejects as `unexpected redirect`. Both generations move to the current channel through a fresh run of the install script.

## Unattended update from the MCP server

`archcore mcp` starts one background attempt 60 s after it begins serving. The attempt runs on its own goroutine, writes nothing to stdout, and never delays a JSON-RPC response. `unattended-update.spec` and `mcp-background-update.spec` carry the normative behavior, including the ordered conditions that must hold before anything is replaced: the official-build marker, a non-`dev` version, no CI variable, an unheld 24 h claim, a writable install directory, and a strictly newer version.

A refusal at the first three conditions is silent and takes no claim. A machine whose install directory is root-owned reports a skip and pays no bandwidth; making the directory root-owned is the supported way to stop a machine from updating itself. There is no environment variable that disables unattended update. `DO_NOT_TRACK` and `ARCHCORE_TELEMETRY_OPTOUT` stop telemetry and leave updates working.

The running process keeps executing the image it started with. The new version takes effect at the next launch. WHEN a replacement completes, the server writes one line to stderr, which lands in the host's server log.

## Plugin update step

After the binary phase, manual `archcore update` refreshes the Archcore plugin on each host that already carries it. `updating-the-plugin.spec` carries the normative behavior.

- The step runs after a replacement and after an already-current result. A failed binary phase skips it, and `archcore update --check` never runs it.
- The step queries each host's own read-only plugin listing before any mutating command. A host without the plugin produces no output.
- Since v0.10.2 the step first migrates a Claude Code or Codex marketplace declaration that still names `archcore-ai/plugin` to `archcore-ai/archcore`, changing only the repository locator; `plugin-source-migration.spec` carries that contract.
- The step never changes the exit code of `archcore update` and never sends a telemetry event. The unattended path and the MCP trigger never reach it.

## Telemetry

The update paths report `cli_updated`, `cli_update_failed`, and `cli_update_skipped`. `cli-update-telemetry.spec` carries the contract. Set `DO_NOT_TRACK` or `ARCHCORE_TELEMETRY_OPTOUT` to a value other than `0` to stop every event; both are read before any filesystem access, so an opted-out machine also creates no identifier file. A build without an injected key sends nothing. WHEN a manual event is delivered, `archcore update` prints one disclosure line naming the opt-out variable and the privacy page.

## Cached advisory

`archcore update --check` is the hook-facing probe. It caches the answer for 24 h, holds a short network timeout, always exits 0, and sends no event. A failed lookup writes a negative stamp that suppresses retries for 1 h. `archcore doctor` reads that cache and prints an advisory when it holds a newer version. The advisory makes no network call and does not count as an issue.

## Verification

Run `archcore --version`. Expected result: the newly installed version.

## Troubleshooting

- `Could not check for updates` — a network problem, or `github.com` answered with something other than a redirect, such as a captive portal, a proxy interstitial, or an outage. Retry later. No API rate limit applies: the check reads the `github.com` web redirect, which carries no rate-limit budget and needs no token.
- `unexpected redirect resolving latest release` — the redirect landed somewhere that is not a `/releases/tag/` page. Three causes occur in practice: a proxy or captive portal intercepting the request; the repository having no published release, which GitHub answers with a `302` to the bare `/releases` page; or a binary that still carries the pre-rename name `archcore-ai/plugin`, which GitHub answers with a `301` to the new name. In the last case reinstall through the install script.
- `no Location header` or `parsing redirect location` — a `3xx` arrived without a usable `Location`. This indicates an intercepting proxy rather than GitHub.
- `Update failed` with a permission error — the binary sits in a directory without write access. Reinstall it to a writable location, or run the command with `sudo`.
- `Checksum mismatch` — the download was corrupted. Run `archcore update` again.
- `Update failed` with `probing staged binary: needs macOS X or later; this Mac runs macOS Y` — the release is built for a newer macOS. The current binary stays in place. Update macOS to run the new release.
- `Update failed` with `probing staged binary` and another reason — the downloaded binary did not start or did not answer `--version` within 3 s. The current binary stays in place. Run `archcore update` again.
- A development build always updates on the manual path. WHILE the binary reports `archcore vdev`, the command downloads the latest release on every run. The unattended path refuses a `dev` build instead.
- A machine that never self-updates in the background — check the unattended conditions in order. A binary built with `go build` carries no official-build marker and stops at the first condition, which is the intended behavior for a local build.
