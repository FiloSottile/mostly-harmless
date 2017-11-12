package cryptopals

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"math/big"
)

type dhParams struct {
	p, g *big.Int
}

var p, _ = new(big.Int).SetString("ffffffffffffffffc90fdaa22168c234c4c6628b80dc1cd129024e088a67cc74020bbea63b139b22514a08798e3404ddef9519b3cd3a431b302b0a6df25f14374fe1356d6d51c245e485b576625e7ec6f44c42e9a637ed6b0bff5cb6f406b7edee386bfb5a899fa5ae9f24117c4b1fe649286651ece45b3dc2007cb8a163bf0598da48361c55d39a69163fa8fd24cf5f83655d23dca3ad961c62f356208552bb9ed529077096966d670c354e4abc9804f1746c08ca237327ffffffffffffffff", 16)

var big1 = big.NewInt(1)
var big2 = big.NewInt(2)
var big3 = big.NewInt(3)

func (dh dhParams) Private() *big.Int {
	a, err := rand.Int(rand.Reader, dh.p)
	if err != nil {
		panic(err)
	}
	return a
}

func (dh dhParams) Public(priv *big.Int) *big.Int {
	return new(big.Int).Exp(dh.g, priv, dh.p)
}

func (dh dhParams) Secret(priv, pub *big.Int) *big.Int {
	return new(big.Int).Exp(pub, priv, dh.p)
}

type dhEchoBot interface {
	getPublic() *big.Int
	echo([]byte) []byte
}

type dhEchoBotPeer struct {
	p    dhParams
	A, b *big.Int
	aes  cipher.Block
}

func initDHEchoBot(p, g, A *big.Int) dhEchoBot {
	e := &dhEchoBotPeer{
		p: dhParams{p: p, g: g},
		A: A,
	}
	e.b = e.p.Private()

	s := sha1.Sum(e.p.Secret(e.b, e.A).Bytes())
	e.aes, _ = aes.NewCipher(s[:16])

	return e
}

func (e *dhEchoBotPeer) getPublic() *big.Int {
	return e.p.Public(e.b)
}

func encryptCBCWithIV(src []byte, b cipher.Block) []byte {
	src = padPKCS7(src, 16)
	iv := make([]byte, 16, 16+len(src))
	rand.Read(iv)
	return append(iv, encryptCBC(src, b, iv)...)
}

func decryptCBCWithIV(in []byte, b cipher.Block) []byte {
	iv, ct := in[:16], in[16:]
	pt := decryptCBC(ct, b, iv)
	return unpadPKCS7(pt)
}

func (e *dhEchoBotPeer) echo(in []byte) []byte {
	pt := decryptCBCWithIV(in, e.aes)
	return encryptCBCWithIV(pt, e.aes)
}

type dhEchoBotMITMp struct {
	bot dhEchoBot
	p   *big.Int
}

func initDHEchoBotMITMp(p, g, A *big.Int) dhEchoBot {
	return &dhEchoBotMITMp{initDHEchoBot(p, g, p), p}
}

func (m *dhEchoBotMITMp) getPublic() *big.Int {
	return m.p
}

func (e *dhEchoBotMITMp) echo(in []byte) []byte {
	return e.bot.echo(in)
}

func decryptEchoBotMITMp(in []byte) []byte {
	s := sha1.Sum(big.NewInt(0).Bytes())
	block, _ := aes.NewCipher(s[:16])
	return decryptCBCWithIV(in, block)
}

type dhEchoBotMITMg1 struct {
	bot  dhEchoBot
	p, B *big.Int
}

func initDHEchoBotMITMg1(p, g, A *big.Int) dhEchoBot {
	return &dhEchoBotMITMg1{initDHEchoBot(p, big1, A), p, nil}
}

func (m *dhEchoBotMITMg1) getPublic() *big.Int {
	m.B = m.bot.getPublic()
	return m.B
}

func (e *dhEchoBotMITMg1) echo(in []byte) []byte {
	return encryptEchoBotMITMg1(decryptEchoBotMITMg1(in))
}

func decryptEchoBotMITMg1(in []byte) []byte {
	s := sha1.Sum(big.NewInt(1).Bytes())
	block, _ := aes.NewCipher(s[:16])
	return decryptCBCWithIV(in, block)
}

func encryptEchoBotMITMg1(in []byte) []byte {
	s := sha1.Sum(big.NewInt(1).Bytes())
	block, _ := aes.NewCipher(s[:16])
	return encryptCBCWithIV(in, block)
}

type dhEchoBotMITMgp struct {
	bot  dhEchoBot
	p, B *big.Int
}

func initDHEchoBotMITMgp(p, g, A *big.Int) dhEchoBot {
	return &dhEchoBotMITMgp{initDHEchoBot(p, p, A), p, nil}
}

func (m *dhEchoBotMITMgp) getPublic() *big.Int {
	m.B = m.bot.getPublic()
	return m.B
}

