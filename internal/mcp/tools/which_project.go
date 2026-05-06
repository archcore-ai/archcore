package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"

	"archcore-cli/internal/projectroot"
)

// NewWhichProjectTool exposes diagnostic info about the resolved base directory
// and guard state. Use this when other archcore tools fail with ERR_NOT_PROJECT,
// ERR_HOME_REFUSED, or similar — the response tells the agent (and the user)
// exactly which directory the server resolved to and why.
//
// This tool never fails: guard violations show up as data in `problems[]`,
// not as MCP errors.
//
// Absolute path carve-out: the response intentionally surfaces the resolved
// `base_dir` as an absolute path. The
// `mcp/no-absolute-paths-in-mcp-errors.rule.md` rule applies to error
// messages — diagnostic responses where the path *is* the payload (this
// tool's whole purpose) are explicitly excluded. Stripping the path here
// would defeat the tool.
func NewWhichProjectTool() mcp.Tool {
	return mcp.NewTool("which_project",
		mcp.WithDescription("Returns the resolved archcore base directory, how it was resolved (flag, env, walk-up), found project markers, and guard state. Use as a diagnostic when other tools fail with ERR_NOT_PROJECT or similar codes."),
	)
}

type whichProjectGuards struct {
	Strict    bool `json:"strict"`
	AllowHome bool `json:"allow_home"`
	Legacy    bool `json:"legacy"`
}

type whichProjectProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type whichProjectResponse struct {
	OK         bool                  `json:"ok"`
	BaseDir    string                `json:"base_dir,omitempty"`
	Source     string                `json:"source,omitempty"`
	CLIVersion string                `json:"cli_version"`
	Guards     whichProjectGuards    `json:"guards"`
	Markers    map[string]string     `json:"markers"`
	Problems   []whichProjectProblem `json:"problems"`
}

// HandleWhichProject returns a handler that surfaces the *Resolution captured
// at server-start time, plus a markers snapshot taken at call time.
//
// res may be nil — in that case the response reports ok=false with an
// ERR_NO_PROJECT problem. version is forwarded verbatim into cli_version.
func HandleWhichProject(res *projectroot.Resolution, version string) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if version == "" {
		version = "dev"
	}
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		guards := projectroot.GuardsFor(res)
		resp := whichProjectResponse{
			CLIVersion: version,
			Guards: whichProjectGuards{
				Strict:    guards.Strict,
				AllowHome: guards.AllowHome,
				Legacy:    guards.Legacy,
			},
			Markers:  map[string]string{},
			Problems: []whichProjectProblem{},
		}

		if res == nil {
			resp.OK = false
			resp.Problems = append(resp.Problems, whichProjectProblem{
				Code:    projectroot.CodeNoProject,
				Message: "no project root resolved at server start",
			})
		} else {
			resp.OK = true
			resp.BaseDir = res.Path
			resp.Source = string(res.Source)
			for marker, found := range projectroot.MarkerStates(res.Path) {
				resp.Markers[marker] = projectroot.MarkerStateLabel(found)
			}
		}

		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
