#!/usr/bin/env bats
# Structure tests: keep every copy of the CLI subcommand allowlist (and the
# version-pinned comment that points at it) in sync.
#
# Why this exists: when bin/CLI_VERSION bumps, the allowlist must be re-synced
# in several files. Catching the drift between those copies — and between the
# allowlist and the "As of <ver>" comment — closes the silent-drift hole that
# let `where` linger after the v0.3.0 -> v0.3.1 revert.
#
# Sources of truth checked here:
#   - test/structure/cli-contract.bats         (ARCHCORE_SUBCOMMANDS=)
#   - test/structure/readme-cli-references.bats(ARCHCORE_SUBCOMMANDS=)
#   - test/unit/validate-archcore.bats          (local allowed=" ... ")
#   - .archcore/plugin/cli-integration-tests.rule.md (example block)
#   - bin/CLI_VERSION                           (for the comment-vs-version test)
#
# Sibling tests:
#   - cli-contract.bats:124 — live cross-check that allowlist matches
#     `archcore --help`. CI-strict (refuses to silently skip in CI).

setup() {
  load '../helpers/common'
  common_setup
}

# Normalize a whitespace-separated token list to a sorted, deduplicated,
# space-joined string. Empty tokens are dropped.
_normalize() {
  printf '%s\n' "$1" | tr -s '[:space:]' '\n' | sed '/^$/d' | sort -u | tr '\n' ' ' | sed 's/ $//'
}

# Assert that exactly one line in $file matches $pattern. Catches the silent
# fail mode where a maintainer accidentally adds a second declaration and the
# extractor (head -1) keeps reading the stale first one.
_assert_single_declaration() {
  local file="$1" pattern="$2" label="$3"
  local count
  count=$(grep -cE "$pattern" "$file" 2>/dev/null)
  [ "$count" = "1" ] || fail "$label ($file): expected exactly 1 declaration matching /$pattern/, found $count"
}

# Extract the value of an `ARCHCORE_SUBCOMMANDS="..."` assignment. Caller must
# have already validated single-declaration via _assert_single_declaration.
_extract_archcore_subcommands_assignment() {
  grep -E '^[[:space:]]*ARCHCORE_SUBCOMMANDS=' "$1" \
    | sed -E 's/^[[:space:]]*ARCHCORE_SUBCOMMANDS=//' \
    | sed -E 's/^"//; s/"$//'
}

# Extract the value of a `local allowed=" ... "` declaration. Used for
# validate-archcore.bats:108, where the allowlist is written as a literal
# bracketed by leading/trailing spaces.
_extract_local_allowed() {
  grep -E '^[[:space:]]*local[[:space:]]+allowed=' "$1" \
    | sed -E 's/^[[:space:]]*local[[:space:]]+allowed=//' \
    | sed -E 's/^"//; s/"$//'
}

@test "allowlist is consistent across all four sources" {
  local cli_contract_file="$PLUGIN_ROOT/test/structure/cli-contract.bats"
  local readme_refs_file="$PLUGIN_ROOT/test/structure/readme-cli-references.bats"
  local validate_unit_file="$PLUGIN_ROOT/test/unit/validate-archcore.bats"
  local rule_doc_file="$PLUGIN_ROOT/.archcore/plugin/cli-integration-tests.rule.md"

  # Each source must declare the allowlist exactly once. Catches drift hidden
  # behind a second, stale declaration that the extractor would otherwise pick.
  _assert_single_declaration "$cli_contract_file"  '^[[:space:]]*ARCHCORE_SUBCOMMANDS=' "cli-contract.bats"
  _assert_single_declaration "$readme_refs_file"   '^[[:space:]]*ARCHCORE_SUBCOMMANDS=' "readme-cli-references.bats"
  _assert_single_declaration "$validate_unit_file" '^[[:space:]]*local[[:space:]]+allowed=' "validate-archcore.bats"
  _assert_single_declaration "$rule_doc_file"      '^[[:space:]]*ARCHCORE_SUBCOMMANDS=' "cli-integration-tests.rule.md"

  local cli_contract readme_refs validate_unit rule_doc
  cli_contract=$(_extract_archcore_subcommands_assignment "$cli_contract_file")
  readme_refs=$(_extract_archcore_subcommands_assignment  "$readme_refs_file")
  validate_unit=$(_extract_local_allowed                  "$validate_unit_file")
  rule_doc=$(_extract_archcore_subcommands_assignment     "$rule_doc_file")

  local n_cli n_readme n_validate n_rule
  n_cli=$(_normalize "$cli_contract")
  n_readme=$(_normalize "$readme_refs")
  n_validate=$(_normalize "$validate_unit")
  n_rule=$(_normalize "$rule_doc")

  # Reject empty/whitespace-only allowlists. Without this, four files with
  # `ARCHCORE_SUBCOMMANDS=" "` would normalize to "" each and silently match.
  [ -n "$n_cli" ]      || fail "cli-contract.bats: ARCHCORE_SUBCOMMANDS is empty after normalization"
  [ -n "$n_readme" ]   || fail "readme-cli-references.bats: ARCHCORE_SUBCOMMANDS is empty after normalization"
  [ -n "$n_validate" ] || fail "validate-archcore.bats: 'local allowed=' is empty after normalization"
  [ -n "$n_rule" ]     || fail "cli-integration-tests.rule.md: ARCHCORE_SUBCOMMANDS example is empty after normalization"

  if [ "$n_cli" = "$n_readme" ] && [ "$n_cli" = "$n_validate" ] && [ "$n_cli" = "$n_rule" ]; then
    return 0
  fi

  printf 'Allowlist drift detected. Each source should hold the same set:\n' >&2
  printf '  cli-contract.bats          : %s\n' "$n_cli"      >&2
  printf '  readme-cli-references.bats : %s\n' "$n_readme"   >&2
  printf '  validate-archcore.bats     : %s\n' "$n_validate" >&2
  printf '  cli-integration-tests.rule : %s\n' "$n_rule"     >&2
  fail "allowlist diverged across sources — re-sync after CLI bump"
}

@test "cli-contract.bats 'As of <ver>' comment matches bin/CLI_VERSION" {
  local cli_version
  cli_version=$(tr -d '[:space:]' < "$PLUGIN_ROOT/bin/CLI_VERSION")
  [ -n "$cli_version" ] || fail "bin/CLI_VERSION is empty"

  # Pull the version token out of the first `# As of <ver>` comment.
  local comment_version
  comment_version=$(grep -oE '^#[[:space:]]*As of[[:space:]]+[0-9]+\.[0-9]+\.[0-9]+' \
                      "$PLUGIN_ROOT/test/structure/cli-contract.bats" \
                      | head -1 \
                      | sed -E 's/^#[[:space:]]*As of[[:space:]]+//')

  [ -n "$comment_version" ] \
    || fail "could not find '# As of X.Y.Z' comment in cli-contract.bats"

  [ "$comment_version" = "$cli_version" ] \
    || fail "Comment in cli-contract.bats says 'As of $comment_version' but bin/CLI_VERSION is $cli_version. Update both the version mention and the listed subcommands."
}
