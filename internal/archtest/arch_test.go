// Package archtest holds structural rules for the codebase, enforced as tests.
//
// Each rule exists because a mistake was made, or could be copied, once. When a
// review comment or an agent correction describes a pattern that should never
// appear again, add a rule here instead of a sentence in CLAUDE.md. Failure
// messages say how to fix the root cause.
package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/juliosaraiva/log2json"

type sourceFile struct {
	path string
	pkg  string
	fset *token.FileSet
	ast  *ast.File
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test directory")
		}
		dir = parent
	}
}

func productionFiles(t *testing.T) []sourceFile {
	t.Helper()
	root := repoRoot(t)
	var files []sourceFile
	for _, top := range []string{"cmd", "internal"} {
		err := filepath.Walk(filepath.Join(root, top), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			parsed, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			files = append(files, sourceFile{
				path: rel,
				pkg:  filepath.ToSlash(filepath.Dir(rel)),
				fset: fset,
				ast:  parsed,
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(files) == 0 {
		t.Fatal("no production Go files found; the walker is broken")
	}
	return files
}

func imports(f sourceFile) []string {
	out := make([]string, 0, len(f.ast.Imports))
	for _, spec := range f.ast.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err == nil {
			out = append(out, path)
		}
	}
	return out
}

// allowedInternalImports is the dependency direction of the pipeline:
// cmd wires reader -> parser -> emitter, and the three stages stay independent
// except that the emitter consumes parser.Entry.
var allowedInternalImports = map[string][]string{
	"cmd/log2json":     {"internal/emitter", "internal/parser", "internal/reader"},
	"internal/emitter": {"internal/parser"},
	"internal/parser":  {},
	"internal/reader":  {},
}

func TestDependencyDirection(t *testing.T) {
	for _, f := range productionFiles(t) {
		allowed, known := allowedInternalImports[f.pkg]
		if !known {
			t.Errorf("%s: package %q is not in allowedInternalImports. Add it there with the internal packages it may import, so its place in the pipeline is explicit.", f.path, f.pkg)
			continue
		}
		for _, imp := range imports(f) {
			if !strings.HasPrefix(imp, modulePath+"/") {
				continue
			}
			rel := strings.TrimPrefix(imp, modulePath+"/")
			ok := false
			for _, a := range allowed {
				if a == rel {
					ok = true
				}
			}
			if !ok {
				t.Errorf("%s: %s must not import %s. Stages of the pipeline stay independent; move the shared type into the package that owns it or pass the value in from cmd/log2json.", f.path, f.pkg, rel)
			}
		}
	}
}

func TestStdlibOnly(t *testing.T) {
	for _, f := range productionFiles(t) {
		for _, imp := range imports(f) {
			if strings.HasPrefix(imp, modulePath+"/") {
				continue
			}
			first := strings.SplitN(imp, "/", 2)[0]
			if strings.Contains(first, ".") {
				t.Errorf("%s: imports %s. log2json ships as a zero-dependency binary; use the standard library or implement the small piece you need.", f.path, imp)
			}
		}
	}
}

func selectorCalls(node ast.Node, visit func(pkg, name string, pos token.Pos)) {
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		visit(ident.Name, sel.Sel.Name, call.Pos())
		return true
	})
}

func TestParsersAreDeterministic(t *testing.T) {
	for _, f := range productionFiles(t) {
		if f.pkg != "internal/parser" {
			continue
		}
		for _, imp := range imports(f) {
			if imp == "math/rand" || imp == "math/rand/v2" || imp == "os" {
				t.Errorf("%s: internal/parser imports %s. A parser must return the same Entry for the same line; take anything environmental as a constructor argument.", f.path, imp)
			}
		}
		selectorCalls(f.ast, func(pkg, name string, pos token.Pos) {
			if pkg == "time" && (name == "Now" || name == "Since") {
				t.Errorf("%s: internal/parser calls time.%s. A parser must return the same Entry for the same line; ingest-time metadata belongs in the emitter.", f.fset.Position(pos), name)
			}
		})
	}
}

func TestNoRegexCompileOnHotPath(t *testing.T) {
	for _, f := range productionFiles(t) {
		for _, decl := range f.ast.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if fn.Name.Name != "Parse" && fn.Name.Name != "CanParse" && fn.Name.Name != "Emit" {
				continue
			}
			selectorCalls(fn.Body, func(pkg, name string, pos token.Pos) {
				if pkg == "regexp" && (strings.HasPrefix(name, "Compile") || strings.HasPrefix(name, "MustCompile") || strings.HasPrefix(name, "Match")) {
					t.Errorf("%s: regexp.%s inside %s runs once per log line. Compile the pattern in the constructor and store it on the struct.", f.fset.Position(pos), name, fn.Name.Name)
				}
			})
		}
	}
}

func TestLibraryCodeDoesNotExitOrPanic(t *testing.T) {
	for _, f := range productionFiles(t) {
		if !strings.HasPrefix(f.pkg, "internal/") {
			continue
		}
		selectorCalls(f.ast, func(pkg, name string, pos token.Pos) {
			if pkg == "os" && name == "Exit" {
				t.Errorf("%s: os.Exit in library code. Return an error; only cmd/log2json decides the exit code.", f.fset.Position(pos))
			}
			if pkg == "log" && (strings.HasPrefix(name, "Fatal") || strings.HasPrefix(name, "Panic")) {
				t.Errorf("%s: log.%s in library code. Return an error; one bad line must not stop the stream.", f.fset.Position(pos), name)
			}
			if pkg == "fmt" && (name == "Print" || name == "Println" || name == "Printf") {
				t.Errorf("%s: fmt.%s in library code writes to stdout, which is the NDJSON stream. Return the value, or take an io.Writer.", f.fset.Position(pos), name)
			}
		})
		ast.Inspect(f.ast, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
				t.Errorf("%s: panic in library code. Return an error; one bad line must not stop the stream.", f.fset.Position(call.Pos()))
			}
			return true
		})
	}
}
