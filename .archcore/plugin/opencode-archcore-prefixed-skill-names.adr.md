---
title: "On OpenCode, the Bundled Skills Carry archcore-<name> Names, So the Commands Are /archcore-init, /archcore-plan, /archcore-document, /archcore-review"
status: accepted
tags:
  - "component:plugin"
  - "multi-host"
  - "opencode"
  - "plugin"
  - "skills"
---

## Context

OpenCode turns every discovered skill into a slash command named by the skill's frontmatter `name`, and it skips a skill whose name a command already holds; `init` and `review` are OpenCode built-in commands (`packages/opencode/src/command/index.ts`, v1.18.34), so bundled skills named `init` and `review` get no command at all. Skills from `skills.paths` are scanned last, and a duplicate name replaces the earlier skill with a warning (`packages/opencode/src/skill/index.ts`), so a bundled skill named `plan` replaces a user's own `plan` skill. OpenCode has no plugin namespace comparable to the `archcore:` prefix Claude Code puts on plugin skills. The maintainer chose the `/archcore-*` command form for OpenCode on 2026-10-18.

## Decision

The pack step of `@archcore/opencode-plugin` rewrites only the frontmatter `name:` line of each bundled `skills/<name>/SKILL.md` from `<name>` to `archcore-<name>`, so OpenCode lists the skills `archcore-init`, `archcore-plan`, `archcore-document`, and `archcore-review` and derives the commands `/archcore-init`, `/archcore-plan`, `/archcore-document`, and `/archcore-review` from them.

## Alternatives Considered

1. **Register `cfg.command["archcore-<name>"]` from the `config` hook and keep the skill names** — rejected because the skill-derived `/plan` and `/document` commands stay in the palette beside the new ones, and the generic skill names still replace a user's skills of the same name.
2. **Keep the skill names unchanged** — rejected because `/init` and `/review` then run the OpenCode built-ins, so two of the four Archcore commands are unreachable.
3. **Rename the source skills on every host to `archcore-<name>`** — rejected because Claude Code already prefixes plugin skills with `archcore:`, so the palette there would read `/archcore:archcore-plan`.

## Consequences

- All four Archcore commands are reachable on OpenCode, and no bundled skill name collides with a built-in command or with a user skill whose name lacks the `archcore-` prefix.
- An OpenCode command passes its arguments by appending them to the skill content, because no `SKILL.md` contains `$ARGUMENTS` (`packages/opencode/src/session/prompt.ts`); `/archcore-plan <topic>` therefore needs no wrapper file.
- Tradeoff: each bundled `SKILL.md` differs from its source in one line. The invariant "skill content is byte-identical across hosts" of `command-surface-v2.spec` holds for every other byte, and the byte-parity test of `opencode-adapter.spec` compares the files with that one line masked.
- Tradeoff: skill bodies cite commands in the colon form, for example `/archcore:plan`, while the OpenCode user types `/archcore-plan`. [expected] The model maps the colon form to the installed command; the live probe of `opencode-adapter.plan` checks one cross-reference.

## Superseded when

- OpenCode adds a plugin-scoped namespace for skills or commands, which would let the bundled skills keep their source names.
- A second host without a plugin namespace ships an Archcore adapter, which would argue for one shared rename rule in the pack tooling instead of an OpenCode-only step.
