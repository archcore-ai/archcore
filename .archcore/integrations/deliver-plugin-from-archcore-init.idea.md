---
title: "One Install Command: Deliver the Archcore Plugin from archcore init"
status: draft
tags:
  - "cli"
  - "integrations"
  - "release"
---

## Idea

Make the install script the single entry point for both Archcore entry points:

```
curl -fsSL https://archcore.ai/install.sh | bash
```

The script installs the CLI. `archcore init` then wires the detected hosts as it does today —
hooks, MCP config, instruction nudge — and, for the hosts where the Archcore plugin ships,
also delivers the plugin.

The plugin identifiers are already fixed and public, so the CLI does not have to discover them:

| Identifier | Value |
|------------|-------|
| Source repository | `archcore-ai/plugin` |
| Claude Code marketplace name | `archcore-plugins` |
| Claude Code plugin id | `archcore@archcore-plugins` |

What remains open is the per-host delivery mechanism. This document records what each host
offers, what was verified, and what is still unknown.

## Value

- **The CLI is already a prerequisite of the plugin.** The plugin's MCP entry runs
  `archcore mcp`, and its hooks delegate to `archcore hooks <host> <event>`. A plugin-first
  install therefore carries an unstated dependency. A CLI-first install states it.
- **It collapses the two-step install on GitHub Copilot and Cursor.** The plugin ships MCP
  config to Claude Code and Codex CLI only; on Cursor and GitHub Copilot the user installs the
  MCP server separately (global `architecture/supported-ai-hosts`). The CLI already writes MCP
  config for both hosts, so CLI-first plus plugin delivery leaves one step where there are now
  two.
- **One install command to publish, teach, and support**, instead of a per-host fork on every
  public surface.
- It keeps `architecture/one-product-two-entry-points`: the entry point is still chosen by the
  host the user runs, not ranked by preference. One install command is not a recommendation
  label.

## Host Feasibility (verified 2026-08-12)

Verified on macOS by reading each host CLI's own `--help` output, plus one write experiment for
Claude Code. Versions under test: Claude Code 2.1.228, codex-cli 0.147.0, GitHub Copilot CLI
1.0.76, opencode 1.15.13.

| Host | Plugin ships | Delivery mechanism | Scope | Verdict |
|------|--------------|--------------------|-------|---------|
| Claude Code | Yes | Config write: `extraKnownMarketplaces` and `enabledPlugins` in `.claude/settings.json`. CLI equivalent: `claude plugin marketplace add <source> --scope <scope>` and `claude plugin install archcore@archcore-plugins --scope <scope>` | `user`, `project`, `local` | **STRONG** — declarative, no subprocess needed |
| GitHub Copilot | Yes | `copilot plugin install <owner>/<repo>[:<path>]` — installs straight from a GitHub repository or subdirectory, with no marketplace registration | [SCOPE REQUIRED] | **STRONG** — one command, no marketplace step |
| Codex CLI | Yes | `codex plugin marketplace add archcore-ai/plugin` then `codex plugin add archcore@<marketplace> --json` | `~/.codex/config.toml`; no project scope observed | **PARTIAL** — machine-level side effect from a project-level command |
| Cursor | Yes | Unknown | Unknown | **UNKNOWN** — no Cursor CLI on the test machine |
| OpenCode | Adapter accepted, not shipped | `opencode plugin <module>` — "install plugin and update config" | [SCOPE REQUIRED] | **DEFERRED** — no adapter to deliver yet |

### Claude Code write experiment

`claude plugin marketplace add <dir> --scope project` followed by
`claude plugin install <plugin>@<marketplace> --scope project` wrote exactly two keys into the
project's `.claude/settings.json` and nothing else:

```json
{
  "extraKnownMarketplaces": {
    "<marketplace>": { "source": { "source": "directory", "path": "<path>" } }
  },
  "enabledPlugins": { "<plugin>@<marketplace>": true }
}
```

A GitHub-sourced marketplace uses `{"source": "github", "repo": "<owner>/<repo>"}` in the same
position, as recorded in Claude Code's own `known_marketplaces.json`.

`.claude/settings.json` is the file the CLI already owns for Claude Code hooks, so this is the
existing host-wiring pattern, not a new class of write.

## Possible Implementation

Three delivery tiers, in order of preference. Prefer the highest tier a host supports.

**Tier A — declarative config write.** WHEN the host reads a plugin declaration from a project
file, the CLI writes that declaration through the existing `internal/wiring/` path. Verified
available for Claude Code. This tier is idempotent, needs no network at `init` time, needs no
host binary on `PATH` (which matters for GUI hosts and CI), is reversible, and is testable with
the repository's table-driven tests. It also moves the code-fetch consent to the host: Claude
Code clones the marketplace on the next session and runs its own trust prompt.

