---
title: "Skill Descriptions Audit"
date: "2026-04-14"
auditor: "Claude Sonnet 4.6 (prompt-engineer persona)"
scope: "skills/*/SKILL.md — description field only"
---

# Skill Descriptions Audit

## 1. Best-Practices Rubric Applied

### Sources

- **Claude Code Skills documentation** — https://docs.anthropic.com/en/docs/claude-code/skills
  Key findings:
  - The `description` field is the primary signal Claude uses to decide whether to load a skill automatically.
  - Official example pattern: `"Explains code with visual diagrams and analogies. Use when explaining how code works, teaching about a codebase, or when the user asks 'how does this work?'"` — note the "Use when…" clause appended.
  - Troubleshooting guidance explicitly says: _"Check the description includes keywords users would naturally say."_
  - Character budget constraint: the combined `description + when_to_use` text is **capped at 1,536 characters per skill**; descriptions are front-loaded (truncated from the back when budget is tight). Lead with the most essential signal.
  - `disable-model-invocation: true` prevents automatic activation; for those skills the description still appears in the slash-command picker.
  - Official frontmatter contract confirms `description` is string, no structural sub-fields.

- **Project-internal spec** — `.archcore/plugin/skills-system.spec.md`
  Normative rules extracted:
  - Track skill descriptions MUST be prefixed with `"Advanced — "`.
  - Type skill descriptions MUST be prefixed with `"Expert — "` **except** `adr`, `prd`, `rule`, `guide`, `plan` (high-frequency; clean descriptions).
  - Intent skills (Layer 1) have no required prefix.
  - Model-invocation matching applies only to type skills (Layer 3). Intent and track skills have `disable-model-invocation: true`, so their descriptions appear only in the slash-command picker UI.

### Rubric Criteria (scored per skill)

| #   | Criterion                                                            | Weight    |
| --- | -------------------------------------------------------------------- | --------- |
| P   | Correct tier prefix (`Advanced —` / `Expert —` / none)               | Must-pass |
| K   | Discoverability keywords — contains words users would naturally type | High      |
| T   | Trigger clarity — tells Claude/user _when_ to activate               | High      |
| D   | Differentiation — distinguishes from overlapping skills              | Medium    |
| L   | Length/density — fits in budget, front-loaded, not truncated-risk    | Medium    |
| C   | Consistency — follows the same structural pattern as peers           | Low       |

Verdicts: **GOOD** = passes all; **NEEDS WORK** = 1-2 issues, fixable in-place; **BROKEN** = fails must-pass or is actively misleading.

---

## 2. Per-Skill Audit

### Layer 1 — Intent Skills (8)

These have `disable-model-invocation: true`. Their description appears in the slash-command picker. Model matching does not apply, so keyword density matters less; clarity and differentiation matter most.

---

#### `capture`

> `Capture documentation for a module, component, or topic — routes to the right document type automatically.`

**Verdict: NEEDS WORK**

Issues:

- K: The word "Capture" mirrors the skill name — adds no signal. Users would say "document", "write docs for", "create reference for", not "capture documentation".
- T: No "when to use" clause in the description itself. The picker user sees only this one line.
- D: Does not differentiate from `/archcore:decide` (also creates docs) or from direct type skills like `/archcore:doc`.
- C: Peers like `review` and `actualize` state what the skill _does_; `capture` says what input it accepts, not what it produces.

**Rewrite:**

```
Document a module, component, or system — automatically picks the right type (ADR, spec, doc, or guide). Use when you need comprehensive docs for a codebase element and don't want to choose the document type yourself.
```

---

#### `plan`

> `Plan a feature or initiative end-to-end with requirements and traceability.`

**Verdict: NEEDS WORK**

Issues:

- T: "end-to-end with requirements and traceability" is accurate but jargon-heavy; users rarely phrase requests that way.
- D: Does not differentiate from `/archcore:product-track` (which also produces idea→prd→plan) or from the `plan` type skill (which creates a single plan document).
- K: Missing natural-language trigger phrases ("let's plan", "create a plan for", "I need to plan").

