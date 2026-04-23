#!/usr/bin/env bats
# Tests for bin/check-code-alignment (PreToolUse source-edit context injection).

setup() {
  load '../helpers/common'
  common_setup
}

# Build a minimal .archcore/ fixture under BATS_TEST_TMPDIR and cd into it.
# Writes a rule that mentions src/hooks/ and an ADR that mentions only src/.
setup_repo() {
  local repo="$BATS_TEST_TMPDIR/repo"
  mkdir -p "$repo/src/hooks" "$repo/.archcore/plugin"
  cat > "$repo/.archcore/plugin/hooks.rule.md" <<'EOF'
---
title: "Hook script naming"
status: accepted
---

All hook scripts live in bin/ and are referenced from src/hooks/ examples.
EOF
  cat > "$repo/.archcore/plugin/api.adr.md" <<'EOF'
---
title: "Use REST for the src/api surface"
status: accepted
---

We picked REST-over-HTTP.
EOF
  cd "$repo"
  echo "$repo"
}

# --- Silent pass paths ---

@test "no file_path → exit 0 silent" {
  run_with_stdin check-code-alignment '{"tool_name":"Write","tool_input":{}}'
  assert_success
  assert_output ""
}

@test "no .archcore/ directory → exit 0 silent" {
  cd "$BATS_TEST_TMPDIR"
  mkdir -p no-archcore/src/foo
  cd no-archcore
  run_with_stdin check-code-alignment '{"tool_name":"Write","tool_input":{"file_path":"src/foo/bar.ts"}}'
  assert_success
  assert_output ""
}

@test ".archcore/*.md path → exit 0 silent (sister hook handles)" {
  setup_repo >/dev/null
  run_with_stdin check-code-alignment '{"tool_name":"Write","tool_input":{"file_path":".archcore/foo.adr.md"}}'
  assert_success
  assert_output ""
}

@test "non-source-root path → exit 0 silent" {
  setup_repo >/dev/null
  run_with_stdin check-code-alignment '{"tool_name":"Write","tool_input":{"file_path":"README.md"}}'
  assert_success
  assert_output ""
}

@test "absolute path outside cwd → exit 0 silent" {
  setup_repo >/dev/null
  run_with_stdin check-code-alignment '{"tool_name":"Write","tool_input":{"file_path":"/somewhere/else/x.ts"}}'
  assert_success
  assert_output ""
}

@test "escape hatch ARCHCORE_DISABLE_INJECTION=1 → exit 0 silent" {
  setup_repo >/dev/null
  run sh -c "printf '%s' '{\"tool_name\":\"Write\",\"tool_input\":{\"file_path\":\"src/hooks/foo.sh\"}}' | ARCHCORE_DISABLE_INJECTION=1 '${PLUGIN_ROOT}/bin/check-code-alignment'"
  assert_success
  assert_output ""
}

# --- Injection behaviour ---

@test "source edit with matching docs emits additionalContext" {
  setup_repo >/dev/null
  run_with_stdin check-code-alignment '{"tool_name":"Write","tool_input":{"file_path":"src/hooks/foo.sh"}}'
  assert_success
  assert_output --partial 'hookEventName":"PreToolUse"'
  assert_output --partial 'additionalContext'
  assert_output --partial '[Archcore Context]'
  assert_output --partial 'src/hooks/foo.sh'
}

@test "longer matching prefix wins over shorter" {
  setup_repo >/dev/null
  run_with_stdin check-code-alignment '{"tool_name":"Write","tool_input":{"file_path":"src/hooks/foo.sh"}}'
  assert_success
  # hooks.rule matches src/hooks/ (longer, specificity 11, prio 5).
  # api.adr matches only src/ (shorter, specificity 4, prio 3).
  # Rule must appear before ADR in the output.
  local rule_pos adr_pos
  rule_pos=$(echo "$output" | grep -bo 'Hook script naming' | head -1 | cut -d: -f1)
  adr_pos=$(echo "$output" | grep -bo 'Use REST' | head -1 | cut -d: -f1)
  [ -n "$rule_pos" ] || fail "rule not present in output"
  [ -n "$adr_pos" ]  || fail "adr not present in output"
  [ "$rule_pos" -lt "$adr_pos" ] || fail "rule should appear before adr (rule=$rule_pos adr=$adr_pos)"
}

