#!/usr/bin/env bats
# Structure tests: validate skill files

setup() {
  load '../helpers/common'
  common_setup
}

@test "every skill directory has a SKILL.md" {
  local count
  count=$(find "$PLUGIN_ROOT/skills" -name "SKILL.md" | wc -l | tr -d ' ')
  [ "$count" -ge 16 ]
}

@test "every skill has name: in frontmatter" {
  local missing=""
  for skill in "$PLUGIN_ROOT"/skills/*/SKILL.md; do
    if ! head -10 "$skill" | grep -q '^name:'; then
      missing="$missing $(basename "$(dirname "$skill")")"
    fi
  done
  [ -z "$missing" ] || fail "Skills missing name: $missing"
}

@test "every skill has description: in frontmatter" {
  local missing=""
  for skill in "$PLUGIN_ROOT"/skills/*/SKILL.md; do
    if ! head -10 "$skill" | grep -q '^description:'; then
      missing="$missing $(basename "$(dirname "$skill")")"
    fi
  done
  [ -z "$missing" ] || fail "Skills missing description: $missing"
}

@test "skill name matches directory name" {
  local mismatches=""
  for skill in "$PLUGIN_ROOT"/skills/*/SKILL.md; do
    local dir_name
    dir_name=$(basename "$(dirname "$skill")")
    local skill_name
    skill_name=$(head -10 "$skill" | grep '^name:' | sed 's/^name:[[:space:]]*//')
    if [ "$dir_name" != "$skill_name" ]; then
      mismatches="$mismatches $dir_name(=$skill_name)"
    fi
  done
  [ -z "$mismatches" ] || fail "Mismatched names: $mismatches"
}

@test "no duplicate skill names" {
  local dupes
  dupes=$(for skill in "$PLUGIN_ROOT"/skills/*/SKILL.md; do
    head -10 "$skill" | grep '^name:' | sed 's/^name:[[:space:]]*//'
  done | sort | uniq -d)
  [ -z "$dupes" ] || fail "Duplicate skill names: $dupes"
}

@test "bootstrap skill calls init_project before operating on uninitialized projects" {
  grep -q "mcp__archcore__init_project" "$PLUGIN_ROOT/skills/bootstrap/SKILL.md" \
    || fail "bootstrap/SKILL.md must instruct the agent to call mcp__archcore__init_project for uninitialized projects"
}

@test "help skill documents the archcore init CLI recovery path" {
  # /archcore:help is a likely first stop when MCP tools fail; it must explain
  # how to install the CLI and run `archcore init` to recover. This is the
  # MCP-unavailable fallback (distinct from the in-session init_project call
  # that bootstrap uses when MCP works but .archcore/ is empty).
  grep -q 'archcore init' "$PLUGIN_ROOT/skills/help/SKILL.md" \
    || fail "help/SKILL.md must include 'archcore init' recovery instruction"
}
