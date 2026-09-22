#!/usr/bin/env bats
# Copilot wiring-missing advisory in bin/session-start.
#
# On Copilot the plugin cannot ship an MCP server (copilot-mcp-architecture.adr)
# — project wiring (an archcore entry in .mcp.json, written by
# `archcore init --agent copilot`) is the ONLY route to document tools. This
# advisory is the session-time nudge for the "CLI present, .archcore/ present,
# wiring absent" state that previously failed silently.
#
# Scope pins that matter:
#   - copilot-only: every other host stays byte-identical
#     (test/unit/session-start-goldens.bats holds the byte-level pin);
#   - detection is pure stat/grep + one `git rev-parse` — never an archcore
#     subcommand beyond the session-start allowlist;
#   - rate-limit stamp is per-project (cksum of the project root), so one
#     repo's advisory can never silence another repo's mandatory step.

setup() {
  load '../helpers/common'
  common_setup
}

ADVISORY_MARKER="not wired for Copilot"

# An initialized (>200-byte doc, so the empty-state nudge stays quiet),
# UNWIRED project directory. Prints its path.
make_unwired_project() {
  local d="$BATS_TEST_TMPDIR/${1:-project}"
  mkdir -p "$d/.archcore"
  awk 'BEGIN { s=""; for (i=0;i<300;i++) s=s "x"; print s }' > "$d/.archcore/big.doc.md"
  printf '%s' "$d"
}

wired_mcp_json() {
  printf '%s' '{"mcpServers":{"archcore":{"command":"archcore","args":["mcp"]}}}'
}

run_copilot_session_start() {
  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
}

stamp_name_for() {
  printf 'last-wiring-advisory-%s' "$(printf '%s' "$1" | cksum | awk '{print $1}')"
}

@test "wiring advisory: fires on copilot when no archcore server is wired" {
  mock_archcore ""
  cd "$(make_unwired_project)"

  run_copilot_session_start
  assert_success
  assert_output --partial "$ADVISORY_MARKER"
  assert_output --partial "archcore init --agent copilot --project"
}

@test "wiring advisory: silent when project .mcp.json carries an archcore server" {
  mock_archcore ""
  local d
  d="$(make_unwired_project wired)"
  wired_mcp_json > "$d/.mcp.json"
  cd "$d"

  run_copilot_session_start
  assert_success
  refute_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: a repo-root .mcp.json suppresses it from a subdirectory" {
  # Copilot discovers .mcp.json from the working directory up to the git
  # root — a monorepo subdir session with root wiring is a WIRED session,
  # and a cwd-only check would nudge exactly where Copilot works fine.
  mock_archcore ""
  local root="$BATS_TEST_TMPDIR/mono"
  mkdir -p "$root/services/api/.archcore" "$root/.archcore"
  awk 'BEGIN { s=""; for (i=0;i<300;i++) s=s "x"; print s }' > "$root/services/api/.archcore/big.doc.md"
  wired_mcp_json > "$root/.mcp.json"
  cd "$root"
  git init -q
  git config user.email "test@test.com"
  git config user.name "Test"
  cd "$root/services/api"

  run_copilot_session_start
  assert_success
  refute_output --partial "$ADVISORY_MARKER"
}

# Copilot's discovery, measured on CLI 1.0.76 (2026-08-03) with probe servers
# planted at four levels of a git repo: it reads a config from EVERY directory
# between the working directory and the git root — not just those two ends —
# and within each directory it reads `.mcp.json`, falling back to
# `.github/mcp.json` only where `.mcp.json` is absent. Detection mirrors that
# exactly. Checking too little cries "not wired" at a project that works;
# checking too much stays silent for a project that does not, which leaves the
# user with no tools and nothing on screen explaining why.

