// Package nodivision defines an Analyzer that flags division and modulo
// operations, which are not constant-time on most architectures.
package nodivision

import (
	"go/ast"
	"go/token"
	"go/types"
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
  - Operations where both operands are "safe" (public) values

Safe values include:
  - Constants and literals
  - Expressions involving len() (lengths leak via cache side-channels anyway)`

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
				// Skip if both operands are safe (public) values.
				if isSafeExpr(pass, n.X) && isSafeExpr(pass, n.Y) {
					return true
				}
				pass.ReportRangef(n, "use of non-constant-time %s operator", n.Op)
			}
		case *ast.AssignStmt:
			if n.Tok == token.QUO_ASSIGN || n.Tok == token.REM_ASSIGN {
				// Skip if both operands are safe (public) values.
				if len(n.Lhs) == 1 && len(n.Rhs) == 1 &&
					isSafeExpr(pass, n.Lhs[0]) && isSafeExpr(pass, n.Rhs[0]) {
					return true
				}
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

// isSafeExpr reports whether expr is a "safe" (public) value that doesn't
// need constant-time protection. This includes constants, literals, and
// expressions derived from len() since lengths leak via cache side-channels.
func isSafeExpr(pass *analysis.Pass, expr ast.Expr) bool {
	// Constants are safe (evaluated at compile time).
	if pass.TypesInfo.Types[expr].Value != nil {
		return true
	}

	// Check if the expression contains a len() call.
	if containsBuiltinCall(pass, expr, "len") {
		return true
	}

	// For binary expressions, check if both sides are safe.
	if bin, ok := expr.(*ast.BinaryExpr); ok {
		return isSafeExpr(pass, bin.X) && isSafeExpr(pass, bin.Y)
	}

	// For parenthesized expressions, check the inner expression.
	if paren, ok := expr.(*ast.ParenExpr); ok {
		return isSafeExpr(pass, paren.X)
	}

	return false
}

// containsBuiltinCall reports whether expr contains a call to the named builtin.
func containsBuiltinCall(pass *analysis.Pass, expr ast.Expr, name string) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		if obj := pass.TypesInfo.Uses[ident]; obj != nil {
			if builtin, ok := obj.(*types.Builtin); ok && builtin.Name() == name {
				found = true
				return false
			}
		}
		return true
	})
	return found
}
