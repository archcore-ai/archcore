---
title: "Write a Usage-Nudge Instruction File per Agent on init"
status: accepted
tags:
  - "integrations"
---

## Context

Hooks and MCP config give agents the *tools*, but not a reason to reach for them. The Archcore plugin (Claude Code / Cursor / Codex) supplies discoverability through skills, a default prompt, and hooks — but **CLI-only users get none of that**, and several supported hosts (Copilot, Gemini CLI, OpenCode, Cline, Roo) have no Archcore plugin at all, so the CLI is their only integration path.

This matters more under **Tool Search**: Claude Code (and others) now defer MCP tools — only tool *names* load at startup, schemas load on demand. So an always-on instruction nudge is the **discovery trigger**: without it, a CLI-only agent may never search for Archcore's tools on a relevant turn.

Per Anthropic's official docs (code.claude.com/docs/en/memory), **Claude Code reads `CLAUDE.md` natively but does NOT auto-read `AGENTS.md`** — AGENTS.md reaches Claude Code only via an `@import` inside CLAUDE.md or a symlink. The `AGENTS.md` standard is read natively by the other supported hosts. Prior art (Context7) ships an `AGENTS.md` block plus a Claude-specific file, showing the pattern converts "type it every time" into "it just happens".

## Decision

`archcore` writes a short, always-on "use Archcore" hint into each detected host's instruction file, as **install-time host-awareness** owned by the **CLI** (the same category as the MCP-config writing it already does in `internal/agents`). This does not violate the "CLI stays host-agnostic at runtime" rule — there is no runtime host detection.

**Triggers:**
- `archcore init` offers it as an **opt-in** step (interactive confirm, default yes) after hooks + MCP install. Non-interactive `init` skips it with a hint, because the nudge lands in user-curated files.
- `archcore instructions install [--agent <id>]` and `archcore instructions remove [--agent <id>]` cover manual and automated runs.

**Per-host targets:**

| Agent | Instruction file(s) | Write mode |
|-------|---------------------|------------|
| Claude Code | `CLAUDE.md` **and** `AGENTS.md` | fenced upsert (both) |
| Gemini CLI | `GEMINI.md` | fenced upsert |
| Cursor, OpenCode, Codex CLI, Roo Code, Cline, GitHub Copilot | `AGENTS.md` | fenced upsert |

**Claude Code gets `CLAUDE.md` and `AGENTS.md`.** `CLAUDE.md` is the file Claude Code reads natively, so it is what actually delivers the nudge. `AGENTS.md` is also written so the repo carries the standard block the plugin and the six other hosts converge on — but note Claude Code does **not** read that AGENTS.md block itself (no `@import` is added; `CLAUDE.md` carries the full nudge body directly, so there is no dependency between the two files). The nudge body is identical in both; because Claude Code loads only `CLAUDE.md`, there is **no duplicated load** in its context. We deliberately did NOT add a `CLAUDE.md → @import AGENTS.md` indirection: a self-contained `CLAUDE.md` block is simpler and does not couple the two files.

*History:* earlier CLI versions instead wrote an owned `.claude/rules/archcore.md` file (on the premise that "Claude Code does not read AGENTS.md, but auto-loads `.claude/rules/*.md`"). That premise about `.claude/rules` is fine, but writing to `CLAUDE.md` is the more direct, canonical channel and keeps Claude's nudge in the file users actually curate. The legacy `.claude/rules/archcore.md` is now **migrated away** (removed) whenever Claude Code is (re)wired, so the nudge is never loaded from two places.

**Ownership and idempotency:**
- **Fenced upsert** — for every target (`CLAUDE.md`, `AGENTS.md`, `GEMINI.md`), archcore only ever replaces the span between `<!-- archcore:start -->` and `<!-- archcore:end -->`; content outside the markers is never touched. A single shared marker pair (not per-host markers) lets the Archcore plugin target the same block later. Writing twice yields byte-identical output, so Claude Code and a co-installed `AGENTS.md` agent both writing the AGENTS.md block collapse to one.
- **Dedup** — callers collapse the selected agents to unique instruction-file paths, so the six `AGENTS.md`-only agents trigger a single write. Claude Code's dedupe key is its own `CLAUDE.md` path (distinct from `AGENTS.md`), which guarantees Claude Code always runs its own write — refreshing both files — regardless of the agent-list order.
- **Remove** — removing Claude Code alone strips only its `CLAUDE.md` block (and deletes the legacy `.claude/rules/archcore.md`), leaving the shared `AGENTS.md` block to the `AGENTS.md` agents' own remove; the "remove all" path runs both, so nothing is orphaned there.

