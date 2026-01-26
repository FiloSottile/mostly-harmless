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

// Constant expressions are allowed (computed at compile time).

const (
	numerator   = 100
	denominator = 7
	quotient    = numerator / denominator   // allowed: const definition
	remainder   = numerator % denominator   // allowed: const definition
	chained     = 1000 / 10 / 2             // allowed: const definition
	mixed       = (100 + 50) / 3 % 7        // allowed: const definition
)

func constants() int {
	// Literal expressions are allowed.
	x := 100 / 7
	y := 100 % 7

	// Constant expressions with named constants are allowed.
	a := numerator / denominator
	b := numerator % denominator

	// Mixed constant expression.
	c := (numerator + 50) / denominator

	// But variable expressions are not.
	v := x + y + a + b + c
	return v / 2 // want `use of non-constant-time / operator`
}

func partialConstant(a int) int {
	// One side constant, one side variable - not allowed.
	x := a / 10  // want `use of non-constant-time / operator`
	y := 100 / a // want `use of non-constant-time / operator`
	z := a % 10  // want `use of non-constant-time % operator`
	return x + y + z
}

// len() expressions are safe because lengths leak via cache side-channels.

func lenBothSides(x, y []byte) int {
	// Both sides involve len() - allowed.
	return len(x) / len(y)
}

func lenAndConstant(x []byte) int {
	// len() and constant - both safe, allowed.
	a := len(x) / 8
	b := len(x) % 16
	c := (len(x) + 7) / 8 // len in arithmetic expression
	return a + b + c
}

func lenAndVariable(x []byte, n int) int {
	// len() and variable - variable is not safe, not allowed.
	a := len(x) / n // want `use of non-constant-time / operator`
	b := n / len(x) // want `use of non-constant-time / operator`
	c := len(x) % n // want `use of non-constant-time % operator`
	return a + b + c
}
