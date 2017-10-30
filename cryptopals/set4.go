package cryptopals

import (
	"bytes"
	"crypto/aes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"regexp"
	"strings"
	"time"
)

func newCTREditOracles(plaintext []byte) (
	ciphertext []byte,
	edit func(ciphertext []byte, offset int, newText []byte) []byte,
) {
	key := make([]byte, 16)
	rand.Read(key)
	b, _ := aes.NewCipher(key)

	iv := make([]byte, 8)
	rand.Read(iv)

	ct := encryptCTR(plaintext, b, iv)
	ciphertext = append(iv, ct...)

	edit = func(ciphertext []byte, offset int, newText []byte) []byte {
		iv, msg := ciphertext[:8], ciphertext[8:]
		plaintext := decryptCTR(msg, b, iv)

		copy(plaintext[offset:], newText)
		ct := encryptCTR(plaintext, b, iv)

		var res []byte
		res = append(res, iv...)
		res = append(res, ct...)
		return res
	}
	return
}

func attackCTREditOracle(ciphertext []byte,
	edit func(ciphertext []byte, offset int, newText []byte) []byte) []byte {
	var plaintext []byte
	for offset := 8; offset < len(ciphertext); offset += 20 {
		newCT := edit(ciphertext, offset-8, make([]byte, 20))
		p := xor(newCT[offset:offset+20], ciphertext[offset:offset+20])
		plaintext = append(plaintext, p...)
	}
	return plaintext
}

func newCTRCookieOracles() (
	generateCookie func(email string) string,
	amIAdmin func(cookie string) bool,
) {
	key := make([]byte, 16)
	rand.Read(key)
	b, _ := aes.NewCipher(key)
	generateCookie = func(email string) string {
		profile := []byte("comment1=cooking%20MCs;userdata=")
		qEmail := bytes.Replace([]byte(email), []byte("="), []byte("%3D"), -1)
		qEmail = bytes.Replace(qEmail, []byte(";"), []byte("%3B"), -1)
		profile = append(profile, qEmail...)
		profile = append(profile, ";comment2=%20like%20a%20pound%20of%20bacon"...)

		iv := make([]byte, 8)
		rand.Read(iv)
		cookie := encryptCTR(profile, b, iv)
		return string(iv) + string(cookie)
	}
	amIAdmin = func(cookie string) bool {
		iv, msg := []byte(cookie[:8]), []byte(cookie[8:])
		cookie = string(decryptCTR(msg, b, iv))
		return strings.Contains(cookie, ";admin=true;")
	}
	return
}

func makeCTRAdminCookie(generateCookie func(email string) string) string {
	prefix := "comment1=cooking%20MCs;userdata="
	tgt := "AA;admin=true;AA"
	msg := strings.Repeat("*", 16)
	out := generateCookie(msg)
	out1 := out[:8+len(prefix)]
	out2 := out[8+len(prefix) : 8+len(prefix)+16]
	out3 := out[8+len(prefix)+16:]
	out2 = xorString(out2, xorString(strings.Repeat("*", 16), tgt))
	return out1 + out2 + out3
}

func newCBCKeyEqIVOracles() (
	encryptMessage func([]byte) []byte,
	decryptMessage func([]byte) error,
	isKeyCorrect func([]byte) bool, // for testing
) {
	key := make([]byte, 16)
	rand.Read(key)
	b, _ := aes.NewCipher(key)
	encryptMessage = func(message []byte) []byte {
		return encryptCBC(padPKCS7(message, 16), b, key)
	}
	decryptMessage = func(ct []byte) error {
		pt := unpadPKCS7(decryptCBC(ct, b, key))
		if !regexp.MustCompile(`^[ -~]+$`).Match(pt) {
			return fmt.Errorf("invalid message: %s", pt)
		}
		return nil
	}
	isKeyCorrect = func(k []byte) bool {
		return bytes.Equal(k, key)
	}
	return
}

