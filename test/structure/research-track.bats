#!/usr/bin/env bats
# Prompt contracts, not a simulation of an agent executing the gates.

setup() {
  load '../helpers/common'
  common_setup
  TRACK="${RESEARCH_TRACK_UNDER_TEST:-$PLUGIN_ROOT/skills/_shared/tracks/research.md}"
  COMPAT="$PLUGIN_ROOT/skills/_shared/research-compatibility.md"
}

gate_text() {
  awk -v heading="### gate: research.$1" '
    $0 == heading { found = 1; next }
    found && /^#/ { exit }
    found { print }
  ' "$TRACK" | tr '\n' ' ' | sed 's/  */ /g'
}

@test "research frame and conclude distinguish coverage from a recommendation" {
  local frame conclude
  frame=$(gate_text frame)
  conclude=$(gate_text conclude)
  [[ "$frame" == *'an rnd draft contains Goal and numbered Questions'* ]] || { fail "missing phrase: an rnd draft contains Goal and numbered Questions"; return 1; }
  [[ "$frame" == *'a research draft contains Goal and Scope'* ]] || { fail "missing phrase: a research draft contains Goal and Scope"; return 1; }
  [[ "$conclude" == *'a research contains Goal, Scope, Coverage, Sources, Findings, Synthesis, and Open Gaps'* ]] || { fail "missing phrase: a research contains Goal, Scope, Coverage, Sources, Findings, Synthesis, and Open Gaps"; return 1; }
  [[ "$conclude" == *'an rnd contains Goal, Questions, Approach, Findings, Recommendation, and Next Action'* ]] || { fail "missing phrase: an rnd contains Goal, Questions, Approach, Findings, Recommendation, and Next Action"; return 1; }
  [[ "$conclude" == *'proceed, refine, defer, or stop'* ]] || { fail "missing phrase: proceed, refine, defer, or stop"; return 1; }
  [[ "$conclude" == *'do not invent a verdict for fallback completion'* ]] || { fail "missing phrase: do not invent a verdict for fallback completion"; return 1; }
}

@test "standalone evidence enters gather without a parent and exits before conclude" {
  local gather
  gather=$(gate_text gather)
  [[ "$gather" == *'an explicit evidence request supplies one material'* ]] || { fail "missing phrase: an explicit evidence request supplies one material"; return 1; }
  [[ "$gather" == *'- budget: 0'* ]] || { fail "missing phrase: - budget: 0"; return 1; }
  [[ "$gather" == *'standalone evidence without an identified consumer needs no edge'* ]] || { fail "missing phrase: standalone evidence without an identified consumer needs no edge"; return 1; }
  [[ "$gather" == *'- Next: exit for standalone evidence; otherwise `research.conclude`'* ]] || { fail "missing phrase: - Next: exit for standalone evidence; otherwise research.conclude"; return 1; }
  grep -Fq 'This entry needs no parent investigation' "$TRACK" || { fail "missing phrase: This entry needs no parent investigation"; return 1; }
  grep -Fq 'the supplied report satisfies frame inputs without an interview' "$PLUGIN_ROOT/skills/document/SKILL.md" || { fail "missing phrase: the supplied report satisfies frame inputs without an interview"; return 1; }
}

@test "gather requires provenance and the investigation edge before close" {
  local gather
  gather=$(gate_text gather)
  [[ "$gather" == *'Locator, Extract, and Notes'* ]] || { fail "missing phrase: Locator, Extract, and Notes"; return 1; }
  [[ "$gather" == *'Address, Access date, Publication date, Publisher'* ]] || { fail "missing phrase: Address, Access date, Publication date, Publisher"; return 1; }
  [[ "$gather" == *'unknown publication dates and publishers use visible placeholders'* ]] || { fail "missing phrase: unknown publication dates and publishers use visible placeholders"; return 1; }
  [[ "$gather" == *'first `supports` or `contradicts` edge before gate close'* ]] || { fail "missing phrase: first supports or contradicts edge before gate close"; return 1; }
  [[ "$gather" == *'no intended evidence write or relation remains pending'* ]] || { fail "missing phrase: no intended evidence write or relation remains pending"; return 1; }
}

