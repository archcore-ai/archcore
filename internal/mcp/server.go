package mcp

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"

	"archcore-cli/internal/config"
	"archcore-cli/internal/mcp/prompts"
	"archcore-cli/internal/mcp/tools"
	"archcore-cli/internal/projectroot"
)

// Option configures a server instance. Use functional options to attach the
// projectroot.Resolution and CLI version so the which_project tool has them.
type Option func(*serverConfig)

type serverConfig struct {
	resolution *projectroot.Resolution
	version    string
}

// WithResolution attaches the resolved project root to the server. Surfaced
// to the which_project diagnostic tool. If omitted, NewServer synthesizes a
// minimal resolution from baseDir.
func WithResolution(r *projectroot.Resolution) Option {
	return func(c *serverConfig) { c.resolution = r }
}

// WithVersion sets the CLI version string surfaced by which_project as
// `cli_version`. Defaults to "dev" if empty.
func WithVersion(v string) Option {
	return func(c *serverConfig) { c.version = v }
}

var mcpServerInstructions = `You are working with a project that uses Archcore — Git-native context for AI coding agents.

PROJECT INITIALIZATION:
If list_documents returns an empty result AND the user is asking to create or manage documents, the project likely has no .archcore/ directory yet. Call init_project once to initialize it, then proceed. init_project is idempotent — safe to call even if already initialized (it just returns existing settings). Do NOT attempt to create documents before the project is initialized; create_document and other mutating tools assume .archcore/ exists.

The .archcore/ directory contains Markdown files with YAML frontmatter (title, status, tags). The directory structure inside .archcore/ is free-form — you can organize documents by domain, feature, team, or any custom structure. Categories (vision, knowledge, experience) are virtual — derived automatically from the document type in the filename (slug.type.md), not from the physical directory.

Example structures:
  .archcore/auth/jwt-strategy.adr.md         → virtual category: knowledge
  .archcore/auth/auth-redesign.prd.md        → virtual category: vision
  .archcore/payments/stripe.adr.md           → virtual category: knowledge
  .archcore/infrastructure/k8s/migration.adr.md → virtual category: knowledge
  .archcore/my-doc.rule.md                   → virtual category: knowledge (root level)

Document types and their virtual categories:
  knowledge: adr (decisions), rfc (proposals), rule (standards), guide (how-tos), doc (reference), spec (contracts)
  vision:    prd (requirements), idea (concepts), plan (action plans), mrd (market requirements), brd (business requirements), urd (user requirements), brs (business req spec), strs (stakeholder req spec), syrs (system req spec), srs (software req spec)
  experience: task-type (typical task patterns), cpat (code pattern changes)

DOCUMENT RELATIONS:
Documents can be linked with directed relations stored in the sync manifest.
  Relation types:
    related     — general association (e.g., two ADRs on the same topic)
    implements  — source implements what target specifies (e.g., plan implements prd)
    extends     — source builds upon target (e.g., rfc extends an existing adr)
    depends_on  — source requires target to proceed (e.g., plan depends_on adr)

  After creating a document, check the nearby_documents hint in the response.
  Use add_relation to link related documents. Use list_relations to see existing links.

TAGS:
Use tags when a document is relevant to multiple teams or domains.
  Format: lowercase alphanumeric with hyphens, underscores, colons, or pipes (e.g., "frontend", "team-platform", "team:payments", "some|flag")
  Tag filtering uses OR semantics — a document matches if it has any of the specified tags.
  Tags narrow results but don't guarantee completeness — combine with type/category filters.
  If a tag-filtered query returns 0 results, retry without the tag filter.

WHEN TO SEARCH CONTENT:
Use search_documents to find documents by path reference, content substring, or metadata filters — not by topic guess. Unlike list_documents, it scans bodies. Prefer it over grep over .archcore/ when you need "which docs mention X".

PATH FORMAT: All tool paths use ".archcore/<path>/<slug>.<type>.md" as returned by list_documents. The add_relation and remove_relation tools also accept paths without the ".archcore/" prefix.

WORKFLOW RULES:
1. Before creating any document, call list_documents first to check whether a relevant document already exists. Do not create duplicates.
2. To read a document, call list_documents to get its path, then pass that path to get_document.
3. Only call create_document after confirming no equivalent document exists.
4. Before updating a document, confirm the intended changes with the user when possible.
5. After creating a document, review nearby_documents and consider adding relations with add_relation.
6. When reading a document, check outgoing_relations and incoming_relations for context.
7. Before deleting a document, confirm explicitly with the user. Prefer setting status to "rejected" when historical context is worth keeping.

WHEN TO CREATE:
- A technical decision is made or finalized → adr
- A significant change is being proposed for team review → rfc
- A team standard or required behavior is established → rule
- Step-by-step instructions for completing a task → guide
- Non-behavioral reference material — registries, lookup tables, glossaries, or component lists → doc
- The canonical normative contract for a concrete system, component, interface, schema, or protocol is being formalized → spec
- A proven workflow for a recurring implementation task is documented → task-type
- A coding pattern, convention, or approach has deliberately changed → cpat
- A product concept or technical idea needs capturing → idea
- An implementation plan with tasks is formed → plan
- Product requirements with goals, scope, and acceptance criteria → prd
- Market analysis with TAM/SAM/SOM, competitive landscape, and market needs → mrd
- Business justification with objectives, ROI, stakeholders, and budget → brd
- User needs with personas, journeys, and usability requirements → urd
- Business requirements formalized into ISO structure with mission, goals, operational concept, and success criteria → brs
- Stakeholder requirements formalized per stakeholder class with ConOps and compliance → strs
- System-level requirements with full boundary definition, interfaces, modes, and verification approach → syrs
- Software component requirements with per-endpoint/per-function specs and verification matrix → srs

WHEN TO UPDATE (use update_document):
- A decision is finalized → change status from "draft" to "accepted"
- A proposal is rejected → change status to "rejected"
- A plan's scope or tasks change → update content
- Do not create a new document when the existing one should be updated.

WHEN TO DELETE (use remove_document):
- A document was created by mistake or is a duplicate → delete it
- A document is entirely irrelevant and has no historical value → delete it
- Prefer update_document with status "rejected" to preserve history when the content was once valid.
- Always confirm with the user before deleting — this is permanent and removes all relations.

TYPE SELECTION RULES (use these to disambiguate):
- rule vs doc: A rule contains imperative statements ("Always do X", "Never do Y") with good/bad code examples and enforcement info. A doc is non-behavioral reference material (tables, registries, glossaries). If the content describes what exists rather than prescribing behavior, use doc.
- adr vs rfc: An adr records a decision already made. An rfc proposes a change open for review. If the decision is final, use adr; if still open for feedback, use rfc.
- guide vs doc: A guide has numbered steps the reader follows sequentially. A doc is non-sequential, non-behavioral reference material. If the reader is meant to do something step-by-step, use guide; if they look things up, use doc.
- spec vs doc: A spec documents the canonical normative contract of a concrete technical boundary — externally observable behavior, constraints, invariants, and conformance requirements. A doc describes what already exists (tables, registries, glossaries) without normative requirements. If the document defines a normative contract for a specific artifact, use spec.
- spec vs rule: A spec is a technical contract for a component or interface — it specifies what correct behavior is. A rule is a team standard for how engineers must act ("Always do X"). If the content is about a system's required behavior rather than a human practice, use spec.
- spec vs adr: A spec is the living canonical truth of how something works. An adr is the decision record explaining why a choice was made. A single decision may produce both: the adr captures the "why", the spec captures the "what". If the content is prescriptive and meant to be kept current, use spec; if it records a past decision, use adr.
- mrd vs prd: MRD analyzes the MARKET (TAM/SAM/SOM, competitors, timing) without proposing a solution. PRD proposes a PRODUCT with requirements and solution overview.
- brd vs prd: BRD focuses on BUSINESS JUSTIFICATION (ROI, budget, organizational impact). PRD focuses on PRODUCT DEFINITION (features, user stories, solution).
- urd vs prd: URD captures user needs via PERSONAS and JOURNEYS (discovery-oriented). PRD defines product requirements with acceptance criteria (specification-oriented).
- mrd vs brd: MRD is MARKET ANALYSIS (external-facing — industry, competitors, TAM). BRD is BUSINESS JUSTIFICATION (internal-facing — ROI, stakeholders, budget).
- brd vs urd: BRD captures ORGANIZATIONAL needs (goals, budget, regulations). URD captures END-USER needs (personas, journeys, usability).
- brs vs prd: BRS has ONLY business objectives with ISO structure (mission, operational concept, success criteria), no user stories or solution. PRD has user stories, functional requirements, solution overview.
- strs vs prd: StRS groups requirements PER STAKEHOLDER CLASS with ConOps (operational scenarios). PRD lists requirements by priority (P0/P1/P2).
- syrs vs adr: SyRS defines WHOLE SYSTEM BOUNDARY with interface contracts and verification approach. ADR records a single architectural decision.
- srs vs prd: SRS has PER-ENDPOINT/PER-FUNCTION requirements with verification matrix. PRD has product-level requirements.
- brs vs strs: BRS = WHY (business outcomes, technology-agnostic mission/goals). StRS = WHAT stakeholders need (operational scenarios, solution-aware, per-class).
- syrs vs srs: SyRS = WHOLE SYSTEM boundary, all interfaces and modes. SRS = SINGLE COMPONENT's detailed behavior, one software module.
- brs vs brd: BRS is a FORMAL ISO SPECIFICATION (mission, goals, operational concept, success criteria). BRD is an INFORMAL SOURCE (business justification: ROI, budget, stakeholders). BRS formalizes what BRD captures informally.
- strs vs urd: StRS is a FORMAL ISO SPECIFICATION (per-stakeholder-class requirements with ConOps). URD is an INFORMAL SOURCE (personas, journeys, usability). StRS formalizes what URD captures informally.

REQUIREMENTS TRACKS:
Three approaches to requirements engineering, choose based on project complexity:
  Product track (simple):  prd — single document covering vision, requirements, and solution. Best for small teams, internal tools, rapid prototyping.
  Sources track (discovery): mrd → brd → urd — captures where requirements come from (market, business, users). Best for product teams doing discovery, stakeholder alignment.
  ISO track (decomposition): brs → strs → syrs → srs — formal requirements cascade per ISO 29148. Best for regulated systems, multi-team projects.
All tracks can coexist — use what fits the project.

REQUIREMENTS LAYERS — Sources and Specifications are SEPARATE layers:
  Layer A (Sources):        mrd, brd, urd, prd — capture raw requirements from market, business, and user perspectives
  Layer B (Specifications): brs, strs, syrs, srs — formalize requirements into ISO-structured specifications
  Specifications formalize sources. Use "implements" relation with the spec as source and the source doc as target:
    brs implements mrd  (BRS formalizes market requirements from MRD)
    brs implements brd  (BRS formalizes business objectives from BRD)
    strs implements urd (StRS formalizes user needs from URD)
    strs implements brs (StRS decomposes requirements from BRS — ISO cascade)
  Within ISO cascade, each level decomposes the previous via "implements" relation:
    strs implements brs → syrs implements strs → srs implements syrs
  PRD can substitute for the entire ISO cascade (use "related" relation to link PRD to ISO types).
  Do NOT confuse source documents (mrd/brd/urd) with specification documents (brs/strs/syrs/srs). Sources are informal, discovery-oriented. Specifications are formal, ISO-structured.

VALID STATUS VALUES:
  draft     — default for new documents; work in progress
  accepted  — finalized or approved; set only when the human confirms
  rejected  — superseded, abandoned, or declined; preserves history

CODE REFERENCES (optional):
Documents may reference source code paths using @-notation (e.g., @cmd/sync.go, @internal/config/).
This is optional but encouraged — it helps agents navigate between documentation and code, and enables future staleness detection.
When writing or updating documents, include relevant code paths where they naturally fit (e.g., in "Implementation Notes", "Key files", "Related" sections).

WORKFLOW PROMPTS (when client supports MCP prompts):
  iso_track          — BRS → StRS → SyRS → SRS cascade
  sources_track      — MRD → BRD → URD discovery flow
  product_track      — PRD → plan
  standard_track     — ADR → rule → guide
  architecture_track — ADR → spec → plan
If the user's request matches one of these patterns and the client exposes MCP prompts, prefer suggesting the matching prompt over manual orchestration. Otherwise, follow the cascade manually using create_document and add_relation.

NEVER create documents for: temporary notes, questions, chat summaries, or speculative content without clear value.
ALWAYS use a descriptive slug (lowercase, hyphens only) and a clear human-readable title.`

