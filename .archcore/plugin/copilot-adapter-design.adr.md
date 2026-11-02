---
title: "Copilot Host Adapter — Native Adapter Files, Subdir Install, CLI-Only Scope"
status: accepted
tags:
  - "architecture"
  - "component:plugin"
  - "copilot"
  - "hooks"
  - "multi-host"
  - "plugin"
---

## Context

GitHub Copilot CLI has a documented plugin system: subdirectory install specs, manifest discovery, a hooks config schema, and plugin-shipped `.mcp.json` (docs.github.com cli-plugin-reference and hooks-reference, verified 2026-07-05). `bin/session-start` and `bin/lib/normalize-stdin.sh` already handled a `copilot` host. The open question was whether to reuse the Claude-format files through Copilot's partly documented Claude-compatibility layer, or to add native per-host adapter files as the plugin does for Cursor and Codex. The maintainer decided on 2026-07-05.

## Decision

Ship the GitHub Copilot host adapter as **native adapter files**: a dedicated `.plugin/plugin.json` manifest, discovered before `.claude-plugin/plugin.json` so the Claude manifest stays untouched, plus `hooks/copilot.hooks.json` in Copilot's documented camelCase hook format (`sessionStart`, `preToolUse`, `postToolUse`). The plugin installs through the subdirectory spec `archcore-ai/plugin:plugins/archcore`, is scoped to Copilot CLI only, and emits pre-mutation denies as stdout `{"permissionDecision":"deny",…}` keyed on `ARCHCORE_HOST=copilot`.

Three shipped details came from premises that failed:

- **Agent file naming.** A plain `NAME.md` does not load. Copilot's `agents` field expects `.agent.md` files. The copies live in `copilot-agents/`, not beside the originals, because `.agent.md` still matches the `*.md` glob that Claude Code and Cursor use, and a sibling copy would give both hosts two files declaring the same `name:`. `cmp` in `@plugin/test/structure/agents.bats` holds the parity.
- **Plugin-shipped MCP.** It is documented and does not work. Copilot launches a plugin's MCP child in the plugin install directory with no project path (github/copilot-cli#4234), so writes land in `~/.copilot/installed-plugins/` while every tool reports success. `copilot-mcp-architecture.adr` supersedes this part of the decision: the manifest ships no `mcpServers`, and Copilot users register the server per project.
- **`${COPILOT_PLUGIN_ROOT}` as the plugin-root substitution.** The variable is documented nowhere. The only documented spelling is `${PLUGIN_ROOT}`. An empty variable collapsed the command to the literal path `/bin/session-start`. Each command now probes `$COPILOT_PLUGIN_ROOT`, `$PLUGIN_ROOT`, and `$CLAUDE_PLUGIN_ROOT` in turn with `-x`, execs the first that holds the script, and otherwise warns on stderr and exits 0. It fails open, because the alternative on this host is denying every edit.

`commands` is a real manifest field with no default path, so the explicit pointer this adapter ships is what makes the four `/archcore:*` wrappers exist.

Static tests that asserted exact command strings matched the third defect instead of catching it. The replacements in `@plugin/test/structure/copilot-plugin.bats` execute the commands under `env -u` and assert behavior. String equality on a shell command is not a test of that command (`cli-integration-tests.rule`).

## Alternatives Considered

1. **Claude-compat reuse, pointing Copilot at the existing PascalCase `hooks/hooks.json`** — rejected because the compatibility layer is documented only in changelog fragments. Loading nested Claude hook groups from a plugin file, and acceptance of Claude's hook-output JSON schema, are both unverified. The design would rest on behavior GitHub can change without notice.
2. **A root-level Copilot manifest at the repo root** — rejected because the plugin's assets live in `plugins/archcore/` per `subdirectory-plugin-layout.adr`, and a repo-root install would recreate the discovery failure that broke Codex installs in GitHub issue #2.
3. **Delaying release until VS Code and cloud-agent parity** — rejected because self-serve plugin install in VS Code agent mode is not documented, and cloud-agent sandboxes load hooks only from repo-level `.github/hooks/*.json`. Waiting would gate CLI users on surfaces this repository cannot fix.

## Consequences

- Adapter cost stays within the per-host budget of `multi-host-plugin-architecture.adr`: one manifest, one hooks config, and one output branch in the `bin/` scripts.
- The host-coverage-matrix structure test has a `copilot` row, so a mutation-tool gap fails CI.
- Tradeoff: deny semantics diverge per host. On Copilot every non-zero exit denies (hooks-reference, re-read 2026-07-27). The deny path emits stdout `permissionDecision` JSON with exit 0 because that is how a deny carries its reason text, not because exit 2 fails to block.
- Tradeoff: a hook timeout fails open on this host. The `preToolUse` entry pins `timeoutSec: 2`. Since v0.7.0 (`cli-owns-layers-4-5.adr`) the launcher does only the plugin-cache guard and the `cli-gte` probe before handing the payload to the CLI. `@plugin/test/unit/hook-latency.bats` holds that budget in CI.
- Tradeoff: because every non-zero exit denies, a guard that cannot start looks like a guard that refuses the write. An unresolved plugin root or a missing interpreter would lock the user out of their own repository. The launchers answer this by exiting 0 without output when the CLI is absent or older than 0.7.0.
- Tradeoff: VS Code and cloud-agent users get no plugin. A repo-level `.github/hooks/*.json` template for cloud agents is a separable follow-up.

## Superseded when

- Copilot documents self-serve plugin install for VS Code agent mode, which reopens the CLI-only scope. Hooks are a harder question there, because hooks-reference names only CLI and cloud agent as supported surfaces.
- Copilot publishes a complete Claude-compatibility reference for plugin manifests and hooks, which reopens the reuse alternative.
- Copilot documents which plugin-root variable a plugin hook receives, and under which load paths, which collapses the candidate chain to one variable. A Copilot row in `host-probe-protocol.spec` would settle it ahead of the documentation.
- The cloud agent documents that `enabledPlugins` plugins contribute hooks inside the sandbox, which reopens cloud-agent support. `copilot-mcp-architecture.adr` governs the MCP half.
