---
title: "Architect Voice as Default Documentation Style"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "precision"
---

## Context

A review of every plugin prompt — `precision-rules.md`, `agents/archcore-assistant.md`, and the SKILL.md files — found no content style constraint anywhere. Agents composing `.archcore/` documents therefore defaulted to developer-style output: implementation code, function signatures, and step-by-step code walkthroughs in document types where architectural rationale is the primary signal. A document serving as AI agent context carries a higher signal-to-noise ratio when it captures why a decision was made and what it costs, rather than how an implementation works, because the code itself is already reachable from an `@path/to/file` reference.

## Decision

Adopted **architect voice** as the default content standard — expert, concise, precise, and argued, such that a senior engineer can extract why, what, and what it costs in 30 seconds — codified as Rule 6 of `@plugins/archcore/skills/_shared/precision-rules.md` and in the Quality Standards of `@plugins/archcore/agents/archcore-assistant.md`.

Under that standard, `@path/to/file` references, identifiers, measurements, and version strings are used freely, because they are the architect's vocabulary; pasted code bodies, implementation walkthroughs, and padded filler are defects. Code blocks belong where the exact textual format is the artifact — `rule` for Good and Bad examples, `guide` for terminal steps, `cpat` for Before and After — and wherever the user asks for them.

## Alternatives Considered

1. **Add the voice rule to the MCP server instructions in `internal/mcp/server.go`** — rejected because MCP instructions are host-agnostic and serve Cursor, Copilot, Codex CLI, and Claude Code equally, so a Claude Code-specific narrative preference does not belong in a shared layer.
2. **Modify the CLI templates in `templates/templates.go` to remove code placeholders** — ruled out because templates define document structure rather than voice, and removing code blocks from the `spec` or `doc` templates would break valid uses where code is genuinely normative, such as a wire format or a protocol contract.
3. **Make no change and rely on per-user prompting** — deferred because it creates inconsistent defaults, where each new user receives developer-style output until they discover an override.

## Consequences

- [expected] An `adr`, `rfc`, `prd`, `plan`, or `spec` produced without an explicit user override contains argued rationale rather than implementation detail, making it shorter and faster for an AI agent to process.
- [expected] Tradeoff: per-composition token overhead from the Rule 6 and agent-definition additions is about 190 tokens, roughly 6–8% on a typical `create_document` workflow.
- Tradeoff: a user who needs inline code in a non-code-native type must request it explicitly. The escape hatch is always available but is not the default.
- Rule 6 is a behavioral default rather than a structural gate, and nothing enforces it mechanically. The precision check moved into the CLI at v0.7.0 (`cli-owns-layers-4-5.adr`) and `ca6dfb4` deleted the `bin/check-precision` script that predated it; the successor reports the forbidden lexicon, the mandatory sections, and the Rule 7 line form, and the Enforcement section of `precision-rules.md` states that Rules 3, 4, and 6 carry no automated check.

## Superseded when

- User research shows that 30% or more of first-time users explicitly request inline code in an ADR or PRD before being prompted, which would indicate the default misaligns with actual usage.
- A content-profile system is introduced that allows per-type voice configuration in `.archcore/settings.json`.
