package testsupport

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// corpusTypes spans the accepted and the ignored halves of the code-alignment
// allowlist, so a benchmark over a corpus built here measures the filter rather
// than a corpus that happens to be all one type.
var corpusTypes = []string{
	"rule", "cpat", "adr", "spec", "guide", // ranked by code alignment
	"plan", "idea", "doc", "rfc", "prd", "rnd", "task-type", // not ranked
}

// bodyTarget keeps a generated body large enough that reading and parsing it is
// a real cost, matching the shape of a written-up document.
const bodyTarget = 1500

// BuildCorpus writes n documents under dir/.archcore/ spread across 20
// subdirectories and the type list above, and returns dir.
//
// Every body mentions "src/api/" so code-alignment correlation has something to
// match; the title and tags vary so recap ranking has something to order.
func BuildCorpus(tb testing.TB, dir string, n int) string {
	tb.Helper()
	archcore := filepath.Join(dir, ".archcore")
	for i := range n {
		sub := filepath.Join(archcore, fmt.Sprintf("domain%02d", i%20))
		if err := os.MkdirAll(sub, 0o755); err != nil {
			tb.Fatal(err)
		}
		name := fmt.Sprintf("doc-%05d.%s.md", i, corpusTypes[i%len(corpusTypes)])
		if err := os.WriteFile(filepath.Join(sub, name), []byte(corpusDoc(i)), 0o644); err != nil {
			tb.Fatal(err)
		}
	}
	return dir
}

func corpusDoc(i int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntitle: \"Document %d\"\nstatus: accepted\ntags:\n  - bench\n  - tier-%d\n---\n\n", i, i%5)
	fmt.Fprintf(&b, "Covers src/api/ and internal/service%02d/.\n\n", i%20)
	for b.Len() < bodyTarget {
		fmt.Fprintf(&b, "Paragraph %d describes a constraint that applies to the module.\n", b.Len())
	}
	return b.String()
}
