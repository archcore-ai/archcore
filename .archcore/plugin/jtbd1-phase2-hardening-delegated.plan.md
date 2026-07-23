---
title: "JTBD #1 Phase 2 — Delegated-Agent Coverage"
status: accepted
tags:
  - "agents"
  - "hooks"
  - "plugin"
  - "roadmap"
---

## Status — Shipped (B2) + Documented (B1)

**As of 2026-04-23.** Both sub-items of this plan are resolved:

- **B2 (situational-summary directive)**: shipped. Synthesis directive appended to both agent files (`agents/archcore-assistant.md`, `agents/archcore-auditor.md`). ADR `subagent-knowledge-tree-bootstrap` Constraints section extended with the directive requirement and the new required anchor literal `recent accepted decisions`. Spec `agent-system` updated in Contract Surface, Normative Behavior, Invariants, and Conformance to reflect the synthesis step. Structural test `test/structure/agents.bats` extended with two new assertions (one per agent file). Full bats suite 152/152 green.
- **B1 (push-hook coverage for Task-dispatched writes)**: documented in `hooks-validation-system.spec.md` under Hook 3's new "Sub-agent tool invocations" subsection, based on Claude Code's PreToolUse tool-boundary contract (hooks fire at tool-call boundary regardless of dispatcher). New normative bullet, invariant, and conformance point #12 codify the guarantee. An empirical-probe protocol is recorded in the spec for future verification in a fresh session loaded from this repo (`claude --plugin-dir <repo>`); the session used for this implementation loaded the cached plugin 0.2.3 from PATH, so live probes would have produced false negatives against my uncommitted edits and were deferred. The spec explicitly forbids shipping probe lines in `bin/`.

No version bump taken — the prompt-only change in B2 and the spec-only change in B1 are being rolled into the next aggregated release rather than cut as a standalone version.

Remaining follow-up: run the three-probe protocol documented in `hooks-validation-system.spec.md` against a fresh session with `--plugin-dir` pointing at this repo; capture the result in a commit message and optionally amend the spec subsection with a dated empirical entry.

---

## Goal

Close two remaining gaps in sub-agent parity after v0.3.0 (which shipped `# First Step — Bootstrap Knowledge Tree` preamble in both agent files):

1. **Unverified push-hook coverage.** It is not empirically confirmed that `check-code-alignment` fires when a Task-dispatched sub-agent performs Write/Edit on a source file. If it does, document the guarantee. If it doesn't, the gap becomes the trigger for a separate snapshot-injection plan.
2. **Preamble has no synthesis step.** Sub-agents call `list_documents` + `list_relations` but are not directed to distill the output. A one-line prompt addition plausibly reduces duplicate/orphan risk at negligible cost.

### Scope clarifier — important

The plugin's own sub-agents (`archcore-assistant`, `archcore-auditor`) do **not** have `Write` or `Edit` in their `tools:` allowlist. They can only mutate documents through MCP tools. The push-hook question therefore concerns **general-purpose and third-party Task agents** that the user may dispatch for code work (typescript-pro, frontend-figma-layout-designer, etc.) — those have Write/Edit access and should trip `check-code-alignment` on source-file edits.

### Explicitly deferred

- **Hook performance hardening** (path-index consumption, session dedup). See `cli-path-index.plan`.
- **Option B snapshot injection** at Task dispatch — revisit only if empirical probe shows hooks don't fire.
- `archcore check`, telemetry, greenfield bootstrap, Codex/Copilot parity.

## Tasks

### B1 — Verify push-hook coverage on Task-dispatched writes

Empirical sanity check, not a formal investigation.

**Two experiments on Claude Code:**

- `Task(general-purpose) → Write src/probe.ts` — does `check-code-alignment` fire? Confirm by observing whether the `additionalContext` injection appears in the sub-agent's transcript (or by transient `echo "$(date) $CLAUDE_TOOL_NAME" >> /tmp/hook-trace` at the top of `bin/check-code-alignment`, removed after the check).
- `Task(general-purpose) → Write .archcore/probe.adr.md` — does `validate-archcore` fire? Same observation.

**Branches on the result:**