**Rewrite:**

```
Plan a feature or initiative — creates a requirements chain (idea → PRD → plan) for large scope, or a single plan document for focused work. Use when someone says "let's plan", "create a roadmap for", or "I need to plan X". Not for recording a decision — use /archcore:decide.
```

---

#### `decide`

> `Record an architectural or technical decision with context and alternatives.`

**Verdict: GOOD**

Passes all criteria. Clear action verb ("Record"), domain keywords ("architectural", "technical decision", "context", "alternatives"), implicit when-to-use. Length is lean. One minor note: does not mention it can also produce rule+guide follow-up, but that's body content, not description scope.

---

#### `standard`

> `Establish a team standard — creates decision, rule, and how-to guide in sequence.`

**Verdict: GOOD**

Solid. The dash-separated clause pattern ("Establish a team standard — creates…") is the clearest pattern in the intent skill set. Keywords: "team standard", "rule", "guide". Flow is explicit. No differentiation gap because `standard` is the only skill that produces the three-document chain.

---

#### `review`

> `Review documentation for gaps, staleness, orphaned documents, and missing relations.`

**Verdict: NEEDS WORK**

Issues:

- K: "Review documentation" is what users would say but also what they say for `/archcore:status` or just asking Claude to "check the docs". The description does not help differentiate.
- D: Does not distinguish from `status` (which is fast counts) or `actualize` (which is code-drift detection).
- T: Missing "Use when" clause.

**Rewrite:**

```
Audit documentation health — finds coverage gaps, stale statuses, orphaned documents, and missing relations. Use when you want a full health report with recommendations. For quick counts use /archcore:status; for code-drift detection use /archcore:actualize.
```

---

#### `status`

> `Show Archcore documentation dashboard — document counts, relation stats, and potential issues.`

**Verdict: GOOD**

Clear, differentiated (explicitly says "counts", "stats" — quick read vs deep review), adequate keywords. No missing elements.

---

#### `actualize`

> `Detect stale documentation and suggest updates based on code changes and relation graph.`

**Verdict: NEEDS WORK**

Issues:

- K: Users are more likely to say "are docs up to date?", "docs out of sync?", "what changed since the refactor?" — none of these keywords appear.
- T: No "Use when" phrase. The skill body says "Session start showed a staleness warning" as a trigger — that is valuable signal that belongs in the description.
- D: Does not distinguish from `review` (which also reports on stale statuses).

**Rewrite:**

```
Detect stale docs and suggest updates — cross-references code changes with documentation, checks the relation graph for cascade staleness. Use when docs may be out of date after a refactor, merge, or when a session-start staleness warning appeared. For coverage gaps use /archcore:review.
```

---

#### `help`

> `Guide to Archcore commands and capabilities.`

**Verdict: NEEDS WORK**

Issues:

- K: "Guide" is very generic. Users say "what can I do?", "list commands", "how do I use archcore?", "onboarding" — none present.
- T: No "Use when" signal.
- L: At 38 characters this is the shortest description in the set. Given it's user-only and appears in the picker, it needs to be more informative.
- C: All other intent skills have 70-120 character descriptions; `help` at 38 chars is an outlier.

**Rewrite:**

```
Show available Archcore commands and how to use them. Use when onboarding, exploring what skills are available, or when you're not sure which command to run.
```

---

### Layer 2 — Track Skills (6)

All have `disable-model-invocation: true` and the `"Advanced — "` prefix (per spec). The cross-cutting issue for this group: every description is a bare document-list after "Advanced —", with no "when to use" clause and no differentiation signal.

---

#### `product-track`

> `Advanced — Create idea, PRD, and plan with full traceability.`

**Verdict: NEEDS WORK**

Issues:

- T: No "when to use" clause. The body says "Lightweight requirements flow. Best for individual features, small teams, rapid prototyping." — this is exactly the differentiating signal that belongs in the description.
- D: Overlaps visually with `feature-track` (which also creates a PRD). User cannot distinguish them from the picker alone.
- K: "with full traceability" is filler — adds no discoverability signal.

