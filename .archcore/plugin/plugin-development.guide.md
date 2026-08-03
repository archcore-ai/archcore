---
title: "Plugin Development Guide"
status: accepted
tags:
  - "development"
  - "plugin"
---

## Purpose

Set up a local development environment for the Archcore plugin, change a skill, hook, agent, or command wrapper, and verify the change on each supported host before opening a pull request.

## Prerequisites

- Claude Code, Cursor, Codex CLI, or GitHub Copilot CLI, installed with plugin support.
- Git.
- bats-core for tests. macOS: `brew install bats-core`.
- jq for JSON validation. macOS: `brew install jq`.
- ShellCheck, optional. macOS: `brew install shellcheck`.
- The Archcore CLI installed globally through the official installer at https://docs.archcore.ai/cli/install/ — `curl -fsSL https://archcore.ai/install.sh | bash` on macOS, Linux, and WSL, or `irm https://archcore.ai/install.ps1 | iex` on Windows PowerShell 5.1 or later. Verify with `archcore --version`.

The plugin bundles no launcher; it assumes the CLI is on PATH. MCP registers automatically for Claude Code through the plugin-root `.mcp.json`, and for Codex CLI through `.codex-plugin/plugin.json` pointing at the plugin-root `.codex.mcp.json`. Both name `archcore` as the command, and the host runtime resolves it through PATH.

Two hosts get no plugin-shipped MCP, deliberately, because each launches a plugin's MCP child outside the user's project, where a plugin-shipped server would read and write the wrong tree:

- **Cursor.** Copy `docs/cursor.mcp.example.json` into `~/.cursor/mcp.json` for user scope or `.cursor/mcp.json` for project scope. `cursor-mcp-architecture.adr` holds the three-layer rationale: Cursor 2.5 and later spawn plugin MCPs from the plugin install directory rather than the workspace, and its MCP stdio schema has no `cwd` field. The template passes `--project ${workspaceFolder}` in `args`, so the server resolves the workspace however Cursor invokes it.
- **GitHub Copilot CLI.** Run `archcore init --agent copilot --project "$PWD"` in the test project, with CLI v0.6.4 or later. That writes the workspace-root `.mcp.json` Copilot CLI actually reads. Per `copilot-mcp-architecture.adr`, Copilot launches a plugin's MCP child in the plugin install directory and passes it no project path (github/copilot-cli#4234), so documents would land in `~/.copilot/installed-plugins/` while every tool reported success.

For Codex, `codex plugin marketplace add /path/to/plugin` registers the marketplace, and the CLI loads enabled plugins from its installed plugin cache. Initialize a test project with `mcp__archcore__init_project` from a host session rather than an out-of-band CLI command; the plugin routes initialization through MCP.

## Steps

### 1. Clone the plugin repository

```bash
git clone https://github.com/archcore-ai/plugin.git
cd plugin
git checkout dev               # development happens on dev, main is synthesized
git submodule update --init    # pulls bats-support and bats-assert
```

The repository uses a `dev → main` split: every pull request lands on `dev`, and `.github/workflows/release.yml` synthesizes `main` from a tagged commit on `dev` with the dev-only artifacts stripped. `docs/release.md` holds the full blocklist and the release procedure.

### 2. Run the host with the plugin loaded locally

```bash
claude   --plugin-dir plugins/archcore    # Claude Code
cursor   --plugin-dir plugins/archcore    # Cursor
copilot  --plugin-dir plugins/archcore    # GitHub Copilot CLI
```

Codex has no `--plugin-dir`; use a local marketplace instead, with `codex plugin marketplace add /path/to/plugin`.

Expected result: the plugin loads without a marketplace installation, and `/reload-plugins` inside the session picks up later file changes.

On Copilot, `--plugin-dir` is not equivalent to an install for hook purposes. Which plugin-root variable a hook process receives under this flag, and whether it receives one at all, is undocumented — see step 4. If session start prints `plugin root unresolved`, that is what happened; run `make test-copilot-smoke` or a real install instead, and record the result per `host-probe-protocol.spec`.

### 3. Modify an existing skill

The plugin ships **7 skills** per `skill-surface-collapse.adr`: `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`. Each lives at `skills/<name>/SKILL.md`. An eighth top-level skill requires a new ADR — prefer adding flow logic under `skills/plan/references/` or `skills/decide/references/`.