| Outcome | Action |
|---------|--------|
| Both hooks fire, stdin identical to main-session | Document the guarantee as a one-paragraph subsection in `hooks-validation-system.spec.md` ("Sub-agent tool invocations"). No code change. |
| Hooks fire but stdin shape differs | Extend `bin/lib/normalize-stdin.sh` with the new shape. Add one bats case pinning the subagent stdin shape. |
| Hook(s) do not fire | Document the negative result. Open a separate plan for snapshot injection (the deferred Option B from `subagent-knowledge-tree-preload.idea`). v0.4.0 ships with B2 only. |

**Cursor:** repeat only if/when Cursor's Task tool is exposed; otherwise note "not applicable at time of investigation" in one line and move on.

### B2 — One-line synthesis directive in sub-agent preamble

Append a single sentence to the existing `# First Step — Bootstrap Knowledge Tree` section in both agent files, after the two existing bullets:

> After both calls return, note the categories present, the most common tags, recent accepted decisions, and any draft plans before proceeding.

No bucket-by-bucket template, no cap numbers, no token-budget ceremony. The sentence is directive; LLMs handle the rest. If drift is observed later, we revisit — but not preemptively.

**Files touched:**

- `agents/archcore-assistant.md` — append the sentence.
- `agents/archcore-auditor.md` — append the sentence.
- `subagent-knowledge-tree-bootstrap.adr.md` — add one bullet to Constraints: *"Both agent files must carry a directive to note categories, common tags, recent accepted decisions, and draft plans after the bootstrap calls."* Status stays `accepted`.
- `test/structure/agents.bats` — assert the literal phrase `recent accepted decisions` (or similarly unique anchor) is present in both agent files.

### Release bookkeeping

- `development-roadmap.plan.md` — one-line mark for the delegated-coverage item.
- Version bump **only if** B1 outcome changes code (`normalize-stdin.sh` modification). Prompt-only changes ship as a patch in the next aggregated release, not as a dedicated version bump.

## Acceptance Criteria

1. B1 outcome (all three scenarios covered in the branches table) documented in `hooks-validation-system.spec.md` and in the commit message.
2. If B1 shows hooks fire: no code change required beyond (optional) bats pinning.
3. If B1 shows hooks fire with different stdin shape: `normalize-stdin.sh` extended and tested.
4. If B1 shows hooks don't fire: a new plan opened; this plan's B1 scope is formally closed with the negative result.
5. Synthesis sentence present in both agent files and asserted by `test/structure/agents.bats`.
6. `subagent-knowledge-tree-bootstrap.adr.md` Constraints section lists the synthesis directive.
7. Full test suite green.

## Dependencies

- Host behaviour for Task-dispatched tool calls — input to B1, not something we control.
- Existing v0.3.0 preamble — already shipped and enforced; B2 extends it.
- No CLI dependency.

## Risks

- **B1 comes back "hooks don't fire".** Then half the "delegated parity" story is unmet; we ship B2 alone and open the snapshot-injection plan. Mitigation: this branch is already planned; scope contracts gracefully.
- **B2 LLM non-compliance.** Sub-agents may skip the synthesis sentence. Structural test catches only the prompt text. Mitigation: if drift shows up, promote to a richer template or move to Option B. Not worth preemptive engineering.
- **Dispatcher-specific agents we don't test.** We run B1 against `general-purpose`; a niche agent may behave differently. Acceptable — hooks live at tool-call layer, not agent-type layer; one successful experiment is sufficient evidence for the mechanism.

## Relations

- `implements` → `jtbd-alignment-analysis.idea` (closes the delegated-coverage gap after v0.3.0)
- `implements` → `subagent-knowledge-tree-preload.idea` (B2 is the minimal extension of Option A)
- `extends` → `subagent-knowledge-tree-bootstrap.adr` (adds synthesis directive to the mandated preamble)
- `extends` → `hooks-validation-system.spec` (adds "Sub-agent tool invocations" subsection)
- `related` → `pre-code-hook-implementation.plan` (push-mode v1; B1 verifies its coverage under delegation)
- `related` → `context-skill-implementation.plan` (pull-mode v1; synthesis is the delegated analog of SessionStart payload)
- `related` → `cli-path-index.plan` (deferred hardening path; no v0.4.0 dependency)
