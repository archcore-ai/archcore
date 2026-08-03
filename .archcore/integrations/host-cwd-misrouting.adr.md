---
title: "AI Hosts May Spawn the MCP Server Inside a Plugin Install Cache — Guard the Resolved Project Root"
status: accepted
tags:
  - "cli"
  - "integrations"
  - "mcp"
---

## Context

`archcore mcp` resolves the project root it will read and write `.archcore/` under. Nothing in the MCP protocol tells a server where the user's workspace is, so the resolver falls back to the process working directory — and hosts do not guarantee that cwd.

Two observed cases, both landing the server inside a plugin *install cache* rather than a workspace:

- **Cursor** does not guarantee the cwd it spawns stdio MCP servers and hooks with (forum #99215).
- **Copilot** launches a plugin's auto-discovered MCP children in the install root — `~/.copilot/installed-plugins/_direct/<mangled-plugin-name>/` — with no project path passed at all (github/copilot-cli#4234). There is no per-invocation flag a user can edit here: the host reads the plugin's bundled `.mcp.json`.

Treating a cache directory as the project root has two failure modes. The read path serves the plugin's own bundled `.archcore/` documents as if they were the user's. The write path is worse: `create_document` silently writes the user's documents into a cache directory that no git repository tracks and that the next plugin update deletes.

The complication is that a plugin **developer** repo is a perfectly ordinary project. Its root carries `.claude-plugin/marketplace.json` or `.cursor-plugin/plugin.json`, and it must keep resolving normally.

## Decision

`resolveProjectRoot` (@cmd/mcp_root.go) picks the root in strict precedence — `--project` flag, then `ARCHCORE_PROJECT_ROOT`, then `os.Getwd()` — and rejects a resolved root that sits inside a known plugin install cache.

**The guard applies to implicit sources only.** An explicit `--project` states user intent, is trusted as-is, and stays usable as the recovery path the guard's own error recommends. Env and cwd are the sources a host can misroute.

**Detection keys on install-cache path fragments, never on plugin manifests.** `isPluginCachePath` lowercases the slash-normalized absolute path, appends a trailing separator, and reports a match if it contains any fragment:

```
/.cursor/plugins/   /.claude/plugins/   /.codex/plugins/
/.copilot/installed-plugins/   /plugins/cache/
```

Three properties of that list are load-bearing and must hold for every fragment added later:

1. **Lowercase** — the candidate is case-folded before matching, because macOS and Windows filesystems are case-insensitive and `.Cursor/Plugins/` must not slip through.
2. **Delimited by `/` at both ends** — fragments match whole path segments only. `filepath.Abs` always returns a Cleaned absolute path, so a genuine cache hit always carries a separator before the fragment; the leading `/` therefore costs no detection while keeping a user project at `.../my.copilot/installed-plugins/app` from being refused.
3. **Trailing separator appended to the candidate** — so a root that *is* the cache directory itself matches, which is exactly the Copilot case.

The rejection error names both recovery paths, because for Copilot the flag is not editable: pass `--project`, or register a project-level server with `archcore init --agent <agent> --project <path>`.

Where a host *does* support interpolation, prevention beats detection: the Cursor MCP entry ships `--project ${workspaceFolder}` (@internal/agents/mcp_helpers.go), so the root never depends on spawn cwd in the first place.

This is a heuristic against host misrouting, **not a security boundary** — symlinks are deliberately not resolved.

## Alternatives

- **Detect plugin manifests in the root** (`.claude-plugin/`, `.cursor-plugin/`): rejects legitimate plugin-developer repos, which are the projects most likely to be running archcore against a plugin. Rejected; pinned against by `TestResolveProjectRoot_AcceptsPluginDeveloperRepo`.
- **Unanchored substring fragments** (the first implementation): matched `my.copilot/installed-plugins/` inside an ordinary user path. Anchoring at segment boundaries was measured to drop zero real detections, so the false positive was not worth accepting.
- **Walk up for a `.git` or `.archcore/` directory**: silently changes which project is served, and picks a wrong one whenever a cache happens to sit inside a repo. Rejected — a wrong root that looks right is worse than a refusal.
- **Apply the guard to `--project` as well**: removes the only recovery path a user has once a host misroutes. Rejected.
- **Refuse to start unless a root is passed explicitly**: breaks every correctly-behaving host and the ordinary `cd project && archcore mcp` case. Rejected.

## Consequences

- The fragment list is a maintenance surface: a new host, or a host relocating its cache, needs a new fragment. Until then the failure is silent — the misrouted root is accepted.
- Contributors adding a fragment must preserve all three properties above. The doc comment on `pluginCacheFragments` states them; `TestResolveProjectRoot_CopilotLookalikeUserPath_NotRejected` pins the anchoring and `TestResolveProjectRoot_CopilotCacheRootItself` pins the trailing separator.
- A user project genuinely living under a path segment named exactly `.copilot/installed-plugins/` (or the other four) is refused on implicit sources and needs `--project`. Judged vanishingly unlikely.
- `install_host_config` and `archcore init --agent` remain the supported way to wire a project-level server, and the guard's error message now points at them.
