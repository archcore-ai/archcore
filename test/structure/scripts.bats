#!/usr/bin/env bats
# Structure tests: validate bin scripts

setup() {
  load '../helpers/common'
  common_setup
}

@test "all bin scripts are executable" {
  local not_exec=""
  for f in "$PLUGIN_ROOT"/bin/check-* "$PLUGIN_ROOT"/bin/validate-* "$PLUGIN_ROOT"/bin/session-start; do
    [ -f "$f" ] || continue
    if [ ! -x "$f" ]; then
      not_exec="$not_exec $(basename "$f")"
    fi
  done
  [ -z "$not_exec" ] || fail "Not executable: $not_exec"
}

@test "all bin scripts have #!/bin/sh shebang" {
  local bad_shebang=""
  for f in "$PLUGIN_ROOT"/bin/check-* "$PLUGIN_ROOT"/bin/validate-* "$PLUGIN_ROOT"/bin/session-start; do
    [ -f "$f" ] || continue
    local first_line
    first_line=$(head -1 "$f")
    if [ "$first_line" != "#!/bin/sh" ]; then
      bad_shebang="$bad_shebang $(basename "$f")"
    fi
  done
  [ -z "$bad_shebang" ] || fail "Bad shebang: $bad_shebang"
}

@test "normalize-stdin.sh exists and has shebang" {
  [ -f "$PLUGIN_ROOT/bin/lib/normalize-stdin.sh" ]
  local first_line
  first_line=$(head -1 "$PLUGIN_ROOT/bin/lib/normalize-stdin.sh")
  [ "$first_line" = "#!/bin/sh" ]
}

@test "check-archcore-write sources normalize-stdin.sh" {
  grep -q 'normalize-stdin.sh' "$PLUGIN_ROOT/bin/check-archcore-write"
}

@test "validate-archcore sources normalize-stdin.sh" {
  grep -q 'normalize-stdin.sh' "$PLUGIN_ROOT/bin/validate-archcore"
}

@test "check-cascade sources normalize-stdin.sh" {
  grep -q 'normalize-stdin.sh' "$PLUGIN_ROOT/bin/check-cascade"
}

@test "session-start sources normalize-stdin.sh" {
  grep -q 'normalize-stdin.sh' "$PLUGIN_ROOT/bin/session-start"
}

@test "check-staleness does NOT source normalize-stdin.sh" {
  ! grep -q 'normalize-stdin.sh' "$PLUGIN_ROOT/bin/check-staleness"
}
