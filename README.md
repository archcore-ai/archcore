# Archcore Claude Plugin

Git-native context for AI coding agents.

## Installation

Install Archcore CLI

```bash
curl -fsSL https://archcore.ai/install.sh | bash
```

### From marketplace

Add the marketplace and install the plugin:

```bash
claude plugin marketplace add archcore-ai/archcore-claude-plugin
claude plugin install archcore@archcore-plugins
```

Or from within Claude Code:

```bash
/plugin marketplace add archcore-ai/archcore-claude-plugin
/plugin install archcore@archcore-plugins
```

### Local (for development)

```bash
claude --plugin-dir ./archcore-claude-plugin
```

## What it does

The plugin integrates Archcore into Claude Code by adding:

- **MCP Server** — exposes Archcore tools (documents, relations) to the agent via `archcore mcp`
- **SessionStart Hook** — loads project context at the beginning of each session via `archcore hooks claude-code session-start`

## Uninstallation

```bash
/plugin uninstall archcore@archcore-plugins
```

## License

Apache-2.0
