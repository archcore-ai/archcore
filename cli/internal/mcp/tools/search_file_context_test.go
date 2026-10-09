package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func acceptedDoc(title, body string) string {
	return fmt.Sprintf("---\ntitle: %q\nstatus: accepted\n---\n\n%s\n", title, body)
}

type forPathRow struct {
	Path   string `json:"path"`
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

type forPathResponse struct {
	Truncated bool `json:"truncated"`
	Omitted   *struct {
		Matched     int    `json:"matched"`
		MoreMatched string `json:"more_matched"`
		General     int    `json:"general"`
		MoreGeneral string `json:"more_general"`
	} `json:"omitted"`
	Results []forPathRow `json:"results"`
}

func searchForPath(t *testing.T, base string, args map[string]any) (forPathResponse, string) {
	t.Helper()
	result, err := callTool(HandleSearchDocuments(StaticRoot(base)), args)
	if err != nil {
		t.Fatal(err)
	}
	text := resultText(t, result)
	if result.IsError {
		return forPathResponse{}, text
	}
	var resp forPathResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("decoding %s: %v", text, err)
	}
	return resp, text
}

// TestSearchForPath_Refusals pins search-documents.spec §1.11, §1.12, and
// file-context-resolution.spec failure rule 3.
func TestSearchForPath_Refusals(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	tests := []struct {
		name string
		args map[string]any
		want string
	}{
		{name: "with path_ref", args: map[string]any{"for_path": "src/a.go", "path_ref": "src/"}, want: "for_path cannot be combined with path_ref or content"},
		{name: "with content", args: map[string]any{"for_path": "src/a.go", "content": "x"}, want: "for_path cannot be combined with path_ref or content"},
		{name: "outside the project", args: map[string]any{"for_path": "../elsewhere/a.go"}, want: "for_path must name a file inside the project"},
		{name: "a document", args: map[string]any{"for_path": ".archcore/a.rule.md"}, want: "not an .archcore document"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, text := searchForPath(t, base, tt.args)
			if !strings.Contains(text, tt.want) {
				t.Errorf("result = %q, want it to contain %q", text, tt.want)
			}
			if strings.Contains(text, base) {
				t.Errorf("the error discloses the absolute project path: %q", text)
			}
		})
	}
}

func TestSearchForPath_RowsCarryReasonAndFullPaths(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	writeDoc(t, base, "web", "seam.doc.md", acceptedDoc("Seam", "FullLayout.tsx renders the shell."))
	writeDoc(t, base, "web", "dir.rule.md", acceptedDoc("Dir Rule", "Applies to src/app/ code."))
	writeDoc(t, base, "", "guard.rule.md", acceptedDoc("Guard", "Return early."))

	resp, text := searchForPath(t, base, map[string]any{"for_path": "src/app/FullLayout.tsx"})

	want := []forPathRow{
		{Path: ".archcore/web/seam.doc.md", Type: "doc", Reason: "file"},
		{Path: ".archcore/web/dir.rule.md", Type: "rule", Reason: "directory"},
		{Path: ".archcore/guard.rule.md", Type: "rule", Reason: "general"},
	}
	if len(resp.Results) != len(want) {
		t.Fatalf("results = %+v, want %+v\n%s", resp.Results, want, text)
	}
	for i := range want {
		if resp.Results[i] != want[i] {
			t.Errorf("row %d = %+v, want %+v", i, resp.Results[i], want[i])
		}
	}
	if resp.Omitted != nil {
		t.Errorf("omitted = %+v, want absent", resp.Omitted)
	}
}

// TestSearchForPath_FiltersAndCaps pins §1.14 and behavior 10 of the file
// context spec: filters narrow the rows, caps state the rest.
func TestSearchForPath_FiltersAndCaps(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	for i := range 7 {
		writeDoc(t, base, "", fmt.Sprintf("m-%d.rule.md", i), acceptedDoc(fmt.Sprintf("Matched %d", i), "Applies to src/api/ code."))
	}
	for i := range 12 {
		writeDoc(t, base, "", fmt.Sprintf("g-%02d.rule.md", i), acceptedDoc(fmt.Sprintf("General %d", i), "Return early."))
	}
	writeDoc(t, base, "", "a.adr.md", acceptedDoc("Decision", "Applies to src/api/ code."))

	resp, text := searchForPath(t, base, map[string]any{"for_path": "src/api/h.go"})
	if resp.Omitted == nil || resp.Omitted.Matched != 3 || resp.Omitted.General != 2 {
		t.Fatalf("omitted = %+v, want 3 matched and 2 general\n%s", resp.Omitted, text)
	}
	if resp.Omitted.MoreMatched != `search_documents(path_ref="src/api/h.go")` || !strings.Contains(resp.Omitted.MoreGeneral, "list_documents") {
		t.Errorf("omitted calls = %+v", resp.Omitted)
	}
	if len(text) > forPathByteBudget {
		t.Errorf("response holds %d bytes, budget %d", len(text), forPathByteBudget)
	}

	resp, _ = searchForPath(t, base, map[string]any{"for_path": "src/api/h.go", "types": []any{"adr"}})
	if len(resp.Results) != 1 || resp.Results[0].Path != ".archcore/a.adr.md" {
		t.Errorf("types filter rows = %+v, want the adr only", resp.Results)
	}
}

func TestSearchForPath_StaysInsideTheByteBudget(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	long := strings.Repeat("Очень длинный заголовок правила ", 8)
	for i := range 12 {
		writeDoc(t, base, "", fmt.Sprintf("g-%02d.rule.md", i), acceptedDoc(fmt.Sprintf("%s %d", long, i), "Return early."))
	}

	resp, text := searchForPath(t, base, map[string]any{"for_path": "src/api/h.go"})

	if len(text) > forPathByteBudget {
		t.Errorf("response holds %d bytes, budget %d", len(text), forPathByteBudget)
	}
	if !resp.Truncated || resp.Omitted == nil || resp.Omitted.General == 0 {
		t.Errorf("a budget cut must set truncated and count the omitted rows: %+v", resp.Omitted)
	}
}
