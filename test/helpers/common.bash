#!/usr/bin/env bash
# Shared setup for all bats tests

# REPO_ROOT = repository root (marketplace catalogs, test/, Makefile, .github, docs/ live here).
# PLUGIN_ROOT = the plugin itself, relocated under plugins/archcore/ so Codex can discover it
# (Codex requires source.path to be a subdirectory, not the marketplace root — see issue #2).
REPO_ROOT="${REPO_ROOT:-$(cd "$(dirname "$BATS_TEST_DIRNAME")/.." && pwd)}"
export REPO_ROOT
PLUGIN_ROOT="${PLUGIN_ROOT:-$REPO_ROOT/plugins/archcore}"
export PLUGIN_ROOT
FIXTURES="$REPO_ROOT/test/fixtures"
export FIXTURES

HELPERS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
load "${HELPERS_DIR}/bats-support/load"
load "${HELPERS_DIR}/bats-assert/load"

common_setup() {
  MOCK_BIN="$BATS_TEST_TMPDIR/mock-bin"
  mkdir -p "$MOCK_BIN"
  # Provide timeout shim for macOS (which lacks coreutils timeout)
  if ! command -v timeout >/dev/null 2>&1; then
    cat > "$MOCK_BIN/timeout" <<'SHIM'
#!/bin/sh
shift  # skip the timeout duration argument
exec "$@"
SHIM
    chmod +x "$MOCK_BIN/timeout"
  fi
  export PATH="$MOCK_BIN:$PLUGIN_ROOT/bin:$PATH"
  # Belt-and-suspenders: keep advisory/staleness rate-limit stamps inside the
  # test sandbox so no test can ever write into the developer's real
  # ~/.local/share/archcore-plugin/. Tests that exercise the XDG/HOME fallback
  # chain unset this explicitly.
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/plugin-data"
}

setup() {
  common_setup
}

# Create a mock archcore CLI that outputs given text
mock_archcore() {
  local output="$1"
  local exit_code="${2:-0}"
  cat > "$MOCK_BIN/archcore" <<MOCK
#!/bin/sh
printf '%s\n' '${output}'
exit ${exit_code}
MOCK
  chmod +x "$MOCK_BIN/archcore"
}

# Create a mock archcore CLI that handles subcommands.
# If MOCK_ARCHCORE_LOG is set in the test, every invocation appends the FULL
# argument list ("$*") to that file. Lets tests assert exactly how archcore
# was called — subcommand AND flags (e.g. `update --check` vs a bare
# `update`, which would be a real self-update in production).
mock_archcore_multi() {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
[ -n "$MOCK_ARCHCORE_LOG" ] && printf '%s\n' "$*" >> "$MOCK_ARCHCORE_LOG"
case "$1" in
  doctor) printf '%s\n' "$MOCK_DOCTOR_OUTPUT"; exit "${MOCK_DOCTOR_EXIT:-0}" ;;
  hooks)  printf '%s\n' "$MOCK_HOOKS_OUTPUT"; exit 0 ;;
  *)      exit 0 ;;
esac
MOCK
  chmod +x "$MOCK_BIN/archcore"
}

# Like mock_archcore but also logs the full invocation ("$*") to
# MOCK_ARCHCORE_LOG. Use when the test needs to assert *how* archcore was
# called (subcommand + flags), not just the script's stdout.
mock_archcore_logging() {
  local output="$1"
  local exit_code="${2:-0}"
  cat > "$MOCK_BIN/archcore" <<MOCK
#!/bin/sh
[ -n "\$MOCK_ARCHCORE_LOG" ] && printf '%s\n' "\$*" >> "\$MOCK_ARCHCORE_LOG"
printf '%s\n' '${output}'
exit ${exit_code}
MOCK
  chmod +x "$MOCK_BIN/archcore"
}

# Mock archcore whose `update --check` reports a pending update ($1, default
# v9.9.9) and whose --version prints $2 (default v0.5.7); everything else
# (hooks, doctor, ...) answers quietly with exit 0. A bare `update` (a REAL
# self-update — forbidden from hooks) exits 1 so any regression that drops
# `--check` surfaces in both the log and the advisory output.
mock_archcore_with_update() {
  local latest="${1:-v9.9.9}"
  local installed="${2:-v0.5.7}"
  cat > "$MOCK_BIN/archcore" <<MOCK
#!/bin/sh
[ -n "\$MOCK_ARCHCORE_LOG" ] && printf '%s\n' "\$*" >> "\$MOCK_ARCHCORE_LOG"
case "\$1" in
  update)
    if [ "\$2" = "--check" ]; then echo "update available: ${latest}"; exit 0; fi
    echo "unexpected real self-update invoked" >&2; exit 1 ;;
  --version) echo "${installed}" ;;
  *) echo "" ;;
esac
exit 0
MOCK
  chmod +x "$MOCK_BIN/archcore"
}

# Run a bin script with stdin from a fixture file
run_with_fixture() {
  local script="$1"
  local fixture="$2"
  run sh -c "cat '${FIXTURES}/stdin/${fixture}' | '${PLUGIN_ROOT}/bin/${script}'"
}

# Run a bin script with inline stdin
run_with_stdin() {
  local script="$1"
  local stdin_data="$2"
  run sh -c "printf '%s' '${stdin_data}' | '${PLUGIN_ROOT}/bin/${script}'"
}

# Run a bin script with stdin from a fixture file and a forced ARCHCORE_HOST.
# Needed for env-only hosts (opencode) that have no stdin detection heuristic.
run_with_fixture_env() {
  local script="$1"
  local fixture="$2"
  local env_host="$3"
  run sh -c "cat '${FIXTURES}/stdin/${fixture}' | ARCHCORE_HOST='${env_host}' '${PLUGIN_ROOT}/bin/${script}'"
}

# Source normalize-stdin.sh with given stdin and print exported vars
run_normalizer() {
  local stdin_data="$1"
  run sh -c "printf '%s' '${stdin_data}' | sh -c '
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    printf \"HOST=%s\n\" \"\$ARCHCORE_HOST\"
    printf \"TOOL=%s\n\" \"\$ARCHCORE_TOOL_NAME\"
    printf \"FILE=%s\n\" \"\$ARCHCORE_FILE_PATH\"
    printf \"DOC=%s\n\" \"\$ARCHCORE_DOC_PATH\"
  '"
}

# Source normalize-stdin.sh with env override and print vars
run_normalizer_with_env() {
  local stdin_data="$1"
  local env_host="$2"
  run sh -c "printf '%s' '${stdin_data}' | ARCHCORE_HOST='${env_host}' sh -c '
    . \"${PLUGIN_ROOT}/bin/lib/normalize-stdin.sh\"
    printf \"HOST=%s\n\" \"\$ARCHCORE_HOST\"
    printf \"TOOL=%s\n\" \"\$ARCHCORE_TOOL_NAME\"
    printf \"FILE=%s\n\" \"\$ARCHCORE_FILE_PATH\"
    printf \"DOC=%s\n\" \"\$ARCHCORE_DOC_PATH\"
  '"
}