**Tier B — call the host CLI.** WHEN no declarative route exists, the CLI runs the host's own
install command. Guard it with `exec.LookPath`, bound it with a timeout, require explicit
confirmation, and IF the call fails, THEN print the exact command instead of failing `init`.
Applies to GitHub Copilot and Codex CLI.

**Tier C — print the command.** WHEN neither route exists, the CLI prints the command and does
not run it. Applies to Cursor if it turns out to be UI-only, and to any host without a plugin.

**Command surface.** Add `archcore plugin install|remove|status [--agent <id>]` and have `init`
call it after confirmation. This mirrors `archcore hooks install` and
`archcore instructions install` and follows the constructor-command pattern in
`.archcore/cli-ui/building-the-cli.guide.md`. Note that it grows the public command surface.

## Risks

- **Scope asymmetry.** Hooks and MCP config are project-local and revert with a `git checkout`.
  Codex CLI and GitHub Copilot plugin installs change the user's machine. A project-level
  command with a machine-level side effect MUST NOT run silently, and `--yes` MUST NOT imply
  consent to it.
- **Committed `enabledPlugins` installs a plugin for the whole team.** Project scope on Claude
  Code writes into a file that is committed, so every teammate who opens the repository gets the
  declaration. Claude Code prompts for trust, but the CLI has to say so at write time — the case
  `.archcore/integrations/report-effective-hook-state.rule.md` already governs. Offer a scope
  flag rather than hard-coding project scope.
- **New coupling to another repository's public names.** The CLI would carry
  `archcore-ai/plugin`, `archcore-plugins`, and `archcore@archcore-plugins` in its binary. IF the
  plugin renames any of them, THEN every already-released CLI writes a dead declaration.
  `.archcore/integrations/plugin-cli-compatibility.rule.md` needs a clause freezing these three
  identifiers, in the same spirit as its existing clause 9 on hook leaves.
- **Do not turn delivery into detection-gated behavior.** Clause 3 of the compatibility rule says
  the CLI MUST NOT change what it does based on whether a plugin is installed. Delivery is a new
  axis and must not become a reason to branch on `detectInstalledPlugin`.
- **The CLI would create the duplicate-hook state it warns about.** Clause 4 requires reporting
  that hooks may run twice when a plugin is present. On the path where the CLI just installed the
  plugin, that notice needs different wording, and the dedup stamp (clause 6) has to cover it.
- **Version pinning is not available.** A marketplace install takes the latest plugin. The
  plugin's own minimum-CLI gate is the only version guard.
- **Trust escalation.** `curl … | bash` that then installs plugins into a user's agents is a
  larger ask than installing a binary. The consent step is the whole mitigation.
- **Narrative change.** Install CTAs, `product/surface-descriptors`, and the landing install tabs
  all describe two install paths today. Changing this is one coordinated pass across surfaces,
  not a CLI-only change.

## Open Questions

- **Cursor.** Does Cursor 2.5+ expose a non-interactive plugin install, a declarative project
  file, or only a UI or deeplink flow? This determines whether Cursor lands in Tier A, B, or C.
- **GitHub Copilot subdirectory.** The exact `owner/repo:path` argument for the Archcore plugin
  is not yet confirmed. [assumption] The plugin lives under `plugins/archcore`.
- **Copilot and OpenCode scope.** Neither host's install scope was determined.
- Does the plugin function at all without the `archcore` binary, or does it fail closed? The
  answer decides whether CLI-first is a convenience or a correctness fix.
- Default host set: deliver to every detected host that has a plugin, or ask per host?
- What `archcore doctor` should report — a declaration written but not picked up by the host.
- Whether `archcore plugin remove` must also undo the marketplace registration, or only the
  plugin.

## Related

- `.archcore/integrations/plugin-cli-compatibility.rule.md` — the contract this idea extends
  with an identifier-freeze clause.
- `.archcore/integrations/supported-ai-agents.doc.md` — per-host config paths and detection.
- `.archcore/integrations/report-effective-hook-state.rule.md` — the reporting obligation that
  covers a written declaration the host may not act on.
- `.archcore/release/install-script-usage.guide.md` — the install script this idea makes the
  single entry point.
- `.archcore/cli-ui/building-the-cli.guide.md` — the command pattern a new `plugin` command
  follows.
- Global `architecture/one-product-two-entry-points` and `architecture/supported-ai-hosts`
  (read-only) — the entry-point framing and the plugin host coverage set; referenced in prose,
  not linked.
