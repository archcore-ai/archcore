---
title: "Plugin Development Roadmap"
status: rejected
tags:
  - "plugin"
  - "roadmap"
---

## Goal

Deliver the complete Archcore plugin feature set, turning the original thin MCP-and-hook wrapper into a guided Archcore experience across Claude Code, Cursor, Codex CLI, and GitHub Copilot CLI.

## Tasks

### Phase 1 — documentation. Done.

Created the project documentation using Archcore's own document types, as dogfooding.

### Phase 2 — skills. Done, then collapsed.

Built the skills across the multi-tier hierarchy as planned, then consolidated them progressively through a series of decisions; Phase 9 records the final state. The evolution ran from the initial build of intent, track, type, and utility tiers at a peak of 34 skill directories; through `remove-document-type-skills.adr`, which removed the type layer and left 18; through `merge-review-status-remove-graph.adr`, which merged `status` into `review` and removed `graph`, leaving 16; to `skill-surface-collapse.adr`, which removed the track tier, merged `actualize` into `audit`, renamed `bootstrap` to `init`, merged `standard` into `decide`, and removed `verify`, leaving **7**.

### Phase 3 — commands and agents. Done.

- [x] Establish the intent skills as the primary user entry points.
- [x] Make every Archcore document type reachable through an intent skill or a direct `create_document(type=<any>)` call.
- [x] Ship `archcore-assistant` as the read/write agent with full MCP tool access.
- [x] Ship `archcore-auditor` as the read-only auditor with code-document correlation.

### Phase 4 — hooks and validation. Done, at 6 hook entries per host.

- [x] SessionStart through `bin/session-start`: the CLI availability check, the project check, context loading, and the staleness check.
- [x] Pre-mutation through `bin/check-archcore-write`: block a direct `.archcore/**/*.md` write.
- [x] Pre-mutation through `bin/check-code-alignment`: inject context for a source-file edit.
- [x] Post-mutation through `bin/validate-archcore`: validate after an MCP document mutation.
- [x] Post-mutation through `bin/check-cascade`: detect cascade staleness after an update.
- [x] Post-mutation through `bin/check-precision`: emit precision warnings after a create or update.
- [x] Keep every hook idempotent, pre-mutation inside 1 second and post-mutation inside 3.
- [x] Register no post-mutation `Write|Edit` validator, with an anti-regression test guarding against reintroduction.

### Phase 5 — multi-host support. Done.

- [x] The Cursor adapter layer: its manifest, hooks config, rules, and the reference MCP template.
- [x] The Codex CLI adapter layer: its manifest, the marketplace catalog, its hooks config, the command wrappers, its MCP config, and the TOML subagent variants.
- [x] The stdin normalization library, detecting Claude Code, Cursor, Copilot, Codex, and OpenCode.
- [x] Plugin-shipped MCP for Claude Code through `.claude.mcp.json`, reached by an explicit manifest key, and for Codex CLI through `.codex.mcp.json`. Both name `archcore` on PATH.
- [x] Cursor MCP setup documented as a one-time copy of the reference template, later superseded by the wiring that `host-wiring-parity.adr` added to init.
- [x] The GitHub Copilot CLI adapter layer: its dedicated manifest, its native camelCase hooks config, the deny-output branch in the guard scripts, and project-level MCP wiring per `copilot-mcp-architecture.adr`.

**Phase 5a — the bundled CLI launcher, reverted.** It shipped briefly and was removed entirely in plugin v0.4.0 on 2026-05-12, after eight bug classes made the zero-setup framing a net loss. The plugin now bundles, downloads, and caches no CLI; users install it globally from the official installer.

### Phase 6 — zero-content onboarding. Done.

