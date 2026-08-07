# Decision Track — Classify, Record as ADR or RFC, Offer a Cascade

Plugin runtime asset. Loaded by the `document` skill (primary executor) and by
the `plan` and `review` skills when their routing resolves to this track. Gate
records and track state follow `skills/_shared/gate-contract.md`; interview
mechanics follow `skills/_shared/elicitation-contract.md`.

## Track notes

- Stages: `decision.classify` → `decision.adr` | `decision.rfc` → `decision.cascade`.
- Ported from the pre-cutover `decide` skill (`skills/decide/SKILL.md` and
  `skills/decide/references/continuations.md`), removed at cutover; "the source"
  in this file names those files.
- Track question maximum: 4. The per-gate split (classify 1, adr 2, rfc 2,
  cascade 1) follows the weight of the source questions [assumption]. Each
  `budget` knob is the per-gate maximum, reached only in expert invocation; in
  auto mode every question draws from the shared per-invocation ceiling in
  `skills/_shared/elicitation-contract.md`.
- Global matches: when the duplicate check in `decision.classify` matches a
  global document (`source_kind: "global"`), the executing skill loads
  `skills/_shared/globals.md`, records the decision as a local document that
  refines or overrides the global, and never modifies the global or targets it
  with `add_relation`. This constraint binds every relation in this track.
  Absent any global match, the track proceeds as usual.
- Cascade question: the source offered the cpat opt-in as a separate question;
  this track merges that offer into the `decision.cascade` gate's single
  cascade-choice question to hold that gate's budget at 1 [assumption].

### gate: decision.classify

- Purpose: Select the branch — settled decision (`decision.adr`) or open proposal (`decision.rfc`).
- Entry conditions:
  - skip_when: the request names the target type `adr` or `rfc`.
  - The request describes one technical decision or proposal.
  - `list_documents(types=["adr", "rfc"])` returns no existing document that already records this topic (global matches: see Track notes).
- Elicitation knobs:
  - trigger: the request does not state whether the decision is settled, or open-proposal wording ("thinking about", "should we", "proposing", "design proposal") appears — confirm the RFC branch before routing.
  - taxonomy: Constraints & Tradeoffs, Completion Signals from `skills/_shared/coverage-taxonomy.md`.
  - budget: 1
- Produces: none — the `decision.adr` and `decision.rfc` gates produce the document.
- Exit checks:
  - blocking: the recorded outcome names `decision.adr` or `decision.rfc`.
- Next: `decision.adr` when the decision is settled or when no answer marks it open (the default); `decision.rfc` when the user confirms the proposal is open.

### gate: decision.adr

- Purpose: Compose and create the ADR for a settled decision per `skills/_shared/adr-contract.md` and `skills/_shared/precision-rules.md`.
- Entry conditions:
  - skip_when: `decision.classify` selected `decision.rfc`.
  - The request or recorded clarifications state the specific choice (version or name), the considered alternatives with rejection reasons, and the conditions that would invalidate the decision.
- Elicitation knobs:
  - trigger: the coverage scan returns `Missing` on a named category — the decision lacks a specific choice, named alternatives with rejection reasons, or invalidation conditions.
  - taxonomy: Constraints & Tradeoffs, Completion Signals from `skills/_shared/coverage-taxonomy.md` [assumption].
  - budget: 2
- Produces:
  - type: adr
  - status: draft
  - relations: adr `related` existing `rfc`, `spec`, or `plan` documents on the same topic — the source relate step names the link but not the relation type, and allowed unnamed further document types; narrowed to `rfc`, `spec`, and `plan` to avoid an open-ended list [assumption].
- Exit checks:
  - blocking: the draft carries every section that `skills/_shared/adr-contract.md` requires.
  - advisory: the draft introduces no word from the forbidden lexicon in `skills/_shared/precision-rules.md`.
- Next: `decision.cascade`.

### gate: decision.rfc

- Purpose: Compose and create the RFC for an open proposal, for team review before a decision is made.
- Entry conditions:
  - skip_when: `decision.classify` selected `decision.adr`.
  - The request states the proposed change and the problem it solves.
- Elicitation knobs:
  - trigger: the request does not state the proposed change or the problem it solves.
  - taxonomy: Functional Scope & Behavior, Constraints & Tradeoffs from `skills/_shared/coverage-taxonomy.md` [assumption].
  - budget: 2
- Produces:
  - type: rfc
  - status: draft
  - relations: rfc `extends` the past ADR it revises, when one exists; rfc `related` the idea that inspired it, when one exists.
- Exit checks:
  - blocking: the draft covers Summary, Motivation, Detailed Design, Drawbacks, and Alternatives.
  - advisory: the draft introduces no word from the forbidden lexicon in `skills/_shared/precision-rules.md`.
- Next: `decision.cascade` — that gate's skip_when ends the track on this branch.

### gate: decision.cascade

- Purpose: Offer the continuation cascade that matches the ADR — standard (rule + guide) or architecture (spec + plan) — and create the documents the user confirms, per `skills/_shared/precision-rules.md`, `skills/_shared/rule-contract.md` (rule), and `skills/_shared/spec-contract.md` (spec).
- Entry conditions:
  - skip_when: the track produced an `rfc`, or the ADR content matches neither signal set below — the ADR alone is a valid endpoint.
  - An ADR draft produced by `decision.adr` exists.
  - Standard-cascade signals — the decision describes enforceable behavior: "we should always", "developers must", "the team should", "going forward all X must Y".
  - Architecture-cascade signals — the decision establishes or changes a boundary contract (API, interface, schema, protocol) or a feature or subsystem with states, field-driven rules, and invariants: "the X system will provide", "the contract is", "the interface exposes", "the API will be", "the feature must behave", "the states are".
- Elicitation knobs:
  - trigger: no recorded user confirmation of a cascade exists — request wording or a recorded clarification that names the cascade counts as the confirmation; when confirmation is absent, one question offers the matching cascade; when both signal sets match, that question asks which cascade fits (standard, architecture, or neither for now); when the decision describes a before/after code-pattern shift, the question's option list includes "standard + cpat" alongside "standard", "architecture", and "neither for now".
  - taxonomy: Constraints & Tradeoffs, Integration & External Dependencies, Completion Signals from `skills/_shared/coverage-taxonomy.md` [assumption].
  - budget: 1
- Produces:
  - type: rule and guide (standard cascade), or spec and plan (architecture cascade); when the decision describes a before/after code-pattern shift, the executing skill MAY create a cpat before the rule.
  - status: draft
  - relations: rule `implements` adr; guide `related` rule; spec `implements` adr; plan `implements` spec; cpat `implements` adr; rule `related` cpat.
- Exit checks:
  - blocking: every cascade document was created after a recorded user confirmation (request wording that names the cascade counts), never before.
  - blocking: every created cascade document carries its relation from the Produces list.
  - blocking: each created rule carries every section that `skills/_shared/rule-contract.md` requires; each created spec carries every section that `skills/_shared/spec-contract.md` requires.
  - blocking: each created guide covers Prerequisites, Steps, Verification, Common Issues; each created plan covers Goal, Tasks, Acceptance Criteria, Dependencies; each created cpat covers What Changed, Why, Before, After, Scope.
  - advisory: the closing report lists document paths, relation edges, and one recommended next action.
- Next: exit.
