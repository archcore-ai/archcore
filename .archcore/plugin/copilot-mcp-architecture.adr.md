---
title: "Copilot MCP Architecture — Project-Level Install Only, No Plugin-Shipped MCP"
status: accepted
tags:
  - "architecture"
  - "copilot"
  - "multi-host"
  - "plugin"
---

## Context

While preparing the GitHub Copilot CLI adapter for release in July 2026, plugin-shipped MCP turned out to be unusable on that host — for the same class of reason that produced `cursor-mcp-architecture.adr`, but with a sharper failure mode.

Two facts produced the decision:

1. **Copilot launches a plugin's MCP children in the plugin install directory and tells them nothing about the project.** [github/copilot-cli#4234](https://github.com/github/copilot-cli/issues/4234) (opened 2026-07-23, open, no maintainer response): "an MCP server loaded from an installed plugin is launched with its working directory set to the plugin installation root, and the child process receives no project/workspace path." The child does not even receive `COPILOT_PROJECT_DIR`, which plugin *hooks* do get — so there is no recovery path from inside the process.

   For Archcore this is worse than a wrong answer. `archcore mcp` resolves the project root from cwd; launched from `~/.copilot/installed-plugins/…` it finds no `.archcore/` there and reports an uninitialized project, at which point `init_project` creates `.archcore/` **inside the plugin cache**. Every tool returns success. The documents land where no git repository will ever see them, and the user's next session — served by a different cache entry or a reinstalled plugin — finds an empty knowledge base.

2. **GitHub's own documentation does not agree on whether removing the manifest key is even sufficient.** The [plugin reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-plugin-reference) gives a default path only to `agents/` and `skills/`; `mcpServers` has none, which makes the manifest key the only source. The [concepts page](https://docs.github.com/en/copilot/concepts/agents/about-plugins) instead describes "a `.mcp.json` file in the plugin root" as where MCP configurations live, phrased as a discovery rule. Documentation cannot settle this; a live probe can. The decision below is deliberately robust to either answer.

The Cursor remedy does not transfer. There we moved the offending file out of the plugin root (`docs/cursor.mcp.example.json`). Here we cannot: `.mcp.json` must stay at the plugin root because Claude Code discovers it there with no manifest key at all.

## Decision

The plugin does **not** ship MCP for Copilot CLI. `.plugin/plugin.json` omits `mcpServers`. Copilot users get the MCP server from their own project's `.mcp.json` — written by `archcore init --agent copilot` (CLI >= v0.6.4) or `/archcore:init` — or from `~/.copilot/mcp-config.json` at the user level.

This is the correct surface regardless of the defect: Copilot CLI discovers `.mcp.json` by walking from the working directory up to the git root, so a repo-root file covers monorepo layouts, and a project-registered server is launched from the project, which makes cwd right by construction rather than by luck.

**Layer 1 — the manifest omits the key**, pinned by `test/structure/copilot-plugin.bats` with the issue number in the assertion message. Without a test the key reads like an oversight and comes back on the next cleanup.

**Layer 2 — host wiring is the documented path.** `archcore init --agent copilot` writes `.mcp.json`, `.github/hooks/archcore.json` and the `AGENTS.md` managed block; `/archcore:init` writes the same set (`host-wiring-parity.adr`). On Copilot this stops being an optimization and becomes the only route to MCP tools.

**Layer 3 — the runtime guard, once it covers this host.** `resolveProjectRoot` in the CLI already refuses to treat a directory inside a plugin install cache as a project root, and `bin/session-start` applies the same rule to hooks. The CLI's fragment list currently names `.cursor/plugins/`, `.claude/plugins/`, `.codex/plugins/` and `plugins/cache/` — **not** `~/.copilot/installed-plugins/`. Until that fragment is added, Layer 3 does not protect Copilot, and Layers 1-2 carry the decision alone.

## Alternatives

**A. Ship `mcpServers` and pass the project path in `args`.**
The canonical way to make a plugin MCP project-aware. Rejected: #4234 states the child receives no project path, and Copilot documents no interpolation token usable inside plugin-MCP `args` — there is nothing to put after `--project`.

**B. Move `.mcp.json` out of the plugin root, as we did for Cursor.**
Rejected: impossible. Claude Code discovers plugin-root `.mcp.json` with no manifest key; moving it would break the primary host to fix the newest one.

**C. Ship a wrapper that re-resolves the working directory before exec'ing `archcore mcp`.**
Rejected twice over. It re-creates `bundled-cli-launcher.adr`, removed in v0.4.0 for eight categories of bugs; and the input such a wrapper needs — the project path — is precisely what #4234 says the host does not pass.

**D. Wait for upstream to fix #4234.**
Rejected as the whole plan. The issue is open with no maintainer response, and the adapter ships now. It is retained as the revisit trigger below.

## Consequences

**Positive**

- Copilot cannot write a user's documents into its own plugin cache through our MCP.
- The install story is the one Cursor users already have: one precedent, one explanation, one place in the docs.
- Claude Code and Codex are untouched — `.mcp.json` and `.codex.mcp.json` stay exactly where they are.

**Negative**

- **The plugin stops being self-contained on Copilot.** Installing it yields skills, commands, agents and hooks, but no MCP tools until host wiring runs. The init skill must therefore always emit the Host wiring line for copilot, which pins that path to CLI >= v0.6.4.
- Layer 3 does not cover Copilot until `.copilot/installed-plugins/` joins the CLI's plugin-cache fragments. Until it does, a user who hand-registers the server from inside the plugin directory gets the silent-write behavior back.
- If upstream closes #4234 — by passing `COPILOT_PROJECT_DIR` to MCP children or launching them from the project directory — the manifest key can be revisited, but the runtime guard stays as belt-and-suspenders.

**Neutral**

- Whether omitting the key is *sufficient* is unverified: if the concepts page is right and a plugin-root `.mcp.json` is auto-discovered, the key was never the only source. Probe P4 (`copilot mcp list` against an installed plugin) settles it, and Layer 3 is what makes the answer non-critical either way.
