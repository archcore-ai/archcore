---
title: "Cross-Project Session — A Plan for Two Repositories From a Session Started in One of Them"
status: accepted
tags:
  - "actor:plugin-user"
  - "component:cli"
  - "component:plugin"
  - "hooks"
  - "mcp"
---

## Subject

A session started in project A plans and records work for project A and project B, with B named by its absolute path in the request. Illustrates `call-root-resolution.spec` clauses 1, 2, 5, 7, 9, 10, 11, 13, 15, 16 and Failure Behavior 4 and 7. Plugin users read it to see where each document lands; the skills' target-project rule depends on its examples.

## Actors

| Actor | Who they are | What they want |
|---|---|---|
| Developer | a user with the plugin installed globally, session open in `/w/a` | every artifact in the `.archcore/` of the project it describes |
| Agent | the coding agent in that session | one way to name the project of a call |

## Flows

### Developer

Anchors: @plugin/plugins/archcore/skills/plan/SKILL.md, @plugin/plugins/archcore/skills/_shared/gate-contract.md, @cli/internal/mcp/tools/create_document.go

1. Developer types `/archcore:plan` naming `/w/a` and `/w/b`; the skill records two target projects.
2. Developer reads the grounding; it lists accepted rules of A and of B under separate headings.
3. Developer answers the interview; the skill writes A's spec with no `project` and B's spec with `project=/w/b`.
4. Developer reads the report; each document path is shown beside the project it landed in.

Extensions:

- 1a. Developer names only a file of B in the request; the skill derives B from the nearest `.archcore/` above it.
- 3a. B holds no `.archcore/`; the skill reports it and asks before any `init_project`.

### Agent

Anchors: @cli/internal/mcp/tools/search_documents.go, @cli/cmd/hook_write_guard.go, @cli/cmd/hook_post_tool_use.go

1. Agent calls `search_documents(for_path="/w/b/src/api.go")`; the server answers from `/w/b/.archcore/` and returns `project: "/w/b"`.
2. Agent calls `create_document(project="/w/b", …)`; the file lands under `/w/b/.archcore/`.
3. Agent edits `/w/b/.archcore/x.spec.md` with the editor; the hook denies and names `project`.
4. Agent calls `add_relation` across `/w/a` and `/w/b`; the server refuses and states relations stay inside one project.
5. Agent calls `update_document(project="/w/b", …)`; the post-tool-use hook validates the file under `/w/b`.

Extensions:

- 2a. Agent passes `project="/w/b/.archcore"`; the server refuses because the value is not a project root.
- 2b. Agent passes `project="/w/c"` where no `.archcore/` exists; the server names `init_project` and `project`.

## Examples

Background: `/w/a` and `/w/b` each hold `.archcore/`; the plugin is installed at user scope; the session started in `/w/a`.

### Writing B's document from A's session

Illustrates: clauses 1, 9, 10, 11.
Given the session root is `/w/a`.
When Agent calls `create_document(project="/w/b", type="spec", filename="api")`.
Then the file exists at `/w/b/.archcore/api.spec.md`.
And the response carries `project: "/w/b"` and `path: ".archcore/api.spec.md"`.

### Reading B's rules by file path

Illustrates: clauses 5, 10.
Given `/w/b/.archcore/go-errors.rule.md` is accepted and names no path.
When Agent calls `search_documents(for_path="/w/b/src/api.go")`.
Then the general rows include `go-errors.rule.md` and the response carries `project: "/w/b"`.

### Editing B's document with the editor

Illustrates: clauses 13, 15.
Given the session root is `/w/a`.
When Agent writes `/w/b/.archcore/api.spec.md` with the `Write` tool.
Then the hook denies the write and the reason names `update_document` and `project`.

### A relation across two projects

Illustrates: Failure Behavior 7.
When Agent calls `add_relation(source="/w/a/.archcore/x.adr.md", target="/w/b/.archcore/api.spec.md")`.
Then the server returns an error stating that a relation stays inside one project.

### A call without project keeps today's behavior

Illustrates: clause 7.
When Agent calls `list_documents()` with no `project`.
Then the response lists `/w/a/.archcore/` only, as it did before this contract.

## Open Questions

None.

## Clarifications

- 2026-11-02 — A Codex `post-tool-use` payload carries `tool_input` with the call arguments: in the host probe run of 2026-10-30 on Codex 0.166.0, `post-tool-use` validated the project-b document named by `project` (@.archcore/plugin/host-probe-protocol.spec.md).
- 2026-11-02 — The grounding report keeps global sources per project: `skills/_shared/target-project.md`, section Grounding and report, runs grounding in each target project, and each project's grounding reads the global sources that project declares. This is read from the skill contract, not measured in a session [assumption].