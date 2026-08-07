#!/usr/bin/env bats
# Trigger-phrase regression suite (Stage 5).
#
# The v2 command surface (init, plan, document, review) is auto-invoked by the
# host from each skill's frontmatter description — the description IS the
# routing input. test/fixtures/routing/fixtures.tsv pins trigger phrases to
# their expected skill, track entry gate, and entry-gate question budget, so a
# silent description or track edit that would re-route a phrase fails here.
#
# Scope note: cross-routing is checked ONLY among the four v2 skills. The
# legacy skills (capture, decide, audit, context, help) still exist until the
# Stage 4 cutover deletes them, and their descriptions intentionally overlap
# the v2 surface during coexistence — they are deliberately excluded. After
# Stage 4 the four-skill scope IS the whole surface, so the suite ratchets on
# its own.

setup() {
  load '../helpers/common'
  common_setup
  ROUTING_TSV="$REPO_ROOT/test/fixtures/routing/fixtures.tsv"
  V2_SKILLS="init plan document review"
}

# Print the tsv data rows only (comment and blank lines stripped).
tsv_rows() {
  grep -v '^#' "$ROUTING_TSV" | grep -v '^[[:space:]]*$'
}

# Print the single-line frontmatter description of a v2 skill.
skill_description() {
  awk '/^---$/ { if (++d == 2) exit; next }
       d == 1 && /^description:/ { print; exit }' \
    "$PLUGIN_ROOT/skills/$1/SKILL.md"
}

