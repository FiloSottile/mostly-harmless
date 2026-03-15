// Package nodivision defines an Analyzer that flags division and modulo
// operations, which are not constant-time on most architectures.
package nodivision

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const Doc = `check for non-constant-time division and modulo operations

Division (/) and modulo (%) operations are not compiled to constant-time
instructions on most architectures. This can lead to timing side-channels
in cryptographic code.

Operations are allowed in:
  - Test files (*_test.go)
  - Functions or methods with a VarTime name suffix (e.g., DivideVarTime)
  - Operations where both operands are "safe" (public) values
  - Division by a constant power of 2 (compiled to shifts)
  - Unsigned modulo by a constant power of 2 (compiled to bitwise AND)

Safe values include:
  - Constants and literals
  - Expressions involving len() (lengths leak via cache side-channels anyway)
  - Return values of functions marked with //cryptovet:return-value-is-not-secret
  - Range loop index variables (already leaked by loop progression)`

var Analyzer = &analysis.Analyzer{
	Name:     "nodivision",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// notSecretPragma is the comment pragma that marks a function's return value as not secret.
const notSecretPragma = "//cryptovet:return-value-is-not-secret"

func run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// First pass: find functions marked with //cryptovet:return-value-is-not-secret.
	notSecretFuncs := findNotSecretFuncs(pass, inspect)

	nodeFilter := []ast.Node{
		(*ast.BinaryExpr)(nil),
		(*ast.AssignStmt)(nil),
		(*ast.RangeStmt)(nil), // Needed in stack for isRangeIndexVar
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
				if isSafeExpr(pass, n.X, notSecretFuncs, stack) && isSafeExpr(pass, n.Y, notSecretFuncs, stack) {
					return true
				}
				// Skip if divisor is a constant power of 2 (compiled to shifts/masks).
				// - Division: always safe (both signed and unsigned)
				// - Modulo: only safe for unsigned (signed requires non-negative proof)
				if isPowerOfTwoConstant(pass, n.Y) {
					if n.Op == token.QUO || isUnsigned(pass, n.X) {
						return true
					}
				}
				pass.ReportRangef(n, "use of non-constant-time %s operator", n.Op)
			}
		case *ast.AssignStmt:
			if n.Tok == token.QUO_ASSIGN || n.Tok == token.REM_ASSIGN {
				// Skip if both operands are safe (public) values.
				if len(n.Lhs) == 1 && len(n.Rhs) == 1 &&
					isSafeExpr(pass, n.Lhs[0], notSecretFuncs, stack) && isSafeExpr(pass, n.Rhs[0], notSecretFuncs, stack) {
					return true
				}
				// Skip if divisor is a constant power of 2.
				if len(n.Rhs) == 1 && isPowerOfTwoConstant(pass, n.Rhs[0]) {
					if n.Tok == token.QUO_ASSIGN || (len(n.Lhs) == 1 && isUnsigned(pass, n.Lhs[0])) {
						return true
					}
				}
				pass.ReportRangef(n, "use of non-constant-time %s operator", n.Tok)
			}
		}

		return true
	})

	return nil, nil
}

