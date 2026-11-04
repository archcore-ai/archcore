---
title: "OpenClaw: archcore Skill on ClawHub, Native MCP Through openclaw mcp add, and a Verified ACP Recipe With a Project-Scoped Inner Plugin"
status: draft
tags:
  - "component:plugin"
  - "integrations"
  - "multi-host"
  - "openclaw"
  - "plugin"
---

## Goal

Reach OpenClaw users through two routes. First, the `archcore` skill on ClawHub and the `archcore` MCP server registered with `openclaw mcp add` give OpenClaw's own agent the document tools; nanobot reads the same skill. Second, a verified guide shows that `/acp spawn claude --cwd <repo>` carries the Archcore Claude Code plugin when that plugin is installed at project scope. OpenClaw's own agent gets no write guard. The contract is `openclaw-skill.spec`; the decision is `clawhub-distribution.adr` (accepted 2026-11-04).

Host facts: `openclaw/openclaw` 2026.10.10, `openclaw/clawhub` CLI 0.25.1, acpx 0.21.0 pinned by OpenClaw, `@agentclientprotocol/claude-agent-acp` 0.95.0.

## Tasks

### Phase 1 — Skill

1. Write `@plugin/plugins/openclaw/archcore/SKILL.md` per `openclaw-skill.spec` behaviors 1–7.
2. Add a structure test under `@plugin/test/structure/` for frontmatter keys, size, slug, and no `metadata.clawdbot`.
3. Exclude `plugins/openclaw/` from `@scripts/export-plugin.sh`, because ClawHub is its channel.
4. Pin that exclusion in `@plugin/test/structure/release-blocklist.bats`.
5. Add a `clawhub skill publish --dry-run` step to `@.github/workflows/test.yml`.
6. Add a `publish-clawhub` job to `@.github/workflows/release.yml` per spec behaviors 8–9.
7. Create the ClawHub publisher account and the `CLAWHUB_TOKEN` repository secret.

### Phase 2 — ACP path

8. Install OpenClaw 2026.10.10 and `@openclaw/acpx` on the probe machine.
9. Run `archcore plugin install --agent claude-code --scope project` in a repository with `.archcore/`.
10. Run `/acp spawn claude --cwd <repo>` and record whether the recap arrives.
11. Ask the session to write `.archcore/x.rule.md` and record whether the write is refused.
12. Repeat tasks 10–11 with the plugin at user scope and `ACPX_CLAUDE_INCLUDE_USER_SETTINGS=1`.
13. Run `/acp spawn codex --cwd <repo>` and record which Archcore surfaces load.
14. Write the guide `openclaw-acp-coding-sessions` with the commands that passed.
15. Record the ACP results in `wrapper-hosts-defer-to-inner-agent-adapter.adr`.

### Phase 3 — Documentation and listing

16. Publish the skill and record the ClawHub URL.
17. Check that `openclaw skills install @<owner>/archcore` lists the skill as eligible.
18. Check that an agent calls `search_documents` through the server from `openclaw mcp add`.
19. Add OpenClaw to `supported-ai-agents.doc` as a skills-and-MCP host with the ACP note.
20. Add the OpenClaw, claw-family, and ACP findings to `agent-host-landscape.rnd`.
21. Report the roster change to the `archcore` global `supported-ai-hosts.doc` through its project.

## Acceptance Criteria

- The structure test and the publish dry run pass in CI.
- The installed skill is eligible with `archcore` on `PATH`, and an OpenClaw agent calls `search_documents`.
- An OpenClaw ACP session on Claude Code, with the project-scope plugin, shows the recap and refuses a direct `.archcore/` write.
- The guide names the OpenClaw, acpx, and adapter versions and the probe date.

## Dependencies

- OpenClaw 2026.10.10 or newer, `@openclaw/acpx`, and Node on the probe machine.
- A ClawHub account and the `CLAWHUB_TOKEN` secret.
- The Archcore Claude Code and Codex plugins for Phase 2.

## Clarifications

- 2026-11-04: the maintainer chose a ClawHub skill, MCP, and the ACP recipe, without an OpenClaw code plugin.
- 2026-11-04: MCP moved from `mcporter` to `openclaw mcp add`, because OpenClaw has a native client and `mcporter`'s stdio `cwd` defaults to the config directory.
- 2026-11-04: acpx loads Claude Code with `settingSources: ["project", "local"]`; the recipe relies on a project-scope install.

## Declared Delta

- create spec `openclaw-skill` (draft, 2026-11-04).
- create adr `clawhub-distribution` (accepted, 2026-11-04).
- create guide `openclaw-acp-coding-sessions` (Phase 2, task 14).
- amend adr `wrapper-hosts-defer-to-inner-agent-adapter` — ACP results.
- amend doc `supported-ai-agents` — OpenClaw row.
- amend doc `agent-host-landscape` (rnd) — OpenClaw and ACP findings.
