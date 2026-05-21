---
title: "Cross-Project MCP Access from a Single Session"
status: draft
---

## Idea

Allow an agent running in one project to query another project's `.archcore/` through the same MCP session, instead of falling back to raw filesystem reads.

## Value

Cross-project work (comparing decisions, copying patterns, referencing a sibling repo) currently loses all MCP affordances — types, relations graph, search. Bridging this keeps the structured layer usable across project boundaries.

## Possible Implementation

Several directions, to be evaluated later:

- Multiple MCP servers in agent config, one per project (works today, static).
- Optional `project_root` parameter on existing tools (stateless, cleanest).
- Dynamic `attach_project` / `use_project` tools (stateful, more ergonomic but riskier).

## Risks

- Duplicated SessionStart context inflates tokens and confuses the model about which graph it's writing to.
- Relations graph is per-project — cross-project links don't fit the current model.
- Hidden state in stateful variants makes tool calls non-deterministic.
