---
title: "Multi-Host Plugin Architecture — Single Repo for Multiple AI Coding Tools"
status: accepted
tags:
  - "architecture"
  - "multi-host"
  - "plugin"
---

## Context

The Archcore plugin must work across multiple AI coding hosts that have converged on common open standards (April 2026 research):

- **Agent Skills standard** (agentskills.io) — adopted by Cursor, GitHub Copilot, Codex CLI, Roo Code, Cline, Gemini CLI, Windsurf, JetBrains Junie, OpenHands
- **MCP (Model Context Protocol)** — adopted by all of the above plus Amazon Q, Continue.dev, Zed AI
- **Markdown agent definitions** — adopted by Cursor, GitHub Copilot, Codex CLI, Gemini CLI

Analysis showed that ~95% of plugin code is host-agnostic: skills, agents, hook scripts, and the stdin-normalization library. Only manifest files and hooks configs are host-specific.

### Drivers

- Users of Cursor, Copilot, and Codex CLI need Archcore integration — Codex CLI v0.117.0+ (March 2026) added a plugin system with near-1:1 surface to Claude Code, validating the third-host bet.
- Industry convergence on Agent Skills + MCP makes cross-host support low-effort.
- Maintaining separate repos per host would mean duplicating 16 skills, 2 agents, and 6 hook scripts.

## Decision

**Support multiple AI coding hosts from a single repository** with a shared core and thin per-host adapter layer.

The plugin ships:

- **Shared core** (host-agnostic): `skills/`, `agents/` (both `.md` and `.toml` variants), `bin/` (hook scripts + stdin normalizer), `commands/` (Codex slash command wrappers).
- **Per-host adapter files** (configuration only, no logic): plugin manifest, hooks config, and an MCP config per host.
- **No bundled CLI**: the Archcore CLI is installed globally by the user per https://docs.archcore.ai/cli/install/. All three MCP configs name `archcore` as the command, resolved via PATH. The plugin does not bundle, download, cache, or pin a CLI binary.

The plugin itself lives in a dedicated `plugins/archcore/` subdirectory; the marketplace catalogs stay at the repo root and point each host's plugin `source`/`path` at that subdirectory.

```
repo-root/                               # marketplace CATALOGS + dev tooling
├── .claude-plugin/marketplace.json      # Claude catalog  → source: ./plugins/archcore
├── .cursor-plugin/marketplace.json      # Cursor catalog  → source: ./plugins/archcore
├── .agents/plugins/marketplace.json     # Codex catalog   → path:   ./plugins/archcore
├── docs/cursor.mcp.example.json         # Reference template users copy into ~/.cursor/mcp.json
│
└── plugins/archcore/                    # ← the plugin (single source of truth; what each host installs)
    ├── commands/                        # Codex CLI slash command wrappers (7, host-adapter shims)
    ├── skills/                          # Shared — Agent Skills standard (7 skills)
    ├── agents/                          # Shared — markdown agent definitions + Codex TOML variants
    │   ├── archcore-assistant.md        # Claude Code / Cursor
    │   ├── archcore-assistant.toml      # Codex CLI (sandbox_mode = "workspace-write")
    │   ├── archcore-auditor.md          # Claude Code / Cursor
    │   └── archcore-auditor.toml        # Codex CLI (sandbox_mode = "read-only" + disabled_tools)
    ├── bin/                             # Shared — hook scripts + stdin normalizer (no CLI binary)
    │   ├── lib/normalize-stdin.sh
    │   ├── session-start
    │   ├── check-archcore-write
    │   ├── check-code-alignment
    │   ├── validate-archcore
    │   ├── check-cascade
    │   ├── check-precision
    │   └── check-staleness
    │
    ├── .claude-plugin/plugin.json       # Claude Code manifest
    ├── .cursor-plugin/plugin.json       # Cursor manifest (no `mcpServers` field — deliberate)
    ├── .codex-plugin/plugin.json        # Codex CLI manifest (single file)
    │
    ├── hooks/
    │   ├── hooks.json                   # Claude Code (PascalCase events)
    │   ├── cursor.hooks.json            # Cursor (camelCase events)
    │   └── codex.hooks.json             # Codex CLI (PascalCase events + apply_patch matcher)
    │
    ├── .mcp.json                        # Claude Code — { "archcore": { "command": "archcore", "args": ["mcp"] } }
    ├── .codex.mcp.json                  # Codex CLI — direct server map
    └── rules/                           # Cursor-only context rules (.mdc)
```

