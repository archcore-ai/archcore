package wiring

import (
	"path/filepath"
	"strings"

	"archcore-cli/internal/agents"
)

// DedupeByInstructionsPath returns one agent per unique instruction-file path,
// preserving registry order. Agents without an instruction target are skipped.
func DedupeByInstructionsPath(baseDir string, list []*agents.Agent) []*agents.Agent {
	seen := make(map[string]bool, len(list))
	out := make([]*agents.Agent, 0, len(list))
	for _, agent := range list {
		// Skip agents missing any instruction hook. The registry wires all three
		// together (enforced by TestAllAgents_RequiredFields), so this guard
		// makes the dedup the single safe gate before callers deref the others.
		if agent.InstructionsPath == nil || agent.WriteInstructions == nil || agent.RemoveInstructions == nil {
			continue
		}
		path := agent.InstructionsPath(baseDir)
		if seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, agent)
	}
	return out
}

// DisplayPath renders a config-file path relative to baseDir with forward
// slashes, for stable user-facing and test output. If path is not under baseDir
// (Rel fails or escapes upward), it falls back to the cleaned path unchanged.
func DisplayPath(baseDir, path string) string {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
