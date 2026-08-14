#!/usr/bin/env bats
# Tests for bin/session-start

setup() {
  load '../helpers/common'
  common_setup
}

@test "reports missing .archcore/ directory and tells agent to call init_project" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "no .archcore/ directory"
  assert_output --partial "mcp__archcore__init_project"
  assert_output --partial "hookSpecificOutput"
}

# --- Per-host emit-shape pins (three shipped hosts) -------------------------
# claude-code gets the SessionStart hookSpecificOutput JSON wrapper; cursor and
# codex fall to the plain-text arm. Pinned so host-expansion edits to the emit
# function provably leave the shipped hosts untouched.

@test "init nudge: claude-code emits SessionStart hookSpecificOutput JSON" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{\"tool_name\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial '"hookEventName":"SessionStart"'
}

@test "init nudge: cursor emits plain text (no JSON wrapper)" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{\"conversation_id\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "no .archcore/ directory"
  refute_output --partial "hookSpecificOutput"
}

# Codex used to be pinned to plain text here. That pin encoded the defect:
# looks_like_json (codex-rs/hooks/src/engine/output_parser.rs) fires on '[' as
# well as '{', so a bare "[Archcore] …" line parses as a malformed JSON array
# and the host discards the whole hook run (codex-adapter.spec failure 2).
@test "init nudge: codex emits the SessionStart JSON wrapper, never a bare bracket line" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{\"turn_id\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "no .archcore/ directory"
  assert_output --partial "hookSpecificOutput"
  case "$output" in
    "[Archcore]"*) fail "codex must not emit a bare [Archcore] line: '$output'" ;;
  esac
}

@test "codex payload routes the CLI hook call to the codex-cli agent id" {
  # The shell host id is "codex" but the CLI registers its dialect as
  # "codex-cli"; an unmapped call hits an unknown subcommand and the session
  # silently loses its context.
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/archcore.log"
  mock_archcore_logging ""
  mkdir -p "$BATS_TEST_TMPDIR/proj/.archcore"
  echo "stub" > "$BATS_TEST_TMPDIR/proj/.archcore/stub.doc.md"
  cd "$BATS_TEST_TMPDIR/proj"
  run sh -c "printf '%s' '{\"turn_id\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  grep -q '^hooks codex-cli session-start' "$MOCK_ARCHCORE_LOG" \
    || fail "expected 'hooks codex-cli session-start' in CLI invocations, got: $(cat "$MOCK_ARCHCORE_LOG")"
}

@test "CLI-missing notice: claude-code emits SessionStart hookSpecificOutput JSON" {
  cd "$BATS_TEST_TMPDIR"
  run sh -c "PATH='/usr/bin:/bin'; export PATH; printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial '"hookEventName":"SessionStart"'
  assert_output --partial "install.sh"
}

@test "CLI-missing notice: cursor emits plain text (no JSON wrapper)" {
  cd "$BATS_TEST_TMPDIR"
  run sh -c "PATH='/usr/bin:/bin'; export PATH; printf '%s' '{\"conversation_id\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "install.sh"
  refute_output --partial "hookSpecificOutput"
}

@test "CLI-missing notice: codex emits the SessionStart JSON wrapper" {
  cd "$BATS_TEST_TMPDIR"
  run sh -c "PATH='/usr/bin:/bin'; export PATH; printf '%s' '{\"turn_id\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "install.sh"
  assert_output --partial "hookSpecificOutput"
  case "$output" in
    "[Archcore]"*) fail "codex must not emit a bare [Archcore] line: '$output'" ;;
  esac
}

# --- Copilot emit shape (native top-level additionalContext) ----------------

@test "init nudge: copilot emits top-level additionalContext JSON" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial '{"additionalContext":"'
  refute_output --partial "hookSpecificOutput"
}

@test "CLI-missing notice: copilot emits top-level additionalContext JSON" {
  cd "$BATS_TEST_TMPDIR"
  run sh -c "PATH='/usr/bin:/bin'; export PATH; ARCHCORE_HOST=copilot; export ARCHCORE_HOST; printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial '{"additionalContext":"'
  assert_output --partial "install.sh"
  refute_output --partial "hookSpecificOutput"
}

