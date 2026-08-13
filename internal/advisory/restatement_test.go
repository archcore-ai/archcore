package advisory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	archsync "archcore-cli/internal/sync"
	"archcore-cli/templates"
)

// The sentence measured in the plugin corpus: it appeared unchanged in a prd's
// Goals section and in the Acceptance Criteria of the plan that implements it.
const copiedLine = `"Investigate X" through /archcore:plan produces an rnd draft; a fully-specified request asks zero questions.`

// writeManifest puts a relation graph in the store the checks read.
func writeManifest(t *testing.T, base string, rels ...archsync.Relation) {
	t.Helper()
	m := archsync.NewManifest()
	m.Relations = rels
	if err := archsync.SaveManifest(base, m); err != nil {
		t.Fatal(err)
	}
}

// numberedDoc builds a document whose body carries each line as a numbered
// clause — the list form every content contract uses, and the only form this
// check compares.
func numberedDoc(lines ...string) string {
	var b strings.Builder
	b.WriteString("---\ntitle: T\nstatus: draft\n---\n\n## Requirements\n\n")
	for _, l := range lines {
		fmt.Fprintf(&b, "1. %s\n", l)
	}
	return b.String()
}

// bodyOf reads a document from the store and splits it through the same parser
// the hook uses, so a test cannot pass on a shape production never sees.
func bodyOf(t *testing.T, base, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(base, ".archcore", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	_, body, err := templates.SplitDocument(data)
	if err != nil {
		t.Fatalf("SplitDocument(%s) = %v", rel, err)
	}
	return body
}

func TestRestatement(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		upstream string
		written  string
		relType  archsync.RelationType
		wantHit  bool
	}{
		{
			name:     "verbatim copy is reported",
			upstream: numberedDoc(copiedLine),
			written:  numberedDoc(copiedLine),
			relType:  archsync.RelImplements,
			wantHit:  true,
		},
		{
			name:     "copy that picked up a trailing clause is reported",
			upstream: numberedDoc(copiedLine),
			written:  numberedDoc(copiedLine + ` A "market research" request still routes to sources mode.`),
			relType:  archsync.RelImplements,
			wantHit:  true,
		},
		{
			name:     "paraphrase across the boundary is left alone",
			upstream: numberedDoc("The palette is init, plan, document, review; no other visible entries."),
			written:  numberedDoc("Constraint: the visible palette is exactly init, plan, document, review; a palette change requires a superseding ADR."),
			relType:  archsync.RelImplements,
			wantHit:  false,
		},
		{
			name:     "notation differences do not hide a copy",
			upstream: numberedDoc("The user starts an export and continues working in the catalog without waiting."),
			written:  numberedDoc("WHEN the user starts an export, the service MUST let the user continue working in the catalog without waiting."),
			relType:  archsync.RelImplements,
			wantHit:  true,
		},
		{
			name:     "a copied Russian line is compared like any other",
			upstream: numberedDoc("Пользователь запускает экспорт и продолжает работать в каталоге без ожидания."),
			written:  numberedDoc("Пользователь запускает экспорт и продолжает работать в каталоге без ожидания."),
			relType:  archsync.RelImplements,
			wantHit:  true,
		},
		{
			name:     "a related edge carries no content, so an overlap is a topic",
			upstream: numberedDoc(copiedLine),
			written:  numberedDoc(copiedLine),
			relType:  archsync.RelRelated,
			wantHit:  false,
		},
		{
			name:     "a depends_on edge orders two documents without moving text",
			upstream: numberedDoc(copiedLine),
			written:  numberedDoc(copiedLine),
			relType:  archsync.RelDependsOn,
			wantHit:  false,
		},
		{
			name:     "short statements are not compared",
			upstream: numberedDoc("Export runs in the background."),
			written:  numberedDoc("Export runs in the background."),
			relType:  archsync.RelImplements,
			wantHit:  false,
		},
		{
			name:     "unrelated statements are left alone",
			upstream: numberedDoc("The user receives the finished export file by email without returning to the page."),
			written:  numberedDoc("An administrator revokes a session token from the security settings screen."),
			relType:  archsync.RelImplements,
			wantHit:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)
			writeArchcoreDoc(t, base, "up.prd.md", tt.upstream)
			writeArchcoreDoc(t, base, "down.plan.md", tt.written)
			writeManifest(t, base, archsync.Relation{
				Source: "down.plan.md", Target: "up.prd.md", Type: tt.relType,
			})

			got := Restatement(base, ".archcore/down.plan.md", bodyOf(t, base, "down.plan.md"))

			if tt.wantHit && len(got) == 0 {
				t.Fatalf("Restatement() = no findings, want a restatement finding")
			}
			if !tt.wantHit {
				if len(got) > 0 {
					t.Errorf("Restatement() = %q, want no findings", got)
				}
				return
			}
			// The finding names the neighbour in the store's own path form, the
			// one add_relation and get_document accept.
			if !strings.Contains(got[0], ".archcore/up.prd.md") {
				t.Errorf("finding %q does not name the upstream document", got[0])
			}
		})
	}
}

// A statement inside a fenced block is a sample, not a claim. Comparing it
// would report every document that pastes the same snippet.
func TestRestatementSkipsFencedBlocks(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "up.prd.md", numberedDoc(copiedLine))
	writeArchcoreDoc(t, base, "down.plan.md",
		"---\ntitle: T\nstatus: draft\n---\n\n## Requirements\n\n```\n1. "+copiedLine+"\n```\n")
	writeManifest(t, base, archsync.Relation{
		Source: "down.plan.md", Target: "up.prd.md", Type: archsync.RelImplements,
	})

	if got := Restatement(base, ".archcore/down.plan.md", bodyOf(t, base, "down.plan.md")); len(got) > 0 {
		t.Errorf("Restatement() = %q for a copy inside a fence, want no findings", got)
	}
}

// The edge points from the spec to the prd, so writing the prd sees the copy
// only if the check reads incoming edges too.
func TestRestatementReadsIncomingEdges(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "up.prd.md", numberedDoc(copiedLine))
	writeArchcoreDoc(t, base, "down.spec.md", numberedDoc(copiedLine))
	writeManifest(t, base, archsync.Relation{
		Source: "down.spec.md", Target: "up.prd.md", Type: archsync.RelImplements,
	})

	got := Restatement(base, ".archcore/up.prd.md", bodyOf(t, base, "up.prd.md"))
	if len(got) == 0 {
		t.Fatal("Restatement() = no findings when writing the target of an implements edge")
	}
	if !strings.Contains(got[0], ".archcore/down.spec.md") {
		t.Errorf("finding %q does not name the neighbour", got[0])
	}
}

func TestRestatementWithoutRelationsIsSilent(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "up.prd.md", numberedDoc(copiedLine))
	writeArchcoreDoc(t, base, "down.plan.md", numberedDoc(copiedLine))

	if got := Restatement(base, ".archcore/down.plan.md", bodyOf(t, base, "down.plan.md")); len(got) > 0 {
		t.Errorf("Restatement() = %q with no relation graph, want no findings", got)
	}
}

func TestRestatementMissingUpstreamIsSilent(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "down.plan.md", numberedDoc(copiedLine))
	writeManifest(t, base, archsync.Relation{
		Source: "down.plan.md", Target: "gone.prd.md", Type: archsync.RelImplements,
	})

	if got := Restatement(base, ".archcore/down.plan.md", bodyOf(t, base, "down.plan.md")); len(got) > 0 {
		t.Errorf("Restatement() = %q for an unreadable upstream, want no findings", got)
	}
}
