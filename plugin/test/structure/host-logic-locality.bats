#!/usr/bin/env bats
# Host-logic locality (host-adapter-contract.spec): host-conditional branching
# may live ONLY in bin/lib/normalize-stdin.sh and bin/session-start. The check
# scripts consume the normalized ARCHCORE_* schema and stay host-agnostic;
# skills/agents/commands/rules must carry no host markers at all.

setup() {
  load '../helpers/common'
}

@test "ARCHCORE_HOST branching appears only in normalize-stdin.sh, session-start, and the launchers" {
  # The launchers read ARCHCORE_HOST for exactly one thing: mapping the shell
  # host id to the CLI agent id before delegating. Output shaping per host
  # lives in the CLI dialects, not here.
  local hits expected
  hits=$(grep -rl 'ARCHCORE_HOST' "$PLUGIN_ROOT/bin" | sort)
  expected=$(printf '%s\n%s\n%s\n%s\n' \
    "$PLUGIN_ROOT/bin/lib/normalize-stdin.sh" \
    "$PLUGIN_ROOT/bin/session-start" \
    "$PLUGIN_ROOT/bin/pre-tool-use" \
    "$PLUGIN_ROOT/bin/post-tool-use" | sort)
  [ "$hits" = "$expected" ] \
    || fail "ARCHCORE_HOST leaked outside the allowed files: $hits"
}

@test "launchers carry no per-host output branching (no ARCHCORE_HOST case statements)" {
  local script
  for script in pre-tool-use post-tool-use; do
    if grep -q 'case "\$ARCHCORE_HOST"' "$PLUGIN_ROOT/bin/$script"; then
      fail "bin/$script must not shape output per host — that lives in the CLI dialects"
    fi
  done
}

@test "skills/, agents/, commands/, rules/ contain no host-conditional markers" {
  local hits
  hits=$(grep -rlE 'ARCHCORE_HOST|CLAUDE_PLUGIN_ROOT|CURSOR_PLUGIN_ROOT|COPILOT_PLUGIN_ROOT|PLUGIN_ROOT' \
    "$PLUGIN_ROOT/skills" "$PLUGIN_ROOT/agents" "$PLUGIN_ROOT/commands" "$PLUGIN_ROOT/rules" 2>/dev/null || true)
  [ -z "$hits" ] || fail "host markers found in shared content: $hits"
}