@test "wiring advisory: an intermediate directory's .mcp.json suppresses it" {
  mock_archcore ""
  local root="$BATS_TEST_TMPDIR/mid-mcp"
  mkdir -p "$root/a/b/c/.archcore"
  awk 'BEGIN { s=""; for (i=0;i<300;i++) s=s "x"; print s }' > "$root/a/b/c/.archcore/big.doc.md"
  cd "$root"
  git init -q
  git config user.email "test@test.com"
  git config user.name "Test"
  # Neither the cwd nor the git root — a level Copilot reads and a two-ended
  # check cannot see.
  wired_mcp_json > "$root/a/b/.mcp.json"
  cd "$root/a/b/c"

  run_copilot_session_start
  assert_success
  refute_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: .github/mcp.json at the git root suppresses it" {
  mock_archcore ""
  local root="$BATS_TEST_TMPDIR/gh-root"
  mkdir -p "$root/sub/.archcore" "$root/.github"
  awk 'BEGIN { s=""; for (i=0;i<300;i++) s=s "x"; print s }' > "$root/sub/.archcore/big.doc.md"
  cd "$root"
  git init -q
  git config user.email "test@test.com"
  git config user.name "Test"
  wired_mcp_json > "$root/.github/mcp.json"
  cd "$root/sub"

  run_copilot_session_start
  assert_success
  refute_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: .github/mcp.json in an intermediate directory suppresses it" {
  mock_archcore ""
  local root="$BATS_TEST_TMPDIR/gh-mid"
  mkdir -p "$root/a/b/c/.archcore" "$root/a/b/.github"
  awk 'BEGIN { s=""; for (i=0;i<300;i++) s=s "x"; print s }' > "$root/a/b/c/.archcore/big.doc.md"
  cd "$root"
  git init -q
  git config user.email "test@test.com"
  git config user.name "Test"
  wired_mcp_json > "$root/a/b/.github/mcp.json"
  cd "$root/a/b/c"

  run_copilot_session_start
  assert_success
  refute_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: .github/mcp.json is ignored where a .mcp.json shadows it" {
  # Per-directory precedence, not a union: with both names present in the same
  # directory Copilot reads only .mcp.json, so an archcore entry parked in the
  # .github copy never reaches the session. Treating the pair as a union would
  # be the silent-and-wrong direction.
  mock_archcore ""
  local d
  d="$(make_unwired_project shadowed)"
  mkdir -p "$d/.github"
  printf '%s' '{"mcpServers":{"other":{"command":"other"}}}' > "$d/.mcp.json"
  wired_mcp_json > "$d/.github/mcp.json"
  cd "$d"

  run_copilot_session_start
  assert_success
  assert_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: .mcp.json without an archcore entry still fires" {
  # Detection is a POSIX grep for the server name, not a JSON parse — a
  # foreign-only .mcp.json is an unwired project.
  mock_archcore ""
  local d
  d="$(make_unwired_project foreign)"
  printf '%s' '{"mcpServers":{"other":{"command":"other"}}}' > "$d/.mcp.json"
  cd "$d"

  run_copilot_session_start
  assert_success
  assert_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: silent when user-level mcp-config.json is wired" {
  mock_archcore ""
  local home="$BATS_TEST_TMPDIR/copilot-home"
  mkdir -p "$home"
  printf '%s' '{"mcpServers":{"archcore":{"command":"archcore","args":["mcp"]}}}' > "$home/mcp-config.json"
  cd "$(make_unwired_project userlevel)"

  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot COPILOT_HOME='$home' '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  refute_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: silent on every non-copilot host" {
  mock_archcore ""
  local d host stdin
  d="$(make_unwired_project other-hosts)"
  cd "$d"
  for host in claude-code cursor codex opencode; do
    case "$host" in
      cursor) stdin='{"conversation_id":"x"}' ;;
      codex) stdin='{"turn_id":"x"}' ;;
      *) stdin='{}' ;;
    esac
    if [ "$host" = opencode ]; then
      run sh -c "printf '%s' '$stdin' | ARCHCORE_HOST=opencode '${PLUGIN_ROOT}/bin/session-start'"
    else
      run sh -c "printf '%s' '$stdin' | '${PLUGIN_ROOT}/bin/session-start'"
    fi
    assert_success
    refute_output --partial "$ADVISORY_MARKER" \
      || fail "advisory leaked to host '$host'"
  done
}

