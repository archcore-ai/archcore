---
title: "Call Root Resolution — The project Argument and the Path-Derived Root of One Tool Call or Hook Invocation"
status: accepted
tags:
  - "cli"
  - "component:cli"
  - "hooks"
  - "integrations"
  - "mcp"
  - "multi-host"
---

## Purpose & Scope

This spec defines the call root: the project root one MCP tool call or one hook invocation reads and writes. It is normative for the call-root resolver the eleven tool handlers consult (@cli/internal/mcp/tools/root.go, @cli/internal/mcp/root_provider.go), for the `project` argument of every tool, for the path-derived root of `search_documents` under `for_path`, and for the root the hook leaves resolve from an edited file path (@cli/cmd/hook_command.go, @cli/cmd/hook_write_guard.go). Dependents: coding agents on every supported host, the plugin skills, and the hook launchers.

Out of scope: how the session root is chosen and followed through `roots/list` (the project root resolution contract); the file-context rows themselves; cross-project relations, which no tool supports.

## Surface

- `project` — optional string on every tool: an absolute path to a project root, the directory that holds `.archcore/`, never `.archcore/` itself.
- Response field `project` — the absolute root the call resolved.
- Acceptance checks — the five checks of the project root resolution contract, shared with the session root provider.
- Precedence — `project`, then an absolute `for_path`, then the session root.

## Normative Behavior

1. WHEN a call carries `project`, the resolver MUST serve that root for the call.
2. WHEN a call carries `project`, the resolver MUST ignore the session root and a pinned root for that call.
3. The resolver MUST accept a `project` value only when every acceptance check passes.
4. The resolver MUST evaluate symlinks of a `project` value before the checks and before any comparison with another root.
5. WHEN `search_documents` carries an absolute `for_path` and no `project`, the resolver MUST serve the nearest ancestor of `for_path` that holds `.archcore/`.
6. WHEN that ancestor differs from the session root, the resolver MUST apply the acceptance checks to it.
7. WHEN a call carries neither `project` nor an absolute `for_path`, the resolver MUST serve the session root.
8. A call MUST read and write exactly one root.
9. WHEN a tool returns a document path, the tool MUST return it relative to the call root, with the `.archcore/` prefix.
10. WHEN a tool returns a document path, the tool MUST return the call root in the `project` field.
11. WHEN `create_document`, `update_document`, `remove_document`, `add_relation`, or `remove_relation` resolves a root other than the session root, the tool MUST write under that root's `.archcore/` only.
12. WHEN `init_project` carries `project`, the tool MUST create `.archcore/` under that path only when the path exists, is a directory, and is not a plugin install cache.
13. WHEN a hook payload names an edited file, the hook MUST resolve the call root as the nearest ancestor of that file holding `.archcore/`.
14. WHEN the edited file has no such ancestor, the hook MUST resolve the call root from the payload working directory.
15. WHEN the resolved hook root differs from the payload working directory root, the write guard MUST decide against the resolved root.
16. WHEN a post-tool-use hook validates a document, the hook MUST resolve the root per @.archcore/integrations/hook-runtime.spec.md behavior 17.
17. The resolver MUST NOT keep a `project` value beyond the call that carried it.
18. The server MUST describe `project` in the instructions it sends at `initialize`, inside the instruction cap.
19. WHEN a call succeeds and the call root is the session root, the server MUST append `"source": "local"` after the `project` field.
20. WHEN the session root declares the call root as a global source, the server MUST append the declared id as `source` after `project`.
21. WHEN a call succeeds and the session root does not declare the call root, the server MUST NOT append `source`.
22. WHEN the tool is `add_relation` or `remove_relation`, the server MUST NOT append `source`. The payload of both tools already uses `source` for the relation endpoint.

## Constraints & Invariants

| Constraint | Value | Rationale |
| ---------- | ----- | --------- |
| Roots per call | exactly 1 | A relation joins two documents of one manifest; a second root has no manifest to join |
| `project` form | absolute path to the project root | A relative value would resolve against a working directory no host guarantees |
| Upward walk bound | the depth of the edited path | The write guard runs inside the one-second hook budget |
| Trust of `project` | the five acceptance checks | The same footing as a `roots/list` candidate; no allowlist |
| Instruction growth | 1 sentence | The instruction cap of 2,048 characters holds |

- Invariant: a call without `project` and without an absolute `for_path` answers exactly as a call did before this contract.
- Invariant: a document path in a response never carries an absolute prefix; the `project` field is the only absolute path a response carries.
- Invariant: a `project` value that fails a check changes no state and leaves the session root as it was.
- Invariant: process-global caches keyed by root — the scan cache, the manifest store, the main-checkout memo — serve several roots in one process without cross-talk, because a key names one root.
- Invariant: two `project` values that evaluate to one directory are one root.

## Failure Behavior

1. IF `project` is not an absolute path, THEN the tool MUST return an error naming the required form and no path.
2. IF `project` does not exist or is not a directory, THEN the tool MUST return an error that holds no absolute path.
3. IF `project` sits in a plugin install cache, THEN the tool MUST return an error naming the cache kind.
4. IF `project` holds no `.archcore/`, THEN every tool except `init_project` MUST return an error naming `init_project` and the `project` argument.
5. IF the declared global sources of `project` hold a fatal state, THEN the tool MUST return an error naming the declared source id.
6. IF an absolute `for_path` has no ancestor holding `.archcore/`, THEN `search_documents` MUST return an error naming `project` as the recovery.
7. IF `add_relation` or `remove_relation` receives document paths that resolve under two roots, THEN the tool MUST return an error stating that relations stay inside one project.
8. IF a hook cannot read the directories above the edited file, THEN the hook MUST resolve from the payload working directory.
9. IF the session root holds no `.archcore/` and the call carries no `project`, THEN the tool MUST answer as it does today for an uninitialized root.
10. IF `update_document` or `remove_document` receives a path that starts with `../`, THEN the tool MUST return an error that names `project` and no absolute path.
11. IF that path lies under a global source the call root declares, THEN the error MUST also name the source id.

## Conformance

An implementation is conformant when it satisfies behaviors 1–22, holds every constraint and invariant, and degrades per the failure rules.

Given a session started in `/w/a` and a second project at `/w/b`, each holding `.archcore/`,
When the agent calls `create_document(project="/w/b", type="adr", filename="x")`,
Then the file lands at `/w/b/.archcore/x.adr.md` and the response carries `project: "/w/b"` and `path: ".archcore/x.adr.md"`.

## Clarifications

- 2026-11-02 — A Codex `post-tool-use` payload carries `tool_input` with the call arguments: in the host probe run of 2026-10-30 on Codex 0.166.0, `post-tool-use` validated the document of the second project named by `project` (@.archcore/plugin/host-probe-protocol.spec.md).
- 2026-11-02 — The grounding report of a plugin skill keeps global sources per project: `skills/_shared/target-project.md` runs grounding in each target project, and each project's grounding reads the global sources that project declares. This is read from the skill contract, not measured in a session [assumption].
