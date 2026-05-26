---
title: "Architect Voice as Default Documentation Style"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "precision"
---

## Context

Research across all plugin prompts (`precision-rules.md`, `agents/archcore-assistant.md`, 7 SKILL.md files) found no content style constraint: agents composing `.archcore/` documents defaulted to developer-style output — implementation code, function signatures, and step-by-step code walkthroughs in document types where architectural rationale is the primary signal. Documents serving as AI agent context deliver higher signal-to-noise when they capture *why* decisions were made and *what they cost*, not *how* implementations work — the code is already readable from `@path/to/file` references in source.

## Decision

Adopted **architect voice** as the default content standard: expert, concise, precise, and argued. A document should let a senior engineer understand *why*, *what*, and *what it costs* in 30 seconds. `@path/to/file` references, identifiers, measurements, and version strings are used freely — they are the architect's vocabulary. Pasting code bodies, implementation walkthroughs, and AI-padded filler are defects. The constraint is codified in `skills/_shared/precision-rules.md` Rule 6 and `agents/archcore-assistant.md` Quality Standards. Code blocks remain appropriate for types where the exact textual format is the artifact: `rule` (Good/Bad), `guide` (steps), `cpat` (Before/After), and on explicit user request.

## Alternatives Considered

1. **Add voice rule to MCP server instructions (`internal/mcp/server.go`)** — rejected because MCP instructions are host-agnostic and serve Cursor, Copilot, Codex CLI, and Claude Code equally; Claude Code-specific narrative preferences do not belong in a shared layer.
2. **Modify CLI templates (`templates/templates.go`) to remove code placeholders** — ruled out because templates define document *structure*, not voice; removing code blocks from `spec` or `doc` templates would break valid use cases where code is genuinely normative (wire formats, protocol contracts).
3. **No change, rely on per-user prompting** — deferred; creates inconsistent defaults where each new user gets developer-style output until they discover an override.

## Consequences

- [expected] ADR, RFC, PRD, plan, and spec documents produced without explicit user override will contain argued rationale rather than implementation detail — shorter, faster for AI agents to process.
- Tradeoff: users who need inline code examples in non-code-native types must explicitly request them; the escape hatch is always available but not the default.
- [expected] Per-composition token overhead from Rule 6 and agent definition additions: ~190 tokens (~6–8% on a typical `create_document` workflow).
- Rule 6 is a behavioral default, not a structural gate — `bin/check-precision` does not enforce it.

## Superseded when

- User research shows ≥ 30% of first-time users explicitly request inline code in ADRs or PRDs before being prompted — indicating the default misaligns with actual usage patterns.
- A content-profile system is introduced that allows per-type voice configuration in `.archcore/settings.json`.
