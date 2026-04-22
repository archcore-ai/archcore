---
title: "Preload Knowledge Tree into Sub-Agents at Invocation"
status: draft
tags:
  - "agents"
  - "architecture"
  - "plugin"
---

## Idea

When a sub-agent (`archcore-assistant`, `archcore-auditor`) is spawned via the Task tool, it starts without any view of the existing `.archcore/` knowledge base. The `SessionStart` hook only runs for the main conversation — sub-agents do not inherit that `additionalContext`. As a result, sub-agents operate blind until they happen to call `list_documents` / `list_relations` on their own, and nothing in their system prompt makes that the mandatory first step.

Preload the knowledge tree (documents grouped by category, tags, relation count, possibly top-level relation summary) into every sub-agent's context at invocation time, so the agent begins with the same "EXISTING DOCUMENTS" snapshot that the main session gets from `SessionStart`.

Two candidate mechanisms:

1. **Prompt preamble**: add an explicit first-step instruction to `agents/archcore-assistant.md` and `agents/archcore-auditor.md`: *"Before any other action, call `list_documents` and `list_relations` to load the project's knowledge tree."* Low-risk, works in any host.
2. **Injected snapshot**: at dispatch time, prepend a snapshot block (same shape as `SessionStart` output) to the sub-agent's system prompt — either via a plugin-level wrapper or by having the Task invocation fetch it. Zero tool-calls overhead but requires host-specific plumbing.

## Value

- **Better first-shot decisions**: the sub-agent can disambiguate (e.g., "is there already an ADR on this?") before proposing an action, instead of creating near-duplicates.
- **Fewer round-trips**: today a well-behaved sub-agent still burns 1–2 tool calls bootstrapping its view; a preloaded snapshot removes that cost.
- **Consistency with main session**: the main session already gets the tree for free via `SessionStart`; sub-agents should operate from the same baseline so user experience doesn't degrade when work is delegated.
- **Lower risk of orphaned documents**: agents that "see" the relation graph are more likely to link new docs to relevant existing ones, which is one of the current documented gaps audited by `archcore-auditor`.

## Possible Implementation

**Option A — Preamble (cheapest, portable):**
- Edit `agents/archcore-assistant.md` and `agents/archcore-auditor.md` to add a `# First Step` section mandating `list_documents` + `list_relations` before any domain action.
- Update tests in `test/structure/` if they assert on agent file structure.
- Document the pattern in `.archcore/plugin/plugin-component-architecture.adr.md` or the skills/agents spec.

**Option B — Snapshot injection (richest, more work):**
- Extend `archcore hooks <host>` with a `subagent-start` sub-command that emits the same payload as `session-start`.
- Wrap sub-agent invocation so the snapshot is prepended. On Claude Code this likely requires a dedicated mechanism (sub-agents don't currently consume `SessionStart` output); on Cursor the hook surface may differ.
- Cache the snapshot per-session to avoid re-computing for each sub-agent dispatch.

**Option C — Hybrid:** ship Option A immediately (unblocks the problem today), keep Option B as a follow-up once the sub-agent lifecycle story across hosts matures.

Recommend starting with Option A. It's a small diff, does not require host-specific coordination, and can be removed without harm if Option B later supersedes it.

## Risks and Constraints

- **Preamble drift**: adding a mandatory preamble re-introduces a pattern that was previously removed from SKILL.md files (see `experience/remove-skill-verify-mcp-preamble.cpat.md`). The contexts differ — skills run inside the main session with `SessionStart` already applied, sub-agents do not — but this distinction must be made explicit in the preamble rationale to avoid future cleanup passes deleting it by analogy.
- **Token cost**: for large `.archcore/` trees the snapshot is non-trivial. The current main-session snapshot for this project is ~30 documents + tag list + relation count; at 10× scale the preamble/injection could meaningfully bloat the sub-agent's context. May need a compact format or pagination.
- **Staleness within sub-agent session**: if the sub-agent mutates the tree and then reads the cached snapshot, its view goes stale. Option A sidesteps this (the agent calls `list_documents` live); Option B must invalidate the cache on mutation.
- **Host compatibility**: Option B depends on sub-agent-start hooks or injection points that may not exist uniformly across Claude Code, Cursor, and future hosts. Pushes complexity into the multi-host compatibility layer.
- **Over-fetching**: not every sub-agent task needs the full graph (e.g., a single-document update). A preamble that always runs `list_relations` costs tool calls even when unused. Mitigation: make the relations call conditional on task intent, or use Option B where it's zero-cost.
