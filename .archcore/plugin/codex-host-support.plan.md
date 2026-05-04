---
title: "Codex CLI Host Support Implementation Plan"
status: draft
tags:
  - "codex"
  - "multi-host"
  - "plugin"
  - "roadmap"
---

## Goal

Implement OpenAI Codex CLI as the third first-class host for the Archcore plugin with Codex-native packaging for skills, plugin-managed MCP, hooks config, and subagent TOML files. Promote the Codex CLI row in the Multi-Host Compatibility Layer Specification's Supported Hosts table from "TBD / Future" to "Implemented". Marketplace registration (`codex plugin marketplace add archcore-ai/plugin`), no manual `codex mcp add`, zero regression for Claude Code/Cursor, no host-specific business logic in shared core.

## Tasks

### Phase 0: Spike — Resolve Open Questions (BLOCKING all other phases)

Estimate: 1–2 days. Run spike before committing to design choices in Phases 1–4.

#### 0.1 Verify plugin-relative path resolution

- Build a minimal test plugin installed locally via `codex plugin marketplace add /path/to/plugin`.
- Verify whether Codex resolves plugin-relative component paths from `.codex-plugin/plugin.json`.
- Treat `${CODEX_PLUGIN_ROOT}` as undocumented unless Codex docs/runtime prove otherwise.

**Output:** Confirmed plugin-relative path resolution for bundled MCP and hooks config; no dependency on `${CODEX_PLUGIN_ROOT}`.

#### 0.2 Verify plugin-side `.mcp.json` schema

- Same minimal test plugin: declare an MCP server in a plugin-root Codex-specific MCP file using `{"mcpServers": {...}}`, referenced from `.codex-plugin/plugin.json` via `mcpServers: "./.codex.mcp.json"`.
- Verify Codex picks up the registration and the MCP server is reachable from a Codex session.
- If wrapper format differs: document actual schema.

**Output:** Confirmed plugin-side MCP config format. Decision: do not reuse Claude `.mcp.json`; ship plugin-root `.codex.mcp.json`.

#### 0.3 Verify plugin-bundled subagents

- Add `agents/test-agent.toml` to the test plugin with minimal fields. Check Codex discovers it after install.
- If not discovered: try `.codex-plugin/agents/test-agent.toml`. If still not discovered: confirm subagents must be installed manually to `~/.codex/agents/`.

**Output:** Confirmed mechanism for plugin-bundled subagents OR documented manual install fallback.

#### 0.4 Verify skill invocation namespacing

- Add `skills/test-skill/SKILL.md` to the test plugin. From a Codex session, type `@` and verify whether the skill appears as `@archcore/test-skill`, `@test-skill`, or other.
- Test conflict scenario: install two test plugins with same skill name, see how Codex disambiguates.

**Output:** Confirmed skill namespacing behavior. Decide whether to rename skills with prefix (e.g., `archcore-decide`) if flat namespace risks collisions.

#### 0.5 Verify per-subagent `disabled_tools[]`

- In test agent TOML, set `sandbox_mode = "read-only"` and `disabled_tools = ["mcp__archcore__create_document"]`. Spawn the agent and try to call the disabled tool — confirm it's rejected.
- If `disabled_tools` is not honored: plan auditor enforcement via `developer_instructions` only (soft guarantee) plus consider read-only MCP server variant (`bin/archcore mcp --read-only`).

**Output:** Confirmed enforcement mechanism for auditor read-only MCP whitelist.

#### 0.6 Verify SKILL.md frontmatter compatibility

- Copy a real SKILL.md from `skills/decide/` into the test plugin. Verify Codex loads it without error (specifically, whether `argument-hint` field causes warnings/failures).
- Test description budget — 17 skills with current descriptions, fits in ~2% / 8000 chars budget?

**Output:** Confirmation that existing SKILL.md files load cleanly OR list of minimal frontmatter fixes needed.

**Phase 0 exit criterion:** All 6 spikes complete; results captured in a short ADR (`codex-spike-findings.adr.md`) or as amendments to this plan's "Risks" section. Unresolved spikes become explicit risks with workaround plans.

### Phase 1: Manifest, MCP Wiring, and Marketplace Listing

Estimate: 0.5–1 day. Depends on 0.1, 0.2.

#### 1.1 Create `.codex-plugin/plugin.json`

- Required fields synchronized with `.claude-plugin/plugin.json`: `name = "archcore"`, `version` (matching), `description`.
- Component pointers (`./...` relative): `skills: "./skills/"`, `hooks: "./hooks/codex.hooks.json"`, `mcpServers: "./.codex.mcp.json"`.
- Optional `interface{}` block — port from `.cursor-plugin/plugin.json`'s richer metadata: `displayName`, `category`, `keywords`, `tags`, plus Codex-specific marketplace fields (`brandColor`, `composerIcon`, `defaultPrompt[]` if known).

