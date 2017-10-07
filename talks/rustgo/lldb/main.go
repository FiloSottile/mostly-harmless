package main

func main() {
	println("Hello, World!")
	a := multiplyByTwo(0x5555555555555555)
	println(a)
}

func multiplyByTwo(a uint64) uint64 {
	var x [128]byte
	for i := uint64(0); i < a; i++ {
		x[a%128] = byte(a)
	}
	return a * 2
}
