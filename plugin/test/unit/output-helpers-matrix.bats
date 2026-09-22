#!/usr/bin/env bats
# Completeness matrix for normalize-stdin.sh output helpers: every core host
# branch of every helper must produce non-empty output in its declared shape.
# Catches the silent-fall-through class of bug (a host missing from a case
# statement prints nothing). The per-host exact-output tests in
# normalize-stdin.bats remain the precision layer.

setup() {
  load '../helpers/common'
  common_setup
}

HOSTS="claude-code cursor codex copilot opencode"

run_helper() {
  local host="$1" helper="$2"
  run sh -c "printf '%s' '{}' | ARCHCORE_HOST='$host' sh -c '
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    $helper \"test message\"
  '"
}

expected_shape_ok() {
  local host="$1" out="$2"
  case "$host" in
    claude-code|codex) [[ "$out" == *'"hookSpecificOutput"'* ]] ;;
    cursor)            [[ "$out" == *'"additional_context"'* ]] ;;
    copilot)           [[ "$out" == *'"additionalContext"'* && "$out" != *'"hookSpecificOutput"'* ]] ;;
    opencode)          [[ "$out" == "test message"* && "$out" != *'{'* ]] ;;
  esac
}

@test "output matrix: archcore_hook_info emits non-empty, host-shaped output for every host" {
  local host
  for host in $HOSTS; do
    run_helper "$host" archcore_hook_info
    assert_success
    [ -n "$output" ] || fail "archcore_hook_info produced empty output for host '$host' (silent fall-through)"
    expected_shape_ok "$host" "$output" \
      || fail "archcore_hook_info output for '$host' does not match its declared shape: $output"
  done
}

@test "output matrix: archcore_hook_pretool_info emits non-empty, host-shaped output for every host" {
  local host
  for host in $HOSTS; do
    run_helper "$host" archcore_hook_pretool_info
    assert_success
    [ -n "$output" ] || fail "archcore_hook_pretool_info produced empty output for host '$host' (silent fall-through)"
    expected_shape_ok "$host" "$output" \
      || fail "archcore_hook_pretool_info output for '$host' does not match its declared shape: $output"
  done
}

@test "output matrix: archcore_hook_block deny semantics per host" {
  local host
  for host in $HOSTS; do
    run sh -c "printf '%s' '{}' | ARCHCORE_HOST='$host' sh -c '
      . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
      archcore_hook_block \"blocked reason\"
    ' 2>&1"
    case "$host" in
      copilot)
        assert_success
        [[ "$output" == *'"permissionDecision":"deny"'* ]] \
          || fail "copilot block must emit permissionDecision deny JSON, got: $output"
        ;;
      *)
        assert_failure 2
        [[ "$output" == *"blocked reason"* ]] \
          || fail "$host block must carry the reason on stderr, got: $output"
        ;;
    esac
  done
}
