//+build ignore

package main

import (
	"fmt"
	"testing"
	"time"

	"golang.org/x/crypto/scrypt"
)

func main() {
	for n := uint8(14); n < 22; n++ {
		b := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				scrypt.Key([]byte("password"), []byte("salt"), 1<<n, 8, 1, 32)
			}
		})
		t := b.T / time.Duration(b.N)
		fmt.Printf("N = 2^%d\t%dms\n", n, t/time.Millisecond)
	}
}
