---
title: "A Rule for an Agent Sits Inside the 2,048-Unit Host Cap of the Tool It Governs"
status: accepted
tags:
  - "component:cli"
  - "mcp"
  - "performance"
  - "relations"
---

## Context

On 2026-09-21 one Claude Code session passed the first 2,048 UTF-16 units of the `create_document` description to the model and replaced the rest with `… [truncated]`. The cut fell inside the document-type catalog. The closing `Returns:` paragraph, which carries the `nearby_documents` guidance, starts at unit 3,364 of 3,696 (`@cli/internal/mcp/tools/create_document.go`), so it did not reach the model.

The same session cut the server instructions at unit 2,048 of 19,285. The `DOCUMENT RELATIONS` section starts at unit 2,092 (`@cli/internal/mcp/server.go`), so no relation guidance from the server instructions reached the model.

The `content` parameter description of `create_document` is longer than 2,048 units and arrived whole in that session. This is one observation.

The relation-authoring policy of the global research `relation-generation-quality.research` was first written into the server instructions (from unit 2,893) and into the closing paragraph of `create_document`. Both places are past the cap on that host. The `add_relation` description arrived whole.

No test bounded the size of a description. `@cli/internal/mcp/tools/response_budget_spec_test.go` bounds tool responses only. `read-tool-responses-survive-host-truncation.adr` records the same 2,048 preview for tool results and does not cover descriptions.

## Decision

A rule that changes agent behavior sits inside the first 2,048 UTF-16 units of the description of the tool that the rule governs, and `@cli/internal/mcp/tool_description_spec_test.go` holds every registered tool description to that cap or lists the tool in `overCapTools` with the reason.

The relation policy is the first rule placed this way. `TestAddRelationDescription_CarriesTheRelationPolicy` pins its phrases inside the host-visible part of the `add_relation` description.

## Alternatives Considered

1. Keep the policy in the server instructions only — rejected because the measured cut at unit 2,048 falls before the section that holds it.
2. Move `DOCUMENT RELATIONS` above the type catalog in the server instructions — deferred because it pushes other text out of the first 2,048 units, and no measurement ranks which text matters most there.
3. Put the rule in a parameter description — deferred because one observation supports it. [assumption] Other hosts pass parameter descriptions whole.
4. Shorten every description under the cap in this change — deferred. `create_document` exceeds the cap by 1,648 units because of the type catalog. `search_documents` exceeds it by 9 units, and `host-truncation-safe-read-tools.plan` already schedules a shorter text.

## Consequences

Positive:

- The `add_relation` description holds 2,001 units and carries the relation policy whole (measured 2026-09-21).
- An edit that pushes a description past the cap fails `TestToolDescriptions_FitTheHostCap`.
- An exemption that stops being true fails the same test, so `overCapTools` cannot keep a stale entry.

Negative:

- The `add_relation` description has 47 units left. Each addition to it needs a removal.
- The cap is measured on one host and one date. [assumption] Other hosts pass at least 2,048 units.
- The phrase test pins wording, so a rewording of the relation policy edits the test in the same change.
- Two tools stay over the cap. On that host the `nearby_documents` guidance of `create_document` does not reach the model; the policy reaches it through `add_relation`.

## Superseded when

- Claude Code changes the cap, or a supported host documents a smaller one.
- The type catalog leaves the `create_document` description, so `overCapTools` can drop the entry.
- A host delivers server instructions whole, so a per-tool placement is no longer the only dependable surface.
