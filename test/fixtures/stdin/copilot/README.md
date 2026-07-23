# Copilot stdin fixtures

Two payload generations:

- `legacy-hybrid-write.json` — the Claude-compat shape (`hookEventName` +
  snake_case `tool_input`). Kept as the fallback-heuristic fixture: detection
  must still route it to `copilot`.
- `pretooluse-*.json`, `sessionstart.json`, `posttooluse-mcp-update.json` —
  the native camelCase shape (`toolName` + `toolArgs` as an escaped JSON
  string), per docs.github.com hooks-reference.

**PROVISIONAL:** the exact keys inside `toolArgs` per native tool
(`create`/`edit`/`str_replace_editor`/`apply_patch`) and the MCP `toolName`
format are not documented. These fixtures encode the current best guess and
MUST be replaced with real redacted captures during the Copilot smoke test
(see `copilot-host-support.rnd` Next Action) before the adapter ships.