**Rewrite:**

```
Advanced — Lightweight product requirements flow: idea → PRD → plan. Best for individual features, small teams, or rapid prototyping. For engineer-led feature delivery use /archcore:feature-track; for full ISO requirements cascade use /archcore:iso-track.
```

---

#### `sources-track`

> `Advanced — Create MRD, BRD, URD discovery documents.`

**Verdict: NEEDS WORK**

Issues:

- T: No "when to use" context. Body says "Best for product teams doing research, stakeholder alignment, business analysis." — critical signal absent from description.
- K: Acronyms MRD/BRD/URD are not self-explanatory to a new user.
- D: No differentiation from `iso-track` (which also produces requirements artifacts).

**Rewrite:**

```
Advanced — Discovery requirements flow: MRD (market) → BRD (business) → URD (user). Best for product teams doing research, stakeholder alignment, or business analysis before committing to a product. Not for technical requirements — use /archcore:iso-track.
```

---

#### `iso-track`

> `Advanced — Create ISO 29148 requirements cascade (BRS → StRS → SyRS → SRS).`

**Verdict: NEEDS WORK**

Issues:

- T: No "when to use" context. Body says "Best for regulated systems, multi-team projects, complex distributed systems." — essential differentiator absent.
- K: "ISO 29148" is good for users who know the standard; users who don't won't match it.
- D: Users don't know when to choose this vs `sources-track`.

**Rewrite:**

```
Advanced — Formal ISO 29148 requirements cascade: BRS → StRS → SyRS → SRS. Best for regulated systems, multi-team projects, or complex distributed systems requiring traceable, auditable requirements. For lighter product discovery use /archcore:sources-track.
```

---

#### `architecture-track`

> `Advanced — Create ADR, spec, and plan for architectural design.`

**Verdict: NEEDS WORK**

Issues:

- T: No "when to use" context. Body says "Best for significant technical decisions that need formal specification and an implementation plan." — absent.
- D: Overlaps with `decide` intent skill (also creates ADR) and `standard-track` (also creates ADR).

**Rewrite:**

```
Advanced — End-to-end architectural design flow: ADR → spec → plan. Best for significant technical decisions that require a formal specification and an implementation plan. For a decision without a follow-up spec, use /archcore:decide. For codifying standards, use /archcore:standard-track.
```

---

#### `standard-track`

> `Advanced — Create ADR, rule, and guide to codify a standard.`

**Verdict: GOOD**

Passes all criteria for a track skill. The flow is clear (ADR → rule → guide), the purpose is expressed ("codify a standard"), and it meaningfully differs from `architecture-track` (no spec/plan). The only minor gap is no "when to use" clause, but this is the most self-explanatory track skill and the description is concise.

---

#### `feature-track`

> `Advanced — Create PRD, spec, plan, and task-type for feature lifecycle.`

**Verdict: NEEDS WORK**

Issues:

- T: No "when to use" clause. Body says "Best for well-scoped features that need formal specification and a recurring delivery pattern." — absent.
- D: Overlaps with `product-track` from a user perspective (both produce PRD). The key differentiator is the `spec + task-type` tail, which is missing from the description's framing.
- K: "feature lifecycle" is vague.

**Rewrite:**

```
Advanced — Full feature delivery flow: PRD → spec → plan → task-type. Best for well-scoped features that need formal specification and a repeatable delivery pattern. For lightweight product planning, use /archcore:product-track.
```

---

### Layer 3 — Type Skills (18)

Type skills are model-invoked. The description is the primary matching signal. The spec mandates `"Expert — "` prefix except for `adr`, `prd`, `rule`, `guide`, `plan`.

---

#### `adr`

> `Records architectural decisions with context, alternatives, and consequences. Activates for finalized technical decisions, technology choices, or trade-off discussions.`

**Verdict: GOOD**

Excellent. Two-sentence pattern: what it is, then explicit model-invocation triggers ("Activates for…"). Keywords: "architectural decisions", "finalized", "technology choices", "trade-off discussions". No `Expert —` prefix per spec (high-frequency exception). Well-differentiated.

