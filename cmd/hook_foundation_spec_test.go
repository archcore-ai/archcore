// Hook foundation contract.
//
// Three properties make a hook safe to install on someone's machine, and none
// of them is visible from a passing happy path:
//
//  1. An unrecognized event emits nothing. Before this, cobra answered an
//     unknown subcommand by printing usage to stdout with exit 0 — and a hook's
//     stdout is the protocol channel, so an old binary meeting a new plugin
//     delivered several hundred bytes of help text into the model's context.
//  2. An internal failure allows. A panic exits 2, which most hosts read as an
//     explicit deny; on Copilot every non-zero exit denies AND discards the
//     reason, so a defect here blocks the user's edit with no explanation.
//  3. One MCP server is recognized under all five of its names, and no other
//     server is ever mistaken for it.
package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func dialectByID(t *testing.T, id string) hostDialect {
	t.Helper()
	for _, d := range hookDialects {
		if string(d.id) == id {
			return d
		}
	}
	t.Fatalf("no dialect registered for %q", id)
	return hostDialect{}
}

// TestEmitDeny_PerHostShape pins the deny protocol per host. The Copilot row is
// the load-bearing one: it exits 0 because any non-zero exit denies there too,
// but throws away the explanation with it.
func TestEmitDeny_PerHostShape(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		wantCode    int
		wantStdout  string
		stdoutEmpty bool
	}{
		{name: "claude-code exits 2 with an empty stdout", host: "claude-code", wantCode: 2, stdoutEmpty: true},
		{name: "cursor exits 2 with an empty stdout", host: "cursor", wantCode: 2, stdoutEmpty: true},
		{name: "gemini-cli exits 2 with an empty stdout", host: "gemini-cli", wantCode: 2, stdoutEmpty: true},
		{name: "codex-cli exits 2 with an empty stdout", host: "codex-cli", wantCode: 2, stdoutEmpty: true},
		{name: "opencode exits 2 with an empty stdout", host: "opencode", wantCode: 2, stdoutEmpty: true},
		{
			name:       "copilot exits 0 and carries the reason in JSON",
			host:       "copilot",
			wantCode:   0,
			wantStdout: `{"permissionDecision":"deny","permissionDecisionReason":"blocked reason"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := dialectByID(t, tt.host)
			var code int
			out := captureStdout(t, func() {
				code = emitDecision(d, eventPreToolUse, denyHook("blocked reason"))
			})
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			if tt.stdoutEmpty && out != "" {
				t.Errorf("stdout = %q, want empty", out)
			}
			if tt.wantStdout != "" && out != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", out, tt.wantStdout)
			}
		})
	}
}

// TestEmitContext_PerHostEnvelope pins the context envelope. A host that does
// not recognize its envelope reads no context at all — a silent failure.
func TestEmitContext_PerHostEnvelope(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{name: "claude-code takes the wrapper", host: "claude-code", want: `{"hookSpecificOutput":{"additionalContext":"note","hookEventName":"PreToolUse"}}`},
		// Gemini takes the same wrapper but is wired as BeforeTool / AfterTool, and
		// the wrapper carries the event name — sending the canonical one names an
		// event this host never registered.
		{name: "gemini-cli takes the wrapper with its own event name", host: "gemini-cli", want: `{"hookSpecificOutput":{"additionalContext":"note","hookEventName":"BeforeTool"}}`},
		{name: "cursor takes a flat snake_case key", host: "cursor", want: `{"additional_context":"note"}`},
		{name: "copilot takes a flat camelCase key", host: "copilot", want: `{"additionalContext":"note"}`},
		{name: "opencode takes plain text", host: "opencode", want: "note\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := dialectByID(t, tt.host)
			out := captureStdout(t, func() {
				emitDecision(d, eventPreToolUse, adviseHook("note"))
			})
			if out != tt.want {
				t.Errorf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

// TestEmitDecision_CopilotStdoutIsOneDocument: Copilot strips progress objects,
// concatenates every remaining stdout line and runs ONE JSON.parse. A failed
// parse is treated as no output at all, so a second document or a stray line
// would not add noise — it would discard the entire payload.
func TestEmitDecision_CopilotStdoutIsOneDocument(t *testing.T) {
	d := dialectByID(t, "copilot")
	for _, dec := range []hookDecision{denyHook("no"), adviseHook("ctx")} {
		out := captureStdout(t, func() { emitDecision(d, eventPostToolUse, dec) })
		var any map[string]any
		if err := json.Unmarshal([]byte(out), &any); err != nil {
			t.Errorf("stdout is not a single JSON document: %v\n%s", err, out)
		}
		if strings.Contains(strings.TrimRight(out, "\n"), "\n") {
			t.Errorf("stdout carries more than one line:\n%q", out)
		}
	}
}

// TestEmitDecision_AllowWritesNothing: silence is the common case, and it must
// cost no bytes on any host.
func TestEmitDecision_AllowWritesNothing(t *testing.T) {
	for _, d := range hookDialects {
		t.Run(string(d.id), func(t *testing.T) {
			out := captureStdout(t, func() {
				if code := emitDecision(d, eventPreToolUse, allowHook()); code != 0 {
					t.Errorf("allow returned exit code %d, want 0", code)
				}
			})
			if out != "" {
				t.Errorf("allow wrote %q to stdout, want nothing", out)
			}
		})
	}
}

// TestSafeHandle_RecoversPanic: an internal defect must not turn into a deny.
func TestSafeHandle_RecoversPanic(t *testing.T) {
	t.Parallel()
	panicking := func(context.Context, hookRequest) hookDecision { panic("boom") }

	dec := safeHandle(bg(), panicking, hookRequest{baseDir: t.TempDir(), payload: &hookPayload{}})

	if dec.deny {
		t.Error("a panicking guard produced a deny; it must allow")
	}
	if dec.context != "" {
		t.Errorf("a panicking guard produced context %q; it must be silent", dec.context)
	}
}

// TestHookCommand_UnknownEventIsSilent closes the cobra trap.
func TestHookCommand_UnknownEventIsSilent(t *testing.T) {
	cases := [][]string{
		{"hooks", "claude-code", "no-such-event"},
		{"hooks", "copilot", "pre-tool-use-typo"},
		{"hooks", "no-such-host", "session-start"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			root := NewRootCmd("test")
			root.SetArgs(args)
			var err error
			out := captureStdout(t, func() { err = root.Execute() })
			if err != nil {
				t.Errorf("%v returned %v, want nil — a hook must not fail on an unknown event", args, err)
			}
			if out != "" {
				t.Errorf("%v wrote %q to stdout, want nothing", args, out)
			}
		})
	}
}

// TestFoldToolName covers the five spellings one server arrives under, and the
// boundary that keeps a different server from being folded into it.
func TestFoldToolName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		in          string
		want        string
		wantArchore bool
	}{
		{name: "canonical passes through", in: "mcp__archcore__create_document", want: "mcp__archcore__create_document", wantArchore: true},
		{name: "plugin-bundled folds", in: "mcp__plugin_archcore_archcore__create_document", want: "mcp__archcore__create_document", wantArchore: true},
		{name: "copilot flatten folds", in: "archcore-create_document", want: "mcp__archcore__create_document", wantArchore: true},
		{name: "gemini underscore form folds", in: "mcp_archcore_create_document", want: "mcp__archcore__create_document", wantArchore: true},
		// OpenCode prefixes an MCP tool with the server name and one underscore.
		// Unfolded, it is not recognized as ours — and the consequence is not a
		// skipped post-write check but a deny: filePath() then reads the bare
		// "path" key, and the write guard blocks the document the MCP tool was
		// sanctioned to write.
		{name: "opencode underscore form folds", in: "archcore_create_document", want: "mcp__archcore__create_document", wantArchore: true},
		{name: "a foreign MCP server is never rewritten", in: "mcp__other__create_document", want: "mcp__other__create_document"},
		{name: "a native tool is not mistaken for a server prefix", in: "Write", want: "Write"},
		{name: "a lookalike native tool is left alone", in: "create", want: "create"},
		// OpenCode's prefix is the loose one — a bare server name and an
		// underscore — so the boundary that matters for it is a foreign server
		// flattened the same way. Its name is not ours, so it must pass through.
		{name: "a foreign underscore server is left alone", in: "other_create_document", want: "other_create_document"},
		{name: "a foreign gemini-style server is left alone", in: "mcp_other_create_document", want: "mcp_other_create_document"},
		// The boundary the loose prefixes actually have to hold: a foreign server
		// whose name STARTS with ours. The separator is inside names as well as
		// between them, so nothing in the string marks where the server name ends
		// — only the tool set does. Folding these would make the write guard skip
		// the "path" key and wave a foreign server's write into .archcore/ through.
		{name: "a foreign server prefixed with ours is left alone", in: "archcore_docs_create_document", want: "archcore_docs_create_document"},
		{name: "a foreign gemini-style server prefixed with ours is left alone", in: "mcp_archcore_docs_create_document", want: "mcp_archcore_docs_create_document"},
		{name: "a foreign copilot-style server prefixed with ours is left alone", in: "archcore-docs-create_document", want: "archcore-docs-create_document"},
		// A tool this server does not have is not ours to claim under a loose
		// spelling either, whoever sent it.
		{name: "an unknown tool under the loose prefix is left alone", in: "archcore_write_file", want: "archcore_write_file"},
		// The unambiguous spellings need no such bound: the double underscore
		// delimits the server name, so an unknown tool is still plainly ours. The
		// plugin row is the one that proves it — it is the only unambiguous
		// spelling the fold actually rewrites, so bounding it would show up here.
		// The canonical row below passes through untouched and would survive that
		// same mistake; it guards the passthrough, not the bound.
		{name: "an unknown tool under the plugin prefix is still ours", in: "mcp__plugin_archcore_archcore__future_tool", want: "mcp__archcore__future_tool", wantArchore: true},
		{name: "an unknown tool under the canonical prefix passes through", in: "mcp__archcore__future_tool", want: "mcp__archcore__future_tool", wantArchore: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := foldToolName(tt.in); got != tt.want {
				t.Errorf("foldToolName(%q) = %q, want %q", tt.in, got, tt.want)
			}
			if got := isArchcoreMCPTool(tt.in); got != tt.wantArchore {
				t.Errorf("isArchcoreMCPTool(%q) = %v, want %v", tt.in, got, tt.wantArchore)
			}
		})
	}
}

// TestHookPayload_NestedShapes: two hosts nest a JSON document inside a string
// field, which is why the payload is read by path rather than by struct.
func TestHookPayload_NestedShapes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		payload  string
		wantFile string
		wantTool string
	}{
		{
			name:     "claude code object tool_input",
			payload:  `{"tool_name":"Write","tool_input":{"file_path":"src/a.go"}}`,
			wantFile: "src/a.go",
			wantTool: "Write",
		},
		{
			name:     "copilot native object toolArgs",
			payload:  `{"toolName":"create","toolArgs":{"path":"src/b.go"}}`,
			wantFile: "src/b.go",
			wantTool: "create",
		},
		{
			name:     "copilot escaped-string toolArgs",
			payload:  `{"toolName":"edit","toolArgs":"{\"path\":\"src/c.go\"}"}`,
			wantFile: "src/c.go",
			wantTool: "edit",
		},
		{
			name:     "cursor escaped-string tool_input with a bare MCP name",
			payload:  `{"conversation_id":"c1","tool_name":"archcore-update_document","tool_input":"{\"path\":\".archcore/x.adr.md\"}"}`,
			wantTool: "mcp__archcore__update_document",
		},
		{
			name:    "unparsable payload yields nothing",
			payload: `{{{`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := decodeHookPayload(strings.NewReader(tt.payload))
			if got := p.filePath(); got != tt.wantFile {
				t.Errorf("filePath() = %q, want %q", got, tt.wantFile)
			}
			if got := p.toolName(); got != tt.wantTool {
				t.Errorf("toolName() = %q, want %q", got, tt.wantTool)
			}
		})
	}
}

// TestHookPayload_ExplicitPathBeatsFirstOccurrence: the shell original searched
// the raw text for the first occurrence of a key, which worked only because
// tool_input happens to precede the tool_response echo. Reading by path gets
// the same answer for a reason that survives a host reordering its fields.
func TestHookPayload_ExplicitPathBeatsFirstOccurrence(t *testing.T) {
	t.Parallel()
	// The echo comes FIRST here — document order that would defeat a text scan.
	payload := `{"tool_response":{"structuredContent":{"path":"WRONG.adr.md"}},` +
		`"tool_input":{"path":".archcore/right.adr.md"}}`

	p := decodeHookPayload(strings.NewReader(payload))

	if got := p.docPath(); got != ".archcore/right.adr.md" {
		t.Errorf("docPath() = %q, want the tool_input value regardless of field order", got)
	}
}