@test "gather skips only after its exit checks pass and pending operations are empty" {
  local condition
  condition=$("$REPO_ROOT/test/helpers/extract-gates.sh" "$TRACK" | awk '
    /^gate:/ { gather = ($2 == "research.gather") }
    gather && /^skip_when:/ { sub(/^skip_when: /, ""); print }
  ')
  assert_equal "$condition" "the local artifact satisfies this gate's exit checks and no intended evidence write or relation remains pending"
}

@test "evidence checkpoint, creation, edge, and gate close stay ordered" {
  # The golden ignores operation prose. Pin this separate safety sequence.
  local ordered
  ordered=$(sed -n '/Evidence produced during an investigation/,/Create and first-edge/p' "$TRACK")
  [[ "$ordered" == *"parent draft's \`deferred\` state before creating evidence"* ]] || { fail "missing phrase: parent draft deferred state before creating evidence"; return 1; }
  [[ "$ordered" == *'create_document'*'list_relations'*'add_relation'*'gate-close `update_document`'* ]] || { fail "missing phrase: create_document"; return 1; }
  [[ "$ordered" == *'from evidence to the local investigation'* ]] || { fail "missing phrase: from evidence to the local investigation"; return 1; }
  [[ "$ordered" == *'`supersedes` from the newer evidence to the older local evidence'* ]] || { fail "missing phrase: supersedes from the newer evidence to the older local evidence"; return 1; }
  grep -Fq 'never create a second evidence document to repair a missing edge' "$TRACK" || { fail "missing phrase: never create a second evidence document to repair a missing edge"; return 1; }
}

@test "source promotion and explicit evidence exception remain bounded" {
  local prose
  prose=$(tr '\n' ' ' < "$TRACK")
  [[ "$prose" == *'two documents rely on it'* ]] || { fail "missing phrase: two documents rely on it"; return 1; }
  [[ "$prose" == *'contradiction involves it'* ]] || { fail "missing phrase: contradiction involves it"; return 1; }
  [[ "$prose" == *'newer material supersedes it'* ]] || { fail "missing phrase: newer material supersedes it"; return 1; }
  [[ "$prose" == *'explicit `evidence` request is an exception'* ]] || { fail "missing phrase: explicit evidence request is an exception"; return 1; }
  [[ "$prose" == *'Raw material stays outside `.archcore/`'* ]] || { fail "missing phrase: Raw material stays outside .archcore/"; return 1; }
}

@test "gather produces evidence only on promotion or explicit material entry" {
  local gather
  gather=$(gate_text gather)
  [[ "$gather" == *'type: evidence — only on promotion or explicit material entry; otherwise no new document'* ]] || { fail "missing phrase: type: evidence — only on promotion or explicit material entry"; return 1; }
  grep -Fq 'Promote a material to `evidence` only when two' "$TRACK" || { fail "missing phrase: Promote a material to evidence only when"; return 1; }
}

@test "frame reuses a complete artifact except on redo, refresh, or extend" {
  local frame
  frame=$(gate_text frame)
  [[ "$frame" == *'the request does not ask to redo, refresh, or extend it'* ]] || { fail "missing phrase: the request does not ask to redo, refresh, or extend it"; return 1; }
  grep -Fq 'asks to redo, refresh, or extend a complete matching artifact, resume that artifact at `research.gather`' "$TRACK" || { fail "missing phrase: redo resumes at research.gather"; return 1; }
}

@test "gather names the delegation split and the type-specific source checks" {
  local gather
  gather=$(gate_text gather)
  grep -Fq '`archcore-auditor` agent MAY collect codebase and `.archcore/` material at `research.gather`' "$TRACK" || { fail "missing phrase: archcore-auditor collects codebase and .archcore material at research.gather"; return 1; }
  grep -Fq 'The main thread gathers web material directly' "$TRACK" || { fail "missing phrase: The main thread gathers web material directly"; return 1; }
  [[ "$gather" == *"a research's Sources records its materials"* ]] || { fail "missing phrase: a research's Sources records its materials"; return 1; }
  [[ "$gather" == *"an rnd's Approach records the method used and names the sources consulted"* ]] || { fail "missing phrase: an rnd's Approach names the sources consulted"; return 1; }
  [[ "$gather" != *"an investigation's Sources"* ]] || { fail "gather checks a Sources section on an rnd, which has none"; return 1; }
}

@test "the computed path breaks the research versus rnd tie deterministically" {
  grep -Fq 'if the request names a pending decision or a set of candidates to choose between, select `rnd`' "$TRACK" || { fail "missing phrase: pending decision or candidate set selects rnd"; return 1; }
  grep -Fq 'Otherwise, select `research`' "$TRACK" || { fail "missing phrase: Otherwise, select research"; return 1; }
  grep -Fq 'A request naming a pending decision or a set of' "$PLUGIN_ROOT/skills/plan/SKILL.md" || { fail "missing phrase in plan/SKILL.md: pending decision tiebreak"; return 1; }
}

@test "sibling tracks recognize the research type beside rnd" {
  grep -Fq '`idea`, `prd`, `rnd`, or `research` covering the topic' "$PLUGIN_ROOT/skills/_shared/tracks/sdd.md" || { fail "sdd.frame skip_when omits research"; return 1; }
  grep -Fq '`rnd`, `research`, or `adr`' "$PLUGIN_ROOT/skills/_shared/tracks/sdd.md" || { fail "sdd compression path omits research"; return 1; }
  grep -Fq '`rnd` or `research` covering the topic' "$PLUGIN_ROOT/skills/_shared/tracks/decision.md" || { fail "decision.adr entry omits research"; return 1; }
  grep -Fq '`research`, `spec`)' "$PLUGIN_ROOT/skills/_shared/tracks/closeout.md" || { fail "closeout implements chain omits research"; return 1; }
  grep -Fq 'Coverage of a territory with no decision pending → `research`' "$PLUGIN_ROOT/skills/_shared/prd-contract.md" || { fail "prd-contract routing omits research"; return 1; }
}

@test "resume preserves old rnd types and makes artifact_type optional" {
  grep -Fq 'derive the artifact type from its filename type' "$TRACK" || { fail "missing phrase: derive the product from its filename type"; return 1; }
  grep -Fq 'new alias binding does not convert an `rnd`' "$TRACK" || { fail "missing phrase: new alias binding does not convert an rnd"; return 1; }
  grep -Fq 'artifact_type: research|rnd|evidence' "$PLUGIN_ROOT/skills/_shared/gate-contract.md" || { fail "missing phrase: artifact_type: research|rnd|evidence"; return 1; }
  grep -Fq 'field is optional on older artifacts' "$PLUGIN_ROOT/skills/_shared/gate-contract.md" || { fail "missing phrase: field is optional on older artifacts"; return 1; }
  grep -Fq 'contradicts the filename is a blocking state error' "$PLUGIN_ROOT/skills/_shared/gate-contract.md" || { fail "missing phrase: contradicts the filename is a blocking state error"; return 1; }
}

@test "expert map fixes research and rnd products and exposes evidence entry" {
  local conductor="$PLUGIN_ROOT/skills/_shared/delta-routing.md"
  grep -Fq '| `research` | research instrument, entry `research.frame`, type fixed to `research` |' "$conductor" || { fail "missing phrase: | research | research instrument, entry research.frame, type fixed to research |"; return 1; }
  grep -Fq '| `rnd` | research instrument, entry `research.frame`, type fixed to `rnd` |' "$conductor" || { fail "missing phrase: | rnd | research instrument, entry research.frame, type fixed to rnd |"; return 1; }
  grep -Fq '| `evidence` | research instrument, entry `research.gather`, standalone material |' "$conductor" || { fail "missing phrase: | evidence | research instrument, entry research.gather, standalone material |"; return 1; }
}

@test "standalone research products bypass the plan implement fork" {
  local step
  step=$(sed -n '/### 5. Map tasks/,/### 6. Implement fork/p' "$PLUGIN_ROOT/skills/plan/SKILL.md" | tr '\n' ' ')
  [[ "$step" == *'WHEN the package produced no `plan` document'* ]] || { fail "missing phrase: WHEN the package produced no plan document"; return 1; }
  [[ "$step" == *'standalone research, rnd, and evidence paths), skip to Result'* ]] || { fail "missing phrase: standalone research, rnd, and evidence paths), skip to Result"; return 1; }
}

@test "both skill entry points gate new search filters before grounding" {
  local skill
  for skill in plan document; do
    grep -Fq 'skills/_shared/research-compatibility.md' "$PLUGIN_ROOT/skills/$skill/SKILL.md" || { fail "missing phrase: skills/_shared/research-compatibility.md"; return 1; }
  done
  grep -Fq 'before the first document search' "$COMPAT" || { fail "missing phrase: before the first document search"; return 1; }
  grep -Fq 'return `needs-vocabulary-probe` to the caller' "$COMPAT" || { fail "missing phrase: needs-vocabulary-probe branch in compatibility contract"; return 1; }
  grep -Fq 'needs-vocabulary-probe' "$PLUGIN_ROOT/skills/plan/SKILL.md" || { fail "plan/SKILL.md has no no-shell branch"; return 1; }
  grep -Fq 'needs-vocabulary-probe' "$PLUGIN_ROOT/skills/document/SKILL.md" || { fail "document/SKILL.md has no no-shell branch"; return 1; }
  grep -Fq 'explicit type is `evidence`' "$COMPAT" || { fail "missing phrase: explicit type is evidence"; return 1; }
  grep -Fq 'exit without a document write' "$COMPAT" || { fail "missing phrase: exit without a document write"; return 1; }
  grep -Fq 'exit without rewriting or converting' "$COMPAT" || { fail "missing phrase: exit without rewriting or converting"; return 1; }
  grep -Fq 'server rejects a new enum after a successful probe' "$COMPAT" || { fail "missing phrase: server rejects a new enum after a successful probe"; return 1; }
}

@test "the documented version probe calls only --version and gates the boundary" {
  local probe ver expected
  probe=$(awk '/^```sh$/ { f=1; next } f && /^```$/ { exit } f { print }' "$COMPAT")
  export CLAUDE_SKILL_DIR="$PLUGIN_ROOT/skills/plan"
  export MOCK_ARCHCORE_LOG="$BATS_TEST_TMPDIR/args"
  for ver in 0.8.2 0.8.3 0.8.4 0.9.0 0.10.0 unknown; do
    : > "$MOCK_ARCHCORE_LOG"
    mock_archcore_logging "$ver"
    run sh -c "$probe"
    assert_success
    case "$ver" in
      0.8.2) expected=no ;;
      unknown) expected=__NO_CLI__ ;;
      *) expected=yes ;;
    esac
    assert_output "$expected"
    [ "$(cat "$MOCK_ARCHCORE_LOG")" = '--version' ] || { fail "probe called something other than --version"; return 1; }
  done
}

@test "category membership and prose coverage include both types" {
  grep -Fq '| vision | prd, idea, plan, rnd, research,' "$PLUGIN_ROOT/skills/_shared/coverage-taxonomy.md" || { fail "missing phrase: | vision | prd, idea, plan, rnd, research,"; return 1; }
  grep -Fq '| knowledge | adr, rfc, rule, guide, doc, spec, evidence |' "$PLUGIN_ROOT/skills/_shared/coverage-taxonomy.md" || { fail "missing phrase: | knowledge | adr, rfc, rule, guide, doc, spec, evidence |"; return 1; }
  grep -Fq '`research`, `evidence`, `cpat`' "$PLUGIN_ROOT/skills/_shared/precision-rules.md" || { fail "missing phrase: research, evidence, cpat"; return 1; }
}
