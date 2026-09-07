// Command checkct rejects repository variable-time functions reachable from
// secret entry points. It is a call-boundary guard, not a constant-time proof.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

const modulePath = "github.com/islishude/secp256k1"

type listedPackage struct {
	ImportPath, Dir, Export string
	GoFiles, CgoFiles       []string
}

type function struct {
	edges     []string
	dynamic   bool
	forbidden bool
}

func main() {
	tags := flag.String("tags", "", "build tags to audit")
	flag.Parse()
	if err := run(*tags); err != nil {
		log.Fatal(err)
	}
}

func run(tags string) error {
	// go list applies the current GOARCH and GOFLAGS/build tags before analysis.
	args := []string{"list", "-deps", "-export", "-json"}
	if tags != "" {
		args = append(args, "-tags="+tags)
	}
	args = append(args, ".")
	cmd := exec.Command("go", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("list packages: %w: %s", err, &stderr)
	}
	var packages []listedPackage
	exports := map[string]string{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var p listedPackage
		if err := decoder.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		packages = append(packages, p)
		exports[p.ImportPath] = p.Export
	}
	fset := token.NewFileSet()
	imp := importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) { return os.Open(exports[path]) })
	graph := map[string]function{}
	for _, p := range packages {
		if p.ImportPath != modulePath && !strings.HasPrefix(p.ImportPath, modulePath+"/internal/") {
			continue
		}
		var files []*ast.File
		for _, name := range append(p.GoFiles, p.CgoFiles...) {
			file, err := parser.ParseFile(fset, filepath.Join(p.Dir, name), nil, 0)
			if err != nil {
				return err
			}
			files = append(files, file)
		}
		if err := collect(fset, p.ImportPath, files, imp, graph); err != nil {
			return err
		}
	}
	roots := []string{
		modulePath + ".PrivateKey.SignDigest", modulePath + ".PrivateKey.SignRecoverableDigest",
		modulePath + ".PrivateKey.PublicKey", modulePath + ".GeneratePrivateKey", modulePath + ".ParsePrivateKey",
	}
	return check(graph, roots)
}

func functionID(fn *types.Func) string {
	prefix := fn.Pkg().Path() + "."
	sig := fn.Type().(*types.Signature)
	if recv := sig.Recv(); recv != nil {
		typ := recv.Type()
		if ptr, ok := typ.(*types.Pointer); ok {
			typ = ptr.Elem()
		}
		if named, ok := typ.(*types.Named); ok {
			prefix += named.Obj().Name() + "."
		}
	}
	return prefix + fn.Name()
}

func collect(fset *token.FileSet, path string, files []*ast.File, imp types.Importer, graph map[string]function) error {
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Types: map[ast.Expr]types.TypeAndValue{}}
	conf := types.Config{Importer: imp}
	if _, err := conf.Check(path, fset, files, info); err != nil {
		return err
	}
	for _, file := range files {
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			fn := info.Defs[fd.Name].(*types.Func)
			node := function{forbidden: strings.Contains(strings.ToLower(fn.Name()), "vartime")}
			if fd.Body != nil {
				ast.Inspect(fd.Body, func(n ast.Node) bool {
					// Function references, including method values and references inside
					// closures, conservatively count as edges even before invocation.
					if id, ok := n.(*ast.Ident); ok {
						if called, ok := info.Uses[id].(*types.Func); ok {
							node.edges = append(node.edges, functionID(called))
						}
					}
					if call, ok := n.(*ast.CallExpr); ok && !resolvedCall(call.Fun, info) {
						node.dynamic = true
					}
					return true
				})
			}
			slices.Sort(node.edges)
			node.edges = slices.Compact(node.edges)
			graph[functionID(fn)] = node
		}
	}
	return nil
}

func resolvedCall(expr ast.Expr, info *types.Info) bool {
	if info.Types[expr].IsType() {
		return true
	} // Conversion, not a function call.
	var obj types.Object
	switch e := expr.(type) {
	case *ast.Ident:
		obj = info.Uses[e]
	case *ast.SelectorExpr:
		if typ := info.TypeOf(e.X); typ != nil {
			if _, isInterface := typ.Underlying().(*types.Interface); isInterface {
				return false
			}
		}
		obj = info.Uses[e.Sel]
	case *ast.FuncLit:
		return true // Body references are visited by collect.
	case *ast.ParenExpr:
		return resolvedCall(e.X, info)
	}
	switch obj.(type) {
	case *types.Func, *types.Builtin:
		return true
	}
	return false
}

func check(graph map[string]function, roots []string) error {
	visited := map[string]bool{}
	var visit func(string, []string) error
	visit = func(id string, path []string) error {
		if visited[id] {
			return nil
		}
		visited[id] = true
		node, ok := graph[id]
		if !ok {
			return nil
		} // External libraries and assembly need separate review.
		path = append(slices.Clone(path), id)
		if node.forbidden {
			return fmt.Errorf("variable-time secret path: %s", strings.Join(path, " -> "))
		}
		if node.dynamic {
			return fmt.Errorf("unresolved indirect call in secret path: %s", strings.Join(path, " -> "))
		}
		for _, edge := range node.edges {
			if err := visit(edge, path); err != nil {
				return err
			}
		}
		return nil
	}
	for _, root := range roots {
		if _, ok := graph[root]; !ok {
			return fmt.Errorf("missing secret entry point %s", root)
		}
		if err := visit(root, nil); err != nil {
			return err
		}
	}
	return nil
}
