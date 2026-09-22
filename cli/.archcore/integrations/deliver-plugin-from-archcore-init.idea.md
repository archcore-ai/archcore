---
title: "One Install Command: Deliver the Archcore Plugin from archcore init"
status: accepted
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

**Accepted (2026-08-15).** The idea ships this release as the plugin-delivery surface —
`.archcore/integrations/plugin-delivery.spec.md` defines `archcore plugin install|update|remove|status`
and the selection-driven init step. The update half of the tier model ships alongside it as the
plugin-update step inside manual `archcore update` — `.archcore/update/updating-the-plugin.spec.md`.
The identifier-freeze clause this document requested landed in
`.archcore/integrations/plugin-cli-compatibility.rule.md` as clause 11, and clause 3 gained the
plugin-surface carve-out the delivery step needs. The sections below record the feasibility
evidence and the risks that shaped the spec.

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

## Host Feasibility (verified 2026-08-12, extended 2026-08-15)

Verified on macOS by reading each host CLI's own `--help` output, plus one write experiment for
Claude Code. Versions under test: Claude Code 2.1.228/2.1.232, codex-cli 0.147.0, GitHub Copilot
CLI 1.0.76, opencode 1.15.13.

| Host | Plugin ships | Delivery mechanism | Scope | Verdict |
|------|--------------|--------------------|-------|---------|
| Claude Code | Yes | `claude plugin marketplace add <source>` and `claude plugin install archcore@archcore-plugins --scope <scope>`; the `autoUpdate: true` marketplace entry in settings makes the host refresh the plugin in the background after session start (documented) | `user`, `project`, `local` | **STRONG** |
| GitHub Copilot | Yes | `copilot plugin install <owner>/<repo>[:<path>]` — installs straight from a GitHub repository or subdirectory, with no marketplace registration | user-level store, no scope flags | **STRONG** — one command, no marketplace step |
| Codex CLI | Yes | `codex plugin marketplace add archcore-ai/plugin` then `codex plugin add archcore@<marketplace>` | `~/.codex/config.toml`; machine-level | **PARTIAL** — machine-level side effect from a project-level command |
| Cursor | Yes | None non-interactive: plugins are UI-only per cursor.com/docs/plugins (verified 2026-08-15) | Unknown | **Tier C** — print the UI instruction |
| OpenCode | Adapter accepted, not shipped | `opencode plugin <module>` — "install plugin and update config" | `-g/--global` or project config | **DEFERRED** — no adapter to deliver yet |

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
existing host-wiring pattern, not a new class of write. One caveat the spec carries: since Claude
Code 2.1.195, an `enabledPlugins` declaration alone does not install the plugin for a reader of
the file — each user installs it themselves behind the trust prompt — so the delivery path runs
the host's own install command and uses the declaration for the `autoUpdate` entry.

## Possible Implementation

Realized in `.archcore/integrations/plugin-delivery.spec.md`: one engine [planned,
`internal/plugin`], three entry points (`archcore plugin`, the init step, the update step), and
per-host tiers — run the host CLI's command when present, print the exact command otherwise,
print the UI instruction for Cursor. Consent rides the host selection: a host checked in init's
multi-select installs with no second prompt, and the selection screen carries the disclosure.
`--yes` without `--agent` and CI sessions run nothing and print the commands; a rerun over an
installed plugin is a reported no-op.

## Risks

- **Scope asymmetry.** Hooks and MCP config are project-local and revert with a `git checkout`.
  Codex CLI and GitHub Copilot plugin installs change the user's machine. The spec answers with
  the machine-level marking on the selection screen; `--yes` without an explicit host selection
  never installs.
- **Committed `enabledPlugins` installs a plugin for the whole team.** Project scope on Claude
  Code writes into a file that is committed, so every teammate who opens the repository gets the
  declaration. Claude Code prompts for trust, but the CLI has to say so at write time — the case
  `.archcore/integrations/report-effective-hook-state.rule.md` already governs. The spec answers
  with user scope by default and `--scope project` as the disclosed opt-in.
- **New coupling to another repository's public names.** The CLI carries `archcore-ai/plugin`,
  `archcore-plugins`, and `archcore@archcore-plugins` in its binary. Clause 11 of
  `plugin-cli-compatibility.rule.md` freezes them.
- **Detection-gated behavior.** Clause 3 of the compatibility rule bounds plugin-state reads to
  the plugin surface; delivery reads the host's own answer and changes nothing outside that
  surface.
- **The CLI would create the duplicate-hook state it warns about.** Clause 4 requires reporting
  that hooks may run twice when a plugin is present. On the path where the CLI just installed the
  plugin, that notice needs different wording, and the dedup stamp (clause 6) has to cover it.
- **Version pinning is not available.** A marketplace install takes the latest plugin. The
  plugin's own minimum-CLI gate is the only version guard.
- **Trust escalation.** `curl … | bash` that then installs plugins into a user's agents is a
  larger ask than installing a binary. The explicit host selection carrying a stated plugin
  disclosure is the mitigation.
- **Narrative change.** Install CTAs, `product/surface-descriptors`, and the landing install tabs
  all describe two install paths today. The rollout plan carries this as a coordinated,
  release-adjacent pass.

## Open Questions

Resolved into the delivery spec: Cursor is Tier C (UI-only, verified); the Copilot argument is
`archcore-ai/plugin:plugins/archcore` ([assumption] until the first run — the subpath matches the
repository layout); Copilot's store is user-level and Codex's is machine-level, both named
machine-level on the selection screen; consent rides init's host selection — a checked host
installs with no second prompt, and a rerun over an installed plugin is a reported no-op;
`archcore plugin remove` undoes what the surface wrote and runs the host's own uninstall;
`archcore plugin status` covers the written-but-not-picked-up report, so `doctor` gains nothing
this release.

Still open:

- Does the plugin function at all without the `archcore` binary, or does it fail closed? The
  answer decides whether CLI-first is a convenience or a correctness fix.

## Related

- `.archcore/integrations/plugin-delivery.spec.md` — the spec this idea became.
- `.archcore/integrations/plugin-cli-compatibility.rule.md` — the contract, now carrying the
  identifier freeze (clause 11) and the plugin-surface carve-out (clause 3).
- `.archcore/update/updating-the-plugin.spec.md` — the plugin-update step that ships the update
  half of the tier model this release.
- `.archcore/integrations/supported-ai-agents.doc.md` — per-host config paths and detection.
- `.archcore/integrations/report-effective-hook-state.rule.md` — the reporting obligation that
  covers a written declaration the host may not act on.
- `.archcore/release/install-script-usage.guide.md` — the install script this idea makes the
  single entry point.
- `.archcore/cli-ui/building-the-cli.doc.md` — the command pattern the `plugin` command follows.
- Global `architecture/one-product-two-entry-points` and `architecture/supported-ai-hosts`
  (read-only) — the entry-point framing and the plugin host coverage set; referenced in prose,
  not linked.