#!/usr/bin/env bats
# No probe harness residue reaches the shipped plugin.
#
# hooks-validation-system.spec conformance 13 says no committed code contains a
# probe line, and that clause is the reason the first attempt at a probe
# protocol stalled: the only way anyone knew to observe a delegated hook call
# was to add a marker to a real script, which the conformance rule forbids and
# review would have caught anyway.
#
# mkprobe resolves it by wrapping scripts in a COPY. That only holds while
# nobody takes the shortcut, so this file asserts the shape rather than trusting
# the habit: the harness lives under test/, and plugins/ carries no wrapper, no
# .real sibling, and no probe marker.

setup() {
  load '../helpers/common'
  common_setup
}

@test "the probe harness lives under test/, not in the plugin" {
  [ -f "$REPO_ROOT/test/probe/mkprobe" ] || fail "test/probe/mkprobe is missing"
  [ -x "$REPO_ROOT/test/probe/mkprobe" ] || fail "test/probe/mkprobe is not executable"
  [ ! -e "$PLUGIN_ROOT/probe" ] || fail "a probe directory exists inside the plugin"
}

@test "no .real wrapper siblings under plugins/" {
  local found
  found=$(find "$REPO_ROOT/plugins" -name '*.real' -print)
  [ -z "$found" ] || fail "probe wrapper leftovers in the shipped tree: $found"
}

@test "no probe markers in any shipped file" {
  # ARCHCORE_PROBE is the wrapper's own marker; ARCHCORE_PROBE_LOG is the env
  # var it writes to. Either one inside plugins/ means the harness was applied
  # in place instead of to a copy.
  local hits
  hits=$(grep -rl 'ARCHCORE_PROBE' "$REPO_ROOT/plugins" 2>/dev/null || true)
  [ -z "$hits" ] || fail "probe markers in shipped files: $hits"
}

@test "mkprobe leaves the working tree untouched" {
  # The strongest form of the check: actually run it, then look at the plugin.
  # A future change that writes into plugins/archcore instead of the copy fails
  # here even if it never uses the string ARCHCORE_PROBE.
  local dest before after
  dest="$BATS_TEST_TMPDIR/probe"
  before=$(find "$PLUGIN_ROOT" -type f | sort | md5 2>/dev/null || find "$PLUGIN_ROOT" -type f | sort | md5sum)

  run env REPO_ROOT="$REPO_ROOT" "$REPO_ROOT/test/probe/mkprobe" "$dest"
  assert_success

  after=$(find "$PLUGIN_ROOT" -type f | sort | md5 2>/dev/null || find "$PLUGIN_ROOT" -type f | sort | md5sum)
  [ "$before" = "$after" ] || fail "mkprobe changed the file list under $PLUGIN_ROOT"

  # And it did produce a wrapped copy, so the check above is not vacuous.
  [ -f "$dest/plugin-src/bin/check-archcore-write.real" ] \
    || fail "mkprobe produced no wrapped copy — the invariance check proves nothing"
}

@test "the probe project is a sibling of the plugin copy, never nested inside it" {
  # bin/session-start walks upward for a plugin manifest and exits silently when
  # it finds one. A probe project inside the plugin copy would therefore produce
  # no session-start line at all, and P0 would read as "hooks broken" instead of
  # "harness built wrong".
  local dest
  dest="$BATS_TEST_TMPDIR/probe-layout"
  run env REPO_ROOT="$REPO_ROOT" "$REPO_ROOT/test/probe/mkprobe" "$dest"
  assert_success

  [ -d "$dest/plugin-src" ] || fail "no plugin-src in the probe tree"
  [ -d "$dest/probe-project" ] || fail "no probe-project in the probe tree"
  [ ! -e "$dest/plugin-src/probe-project" ] \
    || fail "probe-project is nested inside plugin-src — P0 cannot pass"
}
