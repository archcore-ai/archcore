#!/usr/bin/env bats
# Structure tests: the delta-routing conductor contract and its wiring.
#
# The conductor contract (skills/_shared/delta-routing.md) is the route
# selection the plan skill loads at its Select step. These tests pin the
# contract's presence, the companion contracts it references, the plan skill's
# loading reference, the instrument registry's gate targets (every registry
# entry must resolve to a real '### gate:' heading in its track file), the
# owner and unknown tables, the selection line, and the free-text argument hint.

setup() {
  load '../helpers/common'
  common_setup
  SHARED="$PLUGIN_ROOT/skills/_shared"
  CONTRACT="$SHARED/delta-routing.md"
  PLAN_SKILL="$PLUGIN_ROOT/skills/plan/SKILL.md"
}

@test "delta-routing contract and companion contracts exist" {
  [ -f "$CONTRACT" ] || fail "missing skills/_shared/delta-routing.md"
  [ -f "$SHARED/capability-granularity.md" ] \
    || fail "missing skills/_shared/capability-granularity.md"
  [ -f "$SHARED/verdict-contract.md" ] \
    || fail "missing skills/_shared/verdict-contract.md"
  [ -f "$SHARED/guide-contract.md" ] \
    || fail "missing skills/_shared/guide-contract.md"
}

@test "guide contract is wired into every guide producer" {
  local f
  for f in "$SHARED/tracks/sdd.md" "$SHARED/tracks/describe.md" \
           "$SHARED/tracks/experience.md" "$SHARED/tracks/decision.md"; do
    grep -F -q 'skills/_shared/guide-contract.md' "$f" \
      || fail "no guide-contract.md reference in ${f#"$PLUGIN_ROOT"/}"
  done
}

@test "plan skill references the delta-routing contract" {
  grep -F -q 'skills/_shared/delta-routing.md' "$PLAN_SKILL" \
    || fail "plan/SKILL.md does not reference skills/_shared/delta-routing.md"
}

@test "every instrument registry gate entry resolves to a gate heading in its track file" {
  # Registry rows name their entry as: `skills/_shared/tracks/<file>.md`, gate `<gate>`
  local entries
  entries=$(sed -n \
    's|.*`skills/_shared/tracks/\([a-z-]*\)\.md`, gate `\([a-z-]*\.[a-z-]*\)`.*|\1 \2|p' \
    "$CONTRACT")
  [ -n "$entries" ] \
    || fail "no 'skills/_shared/tracks/<file>.md\`, gate \`<gate>\`' entries parsed from delta-routing.md — Instrument registry wording changed?"
  local bad=""
  local file gate track_file
  while read -r file gate; do
    track_file="$SHARED/tracks/${file}.md"
    if [ ! -f "$track_file" ]; then
      bad="$bad ${file}.md(missing track file, entry gate ${gate})"
      continue
    fi
    grep -x -F -q "### gate: ${gate}" "$track_file" \
      || bad="$bad ${gate}(no '### gate: ${gate}' heading in skills/_shared/tracks/${file}.md)"
  done <<< "$entries"
  [ -z "$bad" ] || fail "unresolvable instrument registry entries in skills/_shared/delta-routing.md:$bad"
}

@test "the contract retires route names, size labels, and the four fields" {
  grep -F -q 'The retired route names (`null`, `decision`, `amendment`, `capability`,' "$CONTRACT" \
    || fail "delta-routing.md does not retire the route names"
  [[ "$(flat "$CONTRACT")" == *'Δ, Π, M, and R fields are not computed, printed, or asked.'* ]] \
    || fail "delta-routing.md does not retire the four fields"
  ! grep -F -q 'route: <route>' "$CONTRACT" || fail "the route announcement template is still present"
  ! grep -E -q '^## (Route table|Expert invocation map|Π engagement)' "$CONTRACT" \
    || fail "a retired section heading is still present"
}

