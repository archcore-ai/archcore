---
title: "Remove Document-Type Skills — Collapse Layer 3 into Intent and Track Skills"
status: accepted
---

## Context

Three facts made the per-document-type skill layer (formerly "Layer 3" in `intent-based-skill-architecture.adr.md`) obsolete:

**1. Content duplication.** Every creation-oriented intent skill (`decide`, `capture`, `standard`, `plan`) and every track skill inlines per-type elicitation — the same questions, section lists, and MCP calls that type-skill `Quick Create` sections already carried. The inline line "Ask: What was the decision? What alternatives were considered? Compose content covering Context, Decision, Alternatives Considered, Consequences" appeared verbatim in both `skills/adr/SKILL.md` and `skills/decide/SKILL.md` Step 3. Track skills (iso-track, sources-track, architecture-track, standard-track, feature-track, product-track) contain the full per-type question-and-section flow for every document they create. The type skill's unique contribution had shrunk to a 5-line Relations table and a 2-line "When to use / Not X" disambiguation block.

**2. Cross-host invocation flags are not portable.** The Inverted Invocation Policy used `disable-model-invocation: true` (mainstream types) and `user-invocable: false` (niche types) to stratify the `/` palette. Neither field is in the agentskills.io open standard:

- **Claude Code**: both fields supported (with known implementation quirks in issues #19141 and #26251).
- **Cursor**: has a confirmed bug where `disable-model-invocation: true` on plugin-delivered skills hides them from `/` entirely — acknowledged by Cursor support 2026-03-24, escalated, thread auto-closed 2026-04-17 without a fix. The `user-invocable` field is not documented in Cursor. Effect in Cursor: mainstream type skills were invisible; niche-type flag ignored.
- **Codex (OpenAI)**: neither field supported in SKILL.md. Invocation control lives in a separate `agents/openai.yaml` with `policy.allow_implicit_invocation`. Effect: all type skills auto-invocable in Codex, reverting the inversion.
- **Qoder, Kiro**: neither field supported.

The carefully tiered invocation policy worked only in Claude Code. Cross-host parity — a stated architectural goal per `multi-host-plugin-architecture.adr.md` — was silently broken.

**3. Cognitive load.** At inversion time the Claude Code `/` palette had 26 visible entries (9 intent + 6 track + 10 mainstream type + 1 utility). Users scanning `/archcore:` saw a long list dominated by internal-taxonomy names (ADR, RFC, StRS, SRS, CPAT, MRD, BRD, …) rather than intent-based actions.

### What was reverified before this decision

- Every Archcore document type is reachable through at least one existing intent or track skill, with the exception of `rfc` (covered in `/archcore:decide` only as a redirect) and `cpat` (no track or intent covered it). These two gaps had concrete absorption targets.
- `mcp__archcore__create_document(type=<any>)` accepts every Archcore document type with or without content. No skill is required to create any document.
- Deleting type skills has no functional regression for adr/rule/guide/doc/spec/prd/idea/task-type/niche types — their elicitation already lives in intent/track skills.

## Decision

**Delete all 17 document-type skills. Collapse Layer 3. Keep only intent (11), track (6), and utility (1) skills — 18 total on disk, all visible in `/`.**

Concrete migrations to close the two gaps:

1. **RFC absorbed into `/archcore:decide`.** The `decide` skill gains a branch: if the user's language is "proposing", "should we", "thinking about", or explicitly mentions an RFC, confirm "Draft an RFC for team review?" and run the RFC creation recipe (Summary, Motivation, Detailed Design, Drawbacks, Alternatives). The finalized-decision branch (ADR + optional rule+guide continuation) remains unchanged. `decide`'s description and routing table are broadened to enumerate both paths.

2. **CPAT absorbed into `/archcore:standard-track`.** A new optional Step 3b is inserted between ADR creation and rule creation: "If the decision represents a code-pattern change (before/after), offer a CPAT. Ask 'What pattern changed? Show before/after.' Compose What Changed / Why / Before / After / Scope. Relations: cpat `implements` adr; rule `related` cpat if cpat was created." The flow becomes adr → (optional cpat) → rule → guide.

3. **17 type-skill directories deleted**: `skills/{adr,rfc,rule,guide,doc,spec,prd,idea,task-type,cpat,mrd,brd,urd,brs,strs,syrs,srs}/`.

4. **Count invariants updated** in `README.md` (35 → 18), `test/structure/skills.bats` (`>= 32` → `>= 18`), and every `.archcore/plugin/` document referencing skill counts.

5. **Obsolete lifecycle docs deleted** (user-selected "Delete entirely"): `adding-document-type-skill.guide.md`, `creating-skill-batch.task-type.md`, `keep-document-type-skills.adr.md`.

6. **Per-class invocation flags become simpler**: intent and track skills remain auto-invocable with no flag; utility (`verify`) keeps `disable-model-invocation: true`. No skill uses `user-invocable: false`. This removes reliance on the field that Cursor and Codex do not support portably.

## Alternatives Considered

### Keep type skills, hide mainstream types from `/` via `user-invocable: false`

Considered first. Would reduce Claude-Code `/` palette from 26 to 16 visible entries while preserving type skills on disk for model orchestration. **Rejected** for two reasons: (a) it is a Claude-Code-only fix — Cursor ignores the field, Codex doesn't support it, and the existing Cursor `disable-model-invocation` bug already breaks the mainstream-type tier in that host; (b) it does not address the underlying content duplication between intents/tracks and type skills, which was the deeper quality problem.

### Move per-type knowledge into MCP (rich `create_document` schema or `get_type_schema` tool)

Strategically the cleanest endpoint — MCP is the only host-agnostic layer, and moving elicitation there would give Cursor/Codex the same authoring quality without duplicating skills. **Deferred**, not adopted in this change. Reasons: (a) requires CLI/MCP-server changes, out of scope for a plugin-only change; (b) intent/track skills already inline the elicitation, so there is no immediate regression from deletion; (c) the MCP route can be adopted later as a Phase 2 without blocking this cleanup.

### Keep status quo (post-inversion)

**Rejected.** Content duplication persists. Cross-host parity remains broken. Every subsequent type-skill edit would need to be mirrored in intent and track skills — the spec's "Intent skills must not duplicate content from type skills" invariant was already aspirational and violated in practice.

### Kill type skills without absorbing RFC and CPAT

**Rejected.** Leaving RFC and CPAT unreachable through any intent/track skill would degrade the creation path for two types. Absorbing both (RFC into `decide`, CPAT into `standard-track`) closes those gaps with small additions to existing skills.

### Keep niche type skills (brs, strs, syrs, srs, mrd, brd, urd), delete only mainstream

**Rejected.** Niche types are already fully inlined inside `iso-track` and `sources-track` — those track skills contain the question, section list, `create_document`, and `add_relation` calls per niche type. The niche SKILL.md files were kept historically so tracks could "programmatically invoke" them, but the tracks never did — they inline the flow directly. Keeping them added no value and prevented the palette-unification story.

## Consequences

### Positive

- Visible `/` palette drops from 26 to **18** (11 intent + 6 track + 1 utility). All remaining skills use only the portable invocation flag (`disable-model-invocation: true` for utility, none for intent/track).
- Multi-host parity is restored: no reliance on the Claude-Code-specific `user-invocable: false` field. Cursor and Codex now see the same 18 skills that Claude Code does, with identical invocation semantics (auto for intent/track, user-only for utility).
- Content duplication is eliminated at the skill boundary. Intent and track skills are the single home for per-type elicitation within the plugin. The spec's previously-violated "no duplication" invariant is replaced by an explicit acknowledgement: inline recipes inside intent/track skills are not duplication, they are each entry point's self-containment.
- Session-start token budget is freed: the model no longer loads 17 type-skill descriptions on every session.
- Every Archcore document type remains creatable: via intent (adr/rule/guide/doc/spec/prd/idea), via track (niche types, task-type, plan, cpat inside standard-track, rfc inside decide), or directly via `mcp__archcore__create_document(type=<any>)`.

### Negative

- Loss of the `/archcore:<type> <topic>` power-user shortcut (e.g., `/archcore:adr`). Users who want to create a specific type now go through the matching intent or call MCP directly. The documented migration path in `/archcore:help` covers both options.
- Teaching role that type skills served ("what is an ADR in Archcore?") moves to the Archcore CLI documentation outside this plugin. README references Archcore documentation for per-type reference.
- Skill count stability: future changes to the 17 inlined recipes have no per-type skill to edit — they must be edited inside each intent/track that uses them. This is the duplication cost accepted in exchange for self-containment. When this cost becomes heavy, the MCP-schema alternative (deferred above) becomes attractive.

### Supersedes

- `keep-document-type-skills.adr.md` — the ADR that justified keeping the type-skill layer. Deleted in this change; reasoning preserved in this ADR's Context section.
- Type-skill portion of `inverted-invocation-policy.adr.md`. The intent/track/utility policy from that ADR remains in force; the mainstream/niche type rows are marked historical.
- `Layer 3` in `intent-based-skill-architecture.adr.md`. The 4-layer decomposition remains as a historical framing; the effective runtime layering is now 3 (intent, track, utility) + MCP primitives.

### Constraints (new)

- Intent skills MUST inline per-type elicitation for every document type they create.
- Track skills MUST inline per-type elicitation for every step.
- No new per-type SKILL.md MUST be added. New types are added by extending the matching intent or track, not by creating a new skill.
- `/archcore:help` MUST document direct-MCP access for any document type (as the fallback path for types not covered by a user-facing intent/track).