@test "init nudge: copilot instructs project wiring, not the phantom MCP tool" {
  # Day-one Copilot sessions have NO archcore MCP tools (the plugin cannot
  # ship them there — copilot-mcp-architecture.adr), so pointing the agent at
  # mcp__archcore__init_project is a dead instruction on exactly the host
  # that needs the nudge most.
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "archcore init --agent copilot --project"
  refute_output --partial "mcp__archcore__init_project"
}

@test "init nudge: copilot keeps the /archcore:init suffix and hide-nudge switch" {
  # The suffix sentence is shared with every other host byte-for-byte — it is
  # what skills/init/SKILL.md mirrors. The copilot fork replaces only the
  # instruction body.
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "/archcore:init"

  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot ARCHCORE_HIDE_EMPTY_NUDGE=1 '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "no .archcore/ directory"
  refute_output --partial "/archcore:init"
}

@test "init nudge: copilot payload is one valid JSON document with the quoted wiring step" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  local decoded
  decoded=$(printf '%s' "$output" | jq -re '.additionalContext') \
    || fail "copilot init-nudge payload is not a single valid JSON document: '$output'"
  [[ "$decoded" == *'archcore init --agent copilot --project "$PWD"'* ]] \
    || fail "decoded additionalContext lost the quoted wiring step: '$decoded'"
}

@test "CLI-missing notice: copilot names the mandatory wiring step" {
  # On Copilot the plugin ships no MCP server (copilot-mcp-architecture.adr),
  # so installing the CLI alone is not enough — the notice must carry the
  # project-wiring command or the user lands right back in a toolless session.
  cd "$BATS_TEST_TMPDIR"
  run sh -c "PATH='/usr/bin:/bin'; export PATH; ARCHCORE_HOST=copilot; export ARCHCORE_HOST; printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "archcore init --agent copilot --project"
  assert_output --partial "restart the session"
}

@test "CLI-missing notice: copilot payload is one valid JSON document with the wiring step intact" {
  # First escaping test of this arm: the wiring line carries literal double
  # quotes ('--project "$PWD"', unexpanded by design — no env value leaks
  # into the message), which MUST survive the inline JSON escaping.
  cd "$BATS_TEST_TMPDIR"
  run sh -c "PATH='/usr/bin:/bin'; export PATH; ARCHCORE_HOST=copilot; export ARCHCORE_HOST; printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  local decoded
  decoded=$(printf '%s' "$output" | jq -re '.additionalContext') \
    || fail "copilot CLI-missing payload is not a single valid JSON document: '$output'"
  [[ "$decoded" == *'archcore init --agent copilot --project "$PWD"'* ]] \
    || fail "decoded additionalContext lost the quoted wiring step: '$decoded'"
}

@test "CLI-missing notice: claude-code payload is one valid JSON document" {
  cd "$BATS_TEST_TMPDIR"
  run sh -c "PATH='/usr/bin:/bin'; export PATH; printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  printf '%s' "$output" | jq -e '.hookSpecificOutput.additionalContext' > /dev/null \
    || fail "claude-code CLI-missing payload is not a single valid JSON document: '$output'"
}

@test "CLI-missing notice: copilot JSON stays valid from a hostile cwd" {
  # The message is static, but the harness must not wobble when cwd carries
  # quotes, spaces, percent signs, or non-ASCII.
  local d="$BATS_TEST_TMPDIR/we\"ird %s тест dir"
  mkdir -p "$d"
  cd "$d"
  run sh -c "PATH='/usr/bin:/bin'; export PATH; ARCHCORE_HOST=copilot; export ARCHCORE_HOST; printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  printf '%s' "$output" | jq -e '.additionalContext' > /dev/null \
    || fail "copilot CLI-missing payload broke in a hostile cwd: '$output'"
}

@test "copilot: passes host arg to archcore hooks" {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
if [ "$1" = "hooks" ]; then
  echo "HOST_ARG: $2"
  cat > /dev/null
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"
  git init -q 2>/dev/null || true

  run sh -c "ARCHCORE_HOST=copilot; export ARCHCORE_HOST; printf '%s' '{\"sessionId\":\"s1\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "HOST_ARG: copilot"
}

