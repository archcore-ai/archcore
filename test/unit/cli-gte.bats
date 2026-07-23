#!/usr/bin/env bats
# Contract specs for bin/cli-gte — the deterministic semver gate behind the
# init skill's host-wiring pre-flight (CLI >= v0.6.0).
#
# Contract:
#   - prints exactly one token: yes | no | __NO_CLI__; always exit 0;
#   - numeric field-by-field compare (the whole point: "0.10.0" >= "0.6.0",
#     which a lexical compare gets wrong);
#   - missing fields count as 0; leading 'v' and CLI banner prefixes are
#     stripped; pre-release/build suffixes after the patch number are ignored;
#   - archcore missing or unparsable --version → __NO_CLI__.

setup() {
  load '../helpers/common'
  common_setup
  GTE="$PLUGIN_ROOT/bin/cli-gte"
  [ -x "$GTE" ] || fail "bin/cli-gte missing or not executable"
}

mock_version() {
  # $1 = full --version output line
  cat > "$MOCK_BIN/archcore" <<MOCK
#!/bin/sh
[ "\$1" = "--version" ] && printf '%s\n' '$1'
exit 0
MOCK
  chmod +x "$MOCK_BIN/archcore"
}

@test "cli-gte: equal version → yes" {
  mock_version "v0.6.0"
  run "$GTE" 0.6.0
  assert_success
  assert_output "yes"
}

@test "cli-gte: newer patch → yes" {
  mock_version "v0.6.1"
  run "$GTE" 0.6.0
  assert_success
  assert_output "yes"
}

@test "cli-gte: older minor → no" {
  mock_version "v0.5.9"
  run "$GTE" 0.6.0
  assert_success
  assert_output "no"
}

@test "cli-gte: double-digit minor compares numerically (0.10.0 >= 0.6.0)" {
  # The lexical-compare trap this script exists to close.
  mock_version "v0.10.0"
  run "$GTE" 0.6.0
  assert_success
  assert_output "yes"
}

@test "cli-gte: double-digit min compares numerically (0.6.0 < 0.10.0)" {
  mock_version "v0.6.0"
  run "$GTE" 0.10.0
  assert_success
  assert_output "no"
}

@test "cli-gte: major beats minor and patch" {
  mock_version "v1.0.0"
  run "$GTE" 0.99.99
  assert_success
  assert_output "yes"
}

@test "cli-gte: missing patch field counts as 0" {
  mock_version "v0.6"
  run "$GTE" 0.6.0
  assert_success
  assert_output "yes"
}

@test "cli-gte: banner prefix and pre-release suffix are tolerated" {
  mock_version "archcore v0.7.2-beta.1 (darwin/arm64)"
  run "$GTE" 0.6.0
  assert_success
  assert_output "yes"
}

@test "cli-gte: leading v on the min argument is accepted" {
  mock_version "v0.6.0"
  run "$GTE" v0.6.0
  assert_success
  assert_output "yes"
}

@test "cli-gte: archcore not on PATH → __NO_CLI__" {
  run sh -c "PATH='$MOCK_BIN:/usr/bin:/bin' '$GTE' 0.6.0"
  assert_success
  assert_output "__NO_CLI__"
}

@test "cli-gte: unparsable --version output → __NO_CLI__" {
  mock_version "not a version at all"
  run "$GTE" 0.6.0
  assert_success
  assert_output "__NO_CLI__"
}

@test "cli-gte: no argument → __NO_CLI__, still exit 0" {
  mock_version "v0.6.0"
  run "$GTE"
  assert_success
  assert_output "__NO_CLI__"
}

@test "cli-gte: prints exactly one line" {
  mock_version "v0.6.0"
  run "$GTE" 0.6.0
  assert_success
  [ "$(printf '%s\n' "$output" | wc -l | tr -d ' ')" = "1" ] \
    || fail "expected exactly one output line, got: '$output'"
}
