---
title: "Code-Oriented Intent Skill — /archcore:align"
status: rejected
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

**Status — rejected, superseded by shipped push and pull mechanisms.** As of plugin 0.3.0 the functionality proposed here is delivered by two complementary mechanisms that cover both modes of JTBD #1 without a dedicated `/archcore:align` intent. Pull mode shipped as `/archcore:context <path | topic>`, which classifies scope as path, topic, or pickup and returns grouped rules, ADRs, specs, and cpats for a code area — functionally what this idea proposed, under a more general name that also covers topic-level queries and session pickup. Push mode shipped as the `bin/check-code-alignment` pre-mutation hook, which auto-injects the same top-3 ranked documents on every source edit outside `.archcore/`.

Adding `/archcore:align` on top would duplicate surface area and introduce routing ambiguity between two pull intents, and the invocation policy explicitly prefers a minimal intent set. The reasoning about rankers and resolvers is preserved below, because it informed both shipped mechanisms. If a distinct push command is ever needed — a programmatic `archcore align <path>` for batch scripts or CI — it belongs in the CLI repository rather than as a plugin skill.

## Idea

Add an intent skill `/archcore:align` that takes a code area as its argument — a file path, a directory, a component name, or a feature scope — and returns the applicable constraints from the knowledge base: the ADRs governing the area, the rules that apply, the specs the code must conform to, and the cpats it should follow.

This is the user-pull counterpart to the automatic push mechanism: the hook is invisible and unavoidable, the skill is explicit and scoped. Both serve JTBD #1 — ensuring a code change respects existing architecture and decisions — at different moments in the workflow.

**Why a new intent rather than an extension.** `/archcore:capture` creates documents *about* code while this reads documents *for* code, which is the opposite direction. The audit intent operates on the document graph rather than on code paths. The freshness intent reports when documentation has fallen behind code; it is a reporter rather than a constraint resolver. `/archcore:align` fills the empty slot — the code-centric read — because every existing intent is document-centric.

**Routing.** A file path routes to a path-match resolver returning documents that reference it; a directory routes to a prefix-match resolver; a component name routes to a symbol search followed by path matching; a feature scope routes to a tag match plus content search; and no argument produces one clarifying question. Output ranks by specificity, then type priority `rule > adr > spec > cpat > guide`, then recency — matching the hook.

**Execution.** Parse the argument to determine scope type; query through `list_documents`, `list_relations`, and `get_document` for the matched documents; then rank and format the result under fixed headings — rules that apply, decisions that govern this area, specs the code must conform to, patterns to follow, and related items in the graph — each entry being a one-line summary rather than a body. Close with a follow-up offer to show full content or to start implementation with the constraints loaded.

**Frontmatter.** `name: align`, an argument hint naming the four accepted scopes, and a description whose triggers are phrases such as "what rules apply to X", "before I refactor Y", and "what should I know before touching payments", with anti-triggers pointing document creation at `capture` or `decide` and audits at the inspection intent. No invocation-restricting flag, because an intent skill must auto-invoke.

## Value

**It complements the hook without duplicating it.** The hook injects context at the moment of the write; the skill loads context deliberately, before the user even asks the agent to code. Two moments, two intents, one underlying index.

**It gives users an explicit pull lever.** Some users want to see the constraints before committing to a change — "I'm thinking about refactoring payments, what applies?" — and the hook answers that only once the agent starts writing, which is too late for exploratory thinking.

**It makes the knowledge base queryable by code location.** The graph is queryable today by document type, tag, and relation, but not by code path, and this fills the missing axis.

**It demos well.** The hero prompt becomes "What rules apply to my `src/api/handlers/`?", the agent returns the list, and the value is obvious. It works as a read-only operation without the hook, so the skill could precede the hook in rollout.

## Possible Implementation

**Phase 1 — the path-match resolver**, roughly 1–2 days: a new `SKILL.md` inside the standard conventions, executing through the existing MCP tools, resolving paths by grep for a first version, with the output format fixed by the headings above.

**Phase 2 — symbol and feature resolvers**, roughly 2–3 days: symbol search by grep across the codebase for the identifier followed by path matching on the hits, feature search by tag match plus content search, and a disambiguation question when several resolvers apply.

**Phase 3 — index integration**, roughly 1 day after the hook's own Phase 2: reuse that path index so one index serves two consumers, replacing grep with an index lookup for the path and prefix cases.

The specifications would need the intent count, the component inventory, and the invocation matrix updated alongside.

## Risks

- **Skill count creep.** Adding another intent requires revising the maximum-intent constraint explicitly rather than silently.
- **Duplicate surface with the hook.** If both ship, a user sees the same context twice — once from the skill and once from the hook at write time. The mitigation is session-level de-duplication in the hook.
- **Overlap with `capture`.** Users may confuse them, so the routing description must state plainly that one writes documents about code while the other reads documents for code. Trigger-phrase discipline is required.
- **Empty-result experience.** Where no document covers a code area, the skill must return something useful rather than looking broken on a greenfield area — for example, suggesting that the area be documented.
- **Routing precision.** "Before I refactor X" is ambiguous between loading constraints and planning the refactor, so the anti-trigger must say the skill is read-only and name the planning path.
- **Content truncation.** Returning full bodies explodes output length on a well-documented area, so entries stay one-line summaries with full content one `get_document` away.
- **Line limit.** An intent skill must not exceed 300 lines, and the routing plus execution logic for four resolvers fits inside that.
