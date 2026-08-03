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

GitHub Copilot CLI has a documented plugin system — subdirectory install specs, manifest discovery, a hooks config schema, and plugin-shipped `.mcp.json`, per docs.github.com cli-plugin-reference and hooks-reference, verified 2026-07-05 — and `bin/session-start` together with `bin/lib/normalize-stdin.sh` already handled a `copilot` host. The open design question was whether to reuse the Claude-format files verbatim through Copilot's partially documented Claude-compatibility layer, or to add native per-host adapter files as the plugin already does for Cursor and Codex. The maintainer decided on 2026-07-05.

## Decision

Ship the GitHub Copilot host adapter as **native adapter files** — a dedicated `.plugin/plugin.json` manifest, discovered before `.claude-plugin/plugin.json` so the Claude manifest stays untouched, plus `hooks/copilot.hooks.json` in Copilot's documented camelCase hook format (`sessionStart`, `preToolUse`, `postToolUse`) — installed through the subdirectory spec `archcore-ai/plugin:plugins/archcore`, scoped to Copilot CLI only, with pre-mutation denies emitted by the shared `bin/` guard scripts as stdout `{"permissionDecision":"deny",…}` keyed on `ARCHCORE_HOST=copilot`.

## Alternatives Considered

1. **Claude-compat reuse, pointing Copilot at the existing PascalCase `hooks/hooks.json`** — rejected because the compatibility layer is documented only in changelog fragments: loading nested Claude hook groups from a *plugin* file, and acceptance of Claude's hook-output JSON schema, are both unverified, so the design would rest on behavior GitHub can change without notice and that only live probes could pin.
2. **A root-level Copilot manifest at the repo root** — rejected because the plugin's assets live in `plugins/archcore/` per `subdirectory-plugin-layout.adr`, so a repo-root install would recreate the discovery failure that broke Codex installs in GitHub issue #2.
3. **Delaying release until VS Code and cloud-agent parity** — rejected because self-serve plugin install in VS Code agent mode is not documented, being enterprise-managed distribution in public preview, and cloud-agent sandboxes load hooks solely from repo-level `.github/hooks/*.json`; waiting would gate CLI users on surfaces this repository cannot fix.

## Consequences

- Adapter cost stays within the per-host budget of `multi-host-plugin-architecture.adr`: one manifest of about 14 lines, one hooks config, and one output branch in the `bin/` guard scripts. The hooks config outgrew its original 40 lines when path resolution became a candidate chain.
- The host-coverage-matrix structure test gained its `copilot` row, so a mutation-tool gap now fails CI.
- Tradeoff: deny semantics diverge per host. On Copilot **every** non-zero exit denies — exit 2 explicitly, and any other non-zero exit as `Denied by preToolUse hook (hook errored)` (hooks-reference, re-read 2026-07-27). Guard scripts branch on `ARCHCORE_HOST` to emit stdout `permissionDecision` JSON with exit 0, because that is how a deny carries its reason text, not because exit 2 fails to block.
- Tradeoff: a hook timeout fails **open** on this host. `@test/unit/hook-latency.bats` keeps both PreToolUse guards far enough inside the 1-second budget that the timeout path stays out of reach on realistic knowledge bases.
- Tradeoff: because every non-zero exit denies, a guard that cannot *start* is indistinguishable from a guard that refuses the write. Hook bootstrap is therefore a correctness concern on this host alone — an unresolved plugin root or a missing interpreter locks the user out of their own repository instead of degrading to no enforcement.
- Tradeoff: VS Code and cloud-agent users get no plugin in this release. The limitation is documented rather than papered over, and a repo-level `.github/hooks/*.json` template for cloud agents is a separable follow-up.

### Three premises did not survive contact with the release

The first two were recorded as things the smoke test would confirm, and documentation settled them first in the other direction. The third arrived as a user-visible failure.

- **Agent file naming.** A plain `NAME.md` does not load. Copilot's plugin reference defines the `agents` field as "Path(s) to agent directories (`.agent.md` files)", and the how-to and concepts pages agree. Shipping `*.agent.md` copies is now the design, with one addition: the copies live in `copilot-agents/` rather than beside the originals, because `.agent.md` still matches the `*.md` glob Claude Code and Cursor use, so a sibling copy would give both hosts two files declaring the same `name:`. `cmp` in `@test/structure/agents.bats` holds the parity.
- **Plugin-shipped MCP.** It is documented and it does not work: Copilot launches a plugin's MCP child in the plugin install directory with no project path (github/copilot-cli#4234), so writes land in `~/.copilot/installed-plugins/` while every tool reports success. `copilot-mcp-architecture.adr` supersedes that part of this decision — the manifest ships no `mcpServers`, and Copilot users register the server per project.
- **`${COPILOT_PLUGIN_ROOT}` as the plugin-root substitution.** Reported from a live session on 2026-07-27 as `/bin/sh: /bin/session-start: No such file or directory`, because `"${COPILOT_PLUGIN_ROOT}"/bin/session-start` collapses to the literal path `/bin/session-start` when the variable is empty. Re-reading cli-plugin-reference and hooks-reference found the variable documented nowhere; the only documented spelling is `${PLUGIN_ROOT}`, which `hooks/codex.hooks.json` already used. `copilot-host-support.rnd` had sourced the three-name list to hooks-reference, and that page does not carry it. Each of the six commands now probes `$COPILOT_PLUGIN_ROOT`, `$PLUGIN_ROOT`, and `$CLAUDE_PLUGIN_ROOT` in turn with `-x`, execs the first that holds the script, and otherwise warns on stderr and exits 0 — fail-open, because the alternative on this host is denying every edit. The variables are unbraced throughout, so the command behaves the same whether Copilot expands them or `sh` does.

A fourth premise was confirmed rather than broken: `commands` is a real manifest field with no default path, so the explicit pointer this adapter ships is what makes the seven `/archcore:*` wrappers exist at all.

What the third premise cost is a process finding rather than a code one. Static tests asserted the exact command strings, so they matched the defect instead of catching it. The replacements in `@test/structure/copilot-plugin.bats` execute the commands under `env -u` and assert behavior: exit 0 with a warning when nothing resolves, each candidate sufficient on its own, and a dead candidate skipped rather than fatal. Fault injection confirms each assertion fails only for its own defect. String equality on a shell command is not a test of that command, and `cli-integration-tests.rule` now has a concrete case behind it.

## Superseded when

- Copilot documents self-serve plugin install for VS Code agent mode, which would reopen the CLI-only scope. Hooks are a separate and currently harder question there, because hooks-reference names exactly two supported surfaces, CLI and cloud agent.
- Copilot publishes a complete Claude-compatibility reference for plugin manifests and hooks, which would reopen the reuse alternative and could drop the two Copilot-specific config files.
- Copilot documents which plugin-root variable a plugin hook receives, and under which load paths, which would collapse the candidate chain back to one variable. A recorded Copilot row in `host-probe-protocol.spec` would settle it ahead of the documentation.
- The cloud agent documents that `enabledPlugins` plugins contribute hooks inside the sandbox, which would reopen cloud-agent support. The MCP half of that question is governed by `copilot-mcp-architecture.adr` and github/copilot-cli#4234.
