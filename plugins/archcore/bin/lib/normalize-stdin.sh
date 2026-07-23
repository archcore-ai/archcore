#!/bin/sh
# shellcheck disable=SC2034  # Variables are exported for use by sourcing scripts
# Multi-host stdin normalization for Archcore plugin hook scripts.
# Source this file at the top of each bin/ script that receives hook stdin.
#
# Exports:
#   ARCHCORE_HOST       — "claude-code" | "cursor" | "copilot" | "codex" | "opencode"
#                         (detected from stdin, or forced via the env var)
#   ARCHCORE_RAW_STDIN  — unmodified stdin JSON
#   ARCHCORE_TOOL_NAME  — normalized tool name (mcp__archcore__* prefix for MCP tools)
#   ARCHCORE_FILE_PATH  — target file path from tool input (empty if N/A)
#   ARCHCORE_DOC_PATH   — document path from MCP tool input (empty if N/A)
#
# Functions:
#   archcore_hook_block "reason"  — block operation, exit (PreToolUse)
#   archcore_hook_info  "message" — emit info to agent (PostToolUse)
#   archcore_hook_allow           — allow operation silently, exit 0
#
# OpenCode bridge contract (host-adapter-contract.spec): the TS bridge spawns
# these scripts with ARCHCORE_HOST=opencode in env and a Claude-shaped
# snake_case JSON payload on stdin. Block = exit 2 + reason on stderr (the
# bridge throws Error(stderr)); info output is plain text on stdout, which the
# bridge appends verbatim as context. No JSON parsing on the bridge side.

# --- Read stdin once ---
ARCHCORE_RAW_STDIN=$(cat)

# --- Host detection ---
# Priority: env var override > stdin heuristic > default
#
# Cursor sends "conversation_id" in all hook events.
# GitHub Copilot (native camelCase format) sends "toolName"/"toolArgs"; its
#   legacy Claude-compat payloads carry "hookEventName". The copilot hooks
#   config pins ARCHCORE_HOST=copilot via per-entry env, so this heuristic is
#   a fallback only. camelCase keys never appear in the snake_case schemas of
#   Claude Code / Codex, nor in Cursor payloads (checked first).
# Codex CLI sends "turn_id" in turn-scoped events (PreToolUse/PostToolUse/etc.); SessionStart
#   has no turn_id but Codex shares Claude Code's snake_case schema, so the claude-code
#   fallback handles SessionStart correctly.
# Claude Code sends none of the above — fallback default.
# OpenCode has no stdin heuristic by design: its bridge always sets
#   ARCHCORE_HOST=opencode and sends Claude-shaped snake_case payloads.
if [ -z "$ARCHCORE_HOST" ]; then
  if printf '%s' "$ARCHCORE_RAW_STDIN" | grep -q '"conversation_id"'; then
    ARCHCORE_HOST="cursor"
  elif printf '%s' "$ARCHCORE_RAW_STDIN" | grep -qE '"hookEventName"|"toolName"|"toolArgs"'; then
    ARCHCORE_HOST="copilot"
  elif printf '%s' "$ARCHCORE_RAW_STDIN" | grep -q '"turn_id"'; then
    ARCHCORE_HOST="codex"
  else
    ARCHCORE_HOST="claude-code"
  fi
fi

# --- Extract fields by host ---
# Helper: extract a JSON string value by key (first occurrence, from raw stdin)
_archcore_json_val() {
  printf '%s' "$ARCHCORE_RAW_STDIN" | \
    grep -o "\"$1\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" | head -1 | \
    sed "s/.*\"$1\"[[:space:]]*:[[:space:]]*\"//;s/\"$//"
}

# Helper: extract a JSON string value from escaped JSON strings (e.g. tool_input in afterMCPExecution)
_archcore_json_val_unescaped() {
  printf '%s' "$ARCHCORE_RAW_STDIN" | sed 's/\\"/"/g' | \
    grep -o "\"$1\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" | head -1 | \
    sed "s/.*\"$1\"[[:space:]]*:[[:space:]]*\"//;s/\"$//"
}

