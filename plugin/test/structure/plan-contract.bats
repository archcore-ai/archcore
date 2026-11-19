#!/usr/bin/env bats
# Structure tests: the proof-tracked plan contract and the skills that load it.
# Prompt contracts, not a simulation of an agent composing a plan.

setup() {
  load '../helpers/common'
  common_setup
  SHARED="$PLUGIN_ROOT/skills/_shared"
  PLAN_CONTRACT="$SHARED/plan-contract.md"
}

@test "plan contract keeps legacy plans unchanged and states the proof rules" {
  [ -f "$PLAN_CONTRACT" ] || fail "missing skills/_shared/plan-contract.md"
  local s
  for s in "Recognition" "The frontier" "Task form" "Checking a task" "Proof verification" "Compatibility"; do
    grep -q -x -F "## $s" "$PLAN_CONTRACT" || fail "plan-contract.md has no '## $s' section"
  done
  grep -F -q 'Every other plan is a **legacy plan**' "$PLAN_CONTRACT" || fail "missing legacy-plan rule"
  grep -F -q 'MUST NOT convert a legacy plan' "$PLAN_CONTRACT" || fail "missing no-conversion rule"
  grep -F -q 'proof: <check> @ <commit sha>' "$PLAN_CONTRACT" || fail "missing proof line form"
  grep -F -q 'A checkbox task without `T<n>` keeps' "$PLAN_CONTRACT" || fail "missing T<n> recognition rule"
  grep -F -q 'hexadecimal characters and ends the line' "$PLAN_CONTRACT" || fail "missing pinned proof form"
  grep -F -q 'first commit at which the check passes' "$PLAN_CONTRACT" || fail "missing proof commit rule"
  grep -F -q 'A bare `#<n>`' "$PLAN_CONTRACT" || fail "missing pinned clause reference form"
  grep -F -q 'MUST NOT host the probe' "$PLAN_CONTRACT" || fail "missing revert-probe isolation rule"
  grep -F -q 'MUST ask the user before the probe' "$PLAN_CONTRACT" || fail "missing revert-probe confirmation rule"
  grep -F -q 'at most 17 words after `T<n>`' "$PLAN_CONTRACT" || fail "missing task-line word limit"
}

@test "decompose, plan resume, and closeout load the plan contract" {
  local f
  for f in tracks/sdd.md tracks/closeout.md; do
    grep -F -q 'skills/_shared/plan-contract.md' "$SHARED/$f" || fail "$f does not reference plan-contract.md"
  done
  grep -F -q 'skills/_shared/plan-contract.md' "$PLUGIN_ROOT/skills/plan/SKILL.md" \
    || fail "plan/SKILL.md does not reference plan-contract.md"
}

@test "closeout judges proofs and the spec check map names checks only" {
  grep -F -q '"Proof verification" in `skills/_shared/plan-contract.md` is unfulfilled' "$SHARED/tracks/closeout.md" \
    || fail "closeout.md does not judge a checked task by its proof"
  grep -F -q 'The one exception is the revert probe' "$SHARED/tracks/closeout.md" \
    || fail "closeout.md does not except the revert probe from the no-execute rule"
  grep -F -q 'The check map names checks only' "$SHARED/spec-contract.md" \
    || fail "spec-contract.md check map carries more than check names"
  ! grep -F -q 'verified @' "$SHARED/spec-contract.md" || fail "spec-contract.md duplicates the proof commit"
}

@test "plan and review state the proof-tracked form where they write and judge a plan" {
  # Agents in a long session skip the track files; the form must reach them from SKILL.md itself.
  local plan="$PLUGIN_ROOT/skills/plan/SKILL.md" review="$PLUGIN_ROOT/skills/review/SKILL.md"
  grep -F -q 'WHEN an instrument creates a `plan`, write every task' "$plan" || fail "plan/SKILL.md does not state the task form at the write"
  grep -F -q '1. [ ] T1 <one imperative action, at most 17 words>' "$plan" || fail "plan/SKILL.md has no task skeleton"
  grep -F -q '   - proof: —' "$plan" || fail "plan/SKILL.md skeleton has no proof line"
  grep -F -q 'judge each checked task per "Proof verification" in `skills/_shared/plan-contract.md`' "$review" \
    || fail "review/SKILL.md does not judge proofs at step 3"
}
