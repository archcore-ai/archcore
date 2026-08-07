---
title: "Remove Document-Type Skills — Collapse Layer 3 into Intent and Track Skills"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

**Surface note.** The counts below describe the state at this decision — 18 skills on disk. `merge-review-status-remove-graph.adr` then took the palette to 16, and `skill-surface-collapse.adr` to 7, and `four-command-palette.adr` then collapsed the surface to the current four commands (`init`, `plan`, `document`, `review`) with a gated track layer. What survives is the rule this record established: per-type elicitation lives inline inside the skills that create documents, and no new per-type skill is added.

## Context

Three facts made the per-document-type skill layer — formerly Layer 3 of `intent-based-skill-architecture.adr` — obsolete. Every creation-oriented intent skill and every track skill already inlined the same per-type questions, section lists, and MCP calls that the type skills carried, leaving each type skill's unique contribution at a 5-line relations table and a 2-line disambiguation block. The cross-host invocation flags that the tiered palette depended on were not portable. And the palette had grown to 26 visible entries dominated by internal-taxonomy names — ADR, RFC, StRS, SRS, CPAT, MRD, BRD — rather than intent-based actions.

## Findings

1. **Content duplication.** The line "Ask: What was the decision? What alternatives were considered? Compose content covering Context, Decision, Alternatives Considered, Consequences" appeared verbatim in both `skills/adr/SKILL.md` and Step 3 of `skills/decide/SKILL.md`. The track skills contained the full per-type question-and-section flow for every document they created.
2. **Invocation flags are not portable.** Neither `disable-model-invocation` nor `user-invocable` is in the agentskills.io open standard. Claude Code supported both, with implementation quirks in issues #19141 and #26251. Cursor had a confirmed bug where `disable-model-invocation: true` on a plugin-delivered skill hid it from `/` entirely — acknowledged by Cursor support on 2026-03-24, escalated, and auto-closed on 2026-04-17 without a fix — and does not document `user-invocable`, so mainstream type skills were invisible there and the niche flag was ignored. Codex supported neither field in `SKILL.md`, keeping invocation control in a separate `agents/openai.yaml` under `policy.allow_implicit_invocation`, so every type skill stayed auto-invocable there and the inversion reverted. Qoder and Kiro supported neither. The tiered policy therefore worked only in Claude Code, and the cross-host parity goal of `multi-host-plugin-architecture.adr` was silently broken.
3. **Cognitive load.** At inversion time the Claude Code palette held 26 visible entries: 9 intent, 6 track, 10 mainstream type, and 1 utility.

Three things were reverified before deciding: every Archcore document type was reachable through at least one intent or track skill, except `rfc`, covered in `decide` only as a redirect, and `cpat`, covered nowhere; `mcp__archcore__create_document(type=<any>)` accepts every type with or without content, so no skill is required to create any document; and deleting the type skills carried no functional regression for the types whose elicitation already lived in an intent or track skill.

## Decision

Delete all 17 document-type skills and collapse Layer 3, keeping only intent, track, and utility skills — 18 on disk at the time, all visible in `/` — while absorbing the two uncovered types into existing skills.

`rfc` was absorbed into `decide`, which gained a branch: when the user's language is "proposing", "should we", or "thinking about", or explicitly names an RFC, the skill confirms "Draft an RFC for team review?" and runs the RFC recipe of Summary, Motivation, Detailed Design, Drawbacks, and Alternatives, leaving the finalized-decision branch unchanged. `cpat` was absorbed into `standard-track` as an optional Step 3b between ADR creation and rule creation, asking what pattern changed and composing What Changed, Why, Before, After, and Scope, with `cpat implements adr` and `rule related cpat`, so the flow became adr → optional cpat → rule → guide.