func (e *dhEchoBotMITMgp) echo(in []byte) []byte {
	return encryptEchoBotMITMgp(decryptEchoBotMITMgp(in))
}

func decryptEchoBotMITMgp(in []byte) []byte {
	s := sha1.Sum(big.NewInt(0).Bytes())
	block, _ := aes.NewCipher(s[:16])
	return decryptCBCWithIV(in, block)
}

func encryptEchoBotMITMgp(in []byte) []byte {
	s := sha1.Sum(big.NewInt(0).Bytes())
	block, _ := aes.NewCipher(s[:16])
	return encryptCBCWithIV(in, block)
}

type dhEchoBotMITMgpMinus1 struct {
	bot  dhEchoBot
	p, B *big.Int
}

func initDHEchoBotMITMgpMinus1(p, g, A *big.Int) dhEchoBot {
	g = new(big.Int).Sub(p, big1)
	return &dhEchoBotMITMgpMinus1{initDHEchoBot(p, g, A), p, nil}
}

func (m *dhEchoBotMITMgpMinus1) getPublic() *big.Int {
	m.B = m.bot.getPublic()
	return m.B
}

func (m *dhEchoBotMITMgpMinus1) echo(in []byte) []byte {
	pt, key := decryptEchoBotMITMgpMinus1(m.p, in)
	block, _ := aes.NewCipher(key)
	return encryptCBCWithIV(pt, block)
}

func decryptEchoBotMITMgpMinus1(p *big.Int, in []byte) (key, pt []byte) {
	minus1 := new(big.Int).Sub(p, big1)
	sNeg := sha1.Sum(minus1.Bytes())
	block, _ := aes.NewCipher(sNeg[:16])
	negative := decryptCBCWithIV(in, block)

	sPos := sha1.Sum(big1.Bytes())
	block, _ = aes.NewCipher(sPos[:16])
	positive := decryptCBCWithIV(in, block)

	if len(negative) > 0 {
		println("negative")
		return negative, sNeg[:16]
	}
	println("positive")
	return positive, sPos[:16]
}

type srpSrv interface {
	exchange(A *big.Int) (salt []byte, B *big.Int)
	verify(mac []byte) bool
}

type srpServer struct {
	simple bool

	salt []byte
	v    *big.Int

	key []byte
}

func newSRPServer(password []byte, simple bool) srpSrv {
	salt := make([]byte, 16)
	rand.Read(salt)
	xH := sha256.Sum256(append(salt, password...))
	x := new(big.Int).SetBytes(xH[:])
	v := new(big.Int).Exp(big2, x, p)
	return &srpServer{
		salt: salt, v: v, simple: simple,
	}
}

func (s *srpServer) exchange(A *big.Int) (salt []byte, B *big.Int) {
	dh := &dhParams{p, big2}
	// B = kv + g**b % N
	b := dh.Private()
	if s.simple {
		B = dh.Public(b)
	} else {
		B = new(big.Int)
		B.Mul(big3, s.v)
		B.Add(B, dh.Public(b))
		B.Mod(B, p)
	}

	uH := sha256.Sum256(append(A.Bytes(), B.Bytes()...))
	u := new(big.Int).SetBytes(uH[:])

	// S = (A * v**u) ** b % N
	S := new(big.Int)
	S.Exp(s.v, u, p)
	S.Mul(S, A)
	S.Exp(S, b, p)
	K := sha256.Sum256(S.Bytes())
	s.key = K[:]

	return s.salt, B
}

func (s *srpServer) verify(mac []byte) bool {
	h := hmac.New(sha256.New, s.key)
	h.Write(s.salt)
	return hmac.Equal(h.Sum(nil), mac)
}

func srpClient(password []byte, srv srpSrv, simple bool) ([]byte, error) {
	dh := &dhParams{p, big2}
	a := dh.Private()
	A := dh.Public(a)
	salt, B := srv.exchange(A)

	uH := sha256.Sum256(append(A.Bytes(), B.Bytes()...))
	u := new(big.Int).SetBytes(uH[:])

	xH := sha256.Sum256(append(salt, password...))
	x := new(big.Int).SetBytes(xH[:])

	// S = (B - k * g**x)**(a + u * x) % N

	exp := new(big.Int)
	exp.Mul(u, x)
	exp.Add(exp, a)
	exp.Mod(exp, p)
	S := new(big.Int)
	if simple {
		S.Exp(B, exp, p)
	} else {
		S.Exp(big2, x, p)
		S.Mul(S, big3)
		S.Sub(B, S)
		S.Exp(S, exp, p)
	}

	K := sha256.Sum256(S.Bytes())
	key := K[:]

	h := hmac.New(sha256.New, key)
	h.Write(salt)
	if !srv.verify(h.Sum(nil)) {
		return nil, errors.New("wrong password")
	}
	return key, nil
}

