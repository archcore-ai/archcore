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

Formalize the boundary between the **portable core** of the Archcore plugin and the **per-host adapters**, so host expansion (OpenCode next) reuses the core unchanged instead of refactoring it. This spec is the contract every new host adapter is built against — and the gate contributor PRs are reviewed against.

**Ownership:** the portable core and this contract are maintainer-owned. Key design decisions (boundary changes, new adapter targets, hook-semantics mappings) are recorded by the maintainer before implementation; contributor work is limited to adapter implementation against an accepted decision. See `stack-and-tooling.rule` for the language/tooling side of the same policy.

## Scope

Covers the four implemented hosts (Claude Code, Cursor, Codex CLI, GitHub Copilot CLI), the one planned adapter (OpenCode), and any future host. Applies to everything under `plugins/archcore/` and to adapter code that lives outside this repo (e.g. an npm wrapper package for OpenCode): external adapters are bound by the same MUST NOT rules even where the stack rule's shell-only requirement (which is scoped to this repo) does not reach them.

## Portable core (normative)

The core is maintained once and is host-agnostic:

- `plugins/archcore/skills/` — Agent Skills (SKILL.md). MUST NOT contain host-conditional instructions.
- `plugins/archcore/agents/` — markdown agent definitions plus per-host format variants (Codex TOML in the same directory; Copilot `*.agent.md` in `copilot-agents/`, which must stay outside `agents/` because `.agent.md` matches the `*.md` glob Claude Code and Cursor use), kept in parity by structure tests.
- `plugins/archcore/commands/` — slash-command wrappers.
- `plugins/archcore/bin/` — hook scripts + `bin/lib/normalize-stdin.sh`. POSIX sh only (stack rule). Host awareness is allowed **only** here: stdin detection and field extraction in `normalize-stdin.sh`, and host-conditional *output/exit conventions* in hook scripts (e.g. a Copilot deny-JSON branch), keyed on `ARCHCORE_HOST`.

Everything else an adapter ships is **configuration, not logic**.

## Normative Behavior — adapter MUST provide

1. **Manifest** the host discovers, in the location that host searches first. Copilot CLI checks `.plugin/plugin.json`, then `plugin.json`, then `.github/plugin/plugin.json`, then `.claude-plugin/plugin.json` — so a dedicated manifest wins over the shared one, and `copilot-adapter-design.adr` takes the dedicated path deliberately rather than relying on the Claude fallback.
2. **Hook wiring** for the three canonical lifecycle events, each routed to the shared `bin/` scripts:
   - session-start → `bin/session-start`
   - pre-mutation guard → `bin/check-archcore-write` (+ `bin/check-code-alignment` for source edits)
   - post-mutation validation → `bin/validate-archcore` (+ cascade/precision checks)
3. **A resolvable path to those scripts.** The hook command MUST reach the shared script under every load path the host documents.
   - Where the host's injected plugin-root variable is unconfirmed, the command MUST probe each candidate for the script itself (`-x`) instead of trusting one name.
   - When no candidate resolves, the command MUST degrade to the host's pass outcome rather than to an error exit.
   - Conformance for this item MUST come from executing the command; see Conformance below for why inspecting it does not count. `copilot-adapter-design.adr` records the release that violated all three and what it cost.
