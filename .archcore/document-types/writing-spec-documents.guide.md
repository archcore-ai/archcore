---
title: "How to Write a spec Document"
status: accepted
tags:
  - "document-types"
---

## Overview

A `spec` document is a canonical normative contract for a concrete system, component, interface, schema, or protocol. It defines externally observable behavior, constraints, invariants, and conformance requirements.

A good spec is precise enough for implementation and compliance checking. If implementation differs from the spec, the spec takes precedence.

### Target Audience

Engineers implementing or consuming the specified component, and AI agents that need to understand system contracts.

### Time Estimate

30–90 minutes for a focused component spec.

## Prerequisites

### Required Knowledge

- Clear understanding of the component, API, or protocol being specified
- Knowledge of its current or intended behavior, inputs, outputs, and failure modes

### Before You Start

- Check if an `adr` exists for the decision behind this component — link it via `add_relation`
- Check if an `rfc` preceded this work — the spec formalizes an accepted RFC's design
- Verify this is a normative contract, not a reference document (`doc`), cross-cutting standard (`rule`), or decision record (`adr`)

## Steps

### Step 1: Write Purpose and Scope

State what the spec defines and who it is normative for. Be precise about boundaries.

**Good scope**: "This spec defines the webhook delivery contract: payload format, delivery guarantees, retry policy, and signature verification."

**Bad scope**: "This spec covers the webhook system." (too broad — split into delivery spec + management API spec)

Remember: **one subject per spec**.

### Step 2: Establish Authority

Declare that this document is the normative specification for the subject. Link related artifacts — machine-readable schemas (OpenAPI, protobuf, SQL), related ADRs, rules, guides.

### Step 3: Define the Subject

Name the contract object precisely:
- **Name**: canonical identifier
- **Kind**: service, component, interface, schema, or protocol
- **Primary responsibility**: single sentence
- **Consumers / dependents**: who relies on this contract

### Step 4: List Definitions

Create a term table for domain-specific vocabulary used normatively in the document. Only include terms that appear in normative sections — this is not a general glossary.

### Step 5: Specify the Contract Surface

Document the externally observable contract:
- **Interfaces**: Describe each interface by its canonical identifier and a file path reference (@path/to/file), not by copying its source definition. State what the interface represents, who consumes it, and the semantically significant fields in prose or a table. Use a code block only when the exact textual format is itself normative (e.g., an HTTP endpoint shape, a wire message schema, a CLI flag grammar). Full interface or type definitions copied from source code do not belong in a spec — they will become stale when the code changes.
- **Inputs**: table with type, description, required/optional
- **Outputs**: table with type, description

Focus on what callers/consumers see — not internal implementation.

### Step 6: Document Normative Behavior

Write behavioral requirements using RFC 2119 language:

- **MUST** / **MUST NOT** — absolute requirement; violation breaks the contract
- **SHOULD** / **SHOULD NOT** — recommended; deviation requires justification
- **MAY** — optional; implementer's choice

Number each requirement for traceability. Include preconditions and postconditions.

### Step 7: Define Constraints and Invariants

Document hard limits (rate limits, payload sizes, timeouts) in a constraints table with rationale.

Separately list invariants — conditions that must always hold regardless of state or input.

### Step 8: Specify Error Handling

Document error conditions with response and recovery. Add failure semantics: what is retriable, whether processing is atomic/idempotent, timeout behavior.

### Step 9: Write Conformance Criteria

Define what it means for an implementation to conform. Typically: satisfies all MUST/MUST NOT requirements, all stated invariants, all interface requirements, all error-handling requirements, and all state transition rules if applicable.

### Step 10: Add Optional Sections

Include only when relevant:
- **State Model** — if the subject is stateful
- **Examples** — only for conformance-critical behavior
- **Security / Privacy Considerations** — if trust boundaries or data handling apply
- **Compatibility** — if backward/forward compatibility matters
- **Version History / Migration Notes** — for evolving contracts

## Verification

After writing, check:

- [ ] Purpose section declares normative authority
- [ ] Scope clearly states what is and is not covered
- [ ] Subject identifies exactly one contract object (one subject per spec)
- [ ] Every behavioral requirement uses MUST/SHOULD/MAY
- [ ] Interfaces described by canonical identifier and file path reference, not by copied source definitions
- [ ] Constraints have rationale (not just values)
- [ ] Invariants are listed separately from constraints
- [ ] Error conditions have both response and recovery
- [ ] Conformance section defines what "correct implementation" means
- [ ] No decision rationale in the spec body (that belongs in a linked `adr`)
- [ ] No general reference material dumped into the spec (that belongs in `doc`)

## Common Issues

### Issue 1: Spec is too broad

**Cause**: Trying to specify an entire subsystem in one document.

**Solution**: Split by component boundary. One API endpoint or one protocol = one spec. Link related specs via `add_relation`. Remember: one subject per spec.

### Issue 2: Mixing rationale with contract

**Cause**: Explaining "why" alongside "what" — e.g., "We use JWT because..."

**Solution**: Move rationale to a linked `adr`. The spec states the contract; the adr explains the choice.

### Issue 3: Spec reads like a guide

**Cause**: Writing sequential steps ("First, call X. Then call Y.") instead of behavioral requirements.

**Solution**: Rewrite as normative rules: "The system MUST validate the token before processing the request." If you need step-by-step instructions, create a `guide`.

### Issue 4: General reference dumping

**Cause**: Adding glossaries of everything, historical notes, or inventory lists to the spec.

**Solution**: Move non-normative reference material to a `doc`. The spec should contain only content that can be used to verify implementation.

### Issue 5: Over-specifying with inline code

**Cause**: Copying full interface, type, or struct definitions from source code into the spec because they represent "the contract". This feels precise but is a maintenance liability — the spec now holds a second copy of a definition that will diverge when the source changes.

**Solution**: Reference, don't reproduce. Use the canonical identifier name and an @-path to where it is defined. Describe the semantically significant fields in prose or a table. Reserve code blocks for wire-level or protocol-level contracts where the textual form is itself the normative artifact (HTTP endpoint shapes, CLI flag grammar, binary frame formats).

## Related Resources

- `.archcore/document-types/spec-type-usage.rule.md` — when to use `spec` vs other types
- `.archcore/dir/categories-and-document-types.doc.md` — full type system reference
