package cryptopals

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"log"
	mathrand "math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func padPKCS7(in []byte, size int) []byte {
	if size >= 256 {
		panic("can't pad to size higher than 255")
	}
	padLen := size - len(in)%size
	return append(in, bytes.Repeat([]byte{byte(padLen)}, padLen)...)
}

func encryptCBC(src []byte, b cipher.Block, iv []byte) []byte {
	bs := b.BlockSize()
	if len(src)%bs != 0 {
		panic("wrong input length")
	}
	if len(iv) != bs {
		panic("wrong iv length")
	}
	out := make([]byte, len(src))
	prev := iv
	for i := 0; i < len(src)/bs; i++ {
		copy(out[i*bs:], xor(src[i*bs:(i+1)*bs], prev))
		b.Encrypt(out[i*bs:], out[i*bs:])
		prev = out[i*bs : (i+1)*bs]
	}
	return out
}

func decryptCBC(src []byte, b cipher.Block, iv []byte) []byte {
	bs := b.BlockSize()
	if len(src)%bs != 0 {
		panic("wrong input length")
	}
	if len(iv) != bs {
		panic("wrong iv length")
	}
	out := make([]byte, len(src))
	prev := iv
	buf := make([]byte, bs)
	for i := 0; i < len(src)/bs; i++ {
		b.Decrypt(buf, src[i*bs:])
		copy(out[i*bs:], xor(buf, prev))
		prev = src[i*bs : (i+1)*bs]
	}
	return out
}

func encryptECB(in []byte, b cipher.Block) []byte {
	if len(in)%b.BlockSize() != 0 {
		panic("encryptECB: length not a multiple of BlockSize")
	}
	out := make([]byte, len(in))
	for i := 0; i < len(in); i += b.BlockSize() {
		b.Encrypt(out[i:], in[i:])
	}
	return out
}

func newEBCCBCOracle() func([]byte) []byte {
	key := make([]byte, 16)
	rand.Read(key)
	b, _ := aes.NewCipher(key)
	return func(in []byte) []byte {
		prefix := make([]byte, 5+mathrand.Intn(5))
		rand.Read(prefix)
		suffix := make([]byte, 5+mathrand.Intn(5))
		rand.Read(suffix)
		msg := append(append(prefix, in...), suffix...)
		msg = padPKCS7(msg, 16)

		if mathrand.Intn(10)%2 == 0 {
			iv := make([]byte, 16)
			rand.Read(iv)
			return encryptCBC(msg, b, iv)
		} else {
			return encryptECB(msg, b)
		}
	}
}

func newECBSuffixOracle(secret []byte) func([]byte) []byte {
	key := make([]byte, 16)
	rand.Read(key)
	b, _ := aes.NewCipher(key)
	return func(in []byte) []byte {
		time.Sleep(200 * time.Microsecond)
		msg := append(in, secret...)
		return encryptECB(padPKCS7(msg, 16), b)
	}
}

func recoverECBSuffix(oracle func([]byte) []byte) []byte {
	var bs int
	for blockSize := 2; blockSize < 100; blockSize++ {
		msg := bytes.Repeat([]byte{42}, blockSize*2)
		msg = append(msg, 3)
		if detectECB(oracle(msg)[:blockSize*2], blockSize) {
			bs = blockSize
			break
		}
	}
	if bs == 0 {
		panic("didn't detect block size")
	}
	fmt.Println("bs:", bs)

	buildDict := func(known []byte) map[string]byte {
		dict := make(map[string]byte)

		msg := bytes.Repeat([]byte{42}, bs)
		msg = append(msg, known...)
		msg = append(msg, '?')
		msg = msg[len(msg)-bs:]

		for b := 0; b < 256; b++ {
			msg[bs-1] = byte(b)
			res := string(oracle(msg)[:bs])
			dict[res] = byte(b)
		}
		return dict
	}

	dict := buildDict(nil)
	msg := bytes.Repeat([]byte{42}, bs-1)
	res := string(oracle(msg)[:bs])
	firstByte := dict[res]
	fmt.Printf("First byte: %c / %v\n", firstByte, firstByte)

	var plaintext []byte
	for i := 0; i < len(oracle([]byte{})); i++ {
		dict := buildDict(plaintext)
		msg := bytes.Repeat([]byte{42}, mod(bs-i-1, bs))
		skip := i / bs * bs
		res := string(oracle(msg)[skip : skip+bs])
		plaintext = append(plaintext, dict[res])

		// lines := bytes.SplitAfter(plaintext, []byte{'\n'})
		// fmt.Printf("\r%s", lines[len(lines)-1])
		fmt.Printf("%c", dict[res])
	}
	fmt.Printf("\n")

	return nil
}

func mod(a, b int) int {
	return (a%b + b) % b
}

func profileFor(email string) string {
	v := url.Values{}
	v.Set("email", email)
	v.Set("uid", strconv.Itoa(10+mathrand.Intn(90)))
	v.Set("role", "user")
	return v.Encode()
}

