package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexCLI_WriteMCPConfig_NewFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	agent := ByID(CodexCLI)

	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "[mcp_servers.archcore]") {
		t.Error("missing [mcp_servers.archcore] section")
	}
	if !strings.Contains(content, `command = "archcore"`) {
		t.Error("missing command line")
	}
	if !strings.Contains(content, `args = ["mcp"]`) {
		t.Error("missing args line")
	}
}

func TestCodexCLI_WriteMCPConfig_Idempotent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	agent := ByID(CodexCLI)

	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("first WriteMCPConfig: %v", err)
	}
	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("second WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	count := strings.Count(content, "[mcp_servers.archcore]")
	if count != 1 {
		t.Errorf("expected 1 archcore block, got %d", count)
	}
}

func TestCodexCLI_WriteMCPConfig_AppendsToTOML(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	codexDir := filepath.Join(base, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	existing := `[model]
name = "gpt-4"

[mcp_servers.other]
command = "other"
`
	if err := os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	agent := ByID(CodexCLI)
	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(codexDir, "config.toml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, `[model]`) {
		t.Error("existing [model] section lost")
	}
	if !strings.Contains(content, `[mcp_servers.other]`) {
		t.Error("existing [mcp_servers.other] section lost")
	}
	if !strings.Contains(content, `[mcp_servers.archcore]`) {
		t.Error("archcore section not added")
	}
}

func TestCodexCLI_WriteMCPConfig_EmptyFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	codexDir := filepath.Join(base, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	agent := ByID(CodexCLI)
	if err := agent.WriteMCPConfig(base); err != nil {
		t.Fatalf("WriteMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(codexDir, "config.toml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "[mcp_servers.archcore]") {
		t.Error("archcore section not added to empty file")
	}
}

func TestWriteCodexCLIMCPConfig_RewritesWhenStale(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		seed func(oldBase string) string
	}{
		{
			name: "cwd_mismatch",
			seed: func(oldBase string) string {
				return `[mcp_servers.archcore]
command = "archcore"
args = ["mcp"]
cwd = "` + oldBase + `"

[mcp_servers.archcore.env]
ARCHCORE_BASE_DIR = "` + oldBase + `"
`
			},
		},
		{
			name: "env_mismatch",
			seed: func(oldBase string) string {
				// cwd is up-to-date (CURRENT_BASE), env stale.
				return `[mcp_servers.archcore]
command = "archcore"
args = ["mcp"]
cwd = "CURRENT_BASE"

[mcp_servers.archcore.env]
ARCHCORE_BASE_DIR = "` + oldBase + `"
`
			},
		},
		{
			name: "env_missing",
			seed: func(oldBase string) string {
				return `[mcp_servers.archcore]
command = "archcore"
args = ["mcp"]
cwd = "CURRENT_BASE"
`
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			codexDir := filepath.Join(base, ".codex")
			if err := os.MkdirAll(codexDir, 0o755); err != nil {
				t.Fatalf("MkdirAll: %v", err)
			}

			oldBase := "/old/project"
			content := strings.ReplaceAll(c.seed(oldBase), "CURRENT_BASE", base)
			if err := os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(content), 0o644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			if err := writeCodexCLIMCPConfig(base); err != nil {
				t.Fatalf("writeCodexCLIMCPConfig: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(codexDir, "config.toml"))
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			got := string(data)

			// New cwd & env must point to base.
			if !strings.Contains(got, `cwd = "`+base+`"`) {
				t.Errorf("missing fresh cwd=%q\n--- got ---\n%s", base, got)
			}
			if !strings.Contains(got, `ARCHCORE_BASE_DIR = "`+base+`"`) {
				t.Errorf("missing fresh ARCHCORE_BASE_DIR=%q\n--- got ---\n%s", base, got)
			}
			// Old base must be gone.
			if strings.Contains(got, oldBase) {
				t.Errorf("stale path %q still present\n--- got ---\n%s", oldBase, got)
			}
			// Exactly one archcore block survives.
			if n := strings.Count(got, "[mcp_servers.archcore]"); n != 1 {
				t.Errorf("archcore block count = %d, want 1\n--- got ---\n%s", n, got)
			}
		})
	}
}

func TestWriteCodexCLIMCPConfig_PreservesNeighborTable(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	codexDir := filepath.Join(base, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	neighbor := `[mcp_servers.archcore2]
command = "other"
args = ["serve"]
cwd = "/some/where"
`
	stale := `[mcp_servers.archcore]
command = "archcore"
args = ["mcp"]
cwd = "/old/project"

[mcp_servers.archcore.env]
ARCHCORE_BASE_DIR = "/old/project"

`
	content := stale + neighbor
	if err := os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := writeCodexCLIMCPConfig(base); err != nil {
		t.Fatalf("writeCodexCLIMCPConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(codexDir, "config.toml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)

	// Neighbor must survive byte-for-byte.
	if !strings.Contains(got, "[mcp_servers.archcore2]") {
		t.Errorf("neighbor table [mcp_servers.archcore2] lost\n--- got ---\n%s", got)
	}
	if !strings.Contains(got, `command = "other"`) {
		t.Errorf("neighbor command lost\n--- got ---\n%s", got)
	}
	if !strings.Contains(got, `cwd = "/some/where"`) {
		t.Errorf("neighbor cwd lost\n--- got ---\n%s", got)
	}
	// Stale archcore data must be gone.
	if strings.Contains(got, "/old/project") {
		t.Errorf("stale path still present\n--- got ---\n%s", got)
	}
	// Fresh archcore block present.
	if !strings.Contains(got, `cwd = "`+base+`"`) {
		t.Errorf("fresh archcore cwd missing\n--- got ---\n%s", got)
	}
}

func TestRemoveCodexArchcoreBlock_BoundaryRespected(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		input     string
		mustHave  []string
		mustMiss  []string
	}{
		{
			name: "archcore_with_env_subtable_removed",
			input: `[mcp_servers.archcore]
command = "archcore"
args = ["mcp"]
cwd = "/old"

[mcp_servers.archcore.env]
ARCHCORE_BASE_DIR = "/old"
`,
			mustMiss: []string{"[mcp_servers.archcore]", "[mcp_servers.archcore.env]", "/old"},
		},
		{
			name: "lone_archcore2_unchanged",
			input: `[mcp_servers.archcore2]
command = "other"
cwd = "/some/where"
`,
			mustHave: []string{"[mcp_servers.archcore2]", `command = "other"`, `cwd = "/some/where"`},
		},
		{
			name: "archcore_then_archcore2_only_first_removed",
			input: `[mcp_servers.archcore]
command = "archcore"
cwd = "/old"

[mcp_servers.archcore.env]
ARCHCORE_BASE_DIR = "/old"

[mcp_servers.archcore2]
command = "other"
cwd = "/some/where"
`,
			mustHave: []string{"[mcp_servers.archcore2]", `cwd = "/some/where"`, `command = "other"`},
			mustMiss: []string{"[mcp_servers.archcore]", "[mcp_servers.archcore.env]", "/old"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := removeCodexArchcoreBlock(c.input)
			for _, must := range c.mustHave {
				if !strings.Contains(got, must) {
					t.Errorf("missing %q\n--- got ---\n%s", must, got)
				}
			}
			for _, must := range c.mustMiss {
				if strings.Contains(got, must) {
					t.Errorf("unexpected %q\n--- got ---\n%s", must, got)
				}
			}
		})
	}
}

func TestCodexCLI_Detect_True(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".codex"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if !ByID(CodexCLI).DetectFn(base) {
		t.Error("expected detection")
	}
}

func TestCodexCLI_Detect_False(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if ByID(CodexCLI).DetectFn(base) {
		t.Error("expected no detection")
	}
}
