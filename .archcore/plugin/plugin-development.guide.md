---
title: "Plugin Development Guide"
status: accepted
tags:
  - "development"
  - "plugin"
---

## Prerequisites

- Claude Code, Cursor, Codex CLI, or GitHub Copilot CLI installed with plugin support
- Git for version control
- bats-core for tests (`brew install bats-core` on macOS)
- jq for JSON validation (`brew install jq`)
- ShellCheck (optional, `brew install shellcheck`)
- **Archcore CLI** installed globally via the official installer at https://docs.archcore.ai/cli/install/ — `curl -fsSL https://archcore.ai/install.sh | bash` (macOS/Linux/WSL) or `irm https://archcore.ai/install.ps1 | iex` (Windows PowerShell 5.1+). Verify with `archcore --version`.

That's it. The plugin does not bundle a launcher — it assumes users have the Archcore CLI installed globally on PATH. MCP is registered automatically for Claude Code via plugin-root `.mcp.json`, and for Codex CLI via `.codex-plugin/plugin.json` pointing at plugin-root `.codex.mcp.json`. Both simply name `archcore` as the command — host runtimes resolve it via PATH.

**Two hosts get no plugin-shipped MCP, deliberately.** Cursor and GitHub Copilot CLI each launch a plugin's MCP child outside the user's project, so a plugin-shipped server would read and write the wrong tree.

For Cursor development, register MCP externally by copying `docs/cursor.mcp.example.json` into `~/.cursor/mcp.json` (user-scoped) or `.cursor/mcp.json` (project-scoped). See `cursor-mcp-architecture.adr.md` for the three-layer rationale (Cursor 2.5+ spawns plugin-MCPs from the plugin install dir rather than the workspace, and its MCP stdio schema has no `cwd` field). The template passes `--project ${workspaceFolder}` in `args` so the server always resolves the workspace, regardless of how Cursor invokes it.

For Copilot development, run `archcore init --agent copilot --project "$PWD"` in the test project (CLI ≥ v0.6.4). That writes the workspace-root `.mcp.json` Copilot CLI actually reads. See `copilot-mcp-architecture.adr.md`: Copilot launches a plugin's MCP child in the plugin install directory and passes it no project path at all (github/copilot-cli#4234), so documents would land in `~/.copilot/installed-plugins/` while every tool reported success.

For Codex development, `codex plugin marketplace add /path/to/plugin` registers the marketplace. The current CLI loads enabled plugins from its installed plugin cache; run `make test-codex-smoke` for the local installed-cache smoke that verifies skill discovery and plugin-managed MCP. `make test-copilot-smoke` is the equivalent for Copilot.

Initialize a project for testing with `mcp__archcore__init_project` (via any host session) rather than an out-of-band CLI command; the plugin routes initialization through MCP.

## Steps

### 1. Clone the plugin repository

```bash
git clone https://github.com/archcore-ai/plugin.git
cd plugin
git checkout dev               # development happens on dev, main is synthesized
git submodule update --init    # pulls bats-support and bats-assert
```

The plugin uses a `dev → main` split: all PRs land on `dev`. The `main` branch is synthesized by `.github/workflows/release.yml` from a tagged commit on `dev`, with dev-only artifacts stripped (`.archcore/`, `reference-materials/`, `test/`, `Makefile`, `.github/`, etc.). See `docs/release.md` for the full blocklist and release procedure.

### 2. Run the host with the plugin loaded locally

```bash
claude   --plugin-dir plugins/archcore    # Claude Code
cursor   --plugin-dir plugins/archcore    # Cursor
copilot  --plugin-dir plugins/archcore    # GitHub Copilot CLI
```

Codex has no `--plugin-dir`; use a local marketplace (`codex plugin marketplace add /path/to/plugin`).

This loads the plugin without requiring marketplace installation. Changes to plugin files are picked up after running `/reload-plugins` inside the session.

**On Copilot, `--plugin-dir` is not equivalent to an install for hook purposes.** Which plugin-root variable a hook process receives, and whether it receives one at all under this flag rather than `copilot plugin install`, is undocumented — see step 4. If session start prints `plugin root unresolved`, that is what happened; run `make test-copilot-smoke` or a real install instead, and record the result per `host-probe-protocol.spec.md`.

### 3. Modify an existing skill

