---
title: "Codex Local Plugin Testing Guide"
status: accepted
tags:
  - "codex"
  - "local-development"
  - "plugin"
  - "testing"
---

## Prerequisites

- Codex CLI with plugin support. Check `codex --version`; if the plugin browser or local marketplaces behave differently from this guide, update Codex before debugging plugin packaging.
- A clean Archcore plugin checkout with `jq`, `bats-core`, and optional `shellcheck` available. Initialize test submodules with `git submodule update --init` if bats helpers are missing.
- The Codex package surfaces must exist and be valid: `.codex-plugin/plugin.json`, `.agents/plugins/marketplace.json`, `.codex.mcp.json`, `hooks/codex.hooks.json`, and `skills/*/SKILL.md`.
- **Archcore CLI installed globally on PATH** via the official installer at https://docs.archcore.ai/cli/install/ — `curl -fsSL https://archcore.ai/install.sh | bash` (macOS/Linux/WSL) or `irm https://archcore.ai/install.ps1 | iex` (Windows PowerShell). Verify with `archcore --version`. The plugin no longer bundles a launcher; if the CLI is missing, MCP startup fails at session start.
- Official OpenAI Codex plugin docs are the authority for marketplace behavior:
  - CLI plugin directory: start `codex`, run `/plugins`, then browse by marketplace tab and install from the plugin details screen.
  - Local marketplaces: Codex reads repo marketplaces from `$REPO_ROOT/.agents/plugins/marketplace.json` and personal marketplaces from `~/.agents/plugins/marketplace.json`.
  - Installed plugins are loaded from the Codex plugin cache under `~/.codex/plugins/cache/<marketplace>/<plugin>/<version>/`; do not edit the cache as the source of truth.
- When testing plugin-managed MCP, avoid relying only on this repo's project config. A project `.codex/config.toml` may register an `archcore` MCP server directly and hide plugin packaging defects.
- For end-to-end MCP validation, run Codex from a directory **outside the plugin source repo** (e.g., `cd $(mktemp -d)`). With the launcher gone, `command: "archcore"` resolves via PATH from the working directory the CLI is spawned in — neutral CWD avoids accidental project-source confusion.

## Steps

1. Run the standard repository checks first.

   ```bash
   make all
   ```

   This validates JSON, executable permissions, shell lint, unit tests, and structure tests. Fix failures here before opening Codex; UI testing is noisy when the package is already structurally invalid.

2. Run the Codex-specific automated smoke tests.

   ```bash
   make test-codex-smoke
   ```

   These tests use an isolated temporary `HOME` and verify that `codex plugin marketplace add "$PLUGIN_ROOT"` accepts the repo marketplace. They also simulate an installed plugin cache to check skill loading and plugin-managed MCP registration. This is a fast regression check, not a replacement for an actual `/plugins` install.

3. Inspect the Codex package contract directly when a smoke test fails.

   ```bash
   jq . .codex-plugin/plugin.json
   jq . .agents/plugins/marketplace.json
   jq . .codex.mcp.json
   jq . hooks/codex.hooks.json
   ```

   Confirm these invariants:
   - `.codex-plugin/plugin.json` points to `"./skills/"`, `"./hooks/codex.hooks.json"`, and `"./.codex.mcp.json"`.
   - `.agents/plugins/marketplace.json` has one `archcore` entry, `source.source = "local"`, `source.path = "./"`, `policy.installation`, `policy.authentication`, and `category`.
   - `.codex.mcp.json` has `command: "archcore"` and `args: ["mcp"]` — nothing else. No `cwd: "."`, no `env_vars: ["ARCHCORE_CWD"]`. With the launcher removed, Codex resolves `archcore` from PATH directly; the user's CLI install is the single source.
   - `hooks/codex.hooks.json` uses `${PLUGIN_ROOT}/bin/...` commands (Codex's canonical, host-neutral env var). Do NOT use `${CLAUDE_PLUGIN_ROOT}` (Codex provides it only as a backward-compat alias for old Claude plugins) or `./bin/...` (would resolve against the user's project CWD).

4. Register this checkout as a local repo marketplace.

   ```bash
   codex plugin marketplace add "$PWD"
   ```

   This records the marketplace in `~/.codex/config.toml` and makes Codex read `$PWD/.agents/plugins/marketplace.json`. Close any already-running Codex TUI after changing marketplaces; the plugin browser is read on session startup.

