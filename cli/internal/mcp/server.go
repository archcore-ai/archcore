// Package mcp serves the Archcore document tools over the Model Context
// Protocol on stdio. It shields the protocol channel: anything a dependency
// writes to the real stdout would corrupt a JSON-RPC frame, so the descriptor
// is redirected before the server starts.
//
// The comment lives here rather than beside one of the dup2 implementations
// because server.go is the only file in the package without a build tag, so it
// is the only place the doc survives on every GOOS.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/config"
	"archcore-cli/internal/mcp/tools"

	"github.com/mark3labs/mcp-go/server"
)

// hostInstructionsCap is how many characters of the server instructions Claude
// Code passes to the model; Copilot CLI passes none for a server outside its
// allowlist. Everything an agent must know therefore fits here, and the type
// catalog lives in the tool descriptions — context-address-leads-every-channel.adr.
const hostInstructionsCap = 2048

// sourceListCap bounds each source-id list in the notices, so the instructions
// stay under hostInstructionsCap however many sources a project declares.
const sourceListCap = 64

const instructionsLead = "Archcore project context lives in .archcore/ as <slug>.<type>.md files.\n\n" + agents.ContextAddress

const instructionsRules = `

READ: search_documents first; get_document takes a path an Archcore tool or hook returned. Before you call a search empty, read near_misses and retry with fewer words.

WRITE: only through the Archcore tools. Search for a duplicate first. Create as draft; set accepted only after the user confirms. Ask before remove_document. Call init_project only when the user asks for documents and .archcore/ is missing.

TYPES: create_document lists them. rnd closes on a recommendation; research on scope coverage. evidence records one external material. A spec line obligates a component with a modal; a scenario step takes the actor as subject and illustrates an existing spec; a journey is a user path no spec covers yet.

RELATIONS: related, implements, extends, depends_on (structural); supports, contradicts (evidential: material → the claim it backs or disputes); supersedes (temporal: newer → older). scenario depends_on spec; scenario implements journey; journey related prd.`

// buildInstructions returns the MCP server instructions: the context address
// first, then the notices for mounted globals, globals not on disk, and a
// non-English language, then the working rules. The notices sit before the
// rules so a host that cuts the text keeps them.
func buildInstructions(language string, mounted, missing []string) string {
	var b strings.Builder
	b.WriteString(instructionsLead)
	if len(mounted) > 0 {
		// An agent that does not know a global is mounted never queries it
		// (global-recall-guarantees.rfc).
		fmt.Fprintf(&b, "\n\nGLOBAL SOURCES, read-only, never relation endpoints: %s. Read a matching global too; a local document on one topic overrides it. If a retry returns only local rows, add source=\"global\".", joinSourceIDs(mounted))
	}
	if len(missing) > 0 {
		// Without the notice an empty result from an uncloned source reads as
		// silence from the org-wide context (missing-global-degrades-to-local.adr).
		fmt.Fprintf(&b, "\n\nNOT CLONED: %s. An empty result says nothing about them; tell the user when an answer depends on one.", joinSourceIDs(missing))
	}
	if language != "" && language != "en" {
		fmt.Fprintf(&b, "\n\nLANGUAGE: write titles and bodies in %q. Keep in English: frontmatter keys, status values, slugs, \"##\" headings, MUST, MUST NOT, SHOULD, SHOULD NOT, MAY, WHEN, WHILE, IF, THEN, WHERE.", language)
	}
	b.WriteString(instructionsRules)
	return b.String()
}

// joinSourceIDs lists source ids in order until the next one would pass
// sourceListCap, then states how many it left out.
func joinSourceIDs(ids []string) string {
	var b strings.Builder
	for i, id := range ids {
		if i > 0 && b.Len()+len(id)+2 > sourceListCap {
			fmt.Fprintf(&b, " and %d more", len(ids)-i)
			break
		}
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(id)
	}
	return b.String()
}

