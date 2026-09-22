package update

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The unattended policy lives in this package, and it is the one caller in the
// binary that runs with nobody watching. Every plugin verb is a mutating
// subprocess on a host CLI — `claude plugin install`, `codex plugin marketplace
// upgrade` — so a background replacement that could reach one would run those
// commands on a machine whose owner is not at the keyboard.
//
// updating-the-plugin.spec makes manual `archcore update` the only caller of
// the plugin step, and plugin-delivery.spec states the same as a Constraint:
// the unattended update policy and the MCP trigger MUST NOT reach that surface.
// Nothing in the compiler defends it. `cmd` legitimately imports both packages,
// so a convenience import moved down here — or into any package this one
// already uses — would restore the reach with every test still green, and the
// resulting subprocesses would only ever be seen by the user they surprised.
//
// The guard therefore reads the source rather than running it. That is also the
// only way to prove a negative here: no test can exercise "the policy did not
// start a host command" for the refactor that has not been written yet.
//
// It mirrors TestPackage_DoesNotLinkTheUpdateStack in
// @internal/mcp/server_contract_test.go, which pins the same kind of boundary
// from the other side. The walker is duplicated rather than shared: a test
// helper exported from one package into another is a link of its own, and
// these two guards exist precisely to keep such links from forming.
//
// Test files are excluded on purpose — they are not linked into the binary.
func TestPackage_DoesNotLinkThePluginSurface(t *testing.T) {
	t.Parallel()

	root, module := updateModuleRoot(t)
	self := module + "/internal/update"
	banned := module + "/internal/plugin"

	// Breadth-first over first-party imports only. A package in another module
	// cannot import back into this one, so stopping at the module boundary still
	// reaches every package that could pull the plugin surface in.
	importedBy := map[string]string{self: ""}
	queue := []string{self}
	reached := 0
	for len(queue) > 0 {
		pkg := queue[0]
		queue = queue[1:]
		for _, imp := range updateFirstPartyImports(t, root, module, pkg) {
			if _, seen := importedBy[imp]; seen {
				continue
			}
			importedBy[imp] = pkg
			reached++
			if imp == banned || strings.HasPrefix(imp, banned+"/") {
				t.Errorf("%s links %s via %s — the unattended policy would reach a mutating host command; "+
					"manual `archcore update` is the only caller the plugin step has (updating-the-plugin.spec)",
					self, imp, updateImportChain(importedBy, imp))
			}
			queue = append(queue, imp)
		}
	}

	// A walk that found nothing would pass this test on an empty graph — a
	// mistyped module path, a moved package, a parser that silently read no
	// files. This package really does import first-party code, so the traversal
	// has to have visited some.
	if reached == 0 {
		t.Fatalf("the walk reached no first-party package from %s, so it proves nothing", self)
	}
}

// updateImportChain renders how the traversal reached pkg, so a failure names
// the package that actually added the dependency rather than only its endpoint.
func updateImportChain(importedBy map[string]string, pkg string) string {
	chain := []string{pkg}
	for at := importedBy[pkg]; at != ""; at = importedBy[at] {
		chain = append([]string{at}, chain...)
	}
	return strings.Join(chain, " → ")
}

// updateFirstPartyImports returns the module-local packages pkg imports,
// ignoring test files. Build tags are ignored too: an import that only exists
// on one GOOS is still a link on that GOOS, and CI builds only one of them.
func updateFirstPartyImports(t *testing.T, root, module, pkg string) []string {
	t.Helper()

	dir := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(pkg, module+"/")))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read package %s: %v", pkg, err)
	}

	var imports []string
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", filepath.Join(pkg, name), err)
		}
		for _, spec := range file.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			if strings.HasPrefix(path, module+"/") {
				imports = append(imports, path)
			}
		}
	}
	return imports
}

// updateModuleRoot walks up from the package directory to the go.mod that
// declares this module, and returns the directory and the module path.
func updateModuleRoot(t *testing.T) (dir, module string) {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil {
			for line := range strings.SplitSeq(string(data), "\n") {
				if path, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
					return dir, strings.TrimSpace(path)
				}
			}
			t.Fatalf("no module line in %s/go.mod", dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the package directory")
		}
		dir = parent
	}
}
