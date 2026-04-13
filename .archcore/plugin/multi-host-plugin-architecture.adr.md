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

| Component | Count | Host-specific? |
|-----------|-------|----------------|
| Skills (SKILL.md) | 33 | No — use only `mcp__archcore__*`, `Read`, `Grep`, `Glob` |
| Agents (.md) | 2 | No — same frontmatter format, same MCP tools |
| MCP config | 1 | No — `archcore mcp` is universal |
| Bin scripts | 5 | **Partially** — stdin JSON format varies by host |
| hooks.json | 1 | **Yes** — event names and matcher syntax differ |
| Plugin manifest | 1 | **Yes** — `.claude-plugin/plugin.json` is Claude Code only |

The only host-specific parts are: plugin manifests (~10 lines JSON each), hooks config files (~30 lines JSON each), and stdin parsing in bin scripts.

### Drivers

- Users of Cursor, Copilot, Codex CLI ask for Archcore integration
- Industry convergence on Agent Skills + MCP makes cross-host support low-effort
- Maintaining separate repos per host would mean duplicating 33 skills, 2 agents, and 5 bin scripts

## Decision

**Support multiple AI coding hosts from a single repository** with a shared core and thin per-host adapter layer.

Architecture:

```
archcore-plugin/
├── skills/                      # Shared — Agent Skills standard (33 skills)
├── agents/                      # Shared — markdown agent definitions (2 agents)
├── bin/                         # Shared — hook scripts with stdin normalization
│   ├── lib/normalize-stdin.sh   # NEW: detects host format, outputs normalized JSON
│   ├── session-start
│   ├── check-archcore-write
│   ├── validate-archcore
│   ├── check-cascade
│   └── check-staleness
├── .mcp.json                    # Shared — MCP server config
│
├── .claude-plugin/              # Claude Code manifest
│   ├── plugin.json
│   └── marketplace.json
├── .cursor-plugin/              # Cursor manifest
│   ├── plugin.json
│   └── marketplace.json
│
├── hooks/
│   ├── claude-code.hooks.json   # Claude Code hook events
│   ├── cursor.hooks.json        # Cursor hook events
│   └── copilot.hooks.json       # GitHub Copilot hook events (future)
│
└── rules/                       # Cursor-specific rules (.mdc files, optional)
```

### Shared core principle

Skills, agents, MCP config, and bin scripts are maintained once. All host-specific adapters are pure configuration — no logic duplication.

### Stdin normalization approach

Bin scripts source a shared `lib/normalize-stdin.sh` that detects the host from stdin JSON structure and normalizes it to a canonical format. This avoids separate entry-point scripts per host.

Detection heuristic: each host includes distinct fields in its hook stdin JSON (e.g., Claude Code sends `tool_name` at top level; Cursor sends `hook_event_name`; Copilot sends `hookEventName`). The normalizer maps all variants to a common schema.

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

## Consequences

### Positive

- **Zero skill/agent duplication**: 33 skills and 2 agents maintained in one place
- **Low per-host cost**: Adding a new host requires only a manifest (~10 lines) and hooks config (~30 lines)
- **Standard compliance**: Uses Agent Skills, MCP, and markdown agents — all open standards
- **Single source of truth**: Bug fixes in skills/agents/bin propagate to all hosts automatically

### Negative

- **Stdin normalization complexity**: Bin scripts must handle multiple JSON formats. Mitigation: centralized normalizer (`lib/normalize-stdin.sh`) with clear format detection.
- **Testing matrix**: Must verify plugin works in each supported host. Mitigation: start with 2 hosts (Claude Code + Cursor), expand incrementally.
- **Hook event mapping is imperfect**: Not all hosts have equivalent hook events (e.g., Cursor has no direct `SessionStart` equivalent). Mitigation: use closest available event per host; document gaps per host.
- **Repository naming**: Current name `archcore-claude-plugin` implies Claude Code only. Should rename to `archcore-plugin` or similar.