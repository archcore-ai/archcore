package agents

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func codexCLIAgent() *Agent {
	return &Agent{
		ID:          CodexCLI,
		DisplayName: "Codex CLI",
		MCPConfigPath: func(baseDir string) string {
			return filepath.Join(baseDir, ".codex", "config.toml")
		},
		WriteMCPConfig: func(baseDir string) error {
			return writeCodexCLIMCPConfig(baseDir)
		},
		DetectFn: func(baseDir string) bool {
			return dirExists(filepath.Join(baseDir, ".codex"))
		},
	}
}

// codexArchcoreBlockTemplate emits the archcore MCP server entry plus the
// per-server cwd and env block. baseDir is hardcoded into the cwd and the
// ARCHCORE_BASE_DIR env so the CLI binds to the project even if Codex
// invokes the server from a different working directory.
const codexArchcoreBlockTemplate = `
[mcp_servers.archcore]
command = "archcore"
args = ["mcp"]
cwd = %q

[mcp_servers.archcore.env]
ARCHCORE_BASE_DIR = %q
`

func writeCodexCLIMCPConfig(baseDir string) error {
	codexDir := filepath.Join(baseDir, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		return fmt.Errorf("creating .codex/ directory: %w", err)
	}

	configPath := filepath.Join(codexDir, "config.toml")

	var content string
	data, err := os.ReadFile(configPath)
	if err == nil {
		content = string(data)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("reading %s: %w", configPath, err)
	}

	// Skip only if the existing archcore section is byte-for-byte aligned
	// with baseDir — both `cwd = "<base>"` (in the main table) and
	// `ARCHCORE_BASE_DIR = "<base>"` (in the .env subtable) must match.
	// Anything stale (project moved, missing env, document-wide cwd hit
	// from another server) gets stripped and rewritten.
	if mainBody, envBody, ok := codexArchcoreSection(content); ok {
		want := fmt.Sprintf("cwd = %q", baseDir)
		wantEnv := fmt.Sprintf("ARCHCORE_BASE_DIR = %q", baseDir)
		if strings.Contains(mainBody, want) && strings.Contains(envBody, wantEnv) {
			return nil
		}
		content = removeCodexArchcoreBlock(content)
	}

	block := fmt.Sprintf(codexArchcoreBlockTemplate, baseDir, baseDir)
	content = strings.TrimRight(content, "\n") + block

	return os.WriteFile(configPath, []byte(content), 0o644)
}

// codexArchcoreSection extracts the body of [mcp_servers.archcore] (mainBody)
// and any [mcp_servers.archcore.env] subtable (envBody) from a TOML string.
// Returns ok=false when no archcore table is present. Boundaries respect
// dotted-path identity: [mcp_servers.archcore2] is a *different* table and
// terminates archcore's body.
func codexArchcoreSection(content string) (mainBody, envBody string, ok bool) {
	lines := strings.Split(content, "\n")
	var (
		main strings.Builder
		env  strings.Builder
	)
	inMain := false
	inEnv := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[mcp_servers.archcore]" {
			inMain = true
			inEnv = false
			ok = true
			continue
		}
		if trimmed == "[mcp_servers.archcore.env]" {
			inMain = false
			inEnv = true
			ok = true
			continue
		}
		// Any other table header ends the current archcore section.
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			inMain = false
			inEnv = false
			continue
		}
		if inMain {
			main.WriteString(line)
			main.WriteByte('\n')
		}
		if inEnv {
			env.WriteString(line)
			env.WriteByte('\n')
		}
	}
	return main.String(), env.String(), ok
}

// removeCodexArchcoreBlock deletes the [mcp_servers.archcore] section and its
// `archcore.*` subtables (e.g. archcore.env) from a TOML config string. Used
// when rewriting stale entries on re-run.
//
// Boundary contract: a sibling table like [mcp_servers.archcore2] is *not*
// removed — only the exact `archcore` table and dotted-path subtables match.
func removeCodexArchcoreBlock(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	skipping := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isCodexArchcoreHeader(trimmed) {
			skipping = true
			continue
		}
		if skipping {
			// A new top-level section ends the skip — unless it's another
			// archcore subtable (archcore.env etc), which we also strip.
			if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
				if isCodexArchcoreHeader(trimmed) {
					continue
				}
				skipping = false
				out = append(out, line)
				continue
			}
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// isCodexArchcoreHeader returns true for `[mcp_servers.archcore]` and any
// dotted subtable like `[mcp_servers.archcore.env]`. The trailing dot guards
// against sibling collisions: `[mcp_servers.archcore2]` is not a match.
func isCodexArchcoreHeader(trimmed string) bool {
	return trimmed == "[mcp_servers.archcore]" ||
		strings.HasPrefix(trimmed, "[mcp_servers.archcore.")
}