- [x] The SessionStart empty-state helper, detecting a 200-byte markdown body floor.
- [x] The SessionStart advisory that nudges toward `/archcore:init` on a missing or functionally empty `.archcore/`, suppressible by environment variable.
- [x] The `/archcore:init` intent skill with its three confirmable steps — the stack rule, the run-the-app guide, and the opt-in import of agent-instruction files. It shipped as `bootstrap` and was renamed in Phase 9.
- [x] The skill support libraries under `skills/init/lib/`.
- [x] The tag and body source convention for an imported document.
- [x] Idempotent re-runs through a tag lookup.
- [x] A CLI availability pre-flight pointing at the official installer.

### Phase 7 — type skill removal. Done.

- [x] Absorb the RFC elicitation into `/archcore:decide`.
- [x] Absorb the CPAT elicitation into the standard cascade, later folded into the `decide` continuations in Phase 9.
- [x] Delete the 17 type-skill directories.
- [x] Update the count invariants across the README, the tests, and the `.archcore/` documents.
- [x] Record the decision in `remove-document-type-skills.adr`.

### Phase 8 — inspection skill consolidation. Done.

- [x] Absorb `status` into the default short mode of `review`.
- [x] Run the full audit behind `--deep`.
- [x] Remove `graph` entirely, on near-zero usage.
- [x] Record the decision in `merge-review-status-remove-graph.adr`.

### Phase 9 — skill surface collapse. Done.

- [x] Rename `bootstrap` to `init`.
- [x] Merge `review` and `actualize` into `audit`, with the flag surface `[--deep] [--drift] [filter]` and the drift protocol moved into the audit lib.
- [x] Remove the six track skills, moving their flow logic into the four plan references and the decide continuations.
- [x] Remove `standard`, since the ADR, rule, and guide cascade now lives inside `decide` as continuations.
- [x] Remove `verify`, since `make verify` is the canonical integrity check.
- [x] Reduce the command wrappers to 7.
- [x] Update the count invariants across the README, the structure tests, and the `.archcore/` documents.

## Acceptance Criteria

- All 18 Archcore document types are covered through one of the four commands or a direct `create_document(type=<any>)` call.
- The four commands are operational: `init`, `plan`, `document`, `review` (per `four-command-palette.adr`).
- Exactly four commands are visible in the `/` menu, over a gated track layer per `track-layer.spec`.
- Two agents exist: the read/write assistant and the read-only auditor.
- The pre-mutation guard blocks every direct write attempt against a `.archcore/` markdown file.
- The post-mutation hooks report validation issues, cascade staleness, and precision warnings.
- SessionStart prints the CLI install guidance when `archcore` is missing, loads context when it is present, and nudges toward `/archcore:init` on an empty `.archcore/`.
- Every plugin component uses MCP tools exclusively, with zero direct file writes.
- The plugin behaves identically across Claude Code, Cursor, Codex CLI, and GitHub Copilot CLI, with each documented host gap declared by name.

## Dependencies

- The Archcore CLI installed globally on PATH per https://docs.archcore.ai/cli/install/. The plugin does not bundle, download, or cache it.
- The Claude Code plugin system, supporting skills, agents, hooks, bin scripts, and a manifest-declared MCP config.
- The Cursor plugin system, supporting skills, agents, hooks, and rules, with MCP registered outside the plugin.
- The Codex CLI plugin system, supporting skills, commands, TOML agents, hooks, and plugin-shipped MCP.
- The GitHub Copilot CLI plugin system, supporting skills, `*.agent.md` agents, commands, and hooks, with MCP registered per project.
- The MCP tools: `create_document`, `update_document`, `list_documents`, `get_document`, `add_relation`, `remove_relation`, `list_relations`, `remove_document`, `search_documents`, and `init_project`.
- The governing decisions: always-use-MCP-tools, plugin component architecture, the single universal agent with its read-only auditor extension, intent-based skill architecture, the inverted invocation policy, type-skill removal, inspection consolidation, the skill surface collapse, the actualize system, multi-host plugin architecture, and the three host MCP decisions for Cursor, Copilot, and host-wiring parity.
- The superseded bundled-launcher decision, rejected and replaced.
