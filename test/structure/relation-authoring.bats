#!/usr/bin/env bats
# Relation authoring norm: an edge needs a supported claim, never proximity.

setup() {
  load '../helpers/common'
  common_setup
  PROCEDURE='skills/_shared/relation-authoring.md'
}

flat() { tr '\n' ' ' < "$1" | tr -s ' '; }

@test "every skills/_shared reference in the shipped prompts resolves to a file" {
  local ref missing=""
  while IFS= read -r ref; do
    [ -f "$PLUGIN_ROOT/$ref" ] || missing="$missing $ref"
  done < <(grep -rIhoE 'skills/_shared/[A-Za-z0-9_/.-]+\.md' "$PLUGIN_ROOT" | sort -u)
  [ -z "$missing" ] || { fail "prompts name shared files that do not exist:$missing"; return 1; }
}

@test "the procedure defines the four outcomes" {
  local outcome
  for outcome in add keep no_relation review; do
    grep -Eq "^\| $outcome \|" "$PLUGIN_ROOT/$PROCEDURE" || { fail "outcome table lacks $outcome"; return 1; }
  done
}

@test "every surface that writes or reviews relations names the procedure" {
  local file
  for file in \
    skills/_shared/gate-contract.md \
    skills/_shared/spec-contract.md \
    skills/_shared/journey-contract.md \
    skills/_shared/scenario-contract.md \
    skills/_shared/grounding/convert-routing.md \
    skills/_shared/grounding/detect-hotspots.md \
    skills/_shared/tracks/decision.md \
    skills/_shared/tracks/describe.md \
    skills/_shared/tracks/sdd.md \
    skills/init/SKILL.md \
    skills/init/lib/compose-overview.md \
    skills/init/lib/seed-compose.md \
    skills/review/SKILL.md; do
    grep -Fq "$PROCEDURE" "$PLUGIN_ROOT/$file" || { fail "$file wires relations without $PROCEDURE"; return 1; }
  done
}

@test "no prompt links documents by proximity alone" {
  local phrase hits
  for phrase in \
    'linked to its siblings with `related`' \
    'MUST link sub-specs to each other' \
    'Relate the sub-specs to each other' \
    'related to each other and to the seed facts' \
    'same domain or same module' \
    'semantically related' \
    'pairwise'; do
    hits=$(grep -rIlF -- "$phrase" "$PLUGIN_ROOT/skills" "$PLUGIN_ROOT/agents" "$PLUGIN_ROOT/copilot-agents" || true)
    [ -z "$hits" ] || { fail "proximity wording \"$phrase\" survives in: $hits"; return 1; }
  done
  if grep -Eq '\|[[:space:]]*always[[:space:]]*\|' "$PLUGIN_ROOT/skills/init/lib/compose-overview.md"; then
    fail "compose-overview.md wires an edge unconditionally"; return 1
  fi
}

@test "a split creates no edge by itself" {
  local file
  for file in \
    skills/_shared/spec-contract.md \
    skills/_shared/journey-contract.md \
    skills/_shared/scenario-contract.md \
    skills/_shared/grounding/detect-hotspots.md \
    skills/init/lib/compose-overview.md \
    skills/init/lib/seed-compose.md; do
    flat "$PLUGIN_ROOT/$file" | grep -Eq '[Ss]plitting alone creates no edge' \
      || { fail "$file can link the parts of a split without a claim"; return 1; }
  done
}

@test "a gate's required traceability edge survives the procedure" {
  flat "$PLUGIN_ROOT/skills/_shared/gate-contract.md" | grep -Fq "Required traceability remains part of the gate's checks" \
    || { fail "gate-contract.md lets the procedure drop a required edge"; return 1; }
  flat "$PLUGIN_ROOT/$PROCEDURE" | grep -Fq 'does not satisfy that gate' \
    || { fail "the procedure lets no_relation close a gate that requires an edge"; return 1; }
  flat "$PLUGIN_ROOT/$PROCEDURE" | grep -Fq 'Do not replace the required edge with `related`' \
    || { fail "the procedure lets related stand in for a required edge"; return 1; }
}

@test "one-hop grounding follows depends_on" {
  local file
  for file in skills/review/SKILL.md skills/plan/SKILL.md skills/document/SKILL.md; do
    flat "$PLUGIN_ROOT/$file" | grep -Fq '`implements`, `depends_on`, or `related` relations' \
      || { fail "$file skips depends_on when it pulls linked documents"; return 1; }
  done
  flat "$PLUGIN_ROOT/skills/_shared/tracks/closeout.md" | grep -Fq '`implements` and `depends_on` chain one hop' \
    || { fail "closeout scope cannot reach an rnd or a research"; return 1; }
}

@test "a spec depends on research and never implements it" {
  flat "$PLUGIN_ROOT/skills/_shared/tracks/sdd.md" | grep -Fq '`depends_on` → the `rnd` or `research`' \
    || { fail "sdd.md has no depends_on edge from a spec to its research"; return 1; }
  flat "$PLUGIN_ROOT/skills/_shared/tracks/research.md" | grep -Fq 'Research is never `implements` or `extends`' \
    || { fail "research.md dropped the rule that sdd.md relies on"; return 1; }
  # Split on the arrow, so a target list is read only up to the next relation.
  flat "$PLUGIN_ROOT/skills/_shared/tracks/sdd.md" | awk '{
      n = split($0, seg, "→")
      for (i = 2; i <= n; i++) {
        target = seg[i]; sub(/[.;].*/, "", target)
        if (seg[i-1] ~ /`implements`[[:space:]]*$/ && target ~ /`(rnd|research)`/) bad = 1
      }
    } END { exit bad }' \
    || { fail "sdd.md makes a research or an rnd an implements target"; return 1; }
}

@test "unlinked is the one term for a document without relations" {
  local hits
  hits=$(grep -rIli 'orphan' "$PLUGIN_ROOT/skills" "$PLUGIN_ROOT/agents" "$PLUGIN_ROOT/copilot-agents" || true)
  [ -z "$hits" ] || { fail "\"orphan\" survives beside \"unlinked\" in: $hits"; return 1; }
}

@test "an unlinked document is inventory, not a defect" {
  grep -Fq 'Any document type can remain unlinked' "$PLUGIN_ROOT/$PROCEDURE" \
    || { fail "the procedure no longer permits an unlinked document"; return 1; }
  grep -Fq 'An unlinked document or a high draft count alone is not an issue.' "$PLUGIN_ROOT/skills/review/SKILL.md" \
    || { fail "review short mode can report an unlinked document as an issue"; return 1; }
  grep -Fq 'Unlinked documents: report the count; flag a missing edge only when a specific claim requires it.' "$PLUGIN_ROOT/agents/archcore-auditor.md" \
    || { fail "the auditor can flag an unlinked document without a claim"; return 1; }
}