func recoverCBCKeyEqIV(
	encryptMessage func([]byte) []byte,
	decryptMessage func([]byte) error,
) []byte {
	ct := encryptMessage(bytes.Repeat([]byte("A"), 16*4))
	copy(ct[16:], make([]byte, 16))
	copy(ct[32:], ct[:16])
	err := decryptMessage(ct).Error()
	pt := []byte(strings.TrimPrefix(err, "invalid message: "))
	if len(pt) != 16*4 {
		panic("unexpected plaintext length")
	}
	return xor(pt[:16], pt[32:48])
}

// ************* SHA1 code *************

const (
	chunk = 64
	init0 = 0x67452301
	init1 = 0xEFCDAB89
	init2 = 0x98BADCFE
	init3 = 0x10325476
	init4 = 0xC3D2E1F0
)

type SHA1 struct {
	h   [5]uint32
	x   [chunk]byte
	nx  int
	len uint64
}

func (d *SHA1) Reset() {
	d.h[0] = init0
	d.h[1] = init1
	d.h[2] = init2
	d.h[3] = init3
	d.h[4] = init4
	d.nx = 0
	d.len = 0
}

// New returns a new hash.Hash computing the SHA1 checksum.
func NewSHA1() *SHA1 {
	d := new(SHA1)
	d.Reset()
	return d
}

func (d *SHA1) Write(p []byte) (nn int, err error) {
	nn = len(p)
	d.len += uint64(nn)
	if d.nx > 0 {
		n := copy(d.x[d.nx:], p)
		d.nx += n
		if d.nx == chunk {
			sha1Block(d, d.x[:])
			d.nx = 0
		}
		p = p[n:]
	}
	if len(p) >= chunk {
		n := len(p) &^ (chunk - 1)
		sha1Block(d, p[:n])
		p = p[n:]
	}
	if len(p) > 0 {
		d.nx = copy(d.x[:], p)
	}
	return
}

func (d *SHA1) checkSum() [20]byte {
	len := d.len
	// Padding.  Add a 1 bit and 0 bits until 56 bytes mod 64.
	var tmp [64]byte
	tmp[0] = 0x80
	if len%64 < 56 {
		d.Write(tmp[0 : 56-len%64])
	} else {
		d.Write(tmp[0 : 64+56-len%64])
	}

	// Length in bits.
	len <<= 3
	putUint64(tmp[:], len)
	d.Write(tmp[0:8])

	if d.nx != 0 {
		panic("d.nx != 0")
	}

	var digest [20]byte

	putUint32(digest[0:], d.h[0])
	putUint32(digest[4:], d.h[1])
	putUint32(digest[8:], d.h[2])
	putUint32(digest[12:], d.h[3])
	putUint32(digest[16:], d.h[4])

	return digest
}

func putUint64(x []byte, s uint64) {
	_ = x[7]
	x[0] = byte(s >> 56)
	x[1] = byte(s >> 48)
	x[2] = byte(s >> 40)
	x[3] = byte(s >> 32)
	x[4] = byte(s >> 24)
	x[5] = byte(s >> 16)
	x[6] = byte(s >> 8)
	x[7] = byte(s)
}

func putUint32(x []byte, s uint32) {
	_ = x[3]
	x[0] = byte(s >> 24)
	x[1] = byte(s >> 16)
	x[2] = byte(s >> 8)
	x[3] = byte(s)
}

const (
	_K0 = 0x5A827999
	_K1 = 0x6ED9EBA1
	_K2 = 0x8F1BBCDC
	_K3 = 0xCA62C1D6
)

