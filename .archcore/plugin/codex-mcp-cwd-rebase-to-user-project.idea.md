---
title: "Codex MCP CWD — Opt-In ARCHCORE_CWD Via Shell Wrapper"
status: accepted
tags:
  - "codex"
  - "multi-host"
  - "plugin"
---

## Idea

Make Codex-spawned archcore MCP servers operate in the user's project directory by combining three mechanisms — entirely within the existing shell launcher, no new language dependency:

1. **Manifest passthrough.** `.codex.mcp.json` declares `env_vars: ["ARCHCORE_CWD"]`. Codex spawns MCP children with `.env_clear()` plus a fixed allowlist (HOME LOGNAME PATH SHELL USER __CF_USER_TEXT_ENCODING LANG LC_ALL TERM TMPDIR TZ); `env_vars` is the only knob for carrying additional env through that wall. Using a **custom name** (not `PWD`) is essential — POSIX shells (sh, dash, bash --posix) resync `$PWD` to `getcwd()` at startup, so any passed-through `PWD` is overwritten before our code runs. Custom names pass through both barriers.

2. **`bin/archcore` shell launcher.** Two cooperating blocks at the top:
    - Step 0 — if `ARCHCORE_CWD` is set and points at a real directory, `cd` there before resolving and exec'ing the real archcore CLI.
    - Step 0b — **plugin-install guard.** When invoked as `archcore mcp` AND cwd matches `*/plugins/cache/*` AND all three plugin-root markers exist in cwd, refuse with an actionable error. Converts the silent-fallback-to-plugin-docs failure into a loud, self-documenting one. Escape hatch: `ARCHCORE_ALLOW_PLUGIN_CWD=1`.

3. **User opt-in via shell wrapper.** Users wrap their `codex` invocation so `ARCHCORE_CWD` is set to the current project just-in-time. Examples:

    - fish: `~/.config/fish/functions/codex.fish` — `function codex; env ARCHCORE_CWD=$PWD command codex $argv; end`
    - bash/zsh: in `~/.bashrc` / `~/.zshrc` — `codex() { ARCHCORE_CWD="$PWD" command codex "$@"; }`

   This is opt-in by design: it adds zero runtime dependencies (no Python, no extra binary), but the user has to install the wrapper once. If the wrapper is missing, the **guard refuses to start and prints the wrapper recipe inline** — the user can't silently land on plugin docs.

## Value

Before: every `mcp__archcore__*` call from Codex operated in `~/.codex/plugins/cache/<marketplace>/archcore/<version>/`. A `/archcore:bootstrap` session against an empty `test_project/` cost ~9 m 41 s, half of it spent realizing the docs landed in the wrong place.

After: with the shell wrapper installed, the MCP server's CWD is the project the user `cd`'d into before running `codex`. Without the wrapper, the guard fires loud and shows the user how to install it. No new tools, no new languages, no silent plugin-cache pollution.

## Possible Implementation

Shipped:

- `bin/archcore` (existing sh launcher):
  - Step 0 — `if [ -n "${ARCHCORE_CWD:-}" ] && [ -d "$ARCHCORE_CWD" ]; then cd "$ARCHCORE_CWD" 2>/dev/null || true; fi`. No-op when the var is unset (the normal Claude Code path).
  - Step 0b — guard block: refuse `archcore mcp` from cache cwd when plugin markers are present, unless `ARCHCORE_ALLOW_PLUGIN_CWD=1`.
- `.codex.mcp.json`: `command: "./bin/archcore"`, `args: ["mcp"]`, `cwd: "."`, `env_vars: ["ARCHCORE_CWD"]`.
- `test/unit/launcher.bats`: nine new tests — three for ARCHCORE_CWD chdir behavior, six for the guard (cache-cwd refusal, dev-checkout pass-through, ARCHCORE_CWD-honored bypass, escape hatch, non-mcp subcommand pass-through, partial-marker pass-through).
- `test/structure/codex-plugin.bats`: manifest contract pinned (command, args, cwd, env_vars must include ARCHCORE_CWD).

Discovery work that informed the design:

- Codex source: `codex-rs/rmcp-client/src/stdio_server_launcher.rs:236-267` is the MCP spawn site; `utils.rs::DEFAULT_ENV_VARS` is the allowlist. PWD is not on it. `env_vars` from the manifest is the only passthrough hook.
- POSIX `$PWD` resync is universal across `/bin/sh`, `dash`, `bash --posix` on macOS. Verified with `env PWD=/A sh -c 'echo $PWD'` from a different physical dir — child reports `getcwd()` even though parent set PWD to /A. Custom env var names (e.g. `ARCHCORE_CWD`) are untouched.

