---
title: "Host Adapter Contract — Portable Core Boundary and Adapter Obligations"
status: accepted
tags:
  - "architecture"
  - "hooks"
  - "multi-host"
  - "plugin"
---

## Purpose & Scope

This spec defines the boundary between the **portable core** of the Archcore plugin and the **per-host adapters**, so that host expansion reuses the core unchanged instead of refactoring it. Normative for everything under `plugins/archcore/` and for adapter code that lives outside this repository, such as an npm wrapper package for OpenCode. Depended on by the four implemented hosts (Claude Code, Cursor, Codex CLI, GitHub Copilot CLI), by the planned OpenCode adapter, and by contributor pull requests, which are reviewed against it. An external adapter is bound by the MUST NOT items below even where the shell-only requirement of `stack-and-tooling.rule` — scoped to this repository — does not reach it. Out of scope: the language and tooling policy itself, and the per-host support documents.

Ownership: the maintainer owns the portable core and this contract, and records a boundary change, a new adapter target, or a hook-semantics mapping before implementation. Contributor work is limited to adapter implementation against an accepted decision.

## Surface

- `plugins/archcore/skills/` — Agent Skills (`SKILL.md`).
- `plugins/archcore/agents/` — markdown agent definitions plus the per-host format variants: Codex TOML in the same directory, and Copilot `*.agent.md` in `copilot-agents/`, which stays outside `agents/` because `.agent.md` matches the `*.md` glob that Claude Code and Cursor use.
- `plugins/archcore/commands/` — slash-command wrappers.
- `plugins/archcore/bin/` — three hook launchers (`session-start`, `pre-tool-use`, `post-tool-use`), the CLI version probe `cli-gte`, the host probe `detect-host`, and the shared shell libraries `@plugins/archcore/bin/lib/normalize-stdin.sh` and `@plugins/archcore/bin/lib/plugin-cache-guard.sh`. Since v0.7.0 (`cli-owns-layers-4-5.adr`) the guard, validation, cascade, and precision policy lives inside `archcore hooks <host> <leaf>`; a launcher carries only the host glue the CLI cannot do for itself.
- Canonical env schema published by `normalize-stdin.sh`: `ARCHCORE_HOST`, `ARCHCORE_TOOL_NAME`, `ARCHCORE_FILE_PATH`, `ARCHCORE_DOC_PATH`, `ARCHCORE_RAW_STDIN`.
- Canonical exit convention of the `bin/` scripts: exit 0 passes; exit 2 plus a reason on stderr blocks. Everything else an adapter ships is configuration, not logic.

## Normative Behavior

1. The plugin MUST keep `skills/`, `agents/`, and `commands/` host-agnostic.
2. A `SKILL.md` MUST NOT carry a host-conditional instruction.
3. The plugin MUST confine host awareness to `plugins/archcore/bin/`.
4. WHEN a hook script varies its output or exit convention by host, the script MUST key that branch on `ARCHCORE_HOST`.
5. An adapter MUST provide a manifest in the location its host searches first.
6. An adapter MUST route the session-start event to `bin/session-start`.
7. An adapter MUST route the pre-mutation guard event to `bin/pre-tool-use`.
8. An adapter's pre-mutation matcher MUST cover source-file edits, not only `.archcore/` paths.
9. An adapter MUST route the post-mutation validation event to `bin/post-tool-use`.
10. An adapter's hook command MUST reach the shared script under every load path its host documents.
11. IF the host's injected plugin-root variable is unconfirmed, THEN the hook command MUST probe each candidate for the script itself with `-x`.
12. IF no candidate resolves, THEN the hook command MUST degrade to the host's pass outcome rather than to an error exit.
13. An adapter MUST register MCP so that it launches `archcore mcp` resolved from PATH.
14. IF plugin-shipped MCP is unsafe on a host, THEN the adapter MUST document a user-side fallback in place of that registration.
15. An adapter MUST add a host case to `@plugins/archcore/bin/lib/normalize-stdin.sh`.
16. An adapter MUST add a coverage-matrix row naming the full set of its host's filesystem-mutation tools.
17. An adapter MUST record a dated probe record in `host-probe-protocol.spec` for each shipped guard.
18. An adapter MUST NOT fork, copy, or patch a skill or an agent for one host.
19. An adapter MUST NOT reimplement guard or validation logic outside `bin/` and the CLI hook leaves those launchers delegate to.
20. WHEN a host exposes a programmatic hook system, the adapter MUST implement its hook as a thin bridge over the `bin/` script.
21. A bridge MUST build the canonical stdin JSON, spawn the script, translate the result into host semantics, and carry no decision logic.
22. An adapter MUST NOT introduce a runtime or a language into `plugins/archcore/` without an accepted ADR.
23. An adapter MUST NOT ship logic in a manifest or in a hook config.
24. A blocking-semantics translation MUST live in exactly one place — either the `bin/` script's host-aware output branch or the bridge's exit translation.

## Constraints & Invariants

