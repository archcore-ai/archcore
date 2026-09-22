---
title: "Plugin Update Step in archcore update"
status: accepted
tags:
  - "cli"
  - "component:cli"
  - "integrations"
  - "update"
---

## Purpose & Scope

This spec defines the plugin-update step inside manual `archcore update`. After the binary phase, the step refreshes the Archcore plugin on each host where it is installed, and in every scope the host reports. The step runs the update action of the shared plugin engine — the surface `plugin-delivery.spec` defines; `archcore plugin update` is the same action behind its own command. Dependents: `@cli/cmd/update.go`, the host registry in `internal/agents/`, the host CLIs, and the `archcore-ai/archcore` repository.

Out of scope: the unattended update policy and the MCP background trigger — both MUST NOT reach this step; first-time plugin install, which belongs to the delivery surface.

## Surface

- Caller: manual `archcore update` only.
- Frozen identifiers: repository `archcore-ai/archcore`, marketplace `archcore-plugins`, plugin id `archcore@archcore-plugins`.
- Evidence order: the host's own answer first. With the host CLI on `PATH`, its read-only listing — `claude plugin list --json`, `copilot plugin list`, `codex plugin list --json`; with the CLI absent, the host's on-disk registry — `~/.claude/plugins`, `~/.cursor/plugins`, `~/.copilot/installed-plugins`, `~/.codex/plugins`. Each listing command enumerates installed plugins only — verified 2026-08-17 for the two JSON listings and 2026-09-16 for `copilot plugin list` — and a flag that adds uninstalled marketplace entries, `codex plugin list --available`, stays out of these command lines. A listing shows the Archcore plugin when it carries an entry that names the plugin itself, not the marketplace it ships in, and that the host does not report as uninstalled.
- Installation: one listing entry that shows the plugin. It carries the name the host listed, and on Claude Code a `scope` and, for `project` and `local` scope, a `projectPath`. `claude plugin list --json` lists one entry per scope and project — verified 2026-09-16.
- Timeouts: 30 s per host command [assumption]; the whole step bounded at 120 s [assumption]. One Claude Code update takes about 2 s — measured 2026-09-16.
- Ceiling: at most 32 addressed installations per host, taken in scope-then-project-path order after the skips of requirements 21, 22 and 26 [assumption]; the rest are skipped silently. 32 at about 2 s each leaves the 120 s step room for the other hosts.
- Output: one progress line before each command run; one line per failed command; one line per addressed host that was not run — the exact command to run, or the Cursor UI instruction; and, when the step updated a plugin, one line reporting the session overlap the update caused. A host without the plugin produces no line.
- Exit code: `archcore update` exits with the binary phase's result; this step never changes it.

Per-host update commands, probed live 2026-09-16 on the versions shown, plus official docs:

| Host | Update command | Note |
|---|---|---|
| Claude Code (claude 2.1.273) | `claude plugin marketplace update archcore-plugins`, then per installation `claude plugin update archcore@archcore-plugins --scope <scope>` | Without `--scope` the host updates `user` scope only. For `project` and `local` scope the host resolves the working directory, matches it against `projectPath` exactly, and otherwise updates the first entry of that scope — verified 2026-09-16, anthropics/claude-code#90519. Off a terminal `-y` goes to `plugin update`, `install` and `uninstall` only; `marketplace update` and `marketplace add` reject it. The printed tier omits `--scope`. |
| GitHub Copilot (copilot 1.0.83) | per installation `copilot plugin update <listed name>`; printed tier `copilot plugin update archcore` | A direct install is listed as `archcore` and refuses `archcore@archcore-plugins`. A failed update exits 0 and writes only to stderr — verified 2026-09-16. |
| Codex CLI (codex 0.154.0) | `codex plugin marketplace upgrade archcore-plugins` | Codex has no per-plugin update and no scope; the marketplace upgrade reinstalls the plugin cache (`codex-rs/core-plugins/src/manager.rs`). |
| Cursor | none — no CLI mechanism (UI-only, cursor.com/docs/plugins) | Print a one-line instruction to update the plugin in the Cursor UI. |

OpenCode ships no plugin. Roo Code, Cline, and Gemini CLI have none. The step does not address them.

## Normative Behavior

