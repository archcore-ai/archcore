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

**MCP scope note** — at the time this plan was drafted, MCP server configuration was explicitly out of scope: the plugin did not declare `mcpServers` anywhere and did not ship `.mcp.json` at the plugin root. That scope boundary has since been revised for Claude Code (see Phase 5 below and the Bundled CLI Launcher ADR). The plugin now ships `.mcp.json` at its root pointing at a bundled cross-platform launcher that resolves the CLI on demand. Cursor and other hosts still rely on user-registered MCP — the launcher is host-agnostic, only the plugin-level MCP wiring is host-specific.

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
- ~~Document the external MCP registration step prominently (prerequisite, not an afterthought)~~ → superseded by Phase 5: Claude Code no longer requires external MCP registration. Keep external-registration docs for Cursor.

#### 4.2 ~~Consider repository rename~~ Done

- ~~Current: `archcore-claude-plugin`~~
- Renamed to: `archcore-plugin`

### Phase 5: Bundled CLI Launcher and Plugin-Owned MCP (Claude Code)

Eliminate the out-of-band CLI install and MCP registration steps for Claude Code users by shipping a cross-platform launcher and plugin-provided MCP registration. Rationale and trade-offs: see the Bundled CLI Launcher ADR.

#### 5.1 Create `bin/CLI_VERSION`

- Single-line file containing the pinned semver of the Archcore CLI release the plugin is tested against.
- Launchers read this for the cache key (`archcore-v${VERSION}`) and the GitHub Releases download URL.

**Files:** `bin/CLI_VERSION` (new)

#### 5.2 Create `bin/archcore` (POSIX launcher)

- POSIX shell script. Resolution order: `$ARCHCORE_BIN` → `archcore` on `PATH` (with loop guard) → plugin-managed cache (`<cache>/archcore-v${VERSION}`) → download from GitHub Releases.
- Cache directory first-match: `$CLAUDE_PLUGIN_DATA/archcore/cli` → `$XDG_DATA_HOME/archcore-plugin/cli` → `$HOME/.local/share/archcore-plugin/cli`.
- Download path: `github.com/archcore-ai/cli/releases/download/v${VERSION}/archcore_<os>_<arch>.tar.gz`. Verify SHA-256 against `checksums.txt`. Atomic stage-then-rename into cache. Abort on checksum mismatch.
- Honor `ARCHCORE_SKIP_DOWNLOAD=1` to disable step 4 (used by `bin/session-start`).
- `exec` the resolved binary so exit code + stdio pass through unchanged.

**Files:** `bin/archcore` (new)

#### 5.3 Create `bin/archcore.cmd` and `bin/archcore.ps1` (Windows launcher)

- `archcore.cmd` is a one-line shim: `powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "%~dp0archcore.ps1" %*`.
- `archcore.ps1` implements the same resolution order as the POSIX launcher.
- Cache dirs: `$env:CLAUDE_PLUGIN_DATA\archcore\cli` → `$env:LOCALAPPDATA\archcore-plugin\cli`.
- Use `Invoke-WebRequest` with retry; `Get-FileHash -Algorithm SHA256` for verification.
- Call `Unblock-File` on staged binary to strip MOTW (prevents SmartScreen prompts).
- Architecture detection uses `[System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture` so x64 PowerShell under ARM64 emulation still picks the ARM64 asset.

**Files:** `bin/archcore.cmd`, `bin/archcore.ps1` (new)

#### 5.4 Ship plugin-root `.mcp.json` for Claude Code

```json
{
  "mcpServers": {
    "archcore": {
      "command": "${CLAUDE_PLUGIN_ROOT}/bin/archcore",
      "args": ["mcp"]
    }
  }
}
```

- Claude Code auto-registers this on plugin load.
- Duplicate suppression is acceptable: the launcher defers to any existing global `archcore` on `PATH`, so deduping produces no behavior divergence.

**Files:** `.mcp.json` (new, plugin root)

#### 5.5 Rewire `bin/session-start` for launcher + skip-download

- Source the normalizer.
- If `.archcore/` missing, emit init guidance pointing the agent at `mcp__archcore__init_project` (no manual steps).
- Otherwise: invoke `${SCRIPT_DIR}/archcore` with `ARCHCORE_SKIP_DOWNLOAD=1` for host-specific session-start, then `bin/check-staleness`. SessionStart never blocks on network.

**Files:** `bin/session-start` (modify)

#### 5.6 Rewire `bin/validate-archcore` and `bin/check-cascade` through the launcher

- Replace direct `archcore` calls with `"$SCRIPT_DIR/archcore"` so the launcher is the single entry point. Silent-skip behavior on launcher exit 1 preserved.

**Files:** `bin/validate-archcore`, `bin/check-cascade` (modify)

#### 5.7 Remove "Step 0: Verify MCP" from all skills

- Under the old install model, every SKILL.md started with a "Verify MCP" block instructing the user to install the CLI if MCP tools were missing. With the launcher, MCP is always present on first tool call. The block is dead weight and confuses onboarding.
- Sweep all SKILL.md files (33 today, was 32 at the time of this sweep — `graph` was added later) to drop the block.

**Files:** all `skills/*/SKILL.md` (modify)

#### 5.8 Tests

- New `test/unit/launcher.bats` — covers resolution order, loop guard, `ARCHCORE_BIN`/`ARCHCORE_SKIP_DOWNLOAD`, checksum mismatch, network-failure exit codes.
- Update `test/unit/session-start.bats` — exercises the launcher-mediated path.
- Update `test/structure/scripts.bats` — require the launcher scripts to exist, be executable, and reference `CLI_VERSION`.

**Files:** `test/unit/launcher.bats` (new), `test/unit/session-start.bats`, `test/structure/scripts.bats` (modify)

#### 5.9 Update README

- Quick Start: no prerequisites; first MCP call downloads the CLI.
- "Offline / enterprise / BYO CLI" section documenting `ARCHCORE_BIN` and `ARCHCORE_SKIP_DOWNLOAD=1`.
- Keep Cursor install documented with the external-MCP step.

**Files:** `README.md` (modify)

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
- [x] `bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`, `bin/CLI_VERSION` exist and implement the full resolution order
- [x] Launcher downloads are SHA-256 verified against `checksums.txt`
- [x] `.mcp.json` at plugin root registers `archcore` against `${CLAUDE_PLUGIN_ROOT}/bin/archcore mcp`
- [x] `bin/session-start` passes `ARCHCORE_SKIP_DOWNLOAD=1` to the launcher
- [x] All SKILL.md files have the "Step 0: Verify MCP" block removed (33 files today)
- [x] `test/unit/launcher.bats` covers launcher resolution and failure modes
- [x] Users with a global `archcore` on `PATH` experience no behavior change (launcher defers to `PATH`)

## Dependencies

- Multi-Host Plugin Architecture ADR (`.archcore/plugin/multi-host-plugin-architecture.adr.md`) — architectural decision for the shared-core / per-host split
- Bundled CLI Launcher ADR (`.archcore/plugin/bundled-cli-launcher.adr.md`) — architectural decision for Phase 5 (launcher + plugin-owned MCP)
- Multi-Host Compatibility Layer Specification (`.archcore/plugin/multi-host-compatibility-layer.spec.md`) — technical contract
- Hooks and Validation System Specification (`.archcore/plugin/hooks-validation-system.spec.md`) — hook semantics
- Cursor IDE installed for testing
- Cursor plugin documentation (docs.cursor.com) for format verification
- Archcore CLI GitHub Releases publishing platform-targeted archives with `checksums.txt`