**Files:** `.codex-plugin/plugin.json` (new)

#### 1.2 MCP wiring decision and implementation

- Ship plugin-root `.codex.mcp.json` using the public Codex plugin examples' `{"mcpServers": {...}}` wrapper.
- Do not reuse the existing root `.mcp.json`, because it is Claude-specific and references `${CLAUDE_PLUGIN_ROOT}`.
- The `command` always points at `bin/archcore` with `args: ["mcp"]`. Add `startup_timeout_sec` and `tool_timeout_sec` defaults if Codex requires them.

**Files:** `.codex.mcp.json` (new)

#### 1.3 Create `.agents/plugins/marketplace.json`

- Codex marketplace descriptor with `name = "archcore-plugins"`, plugin source `{ "source": "local", "path": "./" }`, and policy `INSTALLED_BY_DEFAULT` / `ON_INSTALL`.
- Keep `.codex-plugin/` limited to `plugin.json`.

**Files:** `.agents/plugins/marketplace.json` (new)

#### 1.4 Smoke test

- Install plugin locally in Codex via `codex plugin marketplace add file:///path/to/plugin`.
- Verify MCP server starts; `list_documents` MCP tool callable.
- No regression in Claude Code (still installs, still works).

### Phase 2: Hooks Configuration and Stdin Normalization

Estimate: 0.5–1 day. Depends on 0.1.

#### 2.1 Create `hooks/codex.hooks.json`

- Clone `hooks/hooks.json` (Claude Code) semantically, then replace `${CLAUDE_PLUGIN_ROOT}/bin/...` with plugin-relative `./bin/...`.
- PascalCase event names (same as Claude Code: `SessionStart`, `PreToolUse`, `PostToolUse`).
- Match patterns identical except Codex PreToolUse includes `apply_patch` for Codex's native edit primitive; PostToolUse covers the five MCP mutation matchers.
- Timeouts identical: 1s PreToolUse, 3s PostToolUse.
- Do NOT register `validate-archcore` on Write/Edit PostToolUse path (Compatibility Layer invariant).

**Files:** `hooks/codex.hooks.json` (new)

#### 2.2 Add `codex` host detection branch to `bin/lib/normalize-stdin.sh`

- Update host detection heuristic. New order:
  ```
  if stdin contains "conversation_id"  → cursor
  elif stdin contains "hookEventName"  → copilot
  elif stdin contains "turn_id"        → codex
  else                                 → claude-code (fallback)
  ```
- Add `codex)` case in the field-extraction `case` statement. Codex uses snake_case identical to Claude Code, so the body is identical to the `claude-code` case. Explicit branch documented for clarity and future divergence.
- Update `archcore_hook_info` and `archcore_hook_pretool_info` to include `claude-code|copilot|codex` in the case statement (already use the same `hookSpecificOutput` format).
- `archcore_hook_block` uses `exit 2` — works for Codex unchanged.

**Files:** `bin/lib/normalize-stdin.sh` (modify)

#### 2.3 Hook smoke test in Codex

- Enable `[features].codex_hooks = true` in the test Codex config.
- Install plugin locally in Codex.
- Trigger SessionStart — verify context loaded (or appropriate empty-state nudge).
- Try direct Write to `.archcore/somefile.md` via Codex's edit tool — verify exit-2 block.
- Create document via MCP — verify validate/cascade/precision PostToolUse fires.
- Edit a source file — verify code-alignment context injection runs.

### Phase 3: Subagent TOML Conversion

Estimate: 0.5–1 day. Depends on 0.3, 0.5.

#### 3.1 Convert `agents/archcore-auditor.md` to `agents/archcore-auditor.toml`

- Required: `name`, `description`, `developer_instructions` (port the entire MD body).
- `model`: map "sonnet" to Codex equivalent (e.g., `gpt-5.4` or whatever the spike confirms).
- `model_reasoning_effort = "high"` (auditor benefits from thorough reasoning).
- `sandbox_mode = "read-only"`.
- If spike 0.5 confirmed `disabled_tools[]` works: add disabled list for mutating MCP tools (`mcp__archcore__create_document`, `update_document`, `remove_document`, `add_relation`, `remove_relation`).
- If spike 0.5 showed `disabled_tools[]` not honored: strengthen `developer_instructions` with explicit "do not call mutating MCP tools" clause; document the soft-guarantee limitation in this plan's risks.

**Files:** `agents/archcore-auditor.toml` (new). Keep `agents/archcore-auditor.md` unchanged.

