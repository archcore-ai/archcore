---
title: "Session Project Resolution — The Nearest Store at or Above the Session Folder, and the Answer of a Root Without a Store"
status: draft
tags:
  - "cli"
  - "component:cli"
  - "hooks"
  - "integrations"
  - "mcp"
  - "multi-host"
---

## Purpose & Scope

This spec defines the project a session works on when a call names none, and what a tool answers on a root that holds no `.archcore/`. It is normative for `projectroot.Resolve` and `projectroot.ForSession` (@cli/internal/projectroot/nearest.go), the session root provider (@cli/internal/mcp/root_provider.go), the call-root wrapper (@cli/internal/mcp/call_root.go), the hook request (@cli/cmd/hook_command.go), and the plugin launcher (@plugin/plugins/archcore/bin/session-start).

Depended on by every MCP tool call without `project`, by the three `archcore hooks` leaves, and by the plugin skills that read the session root.

Out of scope: how the session folder is chosen and followed through `roots/list` (@.archcore/mcp/project-root-resolution.spec.md); the `project` argument and an absolute `for_path` (@.archcore/mcp/call-root-resolution.spec.md); the text of the session recap (@.archcore/integrations/session-start-context.spec.md); short-lived commands such as `archcore init` and `archcore doctor`, which keep the directory they are given.

## Surface

- **Session folder** — the folder the session works in: the start-time working directory, a pinned `--project` or `ARCHCORE_PROJECT_ROOT` value, an adopted `roots/list` folder, or the working directory of a hook payload.
- **Session project** — `projectroot.ForSession(folder)`: the root the resolver finds, else the session folder itself.
- **Store** — the `.archcore/` directory of a root.
- **Initialized store** — a store that holds a regular file `settings.json`. Every `init_project`, `install_host_config`, and `archcore init` writes one; the implicit store of `create_document` up to 0.15.5 has none.
- **Storeless reads** — `list_documents`, `search_documents`, `get_document`, `list_relations`.
- **Store-creating tools** — `init_project`, and `install_host_config` on the session project. A `project` argument that names a folder without a store is accepted by `init_project` only (@.archcore/mcp/call-root-resolution.spec.md, Failure Behavior 4).
- The response field `"initialized": false`, and the refusal text of a document write on a root without a store.
- **New nested store** — a `.archcore/` directory that does not exist yet, in a folder whose session project is another folder.

## Normative Behavior

1. The resolver MUST return the nearest directory at or above the session folder that holds `.archcore/`.
2. WHEN the session folder is reached through a symbolic link, the resolver MUST also resolve the evaluated path.
3. WHEN the evaluated path resolves to another directory, the resolver MUST return the root of the evaluated path.
4. WHEN no directory at or above the session folder holds `.archcore/`, `projectroot.ForSession` MUST return the session folder.
5. The MCP server MUST resolve the session project on every tool call that names no `project`.
6. The MCP server MUST apply the resolver to a start-time folder, a pinned folder, and an adopted folder alike.
7. Each `archcore hooks` leaf MUST take the session project of the payload working directory as its session root.
8. WHEN a hook target path is relative, the leaf MUST read the path against the payload working directory.
9. WHEN the call root holds no store, the server MUST refuse every tool outside the storeless reads and the store-creating tools.
10. WHEN the server refuses under behavior 9, the server MUST NOT create a file or a directory.
11. The refusal MUST name `init_project` and the `project` argument.
12. WHEN `list_documents`, `search_documents`, or `list_relations` succeeds on a root without a store, the server MUST append `"initialized": false` to the response object.
13. The server MUST write the `initialized` field after the `project` and `source` fields.
14. WHEN the call root holds a store, the server MUST NOT append the `initialized` field.
15. WHEN `init_project` names no `project`, the tool MUST act on the session project.
16. The tool `init_project` MUST create a store below an existing project only for a call that names that folder in `project`.
17. WHEN the server registers a tool this spec does not class, the server MUST treat that tool as refused under behavior 9.
18. WHEN the root the resolver finds is above the session folder and holds no initialized store, `projectroot.ForSession` MUST return the session folder.
19. WHEN `add_relation` or `remove_relation` runs on a root without a store, the server MUST return the refusal of behavior 9.
20. WHEN a hook target would create a new nested store, the pre-tool-use leaf MUST deny the write.
21. WHEN the leaf denies under behavior 20, the leaf MUST name `create_document` and `init_project` in the reason, with no path.
22. WHEN the launcher resolves the session project, it MUST take the session folder if it holds `.archcore/`, else the nearest ancestor with an initialized store.
23. WHEN the session folder is below an initialized project, the launcher MUST NOT advise `archcore init` with the session folder as `--project`.

## Constraints & Invariants

- Invariant: one function, `projectroot.ForSession`, decides the session project for the MCP server and for the hooks.
- Invariant: a session whose folder holds a store gets the tool responses and the hook output it got before this contract, byte for byte, with or without `settings.json`.
- Invariant: only a store-creating tool creates `.archcore/`.
- Invariant: the server reads the store on every call, so a store that appears or disappears is seen by the next call.
- Constraint: the ascent has no git boundary; it stops at the filesystem root, or at a directory the process may not search.
- Constraint: a relative `for_path`, and a relative document path, is read against the session project, not against the session folder.
- Constraint: a store without `settings.json` below an initialized project stays the nearest store for an edited file under it (`projectroot.Resolve`); behavior 18 keeps it from becoming the session project of a folder below it.
- Constraint: the server follows a session to another folder only through `roots/list`. Claude Code 2.1.322 did not answer that query on 2026-11-16, so on that host the session folder is the start-time folder for the process lifetime.

## Failure Behavior

1. IF the server cannot read the store directory of the call root, THEN the server MUST treat the store as absent.
2. IF a directory above the session folder is not searchable, THEN the resolver MUST return the session folder.
3. IF `get_document` runs on a root without a store, THEN the tool MUST return its document-not-found error unchanged.
4. IF `init_project` cannot write the settings, THEN the tool MUST return an error that holds no absolute path.
5. IF the resolver cannot read `settings.json` of a store above the session folder, THEN `projectroot.ForSession` MUST treat that store as not initialized.

## Conformance

An implementation is conformant when it satisfies behaviors 1–23, holds every invariant, and degrades per the failure rules. Checks: `TestResolve` in @cli/internal/projectroot/nearest_test.go; @cli/internal/mcp/integration/session_project_spec_test.go; @cli/internal/mcp/integration/session_project_instructions_test.go; `TestStoreClasses_AgreeWithTheRegistry` and `TestCallRootResolver_WrapRunsStoreCreatingToolsWithoutAStore` in @cli/internal/mcp/call_root_test.go; `TestBinary_CallRoot` in @cli/internal/mcp/integration/call_root_binary_test.go; `TestSessionStart_BelowAProjectRoot`, `TestPreToolUse_SessionBelowAProjectRoot`, and `TestPostToolUse_SessionBelowAProjectRoot` in @cli/cmd/hook_project_root_test.go; `TestHandleSessionStart_KeyIsTheSessionProject` in @cli/cmd/hooks_session_dedup_spec_test.go; `TestWriteGuard_NewStoreInsideAProjectIsBlocked` in @cli/cmd/hook_write_guard_test.go; @plugin/test/unit/session-start-outside-project.bats.
