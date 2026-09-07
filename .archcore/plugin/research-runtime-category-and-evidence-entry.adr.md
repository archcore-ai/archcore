---
title: "Research in Vision and Standalone Evidence Entry"
status: draft
tags:
  - "document-types"
  - "plugin"
  - "skills"
---

## Context

On 2026-09-07, the maintainer confirmed `research` in vision after the CLI implementation diverged from the accepted global vocabulary RFC. The CLI registry in @../cli/templates/templates.go assigns `research` to vision and `evidence` to knowledge. The existing research track in @plugins/archcore/skills/_shared/tracks/research.md produced only `rnd`, and its gather entry required a parent investigation.

## Decision

Use `research` in vision, permit explicit standalone `evidence` drafts without a consumer, and decline evidence writes on engines without the vocabulary.

## Alternatives Considered

- **Research in knowledge** — rejected because the maintainer explicitly confirmed the revised CLI category; the global RFC's category is stale for this release.
- **Require a consumer for every evidence** — rejected because an explicit material request already supplies the record to file and needs no parent investigation.
- **Store unsupported evidence as doc or rnd** — rejected because either substitution changes the explicitly requested document type.

## Consequences

### Enabled

- [expected] Plugin results and taxonomy match the CLI's 12 vision, 7 knowledge, and 2 experience types.
- [expected] A standalone material produces one evidence draft with zero frame questions.

### Costs and limits

- [expected] The global vocabulary RFC needs upstream reconciliation; this repository does not modify mounted globals.
- [expected] Standalone evidence can have no graph edge until a consumer is identified.
- [expected] An older engine records no document for an explicit evidence request.
- The engine gate is CLI 0.8.3, confirmed by the published [CLI v0.8.3](https://github.com/archcore-ai/cli/releases/tag/v0.8.3) and its native MCP probe on 2026-09-07.

## Superseded when

- The engine changes the category mapping of `research` in its published registry.
- A confirmed product requirement makes a consumer mandatory for an explicitly filed material.
