#!/usr/bin/env bats
# Structure tests: lock-in that no distributable file references the
# plugin's bundled .archcore/ (our own dev knowledge base).
#
# The bundled .archcore/ MUST NOT ship in the public distribution (it's
# excluded by the release-strip workflow). These tests catch the other
# half of the contract: even if someone writes a skill/agent/rule that
# embeds a literal path into .archcore/plugin/foo.adr.md, the test fails.
#
# Semantic refs to ".archcore/" as a generic concept (the user's knowledge
# base) are allowed and required — those make the plugin work. The pattern
# below specifically targets path references like:
#   .archcore/plugin/cli-integration-tests.rule.md
#   .archcore/knowledge/jwt.adr.md
# which would only point at our own dev docs (no user would have those
# exact filenames).

setup() {
  load '../helpers/common'
  common_setup
}

DIST_DIRS="skills agents commands rules hooks bin"
DIST_FILES="README.md"

@test "no distributable file references bundled .archcore/<category>/<slug>.<type>.md paths" {
  local pattern='\.archcore/(plugin|knowledge|vision|experience)/[a-zA-Z0-9_-]+\.(adr|spec|prd|plan|idea|rule|guide|doc|task-type|cpat|rfc|mrd|brd|urd|brs|strs|syrs|srs)\.md'
  local hits
  hits=$(cd "$PLUGIN_ROOT" && grep -rEn "$pattern" $DIST_DIRS $DIST_FILES 2>/dev/null \
    | grep -v -E '\.archcore/<' \
    | grep -v -E '\.archcore/auth/jwt-strategy\.adr\.md' \
    | grep -v -E '\.archcore/<path>/<slug>' || true)
  # Above filters: allow generic example placeholders like
  #   `.archcore/<category>/<slug>.<type>.md` and the documented
  #   `.archcore/auth/jwt-strategy.adr.md` illustration used in skills.
  if [ -n "$hits" ]; then
    fail "distributable file references concrete bundled .archcore/ docs:
$hits

Allowed: generic concept refs to '.archcore/'.
Forbidden: paths to specific plugin-team docs (those leak into user installs)."
  fi
}

@test "no distributable file references .archcore/.sync-state.json from a literal location" {
  # Catches refs to *our* sync-state, vs the user's (which lives in the
  # same relative path but is theirs, written by their own MCP usage).
  # The hint: anything reading our sync-state would be using an absolute
  # path or referring to plugin internals.
  local pattern='/\.archcore/\.sync-state\.json|plugin-root.*\.sync-state'
  local hits
  hits=$(cd "$PLUGIN_ROOT" && grep -rEn "$pattern" $DIST_DIRS $DIST_FILES 2>/dev/null || true)
  if [ -n "$hits" ]; then
    fail "distributable file looks like it references plugin's own sync-state:
$hits"
  fi
}

@test "no distributable file imports plugin's bundled archcore settings.json" {
  # Catches accidental refs to plugin-internal settings.
  local pattern='plugin/\.archcore/settings\.json|bundled.*settings\.json'
  local hits
  hits=$(cd "$PLUGIN_ROOT" && grep -rEn "$pattern" $DIST_DIRS $DIST_FILES 2>/dev/null || true)
  if [ -n "$hits" ]; then
    fail "distributable file references plugin-bundled settings.json:
$hits"
  fi
}
