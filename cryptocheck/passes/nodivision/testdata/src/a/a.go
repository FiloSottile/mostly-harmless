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

	// Variable divided by power of 2 is allowed (compiled to shift).
	v := x + y + a + b + c
	return v / 2
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

// Power of 2 divisors are compiled to shifts/masks.

func powerOfTwoDivSigned(a int) int {
	// Signed division by power of 2 is allowed (compiled to shift with bias).
	x := a / 2
	y := a / 4
	z := a / 64
	return x + y + z
}

func powerOfTwoDivUnsigned(a uint) uint {
	// Unsigned division by power of 2 is allowed (compiled to shift).
	x := a / 2
	y := a / 4
	z := a / 64
	return x + y + z
}

func powerOfTwoModSigned(a int) int {
	// Signed modulo by power of 2 is NOT allowed (only optimized if provably non-negative).
	x := a % 2  // want `use of non-constant-time % operator`
	y := a % 4  // want `use of non-constant-time % operator`
	z := a % 64 // want `use of non-constant-time % operator`
	return x + y + z
}

func powerOfTwoModUnsigned(a uint) uint {
	// Unsigned modulo by power of 2 is allowed (compiled to AND mask).
	x := a % 2
	y := a % 4
	z := a % 64
	return x + y + z
}

func powerOfTwoAssignSigned(a int) int {
	// Signed /= by power of 2 is allowed.
	a /= 2
	// Signed %= by power of 2 is NOT allowed.
	a %= 4 // want `use of non-constant-time %= operator`
	return a
}

func powerOfTwoAssignUnsigned(a uint) uint {
	// Unsigned /= and %= by power of 2 are allowed.
	a /= 2
	a %= 4
	return a
}

func nonPowerOfTwo(a int) int {
	// Non-power-of-2 divisors are not allowed.
	x := a / 3  // want `use of non-constant-time / operator`
	y := a / 10 // want `use of non-constant-time / operator`
	z := a % 7  // want `use of non-constant-time % operator`
	return x + y + z
}

const powerOfTwoConst = 16

func powerOfTwoNamedConst(a int) int {
	// Named constant that is power of 2.
	x := a / powerOfTwoConst
	y := a % powerOfTwoConst // want `use of non-constant-time % operator`
	return x + y
}

func powerOfTwoNamedConstUnsigned(a uint) uint {
	// Named constant that is power of 2 with unsigned type.
	x := a / powerOfTwoConst
	y := a % powerOfTwoConst
	return x + y
}

// Functions marked with //cryptocheck:return-value-is-not-secret return safe values.

// notSecret does blah blah blah
//
//cryptocheck:return-value-is-not-secret
func notSecret[T any](a T) T { return a }

//cryptocheck:return-value-is-not-secret
//go:noinline
func addPublic(a, b int) int { return a + b }

func useNotSecret(secret, public int) int {
	// Both operands must be safe for the operation to be safe.
	// notSecret() on one side is not enough.
	a := notSecret(secret) / public           // want `use of non-constant-time / operator`
	b := secret / notSecret(public)           // want `use of non-constant-time / operator`
	c := notSecret(secret) / notSecret(public) // OK: both sides are safe

	// Without notSecret, it's flagged.
	d := secret / public // want `use of non-constant-time / operator`

	return a + b + c + d
}

func useNotSecretMod(secret int, public uint) int {
	// notSecret() works with modulo too.
	a := notSecret(secret) % notSecret(int(public))
	return a
}

func chainedNotSecret(a, b int) int {
	// notSecret in arithmetic expressions.
	x := (notSecret(a) + 1) / (notSecret(b) + 2)
	return x
}

func useAddPublic(secret, public int) int {
	// addPublic returns a safe value.
	a := addPublic(secret, 0) / addPublic(public, 0) // OK: both sides are safe
	b := addPublic(secret, public) / 4               // OK: safe / power-of-2
	c := addPublic(secret, public) / secret          // want `use of non-constant-time / operator`
	return a + b + c
}

// Range loop index variables are safe because they are already leaked by loop progression.

func rangeIndexDiv(f []byte) int {
	// Range index variable used in division is safe.
	// Using non-power-of-2 divisors to actually test range index detection.
	var sum int
	for i := range f {
		sum += i / 7
		sum += i % 5
	}
	return sum
}

func rangeIndexDivWithSecret(f []byte, secret int) int {
	// Range index is safe, but secret is not.
	var sum int
	for i := range f {
		sum += i / secret // want `use of non-constant-time / operator`
		sum += secret / i // want `use of non-constant-time / operator`
	}
	return sum
}

func rangeIndexBothSafe(f []byte) int {
	// Both i (range index) and len(f) are safe.
	for i := range f {
		_ = i / len(f)
		_ = len(f) / i
	}
	return 0
}

func rangeIndexArithmetic(f []byte) int {
	// Range index in arithmetic expressions is still safe.
	for i := range f {
		_ = (i + 1) / 7
		_ = (i * 2) % 5
	}
	return 0
}

func rangeIndexNotValue(f []byte, secret int) int {
	// The value variable from range is NOT safe, only the index.
	for i, v := range f {
		_ = i / 7            // OK: index is safe
		_ = int(v) / secret  // want `use of non-constant-time / operator`
		_ = secret / int(v)  // want `use of non-constant-time / operator`
	}
	return 0
}

func rangeIndexNested(f [][]byte, secret int) int {
	// Nested range loops - both indices are safe.
	for i := range f {
		for j := range f[i] {
			_ = i / 7
			_ = j / 5
			_ = (i + j) / 3
		}
	}
	return 0
}

func rangeIndexOutsideLoop(f []byte, secret int) int {
	// Using the index variable outside its loop - this reuses the variable
	// but outside the range context, so the check won't apply.
	var i int
	for i = range f {
		_ = i / 7 // OK: inside range loop
	}
	// After the loop, i holds the last index value, but semantically
	// it's still bounded by len(f)-1, however our analysis is syntactic
	// and only looks at enclosing range statements.
	_ = i / secret // want `use of non-constant-time / operator`
	return 0
}
