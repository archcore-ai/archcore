---
name: decide
argument-hint: "[decision topic]"
description: Record an architectural or technical decision with context and alternatives.
disable-model-invocation: true
---

# /archcore:decide

Record a decision. Creates an ADR (Architecture Decision Record) and optionally offers to codify it into a team standard (rule + guide).

## When to use

- "Record the decision to use PostgreSQL"
- "We decided to go with microservices"
- "Document why we chose JWT over sessions"
- "Let's capture this decision"

**Not decide:**
- Planning a feature → `/archcore:plan`
- Making a standard → `/archcore:standard` (use this if you already know it should become a rule)
- Documenting a component → `/archcore:capture`
- Still discussing, no decision yet → use `/archcore:rfc` directly

## Routing table

| Signal | Route | Documents |
|---|---|---|
| User describes a **finalized decision** (default) | → adr | Single ADR |
| User says "and make it a standard" or implies enforcement | → adr + standard-track continuation | ADR, then offer rule + guide |
| Still discussing, not decided | → redirect to rfc | Suggest `/archcore:rfc` instead |

Default: create a single ADR. After creation, always offer: "Want to codify this into a team standard? (rule + guide)"

## Execution

### Step 1: Check existing

`mcp__archcore__list_documents(types=["adr", "rfc"])` — check for existing decisions on this topic.

### Step 2: Route

If user language suggests the decision is still open ("thinking about", "should we", "proposal"), suggest `/archcore:rfc` instead and stop. Otherwise proceed with ADR.

### Step 3: Create ADR

- Ask: "What was the decision? What alternatives were considered?"
- Compose content covering Context, Decision, Alternatives Considered, Consequences.
- `mcp__archcore__create_document(type="adr")`

### Step 4: Relate

`mcp__archcore__add_relation` — link to existing RFCs, specs, plans, or other relevant documents.

### Step 5: Offer continuation

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

Minimum: one ADR. Maximum: ADR + rule + guide (the standard-track chain). Report: paths, relations, recommended next actions.
