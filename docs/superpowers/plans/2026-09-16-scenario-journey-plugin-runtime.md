# Scenario and Journey Plugin Runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the plugin produce, read, and verify the `scenario` and `journey` document types that CLI v0.8.4 ships, gated on that engine version.

**Architecture:** Four units, each with its own draft spec under `.archcore/plugin/`: two content contracts under `skills/_shared/`; an `illustrate` instrument on the sdd track plus journey production at `sdd.require`; feature-file evidence on the describe track and readiness/coverage checks on the closeout track; a compatibility file at CLI 0.8.4 with two new `/archcore:document` entries. Everything is Markdown prose read by an agent plus bats tests that pin the prose; there is no compiled code.

**Tech Stack:** Markdown skill files, POSIX shell (`bin/cli-gte`), bats-core tests (`make test-structure`, `make test-integration`), the Archcore CLI 0.8.4 stdio MCP server, GitHub Actions.

**Spec:** `.archcore/plugin/scenario-and-journey-runtime.plan.md` (the Archcore plan; its 37 tasks are the tracked scope), with behavior in `.archcore/plugin/actor-subject-content-contracts.spec.md`, `.archcore/plugin/illustrate-instrument.spec.md`, `.archcore/plugin/scenario-evidence-in-describe-and-closeout.spec.md`, `.archcore/plugin/actor-subject-compatibility.spec.md`, and the design in `docs/superpowers/specs/2026-09-16-scenario-journey-plugin-runtime-design.md`. This document tracks the Archcore plan's tasks; the Archcore plan stays as written. Each task below names the Archcore task numbers it covers.

## Global Constraints

- Minimum engine: CLI `0.8.4` (tag `v0.8.4`, commit `2a8f6e4`, published 2026-09-16). The research vocabulary keeps its own file at `0.8.3`; do not edit `skills/_shared/research-compatibility.md`.
- Every `.archcore/**/*.md` write goes through the Archcore MCP tools (`create_document`, `update_document`, `add_relation`). Never `Write`/`Edit`/`sed` a file under `.archcore/`.
- Every new `.archcore/` record is created with `status: draft`. Accepted specs are edited only after the user confirms that specific edit (Task 12).
- Spec clauses: one modal, one obligated actor, `WHEN`/`IF` first, ≤ 25 words. Procedure steps: imperative, no modal, ≤ 20 words. No `SHALL`. No vagueness lexicon (`appropriate`, `robust`, `flexible`, `etc.`).
- Gate records keep the field order of `skills/_shared/gate-contract.md`: Purpose, Entry conditions (with `skip_when`), Elicitation knobs (trigger, taxonomy, budget), Produces (type, status, relations), Exit checks (`blocking`/`advisory`), Next. Every gate references shared contracts by path and restates none.
- A skill file MUST NOT branch on the host. Skill and agent content is byte-identical across hosts; only container format differs (`.md`, `.toml`, `.agent.md`).
- Existing constraint violation, not fixed here: `plugin-architecture.spec` caps a track file at 200 lines, and `sdd.md` (214) and `closeout.md` (226) already exceed it. Report the new line counts in the closing summary; do not restructure the files.
- Commit messages end with the two attribution lines:
  ```
  Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_011aP5sunXKtVEJW3CTreP9q
  ```
- Run tests from the repository root: `bats test/structure/<file>.bats`, `make test-structure`, `make test-integration` (needs `archcore` ≥ 0.8.4 on PATH or `ARCHCORE_BIN`).

---

## File Structure

| Path | Responsibility | Task |
|---|---|---|
| `plugins/archcore/skills/_shared/scenario-contract.md` | Create. What a `scenario` holds; F6 notation; over-the-cap by actor | 1 |
| `plugins/archcore/skills/_shared/journey-contract.md` | Create. What a `journey` holds; Intent header; routing test against `scenario` | 2 |
| `plugins/archcore/skills/_shared/precision-rules.md` | Modify rules 6 and 7: both types claim-recording, F6 step profile | 3 |
| `plugins/archcore/skills/_shared/prd-contract.md` | Modify ownership table: three actor-subject rows | 3 |
| `plugins/archcore/skills/_shared/spec-contract.md` | Modify Conformance: examples past five lines go to a linked `scenario` | 3 |
| `test/structure/actor-subject-contracts.bats` | Create. Pins the two contract files and the three canon hooks | 1–3 |
| `plugins/archcore/skills/_shared/actor-subject-compatibility.md` | Create. Probe at 0.8.4, result table, fallback, shared repositories | 4 |
| `plugins/archcore/agents/archcore-assistant.md`, `.toml`, `copilot-agents/archcore-assistant.agent.md` | Modify: one paragraph naming both types and the file | 4 |
| `test/structure/actor-subject-compat.bats` | Create. Pins the compat file, agent references, argument-hint parity | 4–5 |
| `plugins/archcore/skills/document/SKILL.md` | Modify: argument hint, expert form, Ground step probe trigger | 5 |
| `test/fixtures/routing/fixtures.tsv` | Modify: two rows for `document scenario` and `document journey` | 5 |
| `plugins/archcore/skills/_shared/delta-routing.md` | Modify: registry row, package contribution, illustrate condition, sequencing rule | 6 |
| `plugins/archcore/skills/plan/SKILL.md` | Modify: Ground step probe trigger and type filter | 6 |
| `plugins/archcore/skills/_shared/tracks/sdd.md` | Modify: `sdd.illustrate` gate, journey at `sdd.require`, advisory check at `sdd.design`, track note | 7 |
| `test/fixtures/goldens/sdd.golden` | Regenerate | 7 |
| `plugins/archcore/skills/_shared/tracks/describe.md` | Modify: feature files at `describe.read`, `scenario` row and Produces | 8 |
| `test/fixtures/goldens/describe.golden` | Regenerate | 8 |
| `plugins/archcore/skills/_shared/tracks/closeout.md` | Modify: readiness and coverage advisory checks, accept offer text | 9 |
| `plugins/archcore/skills/review/SKILL.md` | Modify: closeout scope filter under the probe | 9 |
| `test/fixtures/goldens/closeout.golden` | Regenerate | 9 |
| `test/behavioral/fixtures/routing-bench.tsv` | Modify: two traces | 10 |
| `Makefile`, `.github/workflows/test.yml` | Modify: pinned CLI 0.8.4 and digests | 10 |
| `test/integration/actor-subject-vocabulary.bats` | Create. Real CLI 0.8.4 over stdio MCP | 10 |
| `.archcore/plugin/*.spec.md`, `*.doc.md` (via MCP) | Modify under confirmation: four accepted specs, two docs | 11 |
| `plugins/archcore/.claude-plugin/plugin.json` and three siblings | Modify: version bump | 12 |

---

### Task 1: Scenario content contract

Covers Archcore plan tasks 1 and 6 (scenario half).

**Files:**
- Create: `plugins/archcore/skills/_shared/scenario-contract.md`
- Create: `test/structure/actor-subject-contracts.bats`

**Interfaces:**
- Consumes: section names from CLI `templates.go` — Subject, Actors, Flows, Examples, Open Questions; line format F6.
- Produces: the file path `skills/_shared/scenario-contract.md`, referenced by Tasks 7, 8, 11.

- [ ] **Step 1: Write the failing structure test**

Create `test/structure/actor-subject-contracts.bats`:

```bash
#!/usr/bin/env bats
# Structure tests: the two actor-subject content contracts and their canon hooks.
# Prompt contracts, not a simulation of an agent composing a document.

setup() {
  load '../helpers/common'
  common_setup
  SHARED="$PLUGIN_ROOT/skills/_shared"
  SCENARIO="$SHARED/scenario-contract.md"
  JOURNEY="$SHARED/journey-contract.md"
}

has_section() {
  grep -q -x -F "## $2" "$1" || { fail "missing section '## $2' in ${1#"$PLUGIN_ROOT"/}"; return 1; }
}

@test "scenario contract exists with the spec-contract section set" {
  [ -f "$SCENARIO" ] || fail "missing skills/_shared/scenario-contract.md"
  local s
  for s in "What a scenario is" "When NOT to write a scenario" "Mandatory sections" "Notation" "Body cap" "Status" "Forbidden in the body" "Enforcement" "Rationale" "Examples"; do
    has_section "$SCENARIO" "$s"
  done
  grep -q -x -F "### Over the cap — split by actor, never compress" "$SCENARIO" \
    || fail "scenario-contract.md has no 'Over the cap' section"
}

@test "scenario contract requires the five sections in template order" {
  local body
  body=$(tr '\n' ' ' < "$SCENARIO")
  [[ "$body" == *'1. **Subject**'*'2. **Actors**'*'3. **Flows**'*'4. **Examples**'*'5. **Open Questions**'* ]] \
    || fail "scenario-contract.md mandatory sections are not Subject, Actors, Flows, Examples, Open Questions in order"
}

@test "scenario contract states F6, the Anchors line, the cap, the routing test, and the tags" {
  grep -F -q '`<Actor> <action>; <system> <observable response>.`' "$SCENARIO" || fail "missing F6 step form"
  grep -F -q 'Given|When|Then|And|But <observation>.' "$SCENARIO" || fail "missing F6 observation form"
  grep -F -q '20 words or fewer' "$SCENARIO" || fail "missing 20-word step limit"
  grep -F -q 'MUST NOT carry a BCP 14 modal' "$SCENARIO" || fail "missing no-modal rule"
  grep -F -q '`Anchors:`' "$SCENARIO" || fail "missing Anchors line rule"
  grep -F -q '120 lines' "$SCENARIO" || fail "missing 120-line cap"
  grep -F -q 'a covering `spec` exists' "$SCENARIO" || fail "missing routing test against journey"
  grep -F -q '`actor:<type>`' "$SCENARIO" || fail "missing tag convention"
  grep -F -q '`spec` clause set' "$SCENARIO" || fail "missing second split boundary"
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bats test/structure/actor-subject-contracts.bats`
Expected: 3 failures, first message `missing skills/_shared/scenario-contract.md`.

- [ ] **Step 3: Write the contract file**

Create `plugins/archcore/skills/_shared/scenario-contract.md`:

````markdown
# Scenario Content Contract

Plugin runtime asset. Loaded by skills creating scenarios: `plan` (the illustrate
instrument at `sdd.illustrate`) and `document` (the describe track). Companion to
`skills/_shared/spec-contract.md`, `skills/_shared/journey-contract.md`, and
`skills/_shared/precision-rules.md`. Engine gate: `skills/_shared/actor-subject-compatibility.md`.

## What a scenario is

