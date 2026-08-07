---
title: "Codex Local Plugin Testing Guide"
status: accepted
tags:
  - "codex"
  - "local-development"
  - "plugin"
  - "testing"
---

## Purpose

Verify that the Archcore plugin packages, installs, and runs correctly on Codex CLI from a local checkout — marketplace discovery, plugin installation, MCP registration, and skill and slash-command availability — before a release reaches users.

## Prerequisites

- Codex CLI with plugin support. Run `codex --version` first. If the plugin browser or local marketplaces behave differently from this guide, upgrade Codex before you debug plugin packaging.
- A clean Archcore plugin checkout with `jq` and `bats-core` available, and `shellcheck` optionally. If the bats helpers are missing, run `git submodule update --init`.
- The Codex package surfaces present and valid. The marketplace catalog stays at the **repo root** (`.agents/plugins/marketplace.json`); the plugin lives under **`plugins/archcore/`** (`plugins/archcore/.codex-plugin/plugin.json`, `plugins/archcore/.codex.mcp.json`, `plugins/archcore/hooks/codex.hooks.json`, and `plugins/archcore/skills/*/SKILL.md`). The catalog points `source.path` at `./plugins/archcore` — see `subdirectory-plugin-layout.adr` and issue #2.
- The Archcore CLI installed globally on PATH through the official installer at https://docs.archcore.ai/cli/install/ — `curl -fsSL https://archcore.ai/install.sh | bash` on macOS, Linux, and WSL, or `irm https://archcore.ai/install.ps1 | iex` on Windows PowerShell. Verify with `archcore --version`. The plugin ships no launcher; without the CLI, MCP startup fails at session start.

Notes on the test environment, non-normative:

- Codex reads repo marketplaces from `$REPO_ROOT/.agents/plugins/marketplace.json` and personal marketplaces from `~/.agents/plugins/marketplace.json`.
- Codex loads installed plugins from its cache under `~/.codex/plugins/cache/<marketplace>/<plugin>/<version>/`. Do not treat that cache as the source of truth.
- A project `.codex/config.toml` may register an `archcore` MCP server directly and hide a plugin packaging defect. Do not rely on this repo's project config when you test plugin-managed MCP.
- Run Codex from a directory outside the plugin source repo for end-to-end MCP validation. `command: "archcore"` resolves through PATH from the working directory the CLI is spawned in, so a neutral CWD avoids confusion with the project source.
- The official OpenAI Codex plugin docs are the authority for marketplace behavior. In the CLI, start `codex`, run `/plugins`, then browse by marketplace tab and install from the plugin details screen.

## Steps

1. Run the standard repository checks.

   ```bash
   make all
   ```

   Expected result: exit 0, having validated JSON, executable permissions, shell lint, unit tests, and structure tests. Fix any failure here before opening Codex — UI testing is noisy when the package is already structurally invalid.

2. Run the Codex-specific automated smoke tests.

   ```bash
   make test-codex-smoke
   ```

   Expected result: exit 0. These tests use an isolated temporary `HOME` and run the real discovery cycle — `codex plugin marketplace add "$REPO_ROOT"` accepts the repo marketplace, `codex plugin list` discovers `archcore@archcore-plugins` from the subdirectory, and `codex plugin add archcore@archcore-plugins` succeeds, which is the issue #2 regression. They also simulate an installed plugin cache to check skill loading and plugin-managed MCP registration. They are a fast regression check, not a replacement for an actual `/plugins` install.

3. If a smoke test fails, inspect the Codex package contract directly.

   ```bash
   jq . plugins/archcore/.codex-plugin/plugin.json
   jq . .agents/plugins/marketplace.json
   jq . plugins/archcore/.codex.mcp.json
   jq . plugins/archcore/hooks/codex.hooks.json
   ```

   Expected result: all four invariants below hold.

   - `plugins/archcore/.codex-plugin/plugin.json` points at `"./skills/"`, `"./hooks/codex.hooks.json"`, and `"./.codex.mcp.json"`. Those paths are plugin-root-relative, that is, relative to `plugins/archcore/`.
   - `.agents/plugins/marketplace.json` at the repo root holds one `archcore` entry with `source.source = "local"`, `source.path = "./plugins/archcore"`, plus `policy.installation`, `policy.authentication`, and `category`. Codex does not discover a plugin whose manifest sits at the marketplace root, even when `.codex-plugin/plugin.json` physically exists there — issue #2 and `subdirectory-plugin-layout.adr`.
   - `plugins/archcore/.codex.mcp.json` uses the Codex-documented direct server map: top-level `archcore.command = "archcore"` and `archcore.args = ["mcp"]`, and nothing else. No `mcpServers` wrapper, no `cwd: "."`, no `env_vars: ["ARCHCORE_CWD"]`. Codex resolves `archcore` from PATH, so the user's CLI install is the single source.
   - `plugins/archcore/hooks/codex.hooks.json` uses `${PLUGIN_ROOT}/bin/...` commands, which is Codex's canonical host-neutral variable. Do not use `${CLAUDE_PLUGIN_ROOT}`, which Codex provides only as a backward-compatible alias for old Claude plugins, and do not use `./bin/...`, which would resolve against the user's project CWD.

