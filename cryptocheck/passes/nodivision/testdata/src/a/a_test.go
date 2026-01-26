package a

import "testing"

// TestDivision is in a _test.go file, so division is allowed.
func TestDivision(t *testing.T) {
	if 10/2 != 5 {
		t.Error("math is broken")
	}
}

// TestModulo is in a _test.go file, so modulo is allowed.
func TestModulo(t *testing.T) {
	if 10%3 != 1 {
		t.Error("math is broken")
	}
}

func helperInTest(a, b int) int {
	// Even helper functions in test files are allowed.
	return a / b
}
