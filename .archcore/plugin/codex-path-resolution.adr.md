---
title: "Codex MCP and Hooks Path Resolution"
status: accepted
tags:
  - "codex"
  - "multi-host"
  - "plugin"
---

## Context

Codex 0.130.0 resolves paths in plugin MCP and hooks configs differently from Claude Code. We hit three ENOENT-or-equivalent failures porting the plugin to Codex:

1. **MCP command resolution** — Codex does **not** substitute `${CODEX_PLUGIN_ROOT}` or `${CLAUDE_PLUGIN_ROOT}` in `command`/`args`. The only plugin-aware rewrite is in `core-plugins/src/loader.rs::normalize_plugin_mcp_server_value`, which rebases a relative `cwd` field against the plugin install root. So a relative `command: "./bin/archcore"` only resolves when `cwd: "."` is also set.

2. **MCP env stripping** — Codex's spawn site `codex-rs/rmcp-client/src/stdio_server_launcher.rs:236-267` calls `.env_clear()` unconditionally and rebuilds the child env from an allowlist (`utils.rs::DEFAULT_ENV_VARS`: HOME LOGNAME PATH SHELL USER __CF_USER_TEXT_ENCODING LANG LC_ALL TERM TMPDIR TZ) plus anything declared in the manifest's `env_vars`. `PWD` is **not** in the default allowlist. Combined with the `cwd: "."` rebase above, the MCP child starts with no PWD and `getcwd()` = plugin install dir — every `.archcore/` operation lands in the plugin cache, not the user's project.

3. **Hooks** — Codex's hooks engine (`codex-rs/hooks/src/engine/discovery.rs`) injects two env vars before spawn: a canonical host-neutral `PLUGIN_ROOT` and a `CLAUDE_PLUGIN_ROOT` compat shim for porting old Claude plugins. It does **not** treat `./...` as plugin-relative.

A previous config used `./bin/archcore` for MCP and `./bin/...` for hooks; both broke under Codex. See: <https://github.com/openai/codex/issues/19582>.

## Decision

- `.codex.mcp.json`:
  - `command: "./bin/archcore"`, `args: ["mcp"]`.
  - `cwd: "."` — required; Codex rebases against the plugin install root so the relative command resolves.
  - `env_vars: ["ARCHCORE_CWD"]` — passes through one custom env variable. We deliberately use a **non-PWD name**: POSIX shells (sh, dash, bash --posix) resync `$PWD` to `getcwd()` at script startup, so any passed-through `PWD` is overwritten before the launcher's first line runs. Custom names like `ARCHCORE_CWD` are not touched.
- `bin/archcore` (the sh launcher) has two cooperating blocks at the top:
  - **Step 0 — honor `$ARCHCORE_CWD`.** If set and points at a real directory, `cd` there before resolving and exec'ing the real archcore CLI. No-op when unset (Claude Code and other hosts that don't chdir need nothing).
  - **Step 0b — plugin-install guard.** When invoked as `archcore mcp` AND cwd path matches `*/plugins/cache/*` AND all three plugin-root markers are present (`.codex-plugin/plugin.json`, `.codex.mcp.json`, `bin/archcore`), refuse to start and print an actionable error pointing to the wrapper recipe. Escape hatch: `ARCHCORE_ALLOW_PLUGIN_CWD=1` for plugin maintainers who want to run MCP against the plugin's own docs intentionally. This converts the silent fallback (MCP operates on plugin's own `.archcore/`) into a loud, user-fixable failure.
- User opt-in. Users install a shell wrapper that sets `ARCHCORE_CWD=$PWD` just before invoking codex. Recipes for fish/bash/zsh are in `codex-local-plugin-testing.guide.md` and printed by the guard itself.
- `hooks/codex.hooks.json`: `${PLUGIN_ROOT}/bin/...` (canonical host-neutral name).
- `.mcp.json` (Claude) stays unchanged (`${CLAUDE_PLUGIN_ROOT}/bin/archcore`); `hooks/hooks.json` stays unchanged.

Contracts enforced by:

