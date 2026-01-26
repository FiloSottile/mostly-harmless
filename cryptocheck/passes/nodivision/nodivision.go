// Package nodivision defines an Analyzer that flags division and modulo
// operations, which are not constant-time on most architectures.
package nodivision

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `check for non-constant-time division and modulo operations

Division (/) and modulo (%) operations are not compiled to constant-time
instructions on most architectures. This can lead to timing side-channels
in cryptographic code.

Operations are allowed in:
  - Test files (*_test.go)
  - Functions or methods with a VarTime name suffix (e.g., DivideVarTime)
  - Constant expressions (evaluated at compile time)`

var Analyzer = &analysis.Analyzer{
	Name:     "nodivision",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.BinaryExpr)(nil),
		(*ast.AssignStmt)(nil),
	}

	inspect.WithStack(nodeFilter, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		// Check if we're in a test file.
		filename := pass.Fset.Position(n.Pos()).Filename
		if strings.HasSuffix(filename, "_test.go") {
			return true
		}

		// Check if we're in a VarTime function.
		if inVarTimeFunc(stack) {
			return true
		}

		switch n := n.(type) {
		case *ast.BinaryExpr:
			if n.Op == token.QUO || n.Op == token.REM {
				// Skip constant expressions (evaluated at compile time).
				if pass.TypesInfo.Types[n].Value != nil {
					return true
				}
				pass.ReportRangef(n, "use of non-constant-time %s operator", n.Op)
			}
		case *ast.AssignStmt:
			if n.Tok == token.QUO_ASSIGN || n.Tok == token.REM_ASSIGN {
				pass.ReportRangef(n, "use of non-constant-time %s operator", n.Tok)
			}
		}

		return true
	})

	return nil, nil
}

// inVarTimeFunc reports whether the stack contains an enclosing function
// or method whose name ends with "VarTime".
func inVarTimeFunc(stack []ast.Node) bool {
	for i := len(stack) - 1; i >= 0; i-- {
		if fn, ok := stack[i].(*ast.FuncDecl); ok {
			return strings.HasSuffix(fn.Name.Name, "VarTime")
		}
	}
	return false
}
