#!/usr/bin/env bats
# Structure tests: validate rule files

setup() {
  load '../helpers/common'
  common_setup
}

@test "archcore-context.mdc exists" {
  [ -f "$PLUGIN_ROOT/rules/archcore-context.mdc" ]
}

@test "archcore-files.mdc exists" {
  [ -f "$PLUGIN_ROOT/rules/archcore-files.mdc" ]
}

@test "archcore-context.mdc has alwaysApply: true" {
  grep -q 'alwaysApply: true' "$PLUGIN_ROOT/rules/archcore-context.mdc"
}

@test "archcore-files.mdc has globs for .archcore/" {
  grep -q 'globs:.*\.archcore' "$PLUGIN_ROOT/rules/archcore-files.mdc"
}

@test "archcore-context.mdc has description" {
  grep -q '^description:' "$PLUGIN_ROOT/rules/archcore-context.mdc"
}

@test "archcore-files.mdc has description" {
  grep -q '^description:' "$PLUGIN_ROOT/rules/archcore-files.mdc"
}
