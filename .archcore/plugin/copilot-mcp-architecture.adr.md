---
title: "Copilot MCP Architecture — Project-Level Only, Enforced by Filename and an Empty Manifest Key"
status: accepted
tags:
  - "architecture"
  - "copilot"
  - "multi-host"
  - "plugin"
---

**Update (2026-08-03, second pass).** The first pass settled that omitting the manifest key is not sufficient. Measurement then showed the remedy it proposed — renaming the file and pointing at it from `.claude-plugin/plugin.json` — is also not sufficient, for a reason nobody had: **Copilot reads `.claude-plugin/plugin.json`** when its own `.plugin/plugin.json` does not declare `mcpServers`. The working configuration is three parts, all load-bearing and all measured. Alternative 2 is no longer impossible; it is what shipped.

## Context

While preparing the GitHub Copilot CLI adapter for release in July 2026, plugin-shipped MCP turned out to be unusable on that host, for the same class of reason that produced `cursor-mcp-architecture.adr` but with a sharper failure mode. Copilot launches a plugin's MCP children in the plugin install directory and tells them nothing about the project, so `archcore mcp` resolves the project root from the working directory, finds no `.archcore/` in the cache, reports an uninitialized project, and `init_project` then creates `.archcore/` inside the plugin cache while every tool returns success. This is not hypothetical: `~/.copilot/installed-plugins/_direct/archcore-ai--plugin--plugins-archcore/.archcore/test-solution.adr.md`, dated 2026-07-30, was written by a live session.

## Findings

