---
title: "The OpenCode Bridge Registers mcp.archcore Only When the Entry Is Absent; archcore init Keeps Writing It"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "integrations"
  - "mcp"
  - "opencode"
---

## Context

Two writers can register the archcore MCP server for an OpenCode project: `archcore init --agent opencode` writes `mcp.archcore` into `opencode.json` through `writeOpenCodeMCPConfig` in `@cli/internal/agents/opencode.go` in keep-existing mode, and the `config` hook of `@archcore/opencode-plugin` mutates the live `cfg.mcp` at startup, as `opencode-adapter-packaging.adr` decided. Both emit `{"type":"local","command":["archcore","mcp"]}`, but a user may extend the file entry with `environment` or a timeout, and `mcp-config-converge-ownership.adr` already fixes that `command`, `args`, and `type` are the only archcore-owned fields. Issue archcore-ai/archcore#25 left the precedence between the two writers open; the maintainer settled it on 2026-10-09.

## Decision

The bridge's `config` hook sets `cfg.mcp.archcore` to `{"type":"local","command":["archcore","mcp"]}` only when that key is absent from the merged config, while `archcore init` keeps writing the same entry into `opencode.json` in keep-existing mode, so the first writer wins and a file entry with user fields is never shadowed.

## Alternatives Considered

1. **The bridge overwrites `type` and `command` on every start** — rejected because a config-hook write leaves no trace in the file: a user debugging a changed entry finds `opencode.json` untouched, and converge semantics already belong to `archcore doctor --fix`, which shows what it changed.
2. **`archcore init` is the only writer and the bridge never registers MCP** — rejected because a plugin-only install (the `plugin` array edited by hand, no `archcore init`) would load skills and hooks but no document tools, failing the "MCP tools work" acceptance criterion of issue #25 and item 13 of `host-adapter-contract.spec`.
3. **The bridge registers under a second key such as `archcore-plugin`** — ruled out because two entries start two `archcore mcp` processes per session and list every document tool twice in the model's tool set.

## Consequences

- [expected] A project wired by `archcore init` and a project carrying only the plugin entry reach the same document tool set; neither path depends on the other.
- User fields on the file entry (`environment`, a timeout) survive every OpenCode start, the same guarantee `doctor --fix` gives through `mergeEntryFields` in `@cli/internal/agents/mcp_helpers.go`.
- Tradeoff: a stale file entry, for example a wrong `command` after a binary relocation, is not self-healed by the bridge; `archcore doctor --fix` stays the repair path.
- Tradeoff: the bridge's registration depends on an ordering OpenCode does not document. In v1.18.34 `packages/opencode/src/project/bootstrap.ts` initializes plugins, and so runs `config` hooks, before MCP state is built; probe C of `host-probe-protocol.spec` re-confirms it per tested release.

## Superseded when

- OpenCode documents that plugin `config` mutations run after MCP initialization, which would make the hook path inert and leave `archcore init` as the only effective writer.
- `archcore init` gains converge semantics on install (today only `doctor --fix` converges), which would reopen whether the bridge should also rewrite the owned fields.
