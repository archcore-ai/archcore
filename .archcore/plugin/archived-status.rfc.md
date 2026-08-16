---
title: "Archived Status Value — Kernel Enum Extension for Discharged Documents"
status: draft
tags:
  - "architecture"
  - "plugin"
  - "vision"
---

## Summary

Add a fourth document status value, `archived`, to the Archcore kernel — the CLI's MCP tool schemas, sync, search ranking, and hooks — so the closeout discharge ceremony can transition a document whose unique information is absorbed elsewhere, instead of reporting candidates it cannot act on.

## Motivation

The delta-routing lifecycle defines discharge: a completed `plan` folds into `task-type` or `guide` capture plus git history, an `idea` completes once every implementing document is accepted, a spike `rnd` keeps only its Findings. Today the closeout track lists these candidates and stops — the kernel enums exactly `draft`/`accepted`/`rejected` in `create_document`, `update_document`, `list_documents`, and `search_documents`, so an `archived` write is refused at the tool boundary by every shipped CLI. `rejected` is semantically wrong for a document that succeeded and was absorbed. The corpus keeps growing with completed vision documents that grounding must read past on every invocation.

## Detailed Design

- The status enum becomes `draft` / `accepted` / `rejected` / `archived` in every MCP tool schema and in frontmatter validation — an additive change.
- Semantics: terminal state for a document whose unique information is absorbed elsewhere; the document stays in git and in the relation graph.
- Search and grounding: `search_documents` and `list_documents` exclude `archived` documents unless the caller filters for the status explicitly; session-start recaps and drift detection skip them.
- Transitions: `draft → archived` and `accepted → archived`, each under the per-document confirmation the closeout ceremony already requires; no autonomous transition.
- Rollout: the CLI ships the enum first; the plugin gates discharge transitions behind a `cli-gte`-style version probe, keeping the current report-only behavior on older CLIs.

## Drawbacks

- Every shipped CLI refuses the value, so mixed-version teams need the version probe; a new-CLI corpus read by an old CLI carries a status the old release does not recognize — old-release read behavior on an unknown status needs one verification pass before rollout.
- Search-exclusion semantics change what "all documents" means for existing tooling and dashboards.
- A fourth state adds one more branch to every status-reading feature (drift detection, closeout, global-source inheritance).

## Alternatives

- Reuse `rejected` — wrong meaning: discharged documents completed their purpose; drift tooling treats `rejected` as a red flag on active relations.
- A frontmatter flag (`archived: true`) without an enum change — avoids the tool-boundary break but splits status into two fields every reader must join.
- A physical move to an `archive/` subdirectory — breaks relation paths and git history locality; the manifest would need path rewrites.