5. For the most reliable manual local test, expose the checkout through a personal marketplace.

   Create or merge an entry into `~/.agents/plugins/marketplace.json`. Do not overwrite an existing personal marketplace list without preserving its other plugins.

   ```json
   {
     "name": "archcore-personal",
     "interface": {
       "displayName": "Archcore Local"
     },
     "plugins": [
       {
         "name": "archcore",
         "source": {
           "source": "local",
           "path": "./Documents/archcore/plugin"
         },
         "policy": {
           "installation": "AVAILABLE",
           "authentication": "ON_INSTALL"
         },
         "category": "Coding"
       }
     ]
   }
   ```

   Adjust `source.path` for the checkout path being tested. For a personal marketplace, keep it `./`-prefixed and relative to the home-directory marketplace root when possible.

6. Install the plugin through the Codex CLI plugin browser.

   ```bash
   codex
   /plugins
   ```

   In the browser, switch to the relevant marketplace tab (`Archcore` for the repo marketplace or `Archcore Local` for the personal marketplace), open `Archcore`, and select `Install plugin`. If the plugin is already installed, the details screen should show `Installed` and offer `Uninstall plugin`.

7. Verify Codex wrote the installed state and cache.

   ```bash
   rg -n '\[plugins\."archcore@' ~/.codex/config.toml
   find ~/.codex/plugins/cache -maxdepth 5 -type d -path '*archcore*' -print
   ```

   Expected result: `~/.codex/config.toml` contains an enabled `archcore@<marketplace>` entry, and the cache contains a copied plugin bundle with `.codex-plugin/plugin.json`, `commands/`, `skills/`, `.codex.mcp.json`, `hooks/`, and `bin/`. Note that `bin/` only contains hook scripts and `lib/normalize-stdin.sh` — no `archcore`/`archcore.cmd`/`archcore.ps1` launcher.

