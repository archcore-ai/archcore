---
title: "Cursor MCP Architecture — User-Level Install Only, No Plugin-Shipped MCP"
status: accepted
tags:
  - "architecture"
  - "cursor"
  - "multi-host"
  - "plugin"
---

**Update (2026-08-03).** Finding 3 below was recorded as empirically unresolved; it is now resolved, and the answer is the less comfortable one. Cursor's plugin loader (`Cursor.app/.../cursor-agent-exec/dist/main.js`, Cursor 2.x) auto-discovers both spellings, `[".mcp.json", "mcp.json"]`. The file believed to be ignored by Cursor — and unremovable because Claude Code needed it — was being registered as a plugin MCP server all along, spawned from the plugin install directory with no `cwd` field to correct it. Layer 1 is what kept this from re-reporting the original bug; the mechanism itself never stopped. Closed on 2026-08-03 by renaming the file to `.claude.mcp.json` and pointing Claude Code at it from `.claude-plugin/plugin.json`, the same rename that `copilot-mcp-architecture.adr` forced from a different host, with `@test/structure/plugin-mcp-isolation.bats` pinning that no auto-discovered filename exists at the plugin root for any host.

## Context

In May 2026 a Cursor user reported that, after installing the Archcore plugin and querying an empty project, the MCP returned about 35 documents that did not belong to that project. The documents were the plugin team's own development docs from the bundled `.archcore/`, served because Cursor had spawned the plugin's MCP server with its working directory pointing at the plugin install directory `~/.cursor/plugins/cache/archcore-plugins/archcore/<sha>/`. The CLI contributed by reading `.archcore/` from `os.Getwd()` in `cli/cmd/mcp.go:22`, with no walk-up, no project flag, and no sanity check on the resolved root.

## Findings