---

#### `rfc`

> `Expert — Proposes technical changes for team review before a decision is made.`

**Verdict: NEEDS WORK**

Issues:

- K: Missing activation triggers. When does the model auto-activate this? Keywords like "proposal", "design review", "team feedback", "before we decide", "draft for review" are absent.
- T: No "Activates when…" pattern.
- C: Unlike `adr` and `rule`, this skill lacks the explicit model-invocation signal clause. The body's "When to use" section has exactly the right content that should be reflected here.

**Rewrite:**

```
Expert — Proposes a significant technical change for team review before a decision is finalized. Activates when proposing a design, exploring a change that needs buy-in, or when the user says "draft an RFC", "design proposal", or "let's get feedback before deciding".
```

---

#### `rule`

> `Defines mandatory team standards and required behaviors with rationale and examples. Activates for coding conventions, enforceable practices, or standards codified from decisions.`

**Verdict: GOOD**

Follows the same high-quality two-sentence pattern as `adr`. Keywords: "mandatory", "coding conventions", "enforceable practices", "standards codified from decisions". Clear activation signal. Passes all criteria.

---

#### `guide`

> `Provides step-by-step instructions for completing a specific task with prerequisites and verification. Activates for how-to procedures, setup instructions, or runbooks.`

**Verdict: GOOD**

Strong. "Activates for how-to procedures, setup instructions, or runbooks" covers the natural-language triggers well. Passes all criteria.

---

#### `doc`

> `Expert — Creates reference material: registries, glossaries, lookup tables.`

**Verdict: NEEDS WORK**

Issues:

- T: No activation trigger clause.
- K: The three examples (registries, glossaries, lookup tables) are good but incomplete. Missing: API catalogues, component inventories, terminology documents. Users may say "create a glossary for X" or "document our API list" — neither phrase is guaranteed to match without a broader keywords clause.
- C: Inconsistent with `adr`/`rule`/`guide` which all have "Activates for…" clause.

**Rewrite:**

```
Expert — Creates reference material: glossaries, registries, API catalogues, or lookup tables. Activates when creating reference content someone would look up, not follow or enforce — e.g., "create a glossary", "document our service list", "catalog the APIs".
```

---

#### `spec`

> `Expert — Defines a normative technical contract for a system or interface.`

**Verdict: NEEDS WORK**

Issues:

- K: "Normative technical contract" is precise but not how users talk. They say "write a spec for", "define the interface for", "API spec", "behavioral spec", "define what this system does".
- T: No activation clause.
- C: Inconsistent with `adr`/`rule`/`guide` pattern.

**Rewrite:**

```
Expert — Defines a normative technical contract for a system, API, or interface. Activates when specifying behavioral guarantees, API contracts, or interface protocols — e.g., "write a spec for", "define the interface", "document what this component must do".
```

---

#### `prd`

> `Defines product requirements with vision, goals, and success metrics. Activates for product scoping, feature definitions, or when establishing what to build and why.`

**Verdict: GOOD**

Follows the two-sentence model-invocation pattern. Keywords: "product requirements", "vision", "goals", "success metrics", "what to build and why". Activation clause is strong. No `Expert —` prefix per spec. Passes all criteria.

---

#### `idea`

> `Expert — Captures product or technical concepts worth exploring before commitment. Activates for brainstorming, "what if" discussions, or early-stage concept exploration.`

**Verdict: GOOD**

Strong. The quoted phrase `"what if"` is a high-signal natural-language trigger. "Before commitment" differentiates from `prd` (which is post-commitment). "Early-stage concept exploration" matches user vocabulary. Passes all criteria.

---

#### `mrd`

> `Expert — Documents market analysis, competitive positioning, TAM/SAM/SOM.`

**Verdict: NEEDS WORK**

Issues:

- T: No activation trigger.
- K: "TAM/SAM/SOM" is a niche acronym cluster. A user is more likely to say "market research", "competitive analysis", "market sizing document", "market opportunity".
- C: Inconsistent with peers that have "Activates for…".

