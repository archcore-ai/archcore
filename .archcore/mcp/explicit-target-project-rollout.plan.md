---
title: "Rollout: The project Argument on Every MCP Tool, Path-Derived Hook Roots, and Target-Project Skills"
status: accepted
tags:
  - "component:cli"
  - "component:plugin"
  - "hooks"
  - "mcp"
  - "multi-host"
  - "skills"
---

## Goal

Ship @.archcore/mcp/call-root-resolution.spec.md so that a globally installed plugin and MCP server read and write the `.archcore/` of any project: the session root by default, unchanged for every existing caller, and any other project by its absolute root. Done when the two scenarios under `.archcore/mcp/` pass against a built CLI and the plugin test suite, and no existing test changes its expectation except the byte-budget oracles that the second exception of the first acceptance criterion names.

## Tasks

### Phase 1 — Server: call root

1. Add `project` as an optional string argument to all 11 tool definitions and a shared resolver that returns the call root: `project` → absolute `for_path` ancestor → session root. @cli/internal/mcp/tools/root.go, @cli/internal/mcp/tools/common.go, @cli/internal/mcp/server.go.
2. Run the five acceptance checks on `project` after `filepath.EvalSymlinks`; reuse the check code of the session provider. @cli/internal/mcp/root_provider.go, @cli/internal/projectroot/.
3. Make the resolver override a pinned root for the one call. @cli/cmd/mcp_root.go, @cli/internal/mcp/root_provider.go.
4. Add the `project` field to every tool response. @cli/internal/mcp/tools/*.go (11 handlers), @cli/internal/mcp/tools/docs_bridge.go.
5. Let `search_documents` accept an absolute `for_path` and derive the root; reject a relative path that escapes the root as before. @cli/internal/mcp/tools/search_documents.go, @cli/internal/mcp/tools/search_file_context.go, @cli/internal/docs/guard.go.
6. Make `add_relation` and `remove_relation` refuse a source and target under two roots with an error that holds no absolute path. @cli/internal/mcp/tools/add_relation.go, @cli/internal/mcp/tools/remove_relation.go.
7. Let `init_project` take `project` and create `.archcore/` only there. @cli/internal/mcp/tools/init_project.go.
8. Add the `project` argument and the response field to the `initialize` instructions. @cli/internal/mcp/server.go (`buildInstructions`), @cli/internal/agents/instructions.go.
9. Write the error texts per Failure Behavior 1–9 of the spec through `SanitizeError`; no error echoes a `project` or `for_path` value. @cli/internal/mcp/tools/common.go.

### Phase 2 — Hooks: path-derived root

10. Resolve the write-guard root per target as the nearest ancestor holding `.archcore/`; read a relative path against the payload cwd first. @cli/cmd/hook_write_guard.go, @cli/cmd/hook_command.go (`resolveBaseDir`), @cli/internal/docs/guard.go, @cli/internal/docs/relpath.go.
11. Add the first-touch advisory for a target under another root, claimed by a dedup stamp keyed by session id, root, and host in a directory of its own. @cli/cmd/hook_project_root.go (`projectTouchAdvisory`).
12. Read `project` from `tool_input` in post-tool-use and validate under that root. @cli/cmd/hook_post_tool_use.go, @cli/cmd/hook_payload.go.
13. Change session-start for a cwd without `.archcore/`: list the project roots found by the bounded sources, name the `project` argument, and drop the `create_document` and `init_project` advice. @cli/cmd/hook_session_start.go, @plugin/plugins/archcore/bin/session-start.
14. Keep the pre-tool-use matcher `Write|Edit` on Claude Code and the per-host matcher sets unchanged. @plugin/plugins/archcore/hooks/hooks.json, @plugin/plugins/archcore/hooks/cursor.hooks.json, @plugin/plugins/archcore/hooks/codex.hooks.json, @plugin/plugins/archcore/hooks/copilot.hooks.json — no edit expected; the test of task 20 pins it.

### Phase 3 — Skills: target project

15. Add a target-project contract to `_shared`: one target project per path in the request; grounding and writes per project with `project`; `init_project` only for a named project after the user's answer. @plugin/plugins/archcore/skills/_shared/gate-contract.md (Context resolution), new file @plugin/plugins/archcore/skills/_shared/target-project.md.
16. Replace the unasked `init_project` call in the three skills with the question of command-surface-v2.spec Failure Behavior 2. @plugin/plugins/archcore/skills/plan/SKILL.md (step 6), @plugin/plugins/archcore/skills/document/SKILL.md (line 136), @plugin/plugins/archcore/skills/review/SKILL.md (line 53).
17. Make the plan report list each produced document beside its project. @plugin/plugins/archcore/skills/plan/SKILL.md (Result).
18. Mirror the skill text into the Codex, Cursor, Copilot, and OpenCode packages per the existing export. @plugin/plugins/opencode/, @plugin/plugins/archcore/copilot-agents/ — no target found for the export script; locate it before editing.

### Phase 4 — Tests and documentation

19. Add server tests: `project` on each tool; `project` overriding a pinned root; absolute `for_path` deriving the root; each of the five rejections; symlinked duplicate roots; a relation across two roots; `init_project` with `project`; a no-arg call on a pinned and on a session root unchanged. @cli/internal/mcp/integration/roots_test.go, @cli/internal/mcp/root_provider_test.go, @cli/internal/mcp/tools/*_test.go.
20. Add hook tests: a write into a sibling project's `.archcore/` denied; a relative target unchanged; the first-touch advisory once per session and root; post-tool-use under `project`; session-start in a folder without `.archcore/` listing roots and naming no `init_project`; matchers unchanged. @cli/cmd/hook_write_guard_test.go, @cli/cmd/hook_session_start_test.go, @cli/cmd/hook_post_tool_use_test.go, @plugin/test/structure/hooks.bats, @plugin/test/unit/session-start-goldens.bats.
21. Add plugin structure tests for the target-project contract and the removed unasked `init_project`. @plugin/test/structure/command-grammar.bats.
22. Run the two scenarios as a manual probe on Claude Code and Codex with the probe server and two temporary projects; record the result under `## Clarifications` of the ADR. @.archcore/mcp/cross-project-session.scenario.md, @.archcore/mcp/session-outside-a-project.scenario.md.
23. Update the CLI hooks reference and the plugin usage catalog with the `project` argument and the session-start change. @.archcore/integrations/cli-hooks-reference.doc.md, @.archcore/plugin/plugin-usage-catalog.doc.md, @plugin/plugins/archcore/README.md.

## Acceptance Criteria

- Every tool call without `project` returns the same result as the release before this plan on the same corpus; the existing integration suites pass unchanged. Three exceptions are intended:
  1. Every successful response carries the `project` field, per behavior 10 of @.archcore/mcp/call-root-resolution.spec.md. A client that ignores unknown fields reads the rest of the response unchanged.
  2. A response with a byte budget reserves the bytes of the `project` field, so a response at its budget holds one row or one near miss fewer. The oracle in @cli/internal/mcp/tools/search_near_misses_spec_test.go subtracts that reservation.
  3. A `search_documents` call with an absolute `for_path` inside a nested project answers from the nested project, per behavior 5 of the spec.
- `create_document(project="/w/b", …)` from a session rooted in `/w/a` writes under `/w/b/.archcore/` and returns `project: "/w/b"`.
- `search_documents(for_path="/w/b/src/api.go")` from the same session returns B's rows and `project: "/w/b"`.
- A `Write` into `/w/b/.archcore/x.spec.md` from a session rooted in `/w/a` is denied with a reason naming `update_document` and `project`.
- A session started in a folder without `.archcore/` receives a recap that lists reachable roots and no `init_project` advice, and `/archcore:plan` there creates no `.archcore/` before the user answers.
- `create_document(project="/w/c", …)` where `/w/c` holds no `.archcore/` returns an error naming `init_project` and leaves `/w/c` unchanged.
- `add_relation` across two roots returns an error without an absolute path.
- No error message contains an absolute path; `project` is the only response field that does.
- Project-level MCP configs (`.mcp.json`, `.cursor/mcp.json` with `--project`, `opencode.json`) still serve their pinned root to a call without `project`.

## Dependencies

- @.archcore/mcp/call-root-resolution.spec.md — the contract this plan ships.
- @.archcore/mcp/explicit-target-project-per-call.adr.md — the decision; supersedes the session-following ADR's rejection of a per-call root.
- @.archcore/mcp/project-root-resolution.spec.md — the session root this plan keeps as the default.
- @.archcore/integrations/hook-runtime.spec.md, @.archcore/plugin/hooks-validation-system.spec.md — amended hook behaviors 14–17 and the session-start phase 2 change.
- @.archcore/plugin/command-surface-v2.spec.md — behaviors 49–52 and Failure Behavior 2.
- Open: Codex `post-tool-use` payload shape for `tool_input` (scenario open question); OpenCode edit-tool coverage of the pre-tool-use leaf. Both are measured in task 22 before task 12 and task 14 close.
- Deferred to a later plan: a machine-level project registry under the XDG state dir (user's choice, 2026-10-30).

## Declared Delta

- create adr mcp/explicit-target-project-per-call — done
- create spec mcp/call-root-resolution — done
- amend spec mcp/project-root-resolution §1, §14a, Does Not Cover, constraints — done
- amend spec integrations/hook-runtime behaviors 14–17, Failure 5 — done
- amend spec plugin/hooks-validation-system Hook 1 phase 2, write-guard invariant, conformance 2 — done
- amend spec mcp/search-documents inputs `project`, absolute `for_path`, response `project`, §1.15–17, Security — done
- amend spec integrations/file-context-resolution input, Failure 3 and 3a — done
- amend spec plugin/command-surface-v2 behaviors 49–52, Failure 2 — done
- amend rule mcp/no-absolute-paths-in-mcp-errors items 4–5 — done
- create scenario mcp/cross-project-session — done
- create scenario mcp/session-outside-a-project — done
- create plan mcp/explicit-target-project-rollout — this document
- none: machine-level project registry — deferred