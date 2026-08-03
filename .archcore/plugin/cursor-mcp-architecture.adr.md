---
title: "Cursor MCP Architecture — User-Level Install Only, No Plugin-Shipped MCP"
status: accepted
tags:
  - "architecture"
  - "cursor"
  - "multi-host"
  - "plugin"
---

> **Update (2026-08-03):** Context §3 called the trigger "empirically unresolved" — it is now resolved, and the answer is the less comfortable one. Cursor's plugin loader (`Cursor.app/.../cursor-agent-exec/dist/main.js`, Cursor 2.x) auto-discovers **`[".mcp.json", "mcp.json"]`** — both spellings. So the file we believed Cursor ignored, and could not remove because Claude Code needed it, was being registered as a plugin MCP server all along: spawned from the plugin install directory, with no `cwd` field to correct it. Layer 1 (stripped distribution) is what kept this from re-reporting the original bug; the mechanism itself never stopped. Closed 2026-08-03 by renaming the file to `.claude.mcp.json` and pointing Claude Code at it from `.claude-plugin/plugin.json` — see `copilot-mcp-architecture.adr`, where the same rename was forced by a different host, and `test/structure/plugin-mcp-isolation.bats`, which now pins that no auto-discovered filename exists at the plugin root for any host. Cursor is not exposed by the new manifest key: its manifest resolution iterates `[".cursor-plugin/plugin.json", ".claude-plugin/plugin.json"]` and breaks on the first that parses, and ours parses.

## Context

In May 2026 a Cursor user reported that, after installing the Archcore plugin and querying for documents in an empty project, the MCP returned ~35 documents that did not belong to their project. Investigation revealed they were the plugin team's own development docs from `.archcore/plugin/` — being served because Cursor had spawned the plugin's MCP server with `cwd` pointing at the plugin install directory (`~/.cursor/plugins/cache/archcore-plugins/archcore/<sha>/`), where our bundled `.archcore/` sat.

Three independent facts produced this leak:

1. **Cursor's MCP stdio schema has no `cwd` field.** The documented fields are `type`, `command`, `args`, `env`, `envFile` ([cursor.com/docs/mcp](https://cursor.com/docs/mcp)). The `cwd` field we had been shipping in `cursor.mcp.json` was silently ignored. The community feature request to add it has been open since May 2025 ([forum #74861](https://forum.cursor.com/t/allow-workspacefolder-in-mcp-project-configration/74861)).

