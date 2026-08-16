#!/usr/bin/env bats
# Structure tests: the delta-routing conductor contract and its wiring.
#
# The conductor contract (skills/_shared/delta-routing.md) is the route
# computation the plan skill loads at its Route step. These tests pin the
# contract's presence, the companion contracts it references, the plan skill's
# loading reference, the instrument registry's gate targets (every registry
# entry must resolve to a real '### gate:' heading in its track file), the
# five route names, the route announcement template, and the expert-path
# argument hint.

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

@test "all five route names appear in the delta-routing contract" {
  local missing=""
  local r
  for r in null decision amendment capability umbrella; do
    grep -F -q "\`${r}\`" "$CONTRACT" || missing="$missing ${r}"
  done
  [ -z "$missing" ] || fail "route names missing from skills/_shared/delta-routing.md:$missing"
}

@test "plan skill argument-hint keeps the four expert aliases" {
  local hint
  hint=$(awk '/^---$/ { if (++d == 2) exit; next }
              d == 1 && /^argument-hint:/ { print; exit }' "$PLAN_SKILL")
  [ -n "$hint" ] || fail "plan/SKILL.md frontmatter has no argument-hint: line"
  printf '%s' "$hint" | grep -F -q 'sdd | sources | iso | research' \
    || fail "plan/SKILL.md argument-hint lost 'sdd | sources | iso | research'; got: ${hint}"
}

@test "route announcement template line appears in the delta-routing contract" {
  grep -F -q 'route: <route>' "$CONTRACT" \
    || fail "route announcement template 'route: <route>' missing from skills/_shared/delta-routing.md"
}

@test "route table carries all eight contribution rows" {
  local missing=""
  while IFS= read -r frag; do
    [ -n "$frag" ] || continue
    grep -F -q "$frag" "$CONTRACT" || missing="$missing |${frag}|"
  done <<'ROWS'
| every capability list empty ∧ `decision: none` ∧ `intent_gap: no` | nothing
| `decision` non-empty | decision instrument (`adr` / `rfc`) |
| `modifies` non-empty | per modified capability: one code-wrong/spec-wrong verdict
| `retires` non-empty | per retired capability: a retirement entry reported for closeout discharge |
| `intent_gap: yes` | one `prd` through the intent instrument WHEN the gap names goals or metrics
| `creates` = 1 capability | one `spec` + one `plan` |
| `creates` ≥ 2 capabilities | one umbrella `prd` + one `spec` per capability + one `plan` |
| implementation spanning two or more tasks, on a `decision` or `amendment` route | one `plan` through the decompose instrument |
ROWS
  [ -z "$missing" ] || fail "contribution rows missing from delta-routing.md:$missing"
}

@test "route-name and base-label table pins the six mappings" {
  grep -F -q '| `creates` ≥ 2 | `umbrella` | L |' "$CONTRACT" || fail "umbrella/L row"
  grep -F -q '| `creates` = 1 | `capability` | M |' "$CONTRACT" || fail "capability/M row"
  grep -F -q '| `modifies` or `retires` | `amendment` | S |' "$CONTRACT" || fail "amendment/S row"
  grep -F -q '| `decision` | `decision` | S |' "$CONTRACT" || fail "decision/S row"
  grep -F -q '| `intent_gap` only | `null` — intent recorded, no build package | S |' "$CONTRACT" || fail "intent-gap-only/null row"
  grep -F -q '| nothing | `null` | S |' "$CONTRACT" || fail "nothing/null row"
}

@test "route announcement template carries the full delta-routing grammar" {
  grep -F -q 'route: <route> (size <label>) — Δ: <non-empty fields>; Π: <sources engaged>; M: <pencil or stone>; R: <flags or none>; raised by: <flag or none>; instruments: <ordered list or none>' "$CONTRACT" \
    || fail "full announcement template line missing from delta-routing.md"
}

@test "escalators and portfolio threshold are pinned" {
  grep -F -q 'WHEN iso links engage, raise one' "$CONTRACT" || fail "iso-links extra label step missing"
  grep -F -q 'Cap the label at `XL`' "$CONTRACT" || fail "XL cap missing"
  grep -F -q 'two or more needs name `world`, `undecided`, or `empirical` sources' "$CONTRACT" \
    || fail "portfolio threshold (world/undecided/empirical) missing"
}

@test "expert invocation map keeps the four alias rows" {
  grep -F -q '| `sdd` | full package: intent → contract (per capability) → decompose, at per-gate maxima |' "$CONTRACT" || fail "sdd row"
  grep -F -q '| `sources` | acquisition instrument, entry `requirements-cascade.mrd` |' "$CONTRACT" || fail "sources row"
  grep -F -q '| `iso` | iso links, entry `requirements-cascade.brs` |' "$CONTRACT" || fail "iso row"
  grep -F -q '| `research` | research instrument, entry `research.frame` |' "$CONTRACT" || fail "research row"
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

@test "delta-routing state-carrier example lists the eight fields in order" {
  local got
  got=$(awk '/<!-- archcore:track/ { f = 1; next }
             f && /-->/ { f = 0; next }
             f && /^[a-z_]+:/ { n = $0; sub(/:.*/, "", n); out = (out == "" ? n : out " " n) }
             END { print out }' "$CONTRACT")
  [ "$got" = "track gate route delta taxonomy asked budget deferred" ] \
    || fail "state-carrier example in delta-routing.md must list the eight fields in order; got: '${got}'"
}

@test "sdd pre-gate list_documents type list includes guide" {
  grep -F -q '`idea`, `prd`, `rnd`, `mrd`, `brd`, `urd`, `spec`, `plan`, and `guide`' \
    "$SHARED/tracks/sdd.md" || fail "sdd.md pre-gate type list lost 'guide' (or reordered)"
}

@test "pi engagement table maps all five sources" {
  grep -F -q '| `machine` | compose without a question; cite the artifact |' "$CONTRACT" || fail "machine row"
  grep -F -q '| `user` | interview within the ceiling of `skills/_shared/elicitation-contract.md` |' "$CONTRACT" || fail "user row"
  grep -F -q '| `world` | research instrument |' "$CONTRACT" || fail "world row"
  grep -F -q '| `undecided` | decision instrument |' "$CONTRACT" || fail "undecided row"
  grep -F -q '| `empirical` | spike |' "$CONTRACT" || fail "empirical row"
}

@test "delta field and risk flag vocabulary tokens appear in the contract" {
  local missing="" t
  for t in creates modifies retires intent_gap \
           external-contract data-migration security-compliance irreversibility multi-team; do
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