4. **MCP registration** launching `archcore mcp` resolved from PATH — or a documented user-side fallback where plugin-shipped MCP is unsafe. Two hosts now take that fallback for the same reason (the host launches the MCP child outside the user's project): `cursor-mcp-architecture.adr` and `copilot-mcp-architecture.adr`.
5. **A host case in `bin/lib/normalize-stdin.sh`** — detection plus the canonical env schema (`ARCHCORE_HOST`, `ARCHCORE_TOOL_NAME`, `ARCHCORE_FILE_PATH`, …).
6. **A mutation-tool coverage-matrix row**: the full set of the host's filesystem-mutation tools, asserted by a structure test so a coverage gap fails CI instead of surfacing when a user trips it.
7. **A dated probe record** in `host-probe-protocol.spec` — one probe per shipped guard (`P0` gate, then `A` main-session source write, `A-d` delegated write, `B` `.archcore/` write, `C` MCP validation), with unavailable probes recorded as `n/a` or `deferred` rather than omitted.

## Normative Behavior — adapter MUST NOT

- Fork or copy skills/agents per host, or patch skill text for one host. Format variants that a host's loader requires (Codex TOML, Copilot `*.agent.md`) are exempt and parity-tested byte-for-byte.
- Reimplement guard/validation logic outside `bin/` — **including in adapter code written in other languages**. Hosts with programmatic hook systems (OpenCode JS/TS plugins) implement hooks as thin bridges: build the canonical stdin JSON, spawn the `bin/` script, translate its exit/output into host semantics. No decision logic lives in the bridge.
- Introduce runtimes or languages into `plugins/archcore/` without an accepted ADR (`stack-and-tooling.rule`).
- Ship logic in manifests or hook configs (configuration only). Path resolution per item 3 is the single exception, and it extends no further than a candidate probe — never to a decision about what to block.

## Constraints — blocking-semantics translation per host

The canonical convention is defined by the `bin/` scripts: exit 0 = pass; block = exit 2 + reason on stderr (the Claude Code / Codex native semantics). Adapters translate:

| Host | Pre-mutation deny mechanism | Notes |
|---|---|---|
| Claude Code / Codex CLI | exit 2 + stderr | native convention |
| Cursor | camelCase events per `cursor.hooks.json` | `Write`-only matcher gap documented |
| GitHub Copilot | stdout JSON `{"permissionDecision":"deny","permissionDecisionReason":…}` with exit 0 | every non-zero exit denies here — exit 2 explicitly, others as `hook errored` — so the JSON arm exists to carry the *reason*, not to make the deny land; timeout fails **open** (hooks-reference, 2026-07-27) |
| OpenCode | bridge throws `Error(reason)` from `tool.execute.before` | model receives a failed tool result carrying the message |

A given translation lives in exactly one place: either the `bin/` script's host-aware output branch (shell) or the bridge's mechanical exit-to-host translation (adapter) — never duplicated in both.

**A host where every non-zero exit denies inverts the cost of a broken guard.** Elsewhere a guard that cannot start degrades to no enforcement; there it degrades to locking the user out of their own repository, and the two are indistinguishable from the host's side. That asymmetry is why item 3 is normative rather than a quality-of-implementation note.

## Invariants

- Skill and agent *content* is byte-identical across hosts (format variants exempted, parity-tested).
- A mutation blocked on one host is blocked on every host that supports pre-mutation hooks; genuine host gaps are documented per host, never silently accepted.
- The MCP server is always the globally-installed `archcore` from PATH; no adapter bundles, downloads, or pins a CLI binary.
- A hook command's *behavior* is what tests assert. String equality against a command is not a test of it: the assertion that pinned Copilot's commands character-for-character matched the defect exactly and shipped it.

## Error Handling

- Guard scripts stay authoritative for *what* is blocked and *why* (the reason text); adapters only decide *how* the host is told.
- Fail-open/fail-closed differences per host are documented in the host's support doc and observed by probe D, never papered over. Copilot's `preToolUse` timeout fails open, which makes guard latency a correctness concern there rather than a comfort one — `test/unit/hook-latency.bats` keeps both PreToolUse scripts far enough inside the 1 s budget that the timeout path stays unreachable.
- A hook that cannot locate its script MUST report that on stderr. Silence is the worse failure mode: enforcement is off, and nothing distinguishes that session from a clean one.

## Conformance

- Structure tests pin: the per-host coverage matrix; no host-conditional text in `skills/`; `bin/*` are `#!/bin/sh`; agent format parity; and, per host, that hook commands name only plugin-root variables that host provides.
- Item 3 is conformance-tested by execution: `test/structure/copilot-plugin.bats` runs each hook command under `env -u` and asserts the pass outcome plus a warning when no root resolves, each candidate sufficient alone, and a dead candidate skipped rather than fatal. Fault injection confirms each assertion fails only for its own defect.
- A new host counts as "supported" only when all seven MUST-provide items exist and its row in `host-probe-protocol.spec` records `pass` or a justified `n/a` for P0, A, B and C.
- PRs that change anything in the portable core, or add adapter logic outside this contract, require maintainer review and a link to an accepted decision document.
