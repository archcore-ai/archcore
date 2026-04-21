#!/usr/bin/env bats
# Tests for bin/check-staleness

setup() {
  load '../helpers/common'
  common_setup
  # Isolate the rate-limit stamp into a per-test location so prior runs
  # (in this test suite or on the user's machine) do not silence warnings.
  export CLAUDE_PLUGIN_DATA="$BATS_TEST_TMPDIR/plugin-data"
}

setup_git_repo() {
  local repo="$BATS_TEST_TMPDIR/repo"
  mkdir -p "$repo/src/auth" "$repo/.archcore"
  cd "$repo"
  git init -q
  git config user.email "test@test.com"
  git config user.name "Test"

  # Initial commit with source + docs
  echo "auth handler" > src/auth/handler.py
  echo "References src/auth/ for authentication" > .archcore/auth.adr.md
  git add -A && git commit -q -m "initial"
  echo "$repo"
}

@test "not a git repo exits silently" {
  cd "$BATS_TEST_TMPDIR"
  run "$PLUGIN_ROOT/bin/check-staleness"
  assert_success
  assert_output ""
}

@test "no .archcore/ commits exits silently" {
  local repo="$BATS_TEST_TMPDIR/repo-no-archcore"
  mkdir -p "$repo/src"
  cd "$repo"
  git init -q
  git config user.email "test@test.com"
  git config user.name "Test"
  echo "code" > src/app.py
  git add -A && git commit -q -m "initial"

  run "$PLUGIN_ROOT/bin/check-staleness"
  assert_success
  assert_output ""
}

@test "no changes since last doc commit exits silently" {
  local repo
  repo=$(setup_git_repo)
  cd "$repo"

  run "$PLUGIN_ROOT/bin/check-staleness"
  assert_success
  assert_output ""
}

@test "detects staleness when source changes after doc commit" {
  local repo
  repo=$(setup_git_repo)
  cd "$repo"

  # Change source file after doc commit
  echo "updated handler" > src/auth/handler.py
  git add -A && git commit -q -m "update source"

  run "$PLUGIN_ROOT/bin/check-staleness"
  assert_success
  assert_output --partial "Archcore Staleness"
  assert_output --partial "auth.adr.md"
  assert_output --partial "src"
}

@test "output contains actualize suggestion" {
  local repo
  repo=$(setup_git_repo)
  cd "$repo"

  echo "updated" > src/auth/handler.py
  git add -A && git commit -q -m "update source"

  run "$PLUGIN_ROOT/bin/check-staleness"
  assert_success
  assert_output --partial "/archcore:actualize"
}

@test "source changes without doc references exit silently" {
  # Formerly verified the "CHANGED_COUNT > 5" fallback warning.
  # Per the Inverted Invocation Policy / staleness rate-limit refactor,
  # the fallback was dropped — we only warn when there is concrete evidence
  # of affected documents.
  local repo="$BATS_TEST_TMPDIR/repo-many"
  mkdir -p "$repo/.archcore" "$repo/lib"
  cd "$repo"
  git init -q
  git config user.email "test@test.com"
  git config user.name "Test"
  echo "doc" > .archcore/unrelated.adr.md
  for i in $(seq 1 8); do echo "file$i" > "lib/file$i.py"; done
  git add -A && git commit -q -m "initial"

  for i in $(seq 1 8); do echo "updated$i" > "lib/file$i.py"; done
  git add -A && git commit -q -m "bulk update"

  run "$PLUGIN_ROOT/bin/check-staleness"
  assert_success
  assert_output ""
}

@test "rate limit suppresses repeat warnings within 24h" {
  local repo
  repo=$(setup_git_repo)
  cd "$repo"

  echo "updated" > src/auth/handler.py
  git add -A && git commit -q -m "update"

  # First run — warning emitted, stamp created.
  run "$PLUGIN_ROOT/bin/check-staleness"
  assert_success
  assert_output --partial "Archcore Staleness"

  # Second run — stamp is fresh, warning suppressed.
  run "$PLUGIN_ROOT/bin/check-staleness"
  assert_success
  assert_output ""
}

@test "rate limit lets warning through after stamp ages past 24h" {
  local repo
  repo=$(setup_git_repo)
  cd "$repo"

  echo "updated" > src/auth/handler.py
  git add -A && git commit -q -m "update"

  # First run populates the stamp.
  run "$PLUGIN_ROOT/bin/check-staleness"
  assert_success
  assert_output --partial "Archcore Staleness"

  # Age the stamp to >24h ago (86400s + 1 buffer).
  local stamp="$CLAUDE_PLUGIN_DATA/archcore/last-staleness"
  echo "$(($(date +%s) - 86401))" > "$stamp"

  run "$PLUGIN_ROOT/bin/check-staleness"
  assert_success
  assert_output --partial "Archcore Staleness"
}

@test "corrupt stamp is treated as missing" {
  local repo
  repo=$(setup_git_repo)
  cd "$repo"

  echo "updated" > src/auth/handler.py
  git add -A && git commit -q -m "update"

  # Pre-create a stamp with garbage content.
  mkdir -p "$CLAUDE_PLUGIN_DATA/archcore"
  echo "not-a-number" > "$CLAUDE_PLUGIN_DATA/archcore/last-staleness"

  run "$PLUGIN_ROOT/bin/check-staleness"
  assert_success
  assert_output --partial "Archcore Staleness"
}
