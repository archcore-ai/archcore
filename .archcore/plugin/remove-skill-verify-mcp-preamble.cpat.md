---
title: "Remove \"Step 0: Verify MCP\" Preamble from SKILL.md Files"
status: accepted
tags:
  - "plugin"
  - "skills"
---

## Pattern

Every SKILL.md in `skills/` used to begin with a "Step 0: Verify MCP" block that halted execution if `mcp__archcore__list_documents` was unavailable and told the user to install the Archcore CLI out-of-band. With the bundled CLI launcher + plugin-shipped `.mcp.json` (see Bundled CLI Launcher ADR), MCP is always registered on plugin load, and the first tool call auto-resolves the CLI. The preamble is now dead weight — it never triggers, it confuses first-time users, and it misstates the install procedure.

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

Call in parallel:
- `mcp__archcore__list_documents` ...
```

## After

```markdown
## Execution

### Step 1: Gather data

Call in parallel:
- `mcp__archcore__list_documents` ...
```

Existing step numbering stays as-is; the block was always numbered from 0 while everything else started at 1.

## Scope

All skills under `skills/`. The sweep covers 32+ SKILL.md files:

- Intent skills (8): `capture`, `plan`, `decide`, `standard`, `review`, `status`, `actualize`, `help`
- Track skills (6): `product-track`, `sources-track`, `iso-track`, `architecture-track`, `standard-track`, `feature-track`
- Document-type skills (18): `adr`, `rfc`, `rule`, `guide`, `doc`, `spec`, `prd`, `idea`, `plan`, `mrd`, `brd`, `urd`, `brs`, `strs`, `syrs`, `srs`, `task-type`, `cpat`
- Utility (1): `verify`

Not every skill had the block — some document-type skills use a different structure. The pattern is "remove where present, don't add elsewhere."

## Rationale

- **No-op at runtime.** MCP is guaranteed available: Claude Code registers the plugin's `.mcp.json` at load, and the `command` resolves through `bin/archcore` which downloads the CLI on first use if missing. The "tool not found" branch the block was written for cannot occur in the normal install path.
- **Wrong install instructions.** The block tells users to `curl | bash` — no longer the recommended path. With the launcher, no CLI install is required at all for Claude Code users.
- **Wastes context tokens.** ~15 lines × 30+ skills = ~450 lines of boilerplate in the system prompt surface that never does anything useful.
- **Confuses onboarding.** First-time users who see "Archcore CLI is not installed" in the skill output (e.g., if a skill is invoked before plugin fully loads) mistakenly try to install a CLI they don't need.

## Rollout

Already applied in commit `5b5cb24` ("feat: auto install archcore without CLI"). No further action required on existing SKILL.md files.

## Enforcement going forward

- New SKILL.md files MUST NOT include a "Verify MCP" or similar install-check preamble.
- The skill-file-structure rule is the authoritative reference for SKILL.md structure — it should not mention this preamble.
- When adding a new skill: start the Execution section with "Step 1: ..." (or whatever the skill's first real step is). Do not reintroduce the block.
- The `adding-document-type-skill` guide should match this shape.

## Edge cases

- **Cursor users**: Cursor does not auto-register the plugin's MCP. But the correct response is documented in the Cursor-specific MCP-setup docs (README + plugin-development.guide.md), not in each skill. If a Cursor user hasn't registered MCP, the MCP tool call itself will fail with a clear error — no preamble needed.
- **Offline / enterprise environments**: handled by `ARCHCORE_BIN` / `ARCHCORE_SKIP_DOWNLOAD=1`, documented in the README and plugin-development.guide.md. Not a per-skill concern.
- **Truly broken CLI (checksum mismatch, corrupt cache)**: the MCP tool call surfaces the launcher's stderr to the agent, which is a clearer signal than a skill-level preamble would be.
