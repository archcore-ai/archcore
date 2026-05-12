---
title: "Remove \"Step 0: Verify MCP\" Preamble from SKILL.md Files"
status: accepted
tags:
  - "plugin"
  - "skills"
---

## Pattern

Every SKILL.md in `skills/` used to begin with a "Step 0: Verify MCP" block that halted execution if `mcp__archcore__list_documents` was unavailable and told the user to install the Archcore CLI out-of-band. The block was removed across all skills.

Remove the block entirely from all SKILL.md files. The first real step of the skill becomes "Step 1".

## Before

```markdown
## Execution

### Step 0: Verify MCP

Check if `mcp__archcore__list_documents` exists in your available tools. If the tool does not exist or returns an error, **stop immediately** and tell the user:

**Archcore CLI is not installed.** The plugin provides skills and hooks, but document operations need the CLI (it runs the MCP server).

To set up:
1. Install: `curl -fsSL https://archcore.ai/install.sh | bash`
2. Initialize project: `archcore init`
3. Restart the session, then rerun this command.

Do not proceed without MCP tools. Do not write to `.archcore/` directly.

### Step 1: Gather data
...
```

## After

```markdown
## Execution

### Step 1: Gather data
...
```

Existing step numbering stays as-is; the block was always numbered from 0 while everything else started at 1.

## Scope

All skills under `skills/` — currently 16: 9 intent (`bootstrap`, `capture`, `plan`, `decide`, `standard`, `review`, `actualize`, `help`, `context`), 6 track (`product-track`, `sources-track`, `iso-track`, `architecture-track`, `standard-track`, `feature-track`), 1 utility (`verify`).

## Rationale

- **Wrong layer for the check.** Telling the user "the CLI is not installed" inside a skill description fires too late — the user already invoked the skill expecting MCP to work. The right place to surface a missing CLI is `bin/session-start`, which now prints the install message at session boot pointing at https://docs.archcore.ai/cli/install/.
- **Stale install instructions inside the block.** The preamble hard-coded `curl -fsSL https://archcore.ai/install.sh | bash` and `archcore init`. Skills are read into the system prompt of every session; embedding install commands inside every skill duplicates them 16+ times and creates 16+ places to update when the install path changes.
- **Wastes context tokens.** ~15 lines × 16 skills = ~240 lines of boilerplate in the system prompt surface that produces zero value when MCP is actually available (the common case).
- **Confuses onboarding.** First-time users who saw "Archcore CLI is not installed" inside a skill output (e.g., when a skill was invoked while the host was still booting the MCP) mistakenly tried to install a CLI they already had on PATH.
- **No graceful degradation possible inside a skill.** When MCP genuinely is unavailable, the MCP tool call itself surfaces the host's "tool not found" error to the agent immediately — clearer signal than a skill-level preamble.

The corollary is that the plugin must surface CLI-missing **somewhere** — that responsibility lives in `bin/session-start` (one place, runtime-checked), not inside every skill (16 places, statically embedded).

## Sub-agent preambles are NOT this pattern

Both `archcore-assistant` and `archcore-auditor` subagent definitions retain a `# First Step — Bootstrap Knowledge Tree` preamble. That block does NOT do an MCP availability check — it loads the recent decisions index into the subagent's context. The motivation is documented in `subagent-knowledge-tree-bootstrap.adr`. Do not remove that preamble by analogy with this CPAT.

## Enforcement going forward

- New SKILL.md files MUST NOT include a "Verify MCP" or similar install-check preamble.
- The skill-file-structure rule is the authoritative reference for SKILL.md structure — it does not mention this preamble.
- When adding a new skill: start the Execution section with "Step 1: ..." (or whatever the skill's first real step is). Do not reintroduce the block.

## Edge cases

- **Cursor users**: Cursor does not auto-register the plugin's MCP. The correct response is documented in the Cursor-specific MCP-setup section of the README (`cursor.mcp.json` template) and in `plugin-development.guide` ("MCP server not connecting" troubleshooting), not in each skill.
- **Mid-session CLI install**: Claude Code MCP servers register at session start. Installing the CLI mid-session does NOT reconnect a failed MCP — the user must restart the host. The skill-level preamble could never have detected or fixed this; SessionStart guidance is the right surface.