#### 3.2 Convert `agents/archcore-assistant.md` to `agents/archcore-assistant.toml`

- Same conversion process. `sandbox_mode = "workspace-write"` (assistant is read-write).
- No `disabled_tools[]` needed — assistant has full MCP access by design.
- `nickname_candidates = [...]` if Codex's display-name pool feature is desired.

**Files:** `agents/archcore-assistant.toml` (new). Keep `agents/archcore-assistant.md` unchanged.

#### 3.3 Plugin-bundled vs manual install

- If spike 0.3 confirmed plugin-bundled subagents work: add `agents` pointer to `.codex-plugin/plugin.json` if Codex requires it.
- If not supported: add `bin/install-codex-agents` helper script that copies the TOML files to `~/.codex/agents/` on user request, and document the manual step in README.

### Phase 4: Launcher Cache Extension

Estimate: 0.25 day. Depends on 0.1.

#### 4.1 Extend POSIX launcher cache resolution

- In `bin/archcore`, prepend `$CODEX_PLUGIN_DATA/archcore/cli` to the cache directory search list (step 3) before `$CLAUDE_PLUGIN_DATA/...`.
- If `$CODEX_PLUGIN_DATA` is empty/unset: skip (the existing fallbacks work).
- No change to download path or checksum logic.

**Files:** `bin/archcore` (modify)

#### 4.2 Extend Windows launcher cache resolution

- In `bin/archcore.ps1`, prepend `$env:CODEX_PLUGIN_DATA\archcore\cli` to the cache search list before `$env:CLAUDE_PLUGIN_DATA\...`.
- Same fallback behavior.

**Files:** `bin/archcore.ps1` (modify)

#### 4.3 Launcher tests

- Update `test/unit/launcher.bats` with `CODEX_PLUGIN_DATA` cases mirroring `CLAUDE_PLUGIN_DATA` cases.
- Verify resolution order: env > PATH > Codex cache > Claude cache > XDG cache > download.

### Phase 5: Skill Compatibility Verification

Estimate: 0.25 day. Depends on 0.6.

- If spike 0.6 confirmed clean load of all 17 SKILL.md files: no action.
- If spike found `argument-hint` causes warnings: leave as-is if non-fatal; otherwise plan a CPAT to remove `argument-hint` (low-risk: it's a Claude-Code-specific UX hint, removing it doesn't affect functionality).
- If description budget exceeded: identify the most descriptive skills (decide, plan, capture, standard) as priority; consider compressing descriptions of less-frequently-used skills (iso-track, sources-track, product-track, feature-track, architecture-track, standard-track).

**Files:** `skills/*/SKILL.md` (modify only if spike requires)

### Phase 6: Documentation and Spec Promotion

Estimate: 0.5 day.

#### 6.1 Update `multi-host-compatibility-layer.spec.md`

- Supported Hosts table: Codex CLI row from "TBD / TBD / TBD / Future" to actual values (`.codex-plugin/plugin.json`, `hooks/codex.hooks.json`, plugin-shipped MCP wiring path, "Implemented").
- Hook Event Mapping table: add Codex column or note that Codex maps identically to Claude Code (PascalCase events, same matchers).
- Per-Host Hooks Configuration: add Codex subsection mirroring Claude Code's.
- Plugin Manifests: add Codex subsection.
- MCP Server Wiring: add Codex subsection ("Plugin-shipped" parity with Claude Code).
- Normative Behavior, Constraints, Conformance: amend to include `codex` as a recognized host.

**Files:** `multi-host-compatibility-layer.spec.md` (modify)

#### 6.2 Update `multi-host-plugin-architecture.adr.md`

- Update the "Drivers" subsection: Codex now confirmed in production support list, not just an "ask".
- Update the directory tree under Decision: add `.codex-plugin/`, `hooks/codex.hooks.json`, `agents/*.toml`.
- Update Negative consequence about MCP wiring: Codex joins Claude Code in plugin-shipped MCP; Cursor remains the outlier.

**Files:** `multi-host-plugin-architecture.adr.md` (modify)

#### 6.3 Update `bundled-cli-launcher.adr.md`

- Update the Negative consequence about "Multi-host divergence risk": Codex now has parity with Claude Code (zero-setup install). Cursor still requires user-registered MCP. Adjust the framing accordingly.
- Update resolution-order documentation to mention `$CODEX_PLUGIN_DATA` cache directory.

**Files:** `bundled-cli-launcher.adr.md` (modify)

#### 6.4 Update README

- Add "Codex CLI" install section: `codex plugin marketplace add archcore-ai/plugin`.
- Document minimum Codex version (v0.117.0+).
- Update supported-hosts list at the top.
- Note the manual subagent install step if Phase 3.3 fallback is needed.

