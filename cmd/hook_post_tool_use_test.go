package cmd

import (
	"fmt"
	"strings"
	"testing"
)

// Case names mirror check-cascade.bats, validate-archcore.bats and
// check-precision.bats in the plugin.

func mcpPayload(tool, docPath string) *hookPayload {
	return decodeHookPayload(strings.NewReader(
		fmt.Sprintf(`{"tool_name":%q,"tool_input":{"path":%q}}`, tool, docPath)))
}

// TestPostToolUse_NeverDenies: the tool has already run. A deny here would be a
// confusing non-zero exit that blocks nothing.
func TestPostToolUse_NeverDenies(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "knowledge/a.adr.md", "---\ntitle: \"A\"\nstatus: draft\n---\n\nShort.\n")

	dec := postToolUseHandler(bg(), postReq(t, base, "mcp__archcore__update_document", ".archcore/knowledge/a.adr.md"))

	if dec.deny {
		t.Error("post-write check produced a deny")
	}
}

// TestPostToolUse_IgnoresForeignTools: an unrelated tool call is not ours to
// comment on.
func TestPostToolUse_IgnoresForeignTools(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	for _, tool := range []string{"Write", "mcp__other__create_document", "Bash"} {
		t.Run(tool, func(t *testing.T) {
			dec := postToolUseHandler(bg(), postReq(t, base, tool, ".archcore/knowledge/a.adr.md"))
			if dec.context != "" {
				t.Errorf("produced context %q, want none", dec.context)
			}
		})
	}
}

// TestPostToolUse_ReachesThroughEveryMCPNaming: one server, four spellings.
func TestPostToolUse_ReachesThroughEveryMCPNaming(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "knowledge/a.adr.md", "---\ntitle: \"A\"\nstatus: draft\n---\n\nToo short to pass.\n")

	for _, tool := range []string{
		"mcp__archcore__create_document",
		"mcp__plugin_archcore_archcore__create_document",
		"archcore-create_document",
		"mcp_archcore_create_document",
	} {
		t.Run(tool, func(t *testing.T) {
			dec := postToolUseHandler(bg(), postReq(t, base, tool, ".archcore/knowledge/a.adr.md"))
			if !strings.Contains(dec.context, "Precision") {
				t.Errorf("did not reach the precision check; context=%q", dec.context)
			}
		})
	}
}

func TestCascadeAdvisory(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		tool     string
		relation string
		wantText string
	}{
		{name: "implements is reported", tool: "mcp__archcore__update_document", relation: "implements", wantText: "impl.plan.md"},
		{name: "depends_on is reported", tool: "mcp__archcore__update_document", relation: "depends_on", wantText: "impl.plan.md"},
		{name: "extends is reported", tool: "mcp__archcore__update_document", relation: "extends", wantText: "impl.plan.md"},
		{name: "related is ignored", tool: "mcp__archcore__update_document", relation: "related"},
		{name: "create_document does not cascade", tool: "mcp__archcore__create_document", relation: "implements"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)
			manifest := fmt.Sprintf(
				`{"version":1,"files":{},"relations":[{"source":"impl.plan.md","target":"base.adr.md","type":%q}]}`,
				tt.relation)
			writeArchcoreDoc(t, base, ".sync-state.json", manifest)

			got := cascadeAdvisory(base, tt.tool, ".archcore/base.adr.md")

			if tt.wantText == "" {
				if got != "" {
					t.Errorf("cascadeAdvisory = %q, want empty", got)
				}
				return
			}
			if !strings.Contains(got, tt.wantText) {
				t.Errorf("cascadeAdvisory missing %q:\n%s", tt.wantText, got)
			}
			// The plugin's palette is being redesigned; the CLI must not name a
			// command it does not own.
			if strings.Contains(got, "/archcore:") {
				t.Errorf("cascade names a plugin command:\n%s", got)
			}
		})
	}
}

func TestCascadeAdvisory_SilentWithoutManifest(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	if got := cascadeAdvisory(base, "mcp__archcore__update_document", ".archcore/a.adr.md"); got != "" {
		t.Errorf("cascadeAdvisory = %q, want empty", got)
	}
}

// TestValidationAdvisory_CapsAndCounts: five findings, then a count. A wall of
// issues reads as noise.
func TestValidationAdvisory_CapsAndCounts(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	for i := range 8 {
		writeArchcoreDoc(t, base, fmt.Sprintf("knowledge/bad-%d.adr.md", i), "no frontmatter at all\n")
	}

	got := validationAdvisory(base)

	// The literal, not maxValidationIssues: comparing rendered output against the
	// constant follows it wherever it moves, so changing the budget failed nothing.
	const budget = 5
	if n := strings.Count(got, "\n  - "); n != budget {
		t.Errorf("listed %d issues, want the %d-issue budget:\n%s", n, budget, got)
	}
	if !strings.Contains(got, "more — run archcore doctor") {
		t.Errorf("truncated list should name the remainder:\n%s", got)
	}
}

func TestValidationAdvisory_SilentWhenClean(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "knowledge/ok.adr.md", "---\ntitle: \"OK\"\nstatus: draft\n---\n\nBody.\n")

	if got := validationAdvisory(base); got != "" {
		t.Errorf("validationAdvisory = %q, want empty on a clean base", got)
	}
}