# --- OpenCode emit shape (plain text; bridge reads stdout verbatim) ---------

@test "init nudge: opencode emits plain text (no JSON wrapper)" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=opencode '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "no .archcore/ directory"
  refute_output --partial "hookSpecificOutput"
  refute_output --partial '"additionalContext"'
}

@test "CLI-missing notice: opencode emits plain text (no JSON wrapper)" {
  cd "$BATS_TEST_TMPDIR"
  run sh -c "PATH='/usr/bin:/bin'; export PATH; ARCHCORE_HOST=opencode; export ARCHCORE_HOST; printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "install.sh"
  refute_output --partial '"additionalContext"'
}

@test "opencode: passes host arg to archcore hooks" {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
if [ "$1" = "hooks" ]; then
  echo "HOST_ARG: $2"
  cat > /dev/null
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"
  git init -q 2>/dev/null || true

  run sh -c "ARCHCORE_HOST=opencode; export ARCHCORE_HOST; printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "HOST_ARG: opencode"
}

@test "refuses to run from a plugin install dir (.plugin sibling)" {
  mock_archcore ""
  local fake_plugin="$BATS_TEST_TMPDIR/fake-plugin"
  mkdir -p "$fake_plugin/.plugin" "$fake_plugin/.archcore"
  echo '{"name":"fake"}' > "$fake_plugin/.plugin/plugin.json"
  cd "$fake_plugin"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -z "$output" ] || fail "expected silent exit, got: '$output'"
}

@test "survives when launcher cannot resolve CLI (no PATH, no cache, no network)" {
  # Initialized project + restricted PATH + ARCHCORE_SKIP_DOWNLOAD=1:
  # launcher exits 1, but session-start wraps with '|| true' and still succeeds.
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"

  run sh -c "PATH='/usr/bin:/bin' ARCHCORE_SKIP_DOWNLOAD=1 printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
}

@test "runs archcore hooks when both CLI and dir exist" {
  # Create mock archcore that logs the command
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
if [ "$1" = "hooks" ]; then
  echo "HOOKS_CALLED: $*"
  cat > /dev/null
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"

  # Create temp dir with .archcore/
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"
  git init -q 2>/dev/null || true

  run sh -c "printf '%s' '{\"test\":true}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "HOOKS_CALLED: hooks claude-code session-start"
}

@test "passes host from stdin to archcore hooks" {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
if [ "$1" = "hooks" ]; then
  echo "HOST_ARG: $2"
  cat > /dev/null
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"
  git init -q 2>/dev/null || true

  run sh -c "printf '%s' '{\"conversation_id\":\"x\"}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "HOST_ARG: cursor"
}

@test "session-start invokes only allowlisted subcommands" {
  # Lock the contract: session-start may call `archcore hooks` (the context
  # emitter), `archcore update` (only as `update --check` — the cached, quiet
  # freshness probe behind the outdated-CLI advisory), and `archcore
  # --version` (advisory display). Nothing else. Catches accidental
  # regressions where the script swaps to a phantom or different subcommand.
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/archcore.log"
  mock_archcore_logging ""

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"
  git init -q 2>/dev/null || true

  run sh -c "MOCK_ARCHCORE_LOG='$MOCK_ARCHCORE_LOG' printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -f "$MOCK_ARCHCORE_LOG" ] || fail "expected archcore to be invoked"

  # The log records full invocations ("$*"), so the `--check` flag is
  # asserted too: a regression to a bare `archcore update` would be a REAL
  # self-update fired from a session hook — exactly what the ADR forbids.
  local invoked line
  invoked=$(sort -u < "$MOCK_ARCHCORE_LOG")
  echo "$invoked" | grep -q "^hooks" || fail "expected 'hooks' to be invoked, got: '$invoked'"
  while IFS= read -r line; do
    case "$line" in
      hooks|"hooks "*) ;;
      "update --check") ;;
      update|"update "*) fail "only 'update --check' is allowed, got: '$line'" ;;
      --version) ;;
      *) fail "unexpected archcore invocation: '$line' (full log: '$invoked')" ;;
    esac
  done <<< "$invoked"
}