1. **The upstream defect.** [github/copilot-cli#4234](https://github.com/github/copilot-cli/issues/4234), "Plugin MCP servers cannot resolve the active project directory", was opened 2026-07-23 and verified still open with no comments on 2026-08-03. An MCP server loaded from an installed plugin launches with its working directory set to the plugin installation root and receives no project or workspace path — not even `COPILOT_PROJECT_DIR`, which plugin *hooks* do receive — so no recovery is possible from inside the process. The issue does not say a plugin cannot ship an MCP server; plugin MCP servers load fine. It says they cannot find the project.
2. **The real hazard is a name collision rather than the mere existence of a plugin server.** `archcore init --agent copilot` registers the project server under the key `archcore`, and a plugin server under the same key does not sit beside it: Copilot merges MCP sources user → workspace → plugins with last-wins, so the plugin entry *replaces* the project one. Measured on Copilot CLI 1.0.76 on 2026-08-03, with a fresh `COPILOT_HOME` and a workspace `.mcp.json` whose archcore entry used a sentinel command:

| plugin layout | which `archcore` survived |
| --- | --- |
| no plugin installed (control) | workspace |
| `.mcp.json` at plugin root | **plugin** (`cwd=${PLUGIN_ROOT}`) |
| renamed, no manifest key anywhere | workspace |
| renamed + `mcpServers` in `.claude-plugin/plugin.json` | **plugin** |
| `.mcp.json` kept + empty `mcpServers` in `.plugin/plugin.json` | **plugin** |
| renamed + both manifest keys (shipped) | workspace |

   A user who followed the documented two-step install therefore ended up with a server that ignored their project and, from CLI v0.6.7 on, refused to start at all — while the session-start wiring advisory, which greps `"archcore"` out of `.mcp.json`, reported the project as wired.

3. **Two host behaviors produce that table, and each needs its own countermeasure.** Copilot auto-discovers `.mcp.json` in the plugin root — and, per directory, `.github/mcp.json` where the first is absent — regardless of any manifest; no manifest key switches this off, and an empty `mcpServers` declared while `.mcp.json` is still on disk leaves the plugin server loaded, so the filename is the only lever. Separately, Copilot falls back to reading `.claude-plugin/plugin.json` when `.plugin/plugin.json` does not declare `mcpServers`, so the key that re-arms Claude Code after the rename re-arms Copilot too, unless the Copilot manifest declares an `mcpServers` of its own and shadows the fallback.

## Decision

Ship **no MCP server to Copilot**, enforced in three parts, each load-bearing and each pinned by its own test with a negative control in `@test/structure/plugin-mcp-isolation.bats`.

1. **The MCP config carries no auto-discovered name.** It is `plugins/archcore/.claude.mcp.json`, not `.mcp.json`, and the plugin root carries no `mcp.json` and no `.github/mcp.json`.
2. **`.claude-plugin/plugin.json` declares `"mcpServers": "./.claude.mcp.json"`.** With no conventional filename left, this key is the entirety of Claude Code's MCP wiring. Claude Code loads a plugin-root `.mcp.json` and then merges the manifest declaration in addition to it, so the key is a full replacement for the convention rather than a supplement. Verified at runtime rather than from documentation: `claude --plugin-dir … mcp list` reports `plugin:archcore:archcore … Connected`. `claude plugin details` under-reports the MCP count for manifest-declared paths and is not a valid oracle.
3. **`.plugin/plugin.json` declares `"mcpServers": {}`.** An empty object is still a declaration: it stops the fallback in finding 3 while contributing nothing. Deleting this key is what an unwitting "the manifest declares an empty field, remove it" cleanup does, and it restores the collision silently — hence the named test.

Copilot users get the MCP server from their own project's `.mcp.json`, written by `archcore init --agent copilot` on CLI v0.6.4 or later, or by `/archcore:init`, or from `~/.copilot/mcp-config.json` at user level. That is the correct surface regardless of the defect: a project-registered server launches from the project, which makes the working directory right by construction rather than by luck.

**Layer 2 — host wiring is the documented path.** `archcore init --agent copilot` writes `.mcp.json`, `.github/hooks/archcore.json`, and the `AGENTS.md` managed block, and `/archcore:init` writes the same set per `host-wiring-parity.adr`. On Copilot this stops being an optimization and becomes the only route to MCP tools.

**Layer 3 — the runtime guard.** `resolveProjectRoot` in the CLI refuses to treat a directory inside a plugin install cache as a project root, and `bin/session-start` applies the same rule to hooks. Since CLI v0.6.7 and plugin 0.6.2, both fragment lists name `.cursor/plugins/`, `.claude/plugins/`, `.codex/plugins/`, `.copilot/installed-plugins/`, and `plugins/cache/`. The three-part decision removes the route that got a server started in the cache in the first place; Layer 3 stays because it is not the only route — a custom `COPILOT_HOME`, a hand-written user-level config, or a future host that spawns from an install directory all land there. Known limitation: a custom `COPILOT_HOME` moves the cache outside every fragment, so the CLI-side spawn is unguarded there, while plugin hooks stay covered by the layer-2 manifest walk. The fragment set MUST NOT be widened to a bare `installed-plugins/`, which would produce false positives.

## Alternatives Considered

1. **Ship `mcpServers` and pass the project path in `args`**, the canonical way to make a plugin MCP project-aware — rejected because #4234 states the child receives no project path, and Copilot documents no interpolation token usable inside plugin-MCP `args`, so there is nothing to put after `--project`.
2. **Move the MCP config off the auto-discovered filename, as Cursor forced** — accepted; this is what shipped. It had previously been rejected as impossible, on the belief that Claude Code could only discover a plugin-root `.mcp.json` and had no manifest key. That belief was wrong: `mcpServers` accepts a path string, an array, or an inline object, and all forms load at runtime. The rejection stood for one release and cost a broken host. The reusable lesson is that "the primary host has no other way" was never measured before it was believed.
3. **Ship a wrapper that re-resolves the working directory before exec'ing `archcore mcp`** — rejected twice over: it recreates `bundled-cli-launcher.adr`, removed in v0.4.0 for eight categories of bugs, and the input such a wrapper needs — the project path — is precisely what #4234 says the host does not pass.
4. **Rename only the server key, from `archcore` to `archcore-plugin`, so the two servers stop colliding** — rejected because it keeps a server that cannot resolve the project and, from CLI v0.6.7, must fail loudly on every session; trading a silent collision for a permanent visible error is not a fix.
5. **Wait for upstream to fix #4234** — rejected as the whole plan, because the issue is open with no maintainer response and the adapter ships now. It is retained as a revisit trigger below.

## Consequences

- Copilot cannot write a user's documents into its own plugin cache through this MCP, and — the part the first pass missed — the project server the user was told to register survives to serve the session.
- Cursor gains the same fix. Its loader accepts `[".mcp.json", "mcp.json"]`, so the plugin had been contributing an install-directory-spawned server there all along, contrary to what `cursor-mcp-architecture.adr` first claimed. The rename removes it, and Cursor is not exposed by part 2, because its manifest resolution breaks on the first manifest that parses and `.cursor-plugin/plugin.json` exists.
- Neutral: Codex is untouched, because `.codex.mcp.json` and its explicit manifest key were already off the discovered names.
- Neutral: whether omitting the key is *sufficient* was settled on 2026-08-03 — it is not, twice over. `@test/integration/copilot-plugin-smoke.bats` pins the resulting merge against a real `copilot plugin install`, using a sentinel command so it is an identity check rather than a name check.
- Tradeoff: the plugin stops being self-contained on Copilot. Installing it yields skills, commands, agents, and hooks, but no MCP tools until host wiring runs, so the init skill must always emit the host-wiring line for copilot, which pins that path to CLI v0.6.4 or later.
- Tradeoff: Claude Code's MCP now hangs on one manifest key. The conventional filename used to be a second, independent route and is gone; delete the key and the primary host loses every document tool silently. Three tests and three mutations stand on that line.

## Superseded when

- Upstream closes #4234, by passing `COPILOT_PROJECT_DIR` to MCP children or launching them from the project directory, which would reopen part 3. Parts 1 and 2 stay regardless, because the name collision is independent of the working-directory defect.
- Copilot documents a manifest key that suppresses `.mcp.json` auto-discovery, which would make the filename lever unnecessary.
