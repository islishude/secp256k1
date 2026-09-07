package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestIndirectSecretPaths(t *testing.T) {
	for _, test := range []struct {
		name, body string
		reject     bool
	}{
		{"safe", "func Sign() { helper() }; func helper() {}", false},
		{"wrapper", "func Sign() { helper() }; func helper() { scalarInvVartime() }; func scalarInvVartime() {}", true},
		{"method", "type S struct{}; func (S) InvVartime() {}; func Sign() { var s S; helper(s) }; func helper(s S) { s.InvVartime() }", true},
		{"function value", "func Sign() { f := hidden; f() }; func hidden() { scalarInvVartime() }; func scalarInvVartime() {}", true},
		{"closure", "func Sign() { f := func() { scalarInvVartime() }; f() }; func scalarInvVartime() {}", true},
		{"interface call", "type I interface { Run() }; func Sign() { helper(nil) }; func helper(i I) { i.Run() }", true},
		{"unreachable", "func Sign() {}; func Verify() { scalarInvVartime() }; func scalarInvVartime() {}", false},
		{"cycle", "func Sign() { helper() }; func helper() { Sign() }", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "fixture.go", "package fixture\n"+test.body, 0)
			if err != nil {
				t.Fatal(err)
			}
			graph := map[string]function{}
			if err := collect(fset, "fixture", []*ast.File{file}, nil, graph); err != nil {
				t.Fatal(err)
			}
			err = check(graph, []string{"fixture.Sign"})
			if (err != nil) != test.reject {
				t.Fatalf("check = %v, reject=%v", err, test.reject)
			}
			if err != nil && !strings.Contains(err.Error(), "fixture.Sign") {
				t.Fatal("missing call path")
			}
		})
	}
}
