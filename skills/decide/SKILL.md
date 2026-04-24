---
name: decide
argument-hint: "[decision topic]"
description: "Record a decision — finalized (creates ADR, optionally rule + guide) or open proposal for team review (creates RFC). Activate when user says 'we decided', 'record this decision', 'draft an RFC', 'proposing X', 'should we switch to Y'. Do NOT activate for feature planning (use /archcore:plan), documenting existing code (use /archcore:capture), or making a full standard cascade (use /archcore:standard)."
---

# /archcore:decide

Record a decision or a proposal for one. Routes between:

- **Finalized decision** → ADR (optionally followed by rule + guide as a standard)
- **Open proposal** → RFC (for team review before a decision is made)

## When to use

- "Record the decision to use PostgreSQL"
- "We decided to go with microservices"
- "Document why we chose JWT over sessions"
- "Draft an RFC for switching from REST to gRPC"
- "Proposing we adopt feature flags"
- "Should we move to Kubernetes?" (open → RFC)

**Not decide:**

- Planning a feature → `/archcore:plan`
- Making a full standard cascade (ADR → rule → guide) → `/archcore:standard`
- Documenting a component → `/archcore:capture`
- Reading applicable rules/ADRs/specs before coding → `/archcore:context`
- Picking up where work left off → `/archcore:context`

## Routing table

| Signal | Route | Documents |
|---|---|---|
| User describes a **finalized decision** (default) | → adr | Single ADR |
| User describes an **open proposal** ("thinking about", "should we", "proposing") | → rfc | Single RFC |
| User says "and make it a standard" or implies enforcement | → adr + standard-track continuation | ADR, then offer rule + guide |

Default for finalized decisions: create a single ADR. After creation, offer: "Want to codify this into a team standard? (rule + guide)".

## Execution

### Step 1: Check existing

`mcp__archcore__list_documents(types=["adr", "rfc"])` — check for existing decisions or proposals on this topic.

### Step 2: Route

If user language suggests the decision is still open ("thinking about", "should we", "proposing", "design proposal"), confirm with the user: "This sounds like an open proposal — draft an RFC for team review?" If yes, proceed to Step 3b. Otherwise continue with Step 3 (ADR).

### Step 3: Create ADR (finalized decision path)

- Ask: "What was the decision? What alternatives were considered?"
- Compose content covering Context, Decision, Alternatives Considered, Consequences.
- `mcp__archcore__create_document(type="adr")`

Then continue to Step 4.

### Step 3b: Create RFC (open proposal path)

- Ask: "What change are you proposing? What problem does it solve?"
- Compose content covering Summary, Motivation, Detailed Design, Drawbacks, Alternatives.
- `mcp__archcore__create_document(type="rfc")`
- Suggest relations: rfc `extends` existing ADR (if revising a past decision), or rfc `related` idea (if an idea inspired it).

RFC flow ends here — no rule + guide continuation (those belong to finalized decisions).

### Step 4: Relate (ADR path)

`mcp__archcore__add_relation` — link the ADR to existing RFCs, specs, plans, or other relevant documents.

### Step 5: Offer continuation (ADR path only)

Ask: "Want to codify this into a team standard? I can create a rule (mandatory behavior) and guide (how-to) based on this decision."

**If yes:**

**Rule:**
- Ask: "What are the mandatory behaviors? How should this be enforced?"
- Compose content covering Rule (imperative statements), Rationale, Examples (Good/Bad), Enforcement.
- `mcp__archcore__create_document(type="rule")`
- `mcp__archcore__add_relation` — rule `implements` adr

**Guide:**
- Ask: "What steps should developers follow?"
- Compose content covering Prerequisites, Steps (numbered), Verification, Common Issues.
- `mcp__archcore__create_document(type="guide")`
- `mcp__archcore__add_relation` — guide `related` rule

## Result

Minimum: one ADR or one RFC. Maximum: ADR + rule + guide (the standard chain). Report: paths, relations, recommended next actions.
