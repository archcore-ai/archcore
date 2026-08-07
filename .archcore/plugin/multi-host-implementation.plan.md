---
title: "Multi-Host Plugin Implementation Plan"
status: rejected
tags:
  - "multi-host"
  - "plugin"
  - "roadmap"
---

## Goal

Implement multi-host support for the plugin, so it runs in Cursor as the first priority and the architecture is prepared for GitHub Copilot and further hosts. The plugin must behave identically across hosts with zero duplication of skills, agents, or core logic.

**MCP scope note.** When this plan was drafted, MCP server configuration was explicitly out of scope: the plugin declared no `mcpServers` anywhere and shipped no MCP config at its root. That boundary was revised once for Claude Code, through the since-removed bundled launcher of Phase 5, and again when the launcher was removed in v0.4.0. The plugin now ships `.claude.mcp.json` and `.codex.mcp.json` at its root, both naming `archcore` on PATH and both reached only through an explicit manifest key, per `copilot-mcp-architecture.adr`. Cursor still relies on a user-registered server in `~/.cursor/mcp.json` or a project-scoped `.cursor/mcp.json`.

## Tasks

### Phase 1 — the stdin normalization layer

**1.1 Create `bin/lib/normalize-stdin.sh`**, a POSIX shell library sourced by every bin script. It reads stdin once into a variable, detects the host from the JSON structure, exports the normalized variables `ARCHCORE_HOST`, `ARCHCORE_HOOK_EVENT`, `ARCHCORE_TOOL_NAME`, `ARCHCORE_FILE_PATH`, and `ARCHCORE_TOOL_INPUT`, and provides the output helpers for info, block, and allow. It carries no external dependency: `grep` and `sed` only, with no jq.

**1.2 Refactor the existing bin scripts onto the normalizer**, replacing direct stdin parsing. The write guard replaces its inline parsing with the normalized file path and emits through the block helper. The validator uses the normalized tool name and file path and emits through the info helper. The cascade detector uses the normalized tool input for the document path and emits through the info helper. The staleness check needs no stdin change, because `session-start` calls it rather than a hook. And `session-start` changes minimally, possibly using the host variable for a host-specific CLI command.

**1.3 Verify Claude Code still works** across session start, document creation, direct-write blocking, validation, and cascade detection, with zero regression, since the normalizer defaults to the Claude Code format.

### Phase 2 — the Cursor adapter

**2.1 Research and verify the Cursor formats** by fetching the current documentation for the manifest schema, the hooks format including event names and the stdin and stdout protocol, and the rules format, then document every gap against Claude Code.

**2.2 Create the Cursor manifest** with name, version, description, and author, plus references to the skills, agents, hooks, and rules directories, and no `mcpServers` field, because MCP is registered outside the plugin. Verify each field name against the documentation from 2.1.

**2.3 Create the Cursor marketplace catalog** carrying the same metadata as the Claude Code catalog, adapted to the Cursor format.

**2.4 Create `hooks/cursor.hooks.json`**, mapping the active hook functions to the Cursor event names — session start, the pre-mutation `Write` matcher, and the post-MCP event running the validator and the cascade detector. Handle the session-start gap through the available alternative or through rules, use the correct stdin and stdout protocol, and use Cursor's plugin-root variable. Register no post-mutation `Write` validator entry, because the pre-mutation guard already blocks and such an entry would fork a shell repository-wide for no benefit.

**2.5 Rename the Claude Code hooks file** to a host-specific name, update every reference to it, and verify the Claude Code plugin system reads the new path.

**2.6 Create the Cursor rules**, optionally: an always-apply context rule carrying the document-type reference and the MCP tool names, which substitutes for session-start context injection, and a glob-scoped rule over `.archcore/**` reminding the agent that operations are MCP-only.

**2.7 Extend the normalizer for Cursor** by adding its host detection, mapping its stdin fields to the normalized variables, implementing its output format in the helpers, and testing against a sample payload.

### Phase 3 — verification in Cursor

**3.1 Install the plugin locally** through Cursor's local loading mechanism, verify the user-registered MCP server is reachable and its tools are available, and verify the skills appear in the slash-command menu.

**3.2 Test the core flows:** create a document through the decision skill, confirming the skill activates and the MCP tool works; attempt a direct write to `.archcore/` and confirm the hook blocks it; update a document and confirm the validation and cascade hooks fire; run the audit dashboard and confirm it lists documents; run the drift mode and confirm staleness detection works; and invoke the assistant agent on a complex task.