- `test/structure/codex-plugin.bats` — `.codex.mcp.json` shape: command, args[0], cwd, env_vars must include `ARCHCORE_CWD`.
- `test/structure/cli-contract.bats` — args[0] is an allowlisted subcommand.
- `test/unit/launcher.bats` — ARCHCORE_CWD chdir, nonexistent-dir tolerance, no-op when unset, **and six guard tests** covering: cache-cwd refusal, dev-checkout pass-through, ARCHCORE_CWD-honored bypass, `ARCHCORE_ALLOW_PLUGIN_CWD=1` escape hatch, non-mcp subcommand pass-through, partial-marker pass-through.
- `test/structure/hooks.bats` — hooks resolver expands `${PLUGIN_ROOT}`.

## Alternatives Considered

- **Python trampoline** that reads `$PWD` and chdir's before exec'ing the sh launcher. Verified end-to-end with an A/B test against the real plugin cache; works perfectly. Rejected per project convention against introducing new languages/tooling without explicit decision (`stack-and-tooling.rule`). Adds Python as a third runtime to a shell+Go plugin for a problem solvable with two lines of shell + a user-side wrapper + a marker-based guard.
- **Compiled Go trampoline binary.** No new runtime dep, but adds a darwin/linux × amd64/arm64 build matrix and macOS code-signing step. Rejected as over-engineering.
- **`$PWD`-rebase inside the existing sh launcher.** Dead end — POSIX shells resync `$PWD` to `getcwd()` at startup. Verified on macOS across `/bin/sh`, `dash`, `bash --posix`; no shell flag opts out.
- **`env_vars: ["PWD"]` without a non-shell trampoline.** Codex passes PWD through but sh erases it immediately. Useless on its own.
- **Use `${CODEX_PLUGIN_ROOT}` in MCP `command`/`args`.** Codex does no env substitution there. Would silently fail.
- **Drop `cwd: "."` and use a `${PLUGIN_ROOT}`-absolute command.** No env substitution; would not resolve.
- **Use `${CLAUDE_PLUGIN_ROOT}` in Codex hooks.** Works via compat shim but borrowing another host's name in a Codex-native config is misleading; `PLUGIN_ROOT` is canonical.
- **Absolute paths.** Plugin install path is host-controlled and not stable.
- **Silent opt-in (no guard).** Earlier iteration. Rejected because a fresh user without the wrapper experiences the original bug (MCP operates on plugin cache `.archcore/`, listing plugin docs as the user's). Hard to diagnose. The guard converts this to a loud, self-documenting failure.

## Consequences

- Codex MCP cwd is opt-in. Users must install a shell wrapper (`function codex; env ARCHCORE_CWD=$PWD command codex $argv; end` in fish, or equivalent in bash/zsh). Without the wrapper, the **guard refuses to start the MCP** and prints the wrapper recipe inline. No more silent fallback to plugin docs.
- Zero new runtime dependencies. The plugin remains shell + Go. No Python, no compiled trampoline binaries.
- Codex and Claude diverge only in the `env_vars` declaration. Same `bin/archcore` launcher on both hosts; both the `ARCHCORE_CWD` block and the guard are no-ops when not triggered (`ARCHCORE_CWD` unset → no chdir; non-mcp subcommand OR non-cache cwd OR missing markers → guard skipped).
- Plugin maintainers can bypass the guard with `ARCHCORE_ALLOW_PLUGIN_CWD=1` to operate on the plugin's own `.archcore/` (e.g., for plugin development against an installed copy).
- Tests pin all three contracts (manifest shape, allowlisted subcommand, launcher chdir + guard behavior).
- Codex plugin hooks still require `codex features enable plugin_hooks` (currently `under development, false` in 0.130.0). The hooks contract is in place ahead of GA.
- If upstream ships `${PLUGIN_ROOT}` MCP substitution (openai/codex#19582), we may drop `cwd: "."` and inherit Codex's caller CWD directly — retiring the wrapper. The `ARCHCORE_CWD` mechanism and the guard both stay harmless until then.
