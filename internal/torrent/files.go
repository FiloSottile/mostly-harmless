package torrent

import (
	"crypto/sha1"
	"hash"
)

type PieceHash struct {
	size   int
	idx    int
	h      hash.Hash
	pieces []byte
}

func NewPieceHash(size int) *PieceHash {
	return &PieceHash{size: size, h: sha1.New()}
}

func (ph *PieceHash) Write(b []byte) (int, error) {
	in := len(b)
	for len(b) > 0 {
		n := min(len(b), ph.size-ph.idx)
		ph.h.Write(b[:n])
		b = b[n:]
		ph.idx += n
		if ph.idx == ph.size {
			ph.pieces = ph.h.Sum(ph.pieces)
			ph.h.Reset()
			ph.idx = 0
		}
	}
	return in, nil
}

func (ph *PieceHash) Pieces() []byte {
	if ph.idx > 0 {
		return ph.h.Sum(ph.pieces[:len(ph.pieces):len(ph.pieces)])
	}
	return ph.pieces
}