func sha1Block(dig *SHA1, p []byte) {
	var w [16]uint32

	h0, h1, h2, h3, h4 := dig.h[0], dig.h[1], dig.h[2], dig.h[3], dig.h[4]
	for len(p) >= chunk {
		// Can interlace the computation of w with the
		// rounds below if needed for speed.
		for i := 0; i < 16; i++ {
			j := i * 4
			w[i] = uint32(p[j])<<24 | uint32(p[j+1])<<16 | uint32(p[j+2])<<8 | uint32(p[j+3])
		}

		a, b, c, d, e := h0, h1, h2, h3, h4

		// Each of the four 20-iteration rounds
		// differs only in the computation of f and
		// the choice of K (_K0, _K1, etc).
		i := 0
		for ; i < 16; i++ {
			f := b&c | (^b)&d
			a5 := a<<5 | a>>(32-5)
			b30 := b<<30 | b>>(32-30)
			t := a5 + f + e + w[i&0xf] + _K0
			a, b, c, d, e = t, a, b30, c, d
		}
		for ; i < 20; i++ {
			tmp := w[(i-3)&0xf] ^ w[(i-8)&0xf] ^ w[(i-14)&0xf] ^ w[(i)&0xf]
			w[i&0xf] = tmp<<1 | tmp>>(32-1)

			f := b&c | (^b)&d
			a5 := a<<5 | a>>(32-5)
			b30 := b<<30 | b>>(32-30)
			t := a5 + f + e + w[i&0xf] + _K0
			a, b, c, d, e = t, a, b30, c, d
		}
		for ; i < 40; i++ {
			tmp := w[(i-3)&0xf] ^ w[(i-8)&0xf] ^ w[(i-14)&0xf] ^ w[(i)&0xf]
			w[i&0xf] = tmp<<1 | tmp>>(32-1)
			f := b ^ c ^ d
			a5 := a<<5 | a>>(32-5)
			b30 := b<<30 | b>>(32-30)
			t := a5 + f + e + w[i&0xf] + _K1
			a, b, c, d, e = t, a, b30, c, d
		}
		for ; i < 60; i++ {
			tmp := w[(i-3)&0xf] ^ w[(i-8)&0xf] ^ w[(i-14)&0xf] ^ w[(i)&0xf]
			w[i&0xf] = tmp<<1 | tmp>>(32-1)
			f := ((b | c) & d) | (b & c)

			a5 := a<<5 | a>>(32-5)
			b30 := b<<30 | b>>(32-30)
			t := a5 + f + e + w[i&0xf] + _K2
			a, b, c, d, e = t, a, b30, c, d
		}
		for ; i < 80; i++ {
			tmp := w[(i-3)&0xf] ^ w[(i-8)&0xf] ^ w[(i-14)&0xf] ^ w[(i)&0xf]
			w[i&0xf] = tmp<<1 | tmp>>(32-1)
			f := b ^ c ^ d
			a5 := a<<5 | a>>(32-5)
			b30 := b<<30 | b>>(32-30)
			t := a5 + f + e + w[i&0xf] + _K3
			a, b, c, d, e = t, a, b30, c, d
		}

		h0 += a
		h1 += b
		h2 += c
		h3 += d
		h4 += e

		p = p[chunk:]
	}

	dig.h[0], dig.h[1], dig.h[2], dig.h[3], dig.h[4] = h0, h1, h2, h3, h4
}

// ************* end SHA1 code *************

func secretPrefixMAC(key, message []byte) []byte {
	s := NewSHA1()
	s.Write(key)
	s.Write(message)
	sha1 := s.checkSum()
	return sha1[:]
}

func checkSecretPrefixMAC(key, message, mac []byte) bool {
	s := NewSHA1()
	s.Write(key)
	s.Write(message)
	sha1 := s.checkSum()
	return bytes.Equal(mac, sha1[:])
}

func mdPadding(len int) []byte {
	buf := &bytes.Buffer{}

	// Padding.  Add a 1 bit and 0 bits until 56 bytes mod 64.
	var tmp [64]byte
	tmp[0] = 0x80
	if len%64 < 56 {
		buf.Write(tmp[0 : 56-len%64])
	} else {
		buf.Write(tmp[0 : 64+56-len%64])
	}

	// Length in bits.
	len <<= 3
	putUint64(tmp[:], uint64(len))
	buf.Write(tmp[0:8])

	return buf.Bytes()
}

