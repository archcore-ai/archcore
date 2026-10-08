#!/usr/bin/env bats
# Harness integrity uses a fake model process; live model quality is a separate target.

setup() {
  load '../helpers/common'
  common_setup
  export REVIEW_BENCH_FIXTURES="$BATS_TEST_TMPDIR/fixtures.tsv"
  export REVIEW_BENCH_OUTPUT_DIR="$BATS_TEST_TMPDIR/results"
  export BENCH_ARGS="$BATS_TEST_TMPDIR/args"
  unset REVIEW_BENCH_LIMIT REVIEW_BENCH_MODEL BENCH_REPLY_MODE
  printf 'spec\tPublic API, no owner\tcreated=spec;plan=none\nask\tPreview needed\tquestions=>0;plan=none\n' > "$REVIEW_BENCH_FIXTURES"
  cat > "$MOCK_BIN/claude" <<'MOCK'
#!/bin/sh
printf '%s\n' "$@" >> "$BENCH_ARGS"
printf -- '--- call ---\n' >> "$BENCH_ARGS"
prompt=$(cat)
case "${BENCH_REPLY_MODE:-}" in
  error) echo 'host unavailable' >&2; exit 7 ;;
  wrong) plan=offer ;;
  *) plan=none ;;
esac
case "$prompt" in *'Preview needed'*) q=1 ;; *) q=0 ;; esac
reply="review-verdict: created=spec; updated=0; omitted=1; questions=$q; plan=$plan"
case "${BENCH_REPLY_MODE:-}" in
  multiline) reply=$(printf 'Explanation\n%s' "$reply") ;;
  inline) reply='`'"$reply"'`' ;;
  missing) reply="review-verdict: created=spec; questions=$q; plan=$plan" ;;
esac
jq -cn --arg result "$reply" '{is_error:false,result:$result}'
MOCK
  chmod +x "$MOCK_BIN/claude"
  BENCH="$REPO_ROOT/test/behavioral/review-bench.sh"
}

@test "review bench evaluates every fixture against the review contracts and retains raw replies" {
  run sh "$BENCH"
  assert_success
  assert_output --partial '2 pass, 0 fail, 0 errors'
  [ -s "$REVIEW_BENCH_OUTPUT_DIR/2.json" ] || { fail "second model response was discarded"; return 1; }
  grep -Fq -- '--- CONTRACT: skills/_shared/durable-context-selection.md ---' "$REVIEW_BENCH_OUTPUT_DIR/1.prompt" \
    || { fail "prompt lacks the selection procedure"; return 1; }
  assert_equal "$(grep -Fxc -- '--- call ---' "$BENCH_ARGS")" "$(grep -Fxc -- '--strict-mcp-config' "$BENCH_ARGS")" \
    || { fail "a bench call can load project MCP servers"; return 1; }
}

@test "review bench rejects a wrong review verdict instead of reporting green" {
  export BENCH_REPLY_MODE=wrong
  run sh "$BENCH"
  assert_equal "$status" 1
  assert_output --partial '0 pass, 2 fail, 0 errors'
}

@test "review bench rejects zero questions where the fixture expects at least one" {
  printf 'ask\tno preview marker\tquestions=>0\n' > "$REVIEW_BENCH_FIXTURES"
  run sh "$BENCH"
  assert_equal "$status" 1
  assert_output --partial '0 pass, 1 fail, 0 errors'
}

@test "review bench rejects multiline replies and replies missing a key" {
  export BENCH_REPLY_MODE=multiline
  run sh "$BENCH"
  assert_equal "$status" 1
  export BENCH_REPLY_MODE=missing
  run sh "$BENCH"
  assert_equal "$status" 1
  assert_output --partial '0 pass, 2 fail, 0 errors'
}

@test "review bench distinguishes a CLI failure from a verdict mismatch" {
  export BENCH_REPLY_MODE=error
  run sh "$BENCH"
  assert_equal "$status" 2
  assert_output --partial '0 pass, 0 fail, 2 errors'
}