@test "wiring advisory: rate-limited to once per 24h within one project" {
  mock_archcore ""
  cd "$(make_unwired_project ratelimit)"

  run_copilot_session_start
  assert_success
  assert_output --partial "$ADVISORY_MARKER"

  run_copilot_session_start
  assert_success
  refute_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: fires again once the 24h window has passed" {
  # Deterministic boundary check via stamp pre-seeding — no clock mock needed.
  mock_archcore ""
  local d
  d="$(make_unwired_project window)"
  cd "$d"
  mkdir -p "$CLAUDE_PLUGIN_DATA/archcore"
  echo "$(( $(date +%s) - 86500 ))" > "$CLAUDE_PLUGIN_DATA/archcore/$(stamp_name_for "$d")"

  run_copilot_session_start
  assert_success
  assert_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: stays quiet while the 24h window is open" {
  mock_archcore ""
  local d
  d="$(make_unwired_project open-window)"
  cd "$d"
  mkdir -p "$CLAUDE_PLUGIN_DATA/archcore"
  echo "$(( $(date +%s) - 100 ))" > "$CLAUDE_PLUGIN_DATA/archcore/$(stamp_name_for "$d")"

  run_copilot_session_start
  assert_success
  refute_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: garbage stamp treated as due" {
  mock_archcore ""
  local d
  d="$(make_unwired_project garbage)"
  cd "$d"
  mkdir -p "$CLAUDE_PLUGIN_DATA/archcore"
  echo "junk" > "$CLAUDE_PLUGIN_DATA/archcore/$(stamp_name_for "$d")"

  run_copilot_session_start
  assert_success
  assert_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: a fresh stamp for project A does not silence project B" {
  mock_archcore ""
  local a b
  a="$(make_unwired_project proj-a)"
  b="$(make_unwired_project proj-b)"

  cd "$a"
  run_copilot_session_start
  assert_success
  assert_output --partial "$ADVISORY_MARKER"

  cd "$b"
  run_copilot_session_start
  assert_success
  assert_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: independent of the other advisory stamps, both directions" {
  mock_archcore_with_update v9.9.9 v0.5.7
  local d
  d="$(make_unwired_project stamps)"
  cd "$d"
  mkdir -p "$CLAUDE_PLUGIN_DATA/archcore"
  # Fresh foreign stamps must not suppress the wiring advisory...
  date +%s > "$CLAUDE_PLUGIN_DATA/archcore/last-cli-advisory"
  date +%s > "$CLAUDE_PLUGIN_DATA/archcore/last-staleness"

  run_copilot_session_start
  assert_success
  assert_output --partial "$ADVISORY_MARKER"
  # ...and the update advisory still fired alongside a fresh wiring stamp.
  assert_output --partial "CLI update available"
}