**Rewrite:**

```
Expert — Documents market analysis, competitive positioning, and market sizing (TAM/SAM/SOM). Activates when analyzing a market before defining a product, writing a competitive landscape, or when user says "market research", "competitive analysis", or "market opportunity document".
```

---

#### `brd`

> `Expert — Documents business requirements, objectives, and ROI.`

**Verdict: NEEDS WORK**

Issues:

- T: No activation trigger.
- K: Very short. Missing triggers: "business case", "justify this project", "business justification", "business objectives", "ROI analysis".
- L: At 53 characters after the prefix this is among the shortest type descriptions. More activation signal needed.

**Rewrite:**

```
Expert — Documents business requirements, objectives, and ROI to justify a project. Activates when making a business case, writing business objectives, or documenting ROI expectations — e.g., "business case for X", "justify this project", "business requirements document".
```

---

#### `urd`

> `Expert — Documents user requirements, personas, and journeys.`

**Verdict: NEEDS WORK**

Issues:

- T: No activation trigger.
- K: Too brief. Natural triggers: "user research", "user personas", "who are our users?", "user journeys", "usability requirements".
- C: Inconsistent with better peers.

**Rewrite:**

```
Expert — Documents user requirements, personas, and journeys. Activates when defining who the users are and what they need — e.g., "user research", "define our personas", "document user journeys", "usability requirements".
```

---

#### `brs`

> `Expert — Formalizes business requirements per ISO 29148.`

**Verdict: NEEDS WORK**

Issues:

- T: No activation trigger.
- K: "Formalizes" and "ISO 29148" are precise but insufficient. Missing: when to use this vs `brd`, what "traceability" means in context.
- D: Does not distinguish from `brd` (which is an informal version of the same concept).

**Rewrite:**

```
Expert — Formalizes business requirements into a traceable specification per ISO 29148. Activates when converting informal business requirements (BRD) into structured, auditable specs, or when starting an ISO 29148 requirements cascade. Use /archcore:brd for informal business cases.
```

---

#### `strs`

> `Expert — Formalizes stakeholder requirements per ISO 29148.`

**Verdict: NEEDS WORK**

Issues:

- T: No activation trigger.
- K: Same pattern problem as `brs`. Missing: "stakeholder analysis", "decompose by stakeholder group", "multiple stakeholders".
- D: Users may not understand what "stakeholder requirements" means vs "user requirements" (urd) without a differentiating clause.

**Rewrite:**

```
Expert — Formalizes stakeholder requirements per ISO 29148 by decomposing BRS into stakeholder-specific specs. Activates when multiple stakeholder groups have different requirement sets, or when progressing through an ISO 29148 cascade after BRS. Use /archcore:urd for informal user needs.
```

---

#### `syrs`

> `Expert — Formalizes system requirements per ISO 29148.`

**Verdict: NEEDS WORK**

Same pattern problem as `brs`/`strs`.

Issues:

- T/K: No activation trigger. Missing: "system boundaries", "system interface", "operational modes", "translating stakeholder reqs to system level".
- D: Users may confuse with `srs` (software) or `spec` (technical contract).

**Rewrite:**

```
Expert — Formalizes system requirements per ISO 29148 by translating StRS into system-level specifications covering boundaries, interfaces, and operational modes. Activates after completing StRS in an ISO 29148 cascade. Not for software-specific requirements — use /archcore:srs.
```

---

#### `srs`

> `Expert — Formalizes software requirements per ISO 29148.`

**Verdict: NEEDS WORK**

Same pattern problem as the other ISO tier skills.

Issues:

- T/K: No activation trigger. Missing: "functional requirements", "non-functional requirements", "software spec", "detailed software requirements".
- D: Users may confuse with `spec` (technical contract for an existing component).

**Rewrite:**

```
Expert — Formalizes software requirements per ISO 29148 — detailed functional and non-functional requirements for a software system. Activates after SyRS in an ISO 29148 cascade, or standalone when formal software requirements specification is needed. For a technical API contract, use /archcore:spec.
```

---

#### `task-type`

