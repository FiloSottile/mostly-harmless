package main

import "testing"

func TestDivideUp(t *testing.t) {
	if divideUp(10, 2) != 5 {
		t.Fail()
	}
	if divideUp(9, 2) != 5 {
		t.Fail()
	}
	if divideUp(10, 5) != 2 {
		t.Fail()
	}
	if divideUp(8, 5) != 2 {
		t.Fail()
	}
}
