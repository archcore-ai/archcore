#!/bin/sh
# Fails a release whose binary would ship inert.
#
# Two ldflags variables decide whether a released binary does anything at all:
# `archcore-cli/internal/telemetry.apiKey` (without it no event is ever sent) and
# `archcore-cli/internal/update.officialBuild` (without it the unattended policy
# refuses at its first condition). `go tool link -X` on a path that does not name
# an existing variable is a silent no-op — no warning, no nonzero exit — so a
# package rename or a typo in .goreleaser.yaml would produce a perfectly healthy
# release that never updates anyone and reports nothing, and the only symptom
# would be a dashboard that stays empty for a month.
#
# So this checks the artifact, not the inputs: the exact values the workflow
# exported must be findable in the linked binary.
#
# The stripping step below is what makes that check mean anything. The linker
# records the whole `-ldflags` argument verbatim in the binary's build info, so
# a plain grep for the value matches the COMMAND LINE whether or not the `-X`
# actually took — which is precisely the failure this script exists to catch,
# and the first version of it passed a binary built with a misspelled symbol
# path. Removing every occurrence of the recorded argument leaves only values
# that a successful injection wrote into the string table.
#
# Called from the goreleaser post-build hook once per target
# (keeping-the-cli-current-rollout.plan, Phase 0 task 5).
# scripts/assert-not-inert-selftest.sh is its own guard: it builds a binary with
# a misspelled `-X` path and requires this script to reject it.

set -eu

binary="${1:?usage: assert-not-inert.sh <binary>}"

marker="${ARCHCORE_OFFICIAL_BUILD:-}"
key="${POSTHOG_KEY:-}"

# Neither supplied: an unofficial build — a fork's pipeline, or goreleaser run
# locally. Inert is the correct outcome there, not a failure.
if [ -z "$marker" ] && [ -z "$key" ]; then
    exit 0
fi

fail() {
    echo "assert-not-inert: $1" >&2
    echo "  binary: $binary" >&2
    exit 1
}

# One of the two set is never intentional: it means the release meant to be
# official and half its wiring is missing.
[ -n "$marker" ] || fail "POSTHOG_KEY is set but ARCHCORE_OFFICIAL_BUILD is empty — this release would never self-update"
[ -n "$key" ] || fail "ARCHCORE_OFFICIAL_BUILD is set but POSTHOG_KEY is empty — this release would report nothing"

# The recorded build info, as `build -ldflags="..."`. Absent build info fails
# rather than falls back to the naive grep: a check that cannot tell an injected
# value from a command line is worse than no check, because it reports success.
recorded=$(go version -m "$binary" 2>/dev/null |
    sed -n 's/^[[:space:]]*build[[:space:]][[:space:]]*-ldflags=//p' | head -n 1) ||
    fail "could not read the build info — 'go version -m' failed"

[ -n "$recorded" ] ||
    fail "the binary records no -ldflags build setting, so nothing was injected"

# `go version -m` quotes the value when it contains spaces, which it always does
# here. Strip one layer if present.
case "$recorded" in
    \"*\") recorded=${recorded#\"}; recorded=${recorded%\"} ;;
esac

stripped="${TMPDIR:-/tmp}/assert-not-inert.$$"
# shellcheck disable=SC2064
trap "rm -f '$stripped'" EXIT INT TERM

# -0777 slurps the whole file as one byte string; \Q..\E quotes every
# metacharacter in the recorded argument, so nothing in it is read as a pattern.
RECORDED_LDFLAGS="$recorded" perl -0777 -pe '
    BEGIN { $blob = $ENV{RECORDED_LDFLAGS} }
    s/\Q$blob\E//g
' "$binary" > "$stripped" || fail "could not strip the recorded -ldflags argument"

# -F and the full value, not a pattern: the string `phc_` is already in every
# binary as the guard's own prefix test, so a pattern match would pass on a build
# that carries no key at all.
grep -qaF -- "$marker" "$stripped" ||
    fail "the official-build marker is not in the binary — check the -X path 'archcore-cli/internal/update.officialBuild'"
grep -qaF -- "$key" "$stripped" ||
    fail "the telemetry key is not in the binary — check the -X path 'archcore-cli/internal/telemetry.apiKey'"
