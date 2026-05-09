package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"archcore-cli/templates"

	"github.com/mark3labs/mcp-go/mcp"
)

// NewCreateDocumentTool returns the tool definition for create_document.
func NewCreateDocumentTool() mcp.Tool {
	return mcp.NewTool("create_document",
		mcp.WithDescription(`Create a new structured document in the .archcore/ knowledge base.

BEFORE calling this tool: call list_documents to confirm no equivalent document already exists. Do not create duplicates.

Use this tool to permanently capture distilled, high-value knowledge — not temporary notes, chat logs, or speculative content.

Document types and when to use each:
  adr       — A decision that has been made, with context and consequences
                § required sections: Context, Decision, Alternatives Considered, Consequences
  rfc       — A proposal open for team review before a decision is made
                § required sections: Summary, Motivation, Detailed Design, Drawbacks, Alternatives
  rule      — A mandatory team standard or required behavior
                § required sections: Rule (imperative statements), Rationale, Examples (Good/Bad), Enforcement
  guide     — Step-by-step instructions for completing a task
                § required sections: Prerequisites, Steps (numbered), Verification, Common Issues
  doc       — Non-behavioral reference material: registries, glossaries, lookup tables, component lists
                § required sections: Overview, Content (sections/tables), Examples
  spec      — Canonical normative contract for a concrete system, component, interface, schema, or protocol
                § required sections: Purpose, Scope, Authority, Subject, Contract Surface, Normative Behavior, Constraints, Invariants, Error Handling, Conformance
  prd       — Product requirements with goals, scope, and acceptance criteria
                § required sections: Vision, Problem Statement, Goals and Success Metrics, Requirements
  idea      — A product or technical concept worth exploring
                § required sections: Idea, Value, Possible Implementation, Risks and Constraints
  plan      — A concrete implementation plan with defined tasks
                § required sections: Goal, Tasks (phased), Acceptance Criteria, Dependencies
  task-type — A proven pattern for a typical recurring implementation task
                § required sections: What, When to Use, Steps, Example, Things to Watch Out For
  cpat      — A code pattern change: documents how and why a convention or approach changed
                § required sections: What Changed, Why, Before, After, Scope
  mrd       — Market analysis with TAM/SAM/SOM, competitive landscape, and market needs
                § required sections: Market Landscape, TAM/SAM/SOM, Competitive Analysis, Market Needs, Opportunity and Timing
  brd       — Business justification with objectives, ROI, stakeholders, and budget
                § required sections: Business Objectives, Stakeholders, Business Rules and Constraints, Success Metrics and ROI, Dependencies
  urd       — User needs with personas, journeys, and usability requirements
                § required sections: User Personas, User Journeys, User Requirements, Usability Requirements, Acceptance Criteria
  brs       — Business requirements specification (ISO 29148 §9.3): formalized business-level requirements
                § required sections: Business Purpose and Scope, Business Overview (incl. Information Environment), Mission/Goals/Objectives, Business Operations, Business Constraints, High-Level Operational Concept (incl. Operational Quality), Project Constraints, Success Criteria, Assumptions/Dependencies, Traceability
  strs      — Stakeholder requirements specification (ISO 29148 §9.4): formalized stakeholder-level requirements
                § required sections: Purpose and Scope, System Overview, Business Context, Stakeholder Classes, Operational Concept (ConOps), Stakeholder Requirements, System Processes, Operational Policies/Rules, Operational Constraints, Compliance/Regulatory, Project Constraints, Traceability
  syrs      — System requirements specification (ISO 29148 §9.5): formalized system-level requirements
                § required sections: System Purpose and Scope, System Overview, System Requirements (incl. Information Management), System Interfaces, System Operations, Policy/Regulation, Life Cycle Sustainment, Assumptions/Dependencies, Verification Approach, Traceability
  srs       — Software requirements specification (ISO 29148 §9.6): formalized software component requirements
                § required sections: Purpose and Scope, Product Perspective (incl. Assumptions/Dependencies), Software Requirements, External Interfaces, Data Requirements, Usability Requirements, Performance, Design Constraints, Software Quality Attributes, Verification Matrix, Traceability

Returns: JSON with path, type, category, title, status, tags (when present), and optionally nearby_documents — paths of other documents in the same directory. Treat nearby_documents as a hint only: review each candidate and call add_relation explicitly when a semantic link exists. Do not link every neighbor by default.`),
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
			mcp.Enum("draft", "accepted", "rejected"),
		),
		mcp.WithString("content",
			mcp.Description(`Markdown body of the document. RECOMMENDED: omit this parameter to get the standard template for the document type — it contains all required sections with guidance placeholders.

If you provide content, you MUST include the required sections for the chosen type (see § required sections above). Do not invent a structure — follow the template's section layout. Do not include a top-level heading — the title is stored in frontmatter separately.`),
		),
		mcp.WithString("directory",
			mcp.Description(`Optional subdirectory inside .archcore/ where the file should be created. Use to organize documents by domain, feature, or team (e.g. "auth", "payments", "infrastructure/k8s"). If omitted, the file is created in the .archcore/ root. Must not contain ".." or start with "/".`),
		),
		mcp.WithArray("tags",
			mcp.Description(`Optional. Add tags when the document is relevant to multiple teams or domains. Lowercase alphanumeric with hyphens, underscores, colons, or pipes (e.g. "frontend", "team-platform", "team:payments", "some|flag").`),
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
	if len(nearby) > 0 {
		result["nearby_documents"] = nearby
	}
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