**Files:** `README.md` (modify)

### Phase 7: End-to-End Verification

Estimate: 0.5 day.

- Fresh Codex install (no existing `~/.codex/` config). Run: `codex plugin marketplace add archcore-ai/plugin`.
- Open a fresh project. Initialize Archcore via MCP (`init_project`). Verify MCP tools. With `[features].codex_hooks = true` enabled, verify SessionStart context and hook guardrails blocking direct writes.
- Spawn `archcore-auditor`; attempt a mutating MCP call from inside the auditor session — verify rejection (or soft refusal if `disabled_tools[]` not honored).
- Run the existing Claude Code test suite (`make test`). Pass.
- Manual smoke test in Cursor (existing flow). Pass.

## Acceptance Criteria

- [ ] All 6 Phase 0 spikes complete; findings captured (in this plan's risks or in a separate ADR).
- [ ] `.codex-plugin/plugin.json` exists with synchronized metadata, valid component pointers, and `interface{}` marketplace block.
- [ ] Plugin-shipped MCP wiring works in Codex without external `codex mcp add` step.
- [ ] `hooks/codex.hooks.json` maps the five active hook functions with correct matchers and timeouts.
- [ ] `bin/lib/normalize-stdin.sh` has explicit `codex` host detection and field extraction.
- [ ] `bin/archcore` and `bin/archcore.ps1` extend cache resolution with `$CODEX_PLUGIN_DATA` directory.
- [ ] All Archcore `SKILL.md` files load cleanly in Codex (smoke-confirmed through `codex debug prompt-input`).
- [ ] `agents/archcore-auditor.toml` and `agents/archcore-assistant.toml` exist; auditor enforced read-only via `sandbox_mode` and (if supported) `disabled_tools[]`.
- [ ] `codex plugin marketplace add archcore-ai/plugin` succeeds end-to-end on a fresh install.
- [ ] All existing Claude Code tests pass unchanged.
- [ ] Cursor manual smoke test passes unchanged.
- [ ] `multi-host-compatibility-layer.spec.md` Codex CLI row promoted from TBD to actual values.
- [ ] README documents Codex install path.

## Dependencies

- Multi-Host Plugin Architecture ADR (`.archcore/plugin/multi-host-plugin-architecture.adr.md`).
- Multi-Host Compatibility Layer Specification (`.archcore/plugin/multi-host-compatibility-layer.spec.md`).
- Multi-Host Plugin Implementation Plan (`.archcore/plugin/multi-host-implementation.plan.md`) — predecessor; this plan continues the same architecture.
- Bundled CLI Launcher ADR (`.archcore/plugin/bundled-cli-launcher.adr.md`).
- Hooks and Validation System Specification (`.archcore/plugin/hooks-validation-system.spec.md`).
- Codex CLI v0.117.0+ available locally for spike and integration testing.
- GitHub repo `archcore-ai/plugin` reachable as a Codex marketplace source.

## Risks (live; updated after Phase 0 spike)

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `${CODEX_PLUGIN_ROOT}` env var absent | Resolved | Low | Use plugin-relative `./bin/...` commands and manifest pointers; no root env substitution required |
| Plugin-local hooks require feature/runtime support | Medium | Medium | Ship `hooks/codex.hooks.json`; verify live execution only with `[features].codex_hooks = true` and current Codex runtime support |
| Plugin-bundled subagents not supported | Medium | Medium | Ship TOML variants side-by-side; fallback is `bin/install-codex-agents` helper + README step |
| `disabled_tools[]` per-subagent not honored | High | Medium | Spike 0.5 resolves; fallback is `developer_instructions` enforcement (soft) plus optional `bin/archcore mcp --read-only` mode |
| Plugin-side `.mcp.json` schema differs from Claude Code wrapper | Resolved | Low | Use public Codex examples' `{"mcpServers": {...}}` wrapper in plugin-root `.codex.mcp.json` |
| Skill `argument-hint` frontmatter rejected | Low | Low | Spike 0.6 resolves; CPAT to remove if needed |
| Skill flat namespace causes collisions with other plugins | Low | Low | Spike 0.4 resolves; rename skills with `archcore-` prefix if needed |
| Codex CLI rapid-iteration on plugin API breaks compatibility | Medium | Medium | Pin minimum Codex version in README; add version check to `bin/session-start` if practical |
| Marketplace install requires repo-level changes | Resolved | Low | `.agents/plugins/marketplace.json` is accepted by `codex plugin marketplace add` |

## Estimate

Phase 0 spike: 1–2 days (blocking).
Phases 1–7 implementation: ~3.5–4 days.
Total: 4.5–6 days dev work + 1–2 days for documentation/test polish = **6–8 days**.
