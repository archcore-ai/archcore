package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// End-to-end through the real cobra leaves: payload on stdin, verdict on the
// host's channel. Not parallel: runHookCLI changes the working directory.

func TestHooksCLI_DeniesAWriteIntoAnotherProject(t *testing.T) {
	w := newHookWorkspace(t)
	target := filepath.Join(w.b, ".archcore", "knowledge", "x.adr.md")
	payload := writePayload(target, "")

	run := runHookCLI(t, w.a, "claude-code", "pre-tool-use", nil, payload)
	if run.exitCode != 2 || !strings.Contains(run.stderr, "update_document") || !strings.Contains(run.stderr, `project="`+w.b+`"`) {
		t.Errorf("claude-code: exit %d, stderr %q; want exit 2 naming update_document and project B", run.exitCode, run.stderr)
	}
	if run.stdout != "" {
		t.Errorf("claude-code: a deny wrote stdout %q", run.stdout)
	}

	run = runHookCLI(t, w.a, "copilot", "pre-tool-use", nil, payload)
	var doc map[string]any
	if err := json.Unmarshal([]byte(run.stdout), &doc); err != nil || doc["permissionDecision"] != "deny" || run.exitCode != 0 {
		t.Errorf("copilot: exit %d, stdout %q; want one JSON deny document and exit 0", run.exitCode, run.stdout)
	}
	if reason, _ := doc["permissionDecisionReason"].(string); !strings.Contains(reason, `project="`+w.b+`"`) {
		t.Errorf("copilot: reason %q does not name project B", reason)
	}

	run = runHookCLI(t, w.plain, "codex-cli", "pre-tool-use", nil, payload)
	if run.exitCode != 2 {
		t.Errorf("codex from a folder without .archcore/: exit %d, want the deny", run.exitCode)
	}
}

func TestHooksCLI_PayloadCwdStillWins(t *testing.T) {
	w := newHookWorkspace(t)
	cwd, _ := json.Marshal(w.b)
	payload := `{"tool_name":"Write","cwd":` + string(cwd) + `,"tool_input":{"file_path":".archcore/knowledge/x.adr.md"}}`
	run := runHookCLI(t, w.plain, "claude-code", "pre-tool-use", nil, payload)
	if run.exitCode != 2 || strings.Contains(run.stderr, "project=") {
		t.Errorf("exit %d, stderr %q; want the unchanged deny for the payload cwd's project", run.exitCode, run.stderr)
	}
}

func TestHooksCLI_SessionStartOutsideAProject(t *testing.T) {
	w := newHookWorkspace(t)
	run := runHookCLI(t, w.dir, "claude-code", "session-start", nil, `{"session_id":"`+t.Name()+`","source":"startup"}`)
	var out struct {
		HookSpecificOutput struct {
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(run.stdout), &out); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, run.stdout)
	}
	recap := out.HookSpecificOutput.AdditionalContext
	if run.exitCode != 0 || !strings.Contains(recap, "PROJECTS BELOW") || !strings.Contains(recap, "project=<its absolute root>") {
		t.Errorf("exit %d, recap %q; want the project inventory", run.exitCode, recap)
	}
	for _, banned := range []string{"init_project", "use create_document to record"} {
		if strings.Contains(recap, banned) {
			t.Errorf("the recap outside a project advises %q", banned)
		}
	}
}

func TestHooksCLI_PostToolUseValidatesTheNamedProject(t *testing.T) {
	w := newHookWorkspace(t)
	writeArchcoreDoc(t, w.b, "knowledge/bad.adr.md", "no frontmatter at all\n")
	project, _ := json.Marshal(w.b)
	payload := `{"tool_name":"mcp__plugin_archcore_archcore__update_document","tool_input":{"path":".archcore/knowledge/bad.adr.md","project":` + string(project) + `}}`
	run := runHookCLI(t, w.a, "claude-code", "post-tool-use", nil, payload)
	if run.exitCode != 0 || !strings.Contains(run.stdout, "[Archcore Validation]") {
		t.Errorf("exit %d, stdout %q; want B's validation finding", run.exitCode, run.stdout)
	}
}

// TestHooksCLI_PostToolUseFromASubdirectory: a shell that moved below the
// project root sends that directory as the payload cwd, and the write it
// reports still validates under the project root.
func TestHooksCLI_PostToolUseFromASubdirectory(t *testing.T) {
	w := newHookWorkspace(t)
	writeArchcoreDoc(t, w.b, "knowledge/bad.adr.md", "no frontmatter at all\n")
	cwd, _ := json.Marshal(filepath.Join(w.b, "src", "api"))
	root, _ := json.Marshal(w.b)
	result, _ := json.Marshal(`{"path":".archcore/knowledge/bad.adr.md","project":` + string(root) + `}`)

	for _, tt := range []struct{ name, response string }{
		{"no tool result", ""},
		{"claude-code tool result", `,"tool_response":[{"type":"text","text":` + string(result) + `}]`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			payload := `{"cwd":` + string(cwd) + `,"tool_name":"mcp__plugin_archcore_archcore__update_document",` +
				`"tool_input":{"path":".archcore/knowledge/bad.adr.md"}` + tt.response + `}`
			run := runHookCLI(t, w.plain, "claude-code", "post-tool-use", nil, payload)
			if run.exitCode != 0 || !strings.Contains(run.stdout, "bad.adr.md: missing YAML frontmatter") {
				t.Errorf("exit %d, stdout %q; want B's validation finding", run.exitCode, run.stdout)
			}
			if strings.Contains(run.stdout, "directory not found") {
				t.Errorf("reported a missing store after a write: %q", run.stdout)
			}
		})
	}
}