The plugin ships **7 skills** (per `skill-surface-collapse.adr.md`): `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`. Each lives at `skills/<name>/SKILL.md`. Adding an eighth top-level skill requires a new ADR — prefer adding flow logic under `skills/plan/references/` or `skills/decide/references/` instead.

Edit `skills/<name>/SKILL.md`. Required frontmatter fields: `name` (must match directory name), `description`. Optional: `argument-hint`. No skill carries `disable-model-invocation` — all 7 are auto-invocable.

Reload and test: `/reload-plugins`, then try `/archcore:<name>`.

#### 3a. Add a slash command wrapper (required for user-facing skills)

Claude Code, Cursor and Copilot surface skills directly in the `/` menu. Codex CLI does not — it discovers slash commands from `commands/<name>.md` files. The plugin ships 7 wrappers, one per skill. If you ever add a new top-level skill (requires a new ADR), add the matching wrapper:

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

Wrappers carry no workflow logic — behavior lives in the skill, the single source of truth. `test/structure/codex-plugin.bats` and `copilot-plugin.bats` enforce parity: every wrapper must exist, carry `description:`, and reference its matching `skills/<name>/SKILL.md`.

Copilot loads the same wrappers, but only because `.plugin/plugin.json` names `./commands/` explicitly — that field has no default path on Copilot. A skill outranks a command of the same name there, so the wrappers are a fallback surface rather than the primary one; the pointer still has to be present, and a structure test keeps it there.

### 4. Add or modify hooks

Edit `hooks/hooks.json` (Claude Code), `hooks/cursor.hooks.json` (Cursor), `hooks/codex.hooks.json` (Codex CLI), or `hooks/copilot.hooks.json` (GitHub Copilot CLI) to add event handlers. Every hooks config must also be enrolled in `test/structure/host-coverage-matrix.bats` and in the resolution table in `hooks.bats`; both have enrollment guards that fail until it is.

Hook scripts go in `bin/` and must:

- Start with `#!/bin/sh`
- Be executable (`chmod +x`)
- Source `bin/lib/normalize-stdin.sh` if they read hook stdin
- Add `# shellcheck source=lib/normalize-stdin.sh` before the source line
- Invoke the CLI directly as `archcore` (resolved via PATH); the plugin no longer ships a launcher wrapper
- If the script reads `.archcore/` or emits user-visible context, guard against being launched from a plugin install directory by exiting silently when cwd contains — or sits beneath — a `.cursor-plugin/`, `.claude-plugin/`, `.codex-plugin/`, or `.plugin/` manifest (see `bin/session-start` for the canonical pattern, and `cursor-mcp-architecture.adr.md` for the rationale)

Three hosts substitute a single canonical plugin-root env var; Copilot is the exception:

- `${CLAUDE_PLUGIN_ROOT}` — Claude Code's native injection (`hooks/hooks.json`).
- `${CURSOR_PLUGIN_ROOT}` — Cursor's native injection (`hooks/cursor.hooks.json`).
- `${PLUGIN_ROOT}` — Codex CLI's canonical, host-neutral env var (`hooks/codex.hooks.json`). Codex's hooks engine (`codex-rs/hooks/src/engine/discovery.rs`) injects `PLUGIN_ROOT` as the canonical name; `CLAUDE_PLUGIN_ROOT` is also injected but only as a backward-compat alias for porting old Claude plugins — do NOT use it in a Codex-native hook config. `CODEX_PLUGIN_ROOT` does not exist in Codex.
- **Copilot has no confirmed variable, so `hooks/copilot.hooks.json` probes three.** `COPILOT_PLUGIN_ROOT` — which this adapter relied on until 2026-07-27 — appears in no GitHub documentation; the only documented spelling is `${PLUGIN_ROOT}` ("Use `${PLUGIN_ROOT}` to reference paths within the plugin directory"), and for writable state the documented pair is `${COPILOT_PLUGIN_DATA}` / `${CLAUDE_PLUGIN_DATA}`. Each command now tests `$COPILOT_PLUGIN_ROOT`, `$PLUGIN_ROOT` and `$CLAUDE_PLUGIN_ROOT` in turn with `-x`, execs the first that actually holds the script, and otherwise warns on stderr and exits 0. Variables are written unbraced so the command behaves identically whether Copilot expands them or `sh` does. Whichever variable is real, `bin/detect-host` still cannot key on it — it exists only inside hook processes, so a Copilot session resolves to `__UNKNOWN__` there.

