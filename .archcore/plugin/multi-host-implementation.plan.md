---
title: "Multi-Host Plugin Implementation Plan"
status: accepted
tags:
  - "multi-host"
  - "plugin"
  - "roadmap"
---

## Goal

Implement multi-host support for the Archcore plugin, enabling it to run in Cursor (P1) and prepare the architecture for GitHub Copilot and other hosts (P2). The plugin must work identically across hosts with zero duplication of skills, agents, or core logic.

**MCP scope note** — at the time this plan was drafted, MCP server configuration was explicitly out of scope: the plugin did not declare `mcpServers` anywhere and did not ship `.mcp.json` at the plugin root. That scope boundary was revised once for Claude Code (the long-since-removed bundled launcher, Phase 5 below) and then again when the launcher was removed in v0.4.0. The plugin currently ships `.mcp.json` and `.codex.mcp.json` at its root, both pointing at `archcore` on PATH. Cursor still relies on user-registered MCP via `~/.cursor/mcp.json` or project-scoped `.cursor/mcp.json`.

## Tasks

### Phase 1: Stdin Normalization Layer

Create the shared normalization library that makes bin scripts host-agnostic.

#### 1.1 Create `bin/lib/normalize-stdin.sh`

- POSIX shell library sourced by all bin scripts
- Reads stdin once, stores in `$ARCHCORE_RAW_STDIN`
- Detects host from JSON structure (Claude Code vs Cursor vs Copilot)
- Exports normalized variables: `ARCHCORE_HOST`, `ARCHCORE_HOOK_EVENT`, `ARCHCORE_TOOL_NAME`, `ARCHCORE_FILE_PATH`, `ARCHCORE_TOOL_INPUT`
- Provides output helpers: `archcore_hook_info()`, `archcore_hook_block()`, `archcore_hook_allow()`
- No external dependencies (no `jq` — only `grep`/`sed`)

**Files:** `bin/lib/normalize-stdin.sh` (new)

#### 1.2 Refactor existing bin scripts to use normalizer

Update all 5 bin scripts to source the normalizer instead of parsing stdin directly.

- `bin/check-archcore-write` — replace inline `grep`/`sed` with `$ARCHCORE_FILE_PATH`, use `archcore_hook_block()` for output
- `bin/validate-archcore` — use `$ARCHCORE_TOOL_NAME` and `$ARCHCORE_FILE_PATH`, use `archcore_hook_info()` for output
- `bin/check-cascade` — use `$ARCHCORE_TOOL_INPUT` for document path, use `archcore_hook_info()` for output
- `bin/check-staleness` — no stdin changes needed (called from session-start, not directly from hook)
- `bin/session-start` — minimal changes (may use `$ARCHCORE_HOST` for host-specific CLI command)

**Files:** `bin/check-archcore-write`, `bin/validate-archcore`, `bin/check-cascade`, `bin/session-start` (modify)

#### 1.3 Verify Claude Code still works

- Run full test: session start, create document, block direct write, validate, cascade detection
- Ensure zero regression — normalizer defaults to Claude Code format

**Verification:** Manual test in Claude Code session

### Phase 2: Cursor Plugin

Create all Cursor-specific adapter files.

#### 2.1 Research and verify Cursor plugin formats

- Fetch latest Cursor docs for plugin.json manifest schema
- Fetch latest Cursor docs for hooks.json format (event names, stdin/stdout protocol)
- Fetch latest Cursor docs for rules .mdc format
- Document any gaps vs Claude Code (missing events, different capabilities)

**Output:** Verified formats, documented gaps

#### 2.2 Create `.cursor-plugin/plugin.json`

- Plugin manifest with name, version, description, author
- References to skills/, agents/, hooks/, rules/
- No `mcpServers` field — MCP is registered externally by the user/repo
- Verify field names against docs from 2.1

**Files:** `.cursor-plugin/plugin.json` (new)

#### 2.3 Create `.cursor-plugin/marketplace.json`

- Marketplace listing for Cursor plugin marketplace
- Same metadata as Claude Code marketplace.json adapted to Cursor format

**Files:** `.cursor-plugin/marketplace.json` (new)

#### 2.4 Create `hooks/cursor.hooks.json`

