---
title: "Rollout: Unattended Update, Telemetry, Plugin Update and Delivery"
status: draft
tags:
  - "cli"
  - "release"
  - "telemetry"
  - "update"
---

## Goal

One release ships the whole update-and-delivery path: the unattended update policy, the MCP background trigger, the three telemetry events, a plugin-update step in the manual `archcore update`, and plugin delivery — `archcore plugin` plus selection-driven delivery in `archcore init`. The phases below order the implementation inside that one release. Unattended update ships with no opt-out surface; releases are gated by a health probe, an official-build marker, and a non-inertness check instead of signing (decision 2026-08-15, recorded in the RFC).

## Tasks

### Phase 0 — Prerequisites (blocks everything)

1. Publish the `/privacy` update: what the binary sends, the two telemetry opt-out variables (`DO_NOT_TRACK`, `ARCHCORE_TELEMETRY_OPTOUT`), the statement that the CLI updates itself from `archcore mcp`, the disclosure that unattended update cannot be switched off, and the root-owned-directory operator answer. `archcore-ai/landing`.
2. Declare `cli_updated`, `cli_update_failed`, and `cli_update_skipped` under `ExternalAnalyticsEventMap` — `src/lib/analytics/events.ts` in `archcore-ai/landing`.
3. Add the PostHog project key as a release secret and reference it from the build workflow; the key stays out of this repository.
4. Add the official-build marker: a second `-X` ldflags variable injected only by `@.github/workflows/release.yml` via `@.goreleaser.yaml`; the policy refuses without it.
5. Add a release-pipeline assertion that the built artifact carries the key and the marker — one CI step that fails the release when the artifact would ship inert.

### Phase 1 — Telemetry on the manual path