8. Verify MCP registration from a neutral directory.

   ```bash
   tmpdir=$(mktemp -d)
   cd "$tmpdir"
   codex mcp list --json | jq '.[] | select(.name == "archcore")'
   ```

   The JSON should show `command: "archcore"` and `args: ["mcp"]`. There is no `cwd` field and no `env_vars` allowlist — Codex resolves the binary via PATH from the directory it was launched in.

   Critical end-to-end check — verify the MCP operates on the project the user is in, not somewhere else:

   ```bash
   tmpdir=$(mktemp -d)
   cd "$tmpdir"
   mkdir -p .archcore
   # touch a marker file the plugin cache cannot possibly have
   printf -- '---\ntitle: tmpdir marker\nstatus: draft\n---\n' > .archcore/tmpdir-marker.doc.md
   # start a fresh Codex session here, call mcp__archcore__list_documents,
   # confirm the marker is listed.
   ```

   If the marker appears, the MCP picked up the launch CWD correctly. If `list_documents` returns archcore's own docs (`actualize-implementation.plan.md`, etc.) instead of the marker, your Codex CLI version may be silently rebasing CWD — see `codex-path-resolution.adr.md` for the historical context behind that bug class.

   If the MCP fails to start with `command not found`, `archcore` is not on PATH for the Codex process. Re-run the installer (https://docs.archcore.ai/cli/install/) and start a new Codex session.

9. Verify slash commands and skills from a new Codex thread.

   Start a fresh Codex session after installation. Type `/archcore:` and confirm slash commands appear, for example `/archcore:audit`, `/archcore:plan`, `/archcore:context`, and `/archcore:capture`. Type `@` and confirm the matching Archcore skills also appear. Then run a small prompt such as:

   ```text
   /archcore:context
   ```

   If the command or skill list is missing, return to `/plugins`, confirm `Archcore` is installed and enabled, close Codex, and start a new thread.

10. Refresh after local source changes.

    Codex loads the installed copy from its plugin cache, not directly from the source tree in every code path. After changing manifest, skill, MCP, or hook files, restart Codex. If the cache still contains the old copy, reopen `/plugins` and uninstall/install the local plugin, or run the marketplace upgrade command for the registered marketplace and restart Codex.

## Verification

- `make all` exits 0.
- `make test-codex-smoke` exits 0, or skips only when Codex CLI is not installed.
- `/plugins` shows `Archcore` under the expected marketplace tab and the details screen shows `Installed`.
- `~/.codex/config.toml` contains `[plugins."archcore@<marketplace>"]` with `enabled = true`.
- `~/.codex/plugins/cache/<marketplace>/archcore/<version>/` contains the plugin bundle.
- `codex mcp list --json` includes an enabled `archcore` server with `command: "archcore"`, `args: ["mcp"]`. From a directory outside the plugin source repo, calling `mcp__archcore__list_documents` returns docs from THAT directory's `.archcore/`.
- A new Codex thread can discover Archcore slash commands via `/archcore:` and Archcore skills via `@`, without manual `codex mcp add`.
- Optional hook verification: with `codex features enable plugin_hooks`, `hooks/codex.hooks.json` should load `SessionStart`, `PreToolUse`, and `PostToolUse` guardrails. Keep this as a runtime smoke test because the `plugin_hooks` feature is `under development, false` by default in Codex 0.130.0.

## Common Issues

### Marketplace added but plugin is not visible

Close the current Codex TUI and start a new session. The plugin browser groups entries by marketplace and stale sessions may not show newly added marketplace files. Also clear the plugin search box and switch away from `OpenAI Curated` to the local marketplace tab.

### `codex plugin marketplace add` succeeded, but plugin is not installed

Marketplace registration exposes a catalog; installation is a separate `/plugins` action. Open the plugin details and select `Install plugin`. After installation, start a new thread before testing skill invocation.

### Local personal marketplace does not show `Archcore Local`

Validate `~/.agents/plugins/marketplace.json` with `jq`. Confirm `source.path` starts with `./`, points to a directory that contains `.codex-plugin/plugin.json`, and has `policy.installation`, `policy.authentication`, and `category`. Codex skips an unresolvable plugin entry instead of failing the whole marketplace.

### Plugin appears under Available but not Installed

Open the plugin details and select `Install plugin`. Pressing `Space` toggles enabled state only for installed plugins; it does not install an available entry.

### MCP list shows `archcore`, but not from the plugin

Run `codex mcp list --json` from a neutral temporary directory. Project-level `.codex/config.toml` can register `archcore` directly and shadow a missing plugin-managed MCP entry. For plugin-managed MCP validation, inspect the command and confirm the entry came from the installed plugin bundle (the marketplace name will match).

### MCP `list_documents` returns the wrong project's docs

Symptom: from `mktemp -d`, after creating `.archcore/tmpdir-marker.doc.md`, the first `list_documents` call returns archcore's own docs and the marker is absent. Cause: Codex spawned the MCP in a different working directory than expected. Check, in order:

1. Did you start Codex from the project directory? `archcore mcp` reads `.archcore/` relative to its own CWD; if you started Codex elsewhere, that's where it looks. Always `cd` into the project first.
2. Is there a project-level `.codex/config.toml` registering `archcore` with an explicit `cwd`? That would override the plugin-managed MCP's CWD. Remove it or align it to your intent.
3. Restart Codex from the correct directory; MCP servers are spawned at session start.

### MCP startup fails with `command not found` or `archcore: not found`

`archcore` is not on PATH for the Codex process. Install per https://docs.archcore.ai/cli/install/ and confirm with `archcore --version` from a regular terminal. If verify works but Codex still can't find it, the Codex process likely has a stripped PATH — check Codex's environment configuration or invoke `codex` from a shell that has `~/.local/bin` (or the installer's target directory) on PATH.

### Skill discovery works only after restart

Expected. The official Codex flow requires using a new thread after plugin installation. Close the existing session and start a new one before testing `@` skill discovery.

### Local edits are not picked up

Codex installs a copy into `~/.codex/plugins/cache/...`. Restart Codex after source edits. If the installed copy remains stale, uninstall and reinstall the plugin from `/plugins`, or run a marketplace upgrade for the registered marketplace and restart.

### Hook guardrails do not fire

The plugin can package `hooks/codex.hooks.json`, but live hook execution depends on Codex's `plugin_hooks` feature flag. Run `codex features enable plugin_hooks` and retest with a fresh session. If the feature is unavailable in your Codex version (it's `under development, false` by default in Codex 0.130.0), upgrade Codex or treat plugin hooks as best-effort until it stabilizes.

### `codex debug prompt-input` fails with session permission errors

Treat this as a local Codex session-file permission or sandbox problem, not necessarily a plugin packaging failure. Use the `/plugins` details screen and a new interactive thread to validate skill discovery.

## References

- OpenAI Codex Plugins overview: https://developers.openai.com/codex/plugins
- OpenAI Build plugins guide: https://developers.openai.com/codex/plugins/build
- Archcore CLI install docs: https://docs.archcore.ai/cli/install/
- Codex MCP and Hooks Path Resolution ADR: `.archcore/plugin/codex-path-resolution.adr.md` (canonical reference for `cwd` rebase and env_vars passthrough — historical context now that the launcher is gone)
- Codex MCP CWD idea (rejected/historical): `.archcore/plugin/codex-mcp-cwd-rebase-to-user-project.idea.md`
- Upstream issue tracking `${PLUGIN_ROOT}` MCP substitution: https://github.com/openai/codex/issues/19582
- Related Archcore guide: `.archcore/plugin/plugin-testing.guide.md`
- Related Archcore spec: `.archcore/plugin/multi-host-compatibility-layer.spec.md`
- Bundled launcher removal: `.archcore/plugin/remove-bundled-launcher-global-cli.idea.md`
