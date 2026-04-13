---
title: "Multi-Host Plugin Implementation Plan"
status: draft
tags:
  - "multi-host"
  - "plugin"
  - "roadmap"
---

## Goal

Implement multi-host support for the Archcore plugin, enabling it to run in Cursor (P1) and prepare the architecture for GitHub Copilot and other hosts (P2). The plugin must work identically across hosts with zero duplication of skills, agents, or core logic.

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
- Verify MCP config placement for Cursor plugins
- Document any gaps vs Claude Code (missing events, different capabilities)

**Output:** Verified formats, documented gaps

#### 2.2 Create `.cursor-plugin/plugin.json`

- Plugin manifest with name, version, description, author
- References to skills/, agents/, hooks/, mcp config
- Verify field names against docs from 2.1

**Files:** `.cursor-plugin/plugin.json` (new)

#### 2.3 Create `.cursor-plugin/marketplace.json`

- Marketplace listing for Cursor plugin marketplace
- Same metadata as Claude Code marketplace.json adapted to Cursor format

**Files:** `.cursor-plugin/marketplace.json` (new)

#### 2.4 Create `hooks/cursor.hooks.json`

- Map all 5 hook functions to Cursor event names
- Handle SessionStart gap: use `beforeSubmitPrompt` or rules
- Use correct Cursor stdin/stdout protocol
- Use Cursor's plugin root variable name

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
- Verify MCP server starts and tools are available
- Verify skills appear in slash command menu

#### 3.2 Test core flows

- Create a document via `/archcore:adr` — skill activates, MCP tool works
- Try direct Write to `.archcore/` — hook blocks it
- Update a document — validation and cascade hooks fire
- Run `/archcore:status` — lists documents correctly
- Run `/archcore:actualize` — staleness detection works
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
- Renamed to: `archcore-plugin`

## Acceptance Criteria

- [ ] All 5 bin scripts use `bin/lib/normalize-stdin.sh` for stdin parsing
- [ ] Claude Code plugin works identically after refactor (zero regression)
- [ ] `.cursor-plugin/plugin.json` exists with correct manifest format
- [ ] `hooks/cursor.hooks.json` maps all 5 hook functions to Cursor events
- [ ] Plugin loads in Cursor: MCP tools available, skills discoverable
- [ ] Core flow works in Cursor: create document → validate → cascade
- [ ] Direct write blocking works in Cursor
- [ ] All config formats verified against official host documentation
- [ ] No skills or agents contain host-specific references (invariant maintained)

## Dependencies

- Multi-Host Plugin Architecture ADR (`.archcore/plugin/multi-host-plugin-architecture.adr.md`) — architectural decision
- Multi-Host Compatibility Layer Specification (`.archcore/plugin/multi-host-compatibility-layer.spec.md`) — technical contract
- Hooks and Validation System Specification (`.archcore/plugin/hooks-validation-system.spec.md`) — hook semantics
- Cursor IDE installed for testing
- Cursor plugin documentation (docs.cursor.com) for format verification