package vrf

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	r255 "github.com/gtank/ristretto255"
)

type TestVector struct {
	sk          []byte
	pk          []byte
	alpha       []byte
	hash_string []byte
	h           []byte
	k_string    []byte
	k           []byte
	g           []byte // gamma = x*H
	u           []byte // k*B
	v           []byte // k*H
	c_string    []byte
	c           []byte
	s           []byte
	pi          []byte
	beta        []byte
}

func hd(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("hex.DecodeString failed: %v", err)
	}
	return b
}

func checkIsEqualElement(t *testing.T, name string, expected []byte, actual *r255.Element) {
	t.Helper()
	e, err := r255.NewIdentityElement().SetCanonicalBytes(expected)
	if err != nil {
		t.Fatalf("newElement failed: %v", err)
	}
	if actual.Equal(e) != 1 {
		t.Fatalf("%s: elements not equal: e=%+v, a=%+v", name, e, actual)
	}
}

func checkIsEqualScalar(t *testing.T, name string, expected []byte, actual *r255.Scalar) {
	t.Helper()
	e, err := r255.NewScalar().SetCanonicalBytes(expected)
	if err != nil {
		t.Fatalf("newScalar failed: %v", err)
	}
	if actual.Equal(e) != 1 {
		t.Fatalf("%s: scalars not equal: e=%+v, a=%+v", name, e, actual)
	}
}

func TestECVRFRoundTrip(t *testing.T) {
	alpha := []byte("af82")
	sk, err := NewPrivateKey(hd(t, "3431c2b03533e280b23232e280b34e2c3132c2b03238e280b23131e280b34500"))
	if err != nil {
		t.Fatalf("NewPrivateKey failed: %v", err)
	}

	pi := sk.Prove(alpha)

	// roundtrip proof
	p2, err := NewProof(pi.Bytes())
	if err != nil {
		t.Fatalf("NewProof failed: %v", err)
	}
	if !bytes.Equal(pi.Bytes(), p2.Bytes()) {
		t.Fatalf("proof roundtrip failed: original != decoded")
	}

	beta := pi.Hash()

	beta2, err := sk.y.Verify(pi, alpha)
	if err != nil {
		t.Fatalf("sk.Y.Verify failed: %v", err)
	}
	if !bytes.Equal(beta2, beta) {
		t.Fatalf("beta mismatch: %x != %x", beta2, beta)
	}
}

