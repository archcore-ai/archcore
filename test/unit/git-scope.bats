#!/usr/bin/env bats
# Tests for bin/git-scope — working-tree scope resolver for /archcore:context --git-changes.

setup() {
  load '../helpers/common'
  common_setup
}

# Repo with one commit. Echoes repo path (cwd unchanged for caller — cd it yourself).
init_repo() {
  local repo="$BATS_TEST_TMPDIR/repo"
  mkdir -p "$repo"
  cd "$repo" || return 1
  git init -q
  git config user.email "test@test.com"
  git config user.name "Test"
  git config commit.gpgsign false
  mkdir -p src/api src/auth .archcore
  echo "h" > src/api/handler.js
  echo "a" > src/auth/login.js
  echo "doc" > .archcore/x.adr.md
  git add -A && git commit -q -m "initial"
  echo "$repo"
}

@test "missing flag -> __USAGE__" {
  cd "$BATS_TEST_TMPDIR"
  run "$PLUGIN_ROOT/bin/git-scope"
  assert_success
  assert_output "__USAGE__"
}

@test "unknown flag -> __USAGE__" {
  cd "$BATS_TEST_TMPDIR"
  run "$PLUGIN_ROOT/bin/git-scope" --bogus
  assert_success
  assert_output "__USAGE__"
}

@test "not a git repo -> __NOT_REPO__" {
  cd "$BATS_TEST_TMPDIR"
  run "$PLUGIN_ROOT/bin/git-scope" --git-changes
  assert_success
  assert_output "__NOT_REPO__"
}

@test "clean tree -> __CLEAN__" {
  local repo
  repo=$(init_repo)
  cd "$repo"
  run "$PLUGIN_ROOT/bin/git-scope" --git-changes
  assert_success
  assert_output "__CLEAN__"
}

@test "dirty: staged + unstaged + untracked across dirs, excludes .archcore" {
  local repo
  repo=$(init_repo)
  cd "$repo"
  echo "change" >> src/api/handler.js            # unstaged tracked
  echo "new" > src/auth/extra.js
  git add src/auth/extra.js                       # staged
  mkdir -p src/web
  echo "u" > src/web/page.js                      # untracked
  echo "edit" >> .archcore/x.adr.md               # must be excluded
  run "$PLUGIN_ROOT/bin/git-scope" --git-changes
  assert_success
  assert_line "src/api"
  assert_line "src/auth"
  assert_line "src/web"
  assert_line "__TOTAL__ 3"
  refute_line --partial ".archcore"
}

@test "only .archcore changes -> __CLEAN__" {
  local repo
  repo=$(init_repo)
  cd "$repo"
  echo "more" >> .archcore/x.adr.md               # tracked .archcore edit
  echo "newrule" > .archcore/y.rule.md            # untracked .archcore file
  run "$PLUGIN_ROOT/bin/git-scope" --git-changes
  assert_success
  assert_output "__CLEAN__"
}

@test "fresh repo (no commits) with untracked file -> returns dir" {
  local repo="$BATS_TEST_TMPDIR/fresh"
  mkdir -p "$repo"
  cd "$repo"
  git init -q
  git config user.email "t@t.co"
  git config user.name "T"
  mkdir -p src
  echo "x" > src/new.js
  run "$PLUGIN_ROOT/bin/git-scope" --git-changes
  assert_success
  assert_line "src"
  assert_line "__TOTAL__ 1"
}

@test "root-level and subdir changes both contribute scopes" {
  local repo
  repo=$(init_repo)
  cd "$repo"
  echo "root" > rootfile.txt                      # untracked root file -> dirname "."
  mkdir -p src/x
  echo "s" > src/x/a.js                            # untracked subdir file
  run "$PLUGIN_ROOT/bin/git-scope" --git-changes
  assert_success
  assert_line "."
  assert_line "src/x"
  assert_line "__TOTAL__ 2"
}

@test "only root-level changes -> repo-root scope" {
  local repo
  repo=$(init_repo)
  cd "$repo"
  echo "root" > rootonly.txt
  run "$PLUGIN_ROOT/bin/git-scope" --git-changes
  assert_success
  assert_line "."
  assert_line "__TOTAL__ 1"
}

@test "directory names with spaces are preserved" {
  local repo
  repo=$(init_repo)
  cd "$repo"
  mkdir -p "src/my dir"
  echo "space" > "src/my dir/file.js"
  run "$PLUGIN_ROOT/bin/git-scope" --git-changes
  assert_success
  assert_line "src/my dir"
  assert_line "__TOTAL__ 1"
}
