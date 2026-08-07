---
title: "CLI v2 — Layer 4–5 Boundary, Hook Repatriation, No New Commands"
status: rejected
tags:
  - "architecture"
  - "hooks"
  - "multi-host"
  - "plugin"
---

## Goal

Separate the CLI work required by the plugin's 4-command redesign into its own delivery stream. Target boundary: the CLI owns document types, templates, validation, retrieval signals, and host guardrails (layers 4–5). All track and interview logic leaves the CLI. Standing constraint: the public command surface does not grow — `init / config / doctor / status / hooks / mcp / instructions / sync / update` (@cli `cmd/root.go`) remains the full set through both releases; new capability lands inside existing subcommands, never as a new top-level command. Plugin plans reference this plan only through the release ordering in Dependencies.

## Delivery Status

Release A and Release B shipped together in CLI v0.7.0 (tagged 2026-08-06): tasks 1–9 landed, with the task-6 Codex question resolved by adding `codex-cli` to `hooksInstallers`, writing `.codex/hooks.json` with matcher `Write|Edit|apply_patch`. Not shipped: task 10 (`archcore status` counters) — deferred; the session recap's CORPUS line carries category and status counts in the meantime. The plugin-side cutover landed on `dev` 2026-08-07: launchers `bin/pre-tool-use` and `bin/post-tool-use` delegate to the CLI leaves behind a `cli-gte 0.7.0` gate, the seven shell policy scripts are deleted, and the plugin version is 0.7.0. Phase 2 items 11–13 remain open.

## Tasks

Release A — subtraction and repatriation, no plugin surface change:

1. `pre-tool-use` hook command in Go. Add a hidden `pre-tool-use` subcommand beside `session-start` in the per-host command factory (@cli `cmd/hooks_claude_code.go`, `newSessionStartHookCmd` pattern; output shapes `shapeClaudeCompat` / `shapeCopilotNative`). Ports two scripts: `check-archcore-write` (blocks direct Write/Edit to `.archcore/*.md`, allows `settings.json` and `.sync-state.json`; the only blocking hook, so the Go layer needs a deny-capable output shape, which `hookOutput` / `copilotHookOutput` lack today) and `check-code-alignment` (path-token ranking over `.archcore/`, `codeAlignment.sourceRoots` read from `.archcore/settings.json`, `.archcore/global/` excluded). Payload decoding replaces the shell `lib/normalize-stdin.sh` first-occurrence JSON extraction with a real JSON decode of both payload families (Write/Edit `file_path`, MCP document-tool arguments). Budget: ≤ 1s; the advisory half never blocks.

2. `post-tool-use` hook command in Go. Ports three scripts: `validate-archcore` (today shells out to `timeout 2 archcore doctor` and greps for failures — becomes an in-process doctor call), `check-cascade` (reads `.sync-state.json` relations, reports dependents of the updated document via `implements` / `depends_on` / `extends`; Go already loads the manifest via `archsync.LoadManifest`; the ported message names `/archcore:review --drift` — the shell version still says the pre-v2 audit command), and `check-precision` (forbidden-lexicon scan per @plugins/archcore/skills/_shared/precision-rules.md). Budget: ≤ 3s, exit 0 always.

3. Port `check-staleness` as a shared advisory reused by the session-start and pre-tool-use paths — the shell version is invoked from inside `session-start` and `check-code-alignment`, not wired as its own hook.

4. Status filter. Exclude `rejected` documents from code-alignment injection and from the SessionStart output. `LocalDocument.Status` is already parsed (@cli `internal/mcp/tools/common.go`) and unused in @cli `cmd/hooks_common.go`; today rejected documents inject like any other.

5. Ranked recap. Replace the unbounded `EXISTING DOCUMENTS` listing in `buildSessionContext` (@cli `cmd/hooks_common.go`) with a fixed line budget: branch state, `draft` documents by mtime, documents `accepted` in the last 30 days. Today a 300-document repo injects 300 listing lines; the only cap in the file is `maxSessionTags = 20`.

