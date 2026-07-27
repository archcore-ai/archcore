package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadHookInput(t *testing.T) {
	t.Parallel()

	t.Run("valid JSON", func(t *testing.T) {
		t.Parallel()
		r := strings.NewReader(`{"session_id":"abc","cwd":"/tmp","hook_event_name":"SessionStart"}`)
		input, err := readHookInput(r)
		if err != nil {
			t.Fatalf("readHookInput: %v", err)
		}
		if input.CWD != "/tmp" {
			t.Errorf("cwd = %q, want %q", input.CWD, "/tmp")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()
		r := strings.NewReader(`not json`)
		_, err := readHookInput(r)
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}

func TestHandleSessionStart_WithDocuments(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	// Create documents in different categories.
	knowledgeDoc := filepath.Join(base, ".archcore", "knowledge", "use-postgres.adr.md")
	if err := os.WriteFile(knowledgeDoc, []byte("---\ntitle: Use PostgreSQL\nstatus: accepted\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	visionDoc := filepath.Join(base, ".archcore", "vision", "mvp.plan.md")
	if err := os.WriteFile(visionDoc, []byte("---\ntitle: MVP Plan\nstatus: draft\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := handleSessionStart(base, "v0.1.0", shapeClaudeCompat)
	if err != nil {
		t.Fatalf("handleSessionStart: %v", err)
	}

	var out hookOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	ctx, ok := out.HookSpecificOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("missing additionalContext in output")
	}

	if !strings.Contains(ctx, "use-postgres.adr.md") {
		t.Error("context missing knowledge doc")
	}
	if !strings.Contains(ctx, "mvp.plan.md") {
		t.Error("context missing vision doc")
	}
	if !strings.Contains(ctx, "Refer to MCP server instructions") {
		t.Error("context missing MCP referral line")
	}
	if !strings.Contains(ctx, "create_document") {
		t.Error("context missing create_document MCP tool reference")
	}
	if !strings.Contains(ctx, "list_documents") {
		t.Error("context missing list_documents MCP tool reference")
	}
	if strings.Contains(ctx, "archcore create") {
		t.Error("context should not contain CLI command 'archcore create'")
	}
}

func TestHandleSessionStart_Empty(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	data, err := handleSessionStart(base, "v0.1.0", shapeClaudeCompat)
	if err != nil {
		t.Fatalf("handleSessionStart: %v", err)
	}

	var out hookOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	ctx, ok := out.HookSpecificOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("missing additionalContext in output")
	}

	// All categories should show (none).
	for _, cat := range []string{"knowledge", "vision", "experience"} {
		// Check that the category section exists and has (none).
		catIdx := strings.Index(ctx, "["+cat+"]")
		if catIdx == -1 {
			t.Errorf("missing category %s", cat)
			continue
		}
		after := ctx[catIdx:]
		if !strings.Contains(after, "(none)") {
			t.Errorf("category %s should show (none)", cat)
		}
	}
}

// TestResolveBaseDir pins the hook baseDir decision: cwd from the hook payload
// wins; an empty payload falls back to the process working directory.
func TestResolveBaseDir(t *testing.T) {
	got, err := resolveBaseDir(&hookInput{CWD: "/payload/dir"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/payload/dir" {
		t.Errorf("resolveBaseDir(cwd set) = %q, want the payload cwd", got)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got, err = resolveBaseDir(&hookInput{})
	if err != nil {
		t.Fatal(err)
	}
	if got != wd {
		t.Errorf("resolveBaseDir(empty) = %q, want process wd %q", got, wd)
	}
}