2. **Cursor spawns stdio MCP servers with cwd = MCP install location, not the workspace.** Confirmed by [forum #99215](https://forum.cursor.com/t/how-get-the-correct-current-work-directory-in-mcp-server/99215). For plugin-shipped MCPs, the install location is the plugin cache directory.

3. **Cursor 2.5+ auto-detects plugin-shipped MCP configs.** Per the [official plugins reference](https://cursor.com/docs/reference/plugins.md), an `mcp.json` at the plugin root is registered automatically (and a `mcpServers` field in `plugin.json` accepts inline or referenced configs). Our `cursor.mcp.json` at the plugin root appears to have been picked up by a related heuristic and registered under "Plugin MCP Servers." Whether the trigger was `cursor.mcp.json` or `.mcp.json` (our Claude Code config) was left empirically unresolved here — **settled 2026-08-03: the loader accepts both `.mcp.json` and `mcp.json`, so our Claude config qualified.** See the update at the top.

The CLI itself contributed by reading `.archcore/` from `os.Getwd()` (`cli/cmd/mcp.go:22`) with no walk-up, no project flag, and no sanity check on the resolved root.

## Decision

We will **not** ship a plugin-shipped MCP server for Cursor. Cursor users install the MCP at the user or project level (`~/.cursor/mcp.json` or `.cursor/mcp.json`) by copying `docs/cursor.mcp.example.json`. The template passes the workspace path explicitly via `args`, not via the unsupported `cwd` field:

```json
{
  "mcpServers": {
    "archcore": {
      "type": "stdio",
      "command": "archcore",
      "args": ["mcp", "--project", "${workspaceFolder}"]
    }
  }
}
```

This is enforced by three layers of defense:

**Layer 1 — Public distribution is stripped.** The `dev` branch retains the plugin team's `.archcore/`, `reference-materials/`, `test/`, `Makefile`, and CI workflows. The `main` branch is synthesized by `.github/workflows/release.yml` with those paths removed. Users only ever clone `main`. See `docs/release.md` on `dev` for the blocklist and process. In hindsight this layer was doing more work than the others: with §3 now settled, it is what stopped the auto-discovered server from serving our docs, since the server itself was still being registered.

**Layer 2 — No auto-discovered MCP filename at the plugin root.** `cursor.mcp.json` was moved to `docs/cursor.mcp.example.json`, and since 2026-08-03 the Claude config is `.claude.mcp.json`, off the discovery list entirely. `.cursor-plugin/plugin.json` deliberately omits `mcpServers`. This is the layer that changed: it used to be "keep the plugin root clean of *Cursor's* filenames", which was insufficient because Cursor's list includes the dotted spelling too.

**Layer 3 — Runtime guards.**
- `bin/session-start` exits silently when the working directory sits inside a plugin install — install-cache path fragments in `$PWD`, plus a bounded upward walk for `.cursor-plugin/`, `.claude-plugin/`, `.codex-plugin/` manifests (extended per `host-wiring-parity.adr` to catch cwd in a *subdirectory* of the install) — preventing the hook from emitting the plugin's bundled context as if it were the user's.
- `archcore mcp` (CLI repo) applies the same protection via `resolveProjectRoot`, which since `host-wiring-parity.adr` rejects any project root containing `.cursor/plugins/`, `.claude/plugins/`, `.codex/plugins/`, or `plugins/cache/` fragments — while accepting a plugin *developer* repo whose root merely carries the manifests. When rejected, the server refuses to read `.archcore/` and exits with a clear error pointing at `--project`.
- `archcore mcp --project <path>` accepts an explicit project root, with `ARCHCORE_PROJECT_ROOT` as an env-var fallback for hosts without variable interpolation.

Each layer alone is sufficient for the specific scenario observed; in combination they cover scenarios we cannot test (older Cursor versions, future Cursor auto-detect changes, other hosts adopting similar plugin-MCP patterns).

## Alternatives

**A. Ship `mcp.json` at the plugin root with `--project` in args.**
This is the canonical Cursor way to ship a plugin MCP. Rejected: Cursor's `${workspaceFolder}` interpolation inside plugin-MCP `args` is undocumented and the open feature request ([#74861](https://forum.cursor.com/t/allow-workspacefolder-in-mcp-project-configration/74861)) implies it does not work for plugin-MCPs. Until Cursor confirms support, we cannot rely on it.

**B. Bundle a launcher that fixes cwd before exec'ing `archcore mcp`.**
This was the pre-v0.4.0 design (see `bundled-cli-launcher.adr.md`). Rejected then for eight categories of bugs, and re-rejected here: the launcher's CWD-fixing logic depends on environment variables that Cursor does not consistently pass to plugin processes, recreating the bug class we already removed.

**C. Walk up the directory tree from cwd to find `.archcore/`.**
Mimics how `git` finds `.git`. Rejected: in user projects with nested subdirectories, this would surface the correct `.archcore/` — but in our specific failure mode (cwd inside plugin install dir), it surfaces our bundled `.archcore/` _faster_. Walk-up makes the bug worse, not better.

**D. Strip bundled `.archcore/` and rely solely on that.**
Rejected as the only defense: we cannot prove no plugin-shipped MCP file at the plugin root will accidentally trigger Cursor's auto-detect in some future version, and we cannot prove other hosts will not adopt similar quirks. Defense in depth. The 2026-08-03 finding is this alternative's vindication: the auto-detect we could not rule out was in fact firing.

## Consequences

**Positive**

- Cursor users no longer see the plugin team's dev docs in their MCP.
- The fix is robust to Cursor changing its auto-detection rules — the bundled `.archcore/` is gone regardless, and since 2026-08-03 so is every filename the rules key on.
- Hooks become safer across all hosts (the `session-start` guard catches misrouted launches everywhere, not just Cursor).
- The CLI gains an explicit `--project` flag, which is reusable for any host that cannot guarantee correct cwd (Gemini CLI, Continue, etc.).

**Negative**

- ~~Cursor onboarding gains a manual step (copy template into `~/.cursor/mcp.json`).~~ Closed by `host-wiring-parity.adr`: `/archcore:init` and `archcore init --agent cursor` now write a project-level `.cursor/mcp.json` carrying `--project ${workspaceFolder}` (the CLI's dedicated Cursor writer matches the template above; `archcore doctor --fix` converges configs written by older CLIs). The manual user-level copy remains only as a fallback for sessions where no wiring has run yet.
- The `dev → main` split requires discipline: nobody can hotfix on `main` directly without re-syncing from `dev`. Documented in `docs/release.md`.
- If Cursor ever supports `${workspaceFolder}` interpolation in plugin-MCP `args` (feature request #74861), we can revisit shipping `mcp.json` at the plugin root — but the CLI guard (Layer 3) stays as belt-and-suspenders.

**Neutral**

- Codex is unaffected: `.codex.mcp.json` was never on any host's discovery list and is reached through an explicit manifest key.
- Claude Code was the reason the discovered filename could not simply be deleted. That constraint turned out to be false — the manifest accepts an explicit path — and removing it is what let both this ADR and `copilot-mcp-architecture.adr` close. The claim previously recorded here, that Claude's and Codex's plugin-root configs "continue to work because both hosts inherit cwd from the user's project process", was true of the *spawn* but silent about the *discovery*: Cursor was reading Claude's file. It is corrected rather than deleted because the reasoning error — asserting a host ignores a file without measuring it — is the reusable part.
