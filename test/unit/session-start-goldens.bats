#!/usr/bin/env bats
# Byte-identical golden pins for bin/session-start on the NON-copilot hosts.
#
# Purpose: the copilot-only additions (wiring next-step line, init-nudge fork,
# wiring advisory) must leave every other host's output untouched — not
# "still matching a --partial", but byte-identical. Each golden here is the
# exact output captured from the shipped script; any diff on these arms for
# claude-code / cursor / codex / opencode is a regression by definition.
#
# The negative tests additionally pin that no copilot-only string ever leaks
# into a non-copilot arm.
#
# All arms are static or made deterministic by the mocks, so exact equality
# is achievable without a snapshot framework. `$output` and $(cat <<'EOF')
# both strip trailing newlines — the comparison is stable.

setup() {
  load '../helpers/common'
  common_setup
}

# --- expected outputs (captured goldens) -------------------------------------

expected_cli_missing_plain() {
  cat <<'EOF'
[Archcore] CLI not found on PATH. Install it first:
  macOS/Linux/WSL: curl -fsSL https://archcore.ai/install.sh | bash
  Windows (PowerShell): irm https://archcore.ai/install.ps1 | iex
  Then verify: archcore --version
  Docs: https://docs.archcore.ai/cli/install/
EOF
}

expected_cli_missing_claude() {
  cat <<'EOF'
{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"[Archcore] CLI not found on PATH. Install it first:\n  macOS/Linux/WSL: curl -fsSL https://archcore.ai/install.sh | bash\n  Windows (PowerShell): irm https://archcore.ai/install.ps1 | iex\n  Then verify: archcore --version\n  Docs: https://docs.archcore.ai/cli/install/\n"}}
EOF
}

expected_no_archcore_plain() {
  cat <<'EOF'
[Archcore] no .archcore/ directory in this project yet. When the user asks for any Archcore operation (create ADR, audit docs, etc.), call mcp__archcore__init_project once to initialize, then proceed. Do NOT write to .archcore/ directly — hooks will block it. After init, suggest the user run /archcore:init to seed initial content (stack rule, run-the-app guide, optional import of existing CLAUDE.md / AGENTS.md / .cursorrules). Skip with ARCHCORE_HIDE_EMPTY_NUDGE=1.
EOF
}

expected_no_archcore_claude() {
  cat <<'EOF'
{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"[Archcore] no .archcore/ directory in this project yet. When the user asks for any Archcore operation (create ADR, audit docs, etc.), call mcp__archcore__init_project once to initialize, then proceed. Do NOT write to .archcore/ directly — hooks will block it. After init, suggest the user run /archcore:init to seed initial content (stack rule, run-the-app guide, optional import of existing CLAUDE.md / AGENTS.md / .cursorrules). Skip with ARCHCORE_HIDE_EMPTY_NUDGE=1.\n"}}
EOF
}

# Leading blank lines are real: one from the mocked `archcore hooks` call, one
# from the deliberate spacer echo before the nudge.
expected_empty_state() {
  cat <<'EOF'


[Archcore] .archcore/ is empty. Run /archcore:init to seed a stack rule, a run-the-app guide, and (optionally) imports from existing agent-instruction files like CLAUDE.md or AGENTS.md. Skip with ARCHCORE_HIDE_EMPTY_NUDGE=1.
EOF
}

expected_update_advisory() {
  cat <<'EOF'


[Archcore] CLI update available (v9.9.9 — installed: v0.5.7). Run: archcore update
EOF
}

expected_staleness() {
  cat <<'EOF'


[Archcore Staleness] 1 source files changed since last documentation update.
Potentially affected documents:
  - auth.adr.md — references src/ (1 files changed)
Run /archcore:audit --drift for detailed analysis.
EOF
}

# --- helpers ------------------------------------------------------------------

# stdin payload that routes normalize-stdin.sh to the given host.
stdin_for_host() {
  case "$1" in
    cursor) printf '%s' '{"conversation_id":"x"}' ;;
    codex) printf '%s' '{"turn_id":"x"}' ;;
    *) printf '%s' '{}' ;;
  esac
}

# opencode has no stdin heuristic — forced via env, like the shipped tests.
env_for_host() {
  case "$1" in
    opencode) printf '%s' "ARCHCORE_HOST=opencode" ;;
    *) printf '%s' '' ;;
  esac
}

assert_golden() {
  local expected="$1"
  [ "$output" = "$expected" ] || {
    printf 'expected:\n%s\n---\nactual:\n%s\n' "$expected" "$output" >&2
    fail "output diverged from golden"
  }
}

# Seed a >200-byte document so the empty-state nudge stays quiet.
seed_substantial_doc() {
  mkdir -p .archcore
  awk 'BEGIN { s=""; for (i=0;i<300;i++) s=s "x"; print s }' > .archcore/big.doc.md
}

# --- CLI-missing arm ----------------------------------------------------------

