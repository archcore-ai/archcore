package tools

import (
	"encoding/json"
	"strings"
	"unicode"

	"github.com/mark3labs/mcp-go/mcp"
)

// ProjectArgument is the name of the optional argument every tool takes to
// address a project other than the session's (call-root-resolution.spec).
const ProjectArgument = "project"

const projectArgumentDescription = `Absolute path to the root of the project to use: the directory that holds .archcore/, never .archcore/ itself. Omit it to use this session's project. Every response names the project it used in its "project" field; pass that value back to address the same project again.`

// WithProjectArgument adds the optional project argument to tool's input
// schema.
func WithProjectArgument(tool mcp.Tool) mcp.Tool {
	if tool.InputSchema.Properties == nil {
		tool.InputSchema.Properties = map[string]any{}
	}
	tool.InputSchema.Properties[ProjectArgument] = map[string]any{
		"type":        "string",
		"description": projectArgumentDescription,
	}
	return tool
}

// WithProjectField appends the call root as the last key, "project", of a
// successful result whose payload is one JSON object
// (call-root-resolution.spec §10). It is the one absolute path a response
// carries (no-absolute-paths-in-mcp-errors.rule §4), so an error result is
// returned unchanged.
func WithProjectField(result *mcp.CallToolResult, root string) *mcp.CallToolResult {
	if result == nil || result.IsError || len(result.Content) != 1 {
		return result
	}
	text, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		return result
	}
	body := strings.TrimRightFunc(text.Text, unicode.IsSpace)
	if len(body) < 2 || body[0] != '{' || body[len(body)-1] != '}' {
		return result
	}
	field := projectField(root)
	if strings.TrimSpace(body[1:len(body)-1]) == "" {
		body = "{" + field[1:] + "}"
	} else {
		body = body[:len(body)-1] + field + "}"
	}
	result.Content[0] = mcp.NewTextContent(body)
	return result
}

// projectField is the JSON text WithProjectField appends, leading comma
// included.
func projectField(root string) string {
	quoted, _ := json.Marshal(root) // a string always marshals
	return `,"` + ProjectArgument + `":` + string(quoted)
}

// projectFieldBytes is how many bytes WithProjectField adds for root. A byte
// budget reserves it, so the appended field cannot push a response over its
// ceiling.
func projectFieldBytes(root string) int {
	return len(projectField(root))
}