If you edit those commands, note that `test/structure/copilot-plugin.bats` **runs** them under `env -u` rather than comparing strings: exit 0 with a warning when nothing resolves, each candidate sufficient alone, a dead candidate skipped rather than fatal. The old string-equality assertions matched the broken command exactly and shipped it — see `copilot-adapter-design.adr.md` and `cli-integration-tests.rule.md`.

Copilot's config differs from the others in shape, not just in names: entries use `bash` rather than `command`, `timeoutSec` rather than `timeout`, are flat objects rather than nested groups, carry `cwd: "."` so the hook runs from the user's project, and its `postToolUse` entries have no matcher at all — the scripts self-filter. A test written by copying another host's and swapping the filename will iterate an empty set and report `ok`.

Plugin-shipped Codex hooks require `codex features enable plugin_hooks` to actually fire (the `plugin_hooks` feature is `under development, false` by default in Codex 0.130.0). See `codex-path-resolution.adr.md` for the full mechanism.

Two Copilot hook semantics differ from every other host and both are load-bearing. First, **every non-zero exit denies**: `exit 2` is a deny whose stdout JSON is merged with the deny decision, and any other non-zero exit denies as `Denied by preToolUse hook (hook errored)` (hooks-reference, re-read 2026-07-27). Guard scripts still write `{"permissionDecision":"deny","permissionDecisionReason":…}` to stdout with exit 0, because that is how a deny carries its reason — not because exit 2 fails to block. The corollary is that a guard which cannot *start* denies the write too, which is why hook bootstrap gets the candidate chain above. Second, a `preToolUse` **timeout fails open**, which makes guard latency a correctness concern rather than a comfort one; `test/unit/hook-latency.bats` keeps both PreToolUse guards far inside the 1-second budget for that reason.

Hooks are also narrower than plugins on this host: hooks-reference names exactly two supported surfaces, Copilot CLI and Copilot cloud agent. VS Code agent mode is not one of them, even though Copilot Chat ships its own CLI binary under the extension's `globalStorage` and hook machinery has been observed firing there.

### 5. Modify agents

Edit `agents/archcore-assistant.md` or `agents/archcore-auditor.md`:

- Frontmatter: `name`, `description`, `model`, `maxTurns`, `tools`
- The auditor must remain read-only (only list_documents, get_document, list_relations MCP tools)
- Tool lists carry every MCP naming: `mcp__archcore__*`, `mcp__plugin_archcore_archcore__*`, and Copilot's flat `archcore-<tool>`

Then propagate to the two format variants, both checked by `test/structure/agents.bats`:

- **Codex** — `agents/<name>.toml`; TOML and MD must keep identical `developer_instructions` content.
- **Copilot** — `copilot-agents/<name>.agent.md`; a byte-identical copy (`cmp`), because Copilot's loader accepts only the `*.agent.md` extension. Keep it in `copilot-agents/`, never beside the original: `.agent.md` still matches the `*.md` glob Claude Code and Cursor use, so a sibling copy would hand both hosts two files declaring the same `name:`.

### 6. Run tests

After any change, verify everything works:

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

`make verify` is the canonical way to run plugin integrity checks; there is no `/archcore:verify` skill (removed by `skill-surface-collapse.adr.md`).

See `plugin-testing.guide.md` for detailed testing instructions.

### 7. Test all components manually

