package cryptopals

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"math/big"
	"os"
)

func decryptRSAOnceOracle() (
	pub *rsa.PublicKey,
	encrypt func([]byte) []byte,
	decrypt func([]byte) []byte,
) {
	key := rsaGenerate()
	pub = &key.PublicKey
	encrypt = func(m []byte) []byte {
		return rsaEncrypt(m, pub)
	}
	seenMessages := make(map[[sha256.Size]byte]struct{})
	decrypt = func(m []byte) []byte {
		h := sha256.Sum256(m)
		if _, ok := seenMessages[h]; ok {
			return nil
		}
		seenMessages[h] = struct{}{}
		return rsaDecrypt(m, key)
	}
	return
}

func decryptRSAAgain(c []byte, pub *rsa.PublicKey, decrypt func([]byte) []byte) []byte {
	C := new(big.Int).SetBytes(c)
	s, err := rand.Int(rand.Reader, pub.N)
	if err != nil {
		panic(err)
	}
	S := new(big.Int).Exp(s, big.NewInt(int64(pub.E)), pub.N)
	C1 := new(big.Int)
	C1.Mul(C, S).Mod(C1, pub.N)
	p1 := new(big.Int).SetBytes(decrypt(C1.Bytes()))
	p := new(big.Int).Mul(p1, new(big.Int).ModInverse(s, pub.N))
	return p.Mod(p, pub.N).Bytes()
}

func bb06Oracle() (verify func(msg, sig []byte) bool) {
	key := rsaGenerate()
	verify = func(msg, sig []byte) bool {
		asn1 := []byte{0x30, 0x20, 0x30, 0x0c, 0x06, 0x08, 0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x02, 0x05, 0x05, 0x00, 0x04, 0x10}
		s := rsaEncrypt(sig, &key.PublicKey)
		if s[0] != 0x01 {
			return false
		}
		s = s[1:]
		for s[0] == 0xff {
			s = s[1:]
		}
		if s[0] != 0x00 {
			return false
		}
		s = s[1:]
		if !bytes.Equal(s[:len(asn1)], asn1) {
			return false
		}
		s = s[len(asn1):]
		h := md5.Sum(msg)
		return bytes.Equal(s[:len(h)], h[:])
	}
	return
}

func bb06Forgery(msg []byte) []byte {
	keySize := 1024 / 8
	target := make([]byte, keySize)
	for i := range target {
		target[i] = 0xff
	}
	target = target[:0]
	target = append(target, 0x00)
	target = append(target, 0x01)
	target = append(target, 0x00)
	target = append(target, 0x30, 0x20, 0x30, 0x0c, 0x06, 0x08, 0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x02, 0x05, 0x05, 0x00, 0x04, 0x10)
	h := md5.Sum(msg)
	target = append(target, h[:]...)
	target = target[:cap(target)]
	// log.Printf("%x", target)
	t := new(big.Int).SetBytes(target)
	root := cubeRoot(t)
	// log.Printf("%x", new(big.Int).Exp(root, big3, nil))
	return root.Bytes()
}

type dsaParams struct {
	P, Q, G *big.Int
}

var cryptopalsDSAParams dsaParams

func init() {
	cryptopalsDSAParams.P, _ = new(big.Int).SetString("800000000000000089e1855218a0e7dac38136ffafa72eda7859f2171e25e65eac698c1702578b07dc2a1076da241c76c62d374d8389ea5aeffd3226a0530cc565f3bf6b50929139ebeac04f48c3c84afb796d61e5a4f9a8fda812ab59494232c7d2b4deb50aa18ee9e132bfa85ac4374d7f9091abc3d015efc871a584471bb1", 16)
	cryptopalsDSAParams.Q, _ = new(big.Int).SetString("f4f47f05794b256174bba6e9b396a7707e563c5b", 16)
	cryptopalsDSAParams.G, _ = new(big.Int).SetString("5958c9d3898b224b12672c0b98e06c60df923cb8bc999d119458fef538b8fa4046c8db53039db620c094c9fa077ef389b5322a559946a71903f990f1f7e0e025e2d7f7cf494aff1a0470f5b64c36b625a097f1651fe775323556fe00b3608c887892878480e99041be601a62166ca6894bdd41a7054ec89f756ba9fc95302291", 16)
}

func (dsa dsaParams) Generate() (x, y *big.Int) {
	x, _ = rand.Int(rand.Reader, dsa.Q)
	y = new(big.Int).Exp(dsa.G, x, dsa.P)
	return
}

func (dsa dsaParams) Sign(x *big.Int, msg []byte) (r, s *big.Int) {
	r, s = new(big.Int), new(big.Int)
	k, _ := rand.Int(rand.Reader, dsa.Q)
	r.Exp(dsa.G, k, dsa.P).Mod(r, dsa.Q)
	h := sha1.Sum(msg)
	H := new(big.Int).SetBytes(h[:])
	kInv := new(big.Int).ModInverse(k, dsa.Q)
	s.Mul(x, r).Add(s, H).Mul(s, kInv).Mod(s, dsa.Q)
	return
}

