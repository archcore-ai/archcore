#!/usr/bin/env bats
# Harness integrity uses a fake model process; live model quality is a separate target.

setup() {
  load '../helpers/common'
  common_setup
  export GATE_BENCH_FIXTURES="$BATS_TEST_TMPDIR/fixtures.tsv"
  export GATE_BENCH_OUTPUT_DIR="$BATS_TEST_TMPDIR/results"
  export BENCH_ARGS="$BATS_TEST_TMPDIR/args"
  unset GATE_BENCH_LIMIT GATE_BENCH_MODEL BENCH_REPLY_MODE
  printf 'open\tUser interrupted\tgate=open;persist=yes\nask\tPrecedent only\tquestions=>0;gate=open\n' > "$GATE_BENCH_FIXTURES"
  cat > "$MOCK_BIN/claude" <<'MOCK'
#!/bin/sh
printf '%s\n' "$@" >> "$BENCH_ARGS"
printf -- '--- call ---\n' >> "$BENCH_ARGS"
prompt=$(cat)
case "${BENCH_REPLY_MODE:-}" in
  error) echo 'host unavailable' >&2; exit 7 ;;
  wrong) gate=advance ;;
  *) gate=open ;;
esac
case "$prompt" in *'Precedent only'*) q=2 ;; *) q=0 ;; esac
reply="gate-verdict: skip=no; resume=none; questions=$q; gate=$gate; assumptions=0; persist=yes"
case "${BENCH_REPLY_MODE:-}" in
  multiline) reply=$(printf 'Explanation\n%s' "$reply") ;;
  inline) reply='`'"$reply"'`' ;;
  missing) reply="gate-verdict: skip=no; questions=$q; gate=$gate" ;;
esac
jq -cn --arg result "$reply" '{is_error:false,result:$result}'
MOCK
  chmod +x "$MOCK_BIN/claude"
  BENCH="$REPO_ROOT/test/behavioral/gate-bench.sh"
}

@test "gate bench evaluates every fixture against the gate contracts and retains raw replies" {
  run sh "$BENCH"
  assert_success
  assert_output --partial '2 pass, 0 fail, 0 errors'
  [ -s "$GATE_BENCH_OUTPUT_DIR/2.json" ] || { fail "second model response was discarded"; return 1; }
  grep -Fq -- '--- CONTRACT: skills/_shared/gate-contract.md ---' "$GATE_BENCH_OUTPUT_DIR/1.prompt" \
    || { fail "prompt lacks the gate contract"; return 1; }
  assert_equal "$(grep -Fxc -- '--- call ---' "$BENCH_ARGS")" "$(grep -Fxc -- '--strict-mcp-config' "$BENCH_ARGS")" \
    || { fail "a bench call can load project MCP servers"; return 1; }
}

@test "gate bench rejects a wrong gate verdict instead of reporting green" {
  export BENCH_REPLY_MODE=wrong
  run sh "$BENCH"
  assert_equal "$status" 1
  assert_output --partial '0 pass, 2 fail, 0 errors'
}

@test "gate bench rejects zero questions where the fixture expects at least one" {
  printf 'ask\tno precedent marker\tquestions=>0\n' > "$GATE_BENCH_FIXTURES"
  run sh "$BENCH"
  assert_equal "$status" 1
  assert_output --partial '0 pass, 1 fail, 0 errors'
}

@test "gate bench rejects multiline replies and replies missing a key" {
  export BENCH_REPLY_MODE=multiline
  run sh "$BENCH"
  assert_equal "$status" 1
  export BENCH_REPLY_MODE=missing
  run sh "$BENCH"
  assert_equal "$status" 1
  assert_output --partial '0 pass, 2 fail, 0 errors'
}

@test "gate bench distinguishes a CLI failure from a verdict mismatch" {
  export BENCH_REPLY_MODE=error
  run sh "$BENCH"
  assert_equal "$status" 2
  assert_output --partial '0 pass, 0 fail, 2 errors'
}

