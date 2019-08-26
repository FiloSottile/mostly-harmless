// Package cryptosource provides a math/rand Source that draws
// random numbers from crypto/rand.
package cryptosource

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"io"
	"math/rand"
)

// New returns a math/rand.Source64 that generates random numbers
// by reading bytes from crypto/rand.Reader.
func New() rand.Source {
	return NewFromReader(cryptorand.Reader)
}

type source struct {
	r io.Reader
}

var _ rand.Source64 = source{}

// NewFromReader returns a math/rand.Source64 that generates random
// numbers by reading bytes from the io.Reader r.
func NewFromReader(r io.Reader) rand.Source {
	return source{r}
}

const mask63Bits = 1<<63 - 1

func (s source) Int63() int64 {
	return int64(s.Uint64() & mask63Bits)
}

func (s source) Seed(seed int64) {
	panic("cryptosource can't be seeded")
}

func (s source) Uint64() uint64 {
	var buf [8]byte
	if _, err := io.ReadFull(s.r, buf[:]); err != nil {
		panic("cryptosource randomness read error: " + err.Error())
	}
	return binary.LittleEndian.Uint64(buf[:])
}
