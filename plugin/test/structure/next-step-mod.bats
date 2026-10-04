#!/usr/bin/env bats
# Structure tests: the next-step mod ships inside the Claude Code plugin
# (claude-code-hint-mod.adr) and stays inside its stack-and-tooling exception.

setup() {
  load '../helpers/common'
  common_setup
  HOOKS_JSON="$PLUGIN_ROOT/hooks/hooks.json"
  MANIFEST="$PLUGIN_ROOT/.claude-plugin/plugin.json"
  MODULE="$PLUGIN_ROOT/hooks/next-step.tsx"
}

@test "hooks.json names exactly one hooks module, and it exists" {
  run jq -r '.modules | length' "$HOOKS_JSON"
  assert_output "1"
  run jq -r '.modules[0]' "$HOOKS_JSON"
  assert_output "./next-step.tsx"
  [ -f "$MODULE" ]
}

@test "hooks.json keeps its settings hooks beside the module" {
  run jq -r '.hooks | keys | sort | join(",")' "$HOOKS_JSON"
  assert_output "PostToolUse,PreToolUse,SessionStart"
}

@test "the Claude manifest names the mod's state contract, and it exists" {
  run jq -r '.types' "$MANIFEST"
  assert_output "./types/index.d.ts"
  [ -f "$PLUGIN_ROOT/types/index.d.ts" ]
}

@test "the next_step_hints option is a boolean that defaults to on" {
  run jq -r '.userConfig.next_step_hints.type' "$MANIFEST"
  assert_output "boolean"
  run jq -r '.userConfig.next_step_hints.default' "$MANIFEST"
  assert_output "true"
}

@test "only the Claude Code hook config names the module" {
  local other
  for other in cursor.hooks.json codex.hooks.json copilot.hooks.json; do
    run jq -r 'has("modules")' "$PLUGIN_ROOT/hooks/$other"
    assert_output "false"
  done
}

@test "the module computes no coverage or drift (engine-runtime-boundary.adr)" {
  # The empty-search hint names path_ref as a filter of the agent's own search; the
  # module never sends one, and never reads a match's specificity.
  run grep -n -E "path_ref:|specificity" "$MODULE"
  assert_failure
}

@test "TypeScript in plugins/archcore is only the mod and its tests" {
  local found
  found=$(find "$PLUGIN_ROOT" \( -name '*.ts' -o -name '*.tsx' \) \
    -not -path "$PLUGIN_ROOT/.claude-plugin/types/*" \
    -not -path "$PLUGIN_ROOT/hooks/next-step.tsx" \
    -not -path "$PLUGIN_ROOT/types/index.d.ts" \
    -not -path "$PLUGIN_ROOT/tests/*.test.ts" -print)
  [ -z "$found" ] || fail "TypeScript outside the mod exception: $found"
}

@test "export ships the mod and drops the types Claude Code generates" {
  # Not $output: `run` overwrites that variable with the command's stdout.
  local dist="$BATS_TEST_TMPDIR/export"
  run "$WORKSPACE_ROOT/scripts/export-plugin.sh" "$dist" 1.2.3
  assert_success
  [ -f "$dist/plugins/archcore/hooks/next-step.tsx" ]
  [ -f "$dist/plugins/archcore/types/index.d.ts" ]
  [ ! -e "$dist/plugins/archcore/.claude-plugin/types" ]
  [ ! -e "$dist/plugins/archcore/tsconfig.json" ]
}