// findNotSecretFuncs finds all functions marked with //cryptovet:return-value-is-not-secret.
func findNotSecretFuncs(pass *analysis.Pass, inspect *inspector.Inspector) map[*types.Func]bool {
	result := make(map[*types.Func]bool)

	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Doc == nil {
			return
		}
		// Check if any line in the doc contains the pragma.
		for _, comment := range fn.Doc.List {
			if strings.HasPrefix(comment.Text, notSecretPragma) {
				if obj := pass.TypesInfo.Defs[fn.Name]; obj != nil {
					if funcObj, ok := obj.(*types.Func); ok {
						result[funcObj] = true
					}
				}
				return
			}
		}
	})

	return result
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
// need constant-time protection. This includes constants, literals,
// expressions derived from len(), and range loop index variables
// since lengths leak via cache side-channels.
func isSafeExpr(pass *analysis.Pass, expr ast.Expr, notSecretFuncs map[*types.Func]bool, stack []ast.Node) bool {
	// Constants are safe (evaluated at compile time).
	if pass.TypesInfo.Types[expr].Value != nil {
		return true
	}

	// Check if the expression contains a len() call.
	if containsBuiltinCall(pass, expr, "len") {
		return true
	}

	// Check if the expression is a call to a //cryptovet:return-value-is-not-secret function.
	if isNotSecretCall(pass, expr, notSecretFuncs) {
		return true
	}

	// Check if the expression is a range loop index variable.
	if isRangeIndexVar(pass, expr, stack) {
		return true
	}

	// For binary expressions, check if both sides are safe.
	if bin, ok := expr.(*ast.BinaryExpr); ok {
		return isSafeExpr(pass, bin.X, notSecretFuncs, stack) && isSafeExpr(pass, bin.Y, notSecretFuncs, stack)
	}

	// For parenthesized expressions, check the inner expression.
	if paren, ok := expr.(*ast.ParenExpr); ok {
		return isSafeExpr(pass, paren.X, notSecretFuncs, stack)
	}

	return false
}

// isRangeIndexVar reports whether expr is an identifier that refers to
// the index variable of an enclosing range statement. Range indices are
// already leaked by the loop progression itself, so they are public.
func isRangeIndexVar(pass *analysis.Pass, expr ast.Expr, stack []ast.Node) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}

	// Get the object this identifier refers to.
	obj := pass.TypesInfo.Uses[ident]
	if obj == nil {
		return false
	}

	// Walk the stack looking for enclosing RangeStmt nodes.
	for i := len(stack) - 1; i >= 0; i-- {
		rangeStmt, ok := stack[i].(*ast.RangeStmt)
		if !ok {
			continue
		}
		// Check if the identifier refers to the Key (index) variable.
		if key, ok := rangeStmt.Key.(*ast.Ident); ok {
			// For "for i := range x", the key is defined (Defs).
			// For "for i = range x", the key is used (Uses).
			keyObj := pass.TypesInfo.Defs[key]
			if keyObj == nil {
				keyObj = pass.TypesInfo.Uses[key]
			}
			if keyObj != nil && keyObj == obj {
				return true
			}
		}
	}
	return false
}

// isNotSecretCall reports whether expr is a call to a function marked with
// //cryptovet:return-value-is-not-secret.
func isNotSecretCall(pass *analysis.Pass, expr ast.Expr, notSecretFuncs map[*types.Func]bool) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}

	// Get the callee function object from the call expression.
	callee := typeutil.Callee(pass.TypesInfo, call)
	if callee == nil {
		return false
	}
	funcObj, ok := callee.(*types.Func)
	if !ok {
		return false
	}

	// For generic functions, check the origin (uninstantiated) function.
	if origin := funcObj.Origin(); origin != nil {
		funcObj = origin
	}

	return notSecretFuncs[funcObj]
}

// isPowerOfTwoConstant reports whether expr is a constant that is a power of 2.
func isPowerOfTwoConstant(pass *analysis.Pass, expr ast.Expr) bool {
	tv := pass.TypesInfo.Types[expr]
	if tv.Value == nil {
		return false
	}
	// Get the constant value as uint64.
	val, ok := constantToUint64(tv)
	if !ok || val == 0 {
		return false
	}
	return val&(val-1) == 0
}

// constantToUint64 extracts an unsigned integer value from a constant.
func constantToUint64(tv types.TypeAndValue) (uint64, bool) {
	if tv.Value == nil {
		return 0, false
	}
	switch tv.Value.Kind() {
	case constant.Int:
		if val, ok := constant.Uint64Val(tv.Value); ok {
			return val, true
		}
		// Try as signed and convert if positive.
		if val, ok := constant.Int64Val(tv.Value); ok && val > 0 {
			return uint64(val), true
		}
	}
	return 0, false
}

// isUnsigned reports whether expr has an unsigned integer type.
func isUnsigned(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	basic, ok := t.Underlying().(*types.Basic)
	if !ok {
		return false
	}
	return basic.Info()&types.IsUnsigned != 0
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