Rejected alternatives (see `codex-path-resolution.adr` for full context):

- **Python trampoline that reads `$PWD` and chdir's before exec'ing sh launcher.** Verified end-to-end — works perfectly, but introduces Python as a third language to a shell+Go plugin. Rejected per user decision: avoid adding new languages/tooling without explicit decision.
- **Compiled Go trampoline binary.** No new runtime dep, but adds a darwin/linux × amd64/arm64 build matrix and macOS code-signing step. Rejected as over-engineering.
- **`PWD`-rebase inside the existing sh launcher.** Dead end — sh resyncs PWD before the launcher's first line runs.
- **`$PWD`-only `env_vars` declaration without a non-shell trampoline.** Codex passes PWD through, but sh erases it. Useless on its own.
- **Silent opt-in without the guard.** Earlier iteration. Rejected because a fresh user without the wrapper experiences the original bug (MCP operates on plugin cache `.archcore/`, listing plugin docs as the user's). Hard to diagnose. The guard converts this to a loud, self-documenting failure.

Upstream coupling:

- `openai/codex#19582` — `${PLUGIN_ROOT}` substitution in MCP `command`/`args`. If/when shipped, we may drop `cwd: "."` and inherit Codex's caller CWD directly, retiring the wrapper. The `ARCHCORE_CWD` mechanism and the guard both stay harmless until then.

## Risks and Constraints

- **Opt-in surface.** Users who don't install the shell wrapper get a loud refusal at MCP start with the wrapper recipe printed inline. No more silent wrong-cwd operation. Mitigations:
  - Document the wrapper recipe prominently in `codex-local-plugin-testing.guide.md` and the README.
  - The guard itself prints the recipe — no docs lookup required to recover.
  - Consider in a future iteration: a `archcore shell-init <shell>` command that prints the wrapper for the user to paste into their rc.
- **Wrapper hygiene.** The wrapper sets `ARCHCORE_CWD=$PWD` at codex-launch time, not on every directory change inside Codex. If the user starts Codex in dir A and uses Codex to navigate to dir B, MCP still points at A. Acceptable because Codex itself treats the launch dir as the session's project root.
- **Custom env-var name choice.** `ARCHCORE_CWD` mirrors the existing `ARCHCORE_BIN` convention. Anyone who happens to have ARCHCORE_CWD set globally will override the launcher's cwd; documented as a feature.
- **Cross-host neutrality.** `.mcp.json` (Claude) and `.cursor-plugin/...` are untouched. Claude Code does not chdir before MCP spawn, so both the `ARCHCORE_CWD` block (no-op when the var is unset) and the guard (skipped when cwd is not in a plugin-cache path) have zero impact on other hosts.
- **Guard false positives.** Three converging signals must align (subcommand=mcp, cache-path cwd, all three plugin markers present). Plugin development from a non-cache checkout (e.g., this repo on disk) does not trigger the guard. The `ARCHCORE_ALLOW_PLUGIN_CWD=1` escape hatch covers the plugin-maintainer-runs-MCP-against-installed-copy edge case.

## Verification

A/B reproducer (after deploying to the plugin cache):

```sh
SOURCE=/Users/dev/Documents/archcore/plugin
CACHE=~/.codex/plugins/cache/archcore-personal/archcore/0.3.13

# Source-only marker the cache cannot have
printf -- '---\ntitle: marker\nstatus: draft\n---\n' > $SOURCE/.archcore/UNIQUE.md

INIT='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}'
NOTIF='{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}'
LIST='{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_documents","arguments":{}}}'

# WITH ARCHCORE_CWD: launcher chdir's, sees marker
( cd $CACHE && printf '%s\n%s\n%s\n' "$INIT" "$NOTIF" "$LIST" | \
  env -i HOME=$HOME PATH=$PATH USER=$USER LANG=$LANG TERM=$TERM TMPDIR=$TMPDIR ARCHCORE_CWD=$SOURCE \
  ./bin/archcore mcp 2>/dev/null | grep '"id":2' )

# WITHOUT ARCHCORE_CWD: guard refuses to start, prints wrapper recipe on stderr
( cd $CACHE && env -i HOME=$HOME PATH=$PATH USER=$USER LANG=$LANG TERM=$TERM TMPDIR=$TMPDIR \
  ./bin/archcore mcp 2>&1 1>/dev/null )

rm $SOURCE/.archcore/UNIQUE.md
```