func fakeSRPClient(srv srpSrv) ([]byte, error) {
	salt, _ := srv.exchange(p)

	K := sha256.Sum256(big.NewInt(0).Bytes())
	key := K[:]

	h := hmac.New(sha256.New, key)
	h.Write(salt)
	if !srv.verify(h.Sum(nil)) {
		return nil, errors.New("wrong password")
	}
	return key, nil
}

type srpServerMITM struct {
	A, b, B   *big.Int
	salt, mac []byte
}

func (s *srpServerMITM) exchange(A *big.Int) (salt []byte, B *big.Int) {
	s.A = A
	s.salt = make([]byte, 16)
	rand.Read(s.salt)
	dh := &dhParams{p, big2}
	s.b = dh.Private()
	s.B = dh.Public(s.b)
	return s.salt, s.B
}

func (s *srpServerMITM) verify(mac []byte) bool {
	s.mac = mac
	return true
}

func (s *srpServerMITM) tryPassword(password []byte) bool {
	uH := sha256.Sum256(append(s.A.Bytes(), s.B.Bytes()...))
	u := new(big.Int).SetBytes(uH[:])

	xH := sha256.Sum256(append(s.salt, password...))
	x := new(big.Int).SetBytes(xH[:])

	dh := &dhParams{p, big2}
	S := new(big.Int)
	S.Exp(s.B, u, p)
	S.Exp(S, x, p)
	S.Mul(S, dh.Secret(s.b, s.A))
	S.Mod(S, p)

	K := sha256.Sum256(S.Bytes())
	h := hmac.New(sha256.New, K[:])
	h.Write(s.salt)
	return hmac.Equal(h.Sum(nil), s.mac)
}

func rsaGenerate() *rsa.PrivateKey {
	priv := &rsa.PrivateKey{}
	for {
		p, _ := rand.Prime(rand.Reader, 1024)
		q, _ := rand.Prime(rand.Reader, 1024)
		priv.Primes = []*big.Int{p, q}
		priv.N = new(big.Int).Mul(p, q)
		et := new(big.Int).Sub(p, big1)
		et.Mul(et, new(big.Int).Sub(q, big1))
		priv.E = 3
		priv.D = new(big.Int).ModInverse(big3, et)
		if priv.D.Cmp(big1) > 0 {
			break
		}
	}
	return priv
}

func rsaEncrypt(m []byte, pub *rsa.PublicKey) []byte {
	M := new(big.Int).SetBytes(m)
	if M.Cmp(pub.N) >= 0 {
		panic("m is too big")
	}
	return new(big.Int).Exp(M, big.NewInt(int64(pub.E)), pub.N).Bytes()
}

func rsaDecrypt(m []byte, priv *rsa.PrivateKey) []byte {
	M := new(big.Int).SetBytes(m)
	if M.Cmp(priv.N) >= 0 {
		panic("m is too big")
	}
	return new(big.Int).Exp(M, priv.D, priv.N).Bytes()
}

func broadcastRSA(m []byte) map[*rsa.PublicKey][]byte {
	res := make(map[*rsa.PublicKey][]byte)
	for i := 0; i < 3; i++ {
		key := rsaGenerate()
		res[&key.PublicKey] = rsaEncrypt(m, &key.PublicKey)
	}
	return res
}

func crtRSA(cts map[*rsa.PublicKey][]byte) []byte {
	var keys []*rsa.PublicKey
	for k := range cts {
		keys = append(keys, k)
	}
	res := new(big.Int)
	for i, k := range keys {
		x := big.NewInt(1)
		for j := range keys {
			if i == j {
				continue
			}
			x.Mul(x, keys[j].N)
		}
		x.Mul(x, new(big.Int).ModInverse(x, k.N))
		x.Mul(x, new(big.Int).SetBytes(cts[k]))
		res.Add(res, x)
	}
	x := big.NewInt(1)
	for _, k := range keys {
		x.Mul(x, k.N)
	}
	res.Mod(res, x)
	return cubeRoot(res).Bytes()
}

func cubeRoot(cube *big.Int) *big.Int {
	x := new(big.Int).Rsh(cube, uint(cube.BitLen())/3*2)
	if x.Sign() == 0 {
		panic("can't start from 0")
	}
	for {
		d := new(big.Int).Exp(x, big3, nil)
		d.Sub(d, cube)
		d.Div(d, big3)
		d.Div(d, x)
		d.Div(d, x)
		if d.Sign() == 0 {
			break
		}
		x.Sub(x, d)
	}
	for new(big.Int).Exp(x, big3, nil).Cmp(cube) < 0 {
		x.Add(x, big1)
	}
	for new(big.Int).Exp(x, big3, nil).Cmp(cube) > 0 {
		x.Sub(x, big1)
	}
	// Return the cube, rounded down.
	// if new(big.Int).Exp(x, big3, nil).Cmp(cube) != 0 {
	// 	panic("not a cube")
	// }
	return x
}
