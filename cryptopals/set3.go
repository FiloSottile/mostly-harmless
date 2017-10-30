package cryptopals

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	mathrand "math/rand"
	"time"
	"unicode"
)

func newCBCPaddingOracles(plaintext []byte) (
	encryptMessage func() []byte,
	checkMessagePadding func(message []byte) bool,
) {
	key := make([]byte, 16)
	rand.Read(key)
	b, _ := aes.NewCipher(key)
	encryptMessage = func() []byte {
		iv := make([]byte, 16)
		rand.Read(iv)

		ct := encryptCBC(padPKCS7(plaintext, 16), b, iv)
		return append(iv, ct...)
	}
	checkMessagePadding = func(message []byte) bool {
		iv, msg := message[:16], message[16:]
		res := unpadPKCS7(decryptCBC(msg, b, iv))
		return res != nil
	}
	return
}

func attackCBCPaddingOracle(ct []byte, checkMessagePadding func(ct []byte) bool) []byte {
	findNextByte := func(known, iv, block []byte) []byte {
		if len(block) != 16 || len(iv) != 16 || len(known) >= 16 {
			panic("wrong lengths for findNextByte")
		}
		payload := make([]byte, 32)
		copy(payload[16:], block)
		plaintext := append([]byte{0}, known...)

		for p := 0; p < 256; p++ {
			copy(payload, iv)
			plaintext[0] = byte(p)

			// neuter the plaintext bytes
			for i := range plaintext {
				payload[len(payload)-1-16-i] ^= plaintext[len(plaintext)-1-i]
			}

			// apply valid padding
			for i := range plaintext {
				payload[len(payload)-1-16-i] ^= byte(len(plaintext))
			}

			// check we actually changed something
			if bytes.Equal(payload[:16], iv) {
				continue
			}

			if checkMessagePadding(payload) {
				return plaintext
			}
		}

		// if the only one that works is not changing anything,
		// there's already a padding of len len(plaintext)
		plaintext[0] = byte(len(plaintext))
		for _, c := range plaintext {
			if c != byte(len(plaintext)) {
				// TODO: make test case for this
				plaintext[1] ^= byte(len(plaintext))
				return plaintext[1:] // correct and retry
			}
		}
		return plaintext
	}

	if len(ct)%16 != 0 {
		panic("attackCBCPaddingOracle: invalid ciphertext length")
	}

	var plaintext []byte
	for b := 0; b < len(ct)/16-1; b++ {
		var known []byte
		blockStart := len(ct) - b*16 - 16
		block := ct[blockStart : blockStart+16]
		iv := ct[blockStart-16 : blockStart]
		for len(known) < 16 {
			known = findNextByte(known, iv, block)
			// log.Printf("guess: %x", known)
		}
		plaintext = append(known, plaintext...)
	}

	return plaintext
}

func encryptCTR(src []byte, b cipher.Block, nonce []byte) []byte {
	if len(nonce) >= b.BlockSize() {
		panic("nonce should be shorter than blocksize")
	}
	input, output := make([]byte, b.BlockSize()), make([]byte, b.BlockSize())
	copy(input, nonce)
	var dst []byte
	for i := 0; i < len(src); i += b.BlockSize() {
		b.Encrypt(output, input)
		dst = append(dst, xor(output, src[i:])...)

		j := len(nonce)
		for {
			input[j] += 1
			if input[j] != 0 {
				break
			}
			j++
		}
	}
	return dst
}

var decryptCTR = encryptCTR

func newFixedNonceCTROracle() (encryptMessage func([]byte) []byte) {
	key := make([]byte, 16)
	rand.Read(key)
	nonce := make([]byte, 8)
	rand.Read(nonce)
	b, _ := aes.NewCipher(key)
	return func(msg []byte) []byte {
		return encryptCTR(msg, b, nonce)
	}
}

func findFixedNonceCTRKeystream(ciphertexts [][]byte, corpus map[rune]float64) []byte {
	uppercaseCorpus := make(map[rune]float64)
	for c, s := range corpus {
		if !unicode.IsUpper(c) {
			continue
		}
		uppercaseCorpus[c] = s
	}

	column := make([]byte, len(ciphertexts))
	var maxLen int
	for _, c := range ciphertexts {
		if len(c) > maxLen {
			maxLen = len(c)
		}
	}
	keystream := make([]byte, maxLen)
	for col := 0; col < maxLen; col++ {
		var colLen int
		for _, c := range ciphertexts {
			if col >= len(c) {
				continue
			}
			column[colLen] = c[col]
			colLen++
		}

		c := corpus
		if col == 0 {
			c = uppercaseCorpus
		}
		_, k, _ := findSingleXORKey(column[:colLen], c)
		keystream[col] = k
	}
	return keystream
}

