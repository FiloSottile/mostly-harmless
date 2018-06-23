//+build ignore

package main

// int my_C_function(int a, int b);
import "C"
import "testing"

func main() {
	a, b := C.int(40), C.int(2)
	c := C.my_C_function(a, b)
	println(a, b, c)
}

//export myGoFunction
func myGoFunction(a, b C.int) C.int {
	return a + b
}

func benchCgoCall(b *testing.B) {
	const x = C.int(2)
	const y = C.int(3)
	for i := 0; i < b.N; i++ {
		C.my_C_function(x, y)
	}
}
