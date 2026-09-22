package tools

import (
	"archcore-cli/internal/config"
	"archcore-cli/internal/docs"
)

// Seam to internal/docs. The aliases keep the MCP-wire name and the domain name
// both true without a conversion layer.

type (
	// LocalDocument is a document discovered in .archcore/.
	LocalDocument = docs.Document
	// EnrichedDocument extends LocalDocument with relation information.
	EnrichedDocument = docs.EnrichedDocument
	// DocumentRelation represents one side of a relation for enriched output.
	DocumentRelation = docs.DocumentRelation
)

// Classified write-path failures. Handlers map them to their spec-pinned
// per-tool messages via errors.Is.
var (
	errPathReadOnlyGlobal = docs.ErrPathReadOnlyGlobal
	errPathNotDocument    = docs.ErrPathNotDocument
	errPathEscapes        = docs.ErrPathEscapes
)

// scanDocuments discovers all .md files recursively inside .archcore/, including
// declared global sources (read-only).
func scanDocuments(baseDir string) ([]LocalDocument, error) { return docs.Scan(baseDir) }

// scanDocumentsFull mirrors scanDocuments and also populates Content.
func scanDocumentsFull(baseDir string) ([]LocalDocument, error) { return docs.ScanFull(baseDir) }

// readDocumentContent reads a single document fully from a relative path.
func readDocumentContent(baseDir, relPath string) (LocalDocument, error) {
	return docs.ReadDocumentContent(baseDir, relPath)
}

func guardWritablePath(baseDir, relPath string, globals []config.GlobalSource) (string, error) {
	return docs.GuardWritablePath(baseDir, relPath, globals)
}

func validateArchcorePath(relPath string) (string, error) {
	return docs.ValidateArchcorePath(relPath)
}

func validateReadPath(baseDir, relPath string, globals []config.GlobalSource) (string, error) {
	return docs.ValidateReadPath(baseDir, relPath, globals)
}

func annotateSource(doc *LocalDocument, baseDir string, globals []config.GlobalSource) {
	docs.AnnotateSource(doc, baseDir, globals)
}

func normalizeRelPath(p string) string { return docs.NormalizeRelPath(p) }

func writeFileAtomic(absPath string, data []byte) error { return docs.WriteFileAtomic(absPath, data) }

func inspectGlobals(baseDir string) ([]docs.GlobalInspection, error) {
	return docs.InspectGlobals(baseDir)
}
