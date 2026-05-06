package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"archcore-cli/internal/projectroot"
)

func TestWhichProject_OK(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := &projectroot.Resolution{Path: dir, Source: projectroot.SourceFlag}
	handler := HandleWhichProject(res, "0.5.0")
	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := firstTextOf(result)
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}
	if resp["base_dir"] != dir {
		t.Errorf("base_dir = %v, want %s", resp["base_dir"], dir)
	}
	if resp["source"] != "flag" {
		t.Errorf("source = %v, want flag", resp["source"])
	}
	if resp["cli_version"] != "0.5.0" {
		t.Errorf("cli_version = %v, want 0.5.0", resp["cli_version"])
	}
	markers, ok := resp["markers"].(map[string]any)
	if !ok {
		t.Fatalf("markers missing or wrong type: %v", resp["markers"])
	}
	if markers[".git"] != "found" {
		t.Errorf("markers[.git] = %v, want found", markers[".git"])
	}
	guards, _ := resp["guards"].(map[string]any)
	if guards["strict"] != true {
		t.Errorf("guards.strict = %v, want true", guards["strict"])
	}
	if guards["legacy"] != false {
		t.Errorf("guards.legacy = %v, want false", guards["legacy"])
	}
}

func TestWhichProject_LegacyMode(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	res := &projectroot.Resolution{Path: dir, Source: projectroot.SourceFlag, LegacyMode: true}
	handler := HandleWhichProject(res, "0.5.0")
	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	var resp map[string]any
	if err := json.Unmarshal([]byte(firstTextOf(result)), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	guards, _ := resp["guards"].(map[string]any)
	if guards["legacy"] != true {
		t.Errorf("guards.legacy = %v, want true", guards["legacy"])
	}
	if guards["strict"] != false {
		t.Errorf("guards.strict = %v, want false in legacy mode", guards["strict"])
	}
}

func TestWhichProject_NilResolution(t *testing.T) {
	t.Parallel()
	handler := HandleWhichProject(nil, "0.5.0")
	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	var resp map[string]any
	if err := json.Unmarshal([]byte(firstTextOf(result)), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["ok"] != false {
		t.Errorf("ok = %v, want false", resp["ok"])
	}
	probs, _ := resp["problems"].([]any)
	if len(probs) == 0 {
		t.Fatal("expected at least one problem")
	}
	first, _ := probs[0].(map[string]any)
	if first["code"] != projectroot.CodeNoProject {
		t.Errorf("problems[0].code = %v, want %s", first["code"], projectroot.CodeNoProject)
	}
}

func TestWhichProject_DefaultVersion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	res := &projectroot.Resolution{Path: dir, Source: projectroot.SourceCwd}
	handler := HandleWhichProject(res, "")
	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	var resp map[string]any
	if err := json.Unmarshal([]byte(firstTextOf(result)), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["cli_version"] != "dev" {
		t.Errorf("cli_version = %v, want dev fallback", resp["cli_version"])
	}
}

// firstTextOf extracts the first TextContent from a tool result. Mirrors the
// integration harness helper but lives here so unit tests don't need to import
// the integration package.
func firstTextOf(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	for _, c := range result.Content {
		if tc, ok := mcp.AsTextContent(c); ok {
			return tc.Text
		}
	}
	return ""
}
