#!/usr/bin/env bats
# Structure test: pin the large-or-partial-result guidance.
#
# A host can refuse a large tool result, or show only its first bytes. CLI 0.8.5
# answered that with `hits` and `index` ahead of `results`, a response byte
# budget (`truncated`), and shortened bodies (`body_truncated`). The plugin tells
# the model how to read them. Reverting that text kept every other test green, so
# this file pins it — on every assistant surface and in the shared convention.
#
# The body rule guards a write to a LOCAL document, so it must not sit behind a
# "the project mounts global sources" condition.

setup() {
  load '../helpers/common'
  common_setup
}

# One assistant, three host formats. The paragraph is a single line in each.
ASSISTANT_SURFACES="agents/archcore-assistant.md
agents/archcore-assistant.toml
copilot-agents/archcore-assistant.agent.md"

paragraph() {
  grep -F "**$2.**" "$PLUGIN_ROOT/$1" || true
}

@test "every assistant surface carries the Large or partial results paragraph" {
  local surface line count=0
  while IFS= read -r surface; do
    [ -n "$surface" ] || continue
    count=$((count + 1))
    line=$(paragraph "$surface" "Large or partial results")
    [ -n "$line" ] || fail "$surface: missing the Large or partial results paragraph"
    # shellcheck disable=SC2016 # literal backticks are the tokens under test
    for token in '`hits`' '`index`' '`results`' 'body_truncated: true' 'get_document' 'update_document' 'shortened body'; do
      case "$line" in
        *"$token"*) ;;
        *) fail "$surface: Large or partial results paragraph lacks $token" ;;
      esac
    done
  done <<< "$ASSISTANT_SURFACES"
  [ "$count" -eq 3 ] || fail "expected 3 assistant surfaces, iterated $count"
}

@test "the assistant paragraph is data-gated for a CLI that sends no hits or index" {
  # Same invariant as cli-compat-invariant.bats #3: a clause that reads an
  # optional MCP field states what to do when the field is absent.
  local surface line
  while IFS= read -r surface; do
    [ -n "$surface" ] || continue
    line=$(paragraph "$surface" "Large or partial results")
    case "$line" in
      *"older CLI"*) ;;
      *) fail "$surface: no fallback for a CLI that sends neither hits nor index" ;;
    esac
  done <<< "$ASSISTANT_SURFACES"
}

@test "the body rule is not gated on global sources" {
  # `body_truncated` protects update_document, and only a local document is
  # writable. Inside the Global sources paragraph the rule would apply only
  # "if the project mounts global sources".
  local surface line
  while IFS= read -r surface; do
    [ -n "$surface" ] || continue
    line=$(paragraph "$surface" "Global sources")
    [ -n "$line" ] || fail "$surface: missing the Global sources paragraph"
    case "$line" in
      *body_truncated*) fail "$surface: body_truncated rule sits inside the Global sources paragraph" ;;
    esac
    line=$(paragraph "$surface" "Large or partial results")
    case "$line" in
      *"every project"*) ;;
      *) fail "$surface: Large or partial results paragraph does not state that it applies to every project" ;;
    esac
  done <<< "$ASSISTANT_SURFACES"
}

@test "globals.md carries the Large or partial results section with its rules" {
  local file="$PLUGIN_ROOT/skills/_shared/globals.md" section
  grep -qx '## Large or partial results' "$file" \
    || fail "globals.md: missing the '## Large or partial results' heading"
  section=$(awk '/^## Large or partial results$/ {body=1; next} body && /^## / {exit} body {print}' "$file")
  [ -n "$section" ] || fail "globals.md: the Large or partial results section is empty"
  # shellcheck disable=SC2016 # literal backticks are the tokens under test
  for token in '`hits`' '`index`' '`truncated`' 'body_truncated: true' '`get_document`' '`update_document`' 'shortened body' 'local document'; do
    grep -qF -- "$token" <<< "$section" \
      || fail "globals.md: Large or partial results section lacks $token"
  done
}

@test "globals.md names hits as the complete count and gates hits and index on the CLI" {
  local file="$PLUGIN_ROOT/skills/_shared/globals.md" bullet
  # The bullet that introduces the two fields, up to the next bullet.
  bullet=$(awk '/^- \*\*`hits` and `index`/ {body=1; print; next} body && /^(- |## )/ {exit} body {print}' "$file")
  [ -n "$bullet" ] || fail "globals.md: no bullet introduces \`hits\` and \`index\`"
  grep -qF 'older CLI sends neither' <<< "$bullet" \
    || fail "globals.md: hits/index bullet has no old-CLI fallback"
  grep -qF 'byte budget' <<< "$bullet" \
    || fail "globals.md: hits/index bullet does not say the byte budget can cut index"
  grep -qiF 'only `hits` counts every match' <<< "$bullet" \
    || fail "globals.md: hits/index bullet does not name hits as the complete count"
}