func TestECVRFRISTRETTO255SHA512(t *testing.T) {
	// test vector is from https://github.com/C2SP/C2SP/blob/main/vrf-r255.md
	tests := []TestVector{{
		sk:          hd(t, "3431c2b03533e280b23232e280b34e2c3132c2b03238e280b23131e280b34500"),
		pk:          hd(t, "54136cd90d99fbd1d4e855d9556efea87ba0337f2a6ce22028d0f5726fcb854e"),
		alpha:       hd(t, "633273702e6f72672f7672662d72323535"),
		hash_string: hd(t, "3907ed3453d308b0cb4ae071be7e5a80f7db05f11f5569016e3fa3996f7307821142133d0124fb3774d55ba6ccd14c11f71bf66038ec80b3f9973a1a6d69f5db"),
		h:           hd(t, "f245308737c2a888ba56448c8cdbce9d063b57b147e063ce36c580194ef31a63"),
		k_string:    hd(t, "b5eb28143d9defee6faa0c02ff0168b7ac80ea89fe9362845af15cabd100a91ed6251dfa52be36405576eca4a0970f91225b85c8813206d13bd8b42fd11a00fe"),
		k:           hd(t, "d32fcc5ae91ba05704da9df434f22fd4c2c373fdd8294bbb58bf27292aeec00a"),
		g:           hd(t, "0a97d961262fb549b4175c5117860f42ae44a123f93c476c439eddd1c0cff926"),
		u:           hd(t, "9a30709d72de12d67f7af1cd8695ff16214d2d4600ae5f478873d2e7ed0ece73"),
		v:           hd(t, "5e727d972b11f6490b0b1ba8147775bceb1a2cb523b381fa22d5a5c0e97d4744"),
		c_string:    hd(t, "5c805525233e2284dbed45e593b8eea346184b1548e416a11c85f0091b7dba42c92eaea061d0f3378261fc360f5b3cf793020236a9aaec5bbff84c09c91d0555"),
		c:           hd(t, "5c805525233e2284dbed45e593b8eea3"),
		s:           hd(t, "1d5ca9734d72bcbba9738d5237f955f3b2422351149d1312503b6441a47c940c"),
		pi:          hd(t, "0a97d961262fb549b4175c5117860f42ae44a123f93c476c439eddd1c0cff9265c805525233e2284dbed45e593b8eea31d5ca9734d72bcbba9738d5237f955f3b2422351149d1312503b6441a47c940c"),
		beta:        hd(t, "dd653f0879b48c3ef69e13551239bec4cbcc1c18fe8894de2e9e1c790e18273603bf1c6c25d7a797aeff3c43fd32b974d3fcbd4bcce916007097922a3ea3a794"),
	}}
	for i, tv := range tests {
		t.Run(fmt.Sprintf("test vector %d", i), func(t *testing.T) {
			sk, err := NewPrivateKey(tv.sk)
			if err != nil {
				t.Fatalf("NewPrivateKey failed: %v", err)
			}
			checkIsEqualScalar(t, "sk", tv.sk, sk.x)
			checkIsEqualElement(t, "pk", tv.pk, sk.y.y)
			salt := sk.y.y.Bytes()
			x := toUniformBytes(salt, tv.alpha)
			if !bytes.Equal(tv.hash_string, x) {
				t.Fatalf("hash_string mismatch: %x != %x", tv.hash_string, x)
			}
			h := encodeToCurve(salt, tv.alpha)
			checkIsEqualElement(t, "h", tv.h, h)

			g := r255.NewIdentityElement().ScalarMult(sk.x, h)
			checkIsEqualElement(t, "g", tv.g, g)

			k1, err := r255.NewScalar().SetUniformBytes(tv.k_string)
			if err != nil {
				t.Fatalf("SetUniformBytes failed: %v", err)
			}
			checkIsEqualScalar(t, "k1", tv.k, k1)

			k := sk.generateNonce(tv.h)
			checkIsEqualScalar(t, "k", tv.k, k)

			u := r255.NewIdentityElement().ScalarBaseMult(k)
			checkIsEqualElement(t, "U", tv.u, u)

			v := r255.NewIdentityElement().ScalarMult(k, h)
			checkIsEqualElement(t, "V", tv.v, v)
			c_string := hashToChallenge(sk.y, h, g, u, v)
			if !bytes.Equal(tv.c_string, c_string) {
				t.Fatalf("c_string mismatch: %x != %x", tv.c_string, c_string)
			}
			c := generateChallenge(sk.y, h, g, u, v)
			if !bytes.Equal(tv.c, c.Bytes()[:16]) {
				t.Fatalf("c mismatch: %x != %x", tv.c, c.Bytes()[:16])
			}
			s := r255.NewScalar()
			s = s.Multiply(c, sk.x)
			s = s.Add(k, s)
			checkIsEqualScalar(t, "s", tv.s, s)
			p1 := &Proof{g, c, s}

			if !bytes.Equal(p1.Bytes(), tv.pi) {
				t.Fatalf("pi mismatch: %x != %x", p1.Bytes(), tv.pi)
			}

			// below: almost the same as round trip test
			p2 := sk.Prove(tv.alpha)
			if !bytes.Equal(p2.Bytes(), p1.Bytes()) {
				t.Fatalf("p2 != p1: %x != %x", p2.Bytes(), p1.Bytes())
			}
			if !bytes.Equal(p2.Bytes(), tv.pi) {
				t.Fatalf("p2 != tv.pi: %x != %x", p2.Bytes(), tv.pi)
			}
			beta := p2.Hash()
			if !bytes.Equal(beta, tv.beta) {
				t.Fatalf("beta mismatch: %x != %x", beta, tv.beta)
			}

			beta2, err := sk.y.Verify(p2, tv.alpha)
			if err != nil {
				t.Fatalf("sk.Y.Verify failed: %v", err)
			}
			if !bytes.Equal(beta2, beta) {
				t.Fatalf("beta2 != beta: %x != %x", beta2, beta)
			}
			if !bytes.Equal(beta2, tv.beta) {
				t.Fatalf("beta2 != tv.beta: %x != %x", beta2, tv.beta)
			}
		})
	}
}
