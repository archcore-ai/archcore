#!/usr/bin/env bats
# Structure tests: the plan → review and document → review loop has no dead end.
# Each test pins one transition a user relies on to leave a stopped, open, or
# unfinished state; phrases are matched on flattened prose so re-wrapping holds.

setup() {
  load '../helpers/common'
  common_setup
  SHARED="$PLUGIN_ROOT/skills/_shared"
  GATE="$SHARED/gate-contract.md"
  ELICIT="$SHARED/elicitation-contract.md"
  TRACKS="$SHARED/tracks"
  PLAN_SKILL="$PLUGIN_ROOT/skills/plan/SKILL.md"
  DOC_SKILL="$PLUGIN_ROOT/skills/document/SKILL.md"
  REVIEW_SKILL="$PLUGIN_ROOT/skills/review/SKILL.md"
}

flat() { tr '\n' ' ' < "$1" | sed 's/  */ /g'; }

gate_of() {
  awk -v h="### gate: $2" '$0 == h { f = 1; next } f && /^### / { exit } f' "$1" | tr '\n' ' ' | sed 's/  */ /g'
}

@test "a resume starts a fresh question ceiling, so a blocked gate cannot loop on a spent budget" {
  [[ "$(flat "$ELICIT")" == *'A new invocation, including a resume, starts a fresh ceiling; the `budget` field of a resumed state block does not reduce it.'* ]] \
    || fail "elicitation-contract.md lets a recorded budget block every resume"
  [[ "$(flat "$GATE")" == *'The `budget` field never reduces a later invocation'"'"'s question ceiling'* ]] \
    || fail "gate-contract.md leaves the budget field's meaning open"
}

@test "a stopped gate names the ways to continue and only a recorded delegation passes it" {
  local elicit
  elicit=$(flat "$ELICIT")
  [[ "$elicit" == *'answer it now, delegate it ("you decide"), or proceed anyway'* ]] \
    || fail "a stopped gate does not tell the user how to continue"
  [[ "$elicit" == *'treat every open material question at that gate as explicitly delegated'* ]] \
    || fail "\"proceed anyway\" has no defined effect"
  [[ "$elicit" == *'An `[assumption]` mark alone is not a delegation.'* ]] \
    || fail "an interrupted [assumption] can pass as a delegation"
  [[ "$elicit" == *'record it in `deferred` with the reason `interrupted`'* ]] \
    || fail "an interruption at sdd.design leaves no record"
  [[ "$(flat "$PLAN_SKILL")" == *'WHEN a blocking exit check stopped the route, skip this step and go to Result.'* ]] \
    || fail "plan/SKILL.md offers the implement fork after a blocked route"
  [[ "$(flat "$PLAN_SKILL")" == *'the resume command `/archcore:plan <spec title>`'* ]] \
    || fail "plan/SKILL.md names a resume command that fails off the branch"
}

@test "resume rules settle precedence, ownership, and several open drafts" {
  local gate
  gate=$(flat "$GATE")
  [[ "$gate" == *'MUST resume at that gate even when its `skip_when` holds.'* ]] \
    || fail "resume and skip_when have no precedence"
  [[ "$gate" == *'except at a gate that resume rule 7 selects'* ]] \
    || fail "execution rule 1 contradicts resume rule 7"
  [[ "$gate" == *'IF the state block names a track that the executing command does not run, THEN the executing skill MUST NOT write to that draft.'* ]] \
    || fail "another command can write into a foreign track's draft"
  [[ "$gate" == *'MUST name the command that resumes it.'* ]] \
    || fail "a foreign block leaves the user without a resume command"
  [[ "$gate" == *'MUST list them and ask one question, recommending the most recently modified draft.'* ]] \
    || fail "several open drafts have no selection rule"
  [[ "$(flat "$DOC_SKILL")" == *'resume it per the resume rules in `skills/_shared/gate-contract.md` instead of opening a new track'* ]] \
    || fail "/archcore:document never resumes its own drafts"
}

@test "a computed route keeps its state block until the route ends" {
  [[ "$(flat "$SHARED/delta-routing.md")" == *'WHILE a computed route has instruments left, keep the `archcore:track` state block'* ]] \
    || fail "a session break between instruments loses the route"
  [[ "$(flat "$GATE")" == *'the track exits at the route'"'"'s last instrument'* ]] \
    || fail "gate-contract.md removes the block after each instrument"
}

