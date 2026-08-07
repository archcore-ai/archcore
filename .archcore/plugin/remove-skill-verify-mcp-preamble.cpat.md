---
title: "Remove \"Step 0: Verify MCP\" Preamble from SKILL.md Files"
status: accepted
tags:
  - "plugin"
  - "skills"
---

## Pattern

Every `SKILL.md` in `skills/` used to open with a "Step 0: Verify MCP" block that halted execution when `mcp__archcore__list_documents` was unavailable and told the user to install the Archcore CLI out of band. The block was removed from every skill.

Remove the block entirely. The first real step of the skill becomes "Step 1".

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

Existing step numbering stays as it was. The removed block was always numbered from 0 while every other step started at 1.

## Scope

Every skill under `skills/`. At the time of the change the surface held 16 skills: 9 intent (`bootstrap`, `capture`, `plan`, `decide`, `standard`, `review`, `actualize`, `help`, `context`), 6 track (`product-track`, `sources-track`, `iso-track`, `architecture-track`, `standard-track`, `feature-track`), and 1 utility (`verify`). The surface is now four commands (`init`, `plan`, `document`, `review`), collapsed by `four-command-palette.adr`; the pattern applies unchanged to each of them and to any skill added later.

## Rationale

- **Wrong layer for the check.** Telling the user the CLI is not installed from inside a skill fires too late — the user already invoked the skill expecting MCP to work. The right surface is `bin/session-start`, which prints the install message at session boot pointing at https://docs.archcore.ai/cli/install/.
- **Stale install instructions inside the block.** The preamble hardcoded `curl -fsSL https://archcore.ai/install.sh | bash` and `archcore init`. Skills are read into the system prompt of every session, so embedding install commands in every skill duplicated them 16 times and created 16 places to update whenever the install path changed.
- **Context cost.** Roughly 15 lines across 16 skills is about 240 lines of boilerplate in the system-prompt surface, producing nothing whenever MCP is available, which is the common case.
- **Confused onboarding.** First-time users who saw "Archcore CLI is not installed" in skill output — for example when a skill was invoked while the host was still booting the MCP — tried to install a CLI they already had on PATH.
- **No graceful degradation was possible inside a skill.** When MCP genuinely is unavailable, the MCP tool call itself surfaces the host's "tool not found" error to the agent immediately, which is a clearer signal than a skill-level preamble.

The corollary is that the plugin must surface a missing CLI somewhere. That responsibility lives in `bin/session-start` — one place, checked at runtime — rather than in every skill.

## Sub-agent preambles are not this pattern

Both the `archcore-assistant` and `archcore-auditor` definitions retain a `# First Step — Bootstrap Knowledge Tree` preamble. That block runs no MCP availability check; it loads the recent-decisions index into the sub-agent's context, for the reasons recorded in `subagent-knowledge-tree-bootstrap.adr`. Do not remove that preamble by analogy with this pattern.

## Enforcement going forward

- A new `SKILL.md` MUST NOT include a "Verify MCP" preamble or any equivalent install check.
- `skill-file-structure.rule` is the authoritative reference for `SKILL.md` structure, and it does not name this preamble.
- When adding a skill, start the Execution section at "Step 1", or at whatever the skill's first real step is.

## Edge cases

- **Cursor users.** Cursor does not auto-register the plugin's MCP. The correct response is documented in the Cursor MCP-setup section of `README.md` and in the "MCP server not connecting" troubleshooting of `plugin-development.guide`, not in each skill.
- **Mid-session CLI install.** Claude Code registers MCP servers at session start, so installing the CLI mid-session does not reconnect a failed MCP and the user must restart the host. A skill-level preamble could never have detected or fixed this; SessionStart guidance is the right surface.
