#!/usr/bin/env bats
# Release pipeline invariants
#
# `.github/workflows/release.yml` synthesizes `main` from `dev` by deleting
# a fixed list of paths (the "blocklist") before force-pushing. The contract
# is mirrored in `docs/release.md`. These tests pin both sides so a change
# to one without the other regresses loudly — and so adding a new marketplace
# surface (e.g. `assets/`) can't silently get stripped.

setup() {
  load '../helpers/common'
  common_setup
}

YML() {
  echo "$REPO_ROOT/.github/workflows/release.yml"
}

is_stripped() {
  # Matches `rm -rf <path>` or `rm -f <path>` lines, allowing trailing
  # comments. The path is anchored exactly (no prefix matches, so
  # `assets` does not match `assets-foo`).
  grep -qE "^\s*rm\s+(-rf|-f)\s+${1}(\s|$|#)" "$(YML)"
}

# --- Files that MUST ship to main (no blocklist entry) ---------------------

@test "release.yml does not strip assets/" {
  ! is_stripped 'assets' || fail "assets/ is in release.yml blocklist — composerIcon/logo will 404 in marketplace"
}

@test "release.yml does not strip docs/TERMS.md" {
  ! is_stripped 'docs/TERMS\.md' \
    || fail "docs/TERMS.md is in release.yml blocklist — termsOfServiceURL will 404"
}

@test "release.yml does not strip docs/cursor.mcp.example.json" {
  ! is_stripped 'docs/cursor\.mcp\.example\.json' \
    || fail "docs/cursor.mcp.example.json is in release.yml blocklist"
}

@test "release.yml does not strip .codex-plugin/" {
  ! is_stripped '\.codex-plugin' \
    || fail ".codex-plugin/ is in release.yml blocklist — Codex install will break"
}

@test "release.yml does not strip .agents/" {
  ! is_stripped '\.agents' \
    || fail ".agents/ is in release.yml blocklist — Codex marketplace registry will be missing"
}

@test "release.yml does not strip .claude-plugin/ or .cursor-plugin/" {
  ! is_stripped '\.claude-plugin' || fail ".claude-plugin/ is in release.yml blocklist"
  ! is_stripped '\.cursor-plugin' || fail ".cursor-plugin/ is in release.yml blocklist"
}

@test "release.yml does not strip skills/, agents/, commands/, rules/, hooks/, bin/" {
  ! is_stripped 'skills' || fail "skills/ is in blocklist"
  ! is_stripped 'agents' || fail "agents/ is in blocklist"
  ! is_stripped 'commands' || fail "commands/ is in blocklist"
  ! is_stripped 'rules' || fail "rules/ is in blocklist"
  ! is_stripped 'hooks' || fail "hooks/ is in blocklist"
  ! is_stripped 'bin' || fail "bin/ is in blocklist"
}

@test "release.yml does not strip top-level MCP configs" {
  ! is_stripped '\.mcp\.json' || fail ".mcp.json is in blocklist"
  ! is_stripped '\.codex\.mcp\.json' || fail ".codex.mcp.json is in blocklist"
}

# --- Files that MUST be stripped from main ---------------------------------

@test "release.yml strips .archcore/" {
  is_stripped '\.archcore' || fail ".archcore/ MUST be stripped — see docs/release.md"
}

@test "release.yml strips test/" {
  is_stripped 'test' || fail "test/ MUST be stripped — bats suite has no place on main"
}

@test "release.yml strips .github/, .claude/, .codex/" {
  is_stripped '\.github' || fail ".github/ MUST be stripped"
  is_stripped '\.claude' || fail ".claude/ MUST be stripped"
  is_stripped '\.codex' || fail ".codex/ MUST be stripped"
}

@test "release.yml strips Makefile and docs/release.md" {
  is_stripped 'Makefile' || fail "Makefile MUST be stripped"
  is_stripped 'docs/release\.md' || fail "docs/release.md MUST be stripped"
}

@test "release.yml strips reference-materials/" {
  is_stripped 'reference-materials' || fail "reference-materials/ MUST be stripped"
}

# --- docs/release.md mirrors workflow contract -----------------------------

@test "docs/release.md mentions assets/ in the ships list" {
  # If assets/ is in the workflow but not documented as shipping, future
  # readers may not realise it's required by the manifests.
  grep -q 'assets/' "$REPO_ROOT/docs/release.md" \
    || fail "docs/release.md must mention assets/ in the 'Everything else ships' section"
}

@test "docs/release.md mentions docs/TERMS.md in the ships list" {
  grep -q 'docs/TERMS\.md' "$REPO_ROOT/docs/release.md" \
    || fail "docs/release.md must mention docs/TERMS.md in the 'Everything else ships' section"
}
