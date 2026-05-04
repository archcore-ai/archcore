---
title: "Multi-Host Plugin Architecture — Single Repo for Multiple AI Coding Tools"
status: accepted
tags:
  - "architecture"
  - "multi-host"
  - "plugin"
---

## Context

The Archcore plugin currently works only in Claude Code. Research (April 2026) shows that 9+ major AI coding tools have adopted the same open standards the plugin already uses:

- **Agent Skills standard** (agentskills.io) — adopted by Cursor, GitHub Copilot, Codex CLI, Roo Code, Cline, Gemini CLI, Windsurf, JetBrains Junie, OpenHands
- **MCP (Model Context Protocol)** — adopted by all of the above plus Amazon Q, Continue.dev, Zed AI
- **Markdown agent definitions** — adopted by Cursor, GitHub Copilot, Codex CLI, Gemini CLI

Analysis of the current plugin shows that **~95% of code is already host-agnostic**:

| Component         | Count | Host-specific?                                             |
| ----------------- | ----- | ---------------------------------------------------------- |
| Skills (SKILL.md) | 33    | No — use only `mcp__archcore__*`, `Read`, `Grep`, `Glob`   |
| Agents (.md)      | 2     | No — same frontmatter format, same MCP tools               |
| Bin scripts       | 5     | **Partially** — stdin JSON format varies by host           |
| hooks.json        | 1     | **Yes** — event names and matcher syntax differ            |
| Plugin manifest   | 1     | **Yes** — `.claude-plugin/plugin.json` is Claude Code only |

The only host-specific parts are: plugin manifests (~10 lines JSON each), hooks config files (~30 lines JSON each), and stdin parsing in bin scripts. The MCP server is a separate concern — it is installed via the Archcore CLI and registered by the user, not shipped with the plugin.

### Drivers

- Users of Cursor, Copilot, Codex CLI ask for Archcore integration — Codex CLI v0.117.0+ (March 2026) added a plugin system with near-1:1 surface to Claude Code, validating the third-host bet.
- Industry convergence on Agent Skills + MCP makes cross-host support low-effort
- Maintaining separate repos per host would mean duplicating 33 skills, 2 agents, and 5 bin scripts

## Decision

**Support multiple AI coding hosts from a single repository** with a shared core and thin per-host adapter layer.

At the time of this decision, the plugin did not ship an MCP server configuration — MCP tools came from the separately-installed Archcore CLI, registered by the user via project `.mcp.json` or `claude mcp add`. **This stance has since been superseded for Claude Code and Codex CLI** (see the Bundled CLI Launcher ADR): the plugin now ships a launcher (`bin/archcore{,.cmd,.ps1}` + `bin/CLI_VERSION`) that resolves the CLI on demand, plus host-specific MCP configs that register `archcore` against that launcher. The shared-core / per-host-adapter split described below remains the governing architecture; the CLI-install / MCP-registration sub-decision evolved.

Architecture (as originally decided, with an addendum below for the current MCP wiring):

```
plugin/
├── skills/                      # Shared — Agent Skills standard (17 skills)
├── agents/                      # Shared — markdown agent definitions (2 agents) + Codex TOML variants
│   ├── archcore-assistant.md    # Claude Code / Cursor
│   ├── archcore-assistant.toml  # Codex CLI
│   ├── archcore-auditor.md      # Claude Code / Cursor
│   └── archcore-auditor.toml    # Codex CLI (sandbox_mode = "read-only" + disabled_tools)
├── bin/                         # Shared — hook scripts with stdin normalization
│   ├── lib/normalize-stdin.sh   # Detects host format (claude-code/cursor/copilot/codex), outputs normalized JSON
│   ├── archcore                 # Cross-platform CLI launcher (POSIX)
│   ├── archcore.cmd             # Windows shim
│   ├── archcore.ps1             # Windows launcher (PowerShell)
│   ├── CLI_VERSION              # Pinned CLI semver
│   ├── session-start
│   ├── check-archcore-write
│   ├── check-code-alignment
│   ├── validate-archcore
│   ├── check-cascade
│   ├── check-precision
│   └── check-staleness
│
├── .claude-plugin/              # Claude Code manifest
│   ├── plugin.json
│   └── marketplace.json
├── .cursor-plugin/              # Cursor manifest
│   ├── plugin.json
│   └── marketplace.json
├── .codex-plugin/               # Codex CLI manifest
│   └── plugin.json
├── .agents/plugins/             # Codex marketplace descriptor
│   └── marketplace.json
│
├── hooks/
│   ├── hooks.json               # Claude Code hook events (PascalCase)
│   ├── cursor.hooks.json        # Cursor hook events (camelCase)
│   ├── codex.hooks.json         # Codex CLI hook events (PascalCase, mirrors Claude Code)
│   └── copilot.hooks.json       # GitHub Copilot hook events (future)
│
├── .mcp.json                    # Plugin-shipped MCP for Claude Code
├── .codex.mcp.json              # Plugin-shipped MCP for Codex CLI
└── rules/                       # Cursor-specific rules (.mdc files, optional)
```

