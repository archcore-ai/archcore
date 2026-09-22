# Cursor stdin fixtures

- `preToolUse-write.json`, `write-archcore.json` — `preToolUse` payloads with
  an object `tool_input`.
- `mcp-create.json`, `mcp-update.json` — `afterMCPExecution` payloads in the
  shape the normalizer was first written against: bare `tool_name`, escaped
  JSON string `tool_input`, no server identity. The launcher passes these
  through untouched because nothing in them proves Archcore owns the tool.
- `afterMCPExecution-update-archcore.json`,
  `afterMCPExecution-update-foreign.json` — `afterMCPExecution` payloads
  composed from the Cursor hooks reference (read 2026-09-07): `mcp_server_name`
  carries the server's key in `mcp.json`, `result_json` is a JSON string, and
  `duration` is in milliseconds. `archcore` is the key `archcore init --agent
  cursor` writes. These two are composed, not captured from a live Cursor
  session.