> `Expert — Documents a recurring task pattern with steps and pitfalls.`

**Verdict: NEEDS WORK**

Issues:

- T: No activation trigger.
- K: "Recurring task pattern" is accurate but not how users talk. They say "we do this every sprint", "standard procedure for", "playbook for", "document our release process", "tribal knowledge for X".
- D: Does not distinguish from `guide` (single procedure) or `plan` (one-off implementation).

**Rewrite:**

```
Expert — Documents a recurring task pattern with steps, variations, and known pitfalls. Activates when a task is performed repeatedly (e.g., "document our release process", "playbook for X", "we always do this the same way"). For one-time procedures use /archcore:guide; for one-off plans use /archcore:plan.
```

---

#### `cpat`

> `Expert — Documents a code pattern change with before/after examples.`

**Verdict: NEEDS WORK**

Issues:

- T: No activation trigger.
- K: Natural triggers absent: "we're switching from X to Y", "refactoring pattern", "code migration", "before/after", "we changed how we do X".
- D: Does not differentiate from `adr` (deciding to change) or `rule` (defining the new standard).

**Rewrite:**

```
Expert — Documents a code pattern change with before/after examples and migration scope. Activates when a coding pattern has changed and the team needs a reference — e.g., "we switched from X to Y", "document this refactor pattern", "before/after code change". For deciding to change, use /archcore:adr; for defining the new standard, use /archcore:rule.
```

---

### Utility Skill (1)

#### `verify`

> `Run plugin integrity checks — validates configs, scripts, skills, hooks, agents, and runs test suite.`

**Verdict: GOOD**

Clear, complete enumeration of what it validates. Appropriate for its role (developer/CI tool). Has `disable-model-invocation: true`. No "when to use" clause needed given its specific technical scope.

---

## 3. Cross-Cutting Issues

### Issue A — Track skills have no "when to use" signal (affects all 5 with NEEDS WORK verdict)

Every track-skill description is `"Advanced — Create X, Y, Z."` with no context about _when_ the user should pick this track vs the intent skills or peer tracks. The body's intro paragraph (e.g., "Best for regulated systems…") never surfaces in the picker. A user looking at six track skills in a picker cannot distinguish between `product-track` and `feature-track` without reading the body.

**Fix pattern:** Append a "Best for…" clause and one "Not for…" pointer to each track description.

---

### Issue B — ISO tier skills (brs/strs/syrs/srs) have no activation triggers (affects 4 skills)

All four ISO-tier type skills follow the template `"Expert — Formalizes X requirements per ISO 29148."` with zero activation keywords. Because these skills are model-invoked, Claude must match the description to the conversation context. Without concrete trigger phrases, the matching is unreliable — Claude may never auto-activate these skills unless the user explicitly names the document type.

**Fix pattern:** Add "Activates when…" + 2-3 natural phrases to each.

---

### Issue C — `rfc`, `doc`, `spec`, `mrd`, `brd`, `urd` are missing the "Activates for…" pattern used by `adr`, `rule`, `guide`, `prd`, `idea`

The five high-frequency type skills (`adr`, `prd`, `rule`, `guide`, `idea`) all follow a two-sentence pattern:

1. What the type is.
2. "Activates for…" or "Activates when…" with natural-language triggers.

The remaining type skills omit sentence 2, making their model-invocation matching weaker. This is a consistent gap across 6 skills.

---

### Issue D — Differentiation between `capture` (intent) and individual type skills is absent

A user who sees both `/archcore:capture` and `/archcore:doc`, `/archcore:spec`, etc. in the picker has no signal from the descriptions that `capture` is the "don't-know-which-type" router. The descriptions make them appear parallel. `capture`'s description should explicitly say it is the "automatic routing" path.

---

### Issue E — `plan` intent vs `feature-track` vs `product-track` overlap is unaddressed in descriptions

Three skills produce plans and/or PRDs:

- `/archcore:plan` — intent, routes to single plan or product-track flow
- `/archcore:product-track` — track, idea → PRD → plan
- `/archcore:feature-track` — track, PRD → spec → plan → task-type