- Map all active hook functions to Cursor event names (sessionStart, preToolUse Write, afterMCPExecution running validate-archcore + check-cascade)
- Handle SessionStart gap: use `beforeSubmitPrompt` or rules
- Use correct Cursor stdin/stdout protocol
- Use Cursor's plugin root variable name
- Do NOT register a `postToolUse Write` validate-archcore entry — that path was removed (PreToolUse already blocks; PostToolUse on every Write would fork a shell repo-wide for no benefit)

**Files:** `hooks/cursor.hooks.json` (new)

#### 2.5 Rename `hooks/hooks.json` → `hooks/claude-code.hooks.json`

- Rename existing hooks file to be host-specific
- Update any references (`.claude-plugin/plugin.json`, `.claude/settings.json`)
- Verify Claude Code plugin system reads from the new path

**Files:** `hooks/hooks.json` → `hooks/claude-code.hooks.json` (rename), update references

#### 2.6 Create Cursor rules (optional enhancement)

- `rules/archcore-context.mdc` — alwaysApply rule with document type reference and MCP tool names (replaces SessionStart context injection)
- `rules/archcore-files.mdc` — glob-scoped rule for `.archcore/**` files, reminds about MCP-only

**Files:** `rules/archcore-context.mdc`, `rules/archcore-files.mdc` (new)

#### 2.7 Update normalize-stdin.sh for Cursor format

- Add Cursor host detection (check for `hook_event_name` field)
- Map Cursor stdin fields to normalized variables
- Implement Cursor output format in helper functions
- Test with sample Cursor hook stdin JSON

**Files:** `bin/lib/normalize-stdin.sh` (update)

### Phase 3: Verification in Cursor

#### 3.1 Install plugin locally in Cursor

