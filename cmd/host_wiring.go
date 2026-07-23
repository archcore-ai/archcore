package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/config"
	"archcore-cli/internal/mcp/tools"
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

// ensureProjectInitialized creates .archcore/ with default settings when
// absent, keeping existing settings untouched. Reports whether it created the
// settings now. Shared by `init --agent` and the install_host_config tool.
func ensureProjectInitialized(baseDir string) (created bool, err error) {
	if _, err := config.Load(baseDir); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("existing .archcore/ settings unreadable: %w", err)
	}
	if err := config.InitDir(baseDir); err != nil {
		return false, fmt.Errorf("creating .archcore/ directory: %w", err)
	}
	if err := config.Save(baseDir, config.NewNoneSettings()); err != nil {
		return false, fmt.Errorf("saving settings: %w", err)
	}
	return true, nil
}

// hostWiringExecutor returns the tools.HostWiringFunc backing the
// install_host_config MCP tool. It closes over the server's baseDir — the
// one root that is correct by construction — validates the host id before
// any write, and reuses the same installers as `archcore init --agent`.
// Individual artifact failures are reported per-agent, not fatal: partial
// wiring plus a readable report beats an aborted install.
func hostWiringExecutor(baseDir string) tools.HostWiringFunc {
	return func(host string, allDetected bool) ([]byte, error) {
		agent := agents.ByID(agents.AgentID(host))
		if agent == nil {
			return nil, fmt.Errorf("unknown agent %q — valid agents: %v", host, agents.AllIDs())
		}

		list := []*agents.Agent{agent}
		if allDetected {
			for _, detected := range agents.Detect(baseDir) {
				if detected.ID != agent.ID {
					list = append(list, detected)
				}
			}
		}

		created, err := ensureProjectInitialized(baseDir)
		if err != nil {
			return nil, err
		}

		report := wiringReport{ArchcoreInitialized: created}
		for _, a := range list {
			r := wiringAgentReport{Agent: string(a.ID)}

			supported, err := installHooksForAgent(baseDir, a)
			r.HooksSupported = supported
			if err != nil {
				r.Errors = append(r.Errors, tools.SanitizeError("hooks", err))
			}

			if a.ManualMCPInstallHint != "" {
				r.MCPManualHint = a.ManualMCPInstallHint
			} else if err := a.WriteMCPConfig(baseDir); err != nil {
				r.Errors = append(r.Errors, tools.SanitizeError("mcp", err))
			} else if a.MCPConfigPath != nil {
				r.MCPConfigPath = displayPath(baseDir, a.MCPConfigPath(baseDir))
			}

			report.Agents = append(report.Agents, r)
		}

		// Instructions last, deduped by path (several agents share AGENTS.md).
		for _, a := range dedupeByInstructionsPath(baseDir, list) {
			path := a.InstructionsPath(baseDir)
			if err := a.WriteInstructions(baseDir); err != nil {
				for i := range report.Agents {
					if report.Agents[i].Agent == string(a.ID) {
						report.Agents[i].Errors = append(report.Agents[i].Errors,
							tools.SanitizeError("instructions", err))
					}
				}
				continue
			}
			// Mark every listed agent that shares this instructions file.
			for i := range report.Agents {
				la := agents.ByID(agents.AgentID(report.Agents[i].Agent))
				if la != nil && la.InstructionsPath != nil && la.InstructionsPath(baseDir) == path {
					report.Agents[i].Instructions = displayPath(baseDir, path)
				}
			}
		}

		return json.Marshal(report)
	}
}