// ServerOption customizes optional server capabilities.
type ServerOption func(*serverConfig)

type serverConfig struct {
	hostWiring tools.HostWiringFunc
	// pinnedRoot marks a server whose root was stated explicitly, with
	// --project or ARCHCORE_PROJECT_ROOT. Such a server never follows the
	// client's working directory (project-root-resolution.spec §1).
	pinnedRoot bool
	// warnTo receives the root provider's refusal lines. Tests supply a buffer;
	// production leaves it nil, which means stderr.
	warnTo io.Writer
}

// WithPinnedRoot declares that the project root came from --project or
// ARCHCORE_PROJECT_ROOT. The server then serves that root for its whole life
// and asks the client for nothing.
func WithPinnedRoot() ServerOption {
	return func(cfg *serverConfig) { cfg.pinnedRoot = true }
}

// WithRootWarnings redirects the root provider's refusal lines, which default
// to stderr. A test reads them from a buffer; an embedder that already owns a
// log stream sends them there.
func WithRootWarnings(w io.Writer) ServerOption {
	return func(cfg *serverConfig) { cfg.warnTo = w }
}

// WithHostWiring registers the install_host_config tool backed by the given
// executor. The executor is injected from the cmd layer, where the host-wiring
// installers live alongside the CLI commands that share them.
func WithHostWiring(fn tools.HostWiringFunc) ServerOption {
	return func(cfg *serverConfig) { cfg.hostWiring = fn }
}

// BackgroundTask is work RunStdio runs alongside the session, on a goroutine it
// starts once per process. A nil BackgroundTask starts nothing.
//
// It is a parameter of RunStdio rather than a ServerOption on purpose. Only
// RunStdio can start a goroutine, so an option form would be accepted by
// NewServer and then silently dropped — the task would never run, and nothing
// would say so. As a bare func rather than an update-policy type it also keeps
// this package from importing the update stack: serving documents over stdio is
// the whole contract here, and an embedder must not inherit a self-update side
// effect by linking the server. The delay, the policy and the telemetry live in
// the closure the cmd layer builds — mcp-background-update.spec.
//
// A BackgroundTask owns its own stream discipline. RunStdio's shield is not
// guaranteed to outlive it (see RunStdio), so it must write to stderr only.
type BackgroundTask func(context.Context)

// NewServer creates a new MCP server with archcore tools. version is the CLI
// build version threaded from main (never a package-level global).
func NewServer(baseDir, version string, opts ...ServerOption) *server.MCPServer {
	var cfg serverConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	return newServerWithConfig(baseDir, version, cfg)
}

// globalIDsByPresence splits the declared global source ids into the sources on
// disk and the sources not cloned yet. An advisory read (fail-open): only
// config.ErrGlobalMissing moves an id to missing, because checkGlobals already
// refused to start on every fatal state.
func globalIDsByPresence(baseDir string, globals []config.GlobalSource) (mounted, missing []string) {
	for _, gs := range globals {
		dirErr := config.CheckGlobalDir(baseDir, config.ResolveGlobalPath(baseDir, gs.Path))
		if errors.Is(dirErr, config.ErrGlobalMissing) {
			missing = append(missing, gs.ID)
			continue
		}
		mounted = append(mounted, gs.ID)
	}
	return mounted, missing
}

