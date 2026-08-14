---
title: "Rollout: Unattended Update and Update Telemetry"
status: draft
tags:
  - "cli"
  - "release"
  - "telemetry"
  - "update"
---

## Goal

Ship the update path in two releases: telemetry on the manual path first, the unattended path second. The order is deliberate — the first release proves the key injection, the shared identifier, the endpoint, and the published disclosure while a human is still the one starting every update. Only then does a binary start replacing itself with nobody watching.

## Tasks

### Phase 0 — Prerequisites (blocks everything)

1. Publish the `/privacy` update: what the binary sends, both opt-out variables, and the statement that the CLI updates itself from `archcore mcp`. `archcore-ai/landing`.
2. Declare `cli_updated`, `cli_update_failed`, and `cli_update_skipped` under `ExternalAnalyticsEventMap` — `src/lib/analytics/events.ts` in `archcore-ai/landing`.
3. Add the PostHog project key as a release secret and reference it from the build workflow. It MUST NOT enter this repository.

### Phase 1 — Telemetry on the manual path (release N)

4. Create `internal/telemetry`: a package-level key variable, the three guards in order (key prefix, `DO_NOT_TRACK`, `ARCHCORE_TELEMETRY_OPTOUT`), a capture call bounded by connect and total timeouts, and a fire-and-discard error path.
5. Add identifier resolution reading and creating `${XDG_STATE_HOME:-$HOME/.local/state}/archcore/install-id` in the installers' 32-character lowercase hex format — mirror `install_id_path()` in `@install.sh`.
6. Add `-X` key injection to the release build — `@.goreleaser.yaml`.
7. Wire `cli_updated` and `cli_update_failed` into the manual path with `trigger` set to `manual`, and add the post-delivery disclosure line — `@cmd/update.go`.
8. Add stage classification over the failure points of `@internal/update/update.go`: `check`, `download`, `checksum`, `extract`, `replace`.
9. Tests: each guard suppresses the send; a build with no key sends nothing and creates no identifier file; a failed send leaves the exit code and the output unchanged; the disclosure prints only after a 2xx.

### Phase 2 — Unattended policy, no caller (release N or N+1)

10. Add a fail-closed claim to `@internal/stamp/stamp.go` for the update scope. The existing `Claim` returns `true` on a failed `MkdirAll`, on any non-`ErrExist` open error, and on a failed `Chtimes` — correct for hooks, and the inverse of what a binary replacement needs.
11. Add the policy entry point to `internal/update`: conditions in the fixed order `dev` → CI → claim → opt-out, then cache read, then write-access check.
12. Wire the claim scope keyed by the resolved binary path with a 24 h window.
13. Reuse `updateCheckCachePath()` and `updateCheckTTL` for the freshness read; make `writeCachedLatest` write through a temporary file and rename, replacing the bare `os.WriteFile` at `@cmd/update.go:83`.
14. Bound the whole run at 120 s, and sweep `<base>.tmp.*` in the target directory before writing. The Windows `<target>.old` leftover needs no work — `atomicReplace` already removes it.
15. Wire `cli_updated`, `cli_update_failed`, and `cli_update_skipped` with `trigger` set to `auto`; emit `skipped` only for `optout`, `current`, and `not_writable`.
16. Tests: a `dev` version refuses first and takes no claim; a set CI variable refuses; a read-only state directory refuses and sends nothing; two concurrent callers produce one attempt; an opt-out emits exactly one `skipped` per window; a refusal never emits `cli_update_failed`.

### Phase 3 — Triggers (release N+1)

17. Start the background goroutine after `RunStdio` begins serving, with a 60 s delay, once per process, exiting on context cancellation — `@cmd/mcp.go:50`.
18. Write exactly one stderr line on a completed replacement, and nothing otherwise.
19. Add the cached advisory line to the `doctor` report, reusing `runUpdateCheck` — `@cmd/doctor.go`.
20. Tests: a context cancelled before the delay makes no network call and writes no file; stdout stays byte-identical across a session where an update runs; a `doctor` run with a stale cache and no network prints no advisory and exits unchanged.

### Phase 4 — Verify the mechanism ran

21. Build the PostHog queries: replacement rate per release, failure rate by `stage`, `skipped` split by `reason`, `manual` versus `auto` share, and the plugin advisory's impression trend as the control signal.
22. Update `self-update-command.guide.md` and `install-script-usage.guide.md` for the update behavior and both variables.

### Deferred

23. The install provenance receipt and the package-manager refusal condition activate with the first Homebrew tap, Scoop manifest, or winget package. Not part of either release above.

## Acceptance Criteria

- A binary built with `go build` makes no network request on any path and creates no identifier file.
- Two MCP servers started within one 24 h window on one machine produce at most one binary replacement, and a machine whose state directory is unwritable produces none.
- Across a session where the goroutine runs, replaces the binary, and writes its stderr line, the JSON-RPC stream on stdout is byte-identical to a session where it did not run.
- `ARCHCORE_TELEMETRY_OPTOUT` leaves updates working; `ARCHCORE_NO_AUTO_UPDATE` leaves telemetry working.
- `archcore update` typed by a user produces the same output and exit codes as the release before phase 1, plus at most one disclosure line.
- The `/privacy` page describes the events and the update behavior before the release that emits them is downloadable.

## Dependencies

- Phase 0 task 3 blocks task 6; without the key the release build ships an inert binary.
- Phase 0 task 1 blocks the phase 3 release, not the phase 1 release: the manual path prints its own disclosure, the unattended path cannot.
- Task 10 blocks task 12; wiring the existing fail-open `Claim` into the policy would ship the exclusivity defect rather than the invariant.
- Phase 2 blocks phase 3 entirely — the trigger has nothing to call without the policy.
- Task 13 must land before task 17; a torn cache file read by a background goroutine on a shared machine is the one concurrency hazard the current `os.WriteFile` leaves open.
- [assumption] the first `cli_updated` events arrive one release after phase 1 ships, because the release carrying the code cannot report its own installation.