- Skills: discuss relevant topics and verify the host activates the skill
- Commands: run each `/archcore:<name>` command on every host you can reach and verify behavior — Codex pulls these from `commands/`, Claude Code, Cursor and Copilot from `skills/`
- Agent: invoke the agent on a multi-document task
- Hooks: trigger Write/Edit on `.archcore/` and verify the pre-mutation guard blocks it
- MCP availability: ensure `archcore` is on PATH and `archcore --version` works
- For Codex: from a directory **outside** the plugin source repo (e.g., `cd $(mktemp -d)`), call any `mcp__archcore__*` tool and verify the MCP starts.
- For Cursor: after copying `docs/cursor.mcp.example.json` into `.cursor/mcp.json`, open an empty project. `list_documents` should return empty (not the plugin's own dev docs). If it returns dev docs, the plugin-install-dir guards regressed — file an issue against this repo and `archcore-ai/cli`.
- For Copilot: `copilot mcp list` WILL show a plugin-contributed `archcore` server — settled 2026-08-03: the host auto-discovers plugin-root `.mcp.json` regardless of the manifest (see `copilot-mcp-architecture.adr.md`, update). Verify instead that this server **fails to start** with the plugin-cache guard error (CLI >= v0.6.7) and that the project-wired server from `.mcp.json` at the repo root is the one serving tools. A plugin server that starts and serves is the regression.
- For Copilot: session start must NOT print `archcore: plugin root unresolved`. If it does, no candidate variable was injected and **every guard is silently disabled** for that session — capture which load path produced it, because that is the open question in `copilot-adapter-design.adr.md`.
- Integrity check: `make verify`

For the questions no manual checklist can settle — whether a deny is honored or merely displayed, whether pre-mutation hooks fire on delegated calls, which plugin-root variable Copilot injects — follow `host-probe-protocol.spec.md` and record the result.

## Verification

- `make verify` exits 0 with "All checks passed"
- `/reload-plugins` shows correct count of skills (7), agents (2), hooks (6 entries)
- `/help` lists all `/archcore:*` commands (7)
- `/agents` lists `archcore-assistant` and `archcore-auditor`
- Writing to `.archcore/*.md` via Write/Edit is blocked with a redirect message
- `archcore --version` works (CLI is on PATH or installed globally)
- For all hosts: MCP tools work from any project directory (the CLI resolves via PATH)

## Common Issues

### Plugin not loading

- Ensure the manifest for your host exists and has valid JSON: `.claude-plugin/plugin.json` (Claude Code), `.cursor-plugin/plugin.json` (Cursor), `.codex-plugin/plugin.json` (Codex CLI), `.plugin/plugin.json` (GitHub Copilot CLI)
- Check that directories (skills/, agents/, copilot-agents/, hooks/, commands/) are at the plugin root
- Run `claude --debug` to see plugin loading details

### Skill not activating

- Check the `description` field in SKILL.md frontmatter — it determines when the host activates the skill
- Ensure `name` matches the directory name
- Run `/reload-plugins` after changes

### `/archcore:<name>` missing in the Codex `/` menu

- Confirm `commands/<name>.md` exists and has `description:` frontmatter
- Confirm it references `skills/<name>/SKILL.md` (the bats parity test enforces this)
- Run `make test-structure` — `codex-plugin.bats` will flag missing or malformed wrappers
- Restart Codex after adding new wrappers (the marketplace cache is read once on session start)

### Agents missing in Copilot

- Confirm the file is `copilot-agents/<name>.agent.md`. Copilot derives the agent id from the filename and loads only that extension — a plain `<name>.md` is invisible to it, which is exactly why the copies exist.
- Confirm `.plugin/plugin.json` has `"agents": "./copilot-agents/"`. The field defaults to `agents/`, so an omitted pointer silently looks in the directory that holds no `*.agent.md` files at all.

### Hook not firing

- Ensure bin/ scripts are executable: `chmod +x bin/<name>`
- Check the shebang line: `#!/bin/sh`
- Verify the hook JSON structure matches the expected format for that host
- Test scripts manually: `echo '{"tool_name":"Write","tool_input":{"file_path":".archcore/test.adr.md"}}' | bin/check-archcore-write`
- For Codex specifically: hooks require `codex features enable plugin_hooks` (the `plugin_hooks` feature is under development; absent the flag, Codex does not run plugin-shipped hooks)
- For Copilot specifically: check the entry uses `bash`, not `command`, and `timeoutSec`, not `timeout` — a config written in Claude's shape loads without error and does nothing

### `/bin/sh: /bin/<script>: No such file or directory` (GitHub Copilot CLI)

The historical symptom of an unresolved plugin root: `"${COPILOT_PLUGIN_ROOT}"/bin/session-start` collapses to `/bin/session-start` when the variable is empty. A current plugin prints `archcore: plugin root unresolved` instead and exits 0, so if you see the old message the session is running a build from before 2026-07-27 — reinstall.

Either way the cause is the same, and on `preToolUse` the consequence is severe: a failed exec is a non-zero exit, which Copilot reads as a deny, so **every** `create|edit|str_replace_editor|apply_patch` call is refused with `Denied by preToolUse hook (hook errored)`. Check in this order:

1. **Which surface?** Hooks are documented for Copilot CLI and cloud agent only. A VS Code / Copilot Chat session is neither, even though it ships its own CLI binary — nothing about variable injection is guaranteed there.
2. **Which load path?** `--plugin-dir` is not `copilot plugin install`. Confirm `copilot plugin list` shows `archcore`.
3. **Which config?** Copilot reads hooks from six places besides a plugin (policy `.d` files, `.github/hooks/*.json`, `~/.copilot/hooks/`, `.github/copilot/settings.json`, `~/.copilot/settings.json`, and the shared subset of repo-level `.claude/settings.json`). A plugin's hooks config copied into any of those gets no plugin root by design.

### Tests failing

- Run `git submodule update --init` if bats helpers are missing
- On macOS, the test suite provides a `timeout` shim automatically
- Ensure `archcore` CLI is installed on PATH (`archcore --version`)
- See `plugin-testing.guide.md` for detailed troubleshooting

### MCP server not connecting (Claude Code / Codex CLI)

The plugin ships `.mcp.json` for Claude Code and `.codex.mcp.json` for Codex CLI. Diagnose in this order:

1. **Plugin loaded?** — `/plugin` (Claude Code) or `codex mcp list --json` (Codex CLI) should show `archcore`. If `.mcp.json`, `.codex.mcp.json`, or the Codex `mcpServers` pointer was modified or removed, the MCP server won't register; restore it from git.
2. **CLI available?** — run `archcore --version` from the terminal. Expected: prints a version.
   - Not found? → Install via the official installer: `curl -fsSL https://archcore.ai/install.sh | bash` (macOS/Linux/WSL) or `irm https://archcore.ai/install.ps1 | iex` (Windows). Full docs: https://docs.archcore.ai/cli/install/
   - Permission denied? → Check that the CLI binary is executable
3. **Session lifecycle** — Claude Code registers MCP servers at session start. If the CLI was missing at that moment, installing it mid-session will NOT reconnect the server. Restart the host after a fresh install.
4. **Duplicate suppression?** — if `/plugin` shows "Errors (1)" with an `archcore` MCP message, a user- or project-registered `archcore` has the same command. This is benign; the resolved binary is the same either way. To silence the warning, remove the redundant user/project registration.

### No MCP tools at all (GitHub Copilot CLI)

Expected until the project is wired — the plugin ships no MCP server for Copilot. Run `archcore init --agent copilot --project "$PWD"` (CLI ≥ v0.6.4), which writes the workspace-root `.mcp.json`, then restart the session. Copilot discovers `.mcp.json` by walking from the working directory up to the git root, so a repo-root file covers monorepo layouts.

If tools appear but documents land somewhere unexpected, check where: github/copilot-cli#4234 puts a plugin-contributed MCP child in `~/.copilot/installed-plugins/`. A project-registered server does not have that problem, because the host launches it from the project.

### MCP server not connecting (Cursor)

Cursor uses a user-installed MCP, not a plugin-shipped one (deliberate — see `cursor-mcp-architecture.adr.md`). Copy `docs/cursor.mcp.example.json` into one of:

- `~/.cursor/mcp.json` — user-scoped, available in every workspace
- `.cursor/mcp.json` — project-scoped, only this project

The file ships with the right shape:

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

Key points:

- **`--project ${workspaceFolder}` is mandatory.** Cursor's MCP stdio schema has no `cwd` field; without `--project`, the server falls back to `os.Getwd()` which is unreliable for plugin-launched processes.
- **Do not add a `cwd` field.** Cursor silently ignores it; doing so is just confusing.
- **Do not copy the template to the plugin root.** A `cursor.mcp.json` at the plugin root would let Cursor's plugin-MCP auto-detection register the server with cwd = plugin install dir, leaking any bundled `.archcore/` (or other unintended state) instead of the user's workspace.

### "Plugin MCP Servers → archcore" appears in Cursor settings with stale documents

The plugin deliberately ships no Cursor plugin-MCP. If Cursor's "Plugin MCP Servers" section shows `archcore`, then either (a) an older plugin version with a plugin-root `cursor.mcp.json` is still cached, or (b) a regression introduced a plugin-root MCP file. Steps:

1. Uninstall the plugin from Cursor.
2. Remove `~/.cursor/plugins/cache/archcore-plugins/` (or the relevant cache subtree).
3. Reinstall the plugin from `main` (which is synthesized by the release workflow and has no plugin-root `cursor.mcp.json`).
4. Verify `test/structure/cursor-plugin.bats` passes — it asserts no legacy `cursor.mcp.json` at the plugin root.

If the symptom persists after a fresh `main` install, file an issue: the `cursor-mcp-architecture.adr.md` layered defense has a gap.