func newServerWithConfig(baseDir, version string, cfg serverConfig) *server.MCPServer {
	language := ""
	if settings, err := config.Load(baseDir); err == nil {
		language = settings.Language
	}
	if version == "" {
		version = "dev"
	}

	// ReadGlobals is fail-open by design here: the paragraph is advisory,
	// and `archcore mcp` separately fails startup on an invalid
	// settings.json (checkGlobals).
	mounted, missing := globalIDsByPresence(baseDir, config.ReadGlobals(baseDir))
	s := server.NewMCPServer(
		"archcore",
		version,
		server.WithInstructions(buildInstructions(language, mounted, missing)),
	)

	// One provider serves every tool: the root is resolved per call, and the
	// decision is shared so a session pays for at most one roots/list query per
	// cache window however many tools it calls (project-root-resolution.spec).
	root := newSessionRootProvider(baseDir, cfg.pinnedRoot, cfg.warnTo)

	s.AddTool(tools.NewInitProjectTool(), tools.HandleInitProject(root))
	s.AddTool(tools.NewListDocumentsTool(), tools.HandleListDocuments(root))
	s.AddTool(tools.NewGetDocumentTool(), tools.HandleGetDocument(root))
	s.AddTool(tools.NewSearchDocumentsTool(), tools.HandleSearchDocuments(root))
	s.AddTool(tools.NewCreateDocumentTool(), tools.HandleCreateDocument(root))
	s.AddTool(tools.NewUpdateDocumentTool(), tools.HandleUpdateDocument(root))
	s.AddTool(tools.NewRemoveDocumentTool(), tools.HandleRemoveDocument(root))
	s.AddTool(tools.NewAddRelationTool(), tools.HandleAddRelation(root))
	s.AddTool(tools.NewRemoveRelationTool(), tools.HandleRemoveRelation(root))
	s.AddTool(tools.NewListRelationsTool(), tools.HandleListRelations(root))

	if cfg.hostWiring != nil {
		s.AddTool(tools.NewInstallHostConfigTool(), tools.HandleInstallHostConfig(root, cfg.hostWiring))
	}

	return s
}

// RunStdio starts the MCP server on stdin/stdout. ctx drives shutdown — the
// caller threads cmd.Context(), which the root command already cancels on
// SIGINT/SIGTERM, so no second signal handler is needed here.
//
// stdout shield: the JSON-RPC stream owns the real stdout, so before
// listening shieldStdout reroutes stdout. Tool executors reuse CLI helpers
// (host-wiring installers, display lines) that print via fmt.Println — with
// the shield those prints become stderr logs instead of corrupting protocol
// frames. On unix the shield works at the file-descriptor level (fd 1 is
// repointed at stderr; the protocol stream lives on a private CLOEXEC fd), so
// even raw writes and child processes cannot reach the protocol. On Windows
// the shield is Go-level only — see stdio_shield_windows.go for the residual
// invariant. The swap is process-global: RunStdio must stay the sole purpose
// of its process.
//
// background task: background carries work that is not the server's — the
// update trigger the cmd layer wires. Its placement in this function is the
// contract, not an implementation detail: mcp-background-update.spec §2 puts it
// after the shield and before Listen.
func RunStdio(ctx context.Context, baseDir, version string, background BackgroundTask, opts ...ServerOption) error {
	s := NewServer(baseDir, version, opts...)

	protocolOut, restore := shieldStdout()
	defer restore()

	// The task's context ends when this function does. Listen returns on stdin
	// EOF as well as on cancellation, and EOF is how a host normally ends a
	// session — without this the task would keep running against a context that
	// still says "live" after restore() closed the protocol stream. `archcore
	// mcp` exits immediately either way, so this is for every other caller: the
	// in-process tests and any embedder.
	taskCtx, cancelTask := context.WithCancel(ctx)
	defer cancelTask()

	// Nothing joins this goroutine, deliberately: a session that waited would
	// stall its host on a download, and an attempt killed mid-flight leaves the
	// running binary intact anyway — mcp-background-update.spec §10. Cancelling
	// is not joining: it signals, it does not wait.
	//
	// The unwaited goroutine outliving the deferred restore() is the cost. Once
	// restore() fires fd 1 is the host's protocol channel again, so the task
	// cannot lean on the shield to catch a stray print; it writes to stderr
	// only, which the shield never moves — mcp-background-update.spec §6.
	if background != nil {
		go background(taskCtx)
	}

	return server.NewStdioServer(s).Listen(ctx, os.Stdin, protocolOut)
}