func newCutAndPasteECBOracles() (
	generateCookie func(email string) string,
	amIAdmin func(cookie string) bool,
) {
	key := make([]byte, 16)
	rand.Read(key)
	b, _ := aes.NewCipher(key)
	generateCookie = func(email string) string {
		profile := []byte(profileFor(email))
		cookie := encryptECB(padPKCS7(profile, 16), b)
		return string(cookie)
	}
	amIAdmin = func(cookie string) bool {
		cookie = string(unpadPKCS7(decryptECB([]byte(cookie), b)))
		v, err := url.ParseQuery(cookie)
		if err != nil {
			return false
		}
		return v.Get("role") == "admin"
	}
	return
}

func unpadPKCS7(in []byte) []byte {
	if len(in) == 0 {
		return in
	}
	b := in[len(in)-1]
	if len(in) < int(b) {
		return nil
	}
	for i := 1; i < int(b); i++ {
		if in[len(in)-1-i] != b {
			return nil
		}
	}
	return in[:len(in)-int(b)]
}

func makeECBAdminCookie(generateCookie func(email string) string) string {
	// These could be obtained with recoverECBSuffix
	start, _ := "email=", "&role=user&uid=51"

	genBlock := func(prefix string) string {
		msg := strings.Repeat("A", 16-len(start)) + prefix
		return generateCookie(msg)[16:32]
	}

	block1 := generateCookie("FOO@BAR.AA")[:16] // email=FOO@BAR.AA
	block2 := genBlock("AAAAAAAAAA")            // AAAAAAAAAA&role=
	block3 := genBlock("admin")                 // admin&role=user&
	msg := strings.Repeat("A", 16-1-len(start))
	block4 := generateCookie(msg)[16:48] // role=user&uid=51 + padding

	// email=FOO@BAR.AAAAAAAAAAAA&role=admin&role=user&role=user&uid=51 + padding
	return block1 + block2 + block3 + block4
}

func newECBSuffixOracleWithPrefix(secret []byte) func([]byte) []byte {
	key := make([]byte, 16)
	rand.Read(key)
	b, _ := aes.NewCipher(key)
	prefix := make([]byte, mathrand.Intn(100))
	return func(in []byte) []byte {
		time.Sleep(200 * time.Microsecond)
		rand.Read(prefix)
		msg := append(prefix, append(in, secret...)...)
		return encryptECB(padPKCS7(msg, 16), b)
	}
}

func ecbIndex(in []byte, bs int) int {
	if len(in)%bs != 0 {
		panic("wrong sized input")
	}
	prev := in[:bs]
	for i := 1; i < len(in)/bs; i++ {
		if bytes.Equal(prev, in[i*bs:i*bs+bs]) {
			return i*bs - bs
		}
		prev = in[i*bs : i*bs+bs]
	}
	return -1
}

func recoverECBSuffixWithPrefix(oracle func([]byte) []byte) []byte {
	var bs, pl int
	out := oracle(bytes.Repeat([]byte{42}, 500))
	for blockSize := 2; blockSize < 100; blockSize++ {
		if len(out)%blockSize != 0 {
			continue
		}
		i := ecbIndex(out, blockSize)
		if i < 0 {
			continue
		}
		bs = blockSize
		fmt.Println("bs:", bs)
		for p := 0; p < bs; p++ {
			msg := append(bytes.Repeat([]byte{42}, p+bs*2), 'X')
			if ecbIndex(oracle(msg), bs) == i {
				pl = i - p
				fmt.Println("pl:", pl)
				break
			}
		}
		break
	}
	if bs == 0 || pl == 0 {
		panic("didn't detect block or prefix size")
	}

	return recoverECBSuffix(func(in []byte) []byte {
		p := bs - pl%bs
		msg := append(bytes.Repeat([]byte{42}, p), in...)
		out := oracle(msg)
		return out[pl+p:]
	})
}

func newCBCOracles() (
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

		iv := make([]byte, 16)
		rand.Read(iv)
		cookie := encryptCBC(padPKCS7(profile, 16), b, iv)
		return string(iv) + string(cookie)
	}
	amIAdmin = func(cookie string) bool {
		iv, msg := []byte(cookie[:16]), []byte(cookie[16:])
		cookie = string(unpadPKCS7(decryptCBC(msg, b, iv)))
		log.Printf("%q", cookie)
		return strings.Contains(cookie, ";admin=true;")
	}
	return
}

func xorString(a, b string) string {
	return string(xor([]byte(a), []byte(b)))
}

func makeCBCAdminCookie(generateCookie func(email string) string) string {
	prefix := "comment1=cooking%20MCs;userdata="
	tgt := "AA;admin=true;AA"
	msg := strings.Repeat("*", 16*2)
	out := generateCookie(msg)
	out1 := out[:16+len(prefix)]
	out2 := out[16+len(prefix) : 16+len(prefix)+16]
	out3 := out[16+len(prefix)+16:]
	out2 = xorString(out2, xorString(strings.Repeat("*", 16), tgt))
	return out1 + out2 + out3
}