type MT19937 struct {
	index int
	mt    [624]uint32
}

func NewMT19937(seed uint32) *MT19937 {
	m := &MT19937{index: 624}
	for i := range m.mt {
		if i == 0 {
			m.mt[0] = seed
		} else {
			m.mt[i] = 1812433253*(m.mt[i-1]^m.mt[i-1]>>30) + uint32(i)
		}
	}
	return m
}

func (m *MT19937) ExtractNumber() uint32 {
	if m.index >= 624 {
		m.twist()
	}

	y := m.mt[m.index]

	y ^= y >> 11
	y ^= y << 7 & 2636928640
	y ^= y << 15 & 4022730752
	y ^= y >> 18

	m.index++
	return y
}

func (m *MT19937) twist() {
	for i := range m.mt {
		y := (m.mt[i] & 0x80000000) + (m.mt[(i+1)%624] & 0x7fffffff)
		m.mt[i] = m.mt[(i+397)%624] ^ y>>1
		if y%2 != 0 {
			m.mt[i] ^= 0x9908b0df
		}
	}
	m.index = 0
}

func randomNumberFromTimeSeed() uint32 {
	time.Sleep(40 * time.Millisecond)
	time.Sleep(time.Duration(mathrand.Intn(10000)) * time.Millisecond)

	seed := uint32(time.Now().UnixNano() / int64(time.Millisecond))
	n := NewMT19937(seed).ExtractNumber()

	time.Sleep(40 * time.Millisecond)
	time.Sleep(time.Duration(mathrand.Intn(10000)) * time.Millisecond)

	return n
}

func recoverTimeSeed(output uint32) uint32 {
	seed := uint32(time.Now().UnixNano() / int64(time.Millisecond))
	for {
		seed--
		if output == NewMT19937(seed).ExtractNumber() {
			return seed
		}
	}
}

func untemperMT19937(y uint32) uint32 {
	y ^= y >> 18
	y ^= y << 15 & 4022730752
	for i := 0; i < 7; i++ {
		y ^= y << 7 & 2636928640
	}
	y ^= y>>11 ^ y>>(11*2)
	return y
}

func encryptMT19937(src []byte, seed uint16) []byte {
	mt := NewMT19937(uint32(seed))
	keystream := make([]byte, len(src)+3)
	for i := 0; i < len(src); i += 4 {
		x := mt.ExtractNumber()
		keystream[i] = byte(x)
		keystream[i+1] = byte(x >> 8)
		keystream[i+2] = byte(x >> 16)
		keystream[i+3] = byte(x >> 24)
	}
	return xor(src, keystream)
}

func MT19937Oracle(knownPlaintext []byte) []byte {
	msg := make([]byte, 10+mathrand.Intn(90))
	rand.Read(msg)
	msg = append(msg, knownPlaintext...)
	key := make([]byte, 2)
	rand.Read(key)
	return encryptMT19937(msg, uint16(key[0])<<8+uint16(key[1]))
}

func recoverMT19937Key(ct, knownPlaintext []byte) uint16 {
	// air-quotes "key"
	for s := 0; s < 0xffff; s++ {
		if bytes.HasSuffix(encryptMT19937(ct, uint16(s)), knownPlaintext) {
			return uint16(s)
		}
	}
	panic("key not found")
}

func makeRandomTokenWithMT19937() []byte {
	token := make([]byte, 16)
	mt := NewMT19937(uint32(time.Now().Unix()))
	for i := 0; i < len(token); i += 4 {
		x := mt.ExtractNumber()
		token[i] = byte(x)
		token[i+1] = byte(x >> 8)
		token[i+2] = byte(x >> 16)
		token[i+3] = byte(x >> 24)
	}
	return token
}

func detectMT19937Token(token []byte) bool {
	tk := make([]byte, 16)
	for delta := uint32(0); delta < 60*60*24; delta++ {
		mt := NewMT19937(uint32(time.Now().Unix()) - delta)
		for i := 0; i < len(tk); i += 4 {
			x := mt.ExtractNumber()
			tk[i] = byte(x)
			tk[i+1] = byte(x >> 8)
			tk[i+2] = byte(x >> 16)
			tk[i+3] = byte(x >> 24)
		}
		if bytes.Equal(token, tk) {
			return true
		}
	}
	return false
}
