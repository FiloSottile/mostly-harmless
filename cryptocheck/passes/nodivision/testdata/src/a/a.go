// Package a is test input for the nodivision analyzer.
package a

func Divide(a, b int) int {
	return a / b // want `use of non-constant-time / operator`
}

func Modulo(a, b int) int {
	return a % b // want `use of non-constant-time % operator`
}

func DivideAssign(a, b int) int {
	a /= b // want `use of non-constant-time /= operator`
	return a
}

func ModuloAssign(a, b int) int {
	a %= b // want `use of non-constant-time %= operator`
	return a
}

func Multiple(a, b int) int {
	x := a / b // want `use of non-constant-time / operator`
	y := a % b // want `use of non-constant-time % operator`
	return x + y
}

// DivideVarTime is allowed because its name ends with VarTime.
func DivideVarTime(a, b int) int {
	return a / b
}

// ModuloVarTime is allowed because its name ends with VarTime.
func ModuloVarTime(a, b int) int {
	return a % b
}

// ShortExpVarTime is allowed.
func ShortExpVarTime(a, b int) int {
	a /= b
	a %= b
	return a / b
}

type T struct{}

// DivideVarTime is allowed because its name ends with VarTime.
func (T) DivideVarTime(a, b int) int {
	return a / b
}

// Divide is not allowed even on a method.
func (T) Divide(a, b int) int {
	return a / b // want `use of non-constant-time / operator`
}

func nested(a, b int) int {
	// Nested function literal inside a non-VarTime function is flagged.
	f := func() int {
		return a / b // want `use of non-constant-time / operator`
	}
	return f()
}

// NestedInVarTime shows that nested function literals in VarTime functions
// are allowed because the enclosing FuncDecl has VarTime suffix.
func NestedInVarTime(a, b int) int {
	f := func() int {
		return a / b
	}
	return f()
}
