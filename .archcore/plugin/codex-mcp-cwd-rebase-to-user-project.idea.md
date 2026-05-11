---
title: "Codex MCP — User-Project CWD Without Shell-Resync"
status: draft
tags:
  - "codex"
  - "multi-host"
  - "plugin"
---

## Idea

Restore the archcore MCP server's working directory to the user's project root when invoked from Codex, **without** depending on the inherited `$PWD` env var (which POSIX shells overwrite at startup).

Codex spawns plugin MCP servers from the plugin install dir (via `.codex.mcp.json` `cwd: "."` rebase). The launcher itself is a `#!/bin/sh` script, so any `$PWD` value Codex inherits from the user's shell is **erased on launcher startup** — POSIX shells (sh, dash, bash --posix) resync `$PWD` to match `getcwd()` if the two diverge. That kills the obvious "follow `$PWD`" workaround.

Two viable paths that survive sh's PWD-resync:

1. **An explicit env var the shell does NOT resync** — e.g., `ARCHCORE_CWD` (or `CODEX_PROJECT_DIR`, if Codex ever exposes one). The launcher reads it before exec'ing the real CLI:

    ```sh
    if [ -n "${ARCHCORE_CWD:-}" ] && [ -d "$ARCHCORE_CWD" ]; then
      cd "$ARCHCORE_CWD" 2>/dev/null || true
    fi
    ```

    Sh's PWD-resync only touches `PWD` and `OLDPWD`. A custom name passes through untouched. The cost: someone (Codex, or the user) has to set it. Until Codex exposes a canonical name, the user opts in per-session.

2. **archcore CLI argument** — `archcore --project-root=PATH mcp`. Lets Codex (or a future plugin manifest version) pass the path explicitly. Doesn't require a custom env var, but does require Codex to thread an argument through MCP `args`. As of Codex 0.130.0 there is no env substitution in `args`, so this only helps if upstream adds it (openai/codex#19582).

## Value

Today, calling any `mcp__archcore__*` tool from Codex creates `.archcore/` documents in `~/.codex/plugins/cache/<marketplace>/archcore/<version>/.archcore/` instead of the user's project. Observed during a Codex `/archcore:bootstrap` session against `test_project/`: two seed docs landed in the plugin cache before the agent gave up and started a manual MCP rooted at the project. ~9m41s of confused wallclock.

Closing this gap turns Codex into a first-class Archcore host. Without the fix, the bundled-launcher value-prop of "no global install needed" silently degrades for Codex users — every MCP write pollutes the cache.

## Possible Implementation

1. **Add `ARCHCORE_CWD` support in `bin/archcore`** — guarded `cd "$ARCHCORE_CWD"` near the top of the launcher (after `SCRIPT_DIR=...` resolution), with `[ -d ]` and `cd ... || true` safety. Unit-tested via the existing Bats pattern (env var → fake `ARCHCORE_BIN` stub → assert stub's `pwd` output).
2. **Diagnostic mode for env discovery** — `archcore mcp --debug-env` that streams a single JSON-RPC notification listing all `getenv` keys the MCP server sees at startup. Run it once from a real Codex session to identify which env vars Codex actually injects (PLUGIN_ROOT, CODEX_HOME, CODEX_CWD, ...). If a Codex-provided user-project var exists, switch the launcher to read it first; fall back to `ARCHCORE_CWD`.
3. **Document the contract** in `codex-local-plugin-testing.guide.md` step 8 — add an assertion that an MCP `create_document` call from `mktemp -d` creates files under that tmpdir, **not** under `~/.codex/plugins/cache/...`. Pin the contract with a Bats E2E test if feasible.
4. **Upstream issue** — file or watch openai/codex#19582 for `${PLUGIN_ROOT}` MCP substitution. Once shipped, drop `cwd: "."`, use `${PLUGIN_ROOT}/bin/archcore` directly, and Codex inherits the user's CWD by default — both legs become unnecessary.
5. **One-time cleanup utility** — `archcore mcp cleanup-cache` that removes stray `.archcore/` directories that may have landed under `~/.codex/plugins/cache/<marketplace>/archcore/<version>/` before this fix shipped. Opt-in.

## Risks and Constraints

- **POSIX `$PWD`-resync is non-negotiable.** Verified across `/bin/sh`, `dash`, and `bash --posix` on macOS: each rewrites `$PWD` to match `getcwd()` on every shell startup when they disagree. Any fix that depends on the kernel-inherited `$PWD` is moot in a shell-script launcher.
- **`ARCHCORE_CWD` is opt-in.** Users who don't set it stay in the broken state. Mitigation: surface a clear "your MCP CWD looks like a plugin cache; set `ARCHCORE_CWD` to your project" warning when archcore CLI detects `getcwd()` matches a plugin-cache pattern (e.g., contains `/plugins/cache/` or `/plugin-install/`). Don't refuse to operate — too disruptive — but make the next step obvious.
- **Codex env vars may not exist for MCP spawns.** The existing `codex-path-resolution.adr` confirms `${PLUGIN_ROOT}` is injected for hooks but says nothing about MCP. Step 2 (diagnostic mode) is the way to find out. Do this before committing to `ARCHCORE_CWD` as the canonical name.
- **Cross-host concern.** Claude Code uses `${CLAUDE_PLUGIN_ROOT}/bin/archcore` with no chdir, so `$PWD == $(pwd)` and the issue doesn't surface. Cursor and other hosts may behave like Codex or like Claude — verify each before extending.
- **Naming.** If we name our env var `ARCHCORE_CWD`, that mirrors `ARCHCORE_BIN` already used by the launcher. Alternative `ARCHCORE_PROJECT_ROOT` is more explicit. Stick with `ARCHCORE_CWD` for brevity unless a survey of other plugins' conventions argues otherwise.
