package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"archcore-cli/internal/config"
	"archcore-cli/templates"

	"github.com/mark3labs/mcp-go/mcp"
)

// maxNearbyDocuments caps the nearby_documents hint returned by create_document.
// Capping bounds response size in directories with many sibling documents.
const maxNearbyDocuments = 5

// NewCreateDocumentTool returns the tool definition for create_document.
func NewCreateDocumentTool() mcp.Tool {
	return mcp.NewTool("create_document",
		mcp.WithDescription(`Create a new structured document in the .archcore/ knowledge base.

BEFORE calling this tool: call list_documents to confirm no equivalent document already exists. Do not create duplicates.

Use this tool to permanently capture distilled, high-value knowledge — not temporary notes, chat logs, or speculative content.

Document types (sections shown are core — full templates auto-generated when content is omitted):
  adr       — Decision record · sections: Context, Decision, Alternatives, Consequences
  rfc       — Open proposal · sections: Summary, Motivation, Detailed Design, Drawbacks, Alternatives
  rule      — Team standard · sections: Rule, Rationale, Examples (Good/Bad), Enforcement
  guide     — Step-by-step howto · sections: Prerequisites, Steps, Verification, Common Issues
  doc       — Reference material (tables/glossaries) · sections: Overview, Content, Examples
  spec      — Contract of a depended-on boundary (API/interface/schema/protocol) — captured after code or specified ahead of it
                · core sections: Purpose, Scope, Normative Behavior, Constraints, Invariants, Error Handling, Conformance
  prd       — Product requirements · sections: Vision, Problem, Goals & Metrics, Requirements
  idea      — Concept worth exploring · sections: Idea, Value, Possible Implementation, Risks
  plan      — Implementation plan · sections: Goal, Tasks, Acceptance Criteria, Dependencies
  rnd       — Bounded research · sections: Goal, Questions, Approach, Findings, Recommendation, Next Action
  task-type — Recurring task pattern · sections: What, When, Steps, Example, Pitfalls
  cpat      — Code pattern change · sections: What Changed, Why, Before, After, Scope
  mrd       — Market analysis · sections: Market, TAM/SAM/SOM, Competitive, Needs, Opportunity
  brd       — Business justification · sections: Objectives, Stakeholders, Constraints, ROI
  urd       — User needs · sections: Personas, Journeys, User Reqs, Usability, Acceptance
  brs       — ISO 29148 §9.3 business req spec
                · sections: Mission/Goals, Operational Concept, Business Constraints, Traceability
  strs      — ISO 29148 §9.4 stakeholder req spec
                · sections: Stakeholder Classes, ConOps, Stakeholder Reqs, Traceability
  syrs      — ISO 29148 §9.5 system req spec
                · sections: System Boundary, System Reqs, Interfaces, Verification Approach
  srs       — ISO 29148 §9.6 software req spec
                · sections: Scope, Software Reqs, External Interfaces, Verification Matrix

Returns: JSON with path, type, category, title, status, tags (when present), and optionally nearby_documents — paths of other documents in the same directory (capped at 5, sorted alphabetically). Treat nearby_documents as a hint only: review each candidate and call add_relation explicitly when a semantic link exists. Do not link every neighbor by default.`),
		mcp.WithString("type",
			mcp.Description("Document type. Choose based on the nature of the content, not the topic. If uncertain between adr and rfc: use adr only if the decision is already final."),
			mcp.Required(),
			mcp.Enum(templates.ValidTypes()...),
		),
		mcp.WithString("filename",
			mcp.Description("URL-safe slug for the filename. Use lowercase letters and hyphens only — no spaces, underscores, or special characters. Do not include the file extension. Example: \"use-postgres\", \"rate-limiting-strategy\"."),
			mcp.Required(),
		),
		mcp.WithString("title",
			mcp.Description("Human-readable document title. Write as a short descriptive phrase, not a slug. Example: \"Use PostgreSQL for primary persistence\". If omitted, derived from filename."),
		),
		mcp.WithString("status",
			mcp.Description("Initial document status. Valid values: draft, accepted, rejected. Defaults to \"draft\"."),
			mcp.Enum(templates.ValidStatusStrings()...),
		),
		mcp.WithString("content",
			mcp.Description(`Markdown body of the document. RECOMMENDED: omit this parameter to use the auto-generated template for the chosen type (it contains all required sections with guidance placeholders). If providing content manually, omit the top-level heading and follow the template section order for the chosen type.`),
		),
		mcp.WithString("directory",
			mcp.Description(`Optional subdirectory inside .archcore/ where the file should be created. Use to organize documents by domain, feature, or team (e.g. "auth", "payments", "infrastructure/k8s"). If omitted, the file is created in the .archcore/ root. Must not contain ".." or start with "/".`),
		),
		mcp.WithArray("tags",
			mcp.Description(`Optional. Tags for cross-cutting concerns when a document is relevant to multiple teams or domains. Format per server instructions (TAGS section), e.g. "frontend", "team:payments".`),
			mcp.WithStringItems(),
		),
		mcp.WithTitleAnnotation("Create Document"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
	)
}

// HandleCreateDocument handles the create_document tool call.
func HandleCreateDocument(baseDir string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		docType, err := request.RequireString("type")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		filename, err := request.RequireString("filename")
		if err != nil {
			return errorResult(err.Error()), nil
		}

		if !templates.IsValidType(docType) {
			return errorResult(fmt.Sprintf("invalid document type %q (valid: %s)", docType, strings.Join(templates.ValidTypes(), ", "))), nil
		}

		filename = strings.TrimSpace(filename)
		if filename == "" {
			return errorResult("filename is required"), nil
		}
		if strings.ContainsAny(filename, "/\\") {
			return errorResult(fmt.Sprintf("invalid filename %q: must not contain path separators", filename)), nil
		}
		if !templates.SlugRe.MatchString(filename) {
			return errorResult(fmt.Sprintf("invalid filename %q: must be lowercase alphanumeric with hyphens (e.g. \"use-postgres\")", filename)), nil
		}

		title := request.GetString("title", "")
		if title == "" {
			title = filenameToTitle(filename)
		}

		status := templates.DocStatus(request.GetString("status", "draft"))
		if !templates.IsValidStatus(status) {
			return errorResult(fmt.Sprintf("invalid status %q (valid: %s)", status, strings.Join(templates.ValidStatusStrings(), ", "))), nil
		}
		content := request.GetString("content", "")
		directory := request.GetString("directory", "")

		// Validate directory parameter.
		if directory != "" {
			directory = strings.TrimSpace(directory)
			cleaned := filepath.Clean(directory)
			if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
				return errorResult(fmt.Sprintf("invalid directory %q: must be relative and within .archcore/", directory)), nil
			}
			directory = filepath.ToSlash(cleaned)
			globals, gErr := config.LoadGlobals(baseDir)
			if gErr != nil {
				return errorResult("cannot verify global sources: settings.json is unreadable"), nil
			}
			if isReadOnlyGlobalPath(baseDir, ".archcore/"+directory, globals) {
				return errorResult("cannot create document in a read-only global source"), nil
			}
		}

		tags, tagErr := parseTags(request.GetStringSlice("tags", nil))
		if tagErr != nil {
			return errorResult(tagErr.Error()), nil
		}

		category := templates.CategoryForType(templates.DocumentType(docType))

		// Build target directory: .archcore/<directory>/ or .archcore/ root.
		var dir string
		if directory != "" {
			dir = filepath.Join(baseDir, ".archcore", directory)
		} else {
			dir = filepath.Join(baseDir, ".archcore")
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating directory %q: %w", directory, err)
		}

		outputFile := filepath.Join(dir, filename+"."+docType+".md")

		relPath, _ := filepath.Rel(baseDir, outputFile)
		relPath = filepath.ToSlash(relPath)

		if _, err := os.Stat(outputFile); err == nil {
			return errorResult(fmt.Sprintf("file already exists: %s", relPath)), nil
		}

		body := content
		if body == "" {
			body = templates.GenerateTemplate(templates.DocumentType(docType))
		} else {
			body = stripFrontmatter(body)
		}

		fileContent := buildDocumentFile(title, status, tags, body)

		if err := os.WriteFile(outputFile, []byte(fileContent), 0o644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", relPath, err)
		}

		result := map[string]any{
			"path":     relPath,
			"category": category,
			"type":     docType,
			"title":    title,
			"status":   status,
		}
		if len(tags) > 0 {
			result["tags"] = tags
		}

		populateNearbyDocuments(baseDir, relPath, result)

		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshaling result: %w", err)
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

// populateNearbyDocuments scans for other documents in the same directory as
// relPath and exposes them as a hint in result["nearby_documents"]. The agent
// decides whether any of them warrants an explicit relation via add_relation.
// Results are sorted lexicographically and capped at maxNearbyDocuments to
// bound response size in large directories.
func populateNearbyDocuments(baseDir, relPath string, result map[string]any) {
	allDocs, scanErr := ScanDocuments(baseDir)
	if scanErr != nil {
		return
	}

	createdDir := filepath.Dir(relPath)
	var nearby []string
	for _, d := range allDocs {
		if d.Path != relPath && filepath.Dir(d.Path) == createdDir {
			nearby = append(nearby, d.Path)
		}
	}
	if len(nearby) == 0 {
		return
	}
	slices.Sort(nearby)
	if len(nearby) > maxNearbyDocuments {
		nearby = nearby[:maxNearbyDocuments]
	}
	result["nearby_documents"] = nearby
}

// filenameToTitle converts a slug like "oauth-tokens" to "Oauth Tokens".
func filenameToTitle(filename string) string {
	parts := strings.Split(filename, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