@test "golden: CLI-missing output is byte-stable (claude-code JSON)" {
  cd "$BATS_TEST_TMPDIR"
  run sh -c "PATH='/usr/bin:/bin'; export PATH; printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_golden "$(expected_cli_missing_claude)"
}

@test "golden: CLI-missing output is byte-stable (cursor, codex, opencode plain)" {
  local host
  for host in cursor codex opencode; do
    cd "$BATS_TEST_TMPDIR"
    run sh -c "PATH='/usr/bin:/bin'; export PATH; $(env_for_host "$host") printf '%s' '$(stdin_for_host "$host")' | $(env_for_host "$host") '${PLUGIN_ROOT}/bin/session-start'"
    assert_success
    assert_golden "$(expected_cli_missing_plain)"
  done
}

# --- missing-.archcore arm ------------------------------------------------------

@test "golden: missing-.archcore nudge is byte-stable (claude-code JSON)" {
  mock_archcore ""
  cd "$BATS_TEST_TMPDIR"
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_golden "$(expected_no_archcore_claude)"
}

@test "golden: missing-.archcore nudge is byte-stable (cursor, codex, opencode plain)" {
  mock_archcore ""
  local host
  for host in cursor codex opencode; do
    cd "$BATS_TEST_TMPDIR"
    run sh -c "printf '%s' '$(stdin_for_host "$host")' | $(env_for_host "$host") '${PLUGIN_ROOT}/bin/session-start'"
    assert_success
    assert_golden "$(expected_no_archcore_plain)"
  done
}

# --- empty-state arm ------------------------------------------------------------

@test "golden: empty-state nudge is byte-stable and host-independent" {
  mock_archcore ""
  local host
  for host in claude-code cursor codex opencode; do
    local d="$BATS_TEST_TMPDIR/empty-$host"
    mkdir -p "$d/.archcore"
    echo "stub" > "$d/.archcore/stub.md"
    cd "$d"
    run sh -c "printf '%s' '$(stdin_for_host "$host")' | $(env_for_host "$host") '${PLUGIN_ROOT}/bin/session-start'"
    assert_success
    assert_golden "$(expected_empty_state)"
  done
}

# --- update-advisory arm (host-agnostic echo; claude-code representative) -------

@test "golden: update advisory line is byte-stable" {
  mock_archcore_with_update v9.9.9 v0.5.7
  local d="$BATS_TEST_TMPDIR/update-golden"
  mkdir -p "$d"
  cd "$d"
  seed_substantial_doc
  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_golden "$(expected_update_advisory)"
}

# --- staleness arm (host-agnostic echo; claude-code representative) -------------

@test "golden: staleness passthrough is byte-stable" {
  mock_archcore ""
  local repo="$BATS_TEST_TMPDIR/stale-golden"
  mkdir -p "$repo/src/auth" "$repo/.archcore"
  cd "$repo"
  git init -q
  git config user.email "test@test.com"
  git config user.name "Test"
  echo "auth handler" > src/auth/handler.py
  {
    echo "References src/auth/ for authentication"
    awk 'BEGIN { s=""; for (i=0;i<300;i++) s=s "x"; print s }'
  } > .archcore/auth.adr.md
  git add -A && git commit -q -m "initial"
  echo "changed" >> src/auth/handler.py
  git add -A && git commit -q -m "change"

  run sh -c "printf '%s' '{}' | '${PLUGIN_ROOT}/bin/session-start'"
  assert_success
  assert_golden "$(expected_staleness)"
}

# --- negatives: copilot-only strings must never leak to other hosts -------------

@test "golden negative: no copilot-only strings on any non-copilot arm" {
  local host arm
  for host in claude-code cursor codex opencode; do
    # CLI-missing arm
    cd "$BATS_TEST_TMPDIR"
    run sh -c "PATH='/usr/bin:/bin'; export PATH; $(env_for_host "$host") printf '%s' '$(stdin_for_host "$host")' | $(env_for_host "$host") '${PLUGIN_ROOT}/bin/session-start'"
    assert_success
    refute_output --partial "init --agent copilot"
    refute_output --partial "not wired for Copilot"

    # missing-.archcore arm
    mock_archcore ""
    cd "$BATS_TEST_TMPDIR"
    run sh -c "printf '%s' '$(stdin_for_host "$host")' | $(env_for_host "$host") '${PLUGIN_ROOT}/bin/session-start'"
    assert_success
    refute_output --partial "init --agent copilot"
    refute_output --partial "not wired for Copilot"

    # initialized, unwired project (the copilot wiring-advisory precondition)
    local d="$BATS_TEST_TMPDIR/unwired-$host"
    mkdir -p "$d"
    cd "$d"
    seed_substantial_doc
    run sh -c "printf '%s' '$(stdin_for_host "$host")' | $(env_for_host "$host") '${PLUGIN_ROOT}/bin/session-start'"
    assert_success
    refute_output --partial "init --agent copilot"
    refute_output --partial "not wired for Copilot"
  done
}