| Host | Pre-mutation deny mechanism | Notes |
|---|---|---|
| Claude Code / Codex CLI | exit 2 + stderr | native convention; on Claude Code a PreToolUse timeout fails **open** and prints nothing — the write lands as if no hook existed (probe D, 2026-08-14) |
| Cursor | camelCase events per `cursor.hooks.json` | `Write`-only matcher gap documented |
| GitHub Copilot | stdout JSON `{"permissionDecision":"deny","permissionDecisionReason":…}` with exit 0 | every non-zero exit denies here — exit 2 explicitly, others as `hook errored` — so the JSON arm exists to carry the *reason*, not to make the deny land; timeout fails **open** (hooks-reference, 2026-07-27) |
| OpenCode | bridge throws `Error(reason)` from `tool.execute.before` | model receives a failed tool result carrying the message |

- Constraint: Copilot CLI checks `.plugin/plugin.json`, then `plugin.json`, then `.github/plugin/plugin.json`, then `.claude-plugin/plugin.json`, so a dedicated manifest wins over the shared one. `copilot-adapter-design.adr` takes the dedicated path for that reason rather than relying on the Claude fallback.
- Constraint: a format variant that a host's loader requires (Codex TOML, Copilot `*.agent.md`) is exempt from item 18 and is parity-tested byte for byte.
- Constraint: the candidate probe of items 11 and 12 is the single exception to item 23, and it extends no further than locating a script — never to a decision about what to block.
- Constraint: item 8 covers two PreToolUse concerns arriving on different paths — the write guard on a `.archcore/**/*.md` target, the code-alignment advisory on a source file — both reaching the CLI through one launcher. Cursor's `Write`-only matcher is the documented gap.
- Constraint: items 10–12 are normative rather than quality-of-implementation notes because a host where every non-zero exit denies inverts the cost of a broken guard. Elsewhere a guard that cannot start degrades to no enforcement; there it degrades to locking the user out of their own repository, and the two outcomes are indistinguishable from the host's side. `copilot-adapter-design.adr` records the release that violated all three items and what it cost.
- Invariant: skill and agent content is byte-identical across hosts, format variants exempted.
- Invariant: a mutation blocked on one host is blocked on every host that supports pre-mutation hooks; a genuine host gap is documented per host, never silently accepted.
- Invariant: the MCP server is always the globally installed `archcore` from PATH; no adapter bundles, downloads, or pins a CLI binary.
- Invariant: a hook command's behavior is what tests assert. String equality against a command is not a test of it — the assertion that pinned Copilot's commands character for character matched the defect exactly and shipped it.

## Failure Behavior

1. A guard script MUST stay authoritative for what is blocked and for the reason text.
2. An adapter MUST decide only how its host is told about a block.
3. IF a hook cannot locate its script, THEN the hook MUST report that on stderr.
4. An adapter MUST document its host's fail-open or fail-closed timeout behavior in that host's support document.
5. An adapter MUST record the same timeout behavior under probe D.
6. WHILE a host's pre-mutation hook fails open on timeout, the maintainer MUST treat guard latency as a correctness concern.
7. WHEN the `archcore` CLI is absent or older than 0.7.0, a hook launcher MUST exit 0 without output.

Silence is the worse failure mode behind item 3: enforcement is off, and nothing distinguishes that session from a clean one. Item 6 reaches Claude Code and Copilot alike — Copilot from its documented `timeoutSec: 2` fail-open, Claude Code confirmed by probe D on 2026-08-14. Since v0.7.0 a launcher does only the plugin-cache guard and the `cli-gte` version probe before handing the payload to the CLI. Item 7 exists because a pre-0.7 CLI has no hook leaf, and its usage error would read as a deny on Copilot, where every non-zero exit denies and the reason is discarded.

## Conformance

1. Structure tests pin the per-host coverage matrix, the absence of host-conditional text in `skills/`, the `#!/bin/sh` shebang on `bin/*`, agent format parity, and — per host — that hook commands name only plugin-root variables that host provides.
2. Items 10–12 are conformance-tested by execution: `@test/structure/copilot-plugin.bats` runs each hook command under `env -u` and asserts the pass outcome plus a warning when no root resolves, each candidate sufficient alone, and a dead candidate skipped rather than fatal. Fault injection confirms each assertion fails only for its own defect.
3. Items 6–9 and Failure Behavior items 6–7 are conformance-tested by execution under `test/unit/`: `@test/unit/hook-launchers.bats` asserts the leaf name and host id each launcher passes to the CLI, byte-for-byte stdin, and the silent exit 0 below CLI 0.7.0; `@test/unit/hook-latency.bats` bounds the launcher's own share of the timeout budget, proves through a marginal-cost bound that the glue never reads `.archcore/`, and runs one end-to-end case that skips without a current CLI.
4. A new host counts as supported only when items 5–17 are all satisfied; the probe outcomes that gate the same claim are owned by `host-probe-protocol.spec`.
5. A pull request that changes the portable core, or that adds adapter logic outside this contract, requires maintainer review and a link to an accepted decision document.
