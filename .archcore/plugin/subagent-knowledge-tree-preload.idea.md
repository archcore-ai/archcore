---
title: "Preload Knowledge Tree into Sub-Agents at Invocation"
status: accepted
tags:
  - "agents"
  - "architecture"
  - "plugin"
---

## Idea

A sub-agent spawned through the Task tool starts with no view of the existing `.archcore/` knowledge base, because the `SessionStart` hook runs only for the main conversation and sub-agents do not inherit its `additionalContext`. A sub-agent therefore operates blind until it happens to call `list_documents` and `list_relations` on its own, and nothing in its system prompt makes that the mandatory first step.

Preload the knowledge tree — documents grouped by category, the tag list, the relation count, and possibly a top-level relation summary — into every sub-agent's context at invocation, so the agent begins from the same snapshot the main session receives.

Two mechanisms are candidates. A **prompt preamble** adds an explicit first-step instruction to both agent definitions, which is low-risk and works on any host. An **injected snapshot** prepends a `SessionStart`-shaped block to the sub-agent's system prompt at dispatch time, either through a plugin-level wrapper or by having the Task invocation fetch it, which costs no tool calls but requires host-specific plumbing.

## Value

**Status upgrade, 2026-04-22.** After the JTBD-versus-implementation audit in `jtbd-alignment-analysis.idea`, this proposal sits on the critical path of JTBD #1, making a feature without breaking the repository's logic, and JTBD #2, continuing work without re-explaining the project. Any delegated work breaks both promises while the sub-agent starts blind, so this is the cheapest lever that closes part of the gap between the README promise and the engineered reality.

- Better first-shot decisions: the sub-agent can ask whether an ADR already exists before proposing one, instead of creating a near-duplicate.
- Fewer round-trips: a well-behaved sub-agent still burns one or two tool calls bootstrapping its view, and a preloaded snapshot removes that cost.
- Consistency with the main session, which already gets the tree for free, so the user experience does not degrade when work is delegated.
- Lower risk of orphaned documents, because an agent that sees the relation graph is likelier to link new content to existing nodes — one of the gaps `archcore-auditor` audits.
- A prerequisite for sub-agent coverage of `pre-code-context-injection.idea`, which injects per-edit constraints that a sub-agent cannot contextualize without the surrounding structure. The two mechanisms compound, and delivering one without the other leaves coverage incomplete.

## Possible Implementation

**Option A, the preamble — cheapest and portable.** Add a `# First Step` section to `agents/archcore-assistant.md` and `agents/archcore-auditor.md` mandating `list_documents` and `list_relations` before any domain action; update the structure tests that assert on agent file shape; and document the pattern in the agents spec.

**Option B, snapshot injection — richer and more work.** Extend `archcore hooks <host>` with a `subagent-start` sub-command emitting the same payload as `session-start`, wrap sub-agent invocation so the snapshot is prepended, and cache the snapshot per session to avoid recomputing it per dispatch. Claude Code needs a dedicated mechanism, since sub-agents do not consume `SessionStart` output, and Cursor's hook surface differs.

**Option C, hybrid.** Ship Option A immediately and keep Option B as a follow-up once the sub-agent lifecycle story matures across hosts.

Option A is the recommendation: a small diff, no host-specific coordination, and removable without harm if Option B later supersedes it. `subagent-knowledge-tree-bootstrap.adr` adopted exactly this, as Option C with the A portion implemented.

## Risks

- **Preamble drift.** A mandatory preamble reintroduces a shape that was previously removed from `SKILL.md` files by `remove-skill-verify-mcp-preamble.cpat`. The contexts differ — a skill runs inside the main session with `SessionStart` already applied, a sub-agent does not — and that distinction must be explicit in the preamble itself, so a future cleanup pass does not delete it by analogy.
- **Token cost.** For a large tree the snapshot is non-trivial. This project's main-session snapshot was about 30 documents plus the tag list and relation count when the idea was written; at 10× scale the preamble or injection could bloat the sub-agent's context, which may call for a compact format or pagination.
- **Staleness inside a sub-agent session.** If the sub-agent mutates the tree and then reads a cached snapshot, its view goes stale. Option A sidesteps this because the calls are live; Option B must invalidate the cache on mutation.
- **Host compatibility.** Option B depends on sub-agent-start hooks or injection points that may not exist uniformly across hosts, which pushes complexity into the multi-host compatibility layer.
- **Over-fetching.** Not every sub-agent task needs the full graph — a single-document update does not — so a preamble that always calls `list_relations` costs tool calls when unused. The mitigation is to make that call conditional on task intent, or to use Option B where it is free.
