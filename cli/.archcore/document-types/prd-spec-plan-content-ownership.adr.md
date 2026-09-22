---
title: "One Content Kind, One Owning Document Type"
status: accepted
tags:
  - "docs-style"
  - "document-types"
---

## Context

A track produces several documents on one topic: a prd that states the wanted outcome, the spec that grades the behavior satisfying it, the plan that delivers it, and an adr that records a rejected alternative. The documents are linked with `implements` and `extends` relations.

Until this decision the prd template carried twelve headings and six tables, among them Solution Overview, Constraints, Risks and Mitigations, Timeline, Functional Requirements, and Non-Functional Requirements. An author filled those sections, then wrote the spec and the plan, and the same statements appeared in two documents.

Two documents holding one statement have no single owner. An edit to one leaves the other stating the opposite, and a reader cannot tell which of the two binds. The failure is recorded in the corpus that motivated the check: one sentence appeared unchanged in an idea, a prd, and the plan that implements it.

The prd type is where the pressure concentrates, because a product document is the first one written and the author has nowhere else to put a detail yet.

## Decision

Each kind of content has exactly one owning document type and one section inside it.

| Content kind | Owner | Section |
|---|---|---|
| Wanted outcome, beneficiary, threshold | prd | Requirements |
| Measured goal with units and a target value | prd | Goals and Success Metrics |
| Graded behavior: EARS clauses, BCP 14 modals | spec | Normative Behavior |
| Interfaces, signatures, states, field-driven rules | spec | Surface |
| Error, edge, and degradation handling | spec | Failure Behavior |
| Phases, tasks, milestones, delivery dates | plan | Tasks |
| Rejected alternative and the reason it was rejected | adr | Alternatives Considered |

Three changes follow from the table.

1. The prd template is reduced to four sections: Vision, Problem Statement, Goals and Success Metrics, Requirements. A section the prd does not own is not offered to the author.
2. A foreign-section check names a heading whose content another type owns, and names that type.
3. A restatement check compares the written document against the documents it builds on and names a statement that survived the move nearly word for word.

Both checks are advisory. They print after a write and reject nothing.

The table is data, not prose: `ForeignSections` in `@templates/precision.go` carries the heading-to-owner assignment, and the engines that read it are `@internal/advisory/precision.go` and `@internal/advisory/restatement.go`.

## Alternatives Considered

**Keep the wide prd template and rely on review.** Rejected. The duplication recorded in Context happened while the wide template was in place, so review alone did not hold the boundary.

**Reject the write instead of printing a finding.** Rejected. Both checks are heuristic. The restatement check compares token overlap against a threshold of 0.85 over statements of at least six content-carrying words, so a wrong verdict is possible by construction. A wrong finding costs the author a glance; a wrong rejection costs the author the write.

**Detect paraphrase, not only near-verbatim copy.** Rejected. A prd requirement and the spec behavior that grades it are meant to differ in wording, and paraphrase detection reports exactly those pairs. The check is deliberately restricted to the copy.

**List every heading a prd might carry.** Rejected. Only unambiguous headings are assigned an owner. A prd names a business constraint inside its Problem Statement without owing the reader a Constraints section, so Constraints carries no owner and produces no finding.

## Consequences

A prd written before this decision keeps its headings and now receives findings for Solution Overview and Timeline. Nothing rejects the document, and the required-section contract for a prd is unchanged, so no document loses validity.

The restatement check reads other documents from disk on every write. Two named ceilings bound that work: `maxRestatementTargets` neighbours per write, and `maxRestatementHits` findings per document. Neighbours are sorted before the cut, so adding an unrelated relation does not change which documents are compared.

The `related` and `depends_on` relation types are outside the check. An association implies no content flow, and a dependency orders two documents without moving text between them.

A content kind absent from the table is unconstrained. Adding a kind means adding a row to `ForeignSections` and nothing else — the engines read the table and hold no vocabulary of their own.
