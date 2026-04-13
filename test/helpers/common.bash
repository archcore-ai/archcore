#!/usr/bin/env bash
# Shared setup for all bats tests

PLUGIN_ROOT="${PLUGIN_ROOT:-$(cd "$(dirname "$BATS_TEST_DIRNAME")/.." && pwd)}"
export PLUGIN_ROOT
FIXTURES="$PLUGIN_ROOT/test/fixtures"
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

# Create a mock archcore CLI that handles subcommands
mock_archcore_multi() {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
case "$1" in
  validate) printf '%s\n' "$MOCK_VALIDATE_OUTPUT"; exit "${MOCK_VALIDATE_EXIT:-0}" ;;
  hooks)    printf '%s\n' "$MOCK_HOOKS_OUTPUT"; exit 0 ;;
  *)        exit 0 ;;
esac
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
