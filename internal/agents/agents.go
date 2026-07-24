package agents

import "os"

// AgentID uniquely identifies a supported agent.
type AgentID string

const (
	ClaudeCode AgentID = "claude-code"
	Cursor     AgentID = "cursor"
	GeminiCLI  AgentID = "gemini-cli"
	OpenCode   AgentID = "opencode"
	CodexCLI   AgentID = "codex-cli"
	RooCode    AgentID = "roo-code"
	Cline      AgentID = "cline"
	Copilot    AgentID = "copilot"
)

// Agent describes a coding agent's integration capabilities.
type Agent struct {
	ID                   AgentID
	DisplayName          string
	MCPConfigPath        func(baseDir string) string
	WriteMCPConfig       func(baseDir string) error
	DetectFn             func(baseDir string) bool
	ManualMCPInstallHint string // non-empty if MCP must be installed manually

	// InstructionsPath returns the file archcore writes its usage nudge into for
	// this agent. WriteInstructions installs the nudge; RemoveInstructions strips
	// it. Several agents share one path (six map to AGENTS.md), so callers dedupe
	// by InstructionsPath before writing.
	InstructionsPath   func(baseDir string) string
	WriteInstructions  func(baseDir string) error
	RemoveInstructions func(baseDir string) error

	// ExtraInstructionsPaths lists additional files WriteInstructions touches
	// beyond InstructionsPath (nil for single-target agents). Install-only:
	// RemoveInstructions deliberately leaves these to their owning agents' remove.
	ExtraInstructionsPaths func(baseDir string) []string
}

// AllInstructionsPaths returns every file WriteInstructions touches for this
// agent: the primary InstructionsPath followed by any ExtraInstructionsPaths.
// It is the single source of truth for "what a write touches" — both the CLI
// confirmation line and the wiring report derive from it, so a new multi-target
// agent cannot be under-reported on one path and not the other.
func (a *Agent) AllInstructionsPaths(baseDir string) []string {
	paths := []string{a.InstructionsPath(baseDir)}
	if a.ExtraInstructionsPaths != nil {
		paths = append(paths, a.ExtraInstructionsPaths(baseDir)...)
	}
	return paths
}

// all is the ordered registry of agents. Order matters for display/iteration.
var all = []*Agent{
	claudeCodeAgent(),
	cursorAgent(),
	geminiCLIAgent(),
	openCodeAgent(),
	codexCLIAgent(),
	rooCodeAgent(),
	clineAgent(),
	copilotAgent(),
}

// All returns all registered agents in stable order. The slice is a copy, but
// the *Agent values are shared registry entries — callers must not mutate them.
func All() []*Agent {
	cp := make([]*Agent, len(all))
	copy(cp, all)
	return cp
}

// ByID returns the agent with the given ID, or nil if not found.
func ByID(id AgentID) *Agent {
	for _, a := range all {
		if a.ID == id {
			return a
		}
	}
	return nil
}

// AllIDs returns all agent IDs in stable order.
func AllIDs() []AgentID {
	ids := make([]AgentID, len(all))
	for i, a := range all {
		ids[i] = a.ID
	}
	return ids
}

// Detect returns agents whose marker directories/files exist in baseDir.
func Detect(baseDir string) []*Agent {
	found := make([]*Agent, 0, len(all))
	for _, a := range all {
		if a.DetectFn(baseDir) {
			found = append(found, a)
		}
	}
	return found
}

// dirExists checks whether a directory exists at the given path.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// fileExists checks whether a regular file exists at the given path.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
