---
title: "Plugin Delivery: archcore plugin and the init Step"
status: accepted
tags:
  - "cli"
  - "integrations"
  - "update"
---

## Purpose & Scope

This spec defines the plugin-delivery surface: the `archcore plugin` command and the delivery step inside `archcore init`. One engine (`internal/plugin`) performs install, update, removal, and status per host; the plugin-update step of `archcore update` (`updating-the-plugin.spec`) runs the same engine's update action. Dependents: `@cmd/init.go`, `@cmd/update.go`, the host registry in `internal/agents/`, the host CLIs, and the `archcore-ai/plugin` repository.

Out of scope: the unattended update policy and the MCP background trigger — neither reaches this surface; hook and MCP wiring, which `archcore init` performs today and keeps unchanged.

## Surface

- Commands: `archcore plugin install|update|remove|status [--agent <id>] [--project <path>]`.
- Engine shape: one pure planning function (host evidence → per-host actions) and one executor. Entry points differ only in which actions they select and how they word output.
- Selection screen: init's existing agent multi-select — the project-detection-driven list init already shows for wiring. The four plugin-capable hosts are marked inside that same list; no separate screen and no second prompt exist.
- Init integration: selecting a host in that multi-select is the consent for that host — hooks, MCP config, and the plugin arrive together.
- Frozen identifiers: repository `archcore-ai/plugin`, marketplace `archcore-plugins`, plugin id `archcore@archcore-plugins` (`plugin-cli-compatibility.rule`, requirement 11).
- Host evidence: the host CLI on `PATH` (`exec.LookPath`), the host's read-only plugin listing, and the on-disk registries named in the update-step spec. A listing shows the plugin under the definition in the Surface of `updating-the-plugin.spec`, which requirements 9 and 25 below read: a registered marketplace with nothing installed under it is not a plugin.
- Timeouts: 30 s per host command, 120 s for the whole delivery step [assumption] — the same pairing as the update step. The seam is `@internal/git/git.go` with stderr captured.
- Claude Code auto-update key: `extraKnownMarketplaces["archcore-plugins"]` with `"autoUpdate": true` in `~/.claude/settings.json` — documented Claude Code behavior; the host then refreshes the plugin in the background after session start.

Per-host install actions:

| Host | Install action | Verified |
|---|---|---|
| Claude Code | `claude plugin marketplace add archcore-ai/plugin`, then `claude plugin install archcore@archcore-plugins` (user scope by default); then merge the `autoUpdate: true` marketplace entry into `~/.claude/settings.json` | Commands verified 2026-08-12/15; already-registered marketplace tolerance [assumption] |
| GitHub Copilot | `copilot plugin install archcore-ai/plugin:plugins/archcore` | Command form verified; the `plugins/archcore` subpath matches the repository layout [assumption until first run] |
| Codex CLI | `codex plugin marketplace add archcore-ai/plugin`, then `codex plugin add archcore@archcore-plugins` | Verified 2026-08-12 |
| Cursor | none — print the UI instruction (Marketplace or `/add-plugin`) | UI-only per cursor.com/docs/plugins |

OpenCode ships no plugin. Roo Code, Cline, and Gemini CLI have none. Removal runs the host's own uninstall (`copilot plugin uninstall archcore`, `codex plugin remove archcore@archcore-plugins`; Claude Code equivalents [assumption]) and removes the `autoUpdate` entry this surface wrote.

## Normative Behavior

1. WHEN `archcore init` builds its host selection, the CLI MUST mark every plugin-capable host and state on the selection screen that selecting it also installs the Archcore plugin.
2. WHEN the selection marks Codex CLI or GitHub Copilot, the marking MUST name the install as machine-level, not project-level.
3. WHEN the user selects a plugin-capable host in an interactive init, the CLI MUST run that host's install action after wiring, with no second prompt.
4. IF a host was not selected, THEN init MUST NOT install its plugin.
5. WHEN a selected host is Cursor, the CLI MUST NOT run a host command.
6. WHEN init runs non-interactively with `--agent <id>`, the flag carries the consent, the same way it carries init's wiring consent today (`@cmd/init.go`).
7. WHILE init runs non-interactively without `--agent`, the CLI MUST NOT run any plugin install; it MUST print the per-host commands instead.
8. WHILE a CI variable is set, init MUST NOT run any plugin install; it MUST print the per-host commands instead.
9. WHEN a host's listing already shows the Archcore plugin, the install action MUST report it as already installed and change nothing.
10. WHEN a user types `archcore plugin install`, `update`, or `remove`, the CLI MUST treat the typed command as consent for every host it targets.
11. WHILE a direct `archcore plugin` invocation is non-interactive, the CLI MUST run it exactly as in an interactive session.
12. WHEN `archcore plugin install` runs for Claude Code, the CLI MUST use user scope unless `--scope project` was passed.
13. WHEN `--scope project` writes into `.claude/settings.json`, the CLI MUST print that the committed file delivers the declaration to every teammate.
14. WHEN a Claude Code install succeeds, the CLI MUST merge the `autoUpdate: true` marketplace entry into `~/.claude/settings.json`, preserving unknown fields.
15. WHEN this surface installed or updated a plugin, the duplicate-hook notice of `plugin-cli-compatibility.rule` requirement 4 MUST carry wording adjusted for a self-caused install.
16. `archcore plugin status` MUST report per host: the evidence found, the plugin's presence per the host's listing when the CLI is on `PATH` and per the on-disk registry otherwise, and the version when the host reports one.
17. `archcore plugin remove` MUST undo what this surface wrote and MUST run the host's own uninstall when the host CLI is present.
18. IF an attempted host action fails on a direct `archcore plugin` invocation, THEN the command MUST exit nonzero.
19. A host skipped for missing evidence MUST NOT fail a direct invocation.
20. WHEN `archcore plugin status` prints its report, the command MUST exit zero.
21. The plugin-delivery surface MUST NOT send a telemetry event in this release.
22. The plugin-delivery surface MUST NOT attempt privilege elevation.
23. WHEN a host's CLI is absent and that host's registry names the plugin, the CLI MUST report it as installed.
24. WHEN Cursor's registry names the plugin, the CLI MUST report it as installed rather than print the UI instruction.
25. WHEN a listing ran and does not name the plugin, `archcore plugin remove` MUST skip that host silently.
26. WHEN a mutating action succeeded, the CLI MUST print the self-caused notice exactly once per invocation.
27. IF the step bound elapses before a host is reached, THEN an install-carrying entry point MUST print that host's commands.
28. IF the step bound elapses before a host is reached, THEN an update-carrying entry point MUST print nothing for that host.

