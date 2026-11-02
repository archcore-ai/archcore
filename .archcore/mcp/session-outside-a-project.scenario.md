---
title: "Session Outside a Project — Workspace Parent, Home Directory, Pinned Host Root, and a Misrouted Plugin Cache"
status: accepted
tags:
  - "actor:plugin-user"
  - "component:cli"
  - "component:plugin"
  - "hooks"
  - "mcp"
  - "multi-host"
---

## Subject

A session that does not start in a project root: a workspace parent holding nested projects, the home directory, a host whose server is pinned to one root, and a host that spawned the server inside a plugin cache. Illustrates `call-root-resolution.spec` clauses 2, 7, 12, 13, 14, 18 and Failure Behavior 3, 4, 6, 9. Plugin users read it to know what the recap says and what the agent can do; the session-start hook depends on its examples.

## Actors

| Actor | Who they are | What they want |
|---|---|---|
| Developer | a user who opens the agent wherever the shell is | no `.archcore/` created where none was wanted |
| Agent | the coding agent in that session | a list of the projects it can address |

## Flows

### Developer

Anchors: @plugin/plugins/archcore/bin/session-start, @cli/cmd/hook_session_start.go

1. Developer opens a session in `/w`, which holds `/w/a/.archcore/` and `/w/b/.archcore/`; the recap lists both roots.
2. Developer asks for a document about `/w/b`; Agent writes it with `project=/w/b`.
3. Developer opens a session in `~`; the recap says no project is here and names `project` and `for_path`.
4. Developer names ten repositories by path; Agent addresses each with its own `project`.
5. Developer opens Cursor, whose server is pinned to `/w/a`; Agent reaches `/w/b` with `project=/w/b`.

Extensions:

- 1a. `/w` holds its own `.archcore/`; the recap serves `/w` and lists `/w/a` and `/w/b` as nested roots.
- 3a. Developer asks to record a decision with no path named; Agent asks which project, and creates nothing.
- 5a. Developer runs Copilot, whose server was spawned inside the plugin cache; `project=/w/b` still works.

### Agent

Anchors: @cli/internal/mcp/tools/init_project.go, @cli/internal/mcp/root_provider.go, @cli/internal/projectroot/

1. Agent calls `list_documents()` in `/w` with no `.archcore/`; the server returns the uninitialized-root answer.
2. Agent calls `list_documents(project="/w/a")`; the server answers from `/w/a/.archcore/`.
3. Agent calls `init_project(project="/w/c")` after Developer asked for it; the server creates `/w/c/.archcore/`.
4. Agent calls `create_document(project="~/.cursor/plugins/cache/x")`; the server refuses and names the cache kind.
5. Agent calls `search_documents(for_path="/tmp/notes.md")`; the server refuses and names `project`.

Extensions:

- 3a. Developer asked for nothing; Agent calls no `init_project` because the recap offered no such step.

## Examples

Background: the plugin is installed at user scope; `/w/a` and `/w/b` hold `.archcore/`; `/w` and `~` do not.

### A workspace parent without its own store

Illustrates: clauses 7, 18; Failure Behavior 9.
Given the session started in `/w`.
When the session starts.
Then the recap lists `/w/a` and `/w/b` and names `project`, and offers no `init_project`.

### The home directory

Illustrates: Failure Behavior 9.
Given the session started in `~`.
When Agent calls `list_documents()`.
Then the response is the uninitialized-root answer and no `.archcore/` is created in `~`.

### A pinned host reaches a second project

Illustrates: clause 2.
Given the server runs with `--project /w/a`.
When Agent calls `get_document(project="/w/b", path=".archcore/api.spec.md")`.
Then the response carries B's document and `project: "/w/b"`.

### A project that holds no store

Illustrates: clause 12; Failure Behavior 4.
Given `/w/c` exists without `.archcore/`.
When Agent calls `create_document(project="/w/c", type="adr", filename="x")`.
Then the server returns an error naming `init_project` and `project`, and `/w/c` is unchanged.

### A misrouted cache as the target

Illustrates: Failure Behavior 3.
When Agent calls `list_documents(project="/Users/dev/.cursor/plugins/cache/archcore")`.
Then the server returns an error naming the plugin cache and no absolute path.

### An edit in a nested project from the parent session

Illustrates: clauses 13, 14.
Given the session started in `/w` and `/w/a/.archcore/x.adr.md` exists.
When Agent edits `/w/a/.archcore/x.adr.md` with the editor.
Then the hook denies the write against the root `/w/a`.

## Open Questions

- Does OpenCode ship an edit tool other than `write`, `edit`, `apply_patch`, and `patch` that changes a file without reaching the pre-tool-use leaf?

## Clarifications

- 2026-11-02 — The session-start inventory descends at most 3 levels below the session folder and reads at most 2000 directories; the recap adds "(the search stopped at its bound)" when a bound cuts the walk. The walk has no time bound of its own. Both values are chosen, not measured [assumption]. Source: `inventoryDepth` and `inventoryDirs` in @cli/cmd/hook_project_root.go, pinned by `TestWriteProjectInventory_DepthBound` and `TestWriteProjectInventory_DirectoryBound`.
- 2026-11-02 — The OpenCode bridge hands the call arguments of `write`, `edit`, `apply_patch`, and `patch` to the pre-tool-use leaf as `tool_input` (`PRE_MUTATION` in @plugin/plugins/opencode/src/bridge.ts). The host probe run of 2026-10-30 on OpenCode 2.1.4 passed the write-guard and project-touch probes (@.archcore/plugin/host-probe-protocol.spec.md).