1. **Cursor's MCP stdio schema has no `cwd` field.** The documented fields are `type`, `command`, `args`, `env`, and `envFile` ([cursor.com/docs/mcp](https://cursor.com/docs/mcp)). The `cwd` field previously shipped in `cursor.mcp.json` was ignored silently, and the community feature request to add one has been open since May 2025 ([forum #74861](https://forum.cursor.com/t/allow-workspacefolder-in-mcp-project-configration/74861)).
2. **Cursor spawns a stdio MCP server with its working directory set to the MCP install location rather than the workspace**, confirmed by [forum #99215](https://forum.cursor.com/t/how-get-the-correct-current-work-directory-in-mcp-server/99215). For a plugin-shipped MCP, that location is the plugin cache directory.
3. **Cursor 2.5 and later auto-detect a plugin-shipped MCP config.** Per the [official plugins reference](https://cursor.com/docs/reference/plugins.md), an `mcp.json` at the plugin root is registered automatically, and a `mcpServers` field in `plugin.json` accepts inline or referenced configs. Whether the trigger was `cursor.mcp.json` or the Claude Code `.mcp.json` was left unresolved when this decision was first recorded, and was settled on 2026-08-03: the loader accepts both spellings, so the Claude config qualified.

## Decision

Ship **no plugin-shipped MCP server for Cursor**. A Cursor user installs the MCP at user or project level, in `~/.cursor/mcp.json` or `.cursor/mcp.json`, from the template at `@docs/cursor.mcp.example.json`, which registers a stdio server named `archcore` running `archcore` with `args` of `["mcp", "--project", "${workspaceFolder}"]` — passing the workspace path explicitly through `args`, since the `cwd` field of finding 1 does not exist.

Three layers of defense enforce it.

**Layer 1 — the public distribution is stripped.** The `dev` branch retains the plugin team's `.archcore/`, `reference-materials/`, `test/`, `Makefile`, and CI workflows; `main` is synthesized by `.github/workflows/release.yml` with those paths removed, and users only ever clone `main`. In hindsight this layer was doing more work than the others: with finding 3 settled, it is what stopped the auto-discovered server from serving the plugin's own documents, because the server itself was still being registered.

**Layer 2 — no auto-discovered MCP filename at the plugin root.** `cursor.mcp.json` moved to `docs/cursor.mcp.example.json`, and since 2026-08-03 the Claude config is `.claude.mcp.json`, off the discovery list entirely. `.cursor-plugin/plugin.json` omits `mcpServers` deliberately. This is the layer that changed: it used to mean "keep the plugin root clean of *Cursor's* filenames", which was insufficient, because Cursor's list includes the dotted spelling too.

**Layer 3 — runtime guards.** `bin/session-start` exits silently when the working directory sits inside a plugin install, detected by install-cache path fragments in `$PWD` plus a bounded upward walk for `.cursor-plugin/`, `.claude-plugin/`, and `.codex-plugin/` manifests, extended per `host-wiring-parity.adr` to catch a working directory in a *subdirectory* of the install. `archcore mcp` applies the same protection through `resolveProjectRoot`, which rejects any project root containing a `.cursor/plugins/`, `.claude/plugins/`, `.codex/plugins/`, or `plugins/cache/` fragment while accepting a plugin *developer* repository whose root merely carries the manifests; on rejection the server refuses to read `.archcore/` and exits with an error naming `--project`. `archcore mcp --project <path>` accepts an explicit project root, with `ARCHCORE_PROJECT_ROOT` as an environment fallback for hosts without variable interpolation.

Each layer alone is sufficient for the scenario observed; together they cover scenarios that cannot be tested — older Cursor versions, future auto-detect changes, and other hosts adopting similar plugin-MCP patterns.

## Alternatives Considered

1. **Ship `mcp.json` at the plugin root with `--project` in `args`**, the canonical Cursor way to ship a plugin MCP — rejected because Cursor's `${workspaceFolder}` interpolation inside plugin-MCP `args` is undocumented, and the open feature request #74861 implies it does not work for plugin MCPs; until Cursor confirms support it cannot be relied on.
2. **Bundle a launcher that fixes the working directory before exec'ing `archcore mcp`**, the pre-v0.4.0 design — rejected because `bundled-cli-launcher.adr` already removed it over eight categories of bugs, and because the launcher's CWD-fixing logic depends on environment variables Cursor does not consistently pass to plugin processes, which recreates the removed bug class.
3. **Walk up the directory tree from the working directory to find `.archcore/`**, the way `git` finds `.git` — rejected because in the actual failure mode, a working directory inside the plugin install, walk-up surfaces the bundled `.archcore/` faster, making the bug worse rather than better.
4. **Strip the bundled `.archcore/` and rely on that alone** — rejected as a sole defense, because no proof exists that some future Cursor version will not trigger auto-detect on another plugin-root filename, nor that other hosts will not adopt similar quirks. The 2026-08-03 finding vindicates this reasoning: the auto-detect that could not be ruled out was in fact firing.

## Consequences

- Cursor users no longer see the plugin team's development documents in their MCP.
- The fix survives a change to Cursor's auto-detection rules: the bundled `.archcore/` is gone regardless, and since 2026-08-03 so is every filename those rules key on.
- Hooks became safer on every host, because the `session-start` guard catches a misrouted launch everywhere rather than only on Cursor.
- The CLI gained an explicit `--project` flag, reusable for any host that cannot guarantee the working directory.
- Neutral: Codex is unaffected, because `.codex.mcp.json` was never on any host's discovery list and is reached through an explicit manifest key.
- Neutral: Claude Code was the reason the discovered filename could not simply be deleted. That constraint turned out to be false — the manifest accepts an explicit path — and removing it is what let both this decision and `copilot-mcp-architecture.adr` close. The claim previously recorded here, that Claude's and Codex's plugin-root configs continue to work because both hosts inherit the working directory from the user's project process, was true of the *spawn* but silent about the *discovery*: Cursor was reading Claude's file. It is corrected rather than deleted, because the reasoning error — asserting that a host ignores a file without measuring it — is the reusable part.
- Tradeoff: the `dev → main` split requires discipline, since nobody can hotfix `main` directly without re-syncing from `dev`. `docs/release.md` documents the process.
- Closed tradeoff: Cursor onboarding once gained a manual step, copying the template into `~/.cursor/mcp.json`. `host-wiring-parity.adr` closed it — `/archcore:init` and `archcore init --agent cursor` now write a project-level `.cursor/mcp.json` carrying `--project ${workspaceFolder}`, and `archcore doctor --fix` converges configs written by older CLIs. The manual user-level copy remains only as a fallback before wiring has run.

## Superseded when

- Cursor supports `${workspaceFolder}` interpolation inside plugin-MCP `args`, per feature request #74861, which would reopen shipping `mcp.json` at the plugin root. The CLI guard of Layer 3 stays regardless.
- Cursor documents a `cwd` field for its MCP stdio schema, which would remove finding 1 and with it the reason the template carries `--project`.