func (dsa dsaParams) Verify(y, r, s *big.Int, msg []byte) bool {
	w := new(big.Int).ModInverse(s, dsa.Q)
	h := sha1.Sum(msg)
	H := new(big.Int).SetBytes(h[:])
	u1, u2, v := new(big.Int), new(big.Int), new(big.Int)
	u1.Mul(H, w).Mod(u1, dsa.Q)
	u2.Mul(r, w).Mod(u2, dsa.Q)
	u1.Exp(dsa.G, u1, dsa.P)
	u2.Exp(y, u2, dsa.P)
	v.Mul(u1, u2).Mod(v, dsa.P).Mod(v, dsa.Q)
	return v.Cmp(r) == 0
}

func recoverDSAKeyFromLowK(dsa dsaParams, y, r, s *big.Int, msg []byte) *big.Int {
	h := sha1.Sum(msg)
	H := new(big.Int).SetBytes(h[:])
	rInv := new(big.Int).ModInverse(r, dsa.Q)

	k := big.NewInt(1)
	rr := new(big.Int).Set(dsa.G)
	rrModQ := new(big.Int).Mod(rr, dsa.Q)
	for k.BitLen() <= 16 {
		if rrModQ.Mod(rr, dsa.Q).Cmp(r) == 0 {
			x := new(big.Int).Mul(s, k)
			x.Mod(x, dsa.Q).Sub(x, H).Mul(x, rInv).Mod(x, dsa.Q)
			return x
		}
		rr.Mul(rr, dsa.G).Mod(rr, dsa.P)
		k.Add(k, big1)
	}

	// Old, slower method.
	yy := new(big.Int)
	for k.BitLen() <= 16 {
		x := new(big.Int).Mul(s, k)
		x.Mod(x, dsa.Q).Sub(x, H).Mul(x, rInv).Mod(x, dsa.Q)

		if yy.Exp(dsa.G, x, dsa.P).Cmp(y) == 0 {
			return x
		}

		k.Add(k, big1)
	}

	panic("key not found")
}

func recoverDSAKeyFromRepeatedK(dsa dsaParams, r, s1, s2 *big.Int, msg1, msg2 []byte) *big.Int {
	k := new(big.Int).Sub(s1, s2)
	k.Mod(k, dsa.Q)
	k.ModInverse(k, dsa.Q)

	h1, h2 := sha1.Sum(msg1), sha1.Sum(msg2)
	H1 := new(big.Int).SetBytes(h1[:])
	H := new(big.Int).SetBytes(h2[:])
	H.Sub(H1, H).Mod(H, dsa.Q)

	k.Mul(k, H).Mod(k, dsa.Q)

	rInv := new(big.Int).ModInverse(r, dsa.Q)
	x := new(big.Int).Mul(s1, k)
	x.Mod(x, dsa.Q).Sub(x, H1).Mul(x, rInv).Mod(x, dsa.Q)
	return x
}

func magicDSAg1Signature(dsa dsaParams, y *big.Int) (r, s *big.Int) {
	z, _ := rand.Int(rand.Reader, dsa.Q)
	r = new(big.Int).Exp(y, z, dsa.P)
	r.Mod(r, dsa.Q)
	s = new(big.Int).ModInverse(z, dsa.Q)
	s.Mul(s, r).Mod(s, dsa.Q)
	return
}

func rsaParityOracle() (
	pub *rsa.PublicKey,
	encrypt func([]byte) []byte,
	isPlaintextEven func([]byte) bool,
) {
	key := rsaGenerate()
	pub = &key.PublicKey
	encrypt = func(m []byte) []byte {
		return rsaEncrypt(m, pub)
	}
	isPlaintextEven = func(m []byte) bool {
		p := rsaDecrypt(m, key)
		return p[len(p)-1]%2 == 0
	}
	return
}

func attackRSAParityOracle(pub *rsa.PublicKey, c []byte, isPlaintextEven func([]byte) bool) []byte {
	lower, upper := big.NewInt(0), new(big.Int).Set(pub.N)
	e := big.NewInt(1)
	C := new(big.Int).SetBytes(c)
	two := new(big.Int).Exp(big2, big.NewInt(int64(pub.E)), pub.N)
	for {
		e.Mul(e, big2)
		C.Mul(C, two).Mod(C, pub.N) // vet false positive, TODO check with 1.10!

		diff := new(big.Int).Div(pub.N, e) // done anew not to accumulate error
		if isPlaintextEven(C.Bytes()) {
			upper.Sub(upper, diff) // lower half
		} else {
			lower.Add(lower, diff) // upper half
		}
		fmt.Fprintf(os.Stderr, "%q\n", lower.Bytes())

		if lower.Cmp(upper) == 0 || diff.Sign() == 0 {
			return lower.Bytes()
		}
	}
}

