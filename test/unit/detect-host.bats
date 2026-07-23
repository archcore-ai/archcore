#!/usr/bin/env bats
# Contract specs for bin/detect-host.
#
# Purpose (cwd-independence): the /archcore:init skill needs to know which
# host it runs under WITHOUT trusting cwd or stdin (Cursor does not guarantee
# either for agent shell commands). detect-host resolves the host from
# environment only and prints EXACTLY ONE token on stdout:
#
#   claude-code | cursor | codex-cli | __UNKNOWN__
#
# Contract:
#   - env-only: never reads stdin, never inspects cwd;
#   - exactly one line, no decoration, always exit 0 (callers branch on the
#     token; __UNKNOWN__ is a value, not an error);
#   - precedence when multiple host envs are present: claude-code > cursor >
#     codex-cli (a Claude-spawned subshell may inherit stray vars; the most
#     specific signal wins).
#
# Env signals (keep these constants in sync with the script):
#   claude-code: CLAUDECODE=1 or CLAUDE_SKILL_DIR set
#   cursor:      CURSOR_TRACE_ID set
#   codex-cli:   CODEX_HOME set
#
# The exec-bit check is a hard assertion (not a skip): a lost exec bit must
# turn the suite red, not silently green it via 10 skips. `env -i` gives a
# hermetic environment (only PATH survives) so developer machines running
# inside Claude Code don't leak CLAUDECODE into the tests.

setup() {
  load '../helpers/common'
  common_setup
  DETECT="$PLUGIN_ROOT/bin/detect-host"
  require_detect_host() {
    [ -x "$DETECT" ] || fail "bin/detect-host missing or not executable"
  }
}

@test "detect-host: CLAUDECODE=1 → claude-code" {
  require_detect_host
  run env -i PATH="$PATH" CLAUDECODE=1 "$DETECT"
  assert_success
  assert_output "claude-code"
}

@test "detect-host: CLAUDE_SKILL_DIR set → claude-code" {
  require_detect_host
  run env -i PATH="$PATH" CLAUDE_SKILL_DIR=/tmp/skills/init "$DETECT"
  assert_success
  assert_output "claude-code"
}

@test "detect-host: cursor env → cursor" {
  require_detect_host
  run env -i PATH="$PATH" CURSOR_TRACE_ID=abc123 "$DETECT"
  assert_success
  assert_output "cursor"
}

@test "detect-host: codex env → codex-cli" {
  require_detect_host
  run env -i PATH="$PATH" CODEX_HOME="$HOME/.codex" "$DETECT"
  assert_success
  assert_output "codex-cli"
}

@test "detect-host: no host env → __UNKNOWN__ sentinel, still exit 0" {
  require_detect_host
  run env -i PATH="$PATH" "$DETECT"
  assert_success
  assert_output "__UNKNOWN__"
}

@test "detect-host: claude-code wins when cursor env also present" {
  require_detect_host
  run env -i PATH="$PATH" CLAUDECODE=1 CURSOR_TRACE_ID=abc "$DETECT"
  assert_success
  assert_output "claude-code"
}

@test "detect-host: cursor wins over codex when both present" {
  require_detect_host
  run env -i PATH="$PATH" CURSOR_TRACE_ID=abc CODEX_HOME="$HOME/.codex" "$DETECT"
  assert_success
  assert_output "cursor"
}

@test "detect-host: claude-code wins the 3-way tie (all host envs present)" {
  require_detect_host
  run env -i PATH="$PATH" CLAUDECODE=1 CURSOR_TRACE_ID=abc CODEX_HOME="$HOME/.codex" "$DETECT"
  assert_success
  assert_output "claude-code"
}

@test "detect-host: prints exactly one line" {
  require_detect_host
  run env -i PATH="$PATH" CLAUDECODE=1 "$DETECT"
  assert_success
  [ "$(printf '%s\n' "$output" | wc -l | tr -d ' ')" = "1" ] \
    || fail "expected exactly one output line, got: '$output'"
}

@test "detect-host: ignores stdin entirely (env-only contract)" {
  require_detect_host
  # Cursor-shaped stdin must NOT sway an env-driven claude-code answer.
  run sh -c "printf '%s' '{\"conversation_id\":\"x\"}' | env -i PATH='$PATH' CLAUDECODE=1 '$DETECT'"
  assert_success
  assert_output "claude-code"
}

@test "detect-host: answer does not depend on cwd (plugin cache dir)" {
  require_detect_host
  # Simulate Cursor's misrouted cwd: a fake plugin install dir. detect-host
  # must still answer from env alone.
  local fake_cache="$BATS_TEST_TMPDIR/plugins/cache/archcore"
  mkdir -p "$fake_cache/.cursor-plugin"
  echo '{"name":"fake"}' > "$fake_cache/.cursor-plugin/plugin.json"
  cd "$fake_cache"
  run env -i PATH="$PATH" CURSOR_TRACE_ID=abc "$DETECT"
  assert_success
  assert_output "cursor"
}