## Constraints & Invariants

- Constraint: the unattended update policy and the MCP trigger MUST NOT reach this surface.
- Constraint: `archcore plugin` MUST carry the `--project` flag like every project-aware command; only `--scope project` writes consume the resolved root, and the command MUST NOT join `rootlessCommands` (`@cmd/project_root_flag_test.go`).
- Constraint: WHEN `--agent` names an agent without a shipping plugin, the error MUST name only the four supported hosts — `claude-code`, `cursor`, `codex-cli`, `copilot` — not the full registry.
- Constraint: reads of host plugin state stay inside the plugin surface per `plugin-cli-compatibility.rule` requirement 3; no behavior outside it may change.
- Constraint: a delivery failure MUST NOT fail `archcore init`, and MUST NOT change its exit code. Requirement 18 binds the direct command, not the init step.
- Constraint: the delivery step MUST complete within its 120 s bound; requirement 27 governs the hosts it did not reach.
- Constraint: removal treats an unparseable settings file as nothing to remove. Requirements 6 and 7 of Failure Behavior follow from that: rewriting the file fresh would discard content the CLI never wrote.
- Constraint: the engine MUST use the three frozen identifiers exactly; requirement 11 of the compatibility rule binds them.
- Constraint: version pinning is not available — a marketplace install takes the latest plugin; the plugin's own minimum-CLI gate is the only version guard.
- Invariant: consent is carried by an explicit host selection — a checked host in the interactive screen, a host named with `--agent`, or a typed `archcore plugin` verb. No plugin installs on any other path.
- Invariant: a host detected without a picker is not a consent. `archcore init` on a project that already carries `.claude/` or `.codex/` installs nothing and prints one hint naming `archcore plugin install`.
- Invariant: install is idempotent — a rerun of `archcore init` over an installed plugin reports it and changes nothing, so repeated inits never nag and never re-install.
- Invariant: `archcore update`'s plugin step and `archcore plugin update` produce identical per-host actions — one planner, one executor, two entry points. The plan/execute split makes the invariant a plan-comparison test, not a convention.
- Invariant: hook and MCP wiring behavior of `archcore init` is unchanged by this surface.

## Failure Behavior

1. IF a host CLI is absent at install time, THEN the CLI MUST print that host's exact install command and continue.
2. IF a host command exits nonzero or times out, THEN the CLI MUST print the exact command it ran and continue.
3. IF the `autoUpdate` merge into `~/.claude/settings.json` fails, THEN the CLI MUST report it and MUST NOT undo the completed install.
4. IF the settings file is invalid JSON, THEN the CLI MUST back it up as `.bak` before writing (`backup-invalid-configs.adr`).
5. IF `archcore plugin remove` cannot reach a host CLI, THEN the CLI MUST print the exact uninstall command and continue.
6. IF the settings file is invalid JSON, THEN `archcore plugin remove` MUST report it and stop.
7. IF the settings file is invalid JSON, THEN `archcore plugin remove` MUST NOT write a backup.

## Conformance

An implementation is conformant when a plugin installs only behind an explicit host selection — a checked host, an `--agent` flag, or a typed `archcore plugin` verb; the selection screen names the plugin install and marks machine-level hosts; `--yes` without `--agent` and CI environments print commands and run nothing; a rerun over an installed plugin is a reported no-op; Claude Code defaults to user scope and gains the `autoUpdate: true` entry; every failure prints the exact command; the init step never changes init's exit code while the direct command reports its failures with a nonzero exit; and all entry points share one planner and one executor.

Given an interactive `archcore init` where the user checks Claude Code on a machine with `claude` on `PATH`, when wiring completes, then the CLI installs `archcore@archcore-plugins` at user scope, merges the `autoUpdate: true` entry, and prints the adjusted duplicate-hook notice — with no second prompt.
