package cryptosource

import (
	cryptorand "crypto/rand"
	"io"
	"math/rand"
)

func New() rand.Source {
	return NewFromReader(cryptorand.Reader)
}

type source struct {
	r io.Reader
}

var _ rand.Source64 = source{}

func NewFromReader(r io.Reader) rand.Source {
	return source{r}
}

func (s source) Int63() int64 {
	return int64(s.Uint64() & (1<<63 - 1))
}

func (s source) Seed(seed int64) {
	panic("cryptosource can't be seeded")
}

func (s source) Uint64() uint64 {
	buf := make([]byte, 8)
	if _, err := s.r.Read(buf); err != nil {
		panic("cryptosource read error: " + err.Error())
	}
	return uint64(buf[0])<<56 | uint64(buf[1])<<48 |
		uint64(buf[2])<<40 | uint64(buf[3])<<32 |
		uint64(buf[4])<<24 | uint64(buf[5])<<16 |
		uint64(buf[6])<<8 | uint64(buf[7])
}