@test "closeout never accepts unfinished work or an rfc" {
  local accept
  accept=$(gate_of "$TRACKS/closeout.md" closeout.accept)
  [[ "$accept" == *'blocking: no status transition targets a document that carries an `archcore:track` state block'* ]] \
    || fail "closeout can accept a draft with an open state block"
  [[ "$accept" == *'blocking: no status transition targets an `rfc`'* ]] \
    || fail "closeout can accept an rfc outside decision.resolve"
  [[ "$(flat "$TRACKS/actualize.md")" == *'for an `rfc`, name `/archcore:document decision` to resolve it instead of a status change'* ]] \
    || fail "the temporal fix can change an rfc status"
}

@test "review reports open tracks and always names a next command" {
  local review
  review=$(flat "$REVIEW_SKILL")
  [[ "$review" == *'List open tracks in scope'* ]] || fail "review ignores drafts with an open state block"
  [[ "$review" == *'a `code-wrong` finding: fix the code against the governing document'* ]] \
    || fail "a code-wrong finding has no next action"
  [[ "$review" == *'a coverage gap: `/archcore:document code <subject>`'* ]] || fail "a coverage gap has no next action"
  [[ "$review" == *'drafts the branch created with no matched `plan`: `/archcore:review closeout`'* ]] \
    || fail "unplanned drafts never reach an acceptance offer"
  [[ "$review" == *'In `closeout` and `experience` modes, the `on-default-branch` and `empty-diff` sentinels do not fall back to project health.'* ]] \
    || fail "review closeout on the default branch silently shows the dashboard"
  [[ "$(flat "$TRACKS/actualize.md")" == *'propose a new decision through `/archcore:document decision` that supersedes it'* ]] \
    || fail "drift rewrites an adr's decision in place"
}

@test "decision.classify routes an existing adr or open rfc instead of failing its entry condition" {
  local classify
  classify=$(gate_of "$TRACKS/decision.md" decision.classify)
  [[ "$classify" == *'a local `adr` the request would replace ("switch to", "replace", "we moved to") → record it as the replaced decision'* ]] \
    || fail "\"should we switch to Y\" over an existing adr has no route"
  [[ "$classify" == *'a local `adr` that already records the same choice → report it and exit without a write'* ]] \
    || fail "a duplicate decision has no defined outcome"
  [[ "$classify" == *'an open `rfc` draft → `decision.resolve` when a verdict is stated'* ]] \
    || fail "an open rfc on the topic blocks classification"
  [[ "$(gate_of "$TRACKS/decision.md" decision.adr)" == *'`supersedes` → the local `adr` that `decision.classify` recorded as replaced'* ]] \
    || fail "a replacing adr does not link the replaced one"
  [[ "$(gate_of "$TRACKS/decision.md" decision.cascade)" == *'the spec passes the user-owned choice check of `sdd.design`'* ]] \
    || fail "the architecture cascade creates a spec past an open user-owned choice"
}

@test "document confirms spec rewrites and redirects research with no material" {
  [[ "$(gate_of "$TRACKS/describe.md" describe.draft)" == *'apply a `spec-wrong` update only after the user confirms it, and report a `code-wrong` conflict without writing'* ]] \
    || fail "document code silently rewrites a covering spec"
  [[ "$(flat "$DOC_SKILL")" == *'IF the request supplies neither a report nor a material, THEN write nothing and name `/archcore:plan research <topic>`'* ]] \
    || fail "document research starts an investigation with nothing in hand"
}

@test "durable-context selection asks once, never writes unauthorized, and never accepts its own drafts" {
  local sel capture
  sel=$(flat "$SHARED/durable-context-selection.md")
  capture=$(gate_of "$TRACKS/closeout.md" closeout.capture)
  [[ "$sel" == *'Ask one question for the whole preview'* ]] || fail "the selection asks per document"
  [[ "$sel" == *'Count that question against the per-invocation ceiling.'* ]] || fail "the preview escapes the question ceiling"
  [[ "$sel" == *'do not propose an `adr`; report the rationale as an open gap'* ]] || fail "the selection can invent a rationale"
  [[ "$sel" == *'Never select a `journey`.'* ]] || fail "the selection can produce a journey"
  [[ "$capture" == *'blocking: every write was authorized at the preview before the call.'* ]] || fail "capture writes without authorization"
  [[ "$capture" == *'blocking: no document created at this gate was offered for acceptance in this invocation.'* ]] \
    || fail "capture accepts the drafts it just created"
  [[ "$(gate_of "$TRACKS/experience.md" experience.detect)" == *'selection of this invocation already listed or omitted it'* ]] \
    || fail "the experience offer re-asks a pattern the preview listed"
}
