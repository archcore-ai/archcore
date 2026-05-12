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

```
plugin/
├── commands/                    # Codex CLI slash command wrappers (16, host-adapter shims)
├── skills/                      # Shared — Agent Skills standard (16 skills)
├── agents/                      # Shared — markdown agent definitions + Codex TOML variants
│   ├── archcore-assistant.md    # Claude Code / Cursor
│   ├── archcore-assistant.toml  # Codex CLI (sandbox_mode = "workspace-write")
│   ├── archcore-auditor.md      # Claude Code / Cursor
│   └── archcore-auditor.toml    # Codex CLI (sandbox_mode = "read-only" + disabled_tools)
├── bin/                         # Shared — hook scripts + stdin normalizer (no CLI binary)
│   ├── lib/normalize-stdin.sh
│   ├── session-start
│   ├── check-archcore-write
│   ├── check-code-alignment
│   ├── validate-archcore
│   ├── check-cascade
│   ├── check-precision
│   └── check-staleness
│
├── .claude-plugin/              # Claude Code manifest + marketplace
├── .cursor-plugin/              # Cursor manifest + marketplace
├── .codex-plugin/               # Codex CLI manifest (single file)
├── .agents/plugins/             # Codex marketplace descriptor
│
├── hooks/
│   ├── hooks.json               # Claude Code (PascalCase events)
│   ├── cursor.hooks.json        # Cursor (camelCase events)
│   └── codex.hooks.json         # Codex CLI (PascalCase events + apply_patch matcher)
│
├── .mcp.json                    # Claude Code — { "archcore": { "command": "archcore", "args": ["mcp"] } }
├── .codex.mcp.json              # Codex CLI — same shape
├── cursor.mcp.json              # Reference template users copy into ~/.cursor/mcp.json
└── rules/                       # Cursor-only context rules (.mdc)
```

### Shared core principle

Skills, agents, and hook scripts are maintained once. All host-specific adapters are pure configuration — no logic duplication.

### MCP wiring

MCP is wired via plugin-shipped configs for Claude Code (`.mcp.json`) and Codex CLI (`.codex.mcp.json` pointed at by `.codex-plugin/plugin.json`'s `mcpServers` field). Both name `archcore` directly; the host runtime resolves it from PATH. Cursor users register MCP externally by copying `cursor.mcp.json` into `~/.cursor/mcp.json` or `.cursor/mcp.json` — Cursor does not auto-register plugin-shipped MCP. The plugin does not bundle the CLI, does not download it, and does not cache it; users install it once via the official installer at https://docs.archcore.ai/cli/install/.

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

## Consequences

### Positive

- **Zero skill/agent duplication**: skills and agents maintained in one place.
- **Low per-host cost**: adding a new host requires only a manifest (~10 lines), a hooks config (~30 lines), and an MCP config (~5 lines). Codex was the first real test — port took ~1 dev-day for scaffolding plus tests.
- **Decoupled CLI lifecycle**: the Archcore CLI ships and patches on its own cadence; plugin releases never gate CLI security fixes (and vice versa). `archcore update` is the user-facing upgrade path.
- **Standard compliance**: uses Agent Skills, MCP, and markdown agents — all open standards.
- **Single source of truth**: bug fixes in skills/agents/bin propagate to all hosts automatically.

### Negative

- **CLI install is the user's responsibility**: an unsupported user expectation (e.g., "the plugin should just work") surfaces when `archcore` is missing from PATH. Mitigation: `bin/session-start` prints the install command and a docs link on every fresh session where the CLI is absent; `plugin-development.guide` documents the MCP session-start lifecycle gotcha (installing the CLI mid-session does not reconnect a Claude Code MCP that failed to register at session start — restart required).
- **Stdin normalization complexity**: hook scripts must handle multiple JSON formats. Mitigated by the centralized normalizer.
- **Hook event mapping is imperfect**: not all hosts have equivalent hook events (e.g., Cursor has no direct `SessionStart` equivalent; its PreToolUse matcher is `Write` only, not `Write|Edit`). Mitigation: use the closest available event per host; document gaps.
- **MCP wiring is host-specific**: Claude Code and Codex CLI use plugin-shipped MCP configs; Cursor users still register externally. Cross-host MCP parity for Cursor awaits Cursor-side plugin MCP support.
- **Subagent format divergence**: Claude Code and Cursor read MD agents with YAML frontmatter; Codex requires TOML. Mitigated by shipping both formats side-by-side; `test/structure/agents.bats` enforces parity between MD and TOML `developer_instructions` bodies.
