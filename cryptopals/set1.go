package cryptopals

import (
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"log"
	"math"
	"math/bits"
	"unicode/utf8"
)

func hexToBase64(hs string) (string, error) {
	v, err := hex.DecodeString(hs)
	if err != nil {
		return "", err
	}
	log.Printf("%s", v)
	return base64.StdEncoding.EncodeToString(v), nil
}

func xor(a, b []byte) []byte {
	if len(a) != len(b) {
		panic("xor: mismatched lengths")
	}
	res := make([]byte, len(a))
	for i := range a {
		res[i] = a[i] ^ b[i]
	}
	return res
}

func buildCorpus(text string) map[rune]float64 {
	c := make(map[rune]float64)
	for _, char := range text {
		c[char]++
	}
	total := utf8.RuneCountInString(text)
	for char := range c {
		c[char] = c[char] / float64(total)
	}
	return c
}

func scoreEnglish(text string, c map[rune]float64) float64 {
	var score float64
	for _, char := range text {
		score += c[char]
	}
	return score / float64(utf8.RuneCountInString(text))
}

func singleXOR(in []byte, key byte) []byte {
	res := make([]byte, len(in))
	for i, c := range in {
		res[i] = c ^ key
	}
	return res
}

func findSingleXORKey(in []byte, c map[rune]float64) (res []byte, key byte, score float64) {
	for k := 0; k < 256; k++ {
		out := singleXOR(in, byte(k))
		s := scoreEnglish(string(out), c)
		if s > score {
			res = out
			score = s
			key = byte(k)
		}
	}
	return
}

func repeatingXOR(in, key []byte) []byte {
	res := make([]byte, len(in))
	for i := range in {
		res[i] = in[i] ^ key[i%len(key)]
	}
	return res
}

func hammingDistance(a, b []byte) int {
	if len(a) != len(b) {
		panic("hammingDistance: different lengths")
	}
	var res int
	for i := range a {
		res += bits.OnesCount8(a[i] ^ b[i])
	}
	return res
}

func findRepeatingXORSize(in []byte) int {
	var res int
	bestScore := math.MaxFloat64
	for keyLen := 2; keyLen < 40; keyLen++ {
		a, b := in[:keyLen*4], in[keyLen*4:keyLen*4*2]
		score := float64(hammingDistance(a, b)) / float64(keyLen)
		if score < bestScore {
			res = keyLen
			bestScore = score
		}
	}
	return res
}

func findRepeatingXORKey(in []byte, c map[rune]float64) []byte {
	keySize := findRepeatingXORSize(in)
	column := make([]byte, (len(in)+keySize-1)/keySize)
	key := make([]byte, keySize)
	for col := 0; col < keySize; col++ {
		for row := range column {
			if row*keySize+col >= len(in) {
				continue
			}
			column[row] = in[row*keySize+col]
		}
		_, k, _ := findSingleXORKey(column, c)
		key[col] = k
	}
	return key
}

func decryptECB(in []byte, b cipher.Block) []byte {
	if len(in)%b.BlockSize() != 0 {
		panic("decryptECB: length not a multiple of BlockSize")
	}
	out := make([]byte, len(in))
	for i := 0; i < len(in); i += b.BlockSize() {
		b.Decrypt(out[i:], in[i:])
	}
	return out
}

func detectECB(in []byte, blockSize int) bool {
	if len(in)%blockSize != 0 {
		panic("detectECB: length not a multiple of blockSize")
	}
	seen := make(map[string]struct{})
	for i := 0; i < len(in); i += blockSize {
		val := string(in[i : i+blockSize])
		if _, ok := seen[val]; ok {
			return true
		}
		seen[val] = struct{}{}
	}
	return false
}