@test "refuses to run from a plugin install dir (cursor-plugin sibling)" {
  # If session-start is launched with cwd inside the plugin install cache,
  # it must NOT emit context — otherwise it would surface the plugin's own
  # bundled .archcore/ as the user's knowledge base.
  mock_archcore ""
  local fake_plugin="$BATS_TEST_TMPDIR/fake-plugin"
  mkdir -p "$fake_plugin/.cursor-plugin" "$fake_plugin/.archcore"
  echo '{"name":"fake"}' > "$fake_plugin/.cursor-plugin/plugin.json"
  cd "$fake_plugin"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -z "$output" ] || fail "expected silent exit, got: '$output'"
}

@test "refuses to run from a plugin install dir (claude-plugin sibling)" {
  mock_archcore ""
  local fake_plugin="$BATS_TEST_TMPDIR/fake-plugin"
  mkdir -p "$fake_plugin/.claude-plugin" "$fake_plugin/.archcore"
  echo '{"name":"fake"}' > "$fake_plugin/.claude-plugin/plugin.json"
  cd "$fake_plugin"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -z "$output" ] || fail "expected silent exit, got: '$output'"
}

@test "refuses to run from a plugin install dir (codex-plugin sibling)" {
  mock_archcore ""
  local fake_plugin="$BATS_TEST_TMPDIR/fake-plugin"
  mkdir -p "$fake_plugin/.codex-plugin" "$fake_plugin/.archcore"
  echo '{"name":"fake"}' > "$fake_plugin/.codex-plugin/plugin.json"
  cd "$fake_plugin"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -z "$output" ] || fail "expected silent exit, got: '$output'"
}

@test "staleness check failure is non-fatal" {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
if [ "$1" = "hooks" ]; then
  echo "context loaded"
  cat > /dev/null
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
}

# --- Old-CLI compatibility advisory ----------------------------------------
# An old (pre-globals) CLI rejects `globals` in settings.json and exits non-zero
# on every config-loading command. session-start must turn that crash into one
# clear, rate-limited "update CLI" nudge — never a hard block.

# Mock: `archcore hooks` consumes stdin, writes a config-rejection to stderr,
# and exits non-zero. Args after `mock_old_cli` become the stderr message.
mock_old_cli() {
  local stderr_msg="${1:-field \"globals\" is not allowed for sync type \"none\"}"
  cat > "$MOCK_BIN/archcore" <<MOCK
#!/bin/sh
[ -n "\$MOCK_ARCHCORE_LOG" ] && printf '%s\n' "\$*" >> "\$MOCK_ARCHCORE_LOG"
if [ "\$1" = "hooks" ]; then
  cat > /dev/null
  printf '%s\n' '${stderr_msg}' >&2
  exit 1
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"
}

@test "old CLI + globals in config → emits update-CLI advisory, exit 0" {
  mock_old_cli
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf '%s' '{"sync":"none","globals":[{"id":"company","path":".archcore/global/company"}]}' \
    > "$workdir/.archcore/settings.json"
  cd "$workdir"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "too old"
  assert_output --partial "install.sh"
}

@test "old CLI rejects an unknown field via stderr (no literal globals) → advisory" {
  # Validates the OR branch: settings carries no literal "globals", but stderr
  # shows the parser's rejection signature, so the advisory still fires.
  mock_old_cli 'field "foo" is not allowed for sync type "none"'
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf '%s' '{"sync":"none","foo":true}' > "$workdir/.archcore/settings.json"
  cd "$workdir"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "too old"
}

@test "non-config CLI failure → no advisory (silent fall-through)" {
  # The key guard against bare-non-zero nudging: a generic failure with no
  # config-rejection signal and no globals in settings must NOT nudge.
  mock_old_cli 'fatal: not a git repository'
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf '%s' '{"sync":"none"}' > "$workdir/.archcore/settings.json"
  cd "$workdir"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  refute_output --partial "too old"
}

