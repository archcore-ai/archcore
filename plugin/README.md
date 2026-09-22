# Archcore Plugin

The agent half of [Archcore](../README.md): one plugin tree that Claude Code, Cursor, Codex CLI, and GitHub Copilot CLI load, with four commands, the skills behind them, two agents, and the hook launchers that hand events to the CLI. Product overview and installation: the [repository README](../README.md). This file is for people and agents who change the plugin.

The plugin needs the Archcore CLI on `PATH` (`archcore --version`). The hook launchers require CLI v0.7.0 or later and fail open on anything older; `make verify` requires v0.8.3 or later.

## Layout

| Path | Content |
| --- | --- |
| `.agents/plugins/marketplace.json`, `.claude-plugin/marketplace.json`, `.cursor-plugin/marketplace.json` | The three marketplace catalogs; each resolves `./plugins/archcore` |
| `plugins/archcore/.claude-plugin/`, `.cursor-plugin/`, `.codex-plugin/`, `.plugin/` | One manifest per host; `version` is equal in all four and equals the release tag |
| `plugins/archcore/skills/` | `init`, `plan`, `document`, `review`, and `_shared/`: content contracts, precision rules, tracks, grounding |
| `plugins/archcore/commands/` | Slash-command wrappers for Codex CLI and Copilot; each points at its skill |
| `plugins/archcore/agents/`, `plugins/archcore/copilot-agents/` | `archcore-assistant` and the read-only `archcore-auditor`: Markdown, a TOML twin for Codex, and a byte-identical `.agent.md` copy for Copilot |
| `plugins/archcore/hooks/` | `hooks.json` for Claude Code, `cursor.hooks.json`, `codex.hooks.json`, `copilot.hooks.json` |
| `plugins/archcore/bin/` | POSIX sh launchers `session-start`, `pre-tool-use`, `post-tool-use`, plus `detect-host`, `cli-gte`, and `lib/` |
| `plugins/archcore/rules/`, `plugins/archcore/assets/` | Cursor rules; the icon and logo |
| `plugins/archcore/.claude.mcp.json`, `plugins/archcore/.codex.mcp.json` | MCP registration for Claude Code and Codex CLI |
| `docs/` | `cursor.mcp.example.json`, the user-level MCP template for Cursor; `TERMS.md`; `release.md` |
| `test/` | bats suites under `structure/`, `unit/`, `integration/`, and the on-demand `behavioral/` benches |

## Commands and modes

| Command | Modes | Instrument |
| --- | --- | --- |
| `/archcore:init` | `import`, `refresh` | First-day seed, host wiring, conversion of instruction files and ADR folders |
| `/archcore:plan` | `sdd`, `sources`, `iso`, `research` | A computed route: null route, spec plus plan, or PRD with one spec per capability |
| `/archcore:document` | `decision`, `code`, `research` | ADR or RFC, description of code, a report or one external material as evidence |
| `/archcore:review` | `drift`, `deep`, `closeout`, `experience` | Branch review, staleness, full audit, feature closeout, repeated-pattern capture |

The first word after a command is the mode and the rest is the subject. A gate picks the document type unless the subject names it. Every gate skips itself when an existing document covers it; a vague request stays within five questions; a draft document carries the route state, so an interrupted flow resumes later. The `research` and `evidence` types need CLI v0.8.3 or later; an older CLI falls back to `rnd` and reports the version.

## Hosts

| Host | MCP | Hooks | Install |
| --- | --- | --- | --- |
| Claude Code | `.claude.mcp.json` through the manifest | `hooks/hooks.json`, `${CLAUDE_PLUGIN_ROOT}` | `/plugin marketplace add archcore-ai/archcore`, then `/plugin install archcore@archcore-plugins` |
| Cursor 2.5+ | None shipped: Cursor spawns a plugin MCP from the install directory. The user copies `docs/cursor.mcp.example.json` into `~/.cursor/mcp.json` | `hooks/cursor.hooks.json`, `${CURSOR_PLUGIN_ROOT}` | Plugins → paste the repository URL |
| Codex CLI 0.117+ | `.codex.mcp.json` through the manifest | `hooks/codex.hooks.json`, `${PLUGIN_ROOT}` | `codex plugin marketplace add archcore-ai/archcore` |
| GitHub Copilot CLI | None shipped: Copilot launches a plugin MCP without a project path. `archcore init --agent copilot --project "$PWD"` wires the project | `hooks/copilot.hooks.json`; the command probes three root variables | `copilot plugin install archcore-ai/archcore:plugins/archcore` |

Every launcher sources `bin/lib/plugin-cache-guard.sh` and refuses to serve from a plugin cache. Identifiers frozen across hosts: marketplace `archcore-plugins`, plugin `archcore`, id `archcore@archcore-plugins`, path `plugins/archcore`.

## Develop

```bash
claude  --plugin-dir plugins/archcore    # Claude Code
cursor  --plugin-dir plugins/archcore    # Cursor
copilot --plugin-dir plugins/archcore    # GitHub Copilot CLI
codex plugin marketplace add "$PWD"      # Codex CLI: a local marketplace, then codex plugin add archcore@archcore-plugins
```

`/reload-plugins` inside a session picks up file changes. Copilot under `--plugin-dir` may inject no plugin-root variable; when session start prints `plugin root unresolved`, use `make test-copilot-smoke` or a real install.

```bash
make verify              # JSON, permissions, ShellCheck, unit and structure tests, real MCP tests
make test                # unit and structure tests only
make test-integration    # real MCP tests; needs the CLI on PATH or ARCHCORE_BIN
make test-codex-smoke    # skips without the codex CLI
make test-copilot-smoke  # skips without the copilot CLI
```

Rules of the tree:

- Executable code under `bin/` is POSIX sh and passes ShellCheck. No Go, no bundled CLI, no download-on-first-use.
- A fifth top-level command needs an ADR; flow logic goes into the gated track layer under `skills/_shared/tracks/`.
- The three copies of an agent definition stay identical, and every hooks config is enrolled in `test/structure/host-coverage-matrix.bats`.
- Skills perform every `.archcore/` write through an MCP tool.

## Release

`main` is generated from a tag by the root Release workflow: `scripts/export-plugin.sh` copies this tree, the repository README, `LICENSE`, and `NOTICE` into the public layout and force-pushes it. Set the four manifests to the version with `/bump-plugin-version X.Y.Z` before tagging. Procedure, published files, and recovery: [`docs/release.md`](docs/release.md).

## Where the design lives

Under the repository's `.archcore/plugin/`, by name: `plugin-development.guide`, `plugin-testing.guide`, `component-registry.doc`, `plugin-architecture.spec`, `hooks-validation-system.spec`, `multi-host-plugin-architecture.adr`, `cursor-mcp-architecture.adr`, `copilot-mcp-architecture.adr`, and `stack-and-tooling.rule`. Read them through the MCP tools from the repository root.