// buildInstructions returns MCP server instructions with an optional language directive appended.
func buildInstructions(language string) string {
	if language == "" || language == "en" {
		return mcpServerInstructions
	}
	return mcpServerInstructions + fmt.Sprintf(`

LANGUAGE REQUIREMENT:
All document content (title, body text) MUST be written in %q. YAML frontmatter keys and status values remain in English. Slug must still be lowercase ASCII with hyphens.`, language)
}

// NewServer creates a new MCP server with archcore tools registered.
//
// Options can attach the projectroot.Resolution (surfaced via the
// which_project diagnostic tool) and the CLI version. Both are optional —
// callers that only need a working server can call NewServer(baseDir).
func NewServer(baseDir string, opts ...Option) *server.MCPServer {
	cfg := &serverConfig{
		resolution: &projectroot.Resolution{Path: baseDir, Source: projectroot.SourceCwd},
		version:    "dev",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	language := ""
	if settings, err := config.Load(baseDir); err == nil {
		language = settings.Language
	}

	s := server.NewMCPServer(
		"archcore",
		"1.0.0",
		server.WithInstructions(buildInstructions(language)),
		server.WithPromptCapabilities(true),
	)

	s.AddTool(tools.NewInitProjectTool(), tools.HandleInitProject(baseDir))
	s.AddTool(tools.NewListDocumentsTool(), tools.HandleListDocuments(baseDir))
	s.AddTool(tools.NewGetDocumentTool(), tools.HandleGetDocument(baseDir))
	s.AddTool(tools.NewSearchDocumentsTool(), tools.HandleSearchDocuments(baseDir))
	s.AddTool(tools.NewCreateDocumentTool(), tools.HandleCreateDocument(baseDir))
	s.AddTool(tools.NewUpdateDocumentTool(), tools.HandleUpdateDocument(baseDir))
	s.AddTool(tools.NewRemoveDocumentTool(), tools.HandleRemoveDocument(baseDir))
	s.AddTool(tools.NewAddRelationTool(), tools.HandleAddRelation(baseDir))
	s.AddTool(tools.NewRemoveRelationTool(), tools.HandleRemoveRelation(baseDir))
	s.AddTool(tools.NewListRelationsTool(), tools.HandleListRelations(baseDir))
	s.AddTool(tools.NewWhichProjectTool(), tools.HandleWhichProject(cfg.resolution, cfg.version))

	prompts.RegisterAll(s)

	return s
}

// RunStdio starts the MCP server on stdin/stdout.
func RunStdio(baseDir string, opts ...Option) error {
	s := NewServer(baseDir, opts...)
	return server.ServeStdio(s)
}
