---
name: spec
argument-hint: "[topic]"
description: "Expert — Defines a normative technical contract for a system, API, or interface. Activates when specifying behavioral guarantees, API contracts, or interface protocols — e.g., 'write a spec for', 'define the interface', 'document what this component must do'."
disable-model-invocation: true
---

# Spec — Technical Specification

## When to use

- Defining an API contract, interface, or protocol
- Specifying behavioral guarantees for a component

**Not Spec:**
- Team standard for people → **rule**
- Reference lookup → **doc**
- Requirements (what to build) → **srs** or **prd**

## Quick create

1. `mcp__archcore__list_documents(types=["spec", "prd", "srs"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What system/component/interface is being specified?"
3. Compose content covering Purpose, Scope, Authority, Subject, Contract Surface, Normative Behavior, Constraints, Invariants, Error Handling, Conformance — using user's answers for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` based on existing documents.

## Relations

| Direction | Type | Target | When |
|-----------|------|--------|------|
| Outgoing | `implements` | PRD or SRS | Formalizes requirements |
| Outgoing | `depends_on` | ADR | Depends on decisions |
| Peer | `related` | rule | Overlapping governance |

**Flows:** PRD/SRS → **Spec**; ADR → **Spec**