1. Edit `skills/<name>/SKILL.md`. Required frontmatter: `name`, which must match the directory name, and `description`. Optional: `argument-hint`. No skill carries `disable-model-invocation`; all 7 auto-invoke.
2. Run `/reload-plugins`.
3. Invoke `/archcore:<name>`.

### 4. Add a slash-command wrapper for a new skill

Claude Code, Cursor, and Copilot surface skills directly in the `/` menu. Codex CLI does not — it discovers slash commands from `commands/<name>.md`. The plugin ships 7 wrappers, one per skill. If you add a new top-level skill, which itself requires a new ADR, add the matching wrapper:

```markdown
---
description: <one-line description, ideally matching the skill's first sentence>
---

# /archcore:my-skill

## Arguments

The user invoked this command with: $ARGUMENTS

## Instructions

Use the Archcore skill at `skills/my-skill/SKILL.md`.
```

A wrapper carries no workflow logic; behavior lives in the skill, which is the single source of truth. `test/structure/codex-plugin.bats` and `copilot-plugin.bats` enforce parity: every wrapper exists, carries `description:`, and references its matching `skills/<name>/SKILL.md`. Copilot loads the same wrappers, but only because `.plugin/plugin.json` names `./commands/` explicitly — that field has no default path there. A skill outranks a command of the same name on Copilot, so the wrappers are a fallback surface; the pointer still has to be present, and a structure test keeps it there.

### 5. Add or modify hooks

Edit `hooks/hooks.json` for Claude Code, `hooks/cursor.hooks.json` for Cursor, `hooks/codex.hooks.json` for Codex CLI, or `hooks/copilot.hooks.json` for GitHub Copilot CLI. Enroll every hooks config in `test/structure/host-coverage-matrix.bats` and in the resolution table in `hooks.bats`; both carry enrollment guards that fail until you do.

A hook script goes in `bin/` and must:

- start with `#!/bin/sh`;
- be executable, via `chmod +x`;
- source `bin/lib/normalize-stdin.sh` when it reads hook stdin;
- carry `# shellcheck source=lib/normalize-stdin.sh` before that source line;
- invoke the CLI directly as `archcore`, resolved through PATH;
- guard against a plugin-install launch directory when it reads `.archcore/` or emits user-visible context, by exiting silently when the working directory contains — or sits beneath — a `.cursor-plugin/`, `.claude-plugin/`, `.codex-plugin/`, or `.plugin/` manifest. `bin/session-start` is the canonical pattern, and `cursor-mcp-architecture.adr` holds the rationale.

Three hosts substitute a single canonical plugin-root variable; Copilot is the exception.

- `${CLAUDE_PLUGIN_ROOT}` — Claude Code's native injection, in `hooks/hooks.json`.
- `${CURSOR_PLUGIN_ROOT}` — Cursor's native injection, in `hooks/cursor.hooks.json`.
- `${PLUGIN_ROOT}` — Codex CLI's canonical host-neutral variable, in `hooks/codex.hooks.json`. Codex's hooks engine (`codex-rs/hooks/src/engine/discovery.rs`) injects `PLUGIN_ROOT` as the canonical name and `CLAUDE_PLUGIN_ROOT` only as a backward-compatible alias for porting old Claude plugins, so do not use the alias in a Codex-native config. `CODEX_PLUGIN_ROOT` does not exist.
- Copilot has no confirmed variable, so `hooks/copilot.hooks.json` probes three. `COPILOT_PLUGIN_ROOT`, which this adapter relied on until 2026-07-27, appears in no GitHub documentation; the only documented spelling is `${PLUGIN_ROOT}`, and for writable state the documented pair is `${COPILOT_PLUGIN_DATA}` and `${CLAUDE_PLUGIN_DATA}`. Each command tests `$COPILOT_PLUGIN_ROOT`, `$PLUGIN_ROOT`, and `$CLAUDE_PLUGIN_ROOT` in turn with `-x`, execs the first that holds the script, and otherwise warns on stderr and exits 0. The variables are written unbraced so the command behaves identically whether Copilot expands them or `sh` does. Whichever variable is real, `bin/detect-host` cannot key on it — it exists only inside hook processes, so a Copilot session resolves to `__UNKNOWN__` there.