6. Create `internal/telemetry`: a package-level key variable, the three guards in order (key prefix, `DO_NOT_TRACK`, `ARCHCORE_TELEMETRY_OPTOUT`), a capture call bounded by connect and total timeouts, and a fire-and-discard error path.
7. Extract one XDG state-path helper for the Go side and use it for the `install-id` resolution (installers' 32-character lowercase hex format — mirror `install_id_path()` in `@install.sh`). `stamp.DirFor` and the freshness cache path adopt the same helper; today three copies re-derive `${XDG_STATE_HOME:-$HOME/.local/state}/archcore`.
8. Add `-X` key injection to the release build — `@.goreleaser.yaml`.
9. Wire `cli_updated` and `cli_update_failed` into the manual path with `trigger` set to `manual`, and add the post-delivery disclosure line — `@cmd/update.go`. Thread the manual path's output through `cmd.OutOrStdout()` while touching it.
10. Add stage classification as a typed carrier: a `StageError` wrapped at each failure point of `@internal/update/update.go`, read via `errors.As` at both call sites. Mapping: `CheckLatest` → `check`; both `download()` calls → `download`; `VerifyChecksum` → `checksum`; `ExtractBinary` → `extract`; exec-path resolution and `atomicReplace` → `replace`.
11. Tests: each guard suppresses the send; a build with no key sends nothing and creates no identifier file; a failed send leaves the exit code and the output unchanged; the disclosure prints only after a 2xx; every `Apply` failure classifies to its stage.

### Phase 2 — Unattended policy, no caller

12. Add a fail-closed claim to `@internal/stamp/stamp.go` under a bias-carrying exported name, in its own scope directory. It inverts all four fail-open returns of the existing `Claim` path — the failed `MkdirAll`, the non-`ErrExist` open error, the lock open error in `reclaim`, and the failed `Chtimes` — and refuses on an empty `DirFor` result. Update the package doc, which today declares the whole package fail-open.
13. Move the freshness cache into `internal/update` as the canonical implementation — path, 24 h TTL, 1 h negative TTL, and a pid-suffixed temp-file-and-rename write replacing the bare `os.WriteFile` at `@cmd/update.go:83`. `--check` and the `doctor` advisory call through it. Rationale: the policy lives in `internal/update`, and `cmd` importing it back is a cycle.
14. Add the policy entry point to `internal/update`: conditions in the fixed order marker → `dev` → CI → claim, then cache read, then write-access check. A fresh failure stamp counts as stale and proceeds to the lookup; only a strictly newer, parseable semver replaces.
15. Wire the claim scope keyed by the resolved binary path with a 24 h window.
16. Add the pre-rename health probe: run the extracted binary with `--version` under a 3 s bound; a failure abandons the replacement as stage `replace`.
17. Make the Windows aside name per-attempt — `<target>.old.<pid>` — and sweep `<target>.old.*`, so a second update while an older server survives stops colliding.
18. Bound the whole run at 120 s and sweep `<base>.tmp.*` before writing; the ceiling governs the network-bound stages and never interrupts the synced write or the rename.
19. Wire `cli_updated`, `cli_update_failed`, and `cli_update_skipped` with `trigger` set to `auto`; emit `skipped` only for `current` and `not_writable`.
20. Tests: a marker-less build refuses first; a `dev` version refuses and takes no claim; a set CI variable refuses; a read-only state directory refuses and sends nothing; two concurrent callers produce one attempt; a fresh failure stamp leads to a lookup, not `skipped(current)`; an unparseable tag refuses without an event; a probe failure abandons as `replace`; a current cache emits exactly one `skipped` per claim window; a refusal never emits `cli_update_failed`.

### Phase 3 — Triggers

21. Add `WithBackgroundTask(func(ctx))` as a `ServerOption` in `internal/mcp`, started by `RunStdio` between `shieldStdout()` and `Listen` (`@internal/mcp/server.go:265-268`); `cmd/mcp.go` supplies the closure — 60 s delay, policy invocation, once per process, exit on context cancellation. The update stack stays out of `internal/mcp`.
22. Write exactly one stderr line on a completed replacement, and nothing otherwise.
23. Add the cached advisory line to the `doctor` report: read the cache through `internal/update`, wrap the line in `display.WarnLine` at the `doctor` call site, and leave the `issues` counter untouched — `@cmd/doctor.go`.
24. Tests: a context cancelled before the delay makes no network call and writes no file; stdout stays byte-identical across a session where an update runs; `RunStdio` without the option serves exactly as today; a `doctor` run with a stale cache and no network prints no advisory, exits unchanged, and never counts the advisory as an issue.

### Phase 4 — Plugin engine and the update step

The engine opens the plugin work: task 25 lands before every entry point that consumes it. The update step reaches only the manual command. The unattended policy and the MCP trigger never call it, it never runs on `--check`, and it emits no telemetry event. Only hosts with a shipping plugin appear: Claude Code, GitHub Copilot, Codex CLI, and Cursor.

25. Extract the shared engine `internal/plugin`: the frozen identifiers, a pure planner (host evidence → per-host actions), and one executor on the seam pattern of task 29. `archcore update`'s step, `archcore plugin`, and the init step all run this engine and differ only in action selection and wording.
26. Wire the engine's update action into `@cmd/update.go` as an extracted testable function, with one `display.Dim` progress line per host before its command runs. It runs after the binary phase — after a replacement and after an already-current result — and is skipped when the binary phase failed.
27. Evidence-first tiers: with the host CLI on `PATH`, query its read-only listing (`claude plugin list --json`, `copilot plugin list`, `codex plugin list --json`) and run the update command only for a listed plugin; with the CLI absent, print the exact command when the on-disk registry lists the plugin; Cursor prints the UI instruction on registry evidence; otherwise silent. Registry reads sit inside the clause 3 plugin-surface carve-out.
28. Implement the per-host update commands (probed live 2026-08-15 against claude 2.1.232, copilot 1.0.76, codex 0.147.0):

| Host | Action |
|---|---|
| Claude Code | `claude plugin marketplace update archcore-plugins`, then `claude plugin update archcore@archcore-plugins`; append `-y` for non-TTY safety [assumption]. User-scope record only this release. |
| GitHub Copilot | `copilot plugin update archcore@archcore-plugins`. The binary is often not on `PATH` (VS Code-managed install). |
| Codex CLI | `codex plugin marketplace upgrade archcore-plugins`. The marketplace snapshot refresh is the update. |
| Cursor | No CLI mechanism (UI-only). Print the one-line UI instruction. |

29. Run each command through the `internal/git` seam pattern — package-level `lookPath`, `exec.CommandContext`, `WaitDelay`, capped output — but capture stderr, unlike git's discard: the listing parse and the failure print both need it. 30 s per command, 120 s for the whole step.
30. Failure handling: a listing that fails or does not parse skips the host silently; a nonzero update command or a timeout prints the exact command; the exit code of `archcore update` never changes.
31. Verify the implementation carries the three identifiers exactly as requirement 11 of `plugin-cli-compatibility.rule.md` freezes them: repo `archcore-ai/plugin`, marketplace `archcore-plugins`, plugin id `archcore@archcore-plugins`.
32. Tests: a host CLI present whose listing shows no plugin runs no mutating command and prints nothing; a registry-listed plugin without the CLI prints the command; a host with no evidence produces no output; a failed or timed-out update command prints the exact command and leaves the exit code unchanged; the unattended path runs zero plugin commands. Note: CI runs the suite on Linux only — Windows marker paths and `LookPath` behavior ship untested there.

### Phase 5 — Plugin delivery: `archcore plugin` and the init step

33. Add `archcore plugin install|update|remove|status [--agent <id>] [--project <path>]` with the constructor-command pattern; register it in `@cmd/root.go`. `--project` satisfies the tree-walking flag test (`@cmd/project_root_flag_test.go`) and feeds only `--scope project` writes; an `--agent` value without a shipping plugin errors naming the four supported hosts. A typed verb is consent and runs non-interactively; an attempted-action failure exits nonzero; `status` exits zero.
34. Wire selection-driven delivery into init: mark plugin-capable hosts on the existing agent multi-select with the plugin disclosure (Codex CLI and GitHub Copilot named machine-level); a checked host installs after wiring with no second prompt; `--agent` carries the consent non-interactively as it does for wiring; `--yes` without `--agent` and CI environments print the per-host commands and run nothing; an install over a listed plugin is a reported no-op.
35. Claude Code install: `claude plugin marketplace add archcore-ai/plugin`, then `claude plugin install archcore@archcore-plugins`; merge the `autoUpdate: true` marketplace entry into `~/.claude/settings.json` through `internal/jsonfile` (`ReadOrBackup` + the `mergeEntryFields` pattern of `@internal/agents/mcp_helpers.go`); `--scope project` is the opt-in that writes the committed file, disclosed at write time.
36. Copilot and Codex installs per the delivery spec table (`copilot plugin install archcore-ai/plugin:plugins/archcore` [assumption on subpath]; `codex plugin marketplace add archcore-ai/plugin` + `codex plugin add archcore@archcore-plugins`); Cursor prints the UI instruction.
37. Tests: `--yes` without `--agent` and CI runs execute zero plugin subprocesses and print the commands; a non-interactive `init --agent claude-code` installs; a rerun over an installed plugin reports it and changes nothing; a delivery failure leaves init's exit code unchanged; a direct `archcore plugin install` failure exits nonzero; `archcore update`'s step and `archcore plugin update` produce identical plans (the planner/executor split makes this a plan comparison); `plugin remove` removes the `autoUpdate` entry it wrote; `--agent gemini-cli` errors naming only the four plugin hosts.

### Phase 6 — Verify the mechanism ran

38. Build the PostHog queries: replacement rate per release, failure rate by `stage`, `skipped` split by `reason`, `manual` versus `auto` share, and the plugin advisory's impression trend as the control signal.
39. Update `self-update-command.doc.md`, `install-script-usage.guide.md`, `supported-ai-agents.doc.md`, and the command list in `cli-ui/building-the-cli.doc.md` (nine commands become ten) for the update behavior, the two telemetry variables, the plugin step, and the delivery surface. These accepted documents change in the release commit, not before.
40. Coordinate the one-install-command narrative in `archcore-ai/landing`: install CTAs and the install tabs describe CLI-first with plugin delivery. Release-adjacent, not release-blocking.

### Deferred

41. The install provenance receipt stays deferred whole (decision 2026-08-15), on three grounds: no package-manager channel exists yet, the marker and write-access bounds remain, and `/privacy` discloses the no-opt-out behavior. The receipt activates with the first package-manager channel — a Homebrew tap, a Scoop manifest, or a winget package.

## Acceptance Criteria

- A binary built with `go build` makes no network request on any path, creates no identifier file, and never self-replaces.
- A stamped build without the official marker never self-replaces.
- A release whose artifact lacks the key or the marker fails the pipeline before it is downloadable.
- Two MCP servers started within one 24 h window on one machine produce at most one binary replacement, and a machine whose state directory is unwritable produces none.
- A downloaded binary that fails `--version` never reaches the target path.
- Across a session where the goroutine runs, replaces the binary, and writes its stderr line, the JSON-RPC stream on stdout is byte-identical to a session where it did not run.
- No environment variable disables unattended update; `DO_NOT_TRACK` and `ARCHCORE_TELEMETRY_OPTOUT` leave updates working.
- `archcore update` typed by a user keeps its exit codes; its output adds at most one disclosure line, the per-host progress lines, and the plugin-step lines.
- A user who never installed the plugin sees no plugin output and pays no mutating host command.
- A failed or timed-out plugin command prints the exact command and leaves the exit code of `archcore update` unchanged.
- A background replacement runs zero plugin commands.
- Selecting a plugin-capable host in interactive `archcore init` installs its plugin with no second prompt; the selection screen carries the disclosure.
- `archcore init --yes` without `--agent` and any CI run execute zero plugin subprocesses and print the per-host commands; a non-interactive `init --agent <host>` installs for that host.
- A second `archcore init` over an installed plugin reports it and changes nothing.
- A delivery failure never changes the exit code of `archcore init`.
- The `/privacy` page describes the events, the update behavior, and the absence of an update opt-out before the release that emits them is downloadable.

## Dependencies

- Task 1 blocks the release: the `/privacy` page carries the only disclosure surface the unattended path has.
- Task 3 blocks task 8; task 4 blocks task 14 — without the marker the policy has nothing to check.
- Task 12 blocks task 15; wiring the existing fail-open `Claim` into the policy would ship the exclusivity defect rather than the invariant.
- Task 13 blocks task 14 (the policy reads the cache from its own package) and lands before task 21 — a torn cache file read by a background goroutine is the one concurrency hazard the current `os.WriteFile` leaves open.
- Phase 2 blocks phase 3 entirely — the trigger has nothing to call without the policy.
- Task 25 blocks tasks 26, 33, 34 — the engine opens the plugin work and lands before its three entry points.
- Phases 4–5 depend on no telemetry or policy task; they may run before or after phases 1–3.
- [assumption] the first `cli_updated` events arrive one release after this one ships, because the release carrying the code cannot report its own installation.