package cryptopals

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"math/big"
	"testing"
)

func TestProblem41(t *testing.T) {
	pub, encrypt, decrypt := decryptRSAOnceOracle()
	msg := make([]byte, 50)
	rand.Read(msg)
	msg[0] = 1
	c := encrypt(msg)
	if p := decrypt(c); !bytes.Equal(msg, p) {
		t.Fatal("wrong decryption:", p)
	}
	if p := decrypt(c); p != nil {
		t.Fatal("decrypted the same message twice")
	}
	if p := decryptRSAAgain(c, pub, decrypt); !bytes.Equal(msg, p) {
		t.Error("wrong attack decryption:", p)
	}
}

func TestProblem42(t *testing.T) {
	verify := bb06Oracle()
	msg := []byte("Hello, BB'06!")
	sig := bb06Forgery(msg)
	if !verify(msg, sig) {
		t.Fail()
	}
}

func hexToBig(t *testing.T, s string) *big.Int {
	t.Helper()
	res, ok := new(big.Int).SetString(s, 16)
	if !ok {
		t.Fatal("failed to decode big.Int:", s)
	}
	return res
}

func decToBig(t *testing.T, s string) *big.Int {
	t.Helper()
	res, ok := new(big.Int).SetString(s, 10)
	if !ok {
		t.Fatal("failed to decode big.Int:", s)
	}
	return res
}

func TestProblem43(t *testing.T) {
	x, y := cryptopalsDSAParams.Generate()
	msg := []byte("Hello, DSA!")
	r, s := cryptopalsDSAParams.Sign(x, msg)
	if !cryptopalsDSAParams.Verify(y, r, s, msg) {
		t.Fatal("valid signature did not verify")
	}
	if cryptopalsDSAParams.Verify(y, r, new(big.Int).Add(s, big1), msg) {
		t.Fatal("invalid signature did verify")
	}

	y = hexToBig(t, "84ad4719d044495496a3201c8ff484feb45b962e7302e56a392aee4abab3e4bdebf2955b4736012f21a08084056b19bcd7fee56048e004e44984e2f411788efdc837a0d2e5abb7b555039fd243ac01f0fb2ed1dec568280ce678e931868d23eb095fde9d3779191b8c0299d6e07bbb283e6633451e535c45513b2d33c99ea17")
	msg = []byte("For those that envy a MC it can be hazardous to your health\nSo be friendly, a matter of life and death, just like a etch-a-sketch\n")
	h := sha1.Sum(msg)
	if !bytes.Equal(h[:], decodeHex(t, "d2d0714f014a9784047eaeccf956520045c45265")) {
		t.Fatal("wrong message")
	}
	r = decToBig(t, "548099063082341131477253921760299949438196259240")
	s = decToBig(t, "857042759984254168557880549501802188789837994940")

	x = recoverDSAKeyFromLowK(cryptopalsDSAParams, y, r, s, msg)

	fingerprint := sha1.Sum([]byte(fmt.Sprintf("%x", x)))
	if !bytes.Equal(fingerprint[:], decodeHex(t, "0954edd5e0afe5542a4adf012611a91912a3ec16")) {
		t.Error("wrong key")
	}
}

func TestProblem44(t *testing.T) {
	// y := hexToBig(t, "2d026f4bf30195ede3a088da85e398ef869611d0f68f0713d51c9c1a3a26c95105d915e2d8cdf26d056b86b8a7b85519b1c23cc3ecdc6062650462e3063bd179c2a6581519f674a61f1d89a1fff27171ebc1b93d4dc57bceb7ae2430f98a6a4d83d8279ee65d71c1203d2c96d65ebbf7cce9d32971c3de5084cce04a2e147821")
	msg1 := []byte("Listen for me, you better listen for me now. ")
	s1 := decToBig(t, "1267396447369736888040262262183731677867615804316")
	r := decToBig(t, "1105520928110492191417703162650245113664610474875")
	msg2 := []byte("Pure black people mon is all I mon know. ")
	s2 := decToBig(t, "1021643638653719618255840562522049391608552714967")

	x := recoverDSAKeyFromRepeatedK(cryptopalsDSAParams, r, s1, s2, msg1, msg2)

	fingerprint := sha1.Sum([]byte(fmt.Sprintf("%x", x)))
	if !bytes.Equal(fingerprint[:], decodeHex(t, "ca8f6f7c66fa362d40760d135b763eb8527d3d52")) {
		t.Error("wrong key")
	}
}

func TestProblem45(t *testing.T) {
	g0Params := cryptopalsDSAParams
	g0Params.G = big.NewInt(0)

	x, y := g0Params.Generate()
	msg := []byte("Hello, DSA!")
	r, s := g0Params.Sign(x, msg)
	if !g0Params.Verify(y, r, s, msg) {
		t.Fatal("valid signature did not verify")
	}
	if !g0Params.Verify(y, r, new(big.Int).Add(s, big1), msg) {
		t.Fatal("invalid s did not verify")
	}
	msg[0] = 'X'
	if !g0Params.Verify(y, r, s, msg) {
		t.Fatal("invalid signature did not verify")
	}

	_, y = cryptopalsDSAParams.Generate()
	r, s = magicDSAg1Signature(cryptopalsDSAParams, y)
	g1Params := cryptopalsDSAParams
	g1Params.G = new(big.Int).Add(g1Params.P, big1)
	if !g1Params.Verify(y, r, s, msg) {
		t.Fatal("g = 1 signature did not verify")
	}
}

func TestProblem46(t *testing.T) {
	msg := decodeBase64(t, "VGhhdCdzIHdoeSBJIGZvdW5kIHlvdSBkb24ndCBwbGF5IGFyb3VuZCB3aXRoIHRoZSBGdW5reSBDb2xkIE1lZGluYQ==")
	pub, encrypt, isPlaintextEven := rsaParityOracle()
	c := encrypt(msg)
	attackRSAParityOracle(pub, c, isPlaintextEven)
}

func TestProblem47(t *testing.T) {
	msg := []byte("kick it, CC")
	pub, encrypt, isPaddingValid := rsaPKCS1Oracle(256)
	c := encrypt(msg)
	attackBB98(pub, c, isPaddingValid)
}

func TestProblem48(t *testing.T) {
	msg := decodeBase64(t, "VGhhdCdzIHdoeSBJIGZvdW5kIHlvdSBkb24ndCBwbGF5IGFyb3VuZCB3aXRoIHRoZSBGdW5reSBDb2xkIE1lZGluYQ==")
	pub, encrypt, isPaddingValid := rsaPKCS1Oracle(768)
	c := encrypt(msg)
	attackBB98(pub, c, isPaddingValid)
}
