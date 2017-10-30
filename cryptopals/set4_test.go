package cryptopals

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"testing"
	"time"
)

func TestProblem25(t *testing.T) {
	in := decodeBase64(t, string(readFile(t, "7.txt")))
	b, err := aes.NewCipher([]byte("YELLOW SUBMARINE"))
	fatalIfErr(t, err)
	out := decryptECB(in, b)

	ct, edit := newCTREditOracles(out)
	t.Logf("%s", attackCTREditOracle(ct, edit))
}

func TestProblem26(t *testing.T) {
	generateCookie, amIAdmin := newCTRCookieOracles()
	if amIAdmin(generateCookie(";admin=true;")) {
		t.Fatal("this is too easy")
	}

	if !amIAdmin(makeCTRAdminCookie(generateCookie)) {
		t.Error("not admin")
	}
}

func TestProblem27(t *testing.T) {
	encryptMessage, decryptMessage, isKeyCorrect := newCBCKeyEqIVOracles()
	key := recoverCBCKeyEqIV(encryptMessage, decryptMessage)
	if !isKeyCorrect(key) {
		t.Error("wrong key")
	}
}

func TestProblem28(t *testing.T) {
	key := make([]byte, 16)
	rand.Read(key)
	msg := bytes.Repeat([]byte("hey"), 20)
	mac := secretPrefixMAC(key, msg)
	if !checkSecretPrefixMAC(key, msg, mac) {
		t.Fatal("MAC doesn't validate")
	}
	msg[20] = 'a'
	if checkSecretPrefixMAC(key, msg, mac) {
		t.Error("MAC doesn't invalidate")
	}
}

func TestProblem29(t *testing.T) {
	msg := bytes.Repeat([]byte("hey"), 20)
	s := NewSHA1()
	s.Write(msg)
	s.checkSum()

	ss := NewSHA1()
	ss.Write(msg)
	ss.Write(mdPadding(len(msg)))

	if ss.nx != 0 {
		t.Fatal("data still buffered")
	}
	if s.h != ss.h {
		t.Fatal("wrong h values")
	}

	cookie, amIAdmin := newSecretPrefixMACOracle()
	if amIAdmin(append(cookie, []byte(";admin=true")...)) {
		t.Error("this is too easy")
	}
	if !amIAdmin(makeSHA1AdminCookie(cookie)) {
		t.Error("not admin")
	}
}

func TestProblem30(t *testing.T) {
	msg := bytes.Repeat([]byte("hey"), 20)
	s := NewMD4()
	s.Write(msg)
	s.checkSum()

	ss := NewMD4()
	ss.Write(msg)
	ss.Write(md4Padding(len(msg)))

	if ss.nx != 0 {
		t.Fatal("data still buffered")
	}
	if s.s != ss.s {
		t.Fatal("wrong s values")
	}

	cookie, amIAdmin := newSecretPrefixMD4Oracle()
	if amIAdmin(append(cookie, []byte(";admin=true")...)) {
		t.Error("this is too easy")
	}
	if !amIAdmin(makeMD4AdminCookie(cookie)) {
		t.Error("not admin")
	}
}

func TestProblem31(t *testing.T) {
	check := newHMACOracle(50 * time.Millisecond)
	msg := []byte("I AM ROOT")
	if check(msg, make([]byte, signatureLen)) {
		t.Fatal("too easy")
	}
	sig := recoverSignatureFromTiming(msg, check)
	if !check(msg, sig) {
		t.Error("wrong signature")
	}
}

func TestProblem32(t *testing.T) {
	check := newHMACOracle(5 * time.Millisecond)
	msg := []byte("I AM ROOT")
	if check(msg, make([]byte, signatureLen)) {
		t.Fatal("too easy")
	}
	sig := recoverSignatureFromAverageTiming(msg, check)
	if !check(msg, sig) {
		t.Error("wrong signature")
	}
}
