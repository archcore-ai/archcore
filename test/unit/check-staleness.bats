#!/usr/bin/env bats
# Tests for bin/check-staleness

setup() {
  load '../helpers/common'
  common_setup
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

@test "many source changes without doc references suggests running actualize" {
  local repo="$BATS_TEST_TMPDIR/repo-many"
  mkdir -p "$repo/.archcore" "$repo/lib"
  cd "$repo"
  git init -q
  git config user.email "test@test.com"
  git config user.name "Test"
  echo "doc" > .archcore/unrelated.adr.md
  for i in $(seq 1 8); do echo "file$i" > "lib/file$i.py"; done
  git add -A && git commit -q -m "initial"

  # Change many files
  for i in $(seq 1 8); do echo "updated$i" > "lib/file$i.py"; done
  git add -A && git commit -q -m "bulk update"

  run "$PLUGIN_ROOT/bin/check-staleness"
  assert_success
  assert_output --partial "source files changed"
}