@test "review bench rejects malformed fixtures and invalid limits before any model call" {
  printf '1\tSituation\tplan=maybe\n' > "$REVIEW_BENCH_FIXTURES"
  run sh "$BENCH"
  assert_equal "$status" 2
  [ ! -e "$BENCH_ARGS" ] || { fail "malformed corpus invoked the model"; return 1; }
  export REVIEW_BENCH_LIMIT=invalid
  run sh "$BENCH"
  assert_equal "$status" 2
}

@test "review bench accepts a single inline-code verdict and honours the limit" {
  export BENCH_REPLY_MODE=inline REVIEW_BENCH_LIMIT=1
  run sh "$BENCH"
  assert_success
  assert_output --partial '1 pass, 0 fail, 0 errors'
}

@test "shipped review bench fixtures pass the corpus validator" {
  export REVIEW_BENCH_FIXTURES="$REPO_ROOT/test/behavioral/fixtures/review-bench.tsv" REVIEW_BENCH_LIMIT=1
  run sh "$BENCH"
  [ "$status" -ne 2 ] || { fail "shipped fixtures rejected: $output"; return 1; }
}

@test "review bench repeats each fixture REVIEW_BENCH_REPS times under separate ids" {
  export REVIEW_BENCH_REPS=2
  run sh "$BENCH"
  assert_success
  assert_output --partial '4 pass, 0 fail, 0 errors'
  assert_output --partial 'spec#2'
  [ -s "$REVIEW_BENCH_OUTPUT_DIR/1.2.json" ] || { fail "second repetition reply was discarded"; return 1; }
  export REVIEW_BENCH_REPS=0
  run sh "$BENCH"
  assert_equal "$status" 2
}

@test "review bench reads the contracts from REVIEW_BENCH_REF and rejects an unknown ref" {
  export REVIEW_BENCH_REF=HEAD REVIEW_BENCH_LIMIT=1
  run sh "$BENCH"
  assert_success
  assert_output --partial '(claude @ HEAD)'
  grep -Fq -- '--- CONTRACT: skills/_shared/tracks/closeout.md ---' "$REVIEW_BENCH_OUTPUT_DIR/1.prompt" \
    || { fail "prompt lacks the closeout track at the ref"; return 1; }
  rm -f "$BENCH_ARGS"
  export REVIEW_BENCH_REF=no-such-ref-for-review-bench
  run sh "$BENCH"
  assert_equal "$status" 2
  [ ! -e "$BENCH_ARGS" ] || { fail "an unreadable ref invoked the model"; return 1; }
}

@test "review bench dispatches to codex and copilot and rejects an unknown host" {
  cat > "$MOCK_BIN/codex" <<'MOCK'
#!/bin/sh
while [ $# -gt 0 ]; do [ "$1" = -o ] && out=$2; shift; done
cat > /dev/null
printf 'review-verdict: created=spec; updated=0; omitted=0; questions=1; plan=none\n' > "$out"
MOCK
  cat > "$MOCK_BIN/fake-copilot" <<'MOCK'
#!/bin/sh
cat > /dev/null
printf '\nreview-verdict: created=spec; updated=0; omitted=0; questions=1; plan=none\n'
MOCK
  chmod +x "$MOCK_BIN/codex" "$MOCK_BIN/fake-copilot"
  export REVIEW_BENCH_HOST=codex
  run sh "$BENCH"
  assert_success
  assert_output --partial '(codex): 2 pass'
  export REVIEW_BENCH_HOST=copilot REVIEW_BENCH_COPILOT=fake-copilot
  run sh "$BENCH"
  assert_success
  assert_output --partial '(copilot): 2 pass'
  [ ! -e "$BENCH_ARGS" ] || { fail "a non-claude host invoked claude"; return 1; }
  export REVIEW_BENCH_HOST=cursor
  run sh "$BENCH"
  assert_equal "$status" 2
}