6. Install wiring for the new events. Extend the per-host event maps (@cli `internal/wiring/hooks_agents.go` — each currently holds SessionStart only) and the matcher/ownership writer (@cli `internal/wiring/hooks_install.go`, marker `archcore hooks `) with PreToolUse (`Write|Edit`) and PostToolUse (MCP document tools). `hooksInstallers` lists claude-code / cursor / gemini-cli / copilot; the plugin separately ships `codex.hooks.json` — resolve whether codex-cli joins the hook-capable set or the plugin config remains the only Codex wiring. Hookless registry hosts keep the explicit "does not support hooks" path (@cli `cmd/hooks.go`).

7. Parity and spec tests before shell removal. Per-event tests in the repo's `*_spec_test.go` convention (@cli `cmd/hooks_session_dedup_spec_test.go` as the model) covering: trigger set per event, block vs advisory semantics per host shape, rejected-status exclusion, recap line budget, and graceful exit within the 1s/3s budgets. The plugin deletes its shell scripts only after these pass (plugin plan Stage 4).

Release B — removals:

8. Delete @cli `internal/mcp/prompts/` (5 track prompts: `iso_track`, `sources_track`, `product_track`, `standard_track`, `architecture_track`), their registration and the prompt capability (@cli `internal/mcp/server.go`), and invert the prompt integration tests (@cli `internal/mcp/integration/prompts_test.go` — the registration canary becomes "the server exposes no prompts").

9. Strip the REQUIREMENTS TRACKS, RESEARCH GATE (rnd), and WORKFLOW PROMPTS sections from `mcpServerInstructions` (@cli `internal/mcp/server.go`). Keep TYPE SELECTION RULES (22 rules), REQUIREMENTS LAYERS (type-to-layer catalog, not workflow), relation conventions, and tags. Regenerate `examples/` via @cli `scripts/regen-examples.sh` — the fixture pin (@cli `internal/agents/instructions_fixtures_test.go`) and the CI sync step fail otherwise.

10. `archcore status` counters. `runStatusChecks` (@cli `cmd/status.go`) reports issues, tag totals, and manifest counts only; add document counts by category / status / type, the relation count, and an orphan listing to match the audit short dashboard.

Removed from scope (maintainer decision, 2026-08-04) — no new public commands:

- `archcore scope [--branch]` — dropped because the branch boundary is plain git (`merge-base`, changed-file listing) that the agent runs directly per skill instructions.
- `archcore context <query>` — dropped because the `search_documents` MCP tool already serves manual topic pull; a CLI twin would create a second owner of one retrieval path.
- `archcore check [--ci]` — deferred because post-write hook validation already covers the write moment; a CI mode ships only on demand.

Phase 2 — after the plugin palette swap:

11. `get_type_schema` MCP tool: section skeletons, precision/EARS rules, layer-5 object contracts (mermaid blocks, structured inserts). CLI templates become the single canon; plugin contracts reference them.
12. Path index in the sync manifest for ranking; target < 500ms at 500 documents.
13. UserPromptSubmit topic injection on hosts that support the event; its backend is an internal hook runner, not a public command.

## Acceptance Criteria

- `hooks install` gives every host in `hooksInstallers` the write-guard and post-write validation; hookless registry hosts keep the explicit "does not support hooks" message.
- SessionStart output stays within its line budget at any corpus size.
- Injection and the session recap exclude `rejected` documents (fixture: a rejected document on a matching path is not injected).
- The MCP server exposes no track prompts; server instructions contain no track sections; `examples/` fixtures are regenerated and pass the CI sync step.
- Go hook tests demonstrate behavior parity with the shell versions before the shell versions are removed.
- `archcore --help` lists the same nine commands before and after both releases.

## Dependencies

- Release A precedes the plugin-side removal of `/archcore:context`.
- Release B precedes the plugin palette swap.
- Phase 2 items do not block the swap.