**Current addendum (see Bundled CLI Launcher ADR)**: `bin/` additionally ships `archcore`, `archcore.cmd`, `archcore.ps1`, and `CLI_VERSION` — a cross-platform launcher that resolves the Archcore CLI binary. The plugin root ships `.mcp.json` for Claude Code and `.codex.mcp.json` for Codex CLI, with `.codex-plugin/plugin.json` pointing at the Codex-specific file through `mcpServers`. Both MCP configs register the same launcher. Cursor users still register MCP externally.

### Shared core principle

Skills, agents, bin scripts, and the CLI launcher are maintained once. All host-specific adapters are pure configuration — no logic duplication.

### MCP ownership boundary (original rationale — now partially superseded)

At the time of this ADR, MCP configuration lived outside the plugin deliberately:

- **Plugin** — shipped skills, agents, hooks, and normalization logic. Host-agnostic.
- **Archcore CLI** — provided `archcore mcp` (the MCP server binary). Installed independently.
- **User / repo** — registered the MCP server in `.mcp.json` (team-shared, project-scoped) or via `claude mcp add` (user-scoped).

The rationale was to avoid Claude Code's duplicate-MCP suppression when a repo already declared `archcore` in `.mcp.json` or the user had registered it globally. Shipping MCP in the plugin would have produced a persistent "Errors (1)" in `/plugin` UI with no functional benefit.

**Current state (per Bundled CLI Launcher ADR)**: the plugin ships MCP registration for Claude Code and Codex CLI via host-specific config files. Claude Code consumes plugin-root `.mcp.json` pointing at `${CLAUDE_PLUGIN_ROOT}/bin/archcore mcp`; Codex consumes plugin-root `.codex.mcp.json` via the manifest pointer and points at plugin-relative `./bin/archcore mcp`. Duplicate suppression is no longer a blocker because the launcher defers to an existing global `archcore` on `PATH` — the effective command resolved is identical to what a user-registered server would resolve to, so deduping is benign. Cursor users still register MCP externally.

### Stdin normalization approach

Bin scripts source a shared `lib/normalize-stdin.sh` that detects the host from stdin JSON structure and normalizes it to a canonical format. This avoids separate entry-point scripts per host.

Detection heuristic: each host includes distinct fields in its hook stdin JSON (e.g., Claude Code sends `tool_name` at top level; Cursor sends `conversation_id`; Copilot sends `hookEventName`; Codex sends `turn_id` in turn-scoped events). The normalizer maps all variants to a common schema. Codex shares Claude Code's snake_case stdin schema, so the field-extraction logic for the `codex` branch mirrors `claude-code`.

## Alternatives Considered

### 1. Separate repository per host

One repo for Claude Code, one for Cursor, one for Copilot. Each contains full copies of skills, agents, and bin scripts.

**Rejected because:**

- 33 skills × N hosts = massive duplication
- Any skill update must be synced across all repos
- Agents and bin scripts also duplicated
- Only ~5% of code is actually host-specific

### 2. Build system that generates per-host packages

A mono-repo with a build step (e.g., Node.js script) that reads a canonical source and generates separate plugin directories per host.

