#!/usr/bin/env bats
# Diagnostic stderr must not follow a predictable symlink into another file.

setup() {
  load '../helpers/common'
  common_setup
  mkdir -p "$BATS_TEST_TMPDIR/project/.archcore" "$BATS_TEST_TMPDIR/errors"
  cd "$BATS_TEST_TMPDIR/project"
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/cli-args"
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
printf '%s\n' "$*" >> "$MOCK_ARCHCORE_LOG"
if [ "$1" = hooks ]; then
  echo CLI_DIAGNOSTIC >&2
  exit 1
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"
}

@test "session-start diagnostics do not overwrite a predictable symlink target" {
  local victim="$BATS_TEST_TMPDIR/victim"
  printf 'preserve me\n' > "$victim"
  # exec keeps the shell PID, making the old predictable path reproducible.
  run env TMPDIR="$BATS_TEST_TMPDIR/errors" sh -c '
    ln -s "$1" "$TMPDIR/archcore_ss_err_$$"
    exec "$2"
  ' sh "$victim" "$PLUGIN_ROOT/bin/session-start" <<< '{}'
  assert_success
  assert_equal "$(cat "$victim")" 'preserve me'
  grep -Fxq 'hooks claude-code session-start' "$MOCK_ARCHCORE_LOG" \
    || { fail "the real diagnostic path was not reached"; return 1; }
  local leftover
  leftover=$(find "$BATS_TEST_TMPDIR/errors" -type f -name 'archcore_ss_err.*')
  assert_equal "$leftover" ''
}

@test "session-start still runs the CLI when a private diagnostic file cannot be allocated" {
  cat > "$MOCK_BIN/mktemp" <<'MOCK'
#!/bin/sh
exit 1
MOCK
  chmod +x "$MOCK_BIN/mktemp"
  run "$PLUGIN_ROOT/bin/session-start" <<< '{}'
  assert_success
  grep -Fxq 'hooks claude-code session-start' "$MOCK_ARCHCORE_LOG" \
    || { fail "CLI not invoked when diagnostic storage is unavailable"; return 1; }
  refute_output --partial 'CLI_DIAGNOSTIC'
  local leftover
  leftover=$(find "$BATS_TEST_TMPDIR" -type f -name 'archcore_ss_err*')
  assert_equal "$leftover" ''
}

@test "session-start emits the empty-state nudge when diagnostics cannot be captured" {
  cat > "$MOCK_BIN/mktemp" <<'MOCK'
#!/bin/sh
exit 1
MOCK
  chmod +x "$MOCK_BIN/mktemp"
  run "$PLUGIN_ROOT/bin/session-start" <<< '{}'
  assert_success
  assert_output --partial 'Archcore'
}
