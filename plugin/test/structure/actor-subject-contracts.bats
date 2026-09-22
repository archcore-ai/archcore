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
  grep -F -q 'In order to <goal> / As a <actor> / I want' "$JOURNEY" || fail "missing Intent header form"
  grep -F -q 'no `spec` covers the interaction' "$JOURNEY" || fail "missing routing test against scenario"
  grep -F -q 'MUST NOT carry a BCP 14 modal' "$JOURNEY" || fail "missing no-modal rule"
  grep -F -q '120 lines' "$JOURNEY" || fail "missing 120-line cap"
  grep -F -q '`actor:<type>`' "$JOURNEY" || fail "missing tag convention"
}

@test "precision rules list both types under rule 6 and rule 7 with the F6 profile" {
  local rules="$SHARED/precision-rules.md"
  grep -F -q 'This default applies to: `adr`, `rfc`, `doc`, `prd`, `idea`, `plan`, `scenario`, `journey`, `mrd`' "$rules" \
    || fail "rule 6 architect-voice list lacks scenario and journey"
  grep -F -q '`research`, `evidence`, `cpat`, `scenario`, `journey`, `mrd`, `brd`, `urd`. A numbered clause MUST NOT carry a BCP 14 modal.' "$rules" \
    || fail "rule 7 claim-recording list lacks scenario and journey"
  grep -F -q '**Actor-subject steps**' "$rules" || fail "rule 7 has no actor-subject (F6) profile"
  grep -F -q '`<Actor> <action>; <system> <observable response>.`' "$rules" || fail "rule 7 F6 form missing"
}

@test "prd contract ownership table carries the three actor-subject rows" {
  local prd="$SHARED/prd-contract.md"
  grep -F -q '| Intended user path before a contract exists; the goal-actor-outcome header | `journey` | Journeys; Intent |' "$prd" \
    || fail "ownership table lacks the journey row"
  grep -F -q '| User-perspective flow with extensions, anchored to code | `scenario` | Flows |' "$prd" \
    || fail "ownership table lacks the scenario Flows row"
  grep -F -q '| Concrete example with data illustrating a clause | `scenario` | Examples |' "$prd" \
    || fail "ownership table lacks the scenario Examples row"
}

@test "spec contract sends examples past the allowance to a linked scenario" {
  grep -F -q 'an example past that allowance belongs in a linked `scenario`' "$SHARED/spec-contract.md" \
    || fail "spec-contract.md Conformance does not route long examples to a scenario"
}

@test "describe track reads feature files and may produce a scenario beside the spec" {
  local track="$SHARED/tracks/describe.md"
  grep -F -q '| An actor-subject flow of existing behavior with examples that illustrate a covering `spec` | `scenario` beside the `spec`, `depends_on` → that `spec` |' "$track" \
    || fail "describe.md type heuristics lack the scenario row"
  grep -F -q '`features/*.feature` files for the subject as evidence for Failure Behavior and Conformance' "$track" \
    || fail "describe.read does not record feature files as evidence"
  grep -F -q 'skills/_shared/scenario-contract.md' "$track" || fail "describe.md does not reference the scenario contract"
  grep -F -q 'never copies a feature file into `.archcore/`' "$track" || fail "describe.md lacks the no-copy rule"
}

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
