---
title: "Write a Usage-Nudge Instruction File per Agent on init"
status: accepted
tags:
  - "integrations"
---

## Context

Hooks and MCP config give agents the *tools*, but not a reason to reach for them. The Archcore plugin (Claude Code / Cursor / Codex) supplies discoverability through skills, a default prompt, and hooks — but **CLI-only users get none of that**, and several supported hosts (Copilot, Gemini CLI, OpenCode, Cline, Roo) have no Archcore plugin at all, so the CLI is their only integration path.

This matters more under **Tool Search**: Claude Code (and others) now defer MCP tools — only tool *names* load at startup, schemas load on demand. So an always-on instruction nudge is the **discovery trigger**: without it, a CLI-only agent may never search for Archcore's tools on a relevant turn.

The `AGENTS.md` standard is read natively by most supported hosts, and prior art (Context7 ships a `.claude/rules/` file plus an `AGENTS.md` block) shows the pattern converts "type it every time" into "it just happens".

## Decision

`archcore` writes a short, always-on "use Archcore" hint into each detected host's instruction file, as **install-time host-awareness** owned by the **CLI** (the same category as the MCP-config writing it already does in `internal/agents`). This does not violate the "CLI stays host-agnostic at runtime" rule — there is no runtime host detection.

**Triggers:**
- `archcore init` offers it as an **opt-in** step (interactive confirm, default yes) after hooks + MCP install. Non-interactive `init` skips it with a hint, because the nudge lands in user-curated files.
- `archcore instructions install [--agent <id>]` and `archcore instructions remove [--agent <id>]` cover manual and automated runs.

**Per-host targets (AGENTS.md-first):**

| Agent | Instruction file | Write mode |
|-------|------------------|------------|
| Claude Code | `.claude/rules/archcore.md` | owned (whole file) |
| Gemini CLI | `GEMINI.md` | fenced upsert |
| Cursor, OpenCode, Codex CLI, Roo Code, Cline, GitHub Copilot | `AGENTS.md` | fenced upsert |

**Ownership and idempotency:**
- **Owned files** — archcore owns the whole file and overwrites freely. Claude Code does not read `AGENTS.md` (it reads `.claude/rules/*.md`, which auto-load at CLAUDE.md priority), so it gets a dedicated file.
- **Fenced upsert** — for shared user files, archcore only ever replaces the span between `<!-- archcore:start -->` and `<!-- archcore:end -->`; content outside the markers is never touched. A single shared marker pair (not per-host markers) lets the Archcore plugin target the same block later. Writing twice yields byte-identical output.
- **Dedup** — callers collapse the selected agents to unique instruction-file paths, so the six `AGENTS.md` agents trigger a single write.

**Content** is host-neutral, outcome-first, and references Archcore through its **MCP tools** (not plugin slash commands — CLI-only users have no plugin). It splits the cheap discovery search from the selective deep read, so invocation tracks *relevance* rather than volume: lean on the search, and skip only turns the repo would have no opinion on (syntax trivia, throwaway snippets, pure mechanics). The skip is keyed on the nature of the turn — a prior the agent can form up front — not on whether a rule exists, which the agent cannot know without the very lookup it is being told to skip. Wording uses "project context" rather than "memory" to stay aligned with [Archcore Product Positioning](../marketing/product-positioning.rule.md).

## Alternatives

- **Plugin owns the nudge.** Rejected: CLI-only and plugin-less hosts would get nothing — exactly the users this targets. The CLI owns the baseline block for all hosts; the plugin can later enrich or defer to the same marker.
- **Always write without a confirm** (as hooks/MCP do). Rejected for `init`: `AGENTS.md` / `GEMINI.md` are user-curated content files, not machine config, so prose is appended only with consent. The fenced upsert keeps it non-destructive and `instructions remove` makes it reversible.
- **Per-host native files everywhere** (`.cursor/rules/archcore.mdc` with `alwaysApply`, `.roo/rules/`, `.clinerules/`). Deferred to a later iteration: v1 relies on each host's native `AGENTS.md` support for single-exposure simplicity. `alwaysApply` is an opt-in upgrade for a hard always-on guarantee.
- **Unify Gemini onto `AGENTS.md`** by editing `contextFileName` in `.gemini/settings.json`. Rejected for v1: invasive settings edit; a plain `GEMINI.md` write is simpler.
- **Blunt "Do NOT use Archcore for…" prohibition keyed on rule-presence.** Rejected: gating on "general programming help with no project-specific rule attached" asks the agent to predict a fact about the doc store it cannot know without the very lookup it is told to skip. It silently suppressed invocation exactly when a doc *did* apply — which reads to users as a dead, useless integration — and the failure worsens as the store grows and more areas get covered. The relevance-and-cost framing in **Content** keeps a brake (skip trivial turns) without the unknowable precondition.

## Consequences

- CLI-only users (and plugin-less hosts) now discover and invoke Archcore without manual prompting — the gap this closes.
- Non-destructive (fenced upsert) and reversible (`instructions remove`).
- Cost: one extra opt-in prompt during interactive `init`; non-interactive `init` skips the nudge, so automation must call `archcore instructions install` explicitly.
- Maintenance: every agent must wire `InstructionsPath`, `WriteInstructions`, and `RemoveInstructions`. This is enforced by `TestAllAgents_RequiredFields`, so a new agent cannot silently omit a target.
- Helpers live in `internal/agents/instructions.go`; the command layer (`cmd/instructions.go`) handles dedup, display, and the `install`/`remove` subcommands; the `init` opt-in lives behind the `confirmInstructions` seam in `cmd/init.go`.
