---
title: "JTBD #1 Phase 2 — Delegated-Agent Coverage"
status: rejected
tags:
  - "agents"
  - "hooks"
  - "plugin"
  - "roadmap"
---

**Status — B2 shipped, B1 documented, as of 2026-04-23.**

The synthesis directive of B2 shipped: the sentence was appended to both agent files, the Constraints section of `subagent-knowledge-tree-bootstrap.adr` gained the directive requirement and the required anchor literal `recent accepted decisions`, `agent-system.spec` was updated across its Surface, Normative Behavior, Invariants, and Conformance sections, and `@test/structure/agents.bats` gained two assertions, one per agent file, with the full bats suite green at 152 of 152.

The push-hook coverage of B1 was documented in `hooks-validation-system.spec` under the sub-agent subsection of Hook 3, resting on Claude Code's PreToolUse tool-boundary contract, where hooks fire at the tool-call boundary whichever dispatcher issued the call. A normative item, an invariant, and a conformance point codify the guarantee, and an empirical-probe protocol is recorded for verification in a fresh session loaded from this repository. The session used for the implementation loaded a cached plugin from PATH, so a live probe would have produced a false negative against the uncommitted edits, and it was deferred. The specification forbids shipping a probe line in `bin/`.

No version bump was taken: the prompt-only change of B2 and the specification-only change of B1 roll into the next aggregated release rather than a standalone version. The remaining follow-up is to run the probe protocol against a fresh session pointed at this repository, capture the result in a commit message, and optionally add a dated empirical entry to the specification subsection.

## Goal

Close two remaining gaps in sub-agent parity after v0.3.0, which shipped the bootstrap preamble in both agent files.

1. **Push-hook coverage is unverified.** It is not empirically confirmed that `check-code-alignment` fires when a Task-dispatched sub-agent performs a Write or Edit on a source file. If it does, document the guarantee; if it does not, the gap becomes the trigger for a separate snapshot-injection plan.
2. **The preamble has no synthesis step.** A sub-agent calls `list_documents` and `list_relations` but is not directed to distill the output. [assumption] A one-line prompt addition reduces duplicate and orphan risk at negligible cost.

**Scope clarifier.** The plugin's own sub-agents hold no `Write` or `Edit` in their tool allow-list and can mutate documents only through MCP, so the push-hook question concerns general-purpose and third-party Task agents that a user dispatches for code work, which do have write access and should trip the injection guard on a source-file edit.

**Explicitly deferred.** Hook performance hardening — path-index consumption and session dedup — is tracked in `cli-path-index.plan`. Snapshot injection at Task dispatch is revisited only if the probe shows hooks do not fire. `archcore check`, telemetry, greenfield bootstrap, and Codex or Copilot parity are all out of scope.

## Tasks

### B1 — verify push-hook coverage on Task-dispatched writes

An empirical sanity check rather than a formal investigation, run as two experiments on Claude Code. First, dispatch a general-purpose Task that writes `src/probe.ts` and determine whether the injection guard fires, confirmed either by the injected context appearing in the sub-agent's transcript or by a transient trace line at the top of the script, removed after the check. Second, dispatch a Task that writes a `.archcore/` markdown file and determine whether the validator fires, observed the same way.

| Outcome | Action |
|---------|--------|
| Both hooks fire, stdin identical to the main session | Document the guarantee as one subsection in `hooks-validation-system.spec`. No code change. |
| Hooks fire but the stdin shape differs | Extend `bin/lib/normalize-stdin.sh` with the new shape and add one bats case pinning it. |
| A hook does not fire | Document the negative result and open a separate plan for snapshot injection. The release ships B2 alone. |

On Cursor, repeat only once its Task tool is exposed; otherwise record one line noting that it is not applicable at the time of investigation.

### B2 — a one-line synthesis directive in the sub-agent preamble

Append a single sentence to the existing bootstrap section in both agent files, after the two existing bullets: *"After both calls return, note the categories present, the most common tags, recent accepted decisions, and any draft plans before proceeding."* No bucket-by-bucket template, no cap numbers, and no token-budget ceremony — the sentence is directive and the model handles the rest. If drift appears later, revisit it then rather than preemptively.

Files touched: both agent definitions gain the sentence; `subagent-knowledge-tree-bootstrap.adr` gains one Constraints item requiring the directive, with its status unchanged; and `@test/structure/agents.bats` asserts the anchor literal in both files.

### Release bookkeeping

Mark the delegated-coverage item in `development-roadmap.plan`. Bump the version only if the B1 outcome changes code — that is, only if `normalize-stdin.sh` is modified. A prompt-only change ships as a patch in the next aggregated release rather than as a dedicated bump.

## Acceptance Criteria

1. The B1 outcome, covering all three branches of the table, is documented in `hooks-validation-system.spec` and in the commit message.
2. IF B1 shows the hooks firing, THEN no code change is required beyond optional bats pinning.
3. IF B1 shows the hooks firing with a different stdin shape, THEN `normalize-stdin.sh` is extended and tested.
4. IF B1 shows a hook not firing, THEN a new plan is opened and this plan's B1 scope closes with the negative result.
5. The synthesis sentence is present in both agent files and asserted by the structure test.
6. The Constraints section of `subagent-knowledge-tree-bootstrap.adr` lists the synthesis directive.
7. The full test suite is green.

## Dependencies

- Host behavior for a Task-dispatched tool call, which is input to B1 rather than something this repository controls.
- The existing v0.3.0 preamble, already shipped and enforced, which B2 extends.
- No CLI dependency.

## Risks

- **B1 returns "hooks do not fire".** Half the delegated-parity story would then be unmet, so B2 ships alone and the snapshot-injection plan opens. The branch is already planned, so scope contracts gracefully.
- **The model does not comply with B2.** A sub-agent may skip the synthesis sentence, and the structural test catches only the prompt text. If drift appears, promote to a richer template or move to snapshot injection; it is not worth preemptive engineering.
- **Untested dispatcher-specific agents.** B1 runs against the general-purpose agent, and a niche agent may behave differently. Acceptable, because hooks live at the tool-call layer rather than the agent-type layer, so one successful experiment is sufficient evidence for the mechanism.