func newSecretPrefixMACOracle() (
	cookie []byte,
	amIAdmin func(cookie []byte) bool,
) {
	key := make([]byte, 16)
	rand.Read(key)

	cookieData := []byte("comment1=cooking%20MCs;userdata=foo;comment2=%20like%20a%20pound%20of%20bacon")

	cookie = append(cookie, secretPrefixMAC(key, cookieData)...)
	cookie = append(cookie, cookieData...)

	amIAdmin = func(cookie []byte) bool {
		mac, msg := cookie[:20], cookie[20:]
		if !checkSecretPrefixMAC(key, msg, mac) {
			return false
		}
		return bytes.Contains(msg, []byte(";admin=true;")) ||
			bytes.HasSuffix(msg, []byte(";admin=true"))
	}
	return
}

func extendSHA1(mac, msg, extension []byte) (newMAC, newMSG []byte) {
	newMSG = append(newMSG, msg...)
	newMSG = append(newMSG, mdPadding(len(msg)+16)...)

	s := &SHA1{}
	s.h[0] = binary.BigEndian.Uint32(mac[0:])
	s.h[1] = binary.BigEndian.Uint32(mac[4:])
	s.h[2] = binary.BigEndian.Uint32(mac[8:])
	s.h[3] = binary.BigEndian.Uint32(mac[12:])
	s.h[4] = binary.BigEndian.Uint32(mac[16:])
	s.len = uint64(len(newMSG) + 16)

	s.Write(extension)
	newMSG = append(newMSG, extension...)

	sha1 := s.checkSum()
	return sha1[:], newMSG
}

func makeSHA1AdminCookie(cookie []byte) []byte {
	mac, msg := cookie[:20], cookie[20:]
	newMAC, newMSG := extendSHA1(mac, msg, []byte(";admin=true"))
	return append(newMAC, newMSG...)
}

// ******************** MD4 code **********************

const (
	_Chunk = 64
	_Init0 = 0x67452301
	_Init1 = 0xEFCDAB89
	_Init2 = 0x98BADCFE
	_Init3 = 0x10325476
)

type MD4 struct {
	s   [4]uint32
	x   [_Chunk]byte
	nx  int
	len uint64
}

func (d *MD4) Reset() {
	d.s[0] = _Init0
	d.s[1] = _Init1
	d.s[2] = _Init2
	d.s[3] = _Init3
	d.nx = 0
	d.len = 0
}

func NewMD4() *MD4 {
	d := new(MD4)
	d.Reset()
	return d
}

func (d *MD4) Write(p []byte) (nn int, err error) {
	nn = len(p)
	d.len += uint64(nn)
	if d.nx > 0 {
		n := len(p)
		if n > _Chunk-d.nx {
			n = _Chunk - d.nx
		}
		for i := 0; i < n; i++ {
			d.x[d.nx+i] = p[i]
		}
		d.nx += n
		if d.nx == _Chunk {
			md4Block(d, d.x[0:])
			d.nx = 0
		}
		p = p[n:]
	}
	n := md4Block(d, p)
	p = p[n:]
	if len(p) > 0 {
		d.nx = copy(d.x[:], p)
	}
	return
}

func (d *MD4) checkSum() []byte {
	// Padding.  Add a 1 bit and 0 bits until 56 bytes mod 64.
	len := d.len
	var tmp [64]byte
	tmp[0] = 0x80
	if len%64 < 56 {
		d.Write(tmp[0 : 56-len%64])
	} else {
		d.Write(tmp[0 : 64+56-len%64])
	}

	// Length in bits.
	len <<= 3
	for i := uint(0); i < 8; i++ {
		tmp[i] = byte(len >> (8 * i))
	}
	d.Write(tmp[0:8])

	if d.nx != 0 {
		panic("d.nx != 0")
	}

	var in []byte
	for _, s := range d.s {
		in = append(in, byte(s>>0))
		in = append(in, byte(s>>8))
		in = append(in, byte(s>>16))
		in = append(in, byte(s>>24))
	}
	return in
}