@test "wiring advisory: ARCHCORE_HIDE_EMPTY_NUDGE does not suppress it" {
  mock_archcore ""
  cd "$(make_unwired_project hide-empty)"

  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot ARCHCORE_HIDE_EMPTY_NUDGE=1 '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: ARCHCORE_HIDE_WIRING_NUDGE=1 suppresses it" {
  mock_archcore ""
  cd "$(make_unwired_project hide-wiring)"

  run sh -c "printf '%s' '{}' | ARCHCORE_HOST=copilot ARCHCORE_HIDE_WIRING_NUDGE=1 '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  refute_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: .archcore as a file routes to the init nudge, no advisory, no crash" {
  mock_archcore ""
  local d="$BATS_TEST_TMPDIR/file-not-dir"
  mkdir -p "$d"
  echo "not a dir" > "$d/.archcore"
  cd "$d"

  run_copilot_session_start
  assert_success
  assert_output --partial "no .archcore/ directory"
  refute_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: never reached from a plugin install cache (guard first)" {
  mock_archcore ""
  local dir="$BATS_TEST_TMPDIR/home/.copilot/installed-plugins/some-plugin/deep"
  mkdir -p "$dir/.archcore"
  cd "$dir"

  run_copilot_session_start
  assert_success
  [ -z "$output" ] || fail "expected the cache guard to win over the advisory, got: '$output'"
}

@test "wiring advisory: detection invokes only allowlisted archcore subcommands" {
  # The wiring check must be pure stat/grep (+ git rev-parse) — an archcore
  # subcommand here would run before every session on an unwired repo.
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/archcore.log"
  mock_archcore_logging ""
  cd "$(make_unwired_project allowlist)"

  run_copilot_session_start
  assert_success
  assert_output --partial "$ADVISORY_MARKER"
  [ -f "$MOCK_ARCHCORE_LOG" ] || fail "expected the hooks call to be logged"
  local bad
  bad=$(grep -vE '^(hooks |update --check$|--version$)' "$MOCK_ARCHCORE_LOG" || true)
  [ -z "$bad" ] || fail "non-allowlisted archcore invocation(s) from session-start: $bad"
}

@test "wiring advisory: deep non-git cwd terminates and fires" {
  mock_archcore ""
  local d="$BATS_TEST_TMPDIR/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o"
  mkdir -p "$d/.archcore"
  awk 'BEGIN { s=""; for (i=0;i<300;i++) s=s "x"; print s }' > "$d/.archcore/big.doc.md"
  cd "$d"

  run_copilot_session_start
  assert_success
  assert_output --partial "$ADVISORY_MARKER"
}

@test "wiring advisory: stamp lands under HOME when CLAUDE_PLUGIN_DATA and XDG_DATA_HOME are unset" {
  # Closes the previously-untested third leg of _archcore_stamp_dir.
  mock_archcore ""
  local home="$BATS_TEST_TMPDIR/home"
  mkdir -p "$home"
  cd "$(make_unwired_project home-leg)"

  run sh -c "printf '%s' '{}' | env -u CLAUDE_PLUGIN_DATA -u XDG_DATA_HOME HOME='$home' ARCHCORE_HOST=copilot '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_output --partial "$ADVISORY_MARKER"
  local stamp
  stamp=$(find "$home/.local/share/archcore-plugin" -name 'last-wiring-advisory-*' -type f 2>/dev/null)
  [ -n "$stamp" ] || fail "expected a wiring stamp under \$HOME/.local/share/archcore-plugin"
}

@test "wiring advisory: writes nothing into the project or the plugin" {
  mock_archcore ""
  local d
  d="$(make_unwired_project no-writes)"
  cd "$d"
  local sentinel="$BATS_TEST_TMPDIR/sentinel"
  touch "$sentinel"

  run_copilot_session_start
  assert_success
  assert_output --partial "$ADVISORY_MARKER"
  local leaked
  leaked=$(find "$d" "$PLUGIN_ROOT" -type f -newer "$sentinel" 2>/dev/null)
  [ -z "$leaked" ] || fail "session-start wrote into the project or plugin: $leaked"
}

@test "wiring advisory: no cache paths or env values leak into the text" {
  mock_archcore ""
  local d
  d="$(make_unwired_project no-leaks)"
  cd "$d"

  run_copilot_session_start
  assert_success
  assert_output --partial "$ADVISORY_MARKER"
  refute_output --partial "$CLAUDE_PLUGIN_DATA"
  refute_output --partial "$MOCK_BIN"
  refute_output --partial "installed-plugins"
  # The wiring command keeps "$PWD" literal — the real path must not appear.
  refute_output --partial "$d"
}

@test "wiring advisory: rides inside the CLI JSON, not after it" {
  # Channel pin. Copilot strips progress lines, concatenates everything else
  # and runs ONE JSON.parse; on failure the hook counts as having produced no
  # output at all. Trailing plain text here would therefore have cost the
  # session both Archcore's context AND this advisory — and this advisory
  # exists precisely to explain a session that has no Archcore tools, so
  # losing it is the worst possible failure of the feature.
  # The full contract lives in session-start-emit-matrix.bats.
  export MOCK_HOOKS_OUTPUT='{"additionalContext":"ctx"}'
  mock_archcore_multi
  local d
  d="$(make_unwired_project channel)"
  cd "$d"

  run_copilot_session_start
  assert_success

  printf '%s' "$output" | jq -e -s 'length == 1' > /dev/null 2>&1 \
    || fail "stdout is not a single JSON document: '$output'"

  local ctx
  ctx=$(printf '%s' "$output" | jq -s -r '.[0].additionalContext // ""')
  case "$ctx" in
    *ctx*) ;;
    *) fail "the CLI hook's context did not survive: '$ctx'" ;;
  esac
  case "$ctx" in
    *"$ADVISORY_MARKER"*) ;;
    *) fail "the advisory is not inside the document: '$ctx'" ;;
  esac
}
