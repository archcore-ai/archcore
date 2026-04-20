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
| Skills (SKILL.md) | 32    | No — use only `mcp__archcore__*`, `Read`, `Grep`, `Glob`   |
| Agents (.md)      | 2     | No — same frontmatter format, same MCP tools               |
| Bin scripts       | 5     | **Partially** — stdin JSON format varies by host           |
| hooks.json        | 1     | **Yes** — event names and matcher syntax differ            |
| Plugin manifest   | 1     | **Yes** — `.claude-plugin/plugin.json` is Claude Code only |

The only host-specific parts are: plugin manifests (~10 lines JSON each), hooks config files (~30 lines JSON each), and stdin parsing in bin scripts. The MCP server is a separate concern — it is installed via the Archcore CLI and registered by the user, not shipped with the plugin.

### Drivers

- Users of Cursor, Copilot, Codex CLI ask for Archcore integration
- Industry convergence on Agent Skills + MCP makes cross-host support low-effort
- Maintaining separate repos per host would mean duplicating 32 skills, 2 agents, and 5 bin scripts

## Decision

**Support multiple AI coding hosts from a single repository** with a shared core and thin per-host adapter layer. **The plugin does not ship an MCP server configuration** — MCP tools come from the separately-installed Archcore CLI, registered by the user via project `.mcp.json` or `claude mcp add`.

Architecture:

```
archcore-plugin/
├── skills/                      # Shared — Agent Skills standard (32 skills)
├── agents/                      # Shared — markdown agent definitions (2 agents)
├── bin/                         # Shared — hook scripts with stdin normalization
│   ├── lib/normalize-stdin.sh   # NEW: detects host format, outputs normalized JSON
│   ├── session-start
│   ├── check-archcore-write
│   ├── validate-archcore
│   ├── check-cascade
│   └── check-staleness
│
├── .claude-plugin/              # Claude Code manifest
│   ├── plugin.json
│   └── marketplace.json
├── .cursor-plugin/              # Cursor manifest
│   ├── plugin.json
│   └── marketplace.json
│
├── hooks/
│   ├── hooks.json               # Claude Code hook events
│   ├── cursor.hooks.json        # Cursor hook events
│   └── copilot.hooks.json       # GitHub Copilot hook events (future)
│
└── rules/                       # Cursor-specific rules (.mdc files, optional)
```

### Shared core principle

Skills, agents, and bin scripts are maintained once. All host-specific adapters are pure configuration — no logic duplication.

### MCP ownership boundary

MCP configuration lives outside the plugin deliberately:

- **Plugin** — ships skills, agents, hooks, and normalization logic. Host-agnostic.
- **Archcore CLI** — provides `archcore mcp` (the MCP server binary). Installed independently.
- **User / repo** — registers the MCP server in `.mcp.json` (team-shared, project-scoped) or via `claude mcp add` (user-scoped).

This separation avoids Claude Code's duplicate-MCP suppression when a repo already declares `archcore` in `.mcp.json` or the user has registered it globally. Shipping MCP in the plugin would produce a persistent "Errors (1)" in `/plugin` UI with no functional benefit, since matching command+args are deduplicated in favor of the user's registration.

### Stdin normalization approach

Bin scripts source a shared `lib/normalize-stdin.sh` that detects the host from stdin JSON structure and normalizes it to a canonical format. This avoids separate entry-point scripts per host.

Detection heuristic: each host includes distinct fields in its hook stdin JSON (e.g., Claude Code sends `tool_name` at top level; Cursor sends `hook_event_name`; Copilot sends `hookEventName`). The normalizer maps all variants to a common schema.

## Alternatives Considered

### 1. Separate repository per host

One repo for Claude Code, one for Cursor, one for Copilot. Each contains full copies of skills, agents, and bin scripts.

**Rejected because:**

- 32 skills × N hosts = massive duplication
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

**Rejected because:**

- Claude Code dedupes plugin MCP servers when their `command`/URL match a user- or project-registered server (v2.1.71+). If a repo has `.mcp.json` with `archcore`, or if the user ran `claude mcp add archcore archcore mcp`, the plugin's copy is silently suppressed and a "Errors (1)" appears in `/plugin` UI.
- Shared repos tend to have a canonical `.mcp.json` used by multiple AI tools (Cursor, Windsurf, Codex CLI, Gemini CLI); duplicating it inside the plugin adds noise with zero benefit.
- MCP lifecycle belongs to the CLI install — if the CLI is missing, the MCP server cannot run regardless of where it's declared.

## Consequences

### Positive

- **Zero skill/agent duplication**: 32 skills and 2 agents maintained in one place
- **Low per-host cost**: Adding a new host requires only a manifest (~10 lines) and hooks config (~30 lines)
- **Standard compliance**: Uses Agent Skills, MCP, and markdown agents — all open standards
- **Single source of truth**: Bug fixes in skills/agents/bin propagate to all hosts automatically
- **No MCP conflicts**: Plugin install never triggers duplicate-server warnings; users of repos with team-shared `.mcp.json` see no errors

### Negative

- **Stdin normalization complexity**: Bin scripts must handle multiple JSON formats. Mitigation: centralized normalizer (`lib/normalize-stdin.sh`) with clear format detection.
- **Testing matrix**: Must verify plugin works in each supported host. Mitigation: start with 2 hosts (Claude Code + Cursor), expand incrementally.
- **Hook event mapping is imperfect**: Not all hosts have equivalent hook events (e.g., Cursor has no direct `SessionStart` equivalent). Mitigation: use closest available event per host; document gaps per host.
- **Extra install step for users**: MCP must be registered separately (`claude mcp add archcore archcore mcp -s user`, or in project `.mcp.json`). Mitigation: `bin/session-start` emits structured guidance with the exact command when MCP is unreachable; README documents the prerequisite up front.
- **Repository naming**: ~~`archcore-claude-plugin` implies Claude Code only.~~ Resolved: renamed to `archcore-plugin`.
