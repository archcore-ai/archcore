---
title: "Every MCP Call and Every Hook Resolves Its Own Target Project — the Session Root by Default, an Absolute Root on Request"
status: accepted
tags:
  - "cli"
  - "component:cli"
  - "component:plugin"
  - "hooks"
  - "integrations"
  - "mcp"
  - "multi-host"
---

## Context

`archcore mcp` serves one project root per session: the start-time working directory, re-resolved through `roots/list` (@cli/internal/mcp/root_provider.go), and every tool addresses documents relative to that root. The plugin now installs globally on Claude Code, Codex, and OpenCode, so one server meets sessions started in `~`, in a workspace holding several repositories, or in project A while the task names project B — and on 2026-10-22 a Codex 0.163 session started in an empty folder created `.archcore/` there because the recap advised it. Measured on 2026-10-30: the write guard allows a `Write` into a sibling project's `.archcore/` (@cli/cmd/hook_write_guard.go), and Claude Code 2.1.314 reports every `--add-dir` directory in `roots/list`, so a multi-root session never re-roots under clause 14 of the root resolution contract.

## Decision

Every MCP tool takes an optional `project` argument — an absolute project root that passes the five acceptance checks of the session root provider and overrides the session root and a pinned root for that one call — while `search_documents` with an absolute `for_path`, and every hook with an edited file path, derive the root as the nearest ancestor directory holding `.archcore/`; document paths stay relative to the resolved root.

The session root keeps its contract: a call without `project` and without an absolute path behaves as it does today, including the `roots/list` following of the earlier decision. What that decision rejected — a per-call root argument — is the only part this record replaces.

## Alternatives Considered

1. One MCP server per project in the host config — rejected because a session in `~` or in a workspace parent would need every project listed at config time, and Codex and OpenCode configs carry no per-project interpolation (measured 2026-10-22: `${workspaceFolder}` reaches the server literally through Codex 0.163).
2. A stateful `use_project` tool that switches the server root — rejected because hidden state makes two identical calls answer differently, and a parallel sub-agent switches the root under the main session.
3. An allowlist of `project` values taken from `roots/list` — rejected because Codex declares no `roots` capability and a Claude Code session in `~` reports `~` alone, so the ten-repository case has no allowed value.
4. The nearest `.archcore/` upward from the session working directory for every call — rejected because a call without a path (`list_documents`, `create_document`) has no file to derive from, and a workspace parent with its own `.archcore/` would absorb documents meant for a nested project.
5. A long-lived HTTP daemon, the Hotpot model — rejected because it adds a port and a process lifecycle without answering which project a call addresses; the `project` argument is the part that answers it.

## Consequences

Positive:

- A call without `project` behaves as before; `@cli/internal/mcp/root_provider_test.go` and `@cli/internal/mcp/integration/roots_test.go` pass unchanged. [expected]
- A session in project A writes project B's documents under `B/.archcore/` with `project=<absolute root of B>`; the probe of 2026-10-22 — `list_documents` from a folder without `.archcore/` — returns an error that names `project` instead of a recap that invites `init_project`. [expected]
- A Cursor session pinned with `--project ${workspaceFolder}` (@cli/internal/agents/mcp_helpers.go) reaches a second project through the same argument. [expected] A Copilot entry is plain `archcore mcp` and follows the session root like any other host.
- A `Write` into any `.archcore/*.md` on the machine is denied by the hook, not only one under the session root. [expected]

Negative:

- Eleven tool schemas gain one optional string, and the server instructions gain one sentence inside the 2,048-character cap.
- The agent chooses the root per call; a wrong `project` writes to the wrong project with no error. The `project` field in every response and the post-tool-use validation of the root that response names (@.archcore/integrations/hook-runtime.spec.md behavior 17) are the two checks.
- The `initialize` instructions, built once at start, keep describing the start-time root; `coverage` in `search_documents` names the root a call answered from.
- The write guard gains one upward directory walk per edited path, bounded by the path depth. [expected]
- The rule against absolute paths in MCP responses gains one exception: the `project` field returns the absolute root the caller named, or the root its own `for_path` resolved to.

## Superseded when

- MCP adds a per-request root or workspace field to `tools/call`, which would carry `project` in the protocol instead of eleven schemas.
- Every supported host declares `roots` and reports one root per project of the session, which would let an allowlist replace the acceptance checks as the trust model.
- A machine-level project registry ships and `session-start` lists it, which reopens whether an absolute path stays the only addressing form for `project`.

## Clarifications

- 2026-10-30 — Trust model: any absolute path that passes the five acceptance checks, not an allowlist from `roots/list` (user choice).
- 2026-10-30 — The `project` field in responses carries the absolute root; the rule against absolute paths gains that one exception (user choice, recommended).
- 2026-10-30 — An explicit `project` overrides a pinned `--project` or `ARCHCORE_PROJECT_ROOT` root (user choice, recommended).
- 2026-10-30 — The pre-tool-use matcher stays `Write|Edit`; no `Read` event (user choice, recommended).
- 2026-10-30 — A machine-level project registry is deferred to a separate plan (user choice, recommended).
- 2026-10-30 — Hooks react to a `.archcore/` segment in the edited path, resolving the nearest ancestor root (user choice).
