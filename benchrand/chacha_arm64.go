// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package benchrand

const bufSize = 256

//go:noescape
func xorKeyStreamVX(dst, src []byte, key *[8]uint32, nonce *[3]uint32, counter *uint32)

func (s *ChaCha8Source) keyStream() {
	var zero [bufSize]byte
	xorKeyStreamVX(zero[:], s.buf[:], &s.key, &[3]uint32{}, &s.counter)
	s.counter += 4
}
