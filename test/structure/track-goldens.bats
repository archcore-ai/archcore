#!/usr/bin/env bats
# Structure tests: golden checks for gate prose in track files.
#
# Pins the load-bearing gate fields of every track under
# plugins/archcore/skills/_shared/tracks/ — routing structure (Next targets),
# skip paths (skip_when), budgets, taxonomy knobs, and Produces type/status —
# so any silent drift in gate semantics fails CI. The normalized records come
# from test/helpers/extract-gates.sh; the pinned values live in
# test/fixtures/goldens/<track>.golden.
#
# Regenerating a golden after an INTENTIONAL gate-prose change:
#   1. Re-run the extractor over the changed track:
#        test/helpers/extract-gates.sh \
#          plugins/archcore/skills/_shared/tracks/<track>.md \
#          > test/fixtures/goldens/<track>.golden
#   2. Review `git diff test/fixtures/goldens/<track>.golden`. Every changed
#      line must correspond to the semantic change you meant to make (a gate
#      added or renamed, a routing target, a skip path, a budget, a taxonomy
#      category, a produced type). An unexpected changed line means the prose
#      edit silently altered gate semantics — fix the track, not the golden.
#   3. Commit the regenerated golden together with the track change.
#
# The cross-track tests below are computed live from the track files (not from
# the goldens), so they hold even while a golden is being regenerated.

setup() {
  load '../helpers/common'
  common_setup
  EXTRACT="$REPO_ROOT/test/helpers/extract-gates.sh"
  TRACKS_DIR="$PLUGIN_ROOT/skills/_shared/tracks"
  GOLDENS_DIR="$FIXTURES/goldens"
  ALL_TRACKS="decision sdd requirements-cascade describe actualize experience"
}

# Diff the extractor's live output for one track against its golden; show the
# full diff on failure.
assert_track_golden() {
  local track="$1"
  local golden="$GOLDENS_DIR/${track}.golden"
  local actual="$BATS_TEST_TMPDIR/${track}.actual"
  local diff_out="$BATS_TEST_TMPDIR/${track}.diff"
  [ -f "$golden" ] || fail "missing golden: test/fixtures/goldens/${track}.golden"
  "$EXTRACT" "$TRACKS_DIR/${track}.md" > "$actual"
  if ! diff -u "$golden" "$actual" > "$diff_out"; then
    fail "gate prose in skills/_shared/tracks/${track}.md drifted from test/fixtures/goldens/${track}.golden (intentional change? see regeneration procedure in this file's header):
$(cat "$diff_out")"
  fi
}

@test "decision track gate records match golden" {
  assert_track_golden decision
}

@test "sdd track gate records match golden" {
  assert_track_golden sdd
}

@test "requirements-cascade track gate records match golden" {
  assert_track_golden requirements-cascade
}

@test "describe track gate records match golden" {
  assert_track_golden describe
}

@test "actualize track gate records match golden" {
  assert_track_golden actualize
}

@test "experience track gate records match golden" {
  assert_track_golden experience
}

@test "every Next target resolves to a gate in the same track or a terminal marker" {
  local bad=""
  local t out gates gname targets tok
  for t in $ALL_TRACKS; do
    out=$("$EXTRACT" "$TRACKS_DIR/${t}.md")
    gates=$(printf '%s\n' "$out" | awk '/^gate: /{print $2}')
    while IFS=$(printf '\t') read -r gname targets; do
      if [ -z "$targets" ]; then
        bad="$bad ${t}/${gname}(Next names no gate and no exit)"
        continue
      fi
      for tok in $targets; do
        [ "$tok" = "end" ] && continue
        if ! printf '%s\n' "$gates" | grep -Fqx "$tok"; then
          bad="$bad ${t}/${gname}(target ${tok} has no '### gate: ${tok}' heading)"
        fi
      done
    done < <(printf '%s\n' "$out" \
      | awk '/^gate: /{g=$2} /^next:/{s=$0; sub(/^next: */,"",s); printf "%s\t%s\n", g, s}')
  done
  [ -z "$bad" ] || fail "unresolvable Next targets in skills/_shared/tracks/:$bad"
}

@test "every gate carries the six contract fields in template order" {
  # Field order is fixed by skills/_shared/gate-contract.md ("Gate record
  # template"): Purpose, Entry conditions, Elicitation knobs, Produces,
  # Exit checks, Next.
  local expected="Purpose|Entry conditions|Elicitation knobs|Produces|Exit checks|Next"
  local bad=""
  local t gname seq
  for t in $ALL_TRACKS; do
    while IFS=$(printf '\t') read -r gname seq; do
      if [ "$seq" != "$expected" ]; then
        bad="$bad ${t}/${gname}(got: ${seq})"
      fi
    done < <(awk '
      function flushg() { if (g != "") printf "%s\t%s\n", g, seq; g = ""; seq = "" }
      /^#/ {
        flushg()
        if ($0 ~ /^### gate:/) { g = $0; sub(/^### gate:[ \t]*/, "", g); sub(/[ \t]+$/, "", g) }
        next
      }
      g != "" && /^- [A-Za-z]/ {
        lbl = $0; sub(/^- /, "", lbl); sub(/:.*/, "", lbl)
        seq = seq (seq == "" ? "" : "|") lbl
      }
      END { flushg() }
    ' "$TRACKS_DIR/${t}.md")
  done
  [ -z "$bad" ] || fail "gate records out of template order (expected ${expected}):$bad"
}

@test "every gate declares a budget knob with a question maximum" {
  local bad=""
  local t gname budget
  for t in $ALL_TRACKS; do
    while IFS=$(printf '\t') read -r gname budget; do
      case "$budget" in
        [0-9]*) ;;
        *) bad="$bad ${t}/${gname}(budget: '${budget}')" ;;
      esac
    done < <("$EXTRACT" "$TRACKS_DIR/${t}.md" \
      | awk '/^gate: /{g=$2} /^budget:/{s=$0; sub(/^budget: */,"",s); printf "%s\t%s\n", g, s}')
  done
  [ -z "$bad" ] || fail "gates without a numeric budget knob in skills/_shared/tracks/:$bad"
}

@test "gate-contract state-block template lists exactly the six state fields" {
  local contract="$PLUGIN_ROOT/skills/_shared/gate-contract.md"
  local got
  got=$(awk '
    /<!-- archcore:track/ { f = 1; next }
    f && /-->/ { f = 0; next }
    f && /^[a-z_]+:/ { n = $0; sub(/:.*/, "", n); out = (out == "" ? n : out " " n) }
    END { print out }
  ' "$contract")
  [ "$got" = "track gate taxonomy asked budget deferred" ] \
    || fail "state-block template in skills/_shared/gate-contract.md must list exactly 'track gate taxonomy asked budget deferred' in order; got: '${got}'"
}
