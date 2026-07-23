# Copilot stdin fixtures

Two payload generations:

- `legacy-hybrid-write.json` — the Claude-compat shape (`hookEventName` +
  snake_case `tool_input`). Kept as the fallback-heuristic fixture: detection
  must still route it to `copilot`.
- `pretooluse-*.json` and `sessionstart.json` — redacted native captures from
  GitHub Copilot CLI 1.0.73 (`toolName` + `toolArgs` as an escaped JSON
  string). Native file tools use an absolute `path`; `create` uses
  `file_text`, while `edit` uses `old_str` and `new_str`.
- `posttooluse-mcp-update.json` — the native post-tool shape used to pin
  Archcore MCP tool normalization.

Session IDs, working directories, prompts, and file contents are redacted;
field names and nesting are preserved from the captured payloads.