4. Register this checkout as a local repo marketplace.

   ```bash
   codex plugin marketplace add "$PWD"
   ```

   Expected result: the marketplace is recorded in `~/.codex/config.toml` and Codex reads `$PWD/.agents/plugins/marketplace.json`.

5. Close any running Codex TUI. The plugin browser is read at session startup, so a stale session will not show the marketplace you just added.

   Warning: step 6 edits a file in your home directory. Merge into it rather than replacing it, or you will drop the other plugins registered there.

6. For the most reliable manual local test, add an entry to `~/.agents/plugins/marketplace.json`.

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
           "path": "./Documents/archcore/plugin/plugins/archcore"
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

   Adjust `source.path` for the checkout you are testing. Keep it `./`-prefixed and relative to the home-directory marketplace root where possible, and point it at the `plugins/archcore` subdirectory that holds `.codex-plugin/plugin.json` — never at the repo root, which Codex silently skips (issue #2).

7. Install the plugin through the Codex CLI plugin browser.

   ```bash
   codex
   /plugins
   ```

   Switch to the relevant marketplace tab — `Archcore` for the repo marketplace, `Archcore Local` for the personal one — open `Archcore`, and select `Install plugin`.

   Expected result: the details screen shows `Installed` and offers `Uninstall plugin`.

8. Verify that Codex wrote the installed state and the cache.

   ```bash
   rg -n '\[plugins\."archcore@' ~/.codex/config.toml
   find ~/.codex/plugins/cache -maxdepth 5 -type d -path '*archcore*' -print
   ```

   Expected result: `~/.codex/config.toml` contains an enabled `archcore@<marketplace>` entry, and the cache holds a copied plugin bundle. Codex copies the resolved `source.path` directory — the `plugins/archcore/` subtree — so the cache root holds `.codex-plugin/plugin.json`, `commands/`, `skills/`, `.codex.mcp.json`, `hooks/`, and `bin/` directly. `bin/` contains only hook scripts and `lib/normalize-stdin.sh`, with no `archcore`, `archcore.cmd`, or `archcore.ps1` launcher.

9. Verify MCP registration from a neutral directory.

   ```bash
   tmpdir=$(mktemp -d)
   cd "$tmpdir"
   codex mcp list --json | jq '.[] | select(.name == "archcore")'
   ```

   Expected result: the JSON shows `command: "archcore"` and `args: ["mcp"]`, with no `cwd` field and no `env_vars` allowlist.

10. Verify end to end that the MCP operates on the project the user is in.

    ```bash
    tmpdir=$(mktemp -d)
    cd "$tmpdir"
    mkdir -p .archcore
    # touch a marker file the plugin cache cannot possibly have
    printf -- '---\ntitle: tmpdir marker\nstatus: draft\n---\n' > .archcore/tmpdir-marker.doc.md
    # start a fresh Codex session here, call mcp__archcore__list_documents,
    # confirm the marker is listed.
    ```

    Expected result: the marker appears, which means the MCP picked up the launch CWD correctly.

11. Verify slash commands and skills from a new Codex thread.

    Start a fresh Codex session after installation. Type `/archcore:` and then type `@`.

    Expected result: the slash commands appear — for example `/archcore:review`, `/archcore:plan`, `/archcore:document`, `/archcore:init` — and the matching Archcore skills appear under `@`. Run a small prompt such as `/archcore:review` to confirm execution.

12. After changing manifest, skill, MCP, or hook files, restart Codex.

    Codex loads the installed copy from its plugin cache rather than from the source tree in every code path. If the cache still holds the old copy, reopen `/plugins`, uninstall and reinstall the local plugin, or run the marketplace upgrade command for the registered marketplace, then restart Codex.

## Verification

- `make all` exits 0.
- `make test-codex-smoke` exits 0, or skips only because the Codex CLI is not installed.
- `/plugins` shows `Archcore` under the expected marketplace tab, and the details screen shows `Installed`.
- `~/.codex/config.toml` contains `[plugins."archcore@<marketplace>"]` with `enabled = true`.
- `~/.codex/plugins/cache/<marketplace>/archcore/<version>/` contains the plugin bundle.
- `codex mcp list --json` includes an enabled `archcore` server with `command: "archcore"` and `args: ["mcp"]`.
- From a directory outside the plugin source repo, `mcp__archcore__list_documents` returns documents from that directory's `.archcore/`.
- A new Codex thread discovers Archcore slash commands through `/archcore:` and Archcore skills through `@`, with no manual `codex mcp add`.
- Optional: with `codex features enable plugin_hooks`, `plugins/archcore/hooks/codex.hooks.json` loads the `SessionStart`, `PreToolUse`, and `PostToolUse` guardrails. Keep this a runtime smoke test, because `plugin_hooks` is `under development, false` by default in Codex 0.130.0.

## Common Issues

### Marketplace added but plugin is not visible

Confirm first that the catalog's `source.path` resolves to the `plugins/archcore` subdirectory rather than the repo root. A root `source.path` of `./` is the issue #2 failure mode, and Codex omits the plugin from `/plugins` without an error. Then close the current Codex TUI and start a new session, because the plugin browser groups entries by marketplace and a stale session may not show a newly added marketplace file. Clear the plugin search box and switch away from `OpenAI Curated` to the local marketplace tab.

### `codex plugin marketplace add` succeeded, but the plugin is not installed

Marketplace registration exposes a catalog; installation is a separate `/plugins` action. Open the plugin details and select `Install plugin`, then start a new thread before testing skill invocation.

### The local personal marketplace does not show `Archcore Local`

Validate `~/.agents/plugins/marketplace.json` with `jq`. Confirm that `source.path` starts with `./`, points at the `plugins/archcore` subdirectory holding `.codex-plugin/plugin.json` rather than the repo root, and that `policy.installation`, `policy.authentication`, and `category` are present. Codex skips an unresolvable plugin entry instead of failing the whole marketplace.

### The plugin appears under Available but not Installed

Open the plugin details and select `Install plugin`. `Space` toggles the enabled state for an installed plugin only; it does not install an available entry.

### MCP list shows `archcore`, but not from the plugin

Run `codex mcp list --json` from a neutral temporary directory. A project-level `.codex/config.toml` can register `archcore` directly and shadow a missing plugin-managed MCP entry. Inspect the command and confirm the entry came from the installed plugin bundle — the marketplace name will match.

### MCP `list_documents` returns the wrong project's documents

Symptom: from `mktemp -d`, after creating `.archcore/tmpdir-marker.doc.md`, the first `list_documents` call returns Archcore's own documents and the marker is absent. Cause: Codex spawned the MCP in a different working directory than expected. Check in this order:

1. Confirm you started Codex from the project directory. `archcore mcp` reads `.archcore/` relative to its own CWD, so starting Codex elsewhere is where it looks.
2. Check for a project-level `.codex/config.toml` registering `archcore` with an explicit `cwd`, which overrides the plugin-managed MCP's CWD. Remove it or align it with your intent.
3. Restart Codex from the correct directory. MCP servers are spawned at session start.

If the symptom persists, your Codex CLI version may be rebasing CWD silently; `codex-path-resolution.adr` holds the historical context for that bug class.

### MCP startup fails with `command not found`

`archcore` is not on PATH for the Codex process. Install it per https://docs.archcore.ai/cli/install/ and confirm with `archcore --version` from a regular terminal. If that works and Codex still cannot find it, the Codex process likely has a stripped PATH — check Codex's environment configuration, or launch `codex` from a shell that has `~/.local/bin`, or the installer's target directory, on PATH.

### Skill discovery works only after a restart

Expected. The official Codex flow requires a new thread after plugin installation. Close the existing session and start a new one before testing `@` skill discovery.

### Local edits are not picked up

Codex installs a copy into `~/.codex/plugins/cache/...`. Restart Codex after source edits. If the installed copy stays stale, uninstall and reinstall the plugin from `/plugins`, or run a marketplace upgrade for the registered marketplace, then restart.

### Hook guardrails do not fire

The plugin can package `plugins/archcore/hooks/codex.hooks.json`, but live hook execution depends on Codex's `plugin_hooks` feature flag. Run `codex features enable plugin_hooks` and retest with a fresh session. If the feature is unavailable in your Codex version — it is `under development, false` by default in Codex 0.130.0 — upgrade Codex or treat plugin hooks as best-effort until it stabilizes.

### `codex debug prompt-input` fails with session permission errors

Treat this as a local Codex session-file permission or sandbox problem rather than a plugin packaging failure. Use the `/plugins` details screen and a new interactive thread to validate skill discovery.

## References

- OpenAI Codex Plugins overview: https://developers.openai.com/codex/plugins
- OpenAI Build plugins guide: https://developers.openai.com/codex/plugins/build
- Archcore CLI install docs: https://docs.archcore.ai/cli/install/
- Upstream issue tracking `${PLUGIN_ROOT}` MCP substitution: https://github.com/openai/codex/issues/19582
