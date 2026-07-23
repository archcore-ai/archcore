package tools

import (
	"context"

	"archcore-cli/internal/agents"

	"github.com/mark3labs/mcp-go/mcp"
)

// HostWiringFunc executes the host-wiring installation for one agent id (plus,
// optionally, every agent auto-detected in the project) and returns a
// marshaled JSON report. The implementation is injected by the cmd layer —
// the installers live there next to the CLI commands that share them — which
// also keeps this package free of an import cycle (cmd → internal/mcp).
type HostWiringFunc func(host string, allDetected bool) ([]byte, error)

func agentIDStrings() []string {
	ids := agents.AllIDs()
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

func NewInstallHostConfigTool() mcp.Tool {
	return mcp.NewTool("install_host_config",
		mcp.WithDescription(`GATED — call ONLY when the user has explicitly asked to set up/wire Archcore into this coding agent (hooks, MCP config, usage hint) AND has confirmed a plan you stated first (which agent, which files). Do NOT call this proactively — not because you noticed missing hooks or MCP config, not for a generic "set up my project" request, not as part of routine document work. When in doubt, ask the user first; never call speculatively.

Writes host config outside .archcore/: MCP server entry, SessionStart hook (hosts that support hooks), and the Archcore usage hint in the agent's instructions file (e.g. .mcp.json, .claude/settings.json, AGENTS.md) — the same artifacts 'archcore init' writes. All writes land under the project root this MCP server was started for.

Idempotent: existing archcore entries are kept or updated in place; foreign config content is never touched — safe to call again to converge. Complements init_project, which only initializes .archcore/ itself and never touches host config.

Returns: JSON report per agent — artifact paths written (project-relative), hook support, and any per-artifact errors.`),
		mcp.WithString("host",
			mcp.Required(),
			mcp.Description(`Agent id of the current host to install wiring for.`),
			mcp.Enum(agentIDStrings()...),
		),
		mcp.WithBoolean("all_detected",
			mcp.Description(`Also install wiring for every agent whose marker directory already exists in the project (same auto-detection as 'archcore init'). Default false: only the given host.`),
		),
		mcp.WithTitleAnnotation("Install Host Config"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
}

func HandleInstallHostConfig(wire HostWiringFunc) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		host := request.GetString("host", "")
		if host == "" {
			return errorResult(`missing required parameter "host"`), nil
		}
		allDetected := request.GetBool("all_detected", false)

		report, err := wire(host, allDetected)
		if err != nil {
			return errorResult(sanitizeError("installing host config", err)), nil
		}
		return mcp.NewToolResultText(string(report)), nil
	}
}
