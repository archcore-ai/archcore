package cmd

import (
	"encoding/json"
	"io"
	"strings"
)

// Hook payload decoding.
//
// Five hosts send five shapes for the same event, and two nest a JSON document
// inside a string field, so the payload is held as a generic tree and read
// through explicit paths rather than a typed struct.

// hookPayload is the decoded hook input. A payload that failed to decode is not
// an error state: it is an empty payload, and every guard reading it must then
// allow. A hook that cannot parse its input must never block the user's work.
type hookPayload struct {
	root map[string]any
}

// maxPayloadBytes bounds how much of the hook input is read. A PostToolUse
// payload echoes tool_response, and a Write payload carries the whole file body,
// so the input is only as small as the host chooses to make it. Decoded into
// map[string]any it costs several times its own size in live objects.
//
// 4 MiB rather than something tighter: a payload whose leading object is cut off
// by the cap fails to parse and therefore allows, so the cap has to sit above
// any plausible single-file edit or it becomes a way to slip a write past the
// guard.
const maxPayloadBytes = 4 << 20

// decodeHookPayload reads and parses the hook input. It never fails: empty,
// truncated, or non-JSON stdin yields an empty payload.
//
// The stream decoder reads the leading JSON object and stops, so anything after
// it — a second object, padding, binary junk — is ignored rather than rejected.
// That is the safe direction and is deliberate: requiring the whole input to be
// exactly one object would turn "valid object plus trailing bytes" from a
// payload the write guard can act on into an empty one, which every guard
// treats as allow. Strictness here would BE the bypass.
//
// The consequence to keep in mind: the cap bounds the bytes read, not the size
// of what the host sent.
func decodeHookPayload(r io.Reader) *hookPayload {
	var root map[string]any
	if json.NewDecoder(io.LimitReader(r, maxPayloadBytes)).Decode(&root) != nil {
		return &hookPayload{}
	}
	return &hookPayload{root: root}
}

// str reads a string at a path of nested keys. An intermediate value that is
// itself a JSON document in string form is decoded on the way through, which is
// what makes Copilot's toolArgs and Cursor's tool_input readable by path.
func (p *hookPayload) str(path ...string) string {
	var cur any = p.root
	for _, key := range path {
		node, ok := asObject(cur)
		if !ok {
			return ""
		}
		cur, ok = node[key]
		if !ok {
			return ""
		}
	}
	s, _ := cur.(string)
	return s
}

// firstStr returns the first non-empty value among several candidate paths.
func (p *hookPayload) firstStr(paths ...[]string) string {
	for _, path := range paths {
		if v := p.str(path...); v != "" {
			return v
		}
	}
	return ""
}

// asObject coerces a node to an object, decoding an embedded JSON document when
// the node is a string carrying one.
func asObject(v any) (map[string]any, bool) {
	switch t := v.(type) {
	case map[string]any:
		return t, true
	case string:
		trimmed := strings.TrimSpace(t)
		if !strings.HasPrefix(trimmed, "{") {
			return nil, false
		}
		var nested map[string]any
		if json.Unmarshal([]byte(trimmed), &nested) != nil {
			return nil, false
		}
		return nested, true
	default:
		return nil, false
	}
}

// sessionID returns the host's identifier for the conversation. Claude Code and
// Codex send session_id; Cursor sends conversation_id.
func (p *hookPayload) sessionID() string {
	return p.firstStr([]string{"session_id"}, []string{"conversation_id"})
}

func (p *hookPayload) source() string { return p.str("source") }

func (p *hookPayload) cwd() string { return p.str("cwd") }

// toolName returns the invoked tool, folded to its canonical form.
func (p *hookPayload) toolName() string { return foldToolName(p.rawToolName()) }

// rawToolName returns the tool name exactly as the host sent it.
func (p *hookPayload) rawToolName() string {
	return p.firstStr([]string{"tool_name"}, []string{"toolName"})
}

// filePath returns the target of a file-writing tool. The candidate list spans
// every host shape; the first hit wins, and an absent path means "no file
// involved", which every guard treats as allow.
//
// The bare "path" key is consulted only when the caller is NOT an archcore MCP
// tool. The same key means two different things depending on who sent it: a
// file being edited, or the document an MCP call is acting on. Reading it
// unconditionally would let a sanctioned MCP write look like a direct one and
// get blocked by its own guard.
func (p *hookPayload) filePath() string {
	paths := [][]string{
		{"tool_input", "file_path"},
		{"toolArgs", "file_path"},
		{"toolArgs", "filePath"},
		{"tool_args", "file_path"},
		{"tool_args", "filePath"},
		{"file_path"},
		{"filePath"},
	}
	if !isArchcoreMCPTool(p.rawToolName()) {
		paths = append(paths,
			[]string{"tool_input", "path"},
			[]string{"toolArgs", "path"},
			[]string{"tool_args", "path"})
	}
	return p.firstStr(paths...)
}

// docPath returns the .archcore/ document an MCP tool acted on.
func (p *hookPayload) docPath() string {
	return p.firstStr(
		[]string{"tool_input", "path"},
		[]string{"toolArgs", "path"},
		[]string{"tool_args", "path"},
		[]string{"path"},
	)
}

// MCP tool-name prefixes. One server reaches the CLI under four names,
// depending on whether the plugin bundles it, how the host flattens namespaces,
// and which separator convention the host uses.
const (
	mcpCanonicalPrefix = "mcp__archcore__"
	mcpPluginPrefix    = "mcp__plugin_archcore_archcore__"
	mcpCopilotPrefix   = "archcore-"
	mcpGeminiPrefix    = "mcp_archcore_"
)

// foldToolName rewrites any known spelling of an archcore MCP tool to the
// canonical one. A tool from a different server is returned untouched: folding
// it would make an unrelated write look like a document mutation.
func foldToolName(name string) string {
	for _, prefix := range []string{mcpPluginPrefix, mcpGeminiPrefix, mcpCopilotPrefix} {
		if rest, ok := strings.CutPrefix(name, prefix); ok {
			return mcpCanonicalPrefix + rest
		}
	}
	return name
}

// isArchcoreMCPTool reports whether name is an archcore document tool, in any
// spelling.
func isArchcoreMCPTool(name string) bool {
	return strings.HasPrefix(foldToolName(name), mcpCanonicalPrefix)
}

// mutatingMCPTools are the archcore tools that change the knowledge base. It is
// the same set the host-side matcher filters on (wiring.mcpDocumentTools), and
// a cross-package test holds the two in agreement.
var mutatingMCPTools = map[string]bool{
	"create_document": true,
	"update_document": true,
	"remove_document": true,
	"add_relation":    true,
	"remove_relation": true,
}

// isMutatingArchcoreTool reports whether name is an archcore tool that wrote
// something, in any spelling.
func isMutatingArchcoreTool(name string) bool {
	rest, ok := strings.CutPrefix(foldToolName(name), mcpCanonicalPrefix)
	return ok && mutatingMCPTools[rest]
}