var shift1 = []uint{3, 7, 11, 19}
var shift2 = []uint{3, 5, 9, 13}
var shift3 = []uint{3, 9, 11, 15}

var xIndex2 = []uint{0, 4, 8, 12, 1, 5, 9, 13, 2, 6, 10, 14, 3, 7, 11, 15}
var xIndex3 = []uint{0, 8, 4, 12, 2, 10, 6, 14, 1, 9, 5, 13, 3, 11, 7, 15}

func md4Block(dig *MD4, p []byte) int {
	a := dig.s[0]
	b := dig.s[1]
	c := dig.s[2]
	d := dig.s[3]
	n := 0
	var X [16]uint32
	for len(p) >= _Chunk {
		aa, bb, cc, dd := a, b, c, d

		j := 0
		for i := 0; i < 16; i++ {
			X[i] = uint32(p[j]) | uint32(p[j+1])<<8 | uint32(p[j+2])<<16 | uint32(p[j+3])<<24
			j += 4
		}

		// Round 1.
		for i := uint(0); i < 16; i++ {
			x := i
			s := shift1[i%4]
			f := ((c ^ d) & b) ^ d
			a += f + X[x]
			a = a<<s | a>>(32-s)
			a, b, c, d = d, a, b, c
		}

		// Round 2.
		for i := uint(0); i < 16; i++ {
			x := xIndex2[i]
			s := shift2[i%4]
			g := (b & c) | (b & d) | (c & d)
			a += g + X[x] + 0x5a827999
			a = a<<s | a>>(32-s)
			a, b, c, d = d, a, b, c
		}

		// Round 3.
		for i := uint(0); i < 16; i++ {
			x := xIndex3[i]
			s := shift3[i%4]
			h := b ^ c ^ d
			a += h + X[x] + 0x6ed9eba1
			a = a<<s | a>>(32-s)
			a, b, c, d = d, a, b, c
		}

		a += aa
		b += bb
		c += cc
		d += dd

		p = p[_Chunk:]
		n += _Chunk
	}

	dig.s[0] = a
	dig.s[1] = b
	dig.s[2] = c
	dig.s[3] = d
	return n
}

// ******************** end MD4 code **********************

func secretPrefixMD4(key, message []byte) []byte {
	s := NewMD4()
	s.Write(key)
	s.Write(message)
	md4 := s.checkSum()
	return md4[:]
}

func checkSecretPrefixMD4(key, message, mac []byte) bool {
	s := NewMD4()
	s.Write(key)
	s.Write(message)
	md4 := s.checkSum()
	return bytes.Equal(mac, md4[:])
}

func newSecretPrefixMD4Oracle() (
	cookie []byte,
	amIAdmin func(cookie []byte) bool,
) {
	key := make([]byte, 16)
	rand.Read(key)

	cookieData := []byte("comment1=cooking%20MCs;userdata=foo;comment2=%20like%20a%20pound%20of%20bacon")

	cookie = append(cookie, secretPrefixMD4(key, cookieData)...)
	cookie = append(cookie, cookieData...)

	amIAdmin = func(cookie []byte) bool {
		mac, msg := cookie[:16], cookie[16:]
		if !checkSecretPrefixMD4(key, msg, mac) {
			return false
		}
		return bytes.Contains(msg, []byte(";admin=true;")) ||
			bytes.HasSuffix(msg, []byte(";admin=true"))
	}
	return
}

