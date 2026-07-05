---
title: "Host Adapter Contract — Portable Core Boundary and Adapter Obligations"
status: accepted
tags:
  - "architecture"
  - "hooks"
  - "multi-host"
  - "plugin"
---

## Purpose

Formalize the boundary between the **portable core** of the Archcore plugin and the **per-host adapters**, so host expansion (GitHub Copilot and OpenCode next) reuses the core unchanged instead of refactoring it. This spec is the contract every new host adapter is built against — and the gate contributor PRs are reviewed against.

**Ownership:** the portable core and this contract are maintainer-owned. Key design decisions (boundary changes, new adapter targets, hook-semantics mappings) are recorded by the maintainer before implementation; contributor work is limited to adapter implementation against an accepted decision. See `stack-and-tooling.rule` for the language/tooling side of the same policy.

## Scope

Covers the three implemented hosts (Claude Code, Cursor, Codex CLI), the two planned adapters (GitHub Copilot, OpenCode), and any future host. Applies to everything under `plugins/archcore/` and to adapter code that lives outside this repo (e.g. an npm wrapper package for OpenCode): external adapters are bound by the same MUST NOT rules even where the stack rule's shell-only requirement (which is scoped to this repo) does not reach them.

## Portable core (normative)

The core is maintained once and is host-agnostic:

- `plugins/archcore/skills/` — Agent Skills (SKILL.md). MUST NOT contain host-conditional instructions.
- `plugins/archcore/agents/` — markdown agent definitions plus per-host format variants (e.g. Codex TOML), kept in parity by structure tests.
- `plugins/archcore/commands/` — slash-command wrappers.
- `plugins/archcore/bin/` — hook scripts + `bin/lib/normalize-stdin.sh`. POSIX sh only (stack rule). Host awareness is allowed **only** here: stdin detection and field extraction in `normalize-stdin.sh`, and host-conditional *output/exit conventions* in hook scripts (e.g. a Copilot deny-JSON branch), keyed on `ARCHCORE_HOST`.

Everything else an adapter ships is **configuration, not logic**.

## Normative Behavior — adapter MUST provide

1. **Manifest** the host discovers. A shared manifest is preferred where the host reads it (Copilot's loader checks `.claude-plugin/plugin.json`).
2. **Hook wiring** for the three canonical lifecycle events, each routed to the shared `bin/` scripts:
   - session-start → `bin/session-start`
   - pre-mutation guard → `bin/check-archcore-write` (+ `bin/check-code-alignment` for source edits)
   - post-mutation validation → `bin/validate-archcore` (+ cascade/precision checks)
3. **MCP registration** launching `archcore mcp` resolved from PATH — or a documented user-side fallback where plugin-shipped MCP is unsafe (the Cursor precedent, `cursor-mcp-architecture.adr`).
4. **A host case in `bin/lib/normalize-stdin.sh`** — detection plus the canonical env schema (`ARCHCORE_HOST`, `ARCHCORE_TOOL_NAME`, `ARCHCORE_FILE_PATH`, …).
5. **A mutation-tool coverage-matrix row**: the full set of the host's filesystem-mutation tools, asserted by a structure test so a coverage gap fails CI instead of surfacing when a user trips it.
6. **A dated three-probe verification** (main-session source write, delegated write, `.archcore/` write) recorded in `hooks-validation-system.spec`.

## Normative Behavior — adapter MUST NOT

- Fork or copy skills/agents per host, or patch skill text for one host.
- Reimplement guard/validation logic outside `bin/` — **including in adapter code written in other languages**. Hosts with programmatic hook systems (OpenCode JS/TS plugins) implement hooks as thin bridges: build the canonical stdin JSON, spawn the `bin/` script, translate its exit/output into host semantics. No decision logic lives in the bridge.
- Introduce runtimes or languages into `plugins/archcore/` without an accepted ADR (`stack-and-tooling.rule`).
- Ship logic in manifests or hook configs (configuration only).

## Constraints — blocking-semantics translation per host

The canonical convention is defined by the `bin/` scripts: exit 0 = pass; block = exit 2 + reason on stderr (the Claude Code / Codex native semantics). Adapters translate:

| Host | Pre-mutation deny mechanism | Notes |
|---|---|---|
| Claude Code / Codex CLI | exit 2 + stderr | native convention |
| Cursor | camelCase events per `cursor.hooks.json` | `Write`-only matcher gap documented |
| GitHub Copilot | stdout JSON `{"permissionDecision":"deny","permissionDecisionReason":…}` | exit 2 is a *warning* there; non-2 non-zero exit also denies (fail-closed); timeout fails **open** |
| OpenCode | bridge throws `Error(reason)` from `tool.execute.before` | model receives a failed tool result carrying the message |

A given translation lives in exactly one place: either the `bin/` script's host-aware output branch (shell) or the bridge's mechanical exit-to-host translation (adapter) — never duplicated in both.

## Invariants

- Skill and agent *content* is byte-identical across hosts (format variants exempted, parity-tested).
- A mutation blocked on one host is blocked on every host that supports pre-mutation hooks; genuine host gaps are documented per host, never silently accepted.
- The MCP server is always the globally-installed `archcore` from PATH; no adapter bundles, downloads, or pins a CLI binary.

## Error Handling

- Guard scripts stay authoritative for *what* is blocked and *why* (the reason text); adapters only decide *how* the host is told.
- Fail-open/fail-closed differences per host (e.g. Copilot's preToolUse timeout fails open) are documented in the host's support doc and covered by the probe protocol — not papered over.

## Conformance

- Structure tests pin: the per-host coverage matrix; no host-conditional text in `skills/`; `bin/*` are `#!/bin/sh`; agent format parity.
- A new host counts as "supported" only when all six MUST-provide items exist and dated probe results are recorded.
- PRs that change anything in the portable core, or add adapter logic outside this contract, require maintainer review and a link to an accepted decision document.