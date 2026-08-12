package wiring

import (
	"regexp"
	"strings"
	"testing"
)

// The values in this file are what a host filters and budgets on. None of them
// is exercised by installing — a wrong matcher means the host never starts the
// process, and a wrong timeout means it kills one that would have finished.

// TestMCPDocumentTools_CoversEveryNaming: one MCP server reaches a host under
// four names. The in-process fold (cmd.foldToolName) normalizes them after the
// process starts; this matcher decides whether it starts at all, so a spelling
// missing here is a check that silently never runs.
//
// The fold carries a fifth name, OpenCode's "archcore_<tool>". It is absent
// here on purpose: OpenCode is wired by a TypeScript bridge rather than by this
// matcher, so the only effect of adding it would be to rewrite five other
// hosts' configs. See the comment on mcpDocumentTools.
func TestMCPDocumentTools_CoversEveryNaming(t *testing.T) {
	t.Parallel()
	re := regexp.MustCompile("^(" + mcpDocumentTools + ")$")

	prefixes := []string{
		"mcp__archcore__",
		"mcp__plugin_archcore_archcore__",
		"mcp_archcore_",
		"archcore-",
	}
	writeTools := []string{
		"create_document", "update_document", "remove_document",
		"add_relation", "remove_relation",
	}
	for _, prefix := range prefixes {
		for _, tool := range writeTools {
			name := prefix + tool
			t.Run(name, func(t *testing.T) {
				if !re.MatchString(name) {
					t.Errorf("%q does not match the installed matcher", name)
				}
			})
		}
	}

	// Read tools and foreign servers must not match: the post-write checks scan
	// the corpus, so matching a read costs a full scan per call.
	for _, name := range []string{
		"mcp__archcore__get_document",
		"mcp__archcore__list_documents",
		"mcp__archcore__search_documents",
		"mcp__archcore__list_relations",
		"mcp__other__create_document",
		"Write",
	} {
		t.Run("excludes "+name, func(t *testing.T) {
			if re.MatchString(name) {
				t.Errorf("%q matched the document-tool matcher", name)
			}
		})
	}
}

// TestGeminiHooks_TimeoutIsMilliseconds: Gemini is the one host that does not
// measure hook timeouts in seconds. Writing seconds there gives the hook a
// budget a thousand times too small, and nothing about the config looks wrong.
func TestGeminiHooks_TimeoutIsMilliseconds(t *testing.T) {
	base := setupArchcoreDir(t)
	if err := InstallGeminiCLIHooks(base); err != nil {
		t.Fatal(err)
	}

	got := compactFile(t, configPathFor(base, ".gemini/settings.json"))

	for _, want := range []string{
		`"command":"archcore hooks gemini-cli session-start","timeout":3000`,
		`"command":"archcore hooks gemini-cli pre-tool-use","timeout":1000`,
		`"command":"archcore hooks gemini-cli post-tool-use","timeout":3000`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in:\n%s", want, got)
		}
	}
}

// TestCopilotHooks_TimeoutIsSeconds is the counterpart: everywhere else the
// unit is seconds, and the pre-write budget is deliberately the short one.
func TestCopilotHooks_TimeoutIsSeconds(t *testing.T) {
	base := setupArchcoreDir(t)
	if err := InstallCopilotHooks(base); err != nil {
		t.Fatal(err)
	}

	got := compactFile(t, configPathFor(base, ".github/hooks/archcore.json"))

	for _, want := range []string{
		`"bash":"archcore hooks copilot session-start","timeoutSec":3`,
		`"timeoutSec":1`, // pre-tool-use blocks the user, so it is the short one
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in:\n%s", want, got)
		}
	}
}
