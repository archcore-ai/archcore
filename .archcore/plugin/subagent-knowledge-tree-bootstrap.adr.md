---
title: "Mandate Knowledge Tree Bootstrap in Sub-Agent Preamble"
status: accepted
tags:
  - "agents"
  - "architecture"
  - "plugin"
---

## Context

Sub-agents (`archcore-assistant`, `archcore-auditor`) are spawned through the Task tool and do not receive the `SessionStart` additional context the main conversation gets, so the main session starts with a loaded knowledge tree — document inventory, tags, relation count — while a sub-agent starts blind. Before this decision nothing in either agent's system prompt made loading that tree a mandatory first step: a well-behaved agent often called `list_documents` early, but that was emergent behavior rather than contract. The JTBD-versus-implementation audit in `jtbd-alignment-analysis.idea` identified the gap as the cheapest lever on the critical path of JTBD #1, making a feature without breaking the repository's logic, and JTBD #2, continuing work without re-explaining the project, whenever work is delegated.

## Observed and predicted gaps

- Near-duplicate documents, when a sub-agent proposes an ADR or spec that already exists.
- Orphaned documents, when a sub-agent creates content without linking it to existing related documents.
- Per-document audits from `archcore-auditor` that miss graph-level problems — orphans, broken chains, coverage gaps.

A complicating factor shaped the record itself: `remove-skill-verify-mcp-preamble.cpat` removed a similar-looking "Step 0: Verify MCP" preamble from every `SKILL.md`, so a future cleanup pass could read the new sub-agent preamble and delete it by analogy, reintroducing the original problem. This decision therefore states the boundary explicitly.

## Decision

Every sub-agent invocation MUST bootstrap the knowledge tree first: both `agents/archcore-assistant.md` and `agents/archcore-auditor.md` carry a `# First Step — Bootstrap Knowledge Tree` section at the top of the system prompt mandating parallel `list_documents` and `list_relations` calls before any domain action, followed immediately by a synthesis of the categories present, the most common tags, recent accepted decisions, and any draft plans.

The synthesis step is output-shaping: it uses only data the two bootstrap calls already returned and adds no tool call, mirroring the situational summary the main session receives from `SessionStart`. The implementation is Option A from `subagent-knowledge-tree-preload.idea` — a prompt preamble — chosen because it is host-portable, ships in a small diff, and keeps the agent's view live with no cache-staleness concern.

Scope of the mandate: `archcore-assistant` requires both calls, with a narrow exception for a strictly single-document read at an explicit path, where `get_document` alone is acceptable; `archcore-auditor` requires both calls with no exception, because an audit without the full graph produces incomplete findings.

This decision does not conflict with `remove-skill-verify-mcp-preamble.cpat`. That pattern removed an *availability check* from `SKILL.md` files used inside the main session, where MCP is always available and `SessionStart` already loaded the tree. This one adds a *knowledge bootstrap* to agent files used in sub-agent sessions, where MCP is still available but the tree was never loaded because `SessionStart` did not fire. Different surface, different problem, different rationale — and the sub-agent preamble must not be removed by analogy.

## Alternatives Considered

1. **Option B — snapshot injection at Task dispatch time**, extending `archcore hooks <host>` with a `subagent-start` sub-command emitting the same payload as `session-start`, and wrapping the Task tool so the snapshot is prepended to the sub-agent's system prompt at zero runtime tool-call cost — rejected as the initial implementation because Claude Code documents no sub-agent-start hook surface, so any injection would need a brittle plugin-level wrapper or uncontrolled upstream host changes, and Cursor's distinct sub-agent lifecycle would need its own adapter. Kept on the table as a future optimization.
2. **Option C — hybrid**, shipping Option A now and adding Option B once host support matures — formally adopted as the rollout strategy. This decision implements the A portion, with B deferred until a sub-agent lifecycle hook exists on at least Claude Code.
3. **Status quo, relying on emergent behavior** — rejected because the JTBD audit showed concrete gaps where sub-agents produce worse outcomes than the main session, and emergent behavior is neither a contract nor testable.
4. **Cache the snapshot as a file and have sub-agents read it** — rejected because file-based caching introduces invalidation problems when the main session mutates the tree during the sub-agent's lifetime, which Option A sidesteps because both calls are always live.

## Consequences

- Sub-agents start from the same baseline as the main session, removing the asymmetry where delegated work produced worse outcomes than direct work.
- [expected] Near-duplicate document risk drops, because the sub-agent sees existing documents on turn one.
- [expected] The orphaned-document rate drops, because the sub-agent sees the relation graph and can link new content to existing nodes.
- `archcore-auditor` findings gain graph-level coverage — orphans, broken chains, coverage gaps — without changing the audit dimensions.
- The `pre-code-context-injection.idea` rollout is unblocked, because that hook's value in a sub-agent session depends on the sub-agent already holding the tree structure that contextualizes each injection.
- The synthesis directive closes the remaining asymmetry: the main session receives a pre-distilled SessionStart summary, and sub-agents now derive the equivalent from their bootstrap calls.
- Tradeoff: two additional tool calls open every sub-agent invocation. For a narrow single-read task this is overhead, partially mitigated by the `archcore-assistant` exception.
- Tradeoff: the preamble adds about 20 lines to each system prompt.
- Tradeoff: bootstrap token cost scales with knowledge-base size. At the repository's state when this was decided — about 35 documents and 646 relations — the payload is small; at 10× scale it could be meaningful, and the mitigation path is Option B.
- Tradeoff: the synthesis directive relies on model compliance. Structural tests verify only that the prompt text is present, not that the agent produces the summary. The escalation path is a richer template or the Option B snapshot.
- Tradeoff: a future cleanup could remove the preamble by analogy with the CPAT. Mitigated by the explicit cross-reference in both the preamble text and this record.

## Constraints

1. Both agent files MUST carry a `# First Step — Bootstrap Knowledge Tree` section as the first content section after the YAML frontmatter.
2. That section MUST name both `list_documents` and `list_relations`.
3. That section MUST direct the agent to note categories, common tags, recent accepted decisions, and draft plans after the two calls return, with the anchor literal `recent accepted decisions` present in both files.
4. That section MUST cross-reference `remove-skill-verify-mcp-preamble.cpat`, explaining why removal by analogy is wrong.
5. That section MUST cross-reference this decision.
6. `@test/structure/agents.bats` MUST assert all five strings — `First Step — Bootstrap Knowledge Tree`, `list_documents`, `list_relations`, `subagent-knowledge-tree-bootstrap.adr`, and `recent accepted decisions` — in both agent files.
7. A removal or structural change to the preamble MUST go through an update to this record rather than an ad-hoc edit.

## Superseded when

- Claude Code ships a documented sub-agent-start hook surface, which would make Option B implementable and remove the two runtime tool calls.
- Bootstrap payload size measured on a real knowledge base exceeds the sub-agent's useful context budget, which would force Option B regardless of host support.