# Print the integer on the first "- budget:" knob line inside the named gate's
# record (from its "### gate: <name>" heading to the next section heading).
# Empty output = no budget knob found in that gate block.
gate_budget() {
  local file="$1" gate="$2"
  awk -v g="### gate: $gate" '
    $0 == g { inblock = 1; next }
    inblock && (/^### / || /^## /) { exit }
    inblock && /- budget:/ { print; exit }
  ' "$file" | sed -n 's/.*budget:[[:space:]]*\([0-9][0-9]*\).*/\1/p'
}

@test "routing fixtures tsv is well-formed: 4 tab-separated fields, valid skill/track/budget values" {
  [ -f "$ROUTING_TSV" ] || fail "missing fixture file: $ROUTING_TSV"
  local bad=""
  local phrase skill track budget
  while IFS=$'\t' read -r phrase skill track budget; do
    if [ -z "$phrase" ] || [ -z "$skill" ] || [ -z "$track" ] || [ -z "$budget" ]; then
      bad="$bad
  malformed row (need 4 tab-separated fields): [$phrase|$skill|$track|$budget]"
      continue
    fi
    case " $V2_SKILLS " in
      *" $skill "*) ;;
      *) bad="$bad
  invalid skill '$skill' (expected one of: $V2_SKILLS) for phrase: $phrase" ;;
    esac
    case "$track" in
      -) ;;
      skills/_shared/tracks/*.md:*) ;;
      *) bad="$bad
  invalid track '$track' (expected skills/_shared/tracks/<file>.md:<gate> or '-') for phrase: $phrase" ;;
    esac
    case "$budget" in
      -) ;;
      *[!0-9]*|'') bad="$bad
  invalid budget '$budget' (expected integer or '-') for phrase: $phrase" ;;
    esac
  done < <(tsv_rows)
  [ -z "$bad" ] || fail "schema violations in $ROUTING_TSV:$bad"
}

@test "fixture corpus meets minimum per-skill coverage and hits both cascade modes" {
  # Guards the corpus itself against erosion: plan-task 25 requires 3+ plan,
  # 4+ document, 3+ review, 1+ init fixtures, with sources-mode (mrd) and
  # iso-mode (brs) cascade entries both represented.
  local n_plan n_document n_review n_init
  n_plan=$(tsv_rows | awk -F'\t' '$2 == "plan"' | wc -l | tr -d ' ')
  n_document=$(tsv_rows | awk -F'\t' '$2 == "document"' | wc -l | tr -d ' ')
  n_review=$(tsv_rows | awk -F'\t' '$2 == "review"' | wc -l | tr -d ' ')
  n_init=$(tsv_rows | awk -F'\t' '$2 == "init"' | wc -l | tr -d ' ')
  [ "$n_plan" -ge 3 ] || fail "need 3+ plan fixtures, found $n_plan"
  [ "$n_document" -ge 4 ] || fail "need 4+ document fixtures, found $n_document"
  [ "$n_review" -ge 3 ] || fail "need 3+ review fixtures, found $n_review"
  [ "$n_init" -ge 1 ] || fail "need 1+ init fixtures, found $n_init"
  tsv_rows | grep -F -q ':requirements-cascade.mrd' \
    || fail "no fixture routes to sources-mode entry requirements-cascade.mrd"
  tsv_rows | grep -F -q ':requirements-cascade.brs' \
    || fail "no fixture routes to iso-mode entry requirements-cascade.brs"
}

@test "every fixture phrase appears in its expected skill's description" {
  local failures=""
  local phrase skill track budget desc
  while IFS=$'\t' read -r phrase skill track budget; do
    desc=$(skill_description "$skill")
    if [ -z "$desc" ]; then
      failures="$failures
  $skill/SKILL.md has no description: line in frontmatter"
      continue
    fi
    printf '%s' "$desc" | grep -i -F -q -- "$phrase" || failures="$failures
  phrase '$phrase' missing from $skill/SKILL.md description"
  done < <(tsv_rows)
  [ -z "$failures" ] || fail "trigger phrases lost from descriptions:$failures"
}

@test "no fixture phrase appears in any other v2 skill description (zero cross-routing)" {
  local failures=""
  local phrase skill track budget other desc
  while IFS=$'\t' read -r phrase skill track budget; do
    for other in $V2_SKILLS; do
      [ "$other" = "$skill" ] && continue
      desc=$(skill_description "$other")
      if printf '%s' "$desc" | grep -i -F -q -- "$phrase"; then
        failures="$failures
  phrase '$phrase' (owned by $skill) also matches $other/SKILL.md description"
      fi
    done
  done < <(tsv_rows)
  [ -z "$failures" ] || fail "cross-routing collisions:$failures"
}

@test "every fixture track file exists and contains the entry-gate heading" {
  local failures=""
  local phrase skill track budget track_file gate
  while IFS=$'\t' read -r phrase skill track budget; do
    [ "$track" = "-" ] && continue
    track_file="$PLUGIN_ROOT/${track%:*}"
    gate="${track##*:}"
    if [ ! -f "$track_file" ]; then
      failures="$failures
  track file missing: ${track%:*} (phrase '$phrase')"
      continue
    fi
    grep -x -F -q -- "### gate: $gate" "$track_file" || failures="$failures
  heading '### gate: $gate' missing from ${track%:*} (phrase '$phrase')"
  done < <(tsv_rows)
  [ -z "$failures" ] || fail "track/gate regressions:$failures"
}

@test "entry-gate budget knob carries the fixture's expected value" {
  local failures=""
  local phrase skill track budget track_file gate actual
  while IFS=$'\t' read -r phrase skill track budget; do
    [ "$track" = "-" ] && continue
    [ "$budget" = "-" ] && continue
    track_file="$PLUGIN_ROOT/${track%:*}"
    gate="${track##*:}"
    [ -f "$track_file" ] || continue  # reported by the track-existence test
    actual=$(gate_budget "$track_file" "$gate")
    if [ -z "$actual" ]; then
      failures="$failures
  no '- budget:' knob with an integer found in gate '$gate' of ${track%:*}"
    elif [ "$actual" -ne "$budget" ]; then
      failures="$failures
  gate '$gate' in ${track%:*}: budget is $actual, fixture expects $budget"
    fi
  done < <(tsv_rows)
  [ -z "$failures" ] || fail "budget knob regressions:$failures"
}

@test "shared auto-mode question ceiling in elicitation-contract.md is exactly 5" {
  # Pins the acceptance criterion "a vague request stays within the 5-question
  # ceiling across all gates": every command draws auto-mode questions from
  # this single shared ceiling.
  local contract="$PLUGIN_ROOT/skills/_shared/elicitation-contract.md"
  [ -f "$contract" ] || fail "missing $contract"
  local ceiling
  ceiling=$(sed -n 's/.*hard ceiling of \([0-9][0-9]*\) question.*/\1/p' "$contract")
  [ -n "$ceiling" ] \
    || fail "could not extract 'hard ceiling of N questions' from $contract — wording changed?"
  [ "$(printf '%s\n' "$ceiling" | wc -l | tr -d ' ')" -eq 1 ] \
    || fail "multiple 'hard ceiling of N questions' statements in $contract: $ceiling"
  [ "$ceiling" -eq 5 ] || fail "auto-mode ceiling in $contract is $ceiling, expected 5"
}

@test "all four v2 skills have non-empty descriptions and distinct names" {
  # Cheap sanity the tsv routing checks depend on: an empty description can
  # match nothing, and duplicate name: fields would break host command routing.
  local skill desc name names=""
  for skill in $V2_SKILLS; do
    desc=$(skill_description "$skill" | sed 's/^description:[[:space:]]*//' | tr -d '"')
    [ -n "$desc" ] || fail "$skill/SKILL.md frontmatter description is empty or missing"
    name=$(awk '/^---$/ { if (++d == 2) exit; next }
                d == 1 && /^name:/ { sub(/^name:[[:space:]]*/, ""); print; exit }' \
      "$PLUGIN_ROOT/skills/$skill/SKILL.md")
    [ -n "$name" ] || fail "$skill/SKILL.md frontmatter name: is empty or missing"
    names="$names$name
"
  done
  local distinct
  distinct=$(printf '%s' "$names" | sort -u | wc -l | tr -d ' ')
  [ "$distinct" -eq 4 ] || fail "expected 4 distinct v2 skill names, got: $(printf '%s' "$names" | tr '\n' ' ')"
}
