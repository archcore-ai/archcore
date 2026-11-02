package tools

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestWithProjectField(t *testing.T) {
	t.Parallel()
	const root = `/w/a "quoted" \dir`
	field := projectField(root)

	tests := []struct {
		name   string
		result *mcp.CallToolResult
		want   string
	}{
		{name: "object", result: mcp.NewToolResultText(`{"a":1}`), want: `{"a":1` + field + `}`},
		{name: "object with trailing newline", result: mcp.NewToolResultText("{\"a\":1}\n"), want: `{"a":1` + field + `}`},
		{name: "empty object", result: mcp.NewToolResultText(`{}`), want: `{` + field[1:] + `}`},
		{name: "empty object with spaces", result: mcp.NewToolResultText(`{ }`), want: `{` + field[1:] + `}`},
		{name: "array stays", result: mcp.NewToolResultText(`[1,2]`), want: `[1,2]`},
		{name: "plain text stays", result: mcp.NewToolResultText(`done`), want: `done`},
		{name: "error stays", result: mcp.NewToolResultError(`{"x":1}`), want: `{"x":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := WithProjectField(tt.result, root)
			text, _ := mcp.AsTextContent(got.Content[0])
			if text.Text != tt.want {
				t.Fatalf("text = %q, want %q", text.Text, tt.want)
			}
			if tt.want[0] == '{' && !tt.result.IsError {
				var decoded map[string]any
				if err := json.Unmarshal([]byte(text.Text), &decoded); err != nil {
					t.Fatalf("result is not JSON: %v", err)
				}
				if decoded[ProjectArgument] != root {
					t.Errorf("project = %v, want %q", decoded[ProjectArgument], root)
				}
			}
		})
	}
	if WithProjectField(nil, root) != nil {
		t.Error("nil result was not passed through")
	}
	two := &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(`{}`), mcp.NewTextContent(`{}`)}}
	if text, _ := mcp.AsTextContent(WithProjectField(two, root).Content[0]); text.Text != `{}` {
		t.Error("a result with two contents was rewritten")
	}
}

func TestProjectFieldBytes(t *testing.T) {
	t.Parallel()
	for _, root := range []string{"/a", `/x "y"`, "/путь/проект"} {
		before := `{"k":1}`
		after, _ := mcp.AsTextContent(WithProjectField(mcp.NewToolResultText(before), root).Content[0])
		if got := len(after.Text) - len(before); got != projectFieldBytes(root) {
			t.Errorf("root %q: field added %d bytes, projectFieldBytes = %d", root, got, projectFieldBytes(root))
		}
	}
}

func TestWithProjectArgument(t *testing.T) {
	t.Parallel()
	for _, tool := range []mcp.Tool{NewInitProjectTool(), NewListDocumentsTool(), mcp.NewTool("bare")} {
		got := WithProjectArgument(tool)
		prop, ok := got.InputSchema.Properties[ProjectArgument].(map[string]any)
		if !ok || prop["type"] != "string" || prop["description"] == "" {
			t.Errorf("%s: project property = %#v", tool.Name, got.InputSchema.Properties[ProjectArgument])
		}
		for _, required := range got.InputSchema.Required {
			if required == ProjectArgument {
				t.Errorf("%s: project must stay optional", tool.Name)
			}
		}
	}
}
