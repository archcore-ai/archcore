package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDecodeHookPayload: unparsable input is not an error state. A hook that
// cannot read its stdin must still allow — the alternative is blocking a user's
// edit because a host changed a field.
func TestDecodeHookPayload(t *testing.T) {
	t.Parallel()

	t.Run("valid JSON", func(t *testing.T) {
		t.Parallel()
		p := decodeHookPayload(strings.NewReader(`{"session_id":"abc","cwd":"/tmp","hook_event_name":"SessionStart"}`))
		if got := p.cwd(); got != "/tmp" {
			t.Errorf("cwd = %q, want %q", got, "/tmp")
		}
		if got := p.sessionID(); got != "abc" {
			t.Errorf("sessionID = %q, want %q", got, "abc")
		}
	})

	t.Run("invalid JSON yields an empty payload", func(t *testing.T) {
		t.Parallel()
		p := decodeHookPayload(strings.NewReader(`not json`))
		if got := p.cwd(); got != "" {
			t.Errorf("cwd = %q, want empty", got)
		}
	})

	t.Run("empty stdin yields an empty payload", func(t *testing.T) {
		t.Parallel()
		p := decodeHookPayload(strings.NewReader(``))
		if got := p.sessionID(); got != "" {
			t.Errorf("sessionID = %q, want empty", got)
		}
	})
}

func TestHandleSessionStart_WithDocuments(t *testing.T) {
	// Not parallel: the stdout capture below swaps a process global.
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

	ctx := sessionContextOf(t, "claude-code", base, "v0.1.0")

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
	// Not parallel: the stdout capture below swaps a process global.
	base := setupArchcoreDir(t)

	ctx := sessionContextOf(t, "claude-code", base, "v0.1.0")

	// An empty corpus reports itself in one line; the recap blocks stay absent
	// because there is nothing in progress and nothing recently decided.
	if !strings.Contains(ctx, "CORPUS: no documents yet") {
		t.Errorf("empty corpus should say so; ctx=%q", ctx)
	}
	if strings.Contains(ctx, "IN PROGRESS") || strings.Contains(ctx, "RECENTLY ACCEPTED") {
		t.Errorf("empty corpus should render no recap block; ctx=%q", ctx)
	}
}

// TestResolveBaseDir pins the hook baseDir decision: cwd from the hook payload
// wins; an empty payload falls back to the process working directory.
func TestResolveBaseDir(t *testing.T) {
	got, err := resolveBaseDir("/payload/dir")
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
	got, err = resolveBaseDir("")
	if err != nil {
		t.Fatal(err)
	}
	if got != wd {
		t.Errorf("resolveBaseDir(empty) = %q, want process wd %q", got, wd)
	}
}
