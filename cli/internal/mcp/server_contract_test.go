package mcp

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The two guards in this file read the package's source rather than run it.
// Both pin properties that are decided at build time and are invisible at run
// time: the order of two statements whose observable effects the scheduler is
// free to reorder, and an import nothing would ever fail on.

// mcp-background-update.spec §2 fixes an order inside RunStdio — the task
// starts after the stdout shield and before Listen. Running the code cannot
// hold it. A goroutine launched a line too early still, in practice, first runs
// when the launching goroutine blocks, which is past the swap either way, so
// the fd-identity assertion in background_task_test.go passes on both orders
// (confirmed by moving the statement and watching the suite stay green). The
// window the wrong order opens is narrow and real — a second P can pick the
// goroutine up mid-swap, and the task's first write then lands on the host's
// protocol stream — so the order is pinned where it is decidable.
func TestRunStdio_StartsTheBackgroundTaskAfterTheShieldAndBeforeListen(t *testing.T) {
	t.Parallel()

	fn := parseFuncDecl(t, "server.go", "RunStdio")
	shield := firstCallPos(fn, "shieldStdout")
	listen := firstCallPos(fn, "Listen")
	starts := goStmtPositions(fn)

	// Each of these names a piece of RunStdio directly, so a rename must fail
	// here loudly rather than leave the guard passing on nothing.
	if shield == token.NoPos {
		t.Fatal("RunStdio no longer calls shieldStdout; this guard names it and must move with the code")
	}
	if listen == token.NoPos {
		t.Fatal("RunStdio no longer calls Listen; this guard names it and must move with the code")
	}
	// One session, one attempt: a second go statement here would start a second
	// trigger per process — mcp-background-update.spec §11.
	if len(starts) != 1 {
		t.Fatalf("RunStdio has %d go statements, want exactly 1 (the background task)", len(starts))
	}

	if starts[0] < shield {
		t.Error("the background task starts before the stdout shield: a write of its own can reach the host's protocol stream — mcp-background-update.spec §2")
	}
	if starts[0] > listen {
		t.Error("the background task starts after Listen: it would begin only once the session ends — mcp-background-update.spec §2")
	}
}

// This package must not link the update stack. That constraint is the whole
// reason WithBackgroundTask takes an opaque func: an embedder linking the
// server gets a document server, never a binary that replaces itself, and the
// delay, the policy and the telemetry stay in the closure the cmd layer builds
// — mcp-background-update.spec. Nothing in the compiler defends it: one
// convenience import here or in a package this one already uses restores the
// dependency with every test still green.
//
// Test files are excluded on purpose — they are not linked into anything an
// embedder builds.
func TestPackage_DoesNotLinkTheUpdateStack(t *testing.T) {
	t.Parallel()

	root, module := moduleRoot(t)
	self := module + "/internal/mcp"
	banned := []string{module + "/internal/update", module + "/internal/telemetry"}

	// Breadth-first over first-party imports only. A package in another module
	// cannot import back into this one, so stopping at the module boundary
	// still reaches every package that could pull the update stack in.
	importedBy := map[string]string{self: ""}
	queue := []string{self}
	reached := 0
	for len(queue) > 0 {
		pkg := queue[0]
		queue = queue[1:]
		for _, imp := range firstPartyImports(t, root, module, pkg) {
			if _, seen := importedBy[imp]; seen {
				continue
			}
			reached++
			importedBy[imp] = pkg
			for _, b := range banned {
				if imp == b || strings.HasPrefix(imp, b+"/") {
					t.Errorf("%s links %s via %s", self, imp, importChain(importedBy, imp))
				}
			}
			queue = append(queue, imp)
		}
	}

	// A walk that found nothing would pass this test on an empty graph — a
	// mistyped module path, a moved package, a parser that silently read no
	// files. This package really does import first-party code, so the traversal
	// has to have visited some. Its twin in internal/update carries the same
	// guard for the same reason.
	if reached == 0 {
		t.Fatalf("the walk reached no first-party package from %s, so it proves nothing", self)
	}
}

// importChain renders how the traversal reached pkg, so a failure names the
// package that actually added the dependency rather than only its endpoint.
func importChain(importedBy map[string]string, pkg string) string {
	chain := []string{pkg}
	for at := importedBy[pkg]; at != ""; at = importedBy[at] {
		chain = append([]string{at}, chain...)
	}
	return strings.Join(chain, " → ")
}

// firstPartyImports returns the module-local packages pkg imports, ignoring
// test files. Build tags are ignored too: an import that only exists on one
// GOOS is still a link on that GOOS.
func firstPartyImports(t *testing.T, root, module, pkg string) []string {
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

// moduleRoot walks up from the package directory to the go.mod that declares
// this module, and returns the directory and the module path.
func moduleRoot(t *testing.T) (dir, module string) {
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

// parseFuncDecl returns the named top-level function from a file in this
// package's directory.
func parseFuncDecl(t *testing.T, file, name string) *ast.FuncDecl {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	for _, decl := range parsed.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == name && fn.Recv == nil {
			return fn
		}
	}
	t.Fatalf("no func %s in %s", name, file)
	return nil
}

// firstCallPos returns where fn first calls name, as a plain function or as a
// selector (x.name(…)). token.NoPos means it does not call it at all.
func firstCallPos(fn *ast.FuncDecl, name string) token.Pos {
	pos := token.NoPos
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || pos != token.NoPos {
			return pos == token.NoPos
		}
		switch called := call.Fun.(type) {
		case *ast.Ident:
			if called.Name == name {
				pos = call.Pos()
			}
		case *ast.SelectorExpr:
			if called.Sel.Name == name {
				pos = call.Pos()
			}
		}
		return true
	})
	return pos
}

// goStmtPositions returns where fn starts each of its goroutines, in source
// order.
func goStmtPositions(fn *ast.FuncDecl) []token.Pos {
	var out []token.Pos
	ast.Inspect(fn, func(n ast.Node) bool {
		if stmt, ok := n.(*ast.GoStmt); ok {
			out = append(out, stmt.Pos())
		}
		return true
	})
	return out
}