@test "plan skill argument-hint is free text" {
  local hint
  hint=$(awk '/^---$/ { if (++d == 2) exit; next }
              d == 1 && /^argument-hint:/ { print; exit }' "$PLAN_SKILL")
  [ "$hint" = 'argument-hint: "[topic]"' ] || fail "plan/SKILL.md argument-hint is not [topic]; got: ${hint}"
}

@test "selection line template appears in the contract" {
  grep -F -q 'selection: <instruments in order, or none> — owners: <type → existing or new owner>; checks: <hazard checks or none>; open: <unknowns sent to an instrument or the interview, or none>' "$CONTRACT" \
    || fail "selection line template missing from delta-routing.md"
}

@test "owner table maps needs to owners" {
  local missing="" frag
  while IFS= read -r frag; do
    [ -n "$frag" ] || continue
    grep -F -q "$frag" "$CONTRACT" || missing="$missing |${frag}|"
  done <<'ROWS'
| Product outcome, beneficiary, or measurable goal beyond one contract | `prd` |
| Depended-on behavior, interface, invariant, or failure response | `spec` — amend the covering `spec` when one exists |
| Settled choice with alternatives and reasons named in the request or evidence | `adr` |
| Two or more dependent tasks, a coordination need, or a handoff | `plan` |
| A formal traceability obligation named in the request or accepted canon | only the `brs`, `strs`, `syrs`, or `srs` levels that obligation names |
| A repeatable human operation the work introduces | `guide` |
ROWS
  [ -z "$missing" ] || fail "owner rows missing from delta-routing.md:$missing"
}

@test "selection rules pin the package mismatches the held-out cases found" {
  local c
  c=$(flat "$CONTRACT")
  [[ "$c" == *'A `plan` needs no `spec` or `prd`'* ]] || fail "plan-alone rule missing"
  [[ "$c" == *'MUST NOT create an `idea` beside a `prd` in one invocation'* ]] || fail "no idea beside prd rule missing"
  [[ "$c" == *'It engages requirement documents only under a named traceability obligation.'* ]] || fail "security-without-traceability rule missing"
  [[ "$c" == *'the conductor MUST create an `adr` through the decision instrument'* ]] || fail "adr on named alternatives rule missing"
  [[ "$c" == *'The result can be zero documents.'* ]] || fail "zero-document result missing"
}

@test "unknown table maps the four kinds" {
  grep -F -q '| `fact` answerable from the repository, canon, or git | read it; cite the artifact; no question |' "$CONTRACT" || fail "fact row"
  grep -F -q '| `choice` the user owns | interview within the ceiling of `skills/_shared/elicitation-contract.md` |' "$CONTRACT" || fail "choice row"
  grep -F -q '| `feasibility` | spike |' "$CONTRACT" || fail "feasibility row"
  grep -F -q '| `acceptance` | interview; record the answer under `## Clarifications` |' "$CONTRACT" || fail "acceptance row"
}

@test "legacy route and delta fields resume as a claim ledger" {
  [[ "$(flat "$CONTRACT")" == *'read `delta:` as the claim ledger'* ]] || fail "legacy delta resume rule missing"
}

@test "verdict contract is wired into every consumer" {
  local f
  for f in "$CONTRACT" "$SHARED/tracks/actualize.md" "$SHARED/tracks/closeout.md" \
           "$PLUGIN_ROOT/skills/review/SKILL.md"; do
    grep -F -q 'skills/_shared/verdict-contract.md' "$f" \
      || fail "no verdict-contract.md reference in ${f#"$PLUGIN_ROOT"/}"
  done
}

@test "capability-granularity contract is wired into the conductor" {
  grep -F -q 'skills/_shared/capability-granularity.md' "$CONTRACT" \
    || fail "no capability-granularity.md reference in delta-routing.md"
}

@test "closeout carries the Discharge report section" {
  grep -q -x '## Discharge report' "$SHARED/tracks/closeout.md" \
    || fail "closeout.md lost its '## Discharge report' section"
}

@test "no runtime asset applies an archived status" {
  local bad
  bad=$(grep -rni 'archived' "$PLUGIN_ROOT/skills" | grep -v 'does not exist' || true)
  [ -z "$bad" ] || fail "runtime asset mentions archived outside the kernel-absence statement: $bad"
}

@test "delta-routing state-carrier example lists the seven fields in order" {
  local got
  got=$(awk '/<!-- archcore:track/ { f = 1; next }
             f && /-->/ { f = 0; next }
             f && /^[a-z_]+:/ { n = $0; sub(/:.*/, "", n); out = (out == "" ? n : out " " n) }
             END { print out }' "$CONTRACT")
  [ "$got" = "track gate claims taxonomy asked budget deferred" ] \
    || fail "state-carrier example in delta-routing.md must list the seven fields in order; got: '${got}'"
}

@test "sdd pre-gate list_documents type list includes guide" {
  grep -F -q '`idea`, `prd`, `rnd`, `mrd`, `brd`, `urd`, `spec`, `plan`, and `guide`' \
    "$SHARED/tracks/sdd.md" || fail "sdd.md pre-gate type list lost 'guide' (or reordered)"
}

@test "claim, unknown, and evidence vocabulary tokens appear in the contract" {
  local missing="" t
  for t in create amend retire none fact choice feasibility acceptance supported missing conflicted; do
    grep -F -q "\`${t}\`" "$CONTRACT" || missing="$missing ${t}"
  done
  [ -z "$missing" ] || fail "vocabulary tokens missing from delta-routing.md:$missing"
}

@test "spike hygiene: no mainline merge, three-section rnd" {
  grep -F -q 'Spike code MUST NOT merge into the mainline' "$SHARED/tracks/research.md" \
    || fail "no-merge rule missing from research.md"
  grep -F -q 'sections Goal, Questions, and Findings' "$SHARED/tracks/research.md" \
    || fail "spike three-section pin missing from research.md"
}

@test "actualize and describe declare a Callable mode section" {
  grep -q -x '## Callable mode' "$SHARED/tracks/actualize.md" || fail "actualize.md"
  grep -q -x '## Callable mode' "$SHARED/tracks/describe.md" || fail "describe.md"
}

@test "Declared Delta section is wired through decompose, plan skill, and closeout" {
  local f
  for f in "$SHARED/tracks/sdd.md" "$PLAN_SKILL" "$SHARED/tracks/closeout.md"; do
    grep -F -q '## Declared Delta' "$f" || fail "no '## Declared Delta' in ${f#"$PLUGIN_ROOT"/}"
  done
}

@test "document and review skills share the delta vocabulary reference" {
  grep -F -q 'skills/_shared/delta-routing.md' "$PLUGIN_ROOT/skills/document/SKILL.md" || fail "document/SKILL.md"
  grep -F -q 'skills/_shared/delta-routing.md' "$PLUGIN_ROOT/skills/review/SKILL.md" || fail "review/SKILL.md"
}

@test "experience track keeps the actor boundary rule" {
  grep -F -q 'the actor decides the type' "$SHARED/tracks/experience.md" \
    || fail "actor boundary rule missing from experience.md"
}

@test "every 'entry' gate reference in the contract resolves to a gate heading" {
  local entries g bad=""
  entries=$(sed -n 's|.*entry `\([a-z-]*\.[a-z-]*\)`.*|\1|p' "$CONTRACT" | sort -u)
  [ -n "$entries" ] || fail "no 'entry \`<gate>\`' references parsed from delta-routing.md — wording changed?"
  for g in $entries; do
    grep -rqx "### gate: ${g}" "$SHARED/tracks/" || bad="$bad ${g}"
  done
  [ -z "$bad" ] || fail "unresolvable entry references in delta-routing.md:$bad"
}

@test "instrument registry lists illustrate producing scenario at sdd.illustrate" {
  grep -F -q '| illustrate | `scenario` | `skills/_shared/tracks/sdd.md`, gate `sdd.illustrate` — once per capability, after that capability'"'"'s `sdd.design` |' "$CONTRACT" \
    || fail "delta-routing.md instrument registry lacks the illustrate row"
  grep -F -q 'illustrate condition' "$CONTRACT" || fail "delta-routing.md does not define the illustrate condition"
  grep -F -q 'skills/_shared/actor-subject-compatibility.md' "$PLAN_SKILL" \
    || fail "plan/SKILL.md does not load the actor-subject compatibility file"
}

# Joins wrapped prose into one line so a phrase match survives re-wrapping.
flat() { tr '\n' ' ' < "$1" | sed 's/  */ /g'; }

@test "sdd.design keeps a gate open on an unanswered user-owned choice" {
  local design
  design=$(awk '$0 == "### gate: sdd.design" { f = 1; next } f && /^### / { exit } f' "$SHARED/tracks/sdd.md" | tr '\n' ' ' | sed 's/  */ /g')
  [[ "$design" == *'advisory: compare the draft'"'"'s scope, surface, normative behavior, invariants, failure behavior, and conformance with the request and applicable grounding'* ]] \
    || fail "sdd.design lacks the draft reflection advisory check"
  [[ "$design" == *'blocking: every material `user`-source choice that changes normative behavior, an invariant, or failure behavior is answered by the request or a recorded clarification, or recorded as delegated under `## Clarifications` and marked `[assumption]`'* ]] \
    || fail "sdd.design lacks the user-owned choice blocking check"
  [[ "$design" == *'A blocked gate cannot advance to `sdd.decompose`.'* ]] \
    || fail "sdd.design does not stop decomposition on a blocked gate"
  [[ "$(flat "$PLAN_SKILL")" == *'Do not offer implementation until that check passes'* ]] \
    || fail "plan/SKILL.md offers implementation past an open contract choice"
}

@test "gate and elicitation contracts pin draft scanning, precedent limits, and blocked-gate state" {
  local gate elicit
  gate=$(flat "$SHARED/gate-contract.md")
  elicit=$(flat "$SHARED/elicitation-contract.md")
  [[ "$gate" == *'MUST NOT treat that fact alone as a reason to skip the gate'"'"'s elicitation checks.'* ]] \
    || fail "gate-contract.md lets entry conditions skip elicitation checks"
  [[ "$gate" == *'IF an interruption left a question in `asked` without an answer or a delegation, THEN the executing skill MAY re-ask that question within the budget'* ]] \
    || fail "gate-contract.md never lets an interrupted question be asked again"
  [[ "$gate" == *'IF a blocking exit check stops the gate, THEN the executing skill MUST persist accepted answers and the state block with the current `gate`'* ]] \
    || fail "gate-contract.md does not persist state at a blocked gate"
  [[ "$elicit" == *'A similar example alone is an analogy, not evidence of the user'"'"'s intended behavior.'* ]] \
    || fail "elicitation-contract.md lacks the precedent-is-not-intent rule"
  [[ "$elicit" == *'Each material question still unasked counts as explicitly delegated, including for a gate'"'"'s blocking exit checks'* ]] \
    || fail "elicitation-contract.md lacks the double-delegation scope"
  [[ "$elicit" == *'an interruption does not resolve a material user-owned choice'* ]] \
    || fail "elicitation-contract.md lets an interruption resolve a user-owned choice"
  [[ "$(flat "$CONTRACT")" == *'resolves only the technical unknowns its recorded behavior and relevant constraints cover'* ]] \
    || fail "delta-routing.md lets a precedent resolve unknowns it does not cover"
}