**Rejected because:**

- Introduces build tooling to a project that is currently pure Markdown + Shell
- Complexity not warranted — the per-host differences are purely configuration (JSON files)
- Agent Skills standard already ensures skills work across hosts without transformation
- Would complicate local development and testing

### 3. Symlinks from host-specific directories to shared source

Each host's expected directory (`.cursor/skills/`, `.github/skills/`) symlinks to the shared `skills/` directory.

**Rejected because:**

- Symlinks don't work reliably on Windows
- Plugin marketplace systems distribute files, not symlinks
- Fragile when cloned or copied

### 4. Ship MCP config inside the plugin

Ship `.mcp.json` (Claude Code) and `mcp.json` (Cursor) at the plugin root so MCP "just works" after install.

**Originally rejected because:**

- Claude Code dedupes plugin MCP servers when their `command`/URL match a user- or project-registered server (v2.1.71+). If a repo had `.mcp.json` with `archcore`, or if the user ran `claude mcp add archcore archcore mcp`, the plugin's copy would be silently suppressed and "Errors (1)" would appear in `/plugin` UI.
- Shared repos tend to have a canonical `.mcp.json` used by multiple AI tools (Cursor, Windsurf, Codex CLI, Gemini CLI); duplicating it inside the plugin added noise with zero benefit.
- MCP lifecycle belonged to the CLI install — if the CLI was missing, the MCP server couldn't run regardless of where it was declared.

**Status: reversed for Claude Code and Codex CLI** (see Bundled CLI Launcher ADR). The dedup concern was resolved by introducing the bundled launcher. Claude Code's `.mcp.json` points at `${CLAUDE_PLUGIN_ROOT}/bin/archcore`; Codex's `.codex.mcp.json` points at `./bin/archcore`. Both resolve through the same launcher, which defers to a global `archcore` on `PATH` when one exists. Deduping still happens but becomes benign. The friction of requiring an explicit `claude mcp add` / `codex mcp add` step proved worse than the duplicate-registration concern.

## Consequences

### Positive

- **Zero skill/agent duplication**: skills and agents maintained in one place
- **Low per-host cost**: Adding a new host requires only a manifest (~10 lines) and hooks config (~30 lines). Codex was the first real test — port took ~1 dev-day for scaffolding plus tests, validating the architecture's "low per-host cost" claim.
- **Standard compliance**: Uses Agent Skills, MCP, and markdown agents — all open standards
- **Single source of truth**: Bug fixes in skills/agents/bin/launcher propagate to all hosts automatically

### Negative

- **Stdin normalization complexity**: Bin scripts must handle multiple JSON formats. Mitigation: centralized normalizer (`lib/normalize-stdin.sh`) with clear format detection (claude-code, cursor, copilot, codex).
- **Testing matrix**: Must verify plugin works in each supported host. Mitigation: started with 2 hosts (Claude Code + Cursor), expanded to Codex CLI; expand incrementally for the rest.
- **Hook event mapping is imperfect**: Not all hosts have equivalent hook events (e.g., Cursor has no direct `SessionStart` equivalent). Codex shares Claude Code's PascalCase event names (`SessionStart`, `PreToolUse`, `PostToolUse`) so its hooks config is a near-copy. Mitigation: use closest available event per host; document gaps per host.
- **MCP wiring is host-specific**: Claude Code and Codex CLI use plugin-shipped `.mcp.json`; Cursor users still register externally. Cross-host MCP parity for Cursor awaits Cursor-side plugin MCP support with path substitution. Mitigation: the launcher is host-agnostic — only the "who points at the launcher" differs per host.
- **Subagent format divergence**: Claude Code and Cursor read MD agents with YAML frontmatter; Codex requires TOML. Mitigation: ship both formats side-by-side in `agents/` (`archcore-{assistant,auditor}.md` and `.toml`); shared `developer_instructions` body kept in sync via tests.
- **Repository naming**: ~~`archcore-claude-plugin` implies Claude Code only.~~ ~~Renamed to `archcore-plugin`.~~ Resolved: now `archcore-ai/plugin` — the org name carries the brand and the repo name is host-agnostic.