If you edit those commands, note that `test/structure/copilot-plugin.bats` **runs** them under `env -u` rather than comparing strings: exit 0 with a warning when nothing resolves, each candidate sufficient alone, a dead candidate skipped rather than fatal. The old string-equality assertions matched the broken command exactly and shipped it — see `copilot-adapter-design.adr` and `cli-integration-tests.rule`.

Copilot's config differs in shape, not only in names: entries use `bash` rather than `command` and `timeoutSec` rather than `timeout`, are flat objects rather than nested groups, carry `cwd: "."` so the hook runs from the user's project, and its `postToolUse` entries carry no matcher at all, because the scripts self-filter. A test written by copying another host's and swapping the filename iterates an empty set and reports `ok`.

Two Copilot hook semantics differ from every other host, and both are load-bearing. First, every non-zero exit denies: `exit 2` is a deny whose stdout JSON merges with the deny decision, and any other non-zero exit denies as `Denied by preToolUse hook (hook errored)` (hooks-reference, re-read 2026-07-27). Guard scripts still write `{"permissionDecision":"deny","permissionDecisionReason":…}` to stdout with exit 0, because that is how a deny carries its reason — not because exit 2 fails to block. The corollary is that a guard which cannot *start* denies the write too, which is why hook bootstrap gets the candidate chain above. Second, a `preToolUse` timeout fails open, which makes guard latency a correctness concern; `test/unit/hook-latency.bats` keeps both PreToolUse guards far inside the 1-second budget for that reason.

Hooks are narrower than plugins on this host: hooks-reference names exactly two supported surfaces, Copilot CLI and Copilot cloud agent. VS Code agent mode is not one of them, even though Copilot Chat ships its own CLI binary under the extension's `globalStorage` and hook machinery has been observed firing there.

Plugin-shipped Codex hooks require `codex features enable plugin_hooks` before they fire; the `plugin_hooks` feature is `under development, false` by default in Codex 0.130.0. `codex-path-resolution.adr` holds the full mechanism.

### 6. Modify an agent

1. Edit `agents/archcore-assistant.md` or `agents/archcore-auditor.md`. The frontmatter carries `name`, `description`, `model`, `maxTurns`, and `tools`. Keep the auditor read-only, holding only `list_documents`, `get_document`, and `list_relations`. List every MCP naming: `mcp__archcore__*`, `mcp__plugin_archcore_archcore__*`, and Copilot's flat `archcore-<tool>`.
2. Propagate the change to `agents/<name>.toml` for Codex, keeping the TOML and MD `developer_instructions` content identical.
3. Propagate it to `copilot-agents/<name>.agent.md` for Copilot as a byte-identical copy, checked with `cmp`, because Copilot's loader accepts only the `*.agent.md` extension.

Keep the Copilot copy in `copilot-agents/`, never beside the original: `.agent.md` matches the `*.md` glob Claude Code and Cursor use, so a sibling copy would hand both hosts two files declaring the same `name:`. `test/structure/agents.bats` checks both variants.

### 7. Run the tests

```bash
make verify    # full check: JSON + permissions + shellcheck + tests
```

Or run individual checks:

```bash
make test               # all bats tests
make test-unit          # unit tests (bin script logic)
make test-structure     # structure tests (configs, frontmatter)
make test-codex-smoke   # install smoke, skips without the codex CLI
make test-copilot-smoke # install smoke, skips without the copilot CLI
make lint               # shellcheck
make check-json         # JSON validity
make check-perms        # executable permissions
```

`make verify` is the canonical integrity check; there is no `/archcore:verify` skill, which `skill-surface-collapse.adr` removed.

### 8. Test the components manually