Alongside those absorptions: the 17 type-skill directories were deleted; the count invariants were updated in `README.md`, `@test/structure/skills.bats`, and every `.archcore/` document referencing skill counts; three obsolete lifecycle documents were deleted at the user's direction; and the per-class invocation flags simplified to no flag for intent and track skills and `disable-model-invocation: true` for the utility, removing all reliance on the field that Cursor and Codex do not support.

## Alternatives Considered

1. **Keep the type skills and hide mainstream types from `/` through `user-invocable: false`** — considered first, and rejected because it is a Claude-Code-only fix that Cursor ignores and Codex does not support, on top of the existing Cursor bug that already broke the mainstream tier there, and because it leaves the underlying content duplication — the deeper quality problem — untouched.
2. **Move per-type knowledge into MCP, through a rich `create_document` schema or a `get_type_schema` tool** — strategically the cleanest endpoint, since MCP is the only host-agnostic layer, and deferred rather than adopted because it requires CLI and MCP-server changes outside a plugin-only change, because intent and track skills already inline the elicitation so deletion causes no regression, and because the MCP route can follow later without blocking this cleanup.
3. **Keep the status quo after the inversion** — rejected because content duplication persists, cross-host parity stays broken, and every subsequent type-skill edit would need mirroring into the intent and track skills, so the "intent skills must not duplicate type-skill content" invariant would stay violated in practice.
4. **Delete the type skills without absorbing `rfc` and `cpat`** — rejected because it would leave two types unreachable through any skill, degrading their creation path; absorbing both closes the gaps with small additions to existing skills.
5. **Keep the niche type skills and delete only the mainstream ones** — rejected because the niche types were already fully inlined inside `iso-track` and `sources-track`, which carry the question, section list, `create_document`, and `add_relation` calls per type. The niche files were retained historically so tracks could invoke them programmatically, which the tracks never did.

## Consequences

- The visible palette dropped from 26 entries to 18, and every remaining skill used only the portable invocation flag.
- Multi-host parity was restored, with Cursor and Codex seeing the same skills as Claude Code under identical invocation semantics.
- Content duplication was eliminated at the skill boundary, and the previously violated no-duplication invariant was replaced by an explicit acknowledgement: an inline recipe inside an intent or track skill is not duplication but that entry point's self-containment.
- The session-start token budget freed the 17 type-skill descriptions.
- Every Archcore document type stayed creatable, through an intent, through a track, or directly through `mcp__archcore__create_document(type=<any>)`.
- Tradeoff: the `/archcore:<type> <topic>` power-user shortcut was lost. A user wanting a specific type now goes through the matching intent or calls MCP directly. Help is removed under v2: command descriptions plus CLI help MUST document direct-MCP access.
- Tradeoff: the teaching role the type skills served — explaining what an ADR is in Archcore — moved to the Archcore CLI documentation outside this plugin, which the README references.
- Tradeoff: a change to one of the inlined recipes has no per-type skill to edit and must be edited inside each intent or track that uses it. This is the duplication cost accepted in exchange for self-containment, and when it grows heavy the deferred MCP-schema alternative becomes attractive.
- This decision supersedes `keep-document-type-skills.adr`, deleted in this change with its reasoning preserved above; the type-skill portion of `inverted-invocation-policy.adr`, whose intent, track, and utility policy remains in force; and Layer 3 of `intent-based-skill-architecture.adr`, whose four-layer decomposition remains as historical framing.

## Constraints

1. An intent skill MUST inline the per-type elicitation for every document type it creates.
2. A track skill MUST inline the per-type elicitation for every step.
3. The author MUST NOT add a new per-type `SKILL.md`. A new type is added by extending the matching intent or track.
4. Help is removed under v2: command descriptions plus CLI help MUST document direct-MCP access for any document type, as the fallback for a type no user-facing skill covers.

## Superseded when

- The deferred MCP-schema alternative ships, moving per-type elicitation into the host-agnostic layer, which would remove constraints 1 and 2.
- The cost of editing an inlined recipe across several skills is measured to exceed the cost of maintaining a per-type surface, which would reopen alternative 5.