The record of **how a user or an external actor moves through a system**, and the
concrete examples that illustrate the clauses of one `spec`. A scenario takes the
actor as the subject of every step and carries no modal; the rules stay in the
`spec` it illustrates. It exists so a reader sees the realized path — who does
what, and what the system shows — beside the contract that governs it.

**Routing gate:** the subject of the line decides. A line that obligates the
component with a modal (`WHEN the user requests a refund, the service MUST approve
it`) is a `spec` clause. A line that takes the actor as its subject and carries no
modal (`Anna requests a refund on 10 Sep; she sees the refund approved`) is a
scenario step. Between the pair: a covering `spec` exists → `scenario`; none exists
→ `journey` (`skills/_shared/journey-contract.md`).

## When NOT to write a scenario

- A rule or an obligation → `spec`
- The intended path before any `spec` covers the interaction → `journey`
- A wanted outcome or a metric → `prd`
- A delivery task → `plan`
- A stakeholder need on the Sources or ISO tracks → `urd` User Journeys, `strs` Operational Scenarios
- An executable feature file → the test tree (`features/*.feature`), cited by `@path`, never copied into `.archcore/`

## Mandatory sections

1. **Subject** — the system and the `spec` clauses this document illustrates, by
   clause number; who depends on the illustration.
2. **Actors** — a table with the columns `Actor`, `Who they are`, `What they want`.
   The first column is the actor list every step opens with.
3. **Flows** — one `###` subsection per actor. Each subsection opens with an
   `Anchors:` line of `@path` references to the code and test files the flow walks,
   then numbered steps, then an `Extensions` list for alternative and failure paths.
4. **Examples** — a `Background` block for context shared by every example, then
   one titled example per case: a title naming what is special, an `Illustrates:`
   line with the clause number, and unfenced Given/When/Then lines. Several cases of
   one shape go in an `Examples` table with a `notes` column.
5. **Open Questions** — what the team does not know.

A scenario MAY add `## Clarifications`. It carries no other section.

## Notation

Line format F6, actor-subject step:

- Action: `<Actor> <action>; <system> <observable response>.`
- Observation: `Given|When|Then|And|But <observation>.`

Rules:

1. Every step under Flows opens with an actor named in the Actors table.
2. One step carries one action or one observation, and holds 20 words or fewer.
3. A step MUST NOT carry a BCP 14 modal; an obligation belongs in the linked `spec`.
4. `Given` states preconditions in the past tense; one action per `When`; no UI
   mechanics ("clicks the third button") — state what the actor does and sees.
5. Examples carry no dependency on one another; each runs from its Background alone.
6. A title names an activity the actor performs, not the outcome.

## Body cap

**≤ 120 lines**, counted as the `spec` cap is counted — headings and blank lines
included. The Archcore CLI reports the same cap in `@templates/precision.go`
(`MaxBodyLines`), so the contract and the hook agree.

### Over the cap — split by actor, never compress

WHEN a draft exceeds the cap, the composing skill MUST apply the first remedy the
evidence supports:

1. Reference, do not reproduce — an `Anchors:` line cites files; an utterance,
   payload, or screen longer than one example row cites its fixture by `@path`.
2. Route foreign content to its owner — a rule to the `spec`, a wanted outcome to the
   `prd`, a delivery task to the `plan`, a stakeholder need to the `urd`.
3. Split by the **actor**: Flows is already sectioned per actor, so the document
   becomes one scenario per actor (`filename=<subject-slug>-<actor-slug>`), each with
   its own Subject, linked to its siblings with `related`, and each still
   `depends_on` the one `spec` it illustrates.
4. WHEN the actor is one and the cap still exceeds, split by the `spec` clause set:
   a `spec` split by sub-surface takes its scenarios with it, one per part.
5. IF no boundary is unambiguous, THEN keep the document whole and report the excess.
6. The skill MUST NOT delete an example or a flow to fit the cap.

## Status

Created with `status: draft`. `accepted` means a reader confirmed the examples
against the running system, by a test run or by hand, and the `closeout.accept`
gate names that readiness result in its offer. `rejected` means the examples no
longer hold and no replacement was written. Archcore executes no scenario; the
status is the word of whoever confirmed it.

## Relations and tags

- `scenario depends_on spec` — the one `spec` this document illustrates; the edge the
  engine's cascade notice reads, so a `spec` edit reaches its scenarios.
- `scenario implements journey` — when a `journey` on the topic exists.
- `scenario related scenario` — between the parts of a split.
- No edge from `spec` to `scenario`; a behavior change enters `/archcore:plan` as a
  `modifies` delta with a verdict.
- Tags carry what Gherkin carries as `@tags`: `actor:<type>`, `component:<name>`,
  `nfr:<concern>`. A `plan` task or a backlog item is a tag, never an edge.

## Forbidden in the body

- A `Surface`, `Normative Behavior`, or `Failure Behavior` heading → the `spec`.
- A `Requirements` heading → the `prd`.
- A fenced feature file or a fenced example block: Examples hold unfenced lines.
- A section enumerating other `.archcore/` documents (`skills/_shared/precision-rules.md` Rule 5).
- A prescribed test runner, discovery technique, or feature-file layout.

## Enforcement

The Archcore CLI reports the mechanical part in the post-tool-use hook: the
mandatory sections, a step over 20 words, a modal in a step, a Flows subsection
without an `Anchors:` line, a step opening with neither an Actors-table actor nor
an observation keyword, a foreign heading, and the body cap. Whether the examples
hold against the running system is not decidable there; that judgement stays with
the reader at `closeout.accept`.

## Rationale

Every practice in the territory — BDD, Specification by Example, use cases,
Example Mapping — separates the rule from the example and takes the actor as the
subject of the example. The `spec` owns rules; this type owns the flow and the
examples; the subject of the line is the boundary. The `Anchors:` line is the
price of staying alive: it is what gives a scenario staleness detection and
edit-time injection.

## Examples

### Good

```markdown
## Subject
Refund approval in the orders service; illustrates clauses 2 and 4 of the
refund-policy spec. Support and the checkout UI depend on it.

## Actors
| Actor | Who they are | What they want |
|---|---|---|
| Anna | a buyer within the 14-day window | her money back without a call |

## Flows
### Anna
Anchors: @internal/orders/refund.go, @features/refund.feature
1. Anna opens the order; the page shows a Refund action.
2. Anna requests the refund; the service approves it and shows the credit date.
Extensions: 2a. Outside the window, the page shows the policy and no action.

## Examples
Background: Anna bought a book on 1 Sep.
### Refund inside the window
Illustrates: clause 2.
Given Anna bought the book on 1 Sep.
When she requests a refund on 10 Sep.
Then she sees the refund approved with a credit date of 12 Sep.
```

### Bad

```markdown
## Flows
### Buyer
1. WHEN the user requests a refund, the service MUST approve it within 200 ms.
2. Click the third button in the header, then scroll to the bottom.
```

(Line 1 is a `spec` clause: component subject with a modal. Line 2 is UI
mechanics with no actor and no observable response. Neither is a step.)
````

- [ ] **Step 4: Run the test to verify it passes**

Run: `bats test/structure/actor-subject-contracts.bats`
Expected: 3 passes (the journey and canon-hook tests are added in Tasks 2 and 3).

- [ ] **Step 5: Commit**

```bash
git add plugins/archcore/skills/_shared/scenario-contract.md test/structure/actor-subject-contracts.bats
git commit -m "feat(contracts): add scenario content contract

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_011aP5sunXKtVEJW3CTreP9q"
```

---

### Task 2: Journey content contract

Covers Archcore plan tasks 2 and 6 (journey half).

**Files:**
- Create: `plugins/archcore/skills/_shared/journey-contract.md`
- Modify: `test/structure/actor-subject-contracts.bats`

**Interfaces:**
- Consumes: section names from CLI `templates.go` — Intent, Actors, Journeys, Open Questions.
- Produces: the file path `skills/_shared/journey-contract.md`, referenced by Tasks 5, 7, 11.

- [ ] **Step 1: Add the failing tests**

Append to `test/structure/actor-subject-contracts.bats`:

```bash
@test "journey contract exists with the spec-contract section set" {
  [ -f "$JOURNEY" ] || fail "missing skills/_shared/journey-contract.md"
  local s
  for s in "What a journey is" "When NOT to write a journey" "Mandatory sections" "Notation" "Body cap" "Status" "Forbidden in the body" "Enforcement" "Rationale" "Examples"; do
    has_section "$JOURNEY" "$s"
  done
  grep -q -x -F "### Over the cap — split by actor, never compress" "$JOURNEY" \
    || fail "journey-contract.md has no 'Over the cap' section"
}

@test "journey contract requires the four sections, the Intent header, and the routing test" {
  local body
  body=$(tr '\n' ' ' < "$JOURNEY")
  [[ "$body" == *'1. **Intent**'*'2. **Actors**'*'3. **Journeys**'*'4. **Open Questions**'* ]] \
    || fail "journey-contract.md mandatory sections are not Intent, Actors, Journeys, Open Questions in order"
  grep -F -q 'In order to <goal> / As a <actor> / I want <outcome>' "$JOURNEY" || fail "missing Intent header form"
  grep -F -q 'no `spec` covers the interaction' "$JOURNEY" || fail "missing routing test against scenario"
  grep -F -q 'MUST NOT carry a BCP 14 modal' "$JOURNEY" || fail "missing no-modal rule"
  grep -F -q '120 lines' "$JOURNEY" || fail "missing 120-line cap"
  grep -F -q '`actor:<type>`' "$JOURNEY" || fail "missing tag convention"
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `bats test/structure/actor-subject-contracts.bats`
Expected: 2 failures, first message `missing skills/_shared/journey-contract.md`.

- [ ] **Step 3: Write the contract file**

Create `plugins/archcore/skills/_shared/journey-contract.md`:

````markdown
# Journey Content Contract

Plugin runtime asset. Loaded by skills creating journeys: `plan` (the intent
instrument at `sdd.require`) and `document` (`document journey`, the same gate in
callable mode). Companion to `skills/_shared/scenario-contract.md`,
`skills/_shared/prd-contract.md`, and `skills/_shared/precision-rules.md`. Engine
gate: `skills/_shared/actor-subject-compatibility.md`.

## What a journey is

The record of **the intended path of one user type through the system**, before a
`spec` covers that interaction. A journey says "we want the user to be able to…":
the actor is the subject of every step, there is no data, and there is no modal.
It is the vision half of the pair; a `scenario` is the record it becomes once a
`spec` exists, the way an `idea` becomes a `prd`.

**Routing gate:** no `spec` covers the interaction → `journey`. A covering `spec`
exists → `scenario` (`skills/_shared/scenario-contract.md`). Tense test: a journey
reads "we want the user to be able to…"; a scenario reads "the user does… and
sees…" with data in its examples.

## When NOT to write a journey

- Concrete examples with data, or a flow anchored to code → `scenario`
- A wanted outcome with a metric → `prd`
- A concept and the opportunity it opens → `idea`
- A rule or an obligation → `spec`
- A persona and usability requirement on the Sources track → `urd`

## Mandatory sections

1. **Intent** — one header, goal first: `In order to <goal> / As a <actor> / I want
   <outcome>`, followed by a short narrative of two to four sentences.
2. **Actors** — a table with the columns `Actor`, `Who they are`, `What they want`.
3. **Journeys** — one `###` subsection per actor: numbered steps in the actor's
   words, no data, no UI mechanics; alternative paths as an `Extensions` list.