**Content** is host-neutral, outcome-first, and references Archcore through its **MCP tools** (not plugin slash commands — CLI-only users have no plugin). It splits the cheap discovery search from the selective deep read, so invocation tracks *relevance* rather than volume: lean on the search, and skip only turns the repo would have no opinion on (syntax trivia, throwaway snippets, pure mechanics). The skip is keyed on the nature of the turn — a prior the agent can form up front — not on whether a rule exists, which the agent cannot know without the very lookup it is being told to skip. Wording uses "project context" rather than "memory" to stay aligned with the `archcore` global source `product/messaging-and-voice`.

## Alternatives

- **`CLAUDE.md → @import AGENTS.md`** (Anthropic's documented pattern — a single nudge source in AGENTS.md, imported by CLAUDE.md). Rejected for now: it couples the two files and relies on the import resolving; a self-contained CLAUDE.md block is simpler and the ~20-line body is cheap to duplicate. Kept open as a future simplification if we want one canonical source.
- **Claude Code on `.claude/rules/archcore.md`** (owned whole file, auto-loaded at CLAUDE.md priority on every version). This was the previous implementation. Superseded: `CLAUDE.md` is the file users actually curate and the canonical native channel; the owned rules file is now migrated away.
- **Claude Code on `AGENTS.md` only.** Rejected outright now that the docs are clear: Claude Code does not auto-read AGENTS.md, so this would deliver nothing to Claude Code.
- **Plugin owns the nudge.** Rejected: CLI-only and plugin-less hosts would get nothing — exactly the users this targets. The CLI owns the baseline block for all hosts; the plugin can later enrich or defer to the same marker.
- **Always write without a confirm** (as hooks/MCP do). Rejected for `init`: `CLAUDE.md` / `AGENTS.md` / `GEMINI.md` are user-curated content files, not machine config, so prose is appended only with consent. The fenced upsert keeps it non-destructive and `instructions remove` makes it reversible.
- **Per-host native files everywhere** (`.cursor/rules/archcore.mdc` with `alwaysApply`, `.roo/rules/`, `.clinerules/`). Deferred: v1 relies on each host's native `AGENTS.md` support for single-exposure simplicity.
- **Unify Gemini onto `AGENTS.md`** by editing `contextFileName` in `.gemini/settings.json`. Rejected for v1: invasive settings edit; a plain `GEMINI.md` write is simpler.
- **Blunt "Do NOT use Archcore for…" prohibition keyed on rule-presence.** Rejected: gating on "general programming help with no project-specific rule attached" asks the agent to predict a fact about the doc store it cannot know without the very lookup it is told to skip. It silently suppressed invocation exactly when a doc *did* apply — which reads to users as a dead, useless integration — and the failure worsens as the store grows. The relevance-and-cost framing in **Content** keeps a brake (skip trivial turns) without the unknowable precondition.

## Consequences

- CLI-only users (and plugin-less hosts) now discover and invoke Archcore without manual prompting — the gap this closes.
- Non-destructive (fenced upsert) and reversible (`instructions remove`).
- Claude Code carries the nudge in `CLAUDE.md` (what it reads) plus an `AGENTS.md` block (for the plugin and other hosts). Because Claude Code does not load AGENTS.md, there is no duplicated context cost; the only maintenance hazard is the two blocks drifting if one write path changes without the other — mitigated by both flowing through the same `writeClaudeInstructions` helper and the shared body constant.
- Migration: repos wired by an older CLI have a stale `.claude/rules/archcore.md`; it is removed on the next `writeClaudeInstructions` (init/instructions install) — no manual cleanup needed.
- Cost: one extra opt-in prompt during interactive `init`; non-interactive `init` skips the nudge, so automation must call `archcore instructions install` explicitly.
- Maintenance: every agent must wire `InstructionsPath`, `WriteInstructions`, and `RemoveInstructions`. This is enforced by `TestAllAgents_RequiredFields`, so a new agent cannot silently omit a target.
- Helpers live in `internal/agents/instructions.go`; the command layer (`cmd/instructions.go`) handles dedup, display, and the `install`/`remove` subcommands; the `init` opt-in lives behind the `confirmInstructions` seam in `cmd/init.go`.
