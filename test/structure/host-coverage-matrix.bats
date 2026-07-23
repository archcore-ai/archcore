#!/usr/bin/env bats
# Host coverage matrix (host-adapter-contract.spec): one loop-driven row per
# host hooks config. Adding a host = adding one row here; a hooks config file
# that exists but is not enrolled fails CI (enrollment guard). The hardcoded
# per-file tests in hooks.bats remain as the precision layer.
#
# Row format: file|session-event-key|expected-event-set|write-guard-event|required-matcher-tools
# (expected-event-set is the sorted, comma-joined jq keys output; matcher
# tools are space-separated tokens that MUST appear in the write-guard matcher.)

setup() {
  load '../helpers/common'
  common_setup
}

matrix_rows() {
  cat <<'ROWS'
hooks/hooks.json|SessionStart|PostToolUse,PreToolUse,SessionStart,|PreToolUse|Write Edit
hooks/cursor.hooks.json|sessionStart|afterMCPExecution,preToolUse,sessionStart,|preToolUse|Write
hooks/codex.hooks.json|SessionStart|PostToolUse,PreToolUse,SessionStart,|PreToolUse|Write Edit apply_patch
ROWS
}

# Normalize a hooks config's command list to sorted unique script basenames.
script_basenames() {
  jq -r '.. | .command? // empty' "$1" | sed 's|"||g' | awk -F/ '{print $NF}' | sort -u
}

@test "host matrix: every host hooks file has the expected event set" {
  local row file events expected
  while IFS='|' read -r file _ expected _ _; do
    [ -z "$file" ] && continue
    events=$(jq -r '.hooks | keys[]' "$PLUGIN_ROOT/$file" | sort | tr '\n' ',')
    [ "$events" = "$expected" ] \
      || fail "$file: event set drifted — expected '$expected', got '$events'"
  done < <(matrix_rows)
}

@test "host matrix: every host registers session-start on its session event" {
  local file session_key cmds
  while IFS='|' read -r file session_key _ _ _; do
    [ -z "$file" ] && continue
    cmds=$(jq -r --arg e "$session_key" '.hooks[$e][]?.hooks[]?.command // empty' "$PLUGIN_ROOT/$file")
    echo "$cmds" | grep -q 'bin/session-start' \
      || fail "$file: '$session_key' must invoke bin/session-start; got: $cmds"
  done < <(matrix_rows)
}

@test "host matrix: every host registers the write guard with its native tool matcher" {
  local file guard_event tools guard_matcher tok entry_cmds
  while IFS='|' read -r file _ _ guard_event tools; do
    [ -z "$file" ] && continue
    # The write-guard event must run both check-archcore-write and check-code-alignment.
    entry_cmds=$(jq -r --arg e "$guard_event" '.hooks[$e][]?.hooks[]?.command // empty' "$PLUGIN_ROOT/$file")
    echo "$entry_cmds" | grep -q 'bin/check-archcore-write' \
      || fail "$file: '$guard_event' must invoke bin/check-archcore-write; got: $entry_cmds"
    echo "$entry_cmds" | grep -q 'bin/check-code-alignment' \
      || fail "$file: '$guard_event' must invoke bin/check-code-alignment; got: $entry_cmds"
    # The guard entry's matcher must cover every mutation tool of that host.
    guard_matcher=$(jq -r --arg e "$guard_event" \
      '.hooks[$e][] | select(.hooks[]?.command | test("check-archcore-write")) | .matcher' \
      "$PLUGIN_ROOT/$file" | head -1)
    for tok in $tools; do
      case "$guard_matcher" in
        *"$tok"*) : ;;
        *) fail "$file: write-guard matcher '$guard_matcher' misses mutation tool '$tok'" ;;
      esac
    done
  done < <(matrix_rows)
}

@test "host matrix: script-set parity with hooks.json" {
  local canonical file actual
  canonical=$(script_basenames "$PLUGIN_ROOT/hooks/hooks.json")
  while IFS='|' read -r file _ _ _ _; do
    [ -z "$file" ] && continue
    actual=$(script_basenames "$PLUGIN_ROOT/$file")
    [ "$actual" = "$canonical" ] || {
      echo "canonical (hooks.json): $canonical"
      echo "$file: $actual"
      fail "$file references a different script set than hooks.json"
    }
  done < <(matrix_rows)
}

@test "enrollment guard: every hooks config file is a matrix row" {
  local f rel enrolled
  enrolled=$(matrix_rows | cut -d'|' -f1)
  for f in "$PLUGIN_ROOT"/hooks/*.json; do
    rel="hooks/$(basename "$f")"
    echo "$enrolled" | grep -qx "$rel" \
      || fail "$rel exists but is not enrolled in the host coverage matrix — add a row (and tests) for it"
  done
}

@test "enrollment guard: every stdin fixture dir is a known host" {
  local d name
  for d in "$FIXTURES"/stdin/*/; do
    name=$(basename "$d")
    case "$name" in
      claude-code|cursor|codex|copilot|opencode|malformed) : ;;
      *) fail "unknown fixture host dir: test/fixtures/stdin/$name" ;;
    esac
  done
}
