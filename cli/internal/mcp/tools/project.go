package tools

import (
	"context"
	"encoding/json"
	"strings"
	"unicode"

	"archcore-cli/internal/docs"

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

// SourceArgument is the response field that names how the session's project
// reaches the call root (source-mounts.spec, behaviors 7–10).
const SourceArgument = "source"

type sessionRootKey struct{}

// WithSessionRoot records the session root a call was made from, so a handler
// can name the call root's source next to its project.
func WithSessionRoot(ctx context.Context, session string) context.Context {
	return context.WithValue(ctx, sessionRootKey{}, session)
}

// SourceOf names the call root as the session sees it: "local" for the session
// root, the id the session root declares it under, or "" for a project the
// session root does not declare. A context without a session root is a
// single-project call and answers "local".
func SourceOf(ctx context.Context, root string) string {
	session, ok := ctx.Value(sessionRootKey{}).(string)
	if !ok || session == root {
		return string(docs.SourceKindLocal)
	}
	id, _ := docs.SourceIDFor(session, root)
	return id
}

// WithProjectField appends the call root as "project", then its source as
// "source" when the session names one, as the last keys of a successful result
// whose payload is one JSON object (call-root-resolution.spec §10). The root is
// the one absolute path a response carries (no-absolute-paths-in-mcp-errors.rule
// §4), so an error result is returned unchanged.
func WithProjectField(result *mcp.CallToolResult, root, source string) *mcp.CallToolResult {
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
	field := projectField(root, source)
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
func projectField(root, source string) string {
	quoted, _ := json.Marshal(root) // a string always marshals
	field := `,"` + ProjectArgument + `":` + string(quoted)
	if source != "" {
		quotedSource, _ := json.Marshal(source)
		field += `,"` + SourceArgument + `":` + string(quotedSource)
	}
	return field
}

// projectFieldBytes is how many bytes WithProjectField adds for the call root.
// A byte budget reserves it, so the appended fields cannot push a response over
// its ceiling.
func projectFieldBytes(ctx context.Context, root string) int {
	return len(projectField(root, SourceOf(ctx, root)))
}
