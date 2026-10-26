#!/usr/bin/env bats
# The packed tree of @archcore/opencode-plugin matches its sources
# (opencode-adapter.spec, behaviors 24-26 and 29). The trees exist only after
# plugins/opencode/scripts/pack.sh has run, so every test skips without them
# on a developer machine. On CI, where `make test` packs first, a missing
# output fails instead of passing as a skip.

setup() {
  load '../helpers/common'
  common_setup
  PKG="$REPO_ROOT/plugins/opencode"
  if [ ! -f "$PKG/dist/index.js" ]; then
    [ -z "${CI:-}" ] || fail "pack output absent on CI — make test-opencode did not pack"
    skip "pack output absent — run: sh plugins/opencode/scripts/pack.sh"
  fi
}

@test "opencode package: bin/ is byte-identical to plugins/archcore/bin/" {
  run diff -r "$PLUGIN_ROOT/bin" "$PKG/bin"
  assert_success
}

@test "opencode package: agents/*.md are byte-identical to plugins/archcore/agents/*.md" {
  local src
  for src in "$PLUGIN_ROOT"/agents/*.md; do
    cmp -s "$src" "$PKG/agents/$(basename "$src")" || fail "agents/$(basename "$src") differs from its source"
  done
  [ "$(ls "$PKG"/agents | wc -l)" -eq "$(ls "$PLUGIN_ROOT"/agents/*.md | wc -l)" ] \
    || fail "agents/ holds files that have no source"
}

@test "opencode package: skills/ matches its source except the archcore- name line" {
  local rel
  while IFS= read -r rel; do
    [ -n "$rel" ] || continue
    case "$rel" in
      */SKILL.md) : ;;
      *) fail "skills/$rel differs from its source" ;;
    esac
  done < <(cd "$PKG/skills" && find . -type f | sed 's|^\./||' | while read -r f; do
    cmp -s "$f" "$PLUGIN_ROOT/skills/$f" || echo "$f"
  done)
  [ "$(cd "$PKG/skills" && find . -type f | wc -l)" -eq "$(cd "$PLUGIN_ROOT/skills" && find . -type f | wc -l)" ] \
    || fail "skills/ has a different file count than its source"
}

@test "opencode package: each SKILL.md differs only in its name: line, prefixed archcore-" {
  local skill name changed
  for skill in "$PLUGIN_ROOT"/skills/*/SKILL.md; do
    name=$(basename "$(dirname "$skill")")
    changed=$(diff "$skill" "$PKG/skills/$name/SKILL.md" | grep '^[<>]')
    [ "$changed" = "< name: $name
> name: archcore-$name" ] || fail "skills/$name/SKILL.md: unexpected diff:
$changed"
  done
}

@test "opencode package: the npm tarball carries dist/index.js and no test directory" {
  if ! command -v npm >/dev/null 2>&1; then
    [ -z "${CI:-}" ] || fail "npm not installed on CI"
    skip "npm not installed"
  fi
  local files
  files=$(cd "$PKG" && npm pack --dry-run --json --ignore-scripts 2>/dev/null | jq -r '.[0].files[].path')
  echo "$files" | grep -qx 'dist/index.js' || fail "tarball misses dist/index.js"
  echo "$files" | grep -qx 'bin/pre-tool-use' || fail "tarball misses bin/pre-tool-use"
  echo "$files" | grep -qx 'skills/plan/SKILL.md' || fail "tarball misses skills/plan/SKILL.md"
  echo "$files" | grep -qx 'agents/archcore-auditor.md' || fail "tarball misses agents/archcore-auditor.md"
  ! echo "$files" | grep -Eq '(^|/)test/' || fail "tarball contains a test directory"
  ! echo "$files" | grep -q '^src/' || fail "tarball contains src/"
}