**3.3 Document the findings and fix what surfaces**, recording any Cursor-specific behavior difference, fixing the hook format issues found in testing, and updating the specification where actual behavior differs from the documentation.

### Phase 4 — repository cleanup

**4.1** Update the README with multi-host installation instructions and a supported-hosts section.

**4.2** The repository rename is done. The final name places the brand on the organization and keeps the repository name host-agnostic.

### Phase 5 — the bundled CLI launcher and plugin-owned MCP. Superseded and removed in v0.4.0.

This phase shipped a download-on-first-use launcher plus plugin-owned MCP registration. The launcher caused eight bug classes — offline failures in CI, version coupling to plugin releases, cache pollution across hosts, first-run latency, enterprise friction, security-patch lag, plugin bloat, and uneven host support, since Cursor users still did manual MCP setup — and was removed entirely in v0.4.0, commit `2f99997`.

The plugin-owned MCP shape survived the rollback: the two configs still ship at the plugin root, but they name `archcore` on PATH directly with no launcher indirection. Cursor users still register MCP outside the plugin, and the reference template now lives at `docs/cursor.mcp.example.json` rather than at the plugin root, per `cursor-mcp-architecture.adr`.

`bundled-cli-launcher.adr` holds the original decision, now rejected; `remove-bundled-launcher-global-cli.idea` holds the replacement with the eight-bug-class analysis and the one-time-install trade-off; and `stack-and-tooling.rule` pins that no plugin-side download-on-first-use mechanism returns without a fresh ADR.

The subtasks are preserved here as a historical record of what was built and then removed; none describes the current state. The version pin file, the POSIX launcher with its four-step resolution order, and the Windows launcher pair were all removed. The Claude Code MCP config pointing at the launcher was replaced by one naming `archcore` on PATH. `bin/session-start`, `bin/validate-archcore`, and `bin/check-cascade` were reverted from launcher indirection to direct invocation, with `session-start` gaining the missing-CLI install message. The launcher and CLI-contract test files were removed. The README's offline and bring-your-own-CLI section was removed in favor of a prerequisites section linking the installer. One item was kept rather than reverted: removing the MCP-verification preamble from every `SKILL.md`, which was correct on its own merits, while the sub-agent preambles that load knowledge-tree context were retained for the separate reason recorded in `subagent-knowledge-tree-bootstrap.adr`.

## Acceptance Criteria

- [x] Every bin script parses stdin through the normalizer.
- [x] The Claude Code plugin behaves identically after the refactor, with zero regression.
- [x] The Cursor manifest exists in the correct format and declares no `mcpServers` field.
- [x] The Cursor hooks config maps the active hook functions to the Cursor events and contains no post-mutation entry.
- [x] The plugin loads in Cursor, with skills discoverable and the user-registered MCP tools available.
- [x] The core flow works in Cursor: create a document, validate, and detect cascade.
- [x] Direct-write blocking works in Cursor.
- [x] Every config format is verified against the official host documentation.
- [x] No skill and no agent contains a host-specific reference, so the invariant holds.
- [x] `bin/session-start` emits actionable guidance when `.archcore/` is missing, routing through `init_project`.
- [x] The Claude Code MCP config registers `archcore`, now naming the PATH-resolved binary directly rather than the removed launcher.
- [x] Every `SKILL.md` has the MCP-verification block removed.
- [x] A user with a global `archcore` on PATH sees no behavior change. The invariant is stronger now: a PATH install is the only supported path.

Four criteria were reverted with the launcher and no longer apply: the existence of the launcher scripts and the version pin; SHA-256 verification of a launcher download against a checksum file; passing a skip-download variable from `session-start`; and the launcher test file.

## Dependencies

- `multi-host-plugin-architecture.adr` — the architectural decision for the shared-core and per-host split.
- `multi-host-compatibility-layer.spec` — the technical contract.
- `hooks-validation-system.spec` — the hook semantics.
- `bundled-cli-launcher.adr` was a dependency and is now rejected and superseded by `remove-bundled-launcher-global-cli.idea`.
- Cursor installed for testing, and its plugin documentation for format verification.
- The Archcore CLI installed on PATH per https://docs.archcore.ai/cli/install/, with no plugin-side fetching.