- Skills: discuss a relevant topic and confirm the host activates the skill.
- Commands: run each `/archcore:<name>` on every host you can reach. Codex pulls them from `commands/`; Claude Code, Cursor, and Copilot from `skills/`.
- Agent: invoke it on a multi-document task.
- Hooks: trigger a Write or Edit on `.archcore/` and confirm the pre-mutation guard blocks it.
- MCP availability: confirm `archcore --version` works.
- Codex: from a directory outside the plugin source repo, such as `cd $(mktemp -d)`, call any `mcp__archcore__*` tool and confirm the MCP starts.
- Cursor: after copying `docs/cursor.mcp.example.json` into `.cursor/mcp.json`, open an empty project. Expected result: `list_documents` returns empty rather than the plugin's own dev documents. If it returns dev documents, the plugin-install-dir guards regressed; file an issue against this repo and `archcore-ai/cli`.
- Copilot: `copilot mcp list` will show a plugin-contributed `archcore` server — settled 2026-08-03, because the host auto-discovers a plugin-root `.mcp.json` regardless of the manifest. Confirm instead that this server **fails to start** with the plugin-cache guard error on CLI v0.6.7 or later, and that the project-wired server from the repo-root `.mcp.json` is the one serving tools. A plugin server that starts and serves is the regression.
- Copilot: session start must not print `archcore: plugin root unresolved`. If it does, no candidate variable was injected and every guard is silently disabled for that session — capture which load path produced it, because that is the open question in `copilot-adapter-design.adr`.
- Integrity: `make verify`.

For the questions no manual checklist can settle — whether a deny is honored or merely displayed, whether pre-mutation hooks fire on delegated calls, which plugin-root variable Copilot injects — follow `host-probe-protocol.spec` and record the result.

## Verification

- `make verify` exits 0 with `All checks passed`.
- `/reload-plugins` reports 7 skills, 2 agents, and 6 hook entries.
- `/help` lists all 7 `/archcore:*` commands.
- `/agents` lists `archcore-assistant` and `archcore-auditor`.
- A Write or Edit against a `.archcore/` markdown file is blocked with a redirect message.
- `archcore --version` prints a version.
- MCP tools work from any project directory on every host.

## Common Issues

### Plugin not loading

- Confirm the manifest for your host exists and holds valid JSON: `.claude-plugin/plugin.json` for Claude Code, `.cursor-plugin/plugin.json` for Cursor, `.codex-plugin/plugin.json` for Codex CLI, `.plugin/plugin.json` for GitHub Copilot CLI.
- Confirm `skills/`, `agents/`, `copilot-agents/`, `hooks/`, and `commands/` sit at the plugin root.
- Run `claude --debug` to see plugin loading details.

### Skill not activating

- Check the `description` field in the `SKILL.md` frontmatter; it determines when the host activates the skill.
- Confirm `name` matches the directory name.
- Run `/reload-plugins` after a change.

### `/archcore:<name>` missing from the Codex `/` menu

- Confirm `commands/<name>.md` exists and carries `description:` frontmatter.
- Confirm it references `skills/<name>/SKILL.md`; the bats parity test enforces this.
- Run `make test-structure`; `codex-plugin.bats` flags a missing or malformed wrapper.
- Restart Codex after adding a wrapper. The marketplace cache is read once at session start.

### Agents missing in Copilot

- Confirm the file is `copilot-agents/<name>.agent.md`. Copilot derives the agent id from the filename and loads only that extension, so a plain `<name>.md` is invisible to it, which is exactly why the copies exist.
- Confirm `.plugin/plugin.json` carries `"agents": "./copilot-agents/"`. The field defaults to `agents/`, so an omitted pointer looks silently in the directory that holds no `*.agent.md` file.

### Hook not firing

- Confirm the `bin/` scripts are executable: `chmod +x bin/<name>`.
- Check the shebang line reads `#!/bin/sh`.
- Confirm the hook JSON structure matches the expected format for that host.
- Test the script manually: `echo '{"tool_name":"Write","tool_input":{"file_path":".archcore/test.adr.md"}}' | bin/check-archcore-write`.
- On Codex, hooks require `codex features enable plugin_hooks`. Without the flag, Codex does not run plugin-shipped hooks.
- On Copilot, confirm the entry uses `bash` rather than `command` and `timeoutSec` rather than `timeout`. A config written in Claude's shape loads without error and does nothing.

### `/bin/sh: /bin/<script>: No such file or directory` on GitHub Copilot CLI

This is the historical symptom of an unresolved plugin root: `"${COPILOT_PLUGIN_ROOT}"/bin/session-start` collapses to `/bin/session-start` when the variable is empty. A current plugin prints `archcore: plugin root unresolved` and exits 0, so this older message means the session runs a build from before 2026-07-27 — reinstall.

Either way the cause is the same, and on `preToolUse` the consequence is severe: a failed exec is a non-zero exit, which Copilot reads as a deny, so every `create|edit|str_replace_editor|apply_patch` call is refused with `Denied by preToolUse hook (hook errored)`. Check in this order:

1. Which surface? Hooks are documented for Copilot CLI and cloud agent only. A VS Code or Copilot Chat session is neither, even though it ships its own CLI binary, and nothing about variable injection is guaranteed there.
2. Which load path? `--plugin-dir` is not `copilot plugin install`. Confirm `copilot plugin list` shows `archcore`.
3. Which config? Copilot reads hooks from six places besides a plugin: policy `.d` files, `.github/hooks/*.json`, `~/.copilot/hooks/`, `.github/copilot/settings.json`, `~/.copilot/settings.json`, and the shared subset of a repo-level `.claude/settings.json`. A plugin's hooks config copied into any of those gets no plugin root by design.

### Tests failing

- Run `git submodule update --init` when the bats helpers are missing.
- On macOS, the test suite provides a `timeout` shim automatically.
- Confirm the `archcore` CLI is on PATH with `archcore --version`.

### MCP server not connecting on Claude Code or Codex CLI

The plugin ships `.mcp.json` for Claude Code and `.codex.mcp.json` for Codex CLI. Diagnose in this order:

1. Plugin loaded? `/plugin` on Claude Code, or `codex mcp list --json` on Codex CLI, should show `archcore`. If `.mcp.json`, `.codex.mcp.json`, or the Codex `mcpServers` pointer was modified or removed, the server will not register; restore it from git.
2. CLI available? Run `archcore --version` from a terminal. Expected result: it prints a version. If it is not found, install through the official installer at https://docs.archcore.ai/cli/install/. If permission is denied, confirm the CLI binary is executable.
3. Session lifecycle. Claude Code registers MCP servers at session start, so installing the CLI mid-session does not reconnect the server. Restart the host after a fresh install.
4. Duplicate suppression. If `/plugin` shows `Errors (1)` with an `archcore` MCP message, a user- or project-registered `archcore` shares the same command. This is benign, because the resolved binary is the same; remove the redundant registration to silence the warning.

### No MCP tools at all on GitHub Copilot CLI

Expected until the project is wired, because the plugin ships no MCP server for Copilot. Run `archcore init --agent copilot --project "$PWD"` on CLI v0.6.4 or later, which writes the workspace-root `.mcp.json`, then restart the session. Copilot discovers `.mcp.json` by walking from the working directory up to the git root, so a repo-root file covers monorepo layouts.

If tools appear but documents land somewhere unexpected, check where: github/copilot-cli#4234 puts a plugin-contributed MCP child in `~/.copilot/installed-plugins/`. A project-registered server does not have that problem, because the host launches it from the project.

### MCP server not connecting on Cursor

Cursor uses a user-installed MCP rather than a plugin-shipped one, deliberately — see `cursor-mcp-architecture.adr`. Copy `docs/cursor.mcp.example.json` into `~/.cursor/mcp.json` for user scope, or `.cursor/mcp.json` for project scope. The file ships with the right shape:

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

- `--project ${workspaceFolder}` is mandatory. Cursor's MCP stdio schema has no `cwd` field, and without `--project` the server falls back to `os.Getwd()`, which is unreliable for a plugin-launched process.
- Do not add a `cwd` field. Cursor ignores it silently.
- Do not copy the template to the plugin root. A `cursor.mcp.json` there would let Cursor's plugin-MCP auto-detection register the server with the working directory set to the plugin install directory, leaking bundled state instead of the user's workspace.

### "Plugin MCP Servers → archcore" appears in Cursor settings with stale documents

The plugin ships no Cursor plugin MCP. If Cursor's "Plugin MCP Servers" section shows `archcore`, then either an older plugin version with a plugin-root `cursor.mcp.json` is still cached, or a regression introduced a plugin-root MCP file.

1. Uninstall the plugin from Cursor.
2. Remove `~/.cursor/plugins/cache/archcore-plugins/`, or the relevant cache subtree.
3. Reinstall the plugin from `main`, which the release workflow synthesizes without a plugin-root `cursor.mcp.json`.
4. Confirm `test/structure/cursor-plugin.bats` passes; it asserts that no legacy `cursor.mcp.json` sits at the plugin root.

If the symptom persists after a fresh `main` install, file an issue: the layered defense in `cursor-mcp-architecture.adr` has a gap.
