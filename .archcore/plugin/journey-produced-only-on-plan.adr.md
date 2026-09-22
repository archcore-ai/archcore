---
title: "Journey Produced Only on plan — No document Entry for an Intent Type"
status: draft
tags:
  - "commands"
  - "component:plugin"
  - "document-types"
  - "plugin"
  - "skills"
---

## Context

The command entry grammar of 2026-09-16 gives `/archcore:document` three modes, `decision`, `code`, and `research`, and moves type selection into gates (@.archcore/plugin/command-entry-grammar.adr.md). A `journey` records the intended path of one user type before a `spec` exists, so it fits none of the three: `code` reads source files as evidence, and `decision` records a settled technical choice. The working tree on branch `dev` routes `document journey` to `sdd.require` in callable mode (@plugin/plugins/archcore/skills/document/SKILL.md, Step 2), an entry that the grammar retires.

## Decision

A `journey` is produced only by the intent instrument at `sdd.require` on `/archcore:plan`, beside the `prd` under the illustrate condition; `/archcore:document` has no path that creates a `journey`, and `describe.draft` does not select it.

## Alternatives Considered

1. **Keep a callable `sdd.require` entry behind the `code` mode** — rejected because `code` would then create an intent document with no source evidence, and the mode name would stop describing its product.
2. **A fourth `document` mode `journey` or `intent`** — rejected because a mode that produces one type is the type-name entry the grammar removes, and `intent` is a second name for the `plan` intent instrument.
3. **Keep `document journey` as a named entry** — rejected because the hint would list one type name beside three modes.

## Consequences

### Enabled

- `document` writes records of the present state only: decisions, existing behavior, and supplied materials.
- `sdd.require` keeps one production path for `journey`, the one the illustrate instrument links a `scenario` to with `implements`.

### Costs and limits

- A user who holds a ready journey files it through `/archcore:plan`, which also produces the `prd` beside it; a `journey` with no `prd` has no path.
- The draft decision `document-journey-callable-intent-entry` loses its subject; its status moves to `rejected` only on the user's confirmation.
- The global draft RFC `concepts/scenario-and-journey-types` in the `archcore` global source names `document journey` as an entry; the plugin departs from that clause.

## Superseded when

- Two or more recorded requests on the routing bench file a ready journey without a product intent to record in a `prd`.
- The engine moves `journey` out of vision.
