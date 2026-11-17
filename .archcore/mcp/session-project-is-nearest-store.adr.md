---
title: "The Session Project Is the Nearest Store at or Above the Session Folder, and Only init_project Creates a Store"
status: draft
tags:
  - "cli"
  - "component:cli"
  - "hooks"
  - "integrations"
  - "mcp"
  - "multi-host"
---

## Context

`archcore mcp` served the folder it started in as the session root, whether or not that folder held `.archcore/` (@cli/internal/mcp/root_provider.go). A session started below a project root therefore worked on an empty project. The hooks already resolved an edited file by the nearest directory above it that holds `.archcore/` (@cli/cmd/hook_project_root.go), so the two surfaces disagreed about one session.

A matrix of 2026-11-16 on archcore CLI 0.15.5 — 21 session folders × 14 tool calls, each cell a fresh sandbox with a filesystem snapshot before and after — showed the cost. From a folder below a project root, `list_documents` and `search_documents` returned an empty success in 53 of 294 cells, with the same bytes as an empty store or a filter that matched nothing, and `create_document` created a second `.archcore/` without `settings.json` in 22 cells through an unconditional `MkdirAll` (@cli/internal/mcp/tools/create_document.go). That second store then became the nearest store for its subtree, so the pre-edit hook stopped listing the parent project's rules for files under it. Because the plugin and the CLI ship on separate schedules (`plugin-cli-compatibility.rule`), the fix had to live in the CLI and keep every tool name, argument, hook command line, and output envelope.

This decision reverses three earlier choices: the session-folder root kept by `explicit-target-project-per-call.adr` (its Decision and alternative 4), and the rejection of a walk up from the working directory in `session-following-project-root.adr` and `host-cwd-misrouting.adr`. Those rejections feared a wrong root that looks right. Two facts change that: from this decision on, only `init_project` and `install_host_config` create a store, and the ascent skips a store without `settings.json`, which is the trace of the implicit store an earlier CLI created.

## Decision

From the release after 0.15.5, `archcore mcp` and every `archcore hooks` leaf take as the session project the nearest directory at or above the session folder that holds `.archcore/`, computed by `projectroot.ForSession` on every call; a store above the session folder counts only when it holds `.archcore/settings.json`. Every tool except the four reads and `init_project` and `install_host_config` refuses a root that holds no `.archcore/`. The pre-tool-use write guard denies a direct file write that would create `.archcore/` inside an initialized project.

## Alternatives Considered

1. Keep the session folder as the session root — the choice of `explicit-target-project-per-call.adr` — rejected because of the measurements above: 53 silent empty reads and 22 created stores in 294 cells, and one rule for hooks with another for MCP.
2. Ascend only to the git repository boundary — rejected because the hooks already ascend without that bound, and a second rule would split the two surfaces again in a folder outside git and in a submodule. The `settings.json` condition already keeps an accidental store in a workspace parent or in `$HOME` from capturing the folders below it.
3. Refuse the reads too on a root without a store — rejected because the `init_project` description and the plugin skills up to 0.15.5 read an empty `list_documents` as "not initialized"; an error there changes a response that released plugins depend on.
4. Keep the implicit store of `create_document`, write `settings.json` with it, and announce it — rejected because with the ascent a store created by accident in a workspace parent becomes the session project of every folder below it that has no store of its own.
5. Deny every direct write into a `.archcore/` path that does not exist yet (@cli/cmd/hook_write_guard.go) — rejected because a store outside every project is the user's own folder; the guard denies only a new store inside an initialized project, which is the store that would capture a subtree.
6. Ascend into any store, with or without `settings.json` — rejected because every earlier CLI that ran `create_document` outside a project left such a store, and it would silently become the project of every folder below it.

## Consequences

Positive (matrix of 2026-11-17, the branch binary against a binary built from `dev` at 243b4334):

- 0 of 294 cells create `.archcore/` through a tool other than `init_project` and `install_host_config`; before, 22.
- A session below a project root reads and writes that project: 79 more cells answer from the expected project (105 → 184).
- In the 112 cells whose session folder holds the store, every tool response is byte-identical to the base binary; 84 of 84 hook outputs for those sessions and for sessions with no project above are byte-identical too.
- `init_project` from below a root returns the existing project with `"already_initialized": true` and creates nothing; 5 nested stores before, 0 after.
- A successful read on a root without a store ends with `"initialized": false`, 25 cells.

Negative:

- `create_document` on a folder with no initialized store at or above it returns an error where it returned a success. The plugin skills at 0.10.0 and at 0.15.5 call `init_project` first, and the error names it; the versions between were not read [assumption].
- A package of a monorepo that has no store of its own is served by the store above it until the user creates one with `init_project` and `project` set to the package folder. The `/archcore:init` skill up to 0.15.5 never passes `project`, so from such a folder it reports the project above as already initialized.
- A relative `for_path` is read against the session project, not the session folder; the recap line of a session below a root says so.
- A store without `settings.json` below an initialized project stays the nearest store for the edited files under it, as before this change; the session project of a folder below it is that folder, which holds no store.
- `archcore mcp` started in a folder below a project validates that project's `settings.json` and global sources at startup, so an invalid one stops the server there as it does at the root; before, such a folder started without a store.
- A session that moves to another folder on a host that does not answer `roots/list` keeps the project it started in; the `project` field of each response is the only sign. Claude Code 2.1.322 (MCP protocol `2026-07-28`) did not answer on 2026-11-16: the reply to a tool call arrived 0.502 s after the query, the query timeout.
- `update_document`, `remove_document`, `add_relation`, and `remove_relation` on a root without a store return the new refusal in place of "document not found" or an empty success.
- A fixture store inside this repository (for example under @cli/examples/) can no longer be created by a direct file write; `init_project` with `project` set to the fixture folder creates it.
- Plugins up to 0.15.5 keep their launcher and init skill; the launcher fix and the init skill's use of the `project` field reach a user only with a plugin update.

## Superseded when

- MCP carries the working directory of a call in the request, which would replace the stored session folder.
- A store without `settings.json` becomes a supported project shape, which reopens alternative 6.
- A host with no skill that calls `init_project` first becomes a supported target, which reopens the refusal of `create_document`.
