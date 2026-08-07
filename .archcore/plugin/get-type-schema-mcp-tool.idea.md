---
title: "get_type_schema MCP Tool — CLI Templates as the Single Section Canon"
status: draft
tags:
  - "architecture"
  - "plugin"
  - "precision"
---

## Idea

Add a `get_type_schema` MCP tool that returns, per document type, the section skeleton, the precision and EARS rules that apply, and the layer-5 object contracts (mermaid blocks, structured inserts). Deferred from the CLI v2 boundary work — it did not block the palette swap.

## Value

Today the section canon is duplicated: the CLI generates templates, and the plugin's `skills/_shared/` contracts restate mandatory sections in prose. A queryable schema makes the CLI templates the single canon; plugin contracts shrink to policy (when to write, how to phrase) and stop drifting from the template shapes. Any host or agent — not only this plugin — can compose a conformant document.

## Possible Implementation

Expose the existing template definitions through one read-only MCP tool: `get_type_schema(type)` → `{ sections: [...], notation: { ears, bcp14 }, objects: [...] }`. Plugin contracts then cite the tool instead of restating section lists.

## Risks

A second machine-readable contract surface must not fork from the human-readable templates — generate both from one source or the drift this idea removes reappears one level down.