package cryptosource_test

import (
	"math/rand"
	"testing"

	"filippo.io/mostly-harmless/cryptosource"
)

func TestInt63(t *testing.T) {
	source := cryptosource.New()
	for i := 0; i < 1000; i++ {
		if n := source.Int63(); n < 0 {
			t.Error(n)
		}
	}
}

func TestUint64(t *testing.T) {
	source := cryptosource.New()
	seen := make(map[uint64]struct{})
	for i := 0; i < 1000; i++ {
		n := source.(rand.Source64).Uint64()
		if _, ok := seen[n]; ok {
			t.Error("seen number again:", n)
		}
		seen[n] = struct{}{}
	}
}

func BenchmarkInt63(b *testing.B) {
	r := rand.New(cryptosource.New())
	for n := b.N; n > 0; n-- {
		r.Int63()
	}
}
