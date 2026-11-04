---
title: "OpenClaw Skill — the archcore ClawHub Skill Package and Its Publish Contract"
status: draft
tags:
  - "component:plugin"
  - "integrations"
  - "multi-host"
  - "openclaw"
---

## Purpose & Scope

This spec is normative for the `archcore` skill published on ClawHub: its source directory, frontmatter, body obligations, and the CI publish job. Depended on by OpenClaw 2026.10.10 and nanobot agents that install the skill, and by the structure test that validates it. Out of scope: the MCP server itself, ACP coding sessions (covered by the OpenClaw integration guide), and an OpenClaw code plugin (`clawhub-distribution.adr`).

Host facts come from `openclaw/openclaw` commit `37d5c24f` (`docs/tools/skills.md`, `docs/tools/mcp.md`) and `openclaw/clawhub` commit `ea85448f` (`docs/skill-format.md`, `docs/publishing.md`, `convex/lib/skillSlugValidator.ts`, `convex/lib/securityPrompt.ts`).

## Surface

- Source: `plugin/plugins/openclaw/archcore/SKILL.md`; no other file in the skill folder.
- Frontmatter: `name: archcore`, `description`, `version`, `metadata.openclaw.requires.bins: ["archcore"]`, `metadata.openclaw.homepage`, `metadata.openclaw.emoji`.
- Slug `archcore` (pattern `^[a-z0-9](?:(?!--)[a-z0-9-])*[a-z0-9]$`, 3–96 characters); categories `development`, `knowledge`.
- MCP registration command the body gives: `openclaw mcp add archcore --command archcore --arg mcp --arg --project --arg <absolute project root>`.
- Archcore MCP tools the body names: `search_documents`, `get_document`, `list_documents`, `create_document`, `update_document`, `add_relation`.
- Publish: `clawhub skill publish plugin/plugins/openclaw/archcore --slug archcore --name Archcore`, authenticated by a `CLAWHUB_TOKEN` repository secret, in a job of `@.github/workflows/release.yml`.

## Normative Behavior

1. The `SKILL.md` frontmatter MUST carry the keys of Surface.
2. The `SKILL.md` MUST NOT carry a `metadata.clawdbot` block.
3. The `SKILL.md` MUST stay at or below 6,000 characters.
4. The skill body MUST instruct the agent to call `search_documents` with `for_path` before it changes a file.
5. The skill body MUST instruct the agent to write `.archcore/` documents only through the Archcore MCP tools.
6. The skill body MUST give the MCP registration command of Surface for an agent without Archcore tools.
7. The skill body MUST NOT name an environment variable, because the skill declares none.
8. WHEN a release tag is pushed, the release workflow MUST publish the skill with the version of that tag.
9. IF ClawHub already holds the tag version, THEN the release workflow MUST skip publishing without failing.

## Constraints & Invariants

- Constraint: ClawHub's security check compares declared requirements with skill behavior; behaviors 1 and 7 keep them equal.
- Constraint: the security prompt reads only the first 6,000 characters of `SKILL.md` (`securityPrompt.ts`).
- Constraint: OpenClaw stopped reading `metadata.clawdbot` in July 2026.
- Constraint: ClawHub publishes every skill under `MIT-0`.
- Constraint: OpenClaw's native MCP client stores the entry under `mcp.servers`; `mcporter` configs are not read by it.
- Invariant: the skill holds no executable file and no guard logic.

## Failure Behavior

1. IF `archcore` is not on `PATH`, THEN OpenClaw marks the skill unavailable through `requires.bins`; the skill body MUST give the install URL `https://docs.archcore.ai/cli/install/`.
2. IF `CLAWHUB_TOKEN` is absent, THEN the publish job MUST fail with a message naming the secret.

## Conformance

An implementation conforms when it satisfies behaviors 1–9 and passes: a structure test under `@plugin/test/structure/` for behaviors 1–3 and 7 and for the slug pattern; `clawhub skill publish --dry-run` in CI; and a recorded check that `openclaw skills install @<owner>/archcore` lists the skill and that an agent calls `search_documents` through the registered server.

Given `archcore` on `PATH`, when OpenClaw loads the installed skill, then the skill is eligible and its body names `search_documents`.