None of the three descriptions cross-reference the others. A user cannot determine from the picker alone which to use.

---

### Issue F — `help` description (38 chars) is the shortest in the set by a factor of 2x

The `help` skill is specifically for onboarding and discovery — precisely the scenario where a richer description would help. Its current description is the least informative.

---

## 4. Priority-Ordered Fix List

| Priority | Skill                                                                                             | Verdict    | Effort                                                              |
| -------- | ------------------------------------------------------------------------------------------------- | ---------- | ------------------------------------------------------------------- |
| P1       | `brs`, `strs`, `syrs`, `srs`                                                                      | NEEDS WORK | Low — add "Activates when…" clause to each                          |
| P1       | `rfc`, `doc`, `spec`                                                                              | NEEDS WORK | Low — add "Activates for…" clause to each                           |
| P2       | `capture`                                                                                         | NEEDS WORK | Low — reframe from "capture" to "document", add routing intent      |
| P2       | `help`                                                                                            | NEEDS WORK | Low — expand from 38 chars to ~120 chars                            |
| P2       | `review`                                                                                          | NEEDS WORK | Low — add differentiation vs `status` and `actualize`               |
| P2       | `actualize`                                                                                       | NEEDS WORK | Low — add natural-language triggers and differentiation vs `review` |
| P3       | All 5 NEEDS WORK track skills                                                                     | NEEDS WORK | Medium — add "Best for…" / "Not for…" clauses                       |
| P3       | `mrd`, `brd`, `urd`                                                                               | NEEDS WORK | Low — add "Activates when…" + natural phrases                       |
| P3       | `task-type`, `cpat`                                                                               | NEEDS WORK | Low — add "Activates when…" + natural phrases                       |
| P4       | `plan` intent                                                                                     | NEEDS WORK | Medium — add differentiator vs `product-track`/`feature-track`      |
| —        | `decide`, `standard`, `status`, `adr`, `rule`, `guide`, `prd`, `idea`, `standard-track`, `verify` | GOOD       | No change needed                                                    |

---

## 5. Ready-to-Use Rewrites (YAML-safe)

All values below are drop-in replacements for the `description:` line in the respective SKILL.md frontmatter. Quoted where the value contains a comma or special characters.

