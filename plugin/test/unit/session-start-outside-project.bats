#!/usr/bin/env bats
# bin/session-start in a folder without .archcore/ (explicit-target-project-per-call.adr,
# hooks-validation-system.spec Hook 1 phase 2): a CLI from 0.15.0 answers with
# the project inventory; an older CLI gets the plugin's own message. Neither
# advises initializing the folder.

setup() {
  load '../helpers/common'
  common_setup
}

# mock_cli_version VERSION — a CLI that reports VERSION, logs its calls, and
# answers `hooks` with an inventory marker.
mock_cli_version() {
  cat > "$MOCK_BIN/archcore" <<MOCK
#!/bin/sh
[ -n "\$MOCK_ARCHCORE_LOG" ] && printf '%s\n' "\$*" >> "\$MOCK_ARCHCORE_LOG"
case "\$1" in
  --version) echo "archcore v$1" ;;
  hooks) printf '%s\n' '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"NO PROJECT HERE: INVENTORY-FROM-CLI"}}' ;;
  *) exit 0 ;;
esac
MOCK
  chmod +x "$MOCK_BIN/archcore"
}

@test "CLI 0.15.0 outside a project: the CLI recap is delivered, the plugin adds no init advice" {
  mock_cli_version 0.15.0
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/calls.log"
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "INVENTORY-FROM-CLI"
  refute_output --partial "no .archcore/ directory in this folder"
  refute_output --partial "init_project"
  refute_output --partial ".archcore/ is empty"
  run grep -c '^hooks claude-code session-start$' "$MOCK_ARCHCORE_LOG"
  assert_output "1"
}

@test "CLI newer than 0.15.0 outside a project takes the CLI path" {
  mock_cli_version 1.2.0
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "INVENTORY-FROM-CLI"
}

@test "CLI 0.15.0 outside a project on codex: one JSON document, the codex-cli dialect" {
  mock_cli_version 0.15.0
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/calls.log"
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{\"turn_id\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "INVENTORY-FROM-CLI"
  printf '%s' "$output" | jq -e . > /dev/null || fail "codex output is not one JSON document: $output"
  run grep -c '^hooks codex-cli session-start$' "$MOCK_ARCHCORE_LOG"
  assert_output "1"
}

@test "CLI 0.14.4 outside a project: the plugin's own message, the CLI hook is not called" {
  mock_cli_version 0.14.4
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/calls.log"
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "no .archcore/ directory in this folder"
  assert_output --partial "archcore CLI 0.15.0 or newer"
  refute_output --partial "INVENTORY-FROM-CLI"
  refute_output --partial "init_project"
  run grep -c '^hooks ' "$MOCK_ARCHCORE_LOG"
  assert_output "0"
}

@test "copilot outside a project keeps its wiring message on every CLI version" {
  for v in 0.14.4 0.15.0; do
    mock_cli_version "$v"
    cd "$BATS_TEST_TMPDIR"
    run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
    assert_success
    assert_output --partial 'archcore init --agent copilot --project'
    refute_output --partial "INVENTORY-FROM-CLI"
  done
}

@test "CLI 0.15.0 inside a project: unchanged path, the CLI recap and the empty-state nudge" {
  mock_cli_version 0.15.0
  mkdir -p "$BATS_TEST_TMPDIR/p/.archcore"
  cd "$BATS_TEST_TMPDIR/p"
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "INVENTORY-FROM-CLI"
  assert_output --partial ".archcore/ is empty"
}

# A folder below an initialized project belongs to it
# (session-project-is-nearest-store.adr): Copilot gets the CLI recap of that
# project, and no advice that would create a second .archcore/ in the folder.
@test "copilot below an initialized project: the CLI recap is delivered, no init advice for the folder" {
  mock_cli_version 0.15.6
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/calls.log"
  mkdir -p "$BATS_TEST_TMPDIR/repo/.archcore" "$BATS_TEST_TMPDIR/repo/pkg/api"
  printf '{}\n' > "$BATS_TEST_TMPDIR/repo/.archcore/settings.json"
  cd "$BATS_TEST_TMPDIR/repo/pkg/api"
  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot ARCHCORE_HIDE_EMPTY_NUDGE=1 '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  refute_output --partial "no .archcore/ directory in this project yet"
  assert_output --partial "Do not pass this folder"
  refute_output --partial '--project \"$PWD\"'
  run grep -c '^hooks copilot session-start$' "$MOCK_ARCHCORE_LOG"
  assert_output "1"
}

@test "copilot below a store without settings.json: the folder is outside a project" {
  mock_cli_version 0.15.6
  mkdir -p "$BATS_TEST_TMPDIR/work/.archcore" "$BATS_TEST_TMPDIR/work/repo"
  cd "$BATS_TEST_TMPDIR/work/repo"
  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "no .archcore/ directory in this project yet"
}

@test "below an initialized project: the empty-state nudge reads the project's store" {
  mock_cli_version 0.15.6
  mkdir -p "$BATS_TEST_TMPDIR/repo/.archcore" "$BATS_TEST_TMPDIR/repo/src"
  printf '{}\n' > "$BATS_TEST_TMPDIR/repo/.archcore/settings.json"
  cd "$BATS_TEST_TMPDIR/repo/src"
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial ".archcore/ is empty"
}

@test "below a project with documents: no empty-state nudge" {
  mock_cli_version 0.15.6
  mkdir -p "$BATS_TEST_TMPDIR/repo/.archcore" "$BATS_TEST_TMPDIR/repo/src"
  printf '{}\n' > "$BATS_TEST_TMPDIR/repo/.archcore/settings.json"
  awk 'BEGIN { s=""; for (i=0;i<300;i++) s=s "x"; print s }' > "$BATS_TEST_TMPDIR/repo/.archcore/big.doc.md"
  cd "$BATS_TEST_TMPDIR/repo/src"
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  refute_output --partial ".archcore/ is empty"
}