- Use Cursor's local plugin loading mechanism
- Verify the user-registered MCP server is reachable (via project `mcp.json` or Cursor's MCP settings) and its tools are available
- Verify skills appear in slash command menu

#### 3.2 Test core flows

- Create a document via `/archcore:decide` — skill activates, MCP tool works
- Try direct Write to `.archcore/` — hook blocks it
- Update a document — validation and cascade hooks fire
- Run `/archcore:audit` — lists documents correctly (default dashboard mode)
- Run `/archcore:audit --drift` — staleness detection works
- Invoke archcore-assistant agent — complex task works

#### 3.3 Document findings and fix issues

- Record any Cursor-specific behavior differences
- Fix hook format issues discovered during testing
- Update spec if Cursor's actual behavior differs from documented behavior

### Phase 4: Repository Cleanup

#### 4.1 Update documentation

- Update README.md with multi-host installation instructions
- Add "Supported Hosts" section

#### 4.2 ~~Consider repository rename~~ Done

- ~~Current: `archcore-claude-plugin`~~
- ~~Renamed to: `archcore-plugin`~~
- Final name: `archcore-ai/plugin` — the org carries the brand, the repo name is host-agnostic.

### Phase 5: ~~Bundled CLI Launcher and Plugin-Owned MCP (Claude Code)~~ — Superseded and removed in v0.4.0

**Status: rejected/removed.** Phase 5 shipped a download-on-first-use bundled launcher (`bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`, `bin/CLI_VERSION`) plus plugin-owned MCP registration. The launcher caused eight bug classes — offline failures in CI, version coupling to plugin releases, cache pollution across hosts, first-run latency, enterprise friction, security patch lag, plugin bloat, and uneven host support (Cursor users still did manual MCP setup). It was fully removed in plugin v0.4.0 (commit `2f99997`).

The plugin-owned MCP shape **survived** the rollback — `.mcp.json` and `.codex.mcp.json` still ship at the plugin root, but they now point at `archcore` on PATH directly with no launcher indirection. Cursor users still register MCP externally via `~/.cursor/mcp.json` or `.cursor/mcp.json`; the plugin ships `cursor.mcp.json` as a template they copy.

See:

- `bundled-cli-launcher.adr` — original decision (status: rejected/superseded).
- `remove-bundled-launcher-global-cli.idea` — replacement decision (status: accepted), with the eight-bug-classes analysis and the one-time-install trade-off.
- `stack-and-tooling.rule` — pin: no plugin-side download-on-first-use mechanisms without a fresh ADR.

The Phase 5 subtask checklist (5.1–5.9) is preserved below as a historical archive of what was built and then removed. None of it represents current plugin state.

<details>
<summary>Historical Phase 5 subtasks (rolled back)</summary>

- 5.1 `bin/CLI_VERSION` — pinned semver of the CLI release the plugin shipped against. **Removed.**
- 5.2 `bin/archcore` — POSIX launcher with resolution order `$ARCHCORE_BIN` → PATH → cache → download. **Removed.**
- 5.3 `bin/archcore.cmd` + `bin/archcore.ps1` — Windows launcher pair with same resolution. **Removed.**
- 5.4 `.mcp.json` at plugin root pointing at `${CLAUDE_PLUGIN_ROOT}/bin/archcore mcp`. **Replaced** with `"command": "archcore"` resolving via PATH.
- 5.5 `bin/session-start` rewired through the launcher with `ARCHCORE_SKIP_DOWNLOAD=1`. **Reverted** to direct `archcore` invocation with a missing-CLI install-message fallback.
- 5.6 `bin/validate-archcore` + `bin/check-cascade` rewired through `"$SCRIPT_DIR/archcore"`. **Reverted** to direct `archcore` invocation.
- 5.7 "Step 0: Verify MCP" removed from all SKILL.md files (`remove-skill-verify-mcp-preamble.cpat`). **Kept** — the removal was correct on its own merits; subagent preambles that load knowledge tree context were retained for a different reason (see `subagent-knowledge-tree-bootstrap.adr`).
- 5.8 `test/unit/launcher.bats`, `test/structure/cli-contract.bats`, `test/structure/cli-allowlist-consistency.bats`. **Removed.**
- 5.9 README "Offline / BYO CLI" section + first-MCP-call download note. **Removed**; Prerequisites section now links to https://docs.archcore.ai/cli/install/.

</details>

## Acceptance Criteria

- [x] All 5 bin scripts use `bin/lib/normalize-stdin.sh` for stdin parsing
- [x] Claude Code plugin works identically after refactor (zero regression)
- [x] `.cursor-plugin/plugin.json` exists with correct manifest format and no `mcpServers` field
- [x] `hooks/cursor.hooks.json` maps the active hook functions to Cursor events (sessionStart, preToolUse Write, afterMCPExecution running validate-archcore + check-cascade) and contains no postToolUse entry
- [x] Plugin loads in Cursor: skills discoverable, user-registered MCP tools available
- [x] Core flow works in Cursor: create document → validate → cascade
- [x] Direct write blocking works in Cursor
- [x] All config formats verified against official host documentation
- [x] No skills or agents contain host-specific references (invariant maintained)
- [x] `bin/session-start` emits actionable guidance when `.archcore/` is missing (routes through `mcp__archcore__init_project`)
- [x] ~~`bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`, `bin/CLI_VERSION` exist~~ **Reverted — launcher removed in v0.4.0.**
- [x] ~~Launcher downloads are SHA-256 verified against `checksums.txt`~~ **Reverted — no plugin-side downloads.**
- [x] `.mcp.json` at plugin root registers `archcore` — now points at PATH-resolved `archcore` directly, not the (removed) launcher.
- [x] ~~`bin/session-start` passes `ARCHCORE_SKIP_DOWNLOAD=1` to the launcher~~ **Reverted — no launcher; session-start invokes `archcore` directly and emits install guidance if missing.**
- [x] All SKILL.md files have the "Step 0: Verify MCP" block removed
- [x] ~~`test/unit/launcher.bats` covers launcher resolution and failure modes~~ **Reverted — file removed.**
- [x] Users with a global `archcore` on `PATH` experience no behavior change — invariant strengthened: PATH install is now the only supported path.

## Dependencies

- Multi-Host Plugin Architecture ADR (`.archcore/plugin/multi-host-plugin-architecture.adr.md`) — architectural decision for the shared-core / per-host split
- ~~Bundled CLI Launcher ADR~~ (`.archcore/plugin/bundled-cli-launcher.adr.md`) — **rejected/superseded;** see `remove-bundled-launcher-global-cli.idea` for the replacement decision.
- Multi-Host Compatibility Layer Specification (`.archcore/plugin/multi-host-compatibility-layer.spec.md`) — technical contract
- Hooks and Validation System Specification (`.archcore/plugin/hooks-validation-system.spec.md`) — hook semantics
- Cursor IDE installed for testing
- Cursor plugin documentation (docs.cursor.com) for format verification
- Archcore CLI installed on PATH per https://docs.archcore.ai/cli/install/ (no plugin-side fetching)
