// Package exitinmain detects os.Exit calls inside func main() with defer.
package exitinmain

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer finds os.Exit calls in package main's main function with defer.
var Analyzer = &analysis.Analyzer{
	Name:     "exitinmain",
	Doc:      "find os.Exit in main()",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}

	insp.WithStack(nodeFilter, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := sel.X.(*ast.Ident)
		if !ok || sel.Sel.Name != "Exit" {
			return true
		}

		obj := pass.TypesInfo.Uses[ident]
		if obj == nil {
			return true
		}
		pkgName, ok := obj.(*types.PkgName)
		if !ok || pkgName.Imported().Path() != "os" {
			return true
		}

		filename := pass.Fset.File(call.Pos()).Name()
		if strings.HasSuffix(filename, "_test.go") {
			return true
		}
		var mainFuncDecl *ast.FuncDecl
		for _, ancestor := range stack {
			if fn, ok := ancestor.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
				mainFuncDecl = fn
				break
			}
		}

		if mainFuncDecl != nil {
			hasDefer := false
			ast.Inspect(mainFuncDecl, func(n ast.Node) bool {
				if _, ok := n.(*ast.DeferStmt); ok {
					hasDefer = true
					return false
				}
				return true
			})

			if !hasDefer {
				return true
			}

			pass.Reportf(call.Pos(), "os.Exit in main!")
		}

		return true
	})

	return nil, nil
}
