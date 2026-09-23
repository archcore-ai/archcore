---
title: "The init Selection Screen Opens on Every Interactive Run, With Detected Hosts Pre-Checked"
status: draft
tags:
  - "cli"
  - "component:cli"
  - "integrations"
---

## Context

`resolveAgents` in `@cli/cmd/init.go` opens the agent multi-select only when `agents.Detect` finds no host, so a project that already carries `.claude/` never reaches the selection screen (observed 2026-09-23 on v0.10.2 in this repository: init wired Claude Code and went straight to the usage-hint prompt). The accepted plugin-delivery spec makes a checked host the only consent the init step accepts, so on such a project init installs no plugin and prints one hint, and a second host such as Cursor cannot be added interactively at all. The interactive path therefore carried two consent sources, `outcomeDetected` and `outcomePicked`, and the user-visible one was unreachable exactly where a host was already set up.

## Decision

In interactive `archcore init` without `--agent`, the CLI opens the agent multi-select on every run, pre-checks each host `agents.Detect` found, and takes the confirmed selection as the sole source of both wiring and plugin consent, while a non-interactive run without `--agent` keeps the detection-only path of v0.10.2.

## Alternatives Considered

1. Keep the detection-skips-picker routing of v0.10.2 — rejected because a project carrying a host marker can never reach the screen, so the plugin cannot be delivered from init and a second host cannot be added without `--agent`.
2. A `--pick` flag that forces the screen open — rejected because the default path stays unreachable for the same projects, and a flag the banner never mentions goes unused [assumption].
3. Treat a detected host as a checked box and install its plugin — ruled out because the plugin-delivery spec's consent Invariant forbids an install on a path with no checked host, and a marketplace install nobody consented to is the harm that Invariant prevents.
4. Change `resolveAgents` itself, so `hooks install`, `mcp install`, and `instructions install` open the screen too — deferred because those commands document a detected-hosts contract of their own; init gets its own selector (`selectAgentsForInit`) and the three commands stay as in v0.10.2.

## Consequences

- Positive: every interactive init reaches the selection screen, so a project with `.claude/` can add Cursor or Codex CLI and consent to the plugin in one run; the `selectAgentsForInit` tests in `@cli/cmd/init_test.go` pin it.
- Positive: `outcomeDetected` narrows to the non-interactive path of init, leaving the interactive branch one consent source instead of two.
- Negative: a rerun on an already-wired project shows one extra screen; confirming the pre-checked default costs one Enter keypress [expected].
- Negative: a user who unchecks a detected host gets no wiring for it in that run, where v0.10.2 wired it silently [expected].
- Negative: the Surface line and two Invariants of the plugin-delivery spec change, and `supported-ai-agents.doc`, `building-the-cli.doc`, `@cli/README.md`, and `@README.md` carry sentences on detection-driven wiring that need the same edit.

## Superseded when

- `archcore init` gains a host-management entry outside its own run, such as an `archcore host add <id>` command, that owns host selection.
- The prompt count of an interactive `archcore init` rerun exceeds three (today: reinitialize, selection, usage hint), at which point a consolidated prompt is the better fix.
