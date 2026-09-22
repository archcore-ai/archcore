---
title: "Stored Research and Evidence: Release Outcomes"
status: draft
tags:
  - "document-types"
  - "mcp"
  - "relations"
---

## Vision

The release candidate lets Archcore users retain investigations, reusable materials, and their relationships as typed project context. The accepted vocabulary comes from `concepts/research-and-evidence-types` [global · archcore · read-only].

## Problem Statement

Before this change, researchers lacked dedicated records for an investigation closed by coverage and for an external material reused by several documents. The baseline at commit `e33df77` had 19 types and four structural relations — @templates/templates.go, @internal/sync/manifest.go. MCP edits also lost custom frontmatter because reconstruction wrote only three fields — @internal/mcp/tools/common.go.

## Goals and Success Metrics

The release targets the following measurable outcomes; these are planned delivery targets, not measured adoption results.

| Goal | Target | Verification |
|---|---|---|
| Represent investigations and materials | 2 additional document types | Template and MCP integration suites |
| Express evidence and replacement | 3 additional relation types | Manifest and MCP integration suites |
| Retain user metadata | 0 lost custom values in the round-trip fixture matrix | Update and parser regression suites |
| Keep the engine within storage scope | 0 research execution tools | MCP registration review |

## Requirements

1. Researchers can distinguish territory mapping from decision-bound investigation in stored context [expected].
2. Readers can trace reused materials and disputed findings through explicit document relationships [expected].
3. Authors can edit document content without losing metadata the Archcore MCP server does not own [expected].
4. Existing repositories retain their current document workflows through the vocabulary expansion [expected].

## Out of Scope

Research methods, fetching sources, checking authenticity, plugin command routing, and shared-context migration belong outside this CLI change. The boundary follows `architecture/store-not-method-engine` [global · archcore · read-only].
