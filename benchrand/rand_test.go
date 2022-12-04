package benchrand

import (
	"math/rand"
	"testing"
)

func benchmarkUint64(b *testing.B, src rand.Source64) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		src.Uint64()
	}
}

func BenchmarkUint64(b *testing.B) {
	b.Run("rng", func(b *testing.B) { benchmarkUint64(b, rand.NewSource(1).(rand.Source64)) })
	key := make([]byte, 32)
	b.Run("chacha8", func(b *testing.B) { benchmarkUint64(b, NewChaCha8Source(key)) })
}
