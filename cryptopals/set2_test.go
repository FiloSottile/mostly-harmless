package cryptopals

import (
	"bytes"
	"crypto/aes"
	"testing"
)

func TestProblem9(t *testing.T) {
	if res := padPKCS7([]byte("YELLOW SUBMARINE"), 16); !bytes.Equal(res, []byte("YELLOW SUBMARINE\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10")) {
		t.Errorf("%q", res)
	}
	if res := padPKCS7([]byte("YELLOW SUBMARINE"), 20); !bytes.Equal(res, []byte("YELLOW SUBMARINE\x04\x04\x04\x04")) {
		t.Errorf("%q", res)
	}
}

func TestProblem10(t *testing.T) {
	msg := []byte("YELLOW SUBMARINEYELLOW SUBMARINE")
	iv := make([]byte, 16)
	b, _ := aes.NewCipher([]byte("YELLOW SUBMARINE"))
	res := decryptCBC(encryptCBC(msg, b, iv), b, iv)
	if !bytes.Equal(res, msg) {
		t.Errorf("%q", res)
	}

	msg = decodeBase64(t, string(readFile(t, "10.txt")))
	t.Logf("%s", decryptCBC(msg, b, iv))
}

func TestProblem11(t *testing.T) {
	oracle := newEBCCBCOracle()
	payload := bytes.Repeat([]byte{42}, 16*3)
	cbc, ecb := 0, 0
	for i := 0; i < 1000; i++ {
		out := oracle(payload)
		if detectECB(out, 16) {
			ecb++
		} else {
			cbc++
		}
	}
	t.Log(ecb, cbc)
}

func TestProblem12(t *testing.T) {
	secret := decodeBase64(t,
		`Um9sbGluJyBpbiBteSA1LjAKV2l0aCBteSByYWctdG9wIGRvd24gc28gbXkg
aGFpciBjYW4gYmxvdwpUaGUgZ2lybGllcyBvbiBzdGFuZGJ5IHdhdmluZyBq
dXN0IHRvIHNheSBoaQpEaWQgeW91IHN0b3A/IE5vLCBJIGp1c3QgZHJvdmUg
YnkK`)
	oracle := newECBSuffixOracle(secret)
	recoverECBSuffix(oracle)
}

func TestProblem13(t *testing.T) {
	t.Log(profileFor("example@example.com"))
	t.Log(profileFor("example@example.com&role=admin"))

	generateCookie, amIAdmin := newCutAndPasteECBOracles()
	if amIAdmin(generateCookie("example@example.com")) {
		t.Fatal("this is too easy")
	}

	if !amIAdmin(makeECBAdminCookie(generateCookie)) {
		t.Error("not admin")
	}
}

func TestProblem14(t *testing.T) {
	secret := decodeBase64(t,
		`Um9sbGluJyBpbiBteSA1LjAKV2l0aCBteSByYWctdG9wIGRvd24gc28gbXkg
aGFpciBjYW4gYmxvdwpUaGUgZ2lybGllcyBvbiBzdGFuZGJ5IHdhdmluZyBq
dXN0IHRvIHNheSBoaQpEaWQgeW91IHN0b3A/IE5vLCBJIGp1c3QgZHJvdmUg
YnkK`)
	oracle := newECBSuffixOracleWithPrefix(secret)
	recoverECBSuffixWithPrefix(oracle)
}

func TestProblem15(t *testing.T) {
	assertNil(t, unpadPKCS7([]byte("ICE ICE BABY\x05\x05\x05\x05")))
	assertNil(t, unpadPKCS7([]byte("ICE ICE BABY\x01\x02\x03\x04")))
	assertNil(t, unpadPKCS7([]byte("YELLOW SUBMARINE\x00\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10")))
	assertEqual(t, unpadPKCS7([]byte("ICE ICE BABY\x04\x04\x04\x04")), []byte("ICE ICE BABY"))
	assertEqual(t, unpadPKCS7([]byte("YELLOW SUBMARINE\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10")), []byte("YELLOW SUBMARINE"))
	assertNil(t, unpadPKCS7([]byte("\x04\x04\x04")))
	assertEqual(t, unpadPKCS7([]byte("\x04\x04\x04\x04")), []byte(""))
}

func assertNil(t *testing.T, v []byte) {
	t.Helper()
	if v != nil {
		t.Error("value not nil")
	}
}

func assertEqual(t *testing.T, a, b []byte) {
	t.Helper()
	if !bytes.Equal(a, b) {
		t.Error("value not equal")
	}
}

func TestProblem16(t *testing.T) {
	generateCookie, amIAdmin := newCBCCookieOracles()
	if amIAdmin(generateCookie(";admin=true;")) {
		t.Fatal("this is too easy")
	}

	if !amIAdmin(makeCBCAdminCookie(generateCookie)) {
		t.Error("not admin")
	}
}
