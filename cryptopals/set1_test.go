package cryptopals

import (
	"bytes"
	"crypto/aes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

func TestProblem1(t *testing.T) {
	res, err := hexToBase64("49276d206b696c6c696e6720796f757220627261696e206c696b65206120706f69736f6e6f7573206d757368726f6f6d")
	if err != nil {
		t.Fatal(err)
	}
	if res != "SSdtIGtpbGxpbmcgeW91ciBicmFpbiBsaWtlIGEgcG9pc29ub3VzIG11c2hyb29t" {
		t.Error("wrong string", res)
	}
}

func TestProblem2(t *testing.T) {
	res := xor(decodeHex(t, "1c0111001f010100061a024b53535009181c"),
		decodeHex(t, "686974207468652062756c6c277320657965"))
	if !bytes.Equal(res, decodeHex(t, "746865206b696420646f6e277420706c6179")) {
		t.Errorf("wrong result: %x", res)
	}
}

func decodeHex(t *testing.T, s string) []byte {
	t.Helper()
	v, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal("failed to decode hex:", s)
	}
	return v
}

func corpusFromFile(name string) map[rune]float64 {
	text, err := ioutil.ReadFile(name)
	if err != nil {
		panic(fmt.Sprintln("failed to read corpus file:", err))
	}
	return buildCorpus(string(text))
}

var corpus = corpusFromFile("_testdata/aliceinwonderland.txt")

func TestProblem3(t *testing.T) {
	res, _, _ := findSingleXORKey(decodeHex(t, "1b37373331363f78151b7f2b783431333d78397828372d363c78373e783a393b3736"), corpus)
	t.Logf("%s", res)
}

func TestProblem4(t *testing.T) {
	text := readFile(t, "4.txt")
	var bestScore float64
	var res []byte
	for _, line := range strings.Split(string(text), "\n") {
		out, _, score := findSingleXORKey(decodeHex(t, line), corpus)
		if score > bestScore {
			res = out
			bestScore = score
		}
	}
	t.Logf("%s", res)
}

func TestProblem5(t *testing.T) {
	input := []byte(`Burning 'em, if you ain't quick and nimble
I go crazy when I hear a cymbal`)
	res := repeatingXOR(input, []byte("ICE"))
	if !bytes.Equal(res, decodeHex(t, "0b3637272a2b2e63622c2e69692a23693a2a3c6324202d623d63343c2a26226324272765272a282b2f20430a652e2c652a3124333a653e2b2027630c692b20283165286326302e27282f")) {
		t.Error("wrong result:", res)
	}
}

func TestProblem6(t *testing.T) {
	res := hammingDistance([]byte("this is a test"), []byte("wokka wokka!!!"))
	if res != 37 {
		t.Fatal("wrong Hamming distance:", res)
	}

	text := decodeBase64(t, string(readFile(t, "6.txt")))
	t.Log("likely size:", findRepeatingXORSize(text))

	key := findRepeatingXORKey(text, corpus)
	t.Logf("likely key: %q", key)

	t.Logf("%s", repeatingXOR(text, key))
}

func readFile(t *testing.T, name string) []byte {
	t.Helper()
	data, err := ioutil.ReadFile(name)
	if err != nil {
		t.Fatal("failed to read file:", err)
	}
	return data
}

func decodeBase64(t *testing.T, s string) []byte {
	t.Helper()
	v, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatal("failed to decode base64:", s)
	}
	return v
}

func TestProblem7(t *testing.T) {
	in := decodeBase64(t, string(readFile(t, "7.txt")))
	b, err := aes.NewCipher([]byte("YELLOW SUBMARINE"))
	fatalIfErr(t, err)
	out := decryptECB(in, b)
	t.Logf("%s", out)
}

func fatalIfErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestProblem8(t *testing.T) {
	all := string(readFile(t, "8.txt"))
	for i, hs := range strings.Split(all, "\n") {
		if detectECB(decodeHex(t, hs), 16) {
			t.Logf("ciphertext number %d is encrypted with ECB", i+1)
		}
	}
}