```yaml
# capture
description: "Document a module, component, or system — automatically picks the right type (ADR, spec, doc, or guide). Use when you need comprehensive docs for a codebase element and don't want to choose the document type yourself."

# plan
description: "Plan a feature or initiative — creates a requirements chain (idea → PRD → plan) for large scope, or a single plan document for focused work. Use when someone says 'let's plan', 'create a roadmap for', or 'I need to plan X'. Not for recording a decision."

# review
description: "Audit documentation health — finds coverage gaps, stale statuses, orphaned documents, and missing relations. Use when you want a full health report with recommendations. For quick counts use /archcore:status; for code-drift detection use /archcore:actualize."

# actualize
description: "Detect stale docs and suggest updates — cross-references code changes with documentation, checks the relation graph for cascade staleness. Use when docs may be out of date after a refactor, merge, or when a session-start staleness warning appeared. For coverage gaps use /archcore:review."

# help
description: "Show available Archcore commands and how to use them. Use when onboarding, exploring what skills are available, or when you're not sure which command to run."

# product-track
description: "Advanced — Lightweight product requirements flow: idea → PRD → plan. Best for individual features, small teams, or rapid prototyping. For engineer-led feature delivery use /archcore:feature-track; for ISO requirements cascade use /archcore:iso-track."

# sources-track
description: "Advanced — Discovery requirements flow: MRD (market) → BRD (business) → URD (user). Best for product teams doing research, stakeholder alignment, or business analysis before committing to a product. Not for technical requirements — use /archcore:iso-track."

# iso-track
description: "Advanced — Formal ISO 29148 requirements cascade: BRS → StRS → SyRS → SRS. Best for regulated systems, multi-team projects, or complex distributed systems requiring traceable, auditable requirements. For lighter product discovery use /archcore:sources-track."

# architecture-track
description: "Advanced — End-to-end architectural design flow: ADR → spec → plan. Best for significant technical decisions that require a formal specification and an implementation plan. For a decision without a spec, use /archcore:decide. For codifying standards, use /archcore:standard-track."

# feature-track
description: "Advanced — Full feature delivery flow: PRD → spec → plan → task-type. Best for well-scoped features that need formal specification and a repeatable delivery pattern. For lightweight product planning, use /archcore:product-track."

# rfc
description: "Expert — Proposes a significant technical change for team review before a decision is finalized. Activates when proposing a design, exploring a change that needs buy-in, or when the user says 'draft an RFC', 'design proposal', or 'let's get feedback before deciding'."

# doc
description: "Expert — Creates reference material: glossaries, registries, API catalogues, or lookup tables. Activates when creating reference content someone would look up, not follow or enforce — e.g., 'create a glossary', 'document our service list', 'catalog the APIs'."

# spec
description: "Expert — Defines a normative technical contract for a system, API, or interface. Activates when specifying behavioral guarantees, API contracts, or interface protocols — e.g., 'write a spec for', 'define the interface', 'document what this component must do'."

# mrd
description: "Expert — Documents market analysis, competitive positioning, and market sizing (TAM/SAM/SOM). Activates when analyzing a market before defining a product — e.g., 'market research', 'competitive analysis', 'market opportunity document'."

# brd
description: "Expert — Documents business requirements, objectives, and ROI to justify a project. Activates when making a business case or documenting ROI expectations — e.g., 'business case for X', 'justify this project', 'business requirements document'."

# urd
description: "Expert — Documents user requirements, personas, and journeys. Activates when defining who the users are and what they need — e.g., 'user research', 'define our personas', 'document user journeys', 'usability requirements'."

# brs
description: "Expert — Formalizes business requirements into a traceable specification per ISO 29148. Activates when converting informal business requirements into structured, auditable specs, or starting an ISO 29148 cascade. Use /archcore:brd for informal business cases."

# strs
description: "Expert — Formalizes stakeholder requirements per ISO 29148 by decomposing BRS into stakeholder-specific specs. Activates when multiple stakeholder groups have different requirement sets, or progressing through an ISO 29148 cascade after BRS. Use /archcore:urd for informal user needs."

# syrs
description: "Expert — Formalizes system requirements per ISO 29148 by translating StRS into system-level specifications covering boundaries, interfaces, and operational modes. Activates after completing StRS in an ISO 29148 cascade. Not for software-specific requirements — use /archcore:srs."

# srs
description: "Expert — Formalizes software requirements per ISO 29148 — detailed functional and non-functional requirements for a software system. Activates after SyRS in an ISO 29148 cascade, or standalone when formal software requirements specification is needed. For a technical API contract, use /archcore:spec."

# task-type
description: "Expert — Documents a recurring task pattern with steps, variations, and known pitfalls. Activates when a task is performed repeatedly — e.g., 'document our release process', 'playbook for X', 'we always do this the same way'. For one-time procedures use /archcore:guide."

# cpat
description: "Expert — Documents a code pattern change with before/after examples and migration scope. Activates when a coding pattern has changed — e.g., 'we switched from X to Y', 'document this refactor pattern', 'before/after code change'. For deciding to change, use /archcore:adr."
```

---

## 6. Summary Statistics

| Layer            | Total  | GOOD   | NEEDS WORK | BROKEN |
| ---------------- | ------ | ------ | ---------- | ------ |
| Intent (Layer 1) | 8      | 3      | 5          | 0      |
| Track (Layer 2)  | 6      | 1      | 5          | 0      |
| Type (Layer 3)   | 18     | 5      | 13         | 0      |
| Utility          | 1      | 1      | 0          | 0      |
| **Total**        | **33** | **10** | **23**     | **0**  |

No descriptions are categorically BROKEN (no wrong tier prefix, no actively misleading content). All 23 NEEDS WORK cases are fixable with targeted additions — primarily adding "Activates when…" / "Use when…" clauses and differentiation pointers.