1. WHEN the binary phase replaces the binary, the CLI MUST run the plugin-update step next.
2. WHEN the binary phase finds the binary already current, the CLI MUST run the plugin-update step next.
3. IF the binary phase fails, THEN the CLI MUST skip the plugin-update step.
4. WHEN `archcore update` runs with `--check`, the CLI MUST NOT run the plugin-update step.
5. WHEN a host's CLI is on `PATH`, the CLI MUST query that host's plugin listing under the command timeout before any mutating command.
6. WHEN the listing shows the Archcore plugin, the CLI MUST run that host's update command from the Surface table, bounded by the 30 s timeout.
7. IF the listing does not show the Archcore plugin, THEN the CLI MUST skip that host silently.
8. IF a host's CLI is absent and that host's on-disk registry lists the Archcore plugin, THEN the CLI MUST print the exact update command from the Surface table.
9. WHEN Cursor's plugin registry lists the Archcore plugin, the CLI MUST print the one-line Cursor UI instruction; the CLI MUST NOT run a Cursor command.
10. IF no tier of requirements 5–9 applies to a host, THEN the CLI MUST skip that host silently.
11. WHEN the CLI runs a host command, the CLI MUST print one progress line naming the host before the command starts.
12. IF a host update command exits nonzero or times out, THEN the CLI MUST print the exact command it ran.
13. The plugin-update step MUST NOT change the exit code of `archcore update`.
14. The plugin-update step MUST NOT send a telemetry event.
15. WHEN the step updated at least one installation, the CLI MUST print the self-caused overlap notice exactly once.
16. IF no host command succeeded, THEN the CLI MUST NOT print the self-caused overlap notice.
17. IF a host reports a listing entry as not installed, THEN the CLI MUST read that entry as not showing the plugin.
18. IF a listing name identifies the marketplace rather than the plugin, THEN the CLI MUST read it as not showing the plugin.
19. WHEN a listing shows several installations, the CLI MUST run the per-installation update command once for each installation.
20. WHEN the CLI updates a `project` or `local` installation, the CLI MUST run the command in that installation's `projectPath`.
21. IF an installation's `projectPath` is not an existing, absolute, fully resolved directory, THEN the CLI MUST skip that installation silently.
22. IF an installation reports `managed` scope, THEN the CLI MUST skip that installation silently.
23. IF one per-installation command fails, THEN the CLI MUST report it and continue with the next installation.
24. WHEN a Copilot update command exits zero with empty stdout, the CLI MUST treat the command as failed.
25. IF a listing shows installations and the CLI can address none of them, THEN the CLI MUST skip that host silently.
26. WHEN the CLI addresses a Copilot installation, the CLI MUST pass only `archcore` or `archcore@archcore-plugins` as its name.

## Constraints & Invariants

- Constraint: the unattended update policy and the MCP trigger MUST NOT reach this step. Manual `archcore update` is the only caller.
- Constraint: reads of a host's plugin listing and registry stay inside the plugin surface per requirement 3 of `plugin-cli-compatibility.rule`; nothing outside that surface may change on what they find.
- Constraint: IF a listing query or a registry read fails, THEN the CLI MUST treat the host as not listed and MUST NOT change any other behavior (clause 5 of the same rule).
- Constraint: WHEN this step itself updated the plugin, the duplicate-hook notice (clause 4) MUST carry wording adjusted for a self-caused update.
- Constraint: the dedup stamp (clause 6) MUST cover the overlap window that follows a self-caused plugin update.
- Constraint: the CLI MUST NOT change the three frozen identifiers except in step with the plugin repository.
- Constraint: the step MUST NOT attempt privilege elevation.
- Constraint: with `autoUpdate: true` delivered at install, Claude Code refreshes the plugin on its own after a session starts; the docs do not state that it covers `project` and `local` scope. This step stays as the deterministic path. The redundancy on Claude Code is deliberate — do not remove either side.
- Invariant: `archcore update` emits at most one telemetry event per invocation. This step preserves the invariant by emitting none.
- Invariant: a user who never installed the plugin sees no plugin output and pays no mutating host command.
- Invariant: an update command runs only after the host's own listing confirmed the plugin — silence for non-installers is structural, not parsed from error text.
- Invariant: a per-installation command never runs outside the project its installation names, because the host would otherwise update a different project.

## Failure Behavior

1. IF a listing query fails, exits nonzero, or does not parse, THEN the CLI MUST skip that host silently and continue.
2. IF a host update command exits nonzero, THEN the CLI MUST print the exact command it ran and continue.
3. IF a host command exceeds the 30 s timeout, THEN the CLI MUST kill the subprocess.
4. WHEN the CLI kills a timed-out subprocess, the CLI MUST print the exact command it ran and continue.
5. IF the 120 s step bound elapses, THEN the CLI MUST skip the remaining hosts and print nothing for them.
6. IF any part of this step fails, THEN the CLI MUST keep the exit code of `archcore update` unchanged.

## Conformance

An implementation is conformant when the step runs only from manual `archcore update`, after a binary phase that did not fail and never under `--check`; queries each present host's listing before any mutating command and mutates only on a confirmed plugin; updates every addressable installation the listing reports, each from its own project directory; prints the exact command for a registry-listed host without its CLI and on every failed command; stays silent for every host without the plugin; bounds each command at 30 s and the step at 120 s; sends no telemetry event; and leaves the exit code of `archcore update` untouched.

Given a machine with `claude` on `PATH` and no Archcore plugin installed, when `archcore update` finishes its binary phase, then the step runs one listing query, the listing shows no plugin, the step prints nothing for Claude Code, no mutating command runs, and `archcore update` exits with the binary phase's code.

Given a machine where a host registered the marketplace and installed no plugin from it, when `archcore update` runs the step, then the listing shows no plugin, no mutating command runs, and the step prints nothing for that host.

Given Claude Code lists the plugin in `user` scope and in `local` scope for project `P`, when the step runs, then it runs the marketplace update, one update with `--scope user`, and one with `--scope local` in `P`.
