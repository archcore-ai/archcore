---
title: "Research in Vision, One Research Path on plan, and Standalone Evidence Entry"
status: draft
tags:
  - "document-types"
  - "plugin"
  - "skills"
---

## Context

On 2026-09-07, the maintainer confirmed `research` in vision after the CLI implementation diverged from the accepted global vocabulary RFC. The CLI registry in @../cli/templates/templates.go assigns `research` to vision and `evidence` to knowledge. The existing research track in @plugins/archcore/skills/_shared/tracks/research.md produced only `rnd`, and its gather entry required a parent investigation.

The same day, the first runtime cut exposed three research entries on `/archcore:plan`: `research`, `rnd`, and `evidence`, plus a catch-all rule that any registry type name is an expert entry. The global RFC `concepts/research-and-evidence-types` at that time prescribed the `research` path resolving to the `research` type and `rnd` reached by its own name through that catch-all rule. The maintainer rejected that surface: `evidence` is a knowledge material and belongs to `/archcore:document`; `rnd` is an outcome of an investigation, not a path a user chooses; the catch-all rule leaves hidden entries the argument hint does not show.

## Decision

Use `research` in vision. Expose one research path on `/archcore:plan`, named `research`, that selects `research` or `rnd` by the closing test: a named pending decision or candidate set produces `rnd`; any other investigation produces `research`. Apply the same test to `document research`, where a report that ends in a recommendation is an `rnd`. Remove `rnd` and `evidence` as entries on `plan`, and remove the registry-type catch-all: the argument hint is the complete expert surface. File a standalone material only through `document evidence`, permit it without a consumer, and decline evidence writes on engines without the vocabulary.

The global RFC `concepts/research-and-evidence-types` was revised the same day to this surface; its "Command surface" section and this decision agree.

## Alternatives Considered

- **Research in knowledge** — rejected because the maintainer explicitly confirmed the revised CLI category; the global RFC's earlier category was stale for this release.
- **`research` path fixed to the `research` type, `rnd` by its own name (the global RFC's first surface)** — rejected because it makes the user choose the closing test before the investigation exists, and because the type-name catch-all exposes entries the argument hint hides.
- **Keep the type-name catch-all with an exception for `rnd`, `research`, `evidence`** — rejected because no documented invocation uses a type name on `plan`; removing the rule leaves no hidden surface.
- **`plan evidence` beside `document evidence`** — rejected because `evidence` is knowledge and `plan` writes vision; one entry per material avoids a duplicate path.
- **Require a consumer for every evidence** — rejected because an explicit material request already supplies the record to file and needs no parent investigation.
- **Store unsupported evidence as doc or rnd** — rejected because either substitution changes the explicitly requested document type.

## Consequences

### Enabled

- [expected] Plugin results and taxonomy match the CLI's 12 vision, 7 knowledge, and 2 experience types.
- [expected] The `plan` argument hint returns to `[sdd | sources | iso | research]`; the hint and the expert invocation map name the same surface.
- [expected] A user asks for an investigation once; the instrument records the type by what closes it.
- [expected] A standalone material produces one evidence draft with zero frame questions.

### Costs and limits

- [expected] A user who wants an `rnd` for a request with no named decision must phrase the decision or the candidates; there is no type override on `plan`.
- [expected] Standalone evidence can have no graph edge until a consumer is identified.
- [expected] An older engine records no document for an explicit evidence request.
- The engine gate is CLI 0.8.3, confirmed by the published [CLI v0.8.3](https://github.com/archcore-ai/cli/releases/tag/v0.8.3) and its native MCP probe on 2026-09-07.

## Superseded when

- The engine changes the category mapping of `research` in its published registry.
- A later global decision restores type-name entries on `plan` or adds a type override to the `research` path.
- A confirmed product requirement makes a consumer mandatory for an explicitly filed material.