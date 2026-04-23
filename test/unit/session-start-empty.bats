#!/usr/bin/env bats
# Tests for bin/session-start empty-state nudge (A1 of zero-content onboarding).

setup() {
  load '../helpers/common'
  common_setup
}

# Mock archcore CLI that silently swallows stdin for the `hooks` subcommand.
_silent_hooks_mock() {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
if [ "$1" = "hooks" ]; then
  cat > /dev/null
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"
}

@test "missing .archcore/ — nudge mentions /archcore:bootstrap" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "no .archcore/ directory"
  assert_output --partial "mcp__archcore__init_project"
  assert_output --partial "/archcore:bootstrap"
}

@test "missing .archcore/ with ARCHCORE_HIDE_EMPTY_NUDGE=1 — init nudge stays, bootstrap hint suppressed" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"

  run sh -c "printf '%s' '{}' | ARCHCORE_HIDE_EMPTY_NUDGE=1 '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "mcp__archcore__init_project"
  refute_output --partial "/archcore:bootstrap"
}

@test ".archcore/ with only sub-200-byte stubs — bootstrap nudge fires" {
  _silent_hooks_mock

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  : > "$workdir/.archcore/.gitkeep"
  printf 'stub\n' > "$workdir/.archcore/stub.md"
  cd "$workdir"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "/archcore:bootstrap"
  assert_output --partial "ARCHCORE_HIDE_EMPTY_NUDGE"
}

@test ".archcore/ with a substantial (>200B) document — bootstrap nudge absent" {
  _silent_hooks_mock

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  # 300-byte markdown file — above the 200-byte threshold.
  printf -- '---\ntitle: seeded\nstatus: accepted\n---\n\n' > "$workdir/.archcore/seeded.md"
  # Pad the body to comfortably exceed 200 bytes total.
  yes 'lorem ipsum dolor sit amet. ' 2>/dev/null | head -c 300 >> "$workdir/.archcore/seeded.md"
  cd "$workdir"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  refute_output --partial "/archcore:bootstrap"
}

@test "ARCHCORE_HIDE_EMPTY_NUDGE=1 suppresses the nudge on exists-but-empty .archcore/" {
  _silent_hooks_mock

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"

  run sh -c "printf '%s' '{}' | ARCHCORE_HIDE_EMPTY_NUDGE=1 '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  refute_output --partial "/archcore:bootstrap"
}
