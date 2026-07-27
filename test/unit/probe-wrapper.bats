#!/usr/bin/env bats
# The probe wrapper is transparent.
#
# mkprobe replaces each guard with a logging wrapper that execs the original.
# Everything a probe run concludes rests on that wrapper being invisible: if it
# swallowed a stderr line, a probe would record "the host showed no reason" for
# a guard that did produce one; if it flattened a non-zero status, probe B would
# record a deny that never happened. A green probe would then be an artifact of
# the harness rather than evidence about the host — worse than running no probe
# at all, because it gets written down as fact.
#
# So the wrapper is compared against the untouched script across every stdin
# fixture in the suite: same stdout, same stderr, same exit status. The set
# includes malformed payloads on purpose — fail-open paths are where a wrapper
# is most likely to differ.

setup() {
  load '../helpers/common'
  common_setup

  PROBE_TREE="$BATS_TEST_TMPDIR/probe"
  run env REPO_ROOT="$REPO_ROOT" "$REPO_ROOT/test/probe/mkprobe" "$PROBE_TREE"
  assert_success

  WORK="$BATS_TEST_TMPDIR/work"
  make_project "$WORK"
}

# A project with one document, at a path of its own.
#
# Every invocation gets a fresh project, because `archcore hooks session-start`
# emits its context only the FIRST time it sees a given project — run it twice
# and the second run is legitimately empty. Comparing the original and the
# wrapped script inside one project would therefore always "prove" the wrapper
# ate the output. That is a property of the CLI, not of the wrapper, and it
# cost a false failure to find.
make_project() {
  local dir="$1"
  mkdir -p "$dir/src/probe" "$dir/.archcore/probe"
  cat > "$dir/.archcore/probe/r.rule.md" <<'EOF'
---
title: "Probe rule"
status: accepted
---

Applies to src/probe/ paths.
EOF
}

# Run one script against one fixture in a project and state directory of its
# own, printing a single comparable record: exit status, stdout, stderr.
capture() {
  local script="$1" fixture="$2" slot="$3"
  local project="$BATS_TEST_TMPDIR/proj-$slot" state="$BATS_TEST_TMPDIR/state-$slot"
  local out err status
  make_project "$project"
  mkdir -p "$state"
  out=$(mktemp "$BATS_TEST_TMPDIR/out.XXXXXX")
  err=$(mktemp "$BATS_TEST_TMPDIR/err.XXXXXX")
  (
    cd "$project" || exit 99
    CLAUDE_PLUGIN_DATA="$state" "$script" < "$fixture" > "$out" 2> "$err"
  )
  status=$?
  printf 'exit=%s\n--stdout--\n%s\n--stderr--\n%s\n' \
    "$status" "$(cat "$out")" "$(cat "$err")"
  rm -f "$out" "$err"
}

@test "wrapped guards match the originals byte for byte on every fixture" {
  local guard fixture n=0 name real_out wrapped_out
  for guard in session-start check-archcore-write check-code-alignment validate-archcore; do
    [ -f "$PROBE_TREE/plugin-src/bin/$guard.real" ] \
      || fail "$guard was not wrapped by mkprobe"
    while IFS= read -r fixture; do
      name="$(basename "$(dirname "$fixture")")/$(basename "$fixture")"
      real_out=$(capture "$PLUGIN_ROOT/bin/$guard" "$fixture" "real-$n")
      wrapped_out=$(capture "$PROBE_TREE/plugin-src/bin/$guard" "$fixture" "wrapped-$n")
      n=$((n + 1))
      [ "$real_out" = "$wrapped_out" ] || {
        echo "guard:   $guard"
        echo "fixture: $name"
        echo "--- original ---"; echo "$real_out"
        echo "--- wrapped ----"; echo "$wrapped_out"
        fail "the probe wrapper is not transparent for $guard on $name"
      }
    done < <(find "$FIXTURES/stdin" -name '*.json' | sort)
  done
  [ "$n" -gt 0 ] || fail "no fixtures were compared"
}

@test "the wrapper records every invocation it forwards" {
  # Transparency is only half the contract — a wrapper that passes everything
  # through but logs nothing leaves a probe run with no evidence to cite.
  local log="$PROBE_TREE/probe.log"
  : > "$log"
  (
    cd "$WORK" || exit 1
    printf '%s' '{"tool_name":"Write","tool_input":{"file_path":".archcore/probe/p.adr.md"}}' \
      | ARCHCORE_PROBE_LOG="$log" "$PROBE_TREE/plugin-src/bin/check-archcore-write" >/dev/null 2>&1
  ) || true

  [ -s "$log" ] || fail "the wrapper forwarded the call but logged nothing"
  grep -q 'check-archcore-write' "$log" || fail "log does not name the guard: $(cat "$log")"
  # The captured payload is what turns a probe run into a stdin fixture.
  grep -q 'p.adr.md' "$log" || fail "log does not carry the payload: $(cat "$log")"
}

@test "the wrapper is silent when no log is configured" {
  # A probe tree left behind must not start writing into whatever path a stale
  # ARCHCORE_PROBE_LOG happens to name — absent the variable, the log goes to
  # /dev/null and the guard behaves exactly as shipped.
  local out
  out=$(cd "$WORK" && printf '%s' '{"tool_name":"Write","tool_input":{"file_path":"src/probe/x.ts"}}' \
    | "$PROBE_TREE/plugin-src/bin/check-archcore-write" 2>&1)
  [ -z "$out" ] || fail "unexpected output with no log configured: $out"
}