@test "top-3 truncation holds even with many matches" {
  local repo="$BATS_TEST_TMPDIR/repo-many"
  mkdir -p "$repo/src/app" "$repo/.archcore/p"
  cd "$repo"
  for i in 1 2 3 4 5; do
    cat > ".archcore/p/r${i}.rule.md" <<EOF
---
title: "Rule ${i}"
status: accepted
---

This rule mentions src/app/ module directly.
EOF
  done
  run_with_stdin check-code-alignment '{"tool_name":"Write","tool_input":{"file_path":"src/app/index.ts"}}'
  assert_success
  # At most 3 bullet lines in output.
  local bullets
  bullets=$(echo "$output" | grep -c '^- ' || true)
  [ "$bullets" -le 3 ] || fail "expected at most 3 bullets, got $bullets"
}

@test "types outside the allowlist are ignored" {
  local repo="$BATS_TEST_TMPDIR/repo-noise"
  mkdir -p "$repo/src/x" "$repo/.archcore/p"
  cd "$repo"
  cat > ".archcore/p/irrelevant.prd.md" <<'EOF'
---
title: "PRD — mentions src/x/"
status: draft
---

This PRD mentions src/x/ directly.
EOF
  cat > ".archcore/p/also-irrelevant.idea.md" <<'EOF'
---
title: "Idea mentions src/x/"
status: draft
---

src/x/
EOF
  run_with_stdin check-code-alignment '{"tool_name":"Write","tool_input":{"file_path":"src/x/y.ts"}}'
  assert_success
  # No .prd / .idea should be referenced.
  refute_output --partial 'irrelevant.prd'
  refute_output --partial 'also-irrelevant.idea'
  # No additionalContext should even be emitted (nothing rankable).
  refute_output --partial 'additionalContext'
}

@test "settings.json sourceRoots override default roots" {
  local repo="$BATS_TEST_TMPDIR/repo-roots"
  mkdir -p "$repo/custom/area" "$repo/.archcore/p"
  cd "$repo"
  cat > .archcore/settings.json <<'EOF'
{
  "codeAlignment": { "sourceRoots": ["custom"] }
}
EOF
  cat > .archcore/p/r.rule.md <<'EOF'
---
title: "Custom area rule"
status: accepted
---

This rule applies in custom/area/.
EOF
  # src/ is no longer in the allowed roots → silent
  run_with_stdin check-code-alignment '{"tool_name":"Write","tool_input":{"file_path":"src/a/b.ts"}}'
  assert_success
  assert_output ""

  # custom/ is now allowed → injection happens
  run_with_stdin check-code-alignment '{"tool_name":"Write","tool_input":{"file_path":"custom/area/foo.ts"}}'
  assert_success
  assert_output --partial 'Custom area rule'
}

# --- Multi-host ---

@test "cursor: emits additional_context (no hookSpecificOutput wrapper)" {
  setup_repo >/dev/null
  run sh -c "printf '%s' '{\"tool_name\":\"Write\",\"tool_input\":{\"file_path\":\"src/hooks/foo.sh\"},\"conversation_id\":\"c1\"}' | '${PLUGIN_ROOT}/bin/check-code-alignment'"
  assert_success
  assert_output --partial 'additional_context'
  refute_output --partial 'hookSpecificOutput'
}

# --- Non-blocking safety ---

@test "never returns non-zero exit code" {
  setup_repo >/dev/null
  # Deliberately malformed JSON input should still allow (exit 0).
  run sh -c "printf '%s' 'not json' | '${PLUGIN_ROOT}/bin/check-code-alignment'"
  assert_success
}
