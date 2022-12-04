// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package benchrand

import (
	"encoding/binary"
	"math/bits"
	"math/rand"
)

type ChaCha8Source struct {
	// The ChaCha8 state is 16 words: 4 constant, 8 of key, 1 of counter
	// (incremented after each block), and 3 of nonce (here always zero).
	key     [8]uint32
	counter uint32

	// usable is a slice of buf that contains usable key stream. The size of buf
	// depends on how many blocks are computed at a time by keyStream.
	buf    [bufSize]byte
	usable []byte
}

var _ rand.Source64 = &ChaCha8Source{}

func NewChaCha8Source(key []byte) *ChaCha8Source {
	if len(key) != 32 {
		panic("chacha8: wrong key size")
	}

	c := &ChaCha8Source{}
	c.key = [8]uint32{
		binary.LittleEndian.Uint32(key[0:4]),
		binary.LittleEndian.Uint32(key[4:8]),
		binary.LittleEndian.Uint32(key[8:12]),
		binary.LittleEndian.Uint32(key[12:16]),
		binary.LittleEndian.Uint32(key[16:20]),
		binary.LittleEndian.Uint32(key[20:24]),
		binary.LittleEndian.Uint32(key[24:28]),
		binary.LittleEndian.Uint32(key[28:32]),
	}
	return c
}

func (s *ChaCha8Source) Int63() int64 {
	return int64(s.Uint64() & (1<<63 - 1))
}

func (s *ChaCha8Source) Seed(seed int64) {
	panic("unimplemented")
}

// The constant first 4 words of the ChaCha8 state.
const (
	j0 uint32 = 0x61707865 // expa
	j1 uint32 = 0x3320646e // nd 3
	j2 uint32 = 0x79622d32 // 2-by
	j3 uint32 = 0x6b206574 // te k
)

const blockSize = 64

// quarterRound is the core of ChaCha8. It shuffles the bits of 4 state words.
// It's executed 4 times for each of the 8 ChaCha8 rounds, operating on all 16
// words each round, in columnar or diagonal groups of 4 at a time.
func quarterRound(a, b, c, d uint32) (uint32, uint32, uint32, uint32) {
	a += b
	d ^= a
	d = bits.RotateLeft32(d, 16)
	c += d
	b ^= c
	b = bits.RotateLeft32(b, 12)
	a += b
	d ^= a
	d = bits.RotateLeft32(d, 8)
	c += d
	b ^= c
	b = bits.RotateLeft32(b, 7)
	return a, b, c, d
}

func (s *ChaCha8Source) Uint64() uint64 {
	if len(s.usable) < 8 {
		s.keyStream()
		if s.counter == 0 {
			// TODO: handle counter overflow by spilling into nonce or re-keying.
			panic("rand: counter overflow")
		}
		s.usable = s.buf[:]
	}
	out := binary.LittleEndian.Uint64(s.usable)
	s.usable = s.usable[8:]
	return out
}

func (s *ChaCha8Source) genericKeyStream() {
	// To generate each block of key stream, the initial cipher state
	// (represented below) is passed through 8 rounds of shuffling,
	// alternatively applying quarterRounds by columns (like 1, 5, 9, 13)
	// or by diagonals (like 1, 6, 11, 12).
	//
	//      0:cccccccc   1:cccccccc   2:cccccccc   3:cccccccc
	//      4:kkkkkkkk   5:kkkkkkkk   6:kkkkkkkk   7:kkkkkkkk
	//      8:kkkkkkkk   9:kkkkkkkk  10:kkkkkkkk  11:kkkkkkkk
	//     12:bbbbbbbb  13:nnnnnnnn  14:nnnnnnnn  15:nnnnnnnn
	//
	//            c=constant k=key b=blockcount n=nonce
	var (
		x0, x1, x2, x3     = j0, j1, j2, j3
		x4, x5, x6, x7     = s.key[0], s.key[1], s.key[2], s.key[3]
		x8, x9, x10, x11   = s.key[4], s.key[5], s.key[6], s.key[7]
		x12, x13, x14, x15 = s.counter, uint32(0), uint32(0), uint32(0)
	)

	for i := 0; i < 4; i++ {
		// Column round.
		x0, x4, x8, x12 = quarterRound(x0, x4, x8, x12)
		x1, x5, x9, x13 = quarterRound(x1, x5, x9, x13)
		x2, x6, x10, x14 = quarterRound(x2, x6, x10, x14)
		x3, x7, x11, x15 = quarterRound(x3, x7, x11, x15)

		// Diagonal round.
		x0, x5, x10, x15 = quarterRound(x0, x5, x10, x15)
		x1, x6, x11, x12 = quarterRound(x1, x6, x11, x12)
		x2, x7, x8, x13 = quarterRound(x2, x7, x8, x13)
		x3, x4, x9, x14 = quarterRound(x3, x4, x9, x14)
	}

	// Add back the initial state to generate the key stream.
	binary.LittleEndian.PutUint32(s.buf[0:4], x0+j0)
	binary.LittleEndian.PutUint32(s.buf[4:8], x1+j1)
	binary.LittleEndian.PutUint32(s.buf[8:12], x2+j2)
	binary.LittleEndian.PutUint32(s.buf[12:16], x3+j3)
	binary.LittleEndian.PutUint32(s.buf[16:20], x4+s.key[0])
	binary.LittleEndian.PutUint32(s.buf[20:24], x5+s.key[1])
	binary.LittleEndian.PutUint32(s.buf[24:28], x6+s.key[2])
	binary.LittleEndian.PutUint32(s.buf[28:32], x7+s.key[3])
	binary.LittleEndian.PutUint32(s.buf[32:36], x8+s.key[4])
	binary.LittleEndian.PutUint32(s.buf[36:40], x9+s.key[5])
	binary.LittleEndian.PutUint32(s.buf[40:44], x10+s.key[6])
	binary.LittleEndian.PutUint32(s.buf[44:48], x11+s.key[7])
	binary.LittleEndian.PutUint32(s.buf[48:52], x12+s.counter)
	binary.LittleEndian.PutUint32(s.buf[52:56], x13+0)
	binary.LittleEndian.PutUint32(s.buf[56:60], x14+0)
	binary.LittleEndian.PutUint32(s.buf[60:64], x15+0)

	s.counter += 1
}