func md4Padding(len int) []byte {
	buf := &bytes.Buffer{}

	var tmp [64]byte
	tmp[0] = 0x80
	if len%64 < 56 {
		buf.Write(tmp[0 : 56-len%64])
	} else {
		buf.Write(tmp[0 : 64+56-len%64])
	}

	// Length in bits.
	len <<= 3
	for i := uint(0); i < 8; i++ {
		tmp[i] = byte(len >> (8 * i))
	}
	buf.Write(tmp[0:8])

	return buf.Bytes()
}

func extendMD4(mac, msg, extension []byte) (newMAC, newMSG []byte) {
	newMSG = append(newMSG, msg...)
	newMSG = append(newMSG, md4Padding(len(msg)+16)...)

	s := &MD4{}
	s.s[0] = binary.LittleEndian.Uint32(mac[0:])
	s.s[1] = binary.LittleEndian.Uint32(mac[4:])
	s.s[2] = binary.LittleEndian.Uint32(mac[8:])
	s.s[3] = binary.LittleEndian.Uint32(mac[12:])
	s.len = uint64(len(newMSG) + 16)

	s.Write(extension)
	newMSG = append(newMSG, extension...)

	md4 := s.checkSum()
	return md4[:], newMSG
}

func makeMD4AdminCookie(cookie []byte) []byte {
	mac, msg := cookie[:16], cookie[16:]
	newMAC, newMSG := extendMD4(mac, msg, []byte(";admin=true"))
	return append(newMAC, newMSG...)
}

func equal(a, b []byte, pause time.Duration) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		time.Sleep(pause)
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

var signatureLen = 6

func newHMACOracle(pause time.Duration) func(message, signature []byte) bool {
	key := make([]byte, 16)
	rand.Read(key)
	debug := true

	return func(message, signature []byte) bool {
		h := hmac.New(sha1.New, key)
		h.Write(message)
		expected := h.Sum(nil)
		if debug {
			fmt.Printf("%x\n", expected[:signatureLen])
			debug = false
		}

		return equal(signature, expected[:signatureLen], pause)
	}
}

func recoverSignatureFromTiming(message []byte,
	check func(message, signature []byte) bool) []byte {
	timeIt := func(signature []byte) time.Duration {
		start := time.Now()
		check(message, signature)
		return time.Since(start)
	}
	signature := make([]byte, signatureLen)
	for pos := range signature {
		baseline := timeIt(signature)
		var found bool
		for k := 0; k < 256; k++ {
			signature[pos] = byte(k)
			fmt.Printf("\r%x", signature)
			if timeIt(signature)-baseline > 25*time.Millisecond {
				found = true
				break
			}
		}
		if !found {
			// Maybe it was 0 to begin with.
			signature[pos] = 0
		}
	}
	fmt.Printf("\n")
	return signature
}

func recoverSignatureFromAverageTiming(message []byte,
	check func(message, signature []byte) bool) []byte {
	timeIt := func(signature []byte) time.Duration {
		start := time.Now()
		check(message, signature)
		return time.Since(start)
	}
	delta := 4 * time.Millisecond
	averageTime := func(signature []byte) time.Duration {
		var total time.Duration
		var maxTime time.Duration
		for i := 0; i < 32; i++ {
			t := timeIt(signature)
			total += t
			if t > maxTime {
				maxTime = t
			}
		}
		avg := total / 32
		// if maxTime-avg > delta {
		// 	panic("too much variance")
		// }
		return avg
	}
	signature := make([]byte, signatureLen)
	for pos := range signature {
		baseline := averageTime(signature)
		var found bool
		for k := 0; k < 256; k++ {
			signature[pos] = byte(k)
			fmt.Printf("\r%x", signature)
			if averageTime(signature)-baseline > delta {
				found = true
				break
			}
		}
		if !found {
			// Maybe it was 0 to begin with.
			signature[pos] = 0
		}
	}
	fmt.Printf("\n")
	return signature
}
