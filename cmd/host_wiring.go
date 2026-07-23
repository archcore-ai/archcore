package cmd

import (
	"encoding/json"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/mcp/tools"
	"archcore-cli/internal/wiring"
)

// wiringAgentReport is the per-agent slice of the install_host_config tool's
// JSON result. Paths are the artifacts ensured present (written now or
// already configured — the installers are idempotent either way), rendered
// relative to the project root; error strings are sanitized — the report goes
// to an MCP client (no-absolute-paths-in-mcp-errors.rule).
type wiringAgentReport struct {
	Agent          string   `json:"agent"`
	MCPConfigPath  string   `json:"mcp_config_path,omitempty"`
	MCPManualHint  string   `json:"mcp_manual_hint,omitempty"`
	HooksSupported bool     `json:"hooks_supported"`
	Instructions   string   `json:"instructions_path,omitempty"`
	Errors         []string `json:"errors,omitempty"`
}

type wiringReport struct {
	ArchcoreInitialized bool                `json:"archcore_initialized"` // true when this call created .archcore/
	Agents              []wiringAgentReport `json:"agents"`
}

// hostWiringExecutor returns the tools.HostWiringFunc backing the
// install_host_config MCP tool. It closes over the server's baseDir — the
// one root that is correct by construction — and adapts wiring.Apply for the
// MCP boundary: absolute paths become project-relative, raw per-agent errors
// are sanitized (no-absolute-paths-in-mcp-errors.rule).
func hostWiringExecutor(baseDir string) tools.HostWiringFunc {
	return func(host string, allDetected bool) ([]byte, error) {
		result, err := wiring.Apply(baseDir, agents.AgentID(host), allDetected)
		if err != nil {
			return nil, err
		}

		report := wiringReport{ArchcoreInitialized: result.ArchcoreInitialized}
		for _, a := range result.Agents {
			r := wiringAgentReport{
				Agent:          string(a.Agent),
				MCPManualHint:  a.MCPManualHint,
				HooksSupported: a.HooksSupported,
			}
			if a.MCPConfigPath != "" {
				r.MCPConfigPath = wiring.DisplayPath(baseDir, a.MCPConfigPath)
			}
			if a.Instructions != "" {
				r.Instructions = wiring.DisplayPath(baseDir, a.Instructions)
			}
			for _, e := range a.Errors {
				r.Errors = append(r.Errors, tools.SanitizeError(e.Action, e.Err))
			}
			report.Agents = append(report.Agents, r)
		}

		return json.Marshal(report)
	}
}
