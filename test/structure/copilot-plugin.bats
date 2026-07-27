#!/usr/bin/env bats
# Structure tests: validate the native GitHub Copilot CLI adapter.

setup() {
  load '../helpers/common'
  common_setup
}

MANIFEST_REL=".plugin/plugin.json"
HOOKS_REL="hooks/copilot.hooks.json"

@test "Copilot manifest exists and is valid JSON" {
  local manifest="$PLUGIN_ROOT/$MANIFEST_REL"
  [ -f "$manifest" ]
  jq . < "$manifest" > /dev/null
}

@test "Copilot manifest points explicitly at every shared component" {
  local manifest="$PLUGIN_ROOT/$MANIFEST_REL"
  jq -e '
    .name == "archcore" and
    .hooks == "./hooks/copilot.hooks.json" and
    .mcpServers == "./.mcp.json" and
    .skills == "./skills/" and
    .agents == "./agents/" and
    .commands == "./commands/"
  ' "$manifest" > /dev/null
}

# Copilot's documented component defaults are agents/, skills/, hooks.json and
# .mcp.json — commands/ is NOT among them (copilot-host-support.rnd). Every
# other host picks the wrappers up by default; here the pointer is the only
# thing that makes /archcore:* exist at all, so it is asserted separately from
# the block above with its own reason.
@test "Copilot manifest declares commands explicitly (no default covers it)" {
  local manifest="$PLUGIN_ROOT/$MANIFEST_REL"
  [ "$(jq -r '.commands' "$manifest")" = "./commands/" ]
  [ -d "$PLUGIN_ROOT/commands" ]
}

@test "Copilot exposes the same command wrappers as the other hosts" {
  local expected actual
  expected=$(find "$PLUGIN_ROOT/commands" -name '*.md' -exec basename {} .md \; | sort)
  actual=$(find "$PLUGIN_ROOT/skills" -mindepth 1 -maxdepth 1 -type d \
    ! -name '_*' -exec basename {} \; | sort)
  [ -n "$expected" ] || fail "no command wrappers found"
  # Every wrapper must name a real skill; a wrapper pointing at a removed skill
  # would surface a /archcore:<name> entry that dead-ends on Copilot.
  local cmd
  for cmd in $expected; do
    echo "$actual" | grep -qx "$cmd" \
      || fail "commands/$cmd.md has no matching skills/$cmd/"
  done
}

@test "Copilot manifest version matches the Claude manifest" {
  local copilot="$PLUGIN_ROOT/$MANIFEST_REL"
  local claude="$PLUGIN_ROOT/.claude-plugin/plugin.json"
  [ "$(jq -r '.version' "$copilot")" = "$(jq -r '.version' "$claude")" ]
}

@test "Copilot hooks config exists and is valid version 1 JSON" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  [ -f "$hooks" ]
  jq -e '.version == 1 and (.hooks | type == "object")' "$hooks" > /dev/null
}

@test "Copilot hooks use only native camelCase lifecycle events" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  local events
  events=$(jq -r '.hooks | keys[]' "$hooks" | sort | tr '\n' ',')
  [ "$events" = "postToolUse,preToolUse,sessionStart," ] \
    || fail "unexpected Copilot hook event set: $events"
}

@test "every Copilot hook entry sets deterministic host detection" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  jq -e '
    ([.hooks[][]] | length) == 6 and
    all(.hooks[][]; .type == "command" and .env.ARCHCORE_HOST == "copilot")
  ' "$hooks" > /dev/null
}

@test "every Copilot hook runs from the user project root" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  jq -e '
    all(.hooks[][]; .cwd == ".")
  ' "$hooks" > /dev/null
}

@test "Copilot hook commands use COPILOT_PLUGIN_ROOT and the shared scripts" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  local actual expected
  actual=$(jq -r '.hooks[][] | .bash' "$hooks" | sort)
  expected=$(printf '%s\n' \
    '"${COPILOT_PLUGIN_ROOT}"/bin/check-archcore-write' \
    '"${COPILOT_PLUGIN_ROOT}"/bin/check-cascade' \
    '"${COPILOT_PLUGIN_ROOT}"/bin/check-code-alignment' \
    '"${COPILOT_PLUGIN_ROOT}"/bin/check-precision' \
    '"${COPILOT_PLUGIN_ROOT}"/bin/session-start' \
    '"${COPILOT_PLUGIN_ROOT}"/bin/validate-archcore' | sort)
  [ "$actual" = "$expected" ] || {
    echo "expected: $expected"
    echo "actual: $actual"
    fail "Copilot hook commands must route to the shared bin scripts"
  }
}

@test "Copilot preToolUse covers every native mutation tool" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  local matcher="create|edit|str_replace_editor|apply_patch"
  jq -e --arg matcher "$matcher" '
    (.hooks.preToolUse | length) == 2 and
    all(.hooks.preToolUse[];
      .matcher == $matcher and
      .timeoutSec == 1 and
      (.bash | test("/bin/check-(archcore-write|code-alignment)$"))
    )
  ' "$hooks" > /dev/null
}

@test "Copilot postToolUse self-filters through all shared validation scripts" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  jq -e '
    (.hooks.postToolUse | length) == 3 and
    all(.hooks.postToolUse[];
      (has("matcher") | not) and
      .timeoutSec == 3 and
      (.bash | test("/bin/(validate-archcore|check-cascade|check-precision)$"))
    )
  ' "$hooks" > /dev/null
}
