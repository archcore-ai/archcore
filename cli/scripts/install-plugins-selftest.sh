#!/usr/bin/env bash
# Exercises install_detected_plugins from install.sh against stub binaries.
# The install smoke jobs cannot reach that step: they run under GITHUB_ACTIONS,
# which the step treats as CI and skips.
set -euo pipefail

cd "$(dirname "$0")/.."
unset CI GITHUB_ACTIONS GITLAB_CI BUILDKITE JENKINS_URL TEAMCITY_VERSION ARCHCORE_SKIP_PLUGIN_INSTALL
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
sed '$d' install.sh > "$work/install-functions.sh"
# shellcheck source=/dev/null
source "$work/install-functions.sh"
set +e
failures=0

fail() {
    printf 'FAIL: %s\n' "$1" >&2
    failures=$((failures + 1))
}

# setup <help text> <agent that fails or ""> <host CLIs...>
setup() {
    rm -rf "${work:?}/bin" "${work:?}/log"
    mkdir -p "$work/bin"
    : > "$work/log"
    cat > "$work/bin/archcore" <<EOF
#!/bin/sh
if [ "\$3" = "--help" ]; then echo "$1"; exit 0; fi
echo "\${ARCHCORE_PLUGIN_INSTALLER:-unset} \$*" >> "$work/log"
[ "\$4" = "$2" ] && exit 1
exit 0
EOF
    shift 2
    for host in "$@"; do
        printf '#!/bin/sh\nexit 0\n' > "$work/bin/$host"
    done
    chmod +x "$work/bin"/*
}

run() {
    PATH="$work/bin:/usr/bin:/bin" install_detected_plugins "$work/bin/archcore" > /dev/null 2>&1
}

setup "ARCHCORE_PLUGIN_INSTALLER=1 is the platform installer's mode" "" claude codex
run || fail "a successful run returned nonzero"
expected="1 plugin install --agent claude-code
1 plugin install --agent codex-cli"
[[ "$(cat "$work/log")" == "$expected" ]] || fail "unexpected calls: $(cat "$work/log")"

setup "ARCHCORE_PLUGIN_INSTALLER=1" claude-code claude codex copilot
run && fail "a failed host returned zero"
[[ "$(wc -l < "$work/log" | tr -d ' ')" == "3" ]] || fail "a failed host stopped the other hosts: $(cat "$work/log")"

setup "Install the Archcore plugin on the selected hosts." "" claude
run || fail "an old CLI returned nonzero"
[[ ! -s "$work/log" ]] || fail "an old CLI without installer mode was called: $(cat "$work/log")"

setup "ARCHCORE_PLUGIN_INSTALLER=1" "" claude
ARCHCORE_SKIP_PLUGIN_INSTALL=1 run || fail "a skipped run returned nonzero"
[[ ! -s "$work/log" ]] || fail "ARCHCORE_SKIP_PLUGIN_INSTALL=1 did not skip"

setup "ARCHCORE_PLUGIN_INSTALLER=1" "" claude
ARCHCORE_SKIP_PLUGIN_INSTALL=0 run || fail "ARCHCORE_SKIP_PLUGIN_INSTALL=0 returned nonzero"
[[ -s "$work/log" ]] || fail "ARCHCORE_SKIP_PLUGIN_INSTALL=0 skipped the step"

setup "ARCHCORE_PLUGIN_INSTALLER=1" "" claude
CI=true run || fail "a CI run returned nonzero"
[[ ! -s "$work/log" ]] || fail "CI did not skip"

if ((failures > 0)); then
    exit 1
fi
echo "OK: install_detected_plugins"