@test "globals in config + non-parser failure → no advisory (regression lock)" {
  # A current CLI that understands globals can still fail for unrelated reasons.
  # Presence of globals must NOT be sufficient — only a real parser-rejection
  # stderr signature may fire the advisory. (Codex P2: drop the globals
  # short-circuit so non-config failures stay silent even in globals projects.)
  mock_old_cli 'error: operation not allowed'
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf '%s' '{"sync":"none","globals":[{"id":"c","path":".archcore/global/c"}]}' \
    > "$workdir/.archcore/settings.json"
  cd "$workdir"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  refute_output --partial "too old"
}

@test "CLI succeeds with globals in config → context flows, no advisory" {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
if [ "$1" = "hooks" ]; then
  cat > /dev/null
  echo "context loaded"
fi
MOCK
  chmod +x "$MOCK_BIN/archcore"
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf '%s' '{"sync":"none","globals":[{"id":"c","path":".archcore/global/c"}]}' \
    > "$workdir/.archcore/settings.json"
  cd "$workdir"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "context loaded"
  refute_output --partial "too old"
}

@test "advisory is rate-limited to once per 24h" {
  mock_old_cli
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf '%s' '{"sync":"none","globals":[{"id":"c","path":".archcore/global/c"}]}' \
    > "$workdir/.archcore/settings.json"
  cd "$workdir"

  # First run emits the advisory and writes the stamp.
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "too old"

  # Second run within 24h is suppressed.
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  refute_output --partial "too old"
}

@test "advisory path still invokes only the 'hooks' subcommand" {
  # Rule-mandated (cli-integration-tests.rule): even on the failure path the
  # script must shell out to nothing but `archcore hooks` (the advisory path
  # exits before the update-check probe runs).
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/archcore.log"
  mock_old_cli
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  printf '%s' '{"sync":"none","globals":[{"id":"c","path":".archcore/global/c"}]}' \
    > "$workdir/.archcore/settings.json"
  cd "$workdir"

  run sh -c "MOCK_ARCHCORE_LOG='$MOCK_ARCHCORE_LOG' printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [ -f "$MOCK_ARCHCORE_LOG" ] || fail "expected archcore to be invoked"

  # Log records full invocations ("$*"); every line must be a `hooks` call.
  local line
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    case "$line" in
      hooks|"hooks "*) ;;
      *) fail "expected only 'hooks' invocations, got: '$line'" ;;
    esac
  done < "$MOCK_ARCHCORE_LOG"
}

# Helper: run session-start from a fresh initialized project dir.
run_session_start_in_project() {
  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
}

@test "update advisory: emitted when update --check reports a newer version" {
  mock_archcore_with_update v9.9.9 v0.5.7
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"

  run_session_start_in_project
  assert_success
  [[ "$output" == *"CLI update available"* ]] \
    || fail "expected update advisory in output, got: '$output'"
  [[ "$output" == *"archcore update"* ]] \
    || fail "advisory must name the fix command, got: '$output'"
}

@test "update advisory: names the available and installed versions" {
  # Pins the `${_ac_update#update available: }` prefix-stripping — a format
  # regression would echo the raw CLI line instead of the bare version.
  mock_archcore_with_update v9.9.9 v0.5.7
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"

  run_session_start_in_project
  assert_success
  [[ "$output" == *"(v9.9.9"* ]] \
    || fail "expected stripped latest version 'v9.9.9' after '(', got: '$output'"
  [[ "$output" != *"update available: v9.9.9"* ]] \
    || fail "raw CLI prefix must be stripped from the advisory, got: '$output'"
  [[ "$output" == *"installed: v0.5.7"* ]] \
    || fail "expected installed version in advisory, got: '$output'"
}

@test "update advisory: probe is exactly 'update --check', never a bare update" {
  # A bare `archcore update` here would be a REAL self-update fired from a
  # session hook. The mock exits 1 on it; the log pins the exact invocation.
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/archcore.log"
  mock_archcore_with_update
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"

  run_session_start_in_project
  assert_success
  grep -qx "update --check" "$MOCK_ARCHCORE_LOG" \
    || fail "expected an 'update --check' invocation, got: $(cat "$MOCK_ARCHCORE_LOG")"
  ! grep -qx "update" "$MOCK_ARCHCORE_LOG" \
    || fail "bare 'update' (real self-update) was invoked: $(cat "$MOCK_ARCHCORE_LOG")"
}

