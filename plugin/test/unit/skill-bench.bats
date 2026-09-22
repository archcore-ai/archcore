#!/usr/bin/env bats
# Harness integrity uses a fake host process; live selection quality is a separate target.

setup() {
  load '../helpers/common'
  common_setup
  export SKILL_BENCH_FIXTURES="$BATS_TEST_TMPDIR/fixtures.tsv"
  export SKILL_BENCH_OUTPUT_DIR="$BATS_TEST_TMPDIR/results"
  export BENCH_ARGS="$BATS_TEST_TMPDIR/args"
  unset SKILL_BENCH_LIMIT SKILL_BENCH_MODEL BENCH_REPLY_MODE
  printf '1\tWe decided to use Postgres\tdocument\t-\n2\tFix the README typo\tnone\t-\n3\tAre docs stale?\treview\tdrift\n' > "$SKILL_BENCH_FIXTURES"
  cat > "$MOCK_BIN/claude" <<'MOCK'
#!/bin/sh
printf '%s\n' "$@" >> "$BENCH_ARGS"
printf -- '--- call ---\n' >> "$BENCH_ARGS"
msg=$(cat)
[ "${BENCH_REPLY_MODE:-}" = error ] && { echo 'host unavailable' >&2; exit 7; }
echo '{"type":"system","subtype":"init","skills":["archcore:document"]}'
case "${BENCH_REPLY_MODE:-}:$msg" in
  wrong:*) skill='archcore:plan'; args='x' ;;
  *Postgres*) skill='archcore:document'; args='decision we chose Postgres' ;;
  *README*) skill='simplify'; args='' ;;
  *) skill='archcore:review'; args='drift' ;;
esac
jq -cn --arg s "$skill" --arg a "$args" '{type:"assistant",message:{content:[{type:"tool_use",name:"Skill",input:{skill:$s,args:$a}}]}}'
exit 1
MOCK
  chmod +x "$MOCK_BIN/claude"
  BENCH="$REPO_ROOT/test/behavioral/skill-bench.sh"
}

@test "skill bench reads the first Skill call, checks the mode word, and lets a negative fixture pass on a built-in skill" {
  run sh "$BENCH"
  assert_success
  assert_output --partial '3 pass, 0 fail, 0 errors'
  grep -Fxq -- '--plugin-dir' "$BENCH_ARGS" || { fail "the plugin is not loaded into the host"; return 1; }
  grep -Fxq -- 'Skill' "$BENCH_ARGS" || { fail "the host is not limited to the Skill tool"; return 1; }
  grep -Fq 'disableAllHooks' "$BENCH_ARGS" || { fail "plugin hooks can bias the selection"; return 1; }
}

@test "skill bench fails a wrong skill" {
  export BENCH_REPLY_MODE=wrong
  run sh "$BENCH"
  assert_equal "$status" 1
  assert_output --partial '0 pass, 3 fail, 0 errors'
}

@test "skill bench reports a host with no transcript as an error" {
  export BENCH_REPLY_MODE=error
  run sh "$BENCH"
  assert_equal "$status" 2
  assert_output --partial '0 pass, 0 fail, 3 errors'
}

@test "skill bench rejects an unknown skill or mode in fixtures" {
  printf '1\tx\tdescribe\t-\n' > "$SKILL_BENCH_FIXTURES"
  run sh "$BENCH"
  assert_equal "$status" 2
  printf '1\tx\treview\t--drift\n' > "$SKILL_BENCH_FIXTURES"
  run sh "$BENCH"
  assert_equal "$status" 2
  [ ! -e "$BENCH_ARGS" ] || { fail "an invalid corpus invoked the host"; return 1; }
}