case "$ARCHCORE_HOST" in
  claude-code)
    # Claude Code: tool_name has full prefix (mcp__archcore__create_document)
    ARCHCORE_TOOL_NAME=$(_archcore_json_val "tool_name")
    ARCHCORE_FILE_PATH=$(_archcore_json_val "file_path")
    ARCHCORE_DOC_PATH=$(_archcore_json_val "path")
    ;;
  cursor)
    _event=$(_archcore_json_val "hook_event_name")
    _raw_tool=$(_archcore_json_val "tool_name")
    case "$_event" in
      afterMCPExecution|beforeMCPExecution)
        # MCP events have bare tool names (create_document, update_document).
        # Normalize to mcp__archcore__ prefix so bin scripts work unchanged.
        ARCHCORE_TOOL_NAME="mcp__archcore__${_raw_tool}"
        ;;
      *)
        ARCHCORE_TOOL_NAME="$_raw_tool"
        ;;
    esac
    ARCHCORE_FILE_PATH=$(_archcore_json_val "file_path")
    # In afterMCPExecution, tool_input is an escaped JSON string.
    # Try direct extraction first, then unescaped fallback.
    ARCHCORE_DOC_PATH=$(_archcore_json_val "path")
    if [ -z "$ARCHCORE_DOC_PATH" ]; then
      ARCHCORE_DOC_PATH=$(_archcore_json_val_unescaped "path")
    fi
    ;;
  copilot)
    # Native camelCase payload: toolName + toolArgs (an ESCAPED JSON string,
    # so field extraction goes through the unescaped helper, which also
    # handles plain objects). Legacy Claude-compat payloads (hookEventName +
    # snake_case tool_input) extract via the tool_name fallback. Native file
    # tools use an absolute path key (verified against Copilot CLI 1.0.73).
    ARCHCORE_TOOL_NAME=$(_archcore_json_val "toolName")
    if [ -z "$ARCHCORE_TOOL_NAME" ]; then
      ARCHCORE_TOOL_NAME=$(_archcore_json_val "tool_name")
    fi
    ARCHCORE_FILE_PATH=$(_archcore_json_val_unescaped "file_path")
    if [ -z "$ARCHCORE_FILE_PATH" ]; then
      ARCHCORE_FILE_PATH=$(_archcore_json_val_unescaped "filePath")
    fi
    if [ -z "$ARCHCORE_FILE_PATH" ]; then
      case "$ARCHCORE_TOOL_NAME" in
        create|edit|str_replace_editor|apply_patch)
          ARCHCORE_FILE_PATH=$(_archcore_json_val_unescaped "path")
          ;;
      esac
    fi
    ARCHCORE_DOC_PATH=$(_archcore_json_val "path")
    if [ -z "$ARCHCORE_DOC_PATH" ]; then
      ARCHCORE_DOC_PATH=$(_archcore_json_val_unescaped "path")
    fi
    ;;
  codex)
    # Codex CLI shares Claude Code's snake_case stdin schema:
    # tool_name carries the full mcp__archcore__* prefix for MCP events,
    # tool_input.file_path for Write/Edit/apply_patch, tool_input.path for MCP doc ops.
    ARCHCORE_TOOL_NAME=$(_archcore_json_val "tool_name")
    ARCHCORE_FILE_PATH=$(_archcore_json_val "file_path")
    ARCHCORE_DOC_PATH=$(_archcore_json_val "path")
    ;;
  opencode)
    # OpenCode: env-only host (the bridge always sets ARCHCORE_HOST=opencode;
    # there is no stdin heuristic). Payloads are Claude-shaped snake_case per
    # the bridge contract, so extraction mirrors claude-code. This explicit
    # case is load-bearing: the * fallback below rewrites ARCHCORE_HOST to
    # claude-code, which would clobber the bridge's env and misroute output.
    ARCHCORE_TOOL_NAME=$(_archcore_json_val "tool_name")
    ARCHCORE_FILE_PATH=$(_archcore_json_val "file_path")
    ARCHCORE_DOC_PATH=$(_archcore_json_val "path")
    ;;
  *)
    # Unknown host — best-effort extraction, treat as Claude Code
    ARCHCORE_HOST="claude-code"
    ARCHCORE_TOOL_NAME=$(_archcore_json_val "tool_name")
    ARCHCORE_FILE_PATH=$(_archcore_json_val "file_path")
    ARCHCORE_DOC_PATH=$(_archcore_json_val "path")
    ;;
esac

# --- Output helpers ---

# Block the current operation (for gatekeeping hooks like preToolUse).
# Exit code 2 blocks in Claude Code, Cursor, and Codex; the OpenCode bridge
# translates exit 2 + stderr into a thrown error (see host-adapter-contract).
# GitHub Copilot is different: exit 2 is only a WARNING there — a deny
# requires stdout JSON permissionDecision with exit 0 (hooks-reference).
archcore_hook_block() {
  _reason="$1"
  case "$ARCHCORE_HOST" in
    copilot)
      _escaped=$(printf '%s' "$_reason" | sed 's/\\/\\\\/g; s/"/\\"/g' | awk 'BEGIN{ORS="\\n"}{print}')
      printf '{"permissionDecision":"deny","permissionDecisionReason":"%s"}' "$_escaped"
      exit 0
      ;;
    *)
      echo "$_reason" >&2
      exit 2
      ;;
  esac
}

# Emit informational message to the agent (for post-execution hooks).
# Does NOT exit — caller continues.
archcore_hook_info() {
  _msg="$1"
  _escaped=$(printf '%s' "$_msg" | sed 's/"/\\"/g' | tr '\n' ' ')
  case "$ARCHCORE_HOST" in
    claude-code|codex)
      printf '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"%s"}}' "$_escaped"
      ;;
    copilot)
      # Native postToolUse output: top-level additionalContext (10 KB cap).
      printf '{"additionalContext":"%s"}' "$_escaped"
      ;;
    cursor)
      printf '{"additional_context":"%s"}' "$_escaped"
      ;;
    opencode)
      # Bridge appends raw stdout as context — plain text, no JSON, no escaping.
      printf '%s\n' "$_msg"
      ;;
  esac
}

# Emit context injection for PreToolUse hooks (additive, non-blocking).
# Preserves multi-line output via literal "\n" in JSON (not concatenation).
# Does NOT exit — caller continues. Callers exit 0 afterward.
archcore_hook_pretool_info() {
  _msg="$1"
  _escaped=$(printf '%s' "$_msg" | sed 's/\\/\\\\/g; s/"/\\"/g' | awk 'BEGIN{ORS="\\n"}{print}')
  case "$ARCHCORE_HOST" in
    claude-code|codex)
      printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"%s"}}' "$_escaped"
      ;;
    copilot)
      # preToolUse additionalContext is undocumented on Copilot; shape kept
      # identical to the postToolUse arm so the entry can move events in the
      # hooks config without a core change (contingency A in the plan).
      printf '{"additionalContext":"%s"}' "$_escaped"
      ;;
    cursor)
      printf '{"additional_context":"%s"}' "$_escaped"
      ;;
    opencode)
      printf '%s\n' "$_msg"
      ;;
  esac
}

# Allow the operation silently and exit.
archcore_hook_allow() {
  exit 0
}