@test "update advisory: rate-limited to once per 24h" {
  mock_archcore_with_update
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"

  run_session_start_in_project
  assert_success
  [[ "$output" == *"CLI update available"* ]] || fail "first run must advise"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [[ "$output" != *"CLI update available"* ]] \
    || fail "second run within 24h must stay quiet, got: '$output'"
}

@test "update advisory: fires again once the 24h window has passed" {
  # Deterministic boundary check via stamp pre-seeding — no clock mock needed.
  mock_archcore_with_update
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  mkdir -p "$CLAUDE_PLUGIN_DATA/archcore"
  echo "$(( $(date +%s) - 86500 ))" > "$CLAUDE_PLUGIN_DATA/archcore/last-update-advisory"

  run_session_start_in_project
  assert_success
  [[ "$output" == *"CLI update available"* ]] \
    || fail "advisory must fire again after 24h, got: '$output'"
}

@test "update advisory: stays quiet while the 24h window is open" {
  mock_archcore_with_update
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  mkdir -p "$CLAUDE_PLUGIN_DATA/archcore"
  echo "$(( $(date +%s) - 100 ))" > "$CLAUDE_PLUGIN_DATA/archcore/last-update-advisory"

  run_session_start_in_project
  assert_success
  [[ "$output" != *"CLI update available"* ]] \
    || fail "advisory must stay quiet inside the 24h window, got: '$output'"
}

@test "update advisory: garbage stamp treated as due" {
  # Covers the ''|*[!0-9]* branch — a corrupt stamp must fail open, not
  # suppress the advisory forever.
  mock_archcore_with_update
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  mkdir -p "$CLAUDE_PLUGIN_DATA/archcore"
  echo "junk" > "$CLAUDE_PLUGIN_DATA/archcore/last-update-advisory"

  run_session_start_in_project
  assert_success
  [[ "$output" == *"CLI update available"* ]] \
    || fail "garbage stamp must count as due, got: '$output'"
}

@test "update advisory: not suppressed by a fresh old-CLI advisory stamp" {
  # The two advisories rate-limit independently — last-cli-advisory (old-CLI
  # config rejection) must never clobber last-update-advisory.
  mock_archcore_with_update
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"
  mkdir -p "$CLAUDE_PLUGIN_DATA/archcore"
  date +%s > "$CLAUDE_PLUGIN_DATA/archcore/last-cli-advisory"

  run_session_start_in_project
  assert_success
  [[ "$output" == *"CLI update available"* ]] \
    || fail "a fresh last-cli-advisory stamp must not suppress the update advisory, got: '$output'"
}

@test "update advisory: silent when update --check outputs nothing (current or old CLI)" {
  # An up-to-date CLI prints nothing; an old CLI without --check errors out —
  # both must yield zero advisory noise.
  mock_archcore ""
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/data"

  run_session_start_in_project
  assert_success
  [[ "$output" != *"CLI update available"* ]] \
    || fail "no advisory expected when --check is silent, got: '$output'"
}

@test "update advisory: stamp lands under XDG_DATA_HOME when CLAUDE_PLUGIN_DATA is unset" {
  # Covers the _archcore_stamp_dir fallback chain (CLAUDE_PLUGIN_DATA →
  # XDG_DATA_HOME → HOME); every other advisory test pins the first leg.
  mock_archcore_with_update
  unset CLAUDE_PLUGIN_DATA
  export XDG_DATA_HOME="$BATS_TEST_TMPDIR/xdg"

  local workdir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$workdir/.archcore"
  cd "$workdir"
  run sh -c "printf '%s' '{}' | XDG_DATA_HOME='$XDG_DATA_HOME' '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  [[ "$output" == *"CLI update available"* ]] || fail "advisory expected, got: '$output'"
  [ -f "$XDG_DATA_HOME/archcore-plugin/last-update-advisory" ] \
    || fail "stamp must land in \$XDG_DATA_HOME/archcore-plugin/, found: $(find "$XDG_DATA_HOME" -type f 2>/dev/null)"
}