**Catalog vs. plugin location.** The three marketplace catalogs stay at the repo root; each points its plugin `source`/`path` at the `plugins/archcore/` subdirectory, which holds the per-host manifests and all shared content. This subdirectory layout is *required* for Codex marketplace discovery — a catalog `source.path` of `./` (the marketplace root) is not scanned, so the plugin is never discovered there — and it is the canonical layout for Claude Code and Cursor as well. The reporter that surfaced this is issue #2; the full rationale, the cross-host docs matrix, and the rejected alternatives (generated copy, Windows-breaking symlinks) live in `subdirectory-plugin-layout.adr`, which extends this ADR.

### Shared core principle

Skills, agents, and hook scripts are maintained once. All host-specific adapters are pure configuration — no logic duplication.

### MCP wiring

MCP is wired via plugin-shipped configs for Claude Code (`.mcp.json`) and Codex CLI (`.codex.mcp.json` pointed at by `.codex-plugin/plugin.json`'s `mcpServers` field). Both name `archcore` directly; Codex uses a direct server map in `.codex.mcp.json`, while Claude Code uses its `mcpServers` wrapper. The host runtime resolves `archcore` from PATH. Both hosts launch the MCP with cwd inherited from the user's project process, which is the correct workspace.

Cursor is the exception. Cursor 2.5+ does auto-detect plugin-shipped MCP configs (per the [official plugins reference](https://cursor.com/docs/reference/plugins.md), an `mcp.json` at the plugin root registers under "Plugin MCP Servers"), but it spawns the plugin-MCP from the plugin install directory rather than the workspace, and its MCP stdio schema has no `cwd` field ([forum #74861](https://forum.cursor.com/t/allow-workspacefolder-in-mcp-project-configration/74861), [forum #99215](https://forum.cursor.com/t/how-get-the-correct-current-work-directory-in-mcp-server/99215)). Until those gaps close upstream, shipping a plugin-MCP for Cursor would cause the server to read from the plugin install dir instead of the user's project. We therefore deliberately do **not** ship a plugin-MCP for Cursor: no `mcpServers` field in `.cursor-plugin/plugin.json`, no `mcp.json` at the plugin root, and the reference template lives under `docs/` so it cannot trigger auto-detection. Cursor users copy `docs/cursor.mcp.example.json` into `~/.cursor/mcp.json` or `.cursor/mcp.json`; the template passes `--project ${workspaceFolder}` in `args` so the server resolves the workspace regardless of cwd. See `cursor-mcp-architecture.adr.md` for the full rationale and three-layer defense (release-strip, docs-only template, runtime guards).

The plugin does not bundle the CLI, does not download it, and does not cache it; users install it once via the official installer at https://docs.archcore.ai/cli/install/.

### Stdin normalization

Hook scripts source a shared `bin/lib/normalize-stdin.sh` that detects the host from stdin JSON structure and exposes a canonical schema (`ARCHCORE_HOST`, `ARCHCORE_TOOL_NAME`, `ARCHCORE_FILE_PATH`, etc.). Detection uses each host's distinct stdin fields (Claude Code → `tool_name`; Cursor → `conversation_id`; Copilot → `hookEventName`; Codex → `turn_id`). Codex shares Claude Code's snake_case schema, so the field-extraction logic for `codex` mirrors `claude-code`.

## Alternatives Considered

### 1. Separate repository per host

One repo for Claude Code, one for Cursor, one for Codex CLI. Each contains full copies of skills, agents, and bin scripts.

**Rejected because:** 16 skills × N hosts = massive duplication; any skill update must be synced across all repos; only ~5% of code is actually host-specific.

### 2. Build system that generates per-host packages

A mono-repo with a build step (e.g., Node.js script) that reads a canonical source and generates separate plugin directories per host.

**Rejected because:** introduces build tooling to a project that is currently pure Markdown + Shell; complexity not warranted — the per-host differences are purely configuration (JSON files); Agent Skills standard already ensures skills work across hosts without transformation.

### 3. Symlinks from host-specific directories to shared source

**Rejected because:** symlinks don't work reliably on Windows; plugin marketplace systems distribute files, not symlinks; fragile when cloned or copied.

### 4. Bundle the CLI inside the plugin (download-on-first-use launcher)

Ship a `bin/archcore` launcher that resolves the Archcore CLI on demand from `$ARCHCORE_BIN`, PATH, a plugin-managed cache, or a GitHub Releases download.

**Tried and reverted.** Shipped briefly under `bundled-cli-launcher.adr` (now rejected), then removed in plugin v0.4.0 (2026-05-12) per `remove-bundled-launcher-global-cli.idea`. Eight bug classes — offline CI failures, security patch lag, uneven host support (Cursor still required manual setup), cache pollution, first-run latency, enterprise friction, version coupling to plugin releases, and 2000+ lines of launcher/test code — made the "zero-setup install" framing a net loss. The official installer at https://docs.archcore.ai/cli/install/ is the supported path; one-time user install replaces the bundled-launcher complexity.

### 5. Ship `mcp.json` at the plugin root with `--project ${workspaceFolder}` in args

This is the canonical Cursor 2.5+ way to register a plugin MCP. Rejected because Cursor's `${workspaceFolder}` interpolation inside plugin-MCP `args` is undocumented and the open feature request ([forum #74861](https://forum.cursor.com/t/allow-workspacefolder-in-mcp-project-configration/74861)) implies plugin-MCPs do not get the interpolation that user-config MCPs do. Until Cursor confirms support, we cannot rely on it; see `cursor-mcp-architecture.adr.md` for the layered defense we adopted instead.

## Consequences

### Positive

- **Zero skill/agent duplication**: skills and agents maintained in one place.
- **Low per-host cost**: adding a new host requires only a manifest (~10 lines), a hooks config (~30 lines), and an MCP config (~5 lines). Codex was the first real test — port took ~1 dev-day for scaffolding plus tests.
- **Decoupled CLI lifecycle**: the Archcore CLI ships and patches on its own cadence; plugin releases never gate CLI security fixes (and vice versa). `archcore update` is the user-facing upgrade path.
- **Standard compliance**: uses Agent Skills, MCP, and markdown agents — all open standards.
- **Single source of truth**: bug fixes in skills/agents/bin propagate to all hosts automatically; the plugin lives in exactly one place (`plugins/archcore/`), with no per-host copy or symlink to keep in sync.

### Negative

- **CLI install is the user's responsibility**: an unsupported user expectation (e.g., "the plugin should just work") surfaces when `archcore` is missing from PATH. Mitigation: `bin/session-start` prints the install command and a docs link on every fresh session where the CLI is absent; `plugin-development.guide` documents the MCP session-start lifecycle gotcha (installing the CLI mid-session does not reconnect a Claude Code MCP that failed to register at session start — restart required).
- **Stdin normalization complexity**: hook scripts must handle multiple JSON formats. Mitigated by the centralized normalizer.
- **Hook event mapping is imperfect**: not all hosts have equivalent hook events (e.g., Cursor has no direct `SessionStart` equivalent; its PreToolUse matcher is `Write` only, not `Write|Edit`). Mitigation: use the closest available event per host; document gaps.
- **Cursor MCP is user-installed only**: Cursor users have a one-time copy step from `docs/cursor.mcp.example.json` into `~/.cursor/mcp.json`. This is a deliberate trade-off (see `cursor-mcp-architecture.adr.md`) — shipping a plugin-MCP would leak the plugin's bundled `.archcore/` into every Cursor install. Revisit when Cursor's plugin-MCP cwd handling improves.
- **Subagent format divergence**: Claude Code and Cursor read MD agents with YAML frontmatter; Codex requires TOML. Mitigated by shipping both formats side-by-side; `test/structure/agents.bats` enforces parity between MD and TOML `developer_instructions` bodies.