func padRSA(m []byte, pub *rsa.PublicKey) []byte {
	res := make([]byte, (pub.N.BitLen()+8-1)/8)
	res[0] = 0
	res[1] = 2
	// TODO: missing non-zero bytes
	if len(res)-len(m) < 2 {
		panic("m is too long")
	}
	copy(res[len(res)-len(m):], m)
	return res
}

func rsaPKCS1Oracle(keySize int) (
	pub *rsa.PublicKey,
	encrypt func([]byte) []byte,
	isPaddingValid func([]byte) bool,
) {
	key := &rsa.PrivateKey{}
	for {
		p, _ := rand.Prime(rand.Reader, keySize/2)
		q, _ := rand.Prime(rand.Reader, keySize/2)
		key.Primes = []*big.Int{p, q}
		key.N = new(big.Int).Mul(p, q)
		et := new(big.Int).Sub(p, big1)
		et.Mul(et, new(big.Int).Sub(q, big1))
		key.E = 3
		key.D = new(big.Int).ModInverse(big3, et)
		if key.D.Cmp(big1) > 0 {
			break
		}
	}

	pub = &key.PublicKey
	encrypt = func(m []byte) []byte {
		return rsaEncrypt(padRSA(m, pub), pub)
	}
	isPaddingValid = func(m []byte) bool {
		p := rsaDecrypt(m, key)
		pp := make([]byte, (pub.N.BitLen()+8-1)/8)
		copy(pp[len(pp)-len(p):], p)
		return pp[0] == 0x00 && pp[1] == 0x02
	}
	return
}

func divRoundUp(res, x, y *big.Int) {
	m := new(big.Int) // TODO: optimize out if necessary
	res.DivMod(x, y, m)
	if m.Sign() > 0 {
		res.Add(res, big1)
	}
}

func attackBB98(pub *rsa.PublicKey, cc []byte, isPaddingValid func([]byte) bool) []byte {
	if !isPaddingValid(cc) {
		panic("must start from a valid ciphertext")
	}
	if pub.N.BitLen()%8 != 0 {
		panic("key length must be a multiple of 8")
	}

	c := new(big.Int).SetBytes(cc)
	e := big.NewInt(int64(pub.E))
	tryS := func(s *big.Int) bool {
		c1 := new(big.Int).Exp(s, e, pub.N)
		c1.Mul(c1, c).Mod(c1, pub.N)
		return isPaddingValid(c1.Bytes())
	}

	B := new(big.Int).Lsh(big1, uint(pub.N.BitLen()-16))
	B2, B3 := new(big.Int).Mul(B, big2), new(big.Int).Mul(B, big3)
	lower, upper := new(big.Int).Set(B2), new(big.Int).Sub(B3, big1)

	s, r, r1 := new(big.Int), new(big.Int), new(big.Int)
	a, b, maxS := new(big.Int), new(big.Int), new(big.Int)
	i := 1
	for {
		if i == 1 {
			// n / 3B
			divRoundUp(s, pub.N, B3)
			for !tryS(s) {
				s.Add(s, big1)
			}
		} else {
			// 2 * ( bs − 2B ) / N
			divRoundUp(r, r.Mul(upper, s).Sub(r, B2).Mul(r, big2), pub.N)
		searchLoop:
			for {
				// ( 2B + rn ) / b
				divRoundUp(s, s.Mul(r, pub.N).Add(s, B2), upper)
				// ( 3B + rn ) / a
				divRoundUp(maxS, maxS.Mul(r, pub.N).Add(maxS, B3), lower)
				for s.Cmp(maxS) < 0 {
					if tryS(s) {
						break searchLoop
					}
					s.Add(s, big1)
				}
				r.Add(r, big1)
			}
		}

		// ( as − 3B + 1 ) / n
		r.Mul(lower, s).Sub(r, B3).Add(r, big1)
		divRoundUp(r, r, pub.N)
		// ( bs − 2B ) / n
		r1.Mul(upper, s).Sub(r1, B2).Div(r1, pub.N)

		if r.Cmp(r1) != 0 {
			panic("multiple ranges unimplemented")
		}

		// ( 2B + rn ) / s <- round up
		divRoundUp(a, a.Mul(r, pub.N).Add(a, B2), s)
		if a.Cmp(lower) > 0 {
			lower.Set(a)
		}
		// ( 3B − 1 + rn ) / s <- round down
		b.Mul(r, pub.N).Add(b, B3).Sub(b, big1).Div(b, s)
		if b.Cmp(upper) < 0 {
			upper.Set(b)
		}

		fmt.Fprintf(os.Stderr, "%q\n", lower.Bytes())
		if upper.Cmp(lower) == 0 {
			break
		}

		i++
	}

	return lower.Bytes()
}
