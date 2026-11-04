---
title: "Junie Receives MCP and the Instruction Block From archcore init and Skills From the Existing Marketplace as a Junie Extension"
status: accepted
tags:
  - "architecture"
  - "component:cli"
  - "integrations"
  - "jetbrains"
  - "multi-host"
---

## Context

Junie (docs dated 2026-10-30, junie.jetbrains.com/docs) reads MCP servers from `.junie/mcp/mcp.json` in the project and `~/.junie/mcp/mcp.json` for the user, in `{"mcpServers": {...}}` shape, shared by the IDE plugin and Junie CLI. It reads `.junie/AGENTS.md` exclusively when that file exists, and the root `AGENTS.md` otherwise. It loads skills from `.junie/skills/`, `~/.junie/skills/`, and extensions. Junie CLI installs extensions from marketplaces in the Claude plugin format (`.claude-plugin/marketplace.json`) through the interactive `/extensions` command. Junie hooks are Early Access, run only in the TUI and batch hosts, and do not run under ACP, which AI Chat uses.

Copying skill files into `.junie/skills` from the CLI would be a new distribution mechanism under `stack-and-tooling.rule` item 24, and the copies would go stale between releases.

## Decision

`archcore init` gains a `junie` agent that writes the `archcore` entry to `.junie/mcp/mcp.json` and the managed block to `.junie/AGENTS.md` when that file exists, else to the root `AGENTS.md`. Skills reach Junie as an extension installed from the existing `archcore-plugins` marketplace. The CLI delivers no skill files and runs no Junie install command; `archcore plugin` prints the `/extensions` instruction for Junie, as it does for Cursor. Archcore installs no Junie hooks.

## Alternatives Considered

1. The CLI copies skills into `.junie/skills/` — rejected because it adds a distribution mechanism and stale copies.
2. Junie hooks from the extension's `hooks/hooks.json` — rejected for now because hooks are Early Access and do not run in AI Chat over ACP.

## Consequences

- Positive: no new file-copy path; skills update with the marketplace.
- Positive: the instruction block lands in the file Junie reads.
- Tradeoff: Junie gets no write guard; the support document states the gap.
- Tradeoff: [assumption] Junie installs the `archcore-plugins` marketplace entry unchanged; a probe confirms it before release.

## Superseded when

- Junie hooks leave Early Access and run under ACP.
- Junie CLI gains a non-interactive extension install command.
