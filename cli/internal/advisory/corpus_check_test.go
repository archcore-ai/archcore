package advisory

import (
	"cmp"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckCorpus_SettlesEachKind pins corpus-check.spec: every kind reports its
// seeded defect and stays silent on the look-alike beside it.
func TestCheckCorpus_SettlesEachKind(t *testing.T) {
	t.Parallel()
	base := stalenessRepo(t)
	writeAt(t, base, "src/api/handler.go", "package api\n\n// see checkout-flow.spec and missing-doc.rule\nfunc ServeOrder() {}\n\nconst label = \"Оформить заказ\"\n\nfunc renderCartTotal() {\n\tvalue := computeTotalPrice(order)\n\treturn formatCurrency(value)\n}\n")
	writeAt(t, base, "src/api/data.ts", "const x = data.plan; // Foo.spec.ts; the plan; feed spec §3\n")
	writeAt(t, base, "src/api/README.md", "Contract: `.archcore/audio-gate.spec.md`.\n")
	writeAt(t, base, "src/api/promo.go", "package api\n\n// The contract is promo-banner.spec.\nimport _ \"./promo-gone.spec.ts\"\n")
	writeAt(t, base, "src/app/(shop)/page.tsx", "export default 1\n")
	writeArchcoreDoc(t, base, "promo-banner.spec.md", "---\ntitle: Promo\nstatus: draft\n---\n\nThe banner shows a promo.\n")
	writeArchcoreDoc(t, base, "short.doc.md", "---\ntitle: Short\nstatus: accepted\n---\n\n"+
		"```go\nvalue := computeTotalPrice(order)\nreturn formatCurrency(value)\nfunc renderCartTotal() {\n}\nx++\ny--\nz = 1\n```\n")
	writeArchcoreDoc(t, base, "feed.spec.md", "---\ntitle: Feed\nstatus: draft\n---\n\nThe feed lists cards.\n")
	writeArchcoreDoc(t, base, "audio-gate.spec.md", "---\ntitle: Gate\nstatus: draft\n---\n\nThe gate asks for sign-in.\n")
	writeArchcoreDoc(t, base, "layout.rule.md", "---\ntitle: Layout\nstatus: accepted\n---\n\n"+
		"1. The author MUST edit `src/api/*.py`, `src/*`, and `src/api/*.go`; the screen MUST call `DrawLedger`.\n"+
		"2. The screen MUST NOT call `LegacyDrawCall`.\n3. Errors MUST be named `XxxFailure`.\n")
	writeArchcoreDoc(t, base, "checkout-flow.spec.md", "---\ntitle: Checkout\nstatus: draft\n---\n\n"+
		"The button reads «Оформить заказ» and the banner reads «Входящие».\n"+
		"1. The «Корзина» screen MUST NOT show a «+» button.\n"+
		"`ServeOrder` handles it; `RenderCart` draws it. `isLegacyMode` was removed.\n"+
		"See `src/api/handler.go` and `src/api/gone.go` and `src/api/handler.go:12`.\n"+
		"The page `src/app/(shop)/page.tsx` replaced `src/app/(shop)/cart.tsx`.\n"+
		"The checkout MUST call `src/app/(shop)/pay.tsx`.\n")
	writeArchcoreDoc(t, base, "cart.doc.md", "---\ntitle: Cart\nstatus: accepted\n---\n\n"+
		"```go\nvalue := computeTotalPrice(order)\nreturn formatCurrency(value)\nfunc renderCartTotal() {\n```\n"+
		"The robot reads the cache but never writes it, so a crawler cannot poison an entry for people.\n")
	writeArchcoreDoc(t, base, "cache.rule.md", "---\ntitle: Cache\nstatus: accepted\n---\n\n"+
		"1. The robot reads the cache but never writes it, so a crawler cannot poison an entry for people.\n")
	gitCommit(t, base, "seed")

	got, err := CheckCorpusFor(context.Background(), base, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[CheckKind]string{
		CheckUnresolvedPath: "src/api/gone.go",
		"route-group":       "src/app/(shop)/pay.tsx",
		"sentence-end":      ".archcore/promo-banner.spec.md",
		// The unmatched glob, the loose citation, and the README citation.
		"glob":                 "src/api/*.py",
		"loose":                ".archcore/feed.spec.md",
		"readme":               ".archcore/audio-gate.spec.md",
		CheckLineAnchor:        "src/api/handler.go:12",
		CheckMissingIdentifier: "DrawLedger",
		"identifier":           "RenderCart",
		CheckMissingString:     "Входящие",
		"forbidden-on-screen":  "Корзина",
		CheckDraftCitedByCode:  ".archcore/checkout-flow.spec.md",
		CheckDanglingCodeCite:  "missing-doc.rule",
		CheckDuplicateText:     ".archcore/cache.rule.md",
		CheckRestatedCode:      ".archcore/cart.doc.md",
	}
	seen := map[CheckKind][]string{}
	for _, f := range got.Findings {
		seen[f.Kind] = append(seen[f.Kind], f.Subject+f.Doc)
		switch {
		case f.Kind == CheckMissingIdentifier && f.Subject == "ServeOrder",
			f.Kind == CheckMissingString && f.Subject == "Оформить заказ",
			f.Kind == CheckDanglingCodeCite && f.Subject != "missing-doc.rule",
			f.Kind == CheckUnresolvedPath && (f.Subject == "src/api/handler.go" || f.Subject == "src/app/(shop)/page.tsx"),
			f.Kind == CheckRestatedCode && f.Doc == ".archcore/short.doc.md",
			f.Kind == CheckUnresolvedPath && (f.Subject == "src/api/*.go" || f.Subject == "src/*"),
			f.Kind == CheckDanglingCodeCite && strings.HasPrefix(f.Subject, "the."),
			f.Kind == CheckMissingIdentifier && (f.Subject == "LegacyDrawCall" || f.Subject == "XxxFailure"):
			t.Errorf("look-alike reported: %+v", f)
		case f.Kind == CheckMissingString && f.Subject == "Входящие" && !strings.Contains(f.Line, "banner reads"):
			t.Errorf("a finding must carry its cited line: %+v", f)
		case f.Kind == CheckMissingIdentifier && f.Subject == "isLegacyMode" && !f.Historical,
			f.Kind == CheckUnresolvedPath && f.Subject == "src/app/(shop)/cart.tsx" && !f.Historical:
			t.Errorf("a removal line must be marked historical: %+v", f)
		case f.Kind == CheckUnresolvedPath && f.Subject == "src/app/(shop)/pay.tsx" && f.Historical:
			t.Errorf("a live path claim was marked historical: %+v", f)
		}
	}
	alias := map[CheckKind]CheckKind{"glob": CheckUnresolvedPath, "loose": CheckDraftCitedByCode, "readme": CheckDraftCitedByCode, "identifier": CheckMissingIdentifier, "route-group": CheckUnresolvedPath, "sentence-end": CheckDraftCitedByCode, "forbidden-on-screen": CheckMissingString}
	for kind, subject := range want {
		found := false
		for _, s := range seen[cmp.Or(alias[kind], kind)] {
			found = found || strings.Contains(s, subject)
		}
		if !found {
			t.Errorf("%s: want a finding on %q, got %v", kind, subject, seen[kind])
		}
	}
}

// TestCheckCorpus_AcceptedNormativeLeads pins the order a reader who stops early
// relies on: a live normative claim of an accepted document comes first.
func TestCheckCorpus_AcceptedNormativeLeads(t *testing.T) {
	t.Parallel()
	base := stalenessRepo(t)
	writeAt(t, base, "src/a.go", "package a\n")
	writeArchcoreDoc(t, base, "a-draft.spec.md", "---\ntitle: A\nstatus: draft\n---\n\n1. The screen MUST call `DraftOnlyCall`.\n")
	writeArchcoreDoc(t, base, "b-prose.spec.md", "---\ntitle: B\nstatus: accepted\n---\n\nThe screen calls `ProseOnlyCall`.\n")
	writeArchcoreDoc(t, base, "c-bound.spec.md", "---\ntitle: C\nstatus: accepted\n---\n\nThe screen draws `BoundCall`.\n\n1. The screen MUST call `BoundCall`.\n")
	gitCommit(t, base, "seed")
	got, err := CheckCorpusFor(context.Background(), base, nil)
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, f := range got.Findings {
		if f.Kind == CheckMissingIdentifier {
			order = append(order, f.Subject)
		}
	}
	if strings.Join(order, ",") != "BoundCall,ProseOnlyCall,DraftOnlyCall" {
		t.Errorf("order = %v, want accepted normative, accepted prose, draft", order)
	}
}

// TestCheckCorpus_MovedPathNamesItsTarget pins the evidence of a cited path that
// a commit renamed: the reviewer fixes the citation without a search.
func TestCheckCorpus_MovedPathNamesItsTarget(t *testing.T) {
	t.Parallel()
	base := stalenessRepo(t)
	writeAt(t, base, "src/old_name.go", "package src\n\nfunc Keep() {}\n")
	gitCommit(t, base, "add")
	if err := os.Rename(filepath.Join(base, "src", "old_name.go"), filepath.Join(base, "src", "new_name.go")); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, base, "move")
	writeArchcoreDoc(t, base, "a.spec.md", "---\ntitle: A\nstatus: accepted\n---\n\nSee `src/old_name.go`.\n")
	gitCommit(t, base, "doc")

	got, err := CheckCorpusFor(context.Background(), base, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range got.Findings {
		if f.Kind == CheckUnresolvedPath && f.Subject == "src/old_name.go" {
			if !strings.HasSuffix(f.Evidence, " moved to src/new_name.go") {
				t.Errorf("evidence = %q, want the moved-to path", f.Evidence)
			}
			return
		}
	}
	t.Errorf("no unresolved_path finding on the moved path: %+v", got.Findings)
}

// TestCheckCorpus_OutsideGitSkipsWithReason pins the advisory read of the tracked
// files: outside a work tree, every kind that needs them is skipped with a
// reason, and the others still run.
func TestCheckCorpus_OutsideGitSkipsWithReason(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "a.spec.md", "---\ntitle: A\nstatus: accepted\n---\n\nSee `src/a.go:12`.\n")

	got, err := CheckCorpusFor(context.Background(), base, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []CheckKind{CheckUnresolvedPath, CheckMissingIdentifier, CheckMissingString, CheckDraftCitedByCode, CheckDanglingCodeCite, CheckRestatedCode} {
		if got.Skipped[k] == "" {
			t.Errorf("%s: want a skip reason, got %v", k, got.Skipped)
		}
	}
	if got.Totals[CheckLineAnchor] != 1 {
		t.Errorf("line_anchor must still run outside git: %+v", got)
	}
}
