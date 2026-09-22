// Package wiring implements host wiring: making a project's AI-agent host
// configs (hooks, MCP server entry, instruction nudge) match what `archcore
// init` would produce. It is shared by `archcore init --agent`, `archcore
// hooks install`, `archcore doctor --fix`, and the install_host_config MCP
// tool — cobra commands and MCP adapters stay in cmd/, the domain logic lives
// here.
//
// Results carry raw errors and absolute paths: rendering (terminal output,
// MCP sanitization per no-absolute-paths-in-mcp-errors.rule) is the caller's
// job at its own boundary.
package wiring

import (
	"errors"
	"fmt"
	"io/fs"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/config"
)

// AgentError is one failed wiring action for one agent. Action is the wiring
// surface that failed: "hooks", "mcp", or "instructions".
type AgentError struct {
	Action string
	Err    error
}

// AgentResult reports the artifacts ensured present for one agent (written
// now or already configured — the installers are idempotent either way).
// Paths are absolute; errors are raw.
type AgentResult struct {
	Agent          agents.AgentID
	MCPConfigPath  string // empty when the agent needs a manual MCP install
	MCPManualHint  string
	HooksSupported bool
	Instructions   string // empty when no instruction nudge was written
	// ExtraInstructions lists additional instruction files the agent's write
	// touched beyond Instructions (e.g. claude-code also upserts AGENTS.md).
	// Kept separate from Instructions so existing consumers of the single
	// primary path keep working unchanged.
	ExtraInstructions []string
	Errors            []AgentError
}

// Report is the result of one Apply call.
type Report struct {
	ArchcoreInitialized bool // true when this call created .archcore/
	Agents              []AgentResult
}

// EnsureProjectInitialized creates .archcore/ with default settings when
// absent, keeping existing settings untouched. Reports whether it created the
// settings now.
func EnsureProjectInitialized(baseDir string) (created bool, err error) {
	if _, err := config.Load(baseDir); err == nil {
		return false, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
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

// Apply wires the given host agent (plus, when allDetected, every other agent
// detected in baseDir): ensures .archcore/ exists, then installs hooks, MCP
// config, and the instruction nudge via the same installers as `archcore
// init --agent`. It validates the host id before any write. Individual
// artifact failures are reported per-agent, not fatal: partial wiring plus a
// readable report beats an aborted install.
func Apply(baseDir string, host agents.AgentID, allDetected bool) (Report, error) {
	agent := agents.ByID(host)
	if agent == nil {
		return Report{}, fmt.Errorf("unknown agent %q — valid agents: %v", host, agents.AllIDs())
	}

	list := []*agents.Agent{agent}
	if allDetected {
		for _, detected := range agents.Detect(baseDir) {
			if detected.ID != agent.ID {
				list = append(list, detected)
			}
		}
	}

	created, err := EnsureProjectInitialized(baseDir)
	if err != nil {
		return Report{}, err
	}

	report := Report{ArchcoreInitialized: created}
	for _, a := range list {
		r := AgentResult{Agent: a.ID}

		supported, err := InstallHooksForAgent(baseDir, a)
		r.HooksSupported = supported
		if err != nil {
			r.Errors = append(r.Errors, AgentError{Action: "hooks", Err: err})
		}

		if a.ManualMCPInstallHint != "" {
			r.MCPManualHint = a.ManualMCPInstallHint
		} else if err := a.WriteMCPConfig(baseDir); err != nil {
			r.Errors = append(r.Errors, AgentError{Action: "mcp", Err: err})
		} else if a.MCPConfigPath != nil {
			r.MCPConfigPath = a.MCPConfigPath(baseDir)
		}

		report.Agents = append(report.Agents, r)
	}

	// Instructions last, deduped by path (several agents share AGENTS.md). One
	// deduped write may touch several files (claude-code: CLAUDE.md + AGENTS.md)
	// and may fail partway, so the report is derived from what actually landed
	// on disk — not from the all-or-nothing error alone:
	//
	//   - Paths: each agent reports exactly the files that carry the managed
	//     block now (InstructionBlockPresent), so a partial write names the file
	//     that was written and omits the one that failed — never a silent write.
	//   - Errors: a write failure is attributed to EVERY agent sharing that file,
	//     not just the dedupe representative, so a shared-AGENTS.md failure is not
	//     lost for the non-representative agents (cursor + opencode).
	for _, a := range DedupeByInstructionsPath(baseDir, list) {
		path := a.InstructionsPath(baseDir)
		writeErr := a.WriteInstructions(baseDir)

		for i := range report.Agents {
			la := agents.ByID(report.Agents[i].Agent)
			if la == nil || la.InstructionsPath == nil || la.InstructionsPath(baseDir) != path {
				continue
			}
			// Report each of this agent's touch-set files that is actually on
			// disk. AllInstructionsPaths is [primary, extra...], so index 0 maps
			// to Instructions and the rest to ExtraInstructions.
			for j, p := range la.AllInstructionsPaths(baseDir) {
				if !agents.InstructionBlockPresent(p) {
					continue
				}
				if j == 0 {
					report.Agents[i].Instructions = p
				} else {
					report.Agents[i].ExtraInstructions = append(report.Agents[i].ExtraInstructions, p)
				}
			}
			if writeErr != nil {
				report.Agents[i].Errors = append(report.Agents[i].Errors,
					AgentError{Action: "instructions", Err: writeErr})
			}
		}
	}

	return report, nil
}
