#!/usr/bin/env bats

setup() {
  load '../helpers/common'
  CHECK="$WORKSPACE_ROOT/scripts/check-release-tag.sh"
  cd "$BATS_TEST_TMPDIR"
  git init -q .
  git -c user.name=t -c user.email=t@t commit -q --allow-empty -m init
}

tag() {
  local name
  for name in "$@"; do git tag "$name"; done
}

@test "the next patch, minor, or major version passes" {
  tag v0.9.9 v0.10.3 v0.10.4
  local next
  for next in v0.10.5 v0.11.0 v1.0.0; do
    run "$CHECK" "$next"
    assert_success
    assert_output --partial 'follows v0.10.4'
  done
}

@test "a re-run of the latest release passes against the tag before it" {
  tag v0.10.3 v0.10.4
  run "$CHECK" v0.10.4
  assert_success
  assert_output --partial 'follows v0.10.3'
}

@test "the first release passes without earlier tags" {
  run "$CHECK" v0.1.0
  assert_success
}

@test "a tag that skips, repeats, or goes back a version fails" {
  tag v0.10.3 v0.10.4
  local wrong
  for wrong in v0.10.6 v0.10.40 v0.1.5 v0.10.3 v0.12.0 v0.11.1; do
    run "$CHECK" "$wrong"
    assert_failure
    assert_output --partial 'does not follow v0.10.4'
  done
}

@test "a tag that is not vMAJOR.MINOR.PATCH fails" {
  local wrong
  for wrong in 0.10.5 v0.10 v0.10.5-rc.1 v01.2.3 release-1; do
    run "$CHECK" "$wrong"
    assert_failure
    assert_output --partial 'is not vMAJOR.MINOR.PATCH'
  done
}

@test "non-release tags do not count as the previous version" {
  tag v0.10.4 v9.9.9-rc.1 vnext
  run "$CHECK" v0.10.5
  assert_success
}
