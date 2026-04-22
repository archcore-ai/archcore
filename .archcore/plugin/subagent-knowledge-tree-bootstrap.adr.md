---
title: "Mandate Knowledge Tree Bootstrap in Sub-Agent Preamble"
status: accepted
tags:
  - "agents"
  - "architecture"
  - "plugin"
---

## Context

Sub-agents (`archcore-assistant`, `archcore-auditor`) are spawned via the Task tool and do NOT receive the `SessionStart` additional context that the main conversation gets. The main session starts with a loaded knowledge tree (document inventory, tags, relation count) injected by the session-start hook; sub-agents start blind.

Before this decision, nothing in either agent's system prompt made loading the knowledge tree a mandatory first step. A well-behaved agent would often call `list_documents` early, but this was emergent behavior, not contract. Consequences observed or predicted:

- Near-duplicate documents when the sub-agent proposes creating an ADR/spec that already exists
- Orphaned documents when the sub-agent creates new content without linking to existing related docs
- Per-document audits from `archcore-auditor` that miss graph-level problems (orphans, broken chains, coverage gaps)

The JTBD-vs-implementation audit in `jtbd-alignment-analysis.idea` identified this as the cheapest lever on the critical path of JTBD #1 ("make a feature without breaking the repo's logic") and JTBD #2 ("continue work without re-explaining the project") whenever work is delegated to a sub-agent.

A complicating factor: `remove-skill-verify-mcp-preamble.cpat` (accepted) explicitly removed a similar-looking "Step 0: Verify MCP" preamble from every SKILL.md file because MCP is always available under the bundled CLI launcher and the block was dead code. A future cleanup pass could read the new sub-agent preamble and delete it by analogy, re-introducing the original problem. The decision record here must establish the boundary explicitly.

## Decision

Every sub-agent invocation MUST bootstrap the knowledge tree as its first action. Both `agents/archcore-assistant.md` and `agents/archcore-auditor.md` carry a `# First Step — Bootstrap Knowledge Tree` section at the top of the system prompt that mandates parallel calls to `list_documents` and `list_relations` before any domain action.

Implementation: Option A from `subagent-knowledge-tree-preload.idea` — prompt preamble. Not Option B (snapshot injection) because that requires host-specific plumbing (no sub-agent-start hook exists on Claude Code; Cursor's surface differs) and couples invocation logic to host capabilities. Option A is host-portable, ships in a small diff, and keeps the agent's view live (no cache-staleness concern).

Scope of the mandate:

- **archcore-assistant**: both calls required; narrow exception for strictly single-document reads with explicit paths (e.g., "show me `.archcore/auth/jwt.adr.md`") where `get_document` alone is acceptable.
- **archcore-auditor**: both calls required, no exceptions — audits without the full graph produce incomplete findings.

Enforcement: `test/structure/agents.bats` asserts the preamble section, both tool names, and the cross-reference to this ADR are present in both agent files.

This decision is distinct from, and does not conflict with, `remove-skill-verify-mcp-preamble.cpat`. The cpat removed an *availability check* preamble from SKILL.md files used inside the main session (where MCP is always available and `SessionStart` already loaded the tree). This ADR adds a *knowledge bootstrap* preamble to agent files used in sub-agent sessions (where MCP is still available, but the tree has not been loaded because `SessionStart` did not fire for the sub-agent). Different surface, different problem, different rationale — the two are not redundant and the sub-agent preamble must not be removed by analogy.

## Alternatives Considered

### Option B — Snapshot injection at Task dispatch time

Extend `archcore hooks <host>` with a `subagent-start` sub-command that emits the same payload as `session-start`, and wrap Task-tool invocation so the snapshot is prepended to the sub-agent's system prompt. Zero tool-calls overhead at runtime.

Rejected as the initial implementation because Claude Code has no documented sub-agent-start hook surface; any injection mechanism would require either a plugin-level wrapper (brittle) or upstream host changes (uncontrolled). Cursor's sub-agent lifecycle is distinct and would need its own adapter. The cost of correctness across hosts exceeds the value for a problem that Option A solves adequately. Kept on the table as a future optimization if tool-call overhead from Option A becomes a measurable issue at scale.

### Option C — Hybrid

Ship Option A now; add Option B later once host support matures. Formally adopted as the rollout strategy — this ADR implements the "A" portion, with "B" deferred until a sub-agent lifecycle hook exists on at least Claude Code.

### Status quo — rely on emergent behavior

Rejected. The JTBD audit showed concrete gaps where sub-agents produce worse outcomes than the main session because of the missing context. Emergent behavior is not a contract and cannot be enforced by tests.

### Cache the snapshot as a file, have sub-agents Read it

Rejected. File-based caching introduces invalidation problems when the main session mutates the tree during the sub-agent's lifetime. Option A sidesteps this because `list_documents` and `list_relations` are always live.

## Consequences

### Positive

- Sub-agents start with the same baseline the main session has, eliminating the asymmetry where delegated work produces worse outcomes than direct work.
- Near-duplicate document risk drops because the sub-agent sees existing documents on turn one.
- Orphaned-document rate drops because the sub-agent sees the relation graph and can link new content to existing nodes.
- `archcore-auditor` findings gain graph-level coverage (orphans, broken chains, coverage gaps) without changing the auditor's audit dimensions.
- Unblocks the `pre-code-context-injection.idea` rollout — that hook injects per-edit constraints, but its value in sub-agent sessions depends on the sub-agent already having the tree structure to contextualize each injection.

### Negative

- Two additional tool calls at the start of every sub-agent invocation. For narrow tasks (single read) this is overhead; the explicit exception for `archcore-assistant` partially mitigates.
- The preamble adds ~20 lines to each agent's system prompt. Negligible token cost; trivial maintenance.
- Token cost of the bootstrap scales with knowledge-base size. At ~35 documents and 646 relations in this repo (current state), the payload is small. At 10× scale the payload could be meaningful — mitigation path: switch to Option B at that point.
- Future risk of cleanup-by-analogy with `remove-skill-verify-mcp-preamble.cpat`. Mitigated by explicit cross-reference in both the preamble text and in this ADR.

### Constraints

- Both agent files MUST carry a `# First Step — Bootstrap Knowledge Tree` section as the first content section after the YAML frontmatter.
- The section MUST reference both `list_documents` and `list_relations` by name.
- The section MUST include a cross-reference to `remove-skill-verify-mcp-preamble.cpat` explaining why removal by analogy is wrong.
- The section MUST include a cross-reference to this ADR.
- `test/structure/agents.bats` MUST assert all four strings (`First Step — Bootstrap Knowledge Tree`, `list_documents`, `list_relations`, `subagent-knowledge-tree-bootstrap.adr`) are present in both agent files.
- Removal or structural changes to the preamble MUST go through an ADR update, not an ad-hoc edit.