@test "gate bench rejects malformed fixtures and invalid limits before any model call" {
  printf '1\tSituation\tgate=maybe\n' > "$GATE_BENCH_FIXTURES"
  run sh "$BENCH"
  assert_equal "$status" 2
  [ ! -e "$BENCH_ARGS" ] || { fail "malformed corpus invoked the model"; return 1; }
  export GATE_BENCH_LIMIT=invalid
  run sh "$BENCH"
  assert_equal "$status" 2
}

@test "gate bench accepts a single inline-code verdict and honours the limit" {
  export BENCH_REPLY_MODE=inline GATE_BENCH_LIMIT=1
  run sh "$BENCH"
  assert_success
  assert_output --partial '1 pass, 0 fail, 0 errors'
}

@test "shipped gate bench fixtures pass the corpus validator" {
  export GATE_BENCH_FIXTURES="$REPO_ROOT/test/behavioral/fixtures/gate-bench.tsv" GATE_BENCH_LIMIT=1
  run sh "$BENCH"
  [ "$status" -ne 2 ] || { fail "shipped fixtures rejected: $output"; return 1; }
}

@test "gate bench repeats each fixture GATE_BENCH_REPS times under separate ids" {
  export GATE_BENCH_REPS=2
  run sh "$BENCH"
  assert_success
  assert_output --partial '4 pass, 0 fail, 0 errors'
  assert_output --partial 'open#2'
  [ -s "$GATE_BENCH_OUTPUT_DIR/1.2.json" ] || { fail "second repetition reply was discarded"; return 1; }
  export GATE_BENCH_REPS=0
  run sh "$BENCH"
  assert_equal "$status" 2
}

@test "gate bench reads the contracts from GATE_BENCH_REF and rejects an unknown ref" {
  export GATE_BENCH_REF=HEAD GATE_BENCH_LIMIT=1
  run sh "$BENCH"
  assert_success
  assert_output --partial '(claude @ HEAD)'
  grep -Fq -- '--- CONTRACT: skills/_shared/tracks/sdd.md ---' "$GATE_BENCH_OUTPUT_DIR/1.prompt" \
    || { fail "prompt lacks the sdd track at the ref"; return 1; }
  rm -f "$BENCH_ARGS"
  export GATE_BENCH_REF=no-such-ref-for-gate-bench
  run sh "$BENCH"
  assert_equal "$status" 2
  [ ! -e "$BENCH_ARGS" ] || { fail "an unreadable ref invoked the model"; return 1; }
}

@test "gate bench dispatches to codex and copilot and rejects an unknown host" {
  cat > "$MOCK_BIN/codex" <<'MOCK'
#!/bin/sh
while [ $# -gt 0 ]; do [ "$1" = -o ] && out=$2; shift; done
cat > /dev/null
printf 'gate-verdict: skip=no; resume=none; questions=2; gate=open; assumptions=0; persist=yes\n' > "$out"
MOCK
  cat > "$MOCK_BIN/fake-copilot" <<'MOCK'
#!/bin/sh
cat > /dev/null
printf '\ngate-verdict: skip=no; resume=none; questions=2; gate=open; assumptions=0; persist=yes\n'
MOCK
  chmod +x "$MOCK_BIN/codex" "$MOCK_BIN/fake-copilot"
  export GATE_BENCH_HOST=codex
  run sh "$BENCH"
  assert_success
  assert_output --partial '(codex): 2 pass'
  export GATE_BENCH_HOST=copilot GATE_BENCH_COPILOT=fake-copilot
  run sh "$BENCH"
  assert_success
  assert_output --partial '(copilot): 2 pass'
  [ ! -e "$BENCH_ARGS" ] || { fail "a non-claude host invoked claude"; return 1; }
  export GATE_BENCH_HOST=cursor
  run sh "$BENCH"
  assert_equal "$status" 2
}
