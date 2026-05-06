#!/usr/bin/env bats
# Structure test: every code-quoted `archcore <subcmd>` reference in README.md
# must name a real CLI subcommand. README is user-facing and prescriptive;
# stale CLI instructions there mislead operators.
#
# (Internal .archcore/ design docs are intentionally excluded — they hold
# historical spec text that may legitimately reference renamed commands.)

setup() {
  load '../helpers/common'
  common_setup
}

ARCHCORE_SUBCOMMANDS="config doctor help hooks init mcp status update"

@test "every \`archcore <subcmd>\` reference in README.md names a real subcommand" {
  local file="$PLUGIN_ROOT/README.md"
  [ -f "$file" ] || skip "README.md not found"

  # Match only the first token after backtick-archcore-space, e.g. `archcore doctor`.
  # Strip flags (--fix), trailing punctuation, and pipe args.
  local refs
  refs=$(grep -oE '`archcore[[:space:]]+[a-z][a-z0-9-]*' "$file" \
    | sed -E 's/^`archcore[[:space:]]+//' \
    | sort -u)
  [ -n "$refs" ] || skip "no archcore subcommand references in README.md"

  local offenders=""
  while IFS= read -r sub; do
    [ -z "$sub" ] && continue
    case " $ARCHCORE_SUBCOMMANDS " in
      *" $sub "*) ;;
      *) offenders="$offenders $sub" ;;
    esac
  done <<< "$refs"
  [ -z "$offenders" ] || fail "README.md references phantom subcommand(s):$offenders. Allowed: $ARCHCORE_SUBCOMMANDS"
}
