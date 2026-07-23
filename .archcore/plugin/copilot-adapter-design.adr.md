---
title: "Copilot Host Adapter — Native Adapter Files, Subdir Install, CLI-Only Scope"
status: accepted
tags:
  - "architecture"
  - "copilot"
  - "hooks"
  - "multi-host"
  - "plugin"
---

## Context

GitHub Copilot CLI has a documented plugin system (subdir install specs, manifest discovery, hooks config schema, plugin-shipped `.mcp.json` — docs.github.com cli-plugin-reference and hooks-reference, verified 2026-07-05), and `bin/session-start` plus `bin/lib/normalize-stdin.sh` already handle a `copilot` host. The open design question was whether to reuse our Claude-format files verbatim through Copilot's partially documented Claude-compatibility layer, or to add native per-host adapter files as we already do for Cursor (`hooks/cursor.hooks.json`) and Codex (`hooks/codex.hooks.json`). The maintainer decided on 2026-07-05.

## Decision

Ship the GitHub Copilot host adapter as **native adapter files** — a dedicated `.plugin/plugin.json` manifest (discovered before `.claude-plugin/plugin.json`, so the Claude manifest stays untouched) and `hooks/copilot.hooks.json` in Copilot's documented camelCase hook format (`sessionStart`, `preToolUse`, `postToolUse`) — installed via the subdir spec `archcore-ai/plugin:plugins/archcore`, scoped to **Copilot CLI only**, with pre-mutation denies emitted by the shared `bin/` guard scripts as stdout `{"permissionDecision":"deny",…}` JSON keyed on `ARCHCORE_HOST=copilot`.

## Alternatives Considered

1. **Claude-compat reuse (point Copilot at the existing `hooks/hooks.json`, PascalCase path)** — rejected because the compatibility layer is documented only in changelog fragments: loading nested Claude hook groups from a *plugin* file and acceptance of Claude's hook-output JSON schema are unverified, so the design would rest on behavior GitHub can change without notice and that we could only pin with live probes.
2. **Root-level Copilot manifest at the repo root** — rejected because the plugin's assets live in `plugins/archcore/` per `subdirectory-plugin-layout.adr`; a repo-root install would recreate the discovery-failure pattern that broke Codex installs (GitHub issue #2).
3. **Delaying release until VS Code / cloud-agent parity** — rejected because self-serve plugin install in VS Code agent mode is not documented (enterprise-managed distribution only, public preview) and cloud-agent sandboxes load hooks solely from repo-level `.github/hooks/*.json`; waiting gates CLI users on surfaces we cannot fix.

## Consequences

- Every contract the adapter uses is in the official reference (hooks config schema, stdin JSON payload, `permissionDecision` output, `timeoutSec`, subdir install spec) — release verification is a smoke test plus the standard three-probe protocol, not a research spike. [expected]
- Adapter cost stays within the per-host budget of `multi-host-plugin-architecture.adr`: one ~15-line manifest, one ~40-line hooks config, one output branch in `bin/` guard scripts.
- A fourth hooks config joins the sync surface: the host-coverage-matrix structure test MUST gain a `copilot` row so a mutation-tool gap fails CI.
- Deny semantics diverge per host: Copilot ignores the Claude "exit 2 = block" idiom (exit 2 is a warning there), so guard scripts branch on `ARCHCORE_HOST`; non-2 non-zero exits deny fail-closed, but a hook timeout (30 s default) fails **open** — a documented residual risk on this host.
- VS Code and cloud-agent users get no plugin in this release; the limitation is documented rather than papered over. A repo-level `.github/hooks/*.json` template for cloud agents is a separable follow-up.
- One naming detail rides on the smoke test: whether plain `NAME.md` agents load from a plugin `agents/` dir (plugin docs show only `*.agent.md`); fallback is shipping `*.agent.md` copies of the two agents.

## Superseded when

- Copilot documents self-serve plugin install for VS Code agent mode — revisit the CLI-only scope.
- Copilot publishes a complete Claude-compatibility reference for plugin manifests and hooks — revisit design A to drop the two Copilot-specific config files.
- Cloud agent documents that `enabledPlugins` plugins contribute hooks/MCP inside the sandbox — revisit cloud-agent support.