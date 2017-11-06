package cryptopals

import (
	"bytes"
	"crypto/aes"
	"crypto/sha1"
	"math/big"
	"testing"
)

func TestProblem33(t *testing.T) {
	dh := dhParams{p, big2}

	a := dh.Private()
	A := dh.Public(a)

	b := dh.Private()
	B := dh.Public(b)

	sA, sB := dh.Secret(a, B), dh.Secret(b, A)
	if sA.Cmp(sB) != 0 {
		t.Error("got different results")
	}
}

func TestProblem34(t *testing.T) {
	dh := dhParams{p, big2}
	a := dh.Private()
	A := dh.Public(a)

	{
		bot := initDHEchoBot(p, big2, A)
		B := bot.getPublic()

		s := sha1.Sum(dh.Secret(a, B).Bytes())
		block, _ := aes.NewCipher(s[:16])

		pt := []byte("Hello world")
		ct := encryptCBCWithIV(pt, block)
		echo := bot.echo(ct)
		res := decryptCBCWithIV(echo, block)

		if bytes.Equal(echo, ct) {
			t.Error("should have reencrypted")
		}
		if !bytes.Equal(pt, res) {
			t.Error("wrong echo")
		}
	}

	{
		bot := initDHEchoBotMITMp(p, big2, A)
		B := bot.getPublic()

		s := sha1.Sum(dh.Secret(a, B).Bytes())
		block, _ := aes.NewCipher(s[:16])

		pt := []byte("Hello world")
		ct := encryptCBCWithIV(pt, block)
		echo := bot.echo(ct)
		res := decryptCBCWithIV(echo, block)

		if bytes.Equal(echo, ct) {
			t.Error("mitm: should have reencrypted")
		}
		if !bytes.Equal(pt, res) {
			t.Error("mitm: wrong echo")
		}

		if !bytes.Equal(pt, decryptEchoBotMITMp(echo)) {
			t.Error("mitm: failed to decrypt")
		}
	}
}

func TestProblem35(t *testing.T) {
	dh := dhParams{p, big2}
	a := dh.Private()
	A := dh.Public(a)

	{
		bot := initDHEchoBotMITMg1(p, big2, A)
		B := bot.getPublic()

		s := sha1.Sum(dh.Secret(a, B).Bytes())
		block, _ := aes.NewCipher(s[:16])

		pt := []byte("Hello world")
		ct := encryptCBCWithIV(pt, block)
		echo := bot.echo(ct)
		res := decryptCBCWithIV(echo, block)

		if bytes.Equal(echo, ct) {
			t.Error("g1: should have reencrypted")
		}
		if !bytes.Equal(pt, res) {
			t.Error("g1: wrong echo")
		}

		if !bytes.Equal(pt, decryptEchoBotMITMg1(echo)) {
			t.Error("g1: failed to decrypt")
		}
	}

	{
		bot := initDHEchoBotMITMgp(p, big2, A)
		B := bot.getPublic()

		s := sha1.Sum(dh.Secret(a, B).Bytes())
		block, _ := aes.NewCipher(s[:16])

		pt := []byte("Hello world")
		ct := encryptCBCWithIV(pt, block)
		echo := bot.echo(ct)
		res := decryptCBCWithIV(echo, block)

		if bytes.Equal(echo, ct) {
			t.Error("gp: should have reencrypted")
		}
		if !bytes.Equal(pt, res) {
			t.Error("gp: wrong echo")
		}

		if !bytes.Equal(pt, decryptEchoBotMITMgp(echo)) {
			t.Error("gp: failed to decrypt")
		}
	}

	{
		bot := initDHEchoBotMITMgpMinus1(p, big2, A)
		B := bot.getPublic()

		t.Log(dh.Secret(a, B).Cmp(new(big.Int).Sub(p, big1)), dh.Secret(a, B).Cmp(big1))

		s := sha1.Sum(dh.Secret(a, B).Bytes())
		block, _ := aes.NewCipher(s[:16])

		pt := []byte("Hello world")
		ct := encryptCBCWithIV(pt, block)
		echo := bot.echo(ct)
		res := decryptCBCWithIV(echo, block)

		if bytes.Equal(echo, ct) {
			t.Error("gpMinus1: should have reencrypted")
		}
		if !bytes.Equal(pt, res) {
			t.Error("gpMinus1: wrong echo")
		}

		if res, _ := decryptEchoBotMITMgpMinus1(dh.p, echo); !bytes.Equal(pt, res) {
			t.Error("gpMinus1: failed to decrypt")
		}
	}
}

func TestProblem36(t *testing.T) {
	password := []byte("password")
	srv := newSRPServer(password, false)
	_, err := srpClient(password, srv, false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProblem37(t *testing.T) {
	password := []byte("password")
	srv := newSRPServer(password, false)
	_, err := fakeSRPClient(srv)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProblem38(t *testing.T) {
	password := []byte("password")
	srv := newSRPServer(password, true)
	_, err := srpClient(password, srv, true)
	if err != nil {
		t.Fatal(err)
	}

	mitm := &srpServerMITM{}
	_, err = srpClient(password, mitm, true)
	if err != nil {
		t.Fatal(err)
	}

	if mitm.tryPassword([]byte("foo")) {
		t.Error("wrong password returned true")
	}
	if !mitm.tryPassword(password) {
		t.Error("correct password returned false")
	}
}

func TestProblem39(t *testing.T) {
	m := []byte("YELLOW SUBMARINE")
	for i := 0; i < 100; i++ {
		key := rsaGenerate()
		res := rsaDecrypt(rsaEncrypt(m, &key.PublicKey), key)
		if !bytes.Equal(m, res) {
			t.Errorf("wrong message: %q", res)
		}
	}
}

func TestProblem40(t *testing.T) {
	x := big.NewInt(987654345678987)
	if cubeRoot(new(big.Int).Exp(x, big3, nil)).Cmp(x) != 0 {
		t.Fatal("failed to take cube root")
	}

	m := []byte("YELLOW SUBMARINE")
	broadcast := broadcastRSA(m)
	m1 := crtRSA(broadcast)

	if !bytes.Equal(m, m1) {
		t.Error("plaintext not recovered")
	}
}