4. **Open Questions** — the red cards: what the team does not know.

A journey MAY add `## Clarifications`. It carries no other section.

## Notation

Line format F6, actor-subject step, as in the scenario contract:

- Action: `<Actor> <action>; <system> <observable response>.`
- Observation: `Given|When|Then|And|But <observation>.`

Rules:

1. Every step under Journeys opens with an actor named in the Actors table.
2. One step carries one action or one observation, and holds 20 words or fewer.
3. A step MUST NOT carry a BCP 14 modal.
4. A step carries no data value; a value belongs in a scenario example.
5. The Intent header names the goal before the actor; a journey with no goal is a
   task list, not a journey.

## Body cap

**≤ 120 lines**, counted as the `spec` cap is counted. The Archcore CLI reports the
same cap in `@templates/precision.go` (`MaxBodyLines`).

### Over the cap — split by actor, never compress

WHEN a draft exceeds the cap, the composing skill MUST apply the first remedy the
evidence supports:

1. Route foreign content to its owner — a rule to a future `spec`, a wanted outcome
   to the `prd`, a data-bearing example to a `scenario`. A journey over the cap is a
   signal that the narrative has taken on rules or data; try this remedy first.
2. Split by the **actor**: one journey per user type (`filename=<subject-slug>-<actor-slug>`),
   each with its own Intent, linked to its siblings with `related`.
3. IF no boundary is unambiguous, THEN keep the document whole and report the excess.
4. The skill MUST NOT delete a journey step to fit the cap.

## Status

Created with `status: draft`. `accepted` means the team agreed this is the wanted
interaction. `rejected` means the interaction was abandoned. WHEN a `scenario` takes
over a journey's flow, the composing skill edits the journey down to intent in the
same gate close (ownership rule 2 in `skills/_shared/prd-contract.md`).

## Relations and tags

- `journey related prd`, `journey related idea` — peers on vision.
- `scenario implements journey` — added by the scenario, never by the journey.
- Tags: `actor:<type>`, `component:<name>`, `nfr:<concern>`.

## Forbidden in the body

- A `Requirements` heading → the `prd`.
- A `Surface`, `Normative Behavior`, or `Failure Behavior` heading → the `spec`.
- A Given/When/Then example with data → a `scenario`.
- A section enumerating other `.archcore/` documents (`skills/_shared/precision-rules.md` Rule 5).

## Enforcement

The Archcore CLI reports the mechanical part in the post-tool-use hook: the
mandatory sections, a step over 20 words, a modal in a step, a step opening with
neither an Actors-table actor nor an observation keyword, a foreign heading, and
the body cap. Whether the interaction is the wanted one is the team's judgement at
`closeout.accept`.

## Rationale

Smart's three-layer model names the middle layer directly: the Business Flow layer
is "the user's journey through the system". User stories are transitory planning
artifacts and belong to the `plan`; the durable narrative is the journey. Keeping
data out of it is what keeps the routing test against `scenario` decidable from
the graph rather than from a judgement about text.

## Examples

### Good

```markdown
## Intent
In order to practice English without a tutor / As a beginner / I want short
conversations that correct me gently.
A beginner opens the tutor a few minutes a day and leaves each session knowing
one thing they said wrong and how to say it.

## Actors
| Actor | Who they are | What they want |
|---|---|---|
| Beginner | A2 level, no tutor | daily practice with gentle correction |

## Journeys
### Beginner
1. The beginner starts a session; the tutor greets them and proposes a topic.
2. The beginner answers in their own words; the tutor replies and marks one error.
Extensions: 2a. The beginner asks for the rule; the tutor explains it in one line.

## Open Questions
- How many errors per session before the beginner disengages?
```

### Bad

```markdown
## Journeys
### Beginner
1. Given the beginner said "I goed home", When the tutor replies, Then it shows "went".
2. The tutor MUST correct every past-tense error.
```

(Line 1 carries data — a scenario example. Line 2 is a rule with a modal — a
`spec` clause. A journey holds neither.)
````

- [ ] **Step 4: Run the tests to verify they pass**

Run: `bats test/structure/actor-subject-contracts.bats`
Expected: 5 passes.

- [ ] **Step 5: Commit**

```bash
git add plugins/archcore/skills/_shared/journey-contract.md test/structure/actor-subject-contracts.bats
git commit -m "feat(contracts): add journey content contract

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_011aP5sunXKtVEJW3CTreP9q"
```

---

### Task 3: Canon hooks in precision-rules, prd-contract, spec-contract

Covers Archcore plan tasks 3, 4, 5.

**Files:**
- Modify: `plugins/archcore/skills/_shared/precision-rules.md` (rule 6 "This default applies to" list; rule 7 "Claim-recording types" list)
- Modify: `plugins/archcore/skills/_shared/prd-contract.md` (content-kind ownership table)
- Modify: `plugins/archcore/skills/_shared/spec-contract.md` (Mandatory sections item 6, Conformance)
- Modify: `test/structure/actor-subject-contracts.bats`

- [ ] **Step 1: Add the failing tests**

Append to `test/structure/actor-subject-contracts.bats`:

```bash
@test "precision rules list both types under rule 6 and rule 7 with the F6 profile" {
  local rules="$SHARED/precision-rules.md"
  grep -F -q 'This default applies to: `adr`, `rfc`, `doc`, `prd`, `idea`, `plan`, `scenario`, `journey`, `mrd`' "$rules" \
    || fail "rule 6 architect-voice list lacks scenario and journey"
  grep -F -q '`research`, `evidence`, `scenario`, `journey`, `cpat`, `mrd`, `brd`, `urd`. A numbered clause MUST NOT carry a BCP 14 modal.' "$rules" \
    || fail "rule 7 claim-recording list lacks scenario and journey"
  grep -F -q '**Actor-subject steps**' "$rules" || fail "rule 7 has no actor-subject (F6) profile"
  grep -F -q '`<Actor> <action>; <system> <observable response>.`' "$rules" || fail "rule 7 F6 form missing"
}

@test "prd contract ownership table carries the three actor-subject rows" {
  local prd="$SHARED/prd-contract.md"
  grep -F -q '| Intended user path before a contract exists; the goal-actor-outcome header | `journey` | Journeys; Intent |' "$prd" \
    || fail "ownership table lacks the journey row"
  grep -F -q '| User-perspective flow with extensions, anchored to code | `scenario` | Flows |' "$prd" \
    || fail "ownership table lacs the scenario Flows row"
  grep -F -q '| Concrete example with data illustrating a clause | `scenario` | Examples |' "$prd" \
    || fail "ownership table lacks the scenario Examples row"
}

@test "spec contract sends examples past the allowance to a linked scenario" {
  grep -F -q 'an example past that allowance belongs in a linked `scenario`' "$SHARED/spec-contract.md" \
    || fail "spec-contract.md Conformance does not route long examples to a scenario"
}
```

Fix the typo `lacs` → `lacks` in the second test before running.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `bats test/structure/actor-subject-contracts.bats`
Expected: 3 new failures (8 tests, 5 pass).

- [ ] **Step 3: Edit precision-rules.md rule 6**

In `plugins/archcore/skills/_shared/precision-rules.md`, replace the rule 6 closing line:

```markdown
   This default applies to: `adr`, `rfc`, `doc`, `prd`, `idea`, `plan`, `mrd`,
   `brd`, `urd`, `brs`, `strs`, `syrs`, `srs`.
```

with:

```markdown
   This default applies to: `adr`, `rfc`, `doc`, `prd`, `idea`, `plan`, `scenario`, `journey`, `mrd`,
   `brd`, `urd`, `brs`, `strs`, `syrs`, `srs`. A scenario's Examples section holds
   unfenced Given/When/Then lines; the code-block allowance above does not admit a
   fenced example or a fenced feature file.
```

- [ ] **Step 4: Edit precision-rules.md rule 7**

Replace the rule 7 claim-recording paragraph:

```markdown
   **Claim-recording types** — `adr`, `rfc`, `doc`, `prd`, `plan`, `idea`, `rnd`,
   `research`, `evidence`, `cpat`, `mrd`, `brd`, `urd`. A numbered clause MUST NOT carry a BCP 14 modal.
```

with:

```markdown
   **Claim-recording types** — `adr`, `rfc`, `doc`, `prd`, `plan`, `idea`, `rnd`,
   `research`, `evidence`, `scenario`, `journey`, `cpat`, `mrd`, `brd`, `urd`. A numbered clause MUST NOT carry a BCP 14 modal.
```

Then, after that paragraph (before rule 8), insert:

```markdown
   **Actor-subject steps** — `scenario` (Flows and Examples) and `journey`
   (Journeys). Line format F6: the actor from the Actors table opens every step,
   `<Actor> <action>; <system> <observable response>.` or
   `Given|When|Then|And|But <observation>.`; one action or observation per step,
   20 words or fewer, no modal; every Flows subsection opens with `Anchors: @path`.
   Both types stay under the 120-line body cap and split by actor past it
   (`skills/_shared/scenario-contract.md`, `skills/_shared/journey-contract.md`).
```

- [ ] **Step 5: Edit prd-contract.md ownership table**

In `plugins/archcore/skills/_shared/prd-contract.md`, after the row
`| What makes an implementation correct | `spec` | Conformance |`, insert:

```markdown
| Intended user path before a contract exists; the goal-actor-outcome header | `journey` | Journeys; Intent |
| User-perspective flow with extensions, anchored to code | `scenario` | Flows |
| Concrete example with data illustrating a clause | `scenario` | Examples |
```

- [ ] **Step 6: Edit spec-contract.md Conformance**

In `plugins/archcore/skills/_shared/spec-contract.md`, Mandatory sections item 6, replace:

```markdown
   requirements, all invariants, and all failure rules. MAY close with ONE
   non-normative example block (≤ 5 lines, Given/When/Then) anchoring the most
   load-bearing behavior.
```

with:

```markdown
   requirements, all invariants, and all failure rules. MAY close with ONE
   non-normative example block (≤ 5 lines, Given/When/Then) anchoring the most
   load-bearing behavior; an example past that allowance belongs in a linked `scenario`
   (`skills/_shared/scenario-contract.md`, `scenario depends_on spec`).
```

- [ ] **Step 7: Run the tests to verify they pass**

Run: `bats test/structure/actor-subject-contracts.bats && make test-structure`
Expected: 8 passes in the new file; the full structure suite green (no existing test pins the edited paragraphs).

- [ ] **Step 8: Commit**

```bash
git add plugins/archcore/skills/_shared/precision-rules.md plugins/archcore/skills/_shared/prd-contract.md plugins/archcore/skills/_shared/spec-contract.md test/structure/actor-subject-contracts.bats
git commit -m "feat(contracts): wire scenario and journey into the precision canon

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_011aP5sunXKtVEJW3CTreP9q"
```

---

### Task 4: Compatibility file and agent references

Covers Archcore plan tasks 19 and 22.

**Files:**
- Create: `plugins/archcore/skills/_shared/actor-subject-compatibility.md`
- Modify: `plugins/archcore/agents/archcore-assistant.md`, `plugins/archcore/agents/archcore-assistant.toml`, `plugins/archcore/copilot-agents/archcore-assistant.agent.md`
- Create: `test/structure/actor-subject-compat.bats`

**Interfaces:**
- Consumes: `bin/cli-gte <min>` printing `yes` | `no` | `__NO_CLI__`.
- Produces: the file path `skills/_shared/actor-subject-compatibility.md` and the sentinel `needs-vocabulary-probe`, referenced by Tasks 5, 6, 9.

- [ ] **Step 1: Write the failing structure test**

Create `test/structure/actor-subject-compat.bats`:

```bash
#!/usr/bin/env bats
# Structure tests: the actor-subject vocabulary gate (CLI 0.8.4) and its wiring.

setup() {
  load '../helpers/common'
  common_setup
  SHARED="$PLUGIN_ROOT/skills/_shared"
  COMPAT="$SHARED/actor-subject-compatibility.md"
}

@test "actor-subject compatibility file exists with the research-file structure" {
  [ -f "$COMPAT" ] || fail "missing skills/_shared/actor-subject-compatibility.md"
  local s
  for s in "Version probe" "Fallback" "Shared repositories"; do
    grep -q -x -F "## $s" "$COMPAT" || fail "missing section '## $s'"
  done
  grep -F -q '"$actor_subject_skill_dir/../../bin/cli-gte" 0.8.4' "$COMPAT" || fail "probe does not invoke cli-gte 0.8.4"
  grep -F -q 'The minimum is CLI `0.8.4`' "$COMPAT" || fail "minimum version line missing"
  grep -F -q 'Scenario and journey require Archcore CLI 0.8.4; skipping actor-subject documents.' "$COMPAT" \
    || fail "fallback report sentence missing"
  grep -F -q '`needs-vocabulary-probe`' "$COMPAT" || fail "no-shell sentinel missing"
  grep -F -q 'never converts' "$COMPAT" || fail "no-conversion rule missing"
}

@test "research compatibility file is unchanged by this release" {
  grep -F -q '"$research_skill_dir/../../bin/cli-gte" 0.8.3' "$SHARED/research-compatibility.md" \
    || fail "research-compatibility.md probe line changed"
  ! grep -F -q 'scenario' "$SHARED/research-compatibility.md" \
    || fail "research-compatibility.md mentions scenario; the vocabularies stay in separate files"
}

@test "every agent instruction file names both types and the compatibility file" {
  local f
  for f in "$PLUGIN_ROOT/agents/archcore-assistant.md" \
           "$PLUGIN_ROOT/agents/archcore-assistant.toml" \
           "$PLUGIN_ROOT/copilot-agents/archcore-assistant.agent.md"; do
    grep -F -q 'skills/_shared/actor-subject-compatibility.md' "$f" \
      || fail "no actor-subject-compatibility.md reference in ${f#"$PLUGIN_ROOT"/}"
    grep -F -q '`scenario` belongs to knowledge; `journey` belongs to vision' "$f" \
      || fail "no category sentence for scenario and journey in ${f#"$PLUGIN_ROOT"/}"
  done
}

@test "every skills/_shared/actor-subject-compatibility.md reference resolves" {
  local refs
  refs=$(grep -rlF 'skills/_shared/actor-subject-compatibility.md' "$PLUGIN_ROOT/skills" "$PLUGIN_ROOT/agents" "$PLUGIN_ROOT/copilot-agents" 2>/dev/null || true)
  [ -n "$refs" ] || fail "no file references skills/_shared/actor-subject-compatibility.md"
  [ -f "$COMPAT" ] || fail "referenced compatibility file does not exist"
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bats test/structure/actor-subject-compat.bats`
Expected: tests 1, 3, 4 fail; test 2 passes.

- [ ] **Step 3: Write the compatibility file**

Create `plugins/archcore/skills/_shared/actor-subject-compatibility.md`:

````markdown
# Actor-Subject Vocabulary — Engine Compatibility

Runtime contract for `plan`, `document`, and `review` when the actor-subject
types `scenario` and `journey` are in play. Load this file before the first
document search or write that could name either type. The research vocabulary
keeps its own gate in `skills/_shared/research-compatibility.md`; the two files
are independent, and a request that engages both runs both probes.

## Version probe

1. Run the probe once per invocation, before the first MCP call that would name
   `scenario` or `journey`, when any of these holds: the request or a named type
   names `scenario` or `journey`; the route engages the illustrate instrument
   (`skills/_shared/delta-routing.md`); a grounding result is a `scenario` or
   `journey` document. Otherwise, skip the probe and keep the legacy vocabulary;
   topic search without a type filter still returns documents of both types.
2. If the executor has a shell tool, run the helper below and resolve the
   executing skill's directory in the same shell call. Do not compare versions
   in prose.
3. If the executor has no shell tool and the calling skill supplied a probe
   result, use that result.
4. If the executor has no shell tool and no result is available, return
   `needs-vocabulary-probe` to the caller. The caller runs the helper and
   resumes the task with its result.
5. Before delegating scenario or journey work, the calling skill supplies the
   current invocation's probe result and the absolute plugin root. This handoff
   does not change the agent's tool permissions.
6. Do not infer version support, or an unsupported CLI, from missing handoff
   data or from a document's contents.

```sh
actor_subject_skill_dir="${CLAUDE_SKILL_DIR:-<absolute dir of the executing SKILL.md>}"
"$actor_subject_skill_dir/../../bin/cli-gte" 0.8.4
```

The minimum is CLI `0.8.4`, the published release that adds the actor-subject
vocabulary: <https://github.com/archcore-ai/cli/releases/tag/v0.8.4>.

| Result | Allowed vocabulary |
|---|---|
| `yes` | Add `scenario` and `journey` to type filters; allow both types, the `sdd.illustrate` gate, and the `document scenario` and `document journey` entries. |
| `no` or `__NO_CLI__` | Keep legacy type filters; the illustrate instrument is dropped from the package; the two entries write nothing. |

## Fallback

1. If the probe does not return `yes` and the explicit type is `scenario` or
   `journey`, report the required version and exit without a document write.
2. If the probe does not return `yes` and the route engages the illustrate
   instrument, drop the instrument and report once: "Scenario and journey
   require Archcore CLI 0.8.4; skipping actor-subject documents."
3. If a scenario or journey artifact already exists, report the required
   version and exit without rewriting it; the fallback never converts an
   existing artifact to another type.
4. If the server rejects `scenario` or `journey` after a successful probe,
   report the mismatch and stop the affected operation. Do not retry with a
   different document type.

An unavailable MCP server follows the calling skill's existing recovery path.
The PATH probe identifies the binary a new server would run; a server started
before an upgrade keeps the old engine until restarted.

## Shared repositories

This vocabulary adds no relation value, so an older engine still reads a corpus
that holds `.scenario.md` and `.journey.md` files: it skips those files in the
scan and reports an invalid type in `status`. Before adding either type to a
shared repository, report that teammates on an older CLI will not see those
documents. The local probe cannot verify teammates' versions. No downgrade
conversion is provided.
````

- [ ] **Step 4: Edit the three agent instruction files**

In each of `plugins/archcore/agents/archcore-assistant.md`,
`plugins/archcore/agents/archcore-assistant.toml` (inside the instructions string, same
indentation as the surrounding paragraphs), and
`plugins/archcore/copilot-agents/archcore-assistant.agent.md`, insert after the
paragraph that ends `Missing handoff is not evidence of an old CLI.`:

```markdown
Actor-subject vocabulary follows `skills/_shared/actor-subject-compatibility.md`:
apply that engine gate (CLI 0.8.4) before naming `scenario` or `journey` in a
filter or a write. `scenario` belongs to knowledge; `journey` belongs to vision.
A scenario illustrates one `spec` (`depends_on`); a journey records the intended
path before a `spec` exists. Compose either from
`skills/_shared/scenario-contract.md` or `skills/_shared/journey-contract.md`.
The same caller-supplied probe rule applies: without a probe result and without a
shell tool, return `needs-vocabulary-probe`.
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `bats test/structure/actor-subject-compat.bats && bats test/structure/agents.bats test/structure/agent-contracts.bats`
Expected: all pass (the agent tests compare the three containers for content parity; the paragraph must be identical in all three).

- [ ] **Step 6: Commit**

```bash
git add plugins/archcore/skills/_shared/actor-subject-compatibility.md plugins/archcore/agents plugins/archcore/copilot-agents test/structure/actor-subject-compat.bats
git commit -m "feat(compat): gate scenario and journey on CLI 0.8.4

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_011aP5sunXKtVEJW3CTreP9q"
```

---

### Task 5: `document` command entries for scenario and journey

Covers Archcore plan tasks 20 and 21.

**Files:**
- Modify: `plugins/archcore/skills/document/SKILL.md` (frontmatter `argument-hint`, Routing table, Step 1 Ground, Step 2 Expert form)
- Modify: `test/fixtures/routing/fixtures.tsv`
- Modify: `test/structure/actor-subject-compat.bats`

**Interfaces:**
- Consumes: `skills/_shared/actor-subject-compatibility.md` (Task 4); the callable `sdd.require` entry defined in Task 7.
- Produces: the argument hint `[adr|rfc|spec|doc|guide|rule|research|evidence|scenario|journey]`, pinned by Task 11's `command-surface-v2.spec` edit.

- [ ] **Step 1: Add the failing tests**

Append to `test/structure/actor-subject-compat.bats`:

```bash
@test "document argument-hint lists scenario and journey; plan hint does not" {
  local hint
  hint=$(awk '/^---$/ { if (++d == 2) exit; next }
              d == 1 && /^argument-hint:/ { print; exit }' "$PLUGIN_ROOT/skills/document/SKILL.md")
  printf '%s' "$hint" | grep -F -q 'adr|rfc|spec|doc|guide|rule|research|evidence|scenario|journey' \
    || fail "document/SKILL.md argument-hint lacks scenario and journey: $hint"
  hint=$(awk '/^---$/ { if (++d == 2) exit; next }
              d == 1 && /^argument-hint:/ { print; exit }' "$PLUGIN_ROOT/skills/plan/SKILL.md")
  ! printf '%s' "$hint" | grep -E -q 'scenario|journey' \
    || fail "plan/SKILL.md argument-hint exposes an actor-subject type: $hint"
}

@test "document expert form routes scenario to describe.read and journey to callable sdd.require" {
  local skill="$PLUGIN_ROOT/skills/document/SKILL.md"
  grep -F -q '`scenario` → describe track at `describe.read`; the named type settles' "$skill" \
    || fail "document/SKILL.md expert form lacks the scenario entry"
  grep -F -q '`journey` → sdd track at `sdd.require` in callable mode' "$skill" \
    || fail "document/SKILL.md expert form lacks the journey entry"
  grep -F -q 'skills/_shared/actor-subject-compatibility.md' "$skill" \
    || fail "document/SKILL.md does not load the actor-subject compatibility file"
}
```

Add two rows to `test/fixtures/routing/fixtures.tsv` after the `document evidence` row (tab-separated):

```
# --- actor-subject vocabulary expert forms ---
document scenario	document	skills/_shared/tracks/describe.md:describe.read	0
document journey	document	skills/_shared/tracks/sdd.md:sdd.require	5
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `bats test/structure/actor-subject-compat.bats test/structure/trigger-routing.bats`
Expected: the two new tests fail; trigger-routing fails on the two new rows (phrase not found in the document description).

- [ ] **Step 3: Edit the document skill frontmatter**

In `plugins/archcore/skills/document/SKILL.md`, replace:

```yaml
argument-hint: "[module, topic, or decision] [adr|rfc|spec|doc|guide|rule|research|evidence]"
```

with:

```yaml
argument-hint: "[module, topic, or decision] [adr|rfc|spec|doc|guide|rule|research|evidence|scenario|journey]"
```

and in the `description:` line, after `or `document evidence` to file one external material.`, insert ` Use `document scenario` to record how a user moves through existing behavior, with examples that illustrate one spec, and `document journey` to file the intended path of one user type before a spec exists.` (one line; the description stays a single YAML string).

- [ ] **Step 4: Edit the routing table and Step 1**

In the Routing table, replace the first row's signal:

```markdown
| The invocation names a type — `adr`, `rfc`, `spec`, `doc`, `guide`, `rule`, `research`, `evidence` | → expert form, no routing (Step 2) |
```

with:

```markdown
| The invocation names a type — `adr`, `rfc`, `spec`, `doc`, `guide`, `rule`, `research`, `evidence`, `scenario`, `journey` | → expert form, no routing (Step 2) |
```

In Step 1 Ground, after the paragraph that starts `Apply `skills/_shared/research-compatibility.md` under its condition 1`, insert:

```markdown
Apply `skills/_shared/actor-subject-compatibility.md` under its own condition 1 —
a request or named type naming `scenario` or `journey`, or a grounding result of
either type. When that probe returns `yes`, add `scenario` and `journey` to the
type filter below. On older engines, keep the legacy filter; an explicit
`scenario` or `journey` request then reports the required version and exits
without a write.
```

- [ ] **Step 5: Edit Step 2 Expert form**

After the `- `rule` → decision track, creation at `decision.cascade`; …` bullet, insert:

```markdown
- `scenario` → describe track at `describe.read`; the named type settles
  `describe.draft`'s type question, and `describe.read` records `features/*.feature`
  files as evidence when present. A missing covering `spec` routes to the earliest
  gate that produces it per `skills/_shared/gate-contract.md`.
- `journey` → sdd track at `sdd.require` in callable mode: the request pre-fills
  the scope, the gate produces only the `journey` and no `prd`, and the track
  exits after that gate. This mirrors `document research`: a ready vision
  artifact filed through a plan-side instrument.
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `bats test/structure/actor-subject-compat.bats test/structure/trigger-routing.bats test/structure/skills.bats`
Expected: all pass. If `trigger-routing` reports the `document journey` budget mismatch, the `sdd.require` budget is 5 in the track file; keep the fixture at 5.

- [ ] **Step 7: Commit**

```bash
git add plugins/archcore/skills/document/SKILL.md test/fixtures/routing/fixtures.tsv test/structure/actor-subject-compat.bats
git commit -m "feat(document): add scenario and journey expert entries

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_011aP5sunXKtVEJW3CTreP9q"
```

---

### Task 6: Conductor registry and plan skill wiring

Covers Archcore plan tasks 7 and 12.

**Files:**
- Modify: `plugins/archcore/skills/_shared/delta-routing.md` (Route table, Instrument registry, Sequencing rules)
- Modify: `plugins/archcore/skills/plan/SKILL.md` (Step 1 Ground)
- Modify: `test/structure/delta-routing.bats`

**Interfaces:**
- Consumes: gate `sdd.illustrate` (Task 7 creates the heading; until then the registry test fails — do Task 6 and Task 7 in one working session and run the registry test after Task 7).
- Produces: the term "illustrate condition" and the registry row `illustrate | scenario | skills/_shared/tracks/sdd.md, gate sdd.illustrate`.

- [ ] **Step 1: Add the failing test**

Append to `test/structure/delta-routing.bats`:

```bash
@test "instrument registry lists illustrate producing scenario at sdd.illustrate" {
  grep -F -q '| illustrate | `scenario` | `skills/_shared/tracks/sdd.md`, gate `sdd.illustrate` — once per capability, after that capability'"'"'s `sdd.design` |' "$CONTRACT" \
    || fail "delta-routing.md instrument registry lacks the illustrate row"
  grep -F -q 'illustrate condition' "$CONTRACT" || fail "delta-routing.md does not define the illustrate condition"
  grep -F -q 'skills/_shared/actor-subject-compatibility.md' "$PLAN_SKILL" \
    || fail "plan/SKILL.md does not load the actor-subject compatibility file"
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bats test/structure/delta-routing.bats`
Expected: the new test fails with `lacks the illustrate row`.

- [ ] **Step 3: Edit the route table**

In `plugins/archcore/skills/_shared/delta-routing.md`, in the first route table, after the row
`| `creates` ≥ 2 capabilities | one umbrella `prd` + one `spec` per capability + one `plan` |`, insert:

```markdown
| a capability meets the illustrate condition | one `scenario` for that capability through the illustrate instrument, after its `spec`; the intent instrument additionally produces one `journey` beside the `prd` |
```

After the route table's escalator bullets, add a definition paragraph:

```markdown
**Illustrate condition.** A capability meets it when its Δ names a user-facing
surface — a UI, a conversational skill, an operator-facing flow — or when
grounding finds `features/*.feature` or a BDD runner in the test-runner slot of
`skills/_shared/grounding/detect-stack.md`. The conductor reads Δ and grounding
only; it never asks the user whether to illustrate. IF grounding cannot decide
the condition, THEN record it as a `user`-source Π need. The condition binds only
when `skills/_shared/actor-subject-compatibility.md` returned `yes`; otherwise the
conductor drops the instrument and reports the required version once.
```

- [ ] **Step 4: Edit the instrument registry and sequencing rules**

In the Instrument registry table, after the `contract` row, insert:

```markdown
| illustrate | `scenario` | `skills/_shared/tracks/sdd.md`, gate `sdd.illustrate` — once per capability, after that capability's `sdd.design` |
```

In the `intent` row, change `Produces` from `` `prd` `` to `` `prd`; `journey` under the illustrate condition ``.

Append to the Sequencing rules list:

```markdown
13. WHEN a capability meets the illustrate condition, sequence `sdd.illustrate`
    directly after that capability's `sdd.design` and before `sdd.decompose`.
14. WHEN a `journey` on the topic exists and no `spec` covers the interaction,
    produce no `scenario`; the routing test between the pair is the covering
    `spec`, not the presence of any `spec`.
```

Also update the paragraph under the registry that begins `The decision instrument's `decision.cascade` gate additionally creates…` by appending one sentence:

```markdown
The intent instrument's `journey` beside the `prd` under the illustrate condition
is a second recorded exception of the same kind.
```

- [ ] **Step 5: Edit the plan skill Ground step**

In `plugins/archcore/skills/plan/SKILL.md`, Step 1 Ground, after the paragraph ending
`with the helper path and stop before the first MCP call that names either type.`, insert:

```markdown
The actor-subject probe (`skills/_shared/actor-subject-compatibility.md`) runs
under its own condition 1 — a request naming `scenario` or `journey`, a route
engaging the illustrate instrument, or a grounding result of either type — and
adds both types to the filter below only when it returns `yes`. The same
no-shell rule applies: report `needs-vocabulary-probe` and stop before the first
MCP call that names either type.
```

Extend the example filter in item 1 of the Ground list to
`types=["idea", "prd", "plan", "spec", "rnd", "rfc", "adr", "rule", "task-type", "cpat"]` plus `"scenario"`, `"journey"` when the probe returned `yes` — write it as: `… "cpat"]` — and add `"scenario"`, `"journey"` when the actor-subject probe returned `yes` —`.

- [ ] **Step 6: Run the tests after Task 7 Step 4**

Run: `bats test/structure/delta-routing.bats`
Expected: all pass, including `every instrument registry gate entry resolves to a gate heading in its track file` (needs the `### gate: sdd.illustrate` heading from Task 7).

- [ ] **Step 7: Commit together with Task 7**

(See Task 7 Step 8.)

---

### Task 7: sdd track — `sdd.illustrate`, journey at `sdd.require`, example check at `sdd.design`

Covers Archcore plan tasks 8, 9, 10, 11.

**Files:**
- Modify: `plugins/archcore/skills/_shared/tracks/sdd.md`
- Regenerate: `test/fixtures/goldens/sdd.golden`

**Interfaces:**
- Consumes: `skills/_shared/scenario-contract.md`, `skills/_shared/journey-contract.md` (Tasks 1–2); the illustrate condition (Task 6).
- Produces: gate heading `### gate: sdd.illustrate`; the callable-mode paragraph `document journey` (Task 5) relies on.

- [ ] **Step 1: Confirm the golden test fails after the edit**

Run before editing: `bats test/structure/track-goldens.bats`
Expected: all pass (baseline).

- [ ] **Step 2: Edit the Track notes**

In `plugins/archcore/skills/_shared/tracks/sdd.md`, replace the first Track notes bullet:

```markdown
- This file hosts five instruments the conductor invokes individually:
  concept → `idea` at `sdd.frame`, intent → `prd` at `sdd.require`,
  contract → `spec` at `sdd.design`, decompose → `plan` at `sdd.decompose`,
  runbook → `guide` at `sdd.runbook`.
```

with:

```markdown
- This file hosts six instruments the conductor invokes individually:
  concept → `idea` at `sdd.frame`, intent → `prd` at `sdd.require`,
  contract → `spec` at `sdd.design`, illustrate → `scenario` at
  `sdd.illustrate`, decompose → `plan` at `sdd.decompose`,
  runbook → `guide` at `sdd.runbook`.
- Under the illustrate condition of `skills/_shared/delta-routing.md`,
  `sdd.require` also produces a `journey` beside the `prd` — a recorded
  exception to single-type production, on the pattern of `decision.cascade`.
  `document journey` enters `sdd.require` in callable mode with the scope
  pre-filled from the request, produces only the `journey`, and exits.
- Both actor-subject types are gated on
  `skills/_shared/actor-subject-compatibility.md`; the conductor runs the probe
  before invoking either production.
```

Append to the Track notes (after the `[assumption] Taxonomy knob values…` bullet) the content-contract reference:

```markdown
- Per-type content contracts for the actor-subject types:
  `skills/_shared/scenario-contract.md` (`scenario`) and
  `skills/_shared/journey-contract.md` (`journey`).
```

- [ ] **Step 3: Edit `sdd.require`**

In the `### gate: sdd.require` record, replace the Produces block:

```markdown
- Produces:
  - type: prd
  - status: draft
  - relations: `implements` → the `idea` from `sdd.frame`; none when no
    `idea` exists. `related` → the `mrd`, `brd`, and `urd` on the topic, when
    they exist. A product-level `prd` additionally links each feature-scoped
    `prd` it covers.
```

with:

```markdown
- Produces:
  - type: prd; additionally journey under the illustrate condition of
    `skills/_shared/delta-routing.md`, composed per
    `skills/_shared/journey-contract.md`; journey only in callable mode from
    `document journey`
  - status: draft
  - relations: `implements` → the `idea` from `sdd.frame`; none when no
    `idea` exists. `related` → the `mrd`, `brd`, and `urd` on the topic, when
    they exist. A product-level `prd` additionally links each feature-scoped
    `prd` it covers. `related` from the journey → the `prd`, when both are
    produced.
```

Add to the `sdd.require` Exit checks, after the advisory line:

```markdown
  - blocking: WHEN a journey is produced, it contains the sections Intent,
    Actors, Journeys, and Open Questions per `skills/_shared/journey-contract.md`.
```

Change the `skip_when` of `sdd.require` by appending one clause at the end:
`; in callable mode from `document journey`, a `journey` covering the topic exists`.

- [ ] **Step 4: Add the `sdd.illustrate` gate**

Insert after the `### gate: sdd.design` record and before `### gate: sdd.decompose`:

```markdown
### gate: sdd.illustrate

- Purpose: Illustrate the clauses of the `spec` designed at `sdd.design` with
  the actor's flows and concrete examples, as a `scenario` composed per
  `skills/_shared/scenario-contract.md`.
- Entry conditions:
  - skip_when: a `scenario` that `depends_on` the designed `spec` exists in
    `.archcore/`, or the capability does not meet the illustrate condition of
    `skills/_shared/delta-routing.md`, or
    `skills/_shared/actor-subject-compatibility.md` did not return `yes`.
  - The conductor invokes this gate per its instrument-registry entry in
    `skills/_shared/delta-routing.md` — one invocation per capability, after
    that capability's `sdd.design`.
  - The designed `spec` carries numbered Normative Behavior clauses.
- Elicitation knobs:
  - trigger: the actors, one concrete example per illustrated clause, or the
    anchoring files are not recorded in the `spec`, the `prd`, the `journey`,
    or `## Clarifications`.
  - taxonomy: Interaction & UX Flow, Edge Cases & Failure Handling from
    `skills/_shared/coverage-taxonomy.md`.
  - budget: 2
- Produces:
  - type: scenario
  - status: draft
  - relations: `depends_on` → the `spec` from `sdd.design`; `implements` → the
    `journey` on the topic, when one exists; `related` between the parts of a
    split by actor.
- Exit checks:
  - blocking: the scenario draft contains the sections Subject, Actors, Flows,
    Examples, and Open Questions per `skills/_shared/scenario-contract.md`.
  - blocking: every clause number cited in Subject exists in the designed `spec`.
  - blocking: every Flows subsection opens with an `Anchors:` line.
  - blocking: the draft is within the body cap of
    `skills/_shared/scenario-contract.md`, or was split by actor with the
    remainder reported.
  - advisory: WHEN a `journey` on the topic exists, the journey was edited down
    to intent for the flows this scenario took over, per ownership rule 2 of
    `skills/_shared/prd-contract.md`.
- Next: exit — the conductor names the next instrument per
  `skills/_shared/delta-routing.md`.
```

- [ ] **Step 5: Add the advisory example check to `sdd.design`**

In the `### gate: sdd.design` Exit checks, after the third blocking line, add:

```markdown
  - advisory: each Normative Behavior clause of the spec draft is illustrated by
    one example — in its Conformance block, in a `scenario` that `depends_on`
    it, or in a feature file it cites by `@path`; the closing report lists the
    clauses with none.
```

- [ ] **Step 6: Regenerate the sdd golden and review the diff**

Run:
```bash
test/helpers/extract-gates.sh plugins/archcore/skills/_shared/tracks/sdd.md > test/fixtures/goldens/sdd.golden
git diff test/fixtures/goldens/sdd.golden
```
Expected diff: one new `gate: sdd.illustrate` block (skip_when as written, `budget: 2`, taxonomy `Interaction & UX Flow, Edge Cases & Failure Handling`, `produces: scenario draft`, `next: end`); the `sdd.require` block's `skip_when` gains the callable clause and `produces` reads `prd; additionally journey … draft`. No other line changes. If another line changed, fix the track, not the golden.

- [ ] **Step 7: Run the structure suite**

Run: `bats test/structure/track-goldens.bats test/structure/delta-routing.bats test/structure/trigger-routing.bats`
Expected: all pass (the registry test from Task 6 now resolves `sdd.illustrate`).

- [ ] **Step 8: Commit Tasks 6 and 7**

```bash
git add plugins/archcore/skills/_shared/delta-routing.md plugins/archcore/skills/plan/SKILL.md plugins/archcore/skills/_shared/tracks/sdd.md test/fixtures/goldens/sdd.golden test/structure/delta-routing.bats
git commit -m "feat(plan): add the illustrate instrument and journey production

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_011aP5sunXKtVEJW3CTreP9q"
```

---

### Task 8: Describe track — feature files as evidence, scenario production

Covers Archcore plan tasks 14 and 15.

**Files:**
- Modify: `plugins/archcore/skills/_shared/tracks/describe.md`
- Regenerate: `test/fixtures/goldens/describe.golden`
- Modify: `test/structure/actor-subject-contracts.bats`

- [ ] **Step 1: Add the failing test**

Append to `test/structure/actor-subject-contracts.bats`:

```bash
@test "describe track reads feature files and may produce a scenario beside the spec" {
  local track="$SHARED/tracks/describe.md"
  grep -F -q '| An actor-subject flow of existing behavior with examples that illustrate a covering `spec` | `scenario` beside the `spec`, `depends_on` → that `spec` |' "$track" \
    || fail "describe.md type heuristics lack the scenario row"
  grep -F -q '`features/*.feature` files for the subject are evidence for Failure Behavior and Conformance' "$track" \
    || fail "describe.read does not record feature files as evidence"
  grep -F -q 'skills/_shared/scenario-contract.md' "$track" || fail "describe.md does not reference the scenario contract"
  grep -F -q 'never copies a feature file into `.archcore/`' "$track" || fail "describe.md lacks the no-copy rule"
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bats test/structure/actor-subject-contracts.bats`
Expected: the new test fails on `type heuristics lack the scenario row`.

- [ ] **Step 3: Edit the type heuristics table**

In `plugins/archcore/skills/_shared/tracks/describe.md`, after the `| How-to instructions or procedures | `guide` |` row, insert:

```markdown
| An actor-subject flow of existing behavior with examples that illustrate a covering `spec` | `scenario` beside the `spec`, `depends_on` → that `spec` |
```

After the paragraph that begins `Default after the one type question at `describe.draft``, add:

```markdown
The `scenario` row binds only when `skills/_shared/actor-subject-compatibility.md`
returned `yes`; the executing skill composes it per
`skills/_shared/scenario-contract.md`. `describe.read` reads a feature file as
evidence and never copies a feature file into `.archcore/` — a copy is a second
canon; the scenario cites the file in its `Anchors:` line instead.
```

- [ ] **Step 4: Edit `describe.read` and `describe.draft`**

In `### gate: describe.read`, replace the Purpose line with:

```markdown
- Purpose: Gather the evidence base — files, entry points, observed behavior, and, when present, `features/*.feature` files for the subject as evidence for Failure Behavior and Conformance — and rule out duplicate documents.
```

In `### gate: describe.draft`, replace the Produces `type:` line:

```markdown
  - type: `spec`, `doc`, or `guide` per the type heuristics; the comprehensive route produces more than one document, and an over-cap subject one `spec` per separable sub-surface (`skills/_shared/spec-contract.md` "Over the cap").
```

with:

```markdown
  - type: `spec`, `doc`, `guide`, or `scenario` per the type heuristics; the comprehensive route produces more than one document, and an over-cap subject one `spec` per separable sub-surface (`skills/_shared/spec-contract.md` "Over the cap"); a `scenario` with no covering `spec` produces that `spec` first.
```

In the same gate's `relations:` line, append: `; `depends_on` from a produced `scenario` → its covering `spec``.

Add to the `describe.draft` Exit checks:

```markdown
  - blocking: a `scenario` draft carries the sections `skills/_shared/scenario-contract.md` requires, cites the feature files read at `describe.read` in its `Anchors:` lines, and carries `depends_on` → the covering `spec`.
```

- [ ] **Step 5: Regenerate the describe golden and review**

Run:
```bash
test/helpers/extract-gates.sh plugins/archcore/skills/_shared/tracks/describe.md > test/fixtures/goldens/describe.golden
git diff test/fixtures/goldens/describe.golden
```
Expected diff: only the `describe.draft` `produces:` line changes to include `scenario`. No skip_when, budget, or next changes.

- [ ] **Step 6: Run the tests**

Run: `bats test/structure/actor-subject-contracts.bats test/structure/track-goldens.bats`
Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add plugins/archcore/skills/_shared/tracks/describe.md test/fixtures/goldens/describe.golden test/structure/actor-subject-contracts.bats
git commit -m "feat(document): read feature files and produce scenarios on the describe track

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_011aP5sunXKtVEJW3CTreP9q"
```

---

### Task 9: Closeout track — readiness and coverage checks; review scope filter

Covers Archcore plan tasks 16, 17, 18.

**Files:**
- Modify: `plugins/archcore/skills/_shared/tracks/closeout.md`
- Modify: `plugins/archcore/skills/review/SKILL.md` (Grounding paragraph)
- Regenerate: `test/fixtures/goldens/closeout.golden` (expected: no change)
- Modify: `test/structure/actor-subject-contracts.bats`

- [ ] **Step 1: Add the failing test**

Append to `test/structure/actor-subject-contracts.bats`:

```bash
@test "closeout.verify reports readiness and coverage as advisory and never executes an example" {
  local track="$SHARED/tracks/closeout.md"
  local verify
  verify=$(awk '$0 == "### gate: closeout.verify" { f = 1; next } f && /^### / { exit } f' "$track" | tr '\n' ' ' | sed 's/  */ /g')
  [[ "$verify" == *'advisory: readiness — every example of each scoped `scenario` carries one result: run, confirmed, or unconfirmed'* ]] \
    || fail "closeout.verify lacks the readiness advisory check"
  [[ "$verify" == *'advisory: coverage — every Normative Behavior clause of each scoped `spec` with no example'* ]] \
    || fail "closeout.verify lacks the coverage advisory check"
  grep -F -q 'The executing skill MUST NOT execute a feature file or an example on this track.' "$track" \
    || fail "closeout.md lacks the no-execution rule"
  grep -F -q 'names that scenario'"'"'s readiness result' "$track" || fail "closeout.accept offer does not name readiness"
  grep -F -q 'skills/_shared/actor-subject-compatibility.md' "$PLUGIN_ROOT/skills/review/SKILL.md" \
    || fail "review/SKILL.md grounding does not load the actor-subject compatibility file"
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bats test/structure/actor-subject-contracts.bats`
Expected: the new test fails on `lacks the readiness advisory check`.

- [ ] **Step 3: Edit the Track notes and `closeout.verify`**

In `plugins/archcore/skills/_shared/tracks/closeout.md` Track notes, after the bullet `- The executing skill MUST NOT edit a code file on this track.`, add:

```markdown
- The executing skill MUST NOT execute a feature file or an example on this track.
  Readiness rests on a test-run report in the branch, a scenario body that
  records the confirmation, or the user's confirmation at the gate; the runtime
  infers none. Scope adds `scenario` and `journey` documents only when
  `skills/_shared/actor-subject-compatibility.md` returned `yes`.
```

In `### gate: closeout.verify`, replace the Exit checks' single advisory line:

```markdown
  - advisory: the report ends with a one-line count summary per verdict.
```

with:

```markdown
  - advisory: readiness — every example of each scoped `scenario` carries one result: run, confirmed, or unconfirmed; a user confirmation is recorded in the running report.
  - advisory: coverage — every Normative Behavior clause of each scoped `spec` with no example in its Conformance block, in a `scenario` that `depends_on` it, or in a feature file it cites is listed by clause number; a cited feature file absent from the branch, and a feature file on the branch that no scoped `spec` cites, are listed by path.
  - advisory: a scoped `scenario` whose `Anchors:` paths changed in the diff carries a verdict per `skills/_shared/verdict-contract.md`.
  - advisory: the report ends with a one-line count summary per verdict.
```

- [ ] **Step 4: Edit `closeout.accept`**

In `### gate: closeout.accept` Elicitation knobs, replace:

```markdown
  - budget: 1 question per draft document in scope [assumption] — the offer
    names the document and its verify verdict.
```

with:

```markdown
  - budget: 1 question per draft document in scope [assumption] — the offer
    names the document and its verify verdict; for a `scenario` the offer also
    names that scenario's readiness result from `closeout.verify`.
```

- [ ] **Step 5: Edit the review skill grounding**

In `plugins/archcore/skills/review/SKILL.md`, in the **Grounding.** paragraph, after
`and `research` (when `skills/_shared/research-compatibility.md` returned `yes`)`, insert
`, plus `scenario` and `journey` (when `skills/_shared/actor-subject-compatibility.md` returned `yes`)`.

- [ ] **Step 6: Regenerate the closeout golden and confirm no change**

Run:
```bash
test/helpers/extract-gates.sh plugins/archcore/skills/_shared/tracks/closeout.md > test/fixtures/goldens/closeout.golden
git diff --stat test/fixtures/goldens/closeout.golden
```
Expected: no diff (the extractor ignores exit-check prose and budget text after the integer). If the golden changed, a skip_when or Next line was touched by mistake; revert that edit.

- [ ] **Step 7: Run the tests**

Run: `bats test/structure/actor-subject-contracts.bats test/structure/track-goldens.bats test/structure/skills.bats`
Expected: all pass.

- [ ] **Step 8: Commit**

```bash
git add plugins/archcore/skills/_shared/tracks/closeout.md plugins/archcore/skills/review/SKILL.md test/structure/actor-subject-contracts.bats
git commit -m "feat(review): report scenario readiness and spec coverage at closeout

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_011aP5sunXKtVEJW3CTreP9q"
```

---

### Task 10: Bench traces, CLI pin 0.8.4, integration test

Covers Archcore plan tasks 13, 23, 24, 25 (integration half).

**Files:**
- Modify: `test/behavioral/fixtures/routing-bench.tsv`
- Modify: `Makefile` (two `Install Archcore CLI >= 0.8.3` messages)
- Modify: `.github/workflows/test.yml` (asset digests, directory, cache key, download URL)
- Create: `test/integration/actor-subject-vocabulary.bats`

**Interfaces:**
- Consumes: `test/helpers/mcp.bash` — `mcp_start`, `mcp_stop`, `mcp_call`, `mcp_tool`, `mcp_assert`, `mcp_create <type> <filename>` (sets `MCP_PATH`), `mcp_get <path>`.

- [ ] **Step 1: Write the failing integration test**

Create `test/integration/actor-subject-vocabulary.bats`:

```bash
#!/usr/bin/env bats
# Real CLI, stdio MCP, and temporary storage. This does not simulate LLM gates.

setup() {
  load '../helpers/common'
  load '../helpers/mcp'
  common_setup
  mcp_start
}

teardown() {
  mcp_stop
}

@test "MCP exposes the actor-subject vocabulary among 23 types and keeps seven relation types" {
  mcp_call tools/list '{}'
  mcp_assert '[.tools[] | select(.name == "create_document") | .inputSchema.properties.type.enum[]] | contains(["scenario","journey"])'
  mcp_assert '[.tools[] | select(.name == "create_document") | .inputSchema.properties.type.enum[]] | length == 23'
  mcp_assert '[.tools[] | select(.name == "add_relation") | .inputSchema.properties.type.enum[]] | length == 7'
}

@test "MCP scenario template has the five sections and belongs to knowledge" {
  mcp_create scenario refund-approval
  mcp_assert '.category == "knowledge"'
  mcp_get "$MCP_PATH"
  mcp_assert '.status == "draft" and ([.content | split("\n")[] | select(startswith("## ")) | ltrimstr("## ")] == ["Subject","Actors","Flows","Examples","Open Questions"])'
  mcp_assert '.content | contains("Anchors:") and contains("Illustrates:")'
}

@test "MCP journey template has the four sections and belongs to vision" {
  mcp_create journey beginner-path
  mcp_assert '.category == "vision"'
  mcp_get "$MCP_PATH"
  mcp_assert '.status == "draft" and ([.content | split("\n")[] | select(startswith("## ")) | ltrimstr("## ")] == ["Intent","Actors","Journeys","Open Questions"])'
  mcp_assert '.content | contains("In order to")'
}

@test "list_documents filters by scenario and journey" {
  mcp_create scenario refund-approval
  mcp_create journey beginner-path
  mcp_tool list_documents '{"types":["scenario","journey"]}'
  mcp_assert '[.documents[].type] | sort == ["journey","scenario"]'
}
```

- [ ] **Step 2: Run it against the local CLI**

Run: `archcore --version` (expect `0.8.4` or newer; if not, install it from <https://github.com/archcore-ai/cli/releases/tag/v0.8.4> or set `ARCHCORE_BIN`), then
`ARCHCORE_BIN=archcore PLUGIN_ROOT=$PWD/plugins/archcore REPO_ROOT=$PWD bats test/integration/actor-subject-vocabulary.bats`
Expected: 4 passes. Against CLI 0.8.3 the first test fails on the enum — that is the version gate working.

- [ ] **Step 3: Pin CI and the Makefile to 0.8.4**

In `Makefile`, replace both occurrences of `Install Archcore CLI >= 0.8.3 or set ARCHCORE_BIN` with `Install Archcore CLI >= 0.8.4 or set ARCHCORE_BIN`, and add the new file to the `test-integration` bats list:

```make
	@ARCHCORE_BIN="$(ARCHCORE_BIN)" PLUGIN_ROOT=$(PLUGIN_ROOT) REPO_ROOT=$(REPO_ROOT) bats test/integration/research-vocabulary.bats test/integration/actor-subject-vocabulary.bats test/integration/cursor-post-tool-use.bats
```

In `.github/workflows/test.yml`, in the `Resolve the pinned CLI asset` step, replace the three digests and the directory:

```bash
            Linux-x86_64) asset=archcore_linux_amd64.tar.gz; digest=7a4d1a90c3081e98f4dcb80fde226f13081001a9723d4fc932450aec5cb94ad4 ;;
            Darwin-arm64) asset=archcore_darwin_arm64.tar.gz; digest=cd3c25516abd96a5c3e7eed7d48758415fc68d8c97e654b76fe0a4acf99fc346 ;;
            Darwin-x86_64) asset=archcore_darwin_amd64.tar.gz; digest=55a102482ac7eecf2adc3f43ef23ed95f3074142f5b4456eb73980af8548be88 ;;
```

and `echo "dir=$RUNNER_TEMP/archcore-cli-0.8.4"`; in the cache step `key: archcore-cli-0.8.4-${{ steps.cli-asset.outputs.digest }}`; in the download step `https://github.com/archcore-ai/cli/releases/download/v0.8.4/$ASSET`. The digests are from `checksums.txt` of the v0.8.4 release, read on 2026-09-16.

- [ ] **Step 4: Add two routing-bench traces**

Append to `test/behavioral/fixtures/routing-bench.tsv` (tab-separated, next free ids after the last row):

```
43	Plan the beginner conversation flow for the English tutor skill	no covering spec; Δ creates one capability with a conversational, user-facing surface; no BDD runner	capability
44	Plan pagination for the public listings endpoint in a repo with features/*.feature	accepted spec covers listings; Δ modifies one capability; grounding finds features/listings.feature and a Cucumber runner	amendment
```

The route names stay `capability` and `amendment`; the traces exist so the announcement carries the illustrate instrument. Add the same two rows to the `Content` table of the bench document in Task 11 (via MCP).

- [ ] **Step 5: Run the affected suites**

Run: `make lint && make test-structure && make test-integration`
Expected: green. `make test-routing-bench` is LLM-in-the-loop and on demand; run it once in Task 12.

- [ ] **Step 6: Commit**

```bash
git add test/integration/actor-subject-vocabulary.bats test/behavioral/fixtures/routing-bench.tsv Makefile .github/workflows/test.yml
git commit -m "test: pin CLI 0.8.4 and cover the actor-subject vocabulary over MCP

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_011aP5sunXKtVEJW3CTreP9q"
```

---

### Task 11: Canon updates under confirmation (via MCP only)

Covers Archcore plan tasks 26–32. Every edit goes through `update_document` (full body) and each accepted spec needs the user's confirmation for that specific document first. Ask one confirmation per document, in one message where the host offers a widget.

**Files (via MCP, not the file tools):**
- `.archcore/plugin/track-layer.spec.md` — Surface Catalog line: `sdd` becomes `(frame → require [+ journey] → design → illustrate → decompose)`; `describe` becomes `(read code and feature files → draft spec/doc/guide/scenario → clarify gaps)`; `closeout` gains `; verify reports scenario readiness and spec coverage`. Constraints: append `- Constraint: scenario belongs to knowledge; journey belongs to vision; both are gated by @plugins/archcore/skills/_shared/actor-subject-compatibility.md.`
- `.archcore/plugin/delta-routing-instruments.spec.md` — Surface registry: add `- illustrate → `scenario` — `sdd.illustrate`, once per capability after `sdd.design`.`; intent row: `intent → `prd`, plus `journey` under the illustrate condition — `sdd.require`.`; invariant `the vocabulary release exposes 21 types` → `the vocabulary releases expose 23 types on CLI 0.8.4`; add constraint `- Constraint: the intent instrument's `journey` beside the `prd` is a recorded exception to single-type production, gated on the illustrate condition.`
- `.archcore/plugin/command-surface-v2.spec.md` — Surface expert names: `document` → add `scenario`, `journey`; add behaviors `25. WHEN the user invokes `document scenario`, the document skill MUST enter `describe.read` with the type settled.` and `26. WHEN the user invokes `document journey`, the document skill MUST enter `sdd.require` in callable mode and produce only the `journey`.`; constraint `- Constraint: the actor-subject entries bind only when the engine gate in @plugins/archcore/skills/_shared/actor-subject-compatibility.md returns `yes`.`
- `.archcore/plugin/agent-system.spec.md` — type lists: `all 23 document types` with `scenario` in knowledge and `journey` in vision; after behavior 21 add `22. BEFORE delegating scenario or journey work, the caller MUST supply the current invocation's actor-subject probe result.`; renumber the following clauses.
- `.archcore/plugin/delta-routing-type-engagement.doc.md` — title `23-Type Producer Matrix`; add rows `| `scenario` | none | illustrate instrument at `sdd.illustrate` under the illustrate condition; `describe.draft` for existing behavior; `document scenario` | added |` and `| `journey` | none | intent instrument at `sdd.require` under the illustrate condition; `document journey` | added |`; Overview count 21 → 23.
- `.archcore/plugin/delta-routing-compatibility.doc.md` — add a table `| Actor-subject surface | Current containment |` with rows: new type filters and writes → probe at 0.8.4; old CLI → files skipped in scan, no manifest hazard; illustrate on old CLI → instrument dropped, one report; and the sentence `The engine release handoff is complete: [CLI v0.8.4](https://github.com/archcore-ai/cli/releases/tag/v0.8.4) ships the actor-subject vocabulary, published 2026-09-16.`
- `.archcore/plugin/delta-routing-bench.doc.md` — append the two Task 10 traces to the cross-domain table.

- [ ] **Step 1: Ask for confirmation, one line per accepted document**

Present the six edits above and ask: "Confirm the update to `<path>`?" for each of the four accepted specs. Do not edit a declined document; record the decline in the closing summary.

- [ ] **Step 2: For each confirmed document, `get_document`, apply the edit, `update_document` with the full body**

Keep every other line byte-identical. Clause limits: ≤ 25 words, one modal.

- [ ] **Step 3: Run the CLI hook over each updated document**

Run for each path:
```bash
printf '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"%s"},"hook_event_name":"PostToolUse","cwd":"%s"}' "$p" "$PWD" | archcore hooks claude-code post-tool-use
```
Expected: no new Precision finding beyond those the document carried before the edit.

- [ ] **Step 4: Sweep the old type count**

Run: `grep -rn "21 types\|21 document types\|21 reachable\|expose 21" .archcore plugins/archcore docs README.md`
Expected: every remaining hit describes history (the research release), not the current registry. Fix current-registry hits through `update_document` (`.archcore/`) or the file tools (everything else).

- [ ] **Step 5: Remove the two `[assumption]` release notes**

In `.archcore/plugin/scenario-and-journey-runtime.prd.md` Dependencies and `.archcore/plugin/scenario-and-journey-runtime.plan.md` Dependencies, replace the sentence `[assumption] Whether the tag is published as a GitHub release was not verified from this repository.` (and its plan variant) with `The tag is published as [CLI v0.8.4](https://github.com/archcore-ai/cli/releases/tag/v0.8.4), 2026-09-16.` via `update_document`.

- [ ] **Step 6: Commit the canon**

```bash
git add .archcore
git commit -m "docs(archcore): record the scenario and journey runtime in the canon

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_011aP5sunXKtVEJW3CTreP9q"
```

---

### Task 12: Verification, negative check, version bump

Covers Archcore plan tasks 33–36 (37 stays open: the global RFC status is not this repository's).

- [ ] **Step 1: Full verification**

Run: `make all && make test-integration`
Expected: lint clean, structure and unit suites green, integration green against CLI 0.8.4. Record the command output summary in the Archcore plan's Dependencies section via `update_document` (one paragraph: date, CLI version, suite counts).

- [ ] **Step 2: Routing bench on the two new traces**

Run: `make test-routing-bench`
Expected: 44 of 44 route names match; the announcements for traces 43 and 44 name the illustrate instrument. This spends model tokens; run once. Record the result beside Step 1's paragraph.

- [ ] **Step 3: Negative check on the registry row**

Edit `plugins/archcore/skills/_shared/delta-routing.md`: delete the `| illustrate | …` row. Run `bats test/structure/delta-routing.bats`. Expected: `instrument registry lists illustrate…` fails. Restore the row by editing the line back (never `git checkout`). Run again. Expected: pass.

- [ ] **Step 4: Line counts**

Run: `wc -l plugins/archcore/skills/_shared/tracks/sdd.md plugins/archcore/skills/_shared/tracks/closeout.md plugins/archcore/skills/*/SKILL.md`
Expected: `sdd.md` about 260 lines, `closeout.md` about 235 — both over the 200-line constraint of `plugin-architecture.spec`, which they already exceeded before this work. Report the numbers; a decomposition of `sdd.md` is a separate decision for `/archcore:document`.

- [ ] **Step 5: Bump the plugin version**

Set `"version": "0.8.4"` in `plugins/archcore/.claude-plugin/plugin.json`, `plugins/archcore/.cursor-plugin/plugin.json`, `plugins/archcore/.codex-plugin/plugin.json`, and `plugins/archcore/.plugin/plugin.json` (byte-identical string; the plugin tracks the engine minor it requires). Run: `bats test/structure/manifest-version-parity.bats`. Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add plugins/archcore/.claude-plugin/plugin.json plugins/archcore/.cursor-plugin/plugin.json plugins/archcore/.codex-plugin/plugin.json plugins/archcore/.plugin/plugin.json .archcore
git commit -m "chore: bump plugin to 0.8.4 for the actor-subject vocabulary

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_011aP5sunXKtVEJW3CTreP9q"
```

- [ ] **Step 7: Hand off to review**

Run `/archcore:review` on the branch to reconcile the Declared Delta of `.archcore/plugin/scenario-and-journey-runtime.plan.md` (four `creates`, four `modifies` verdicts) against the diff. The tag waits for the global RFC to reach `accepted` (Archcore plan task 37).

---

## Self-review

- Spec coverage: content contracts (T1–T3), illustrate instrument and journey at `sdd.require` and the `sdd.design` check (T6–T7), describe evidence and scenario (T8), closeout readiness and coverage and the review filter (T9), compatibility file and agents (T4), `document` entries and grounding (T5), canon updates and count sweep (T11), tests, CI pin, bench traces, version bump (T10, T12). Archcore plan tasks 1–36 map to T1–T12; task 37 is external.
- Placeholders: none; every edit carries its text. The one judgement left to the executor is the `sdd.golden` diff review in T7 Step 6, which the test file's own header defines.
- Consistency: the file names `scenario-contract.md`, `journey-contract.md`, `actor-subject-compatibility.md`; the gate `sdd.illustrate`; the phrase "illustrate condition"; the sentinel `needs-vocabulary-probe`; the fallback sentence — identical across tasks and pinned by the tests in T1, T4, T6, T8, T9.
- Overlap note (integration rule 9): this plan tracks the tasks of `.archcore/plugin/scenario-and-journey-runtime.plan.md`; that plan is kept as written, and its task numbers are cited per task above.
