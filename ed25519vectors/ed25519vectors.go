// Copyright 2021 Google LLC
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd

package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"filippo.io/edwards25519"
)

var I = edwards25519.NewIdentityPoint()

type LowOrderPoint struct {
	*edwards25519.Point
	Order int
}

var LowOrderPoints = []*LowOrderPoint{
	{mustDecodePoint("0000000000000000000000000000000000000000000000000000000000000000"), 4},
	{mustDecodePoint("0000000000000000000000000000000000000000000000000000000000000080"), 4},
	{mustDecodePoint("0100000000000000000000000000000000000000000000000000000000000000"), 1},
	{mustDecodePoint("26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc05"), 8},
	{mustDecodePoint("26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc85"), 8},
	{mustDecodePoint("c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac037a"), 8},
	{mustDecodePoint("c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac03fa"), 8},
	{mustDecodePoint("ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f"), 2},
}

type Vector struct {
	A *Point
	R *Point
	S *Scalar
	M string

	Flags Flag
}

type Point struct{ edwards25519.Point }

func (p *Point) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(p.Bytes())), nil
}

type Scalar struct{ edwards25519.Scalar }

func (p *Scalar) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(p.Bytes())), nil
}

type Flag int

const (
	// LowOrderX is true when X is a low-order point.
	LowOrderR Flag = 1 << iota
	LowOrderA
	// LowOrderComponentX is true when X has a low order component, regardless
	// of whether it also has a prime order component. That is, it's true when
	// the point is not on the prime order subgroup (including the identity).
	LowOrderComponentR
	LowOrderComponentA
	// LowOrderResidue is true when the low order components of R and [k]A don't
	// add up to I. That makes these signatures verify only with the formulae
	// that multiply by the cofactor.
	LowOrderResidue
)

func (s Vector) F(f Flag) bool {
	return s.Flags&f != 0
}

func (s *Vector) SetF(f Flag, b bool) {
	if b {
		s.Flags |= f
	} else {
		s.Flags &= ^f
	}
}

func (f Flag) MarshalJSON() ([]byte, error) {
	var flags []string
	if f&LowOrderR != 0 {
		flags = append(flags, "LowOrderR")
	}
	if f&LowOrderA != 0 {
		flags = append(flags, "LowOrderA")
	}
	if f&LowOrderComponentR != 0 {
		flags = append(flags, "LowOrderComponentR")
	}
	if f&LowOrderComponentA != 0 {
		flags = append(flags, "LowOrderComponentA")
	}
	if f&LowOrderResidue != 0 {
		flags = append(flags, "LowOrderResidue")
	}
	return json.Marshal(flags)
}

func main() {
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "\t")
	e.Encode(GenerateVectors())
}

// If jumbo is set, generate vectors for all k mod 8 values, not just the ones
// that lead to a different low order residue.
const jumbo = false

func GenerateVectors() []Vector {
	// Pick an arbitrary private scalar and compute the public key.
	sBytes := bytes.Repeat([]byte{0x42}, 32)
	s := edwards25519.NewScalar().SetBytesWithClamping(sBytes)
	A := edwards25519.NewIdentityPoint().ScalarBaseMult(s)

	// Pick an arbitrary r (normally derived from message and private key, but
	// that's just a way to make it deterministic and unpredictable).
	rBytes := bytes.Repeat([]byte{0x13, 0x37}, 32)
	r := edwards25519.NewScalar().SetUniformBytes(rBytes)
	R := edwards25519.NewIdentityPoint().ScalarBaseMult(r)

	var vectors []Vector

	addVector := func(lowA, lowR *LowOrderPoint, sZero, rZero bool) {
		ss := edwards25519.NewScalar()
		AA := edwards25519.NewIdentityPoint()
		if !sZero {
			ss.Set(s)
			AA.Set(A)
		}

		rr := edwards25519.NewScalar()
		RR := edwards25519.NewIdentityPoint()
		if !rZero {
			rr.Set(r)
			RR.Set(R)
		}

		AA.Add(AA, lowA.Point)
		RR.Add(RR, lowR.Point)

		found := make(map[bool]bool) // LowOrderResidue: true
		for kMod8 := byte(0); kMod8 < 8; kMod8++ {
			message := "use ristretto255"
			k := computeK(AA, RR, message)
			for t := 1; k.Bytes()[0]%8 != kMod8; t++ {
				message = fmt.Sprintf("use ristretto255 %d", t)
				k = computeK(AA, RR, message)
			}

			S := (&edwards25519.Scalar{}).MultiplyAdd(k, ss, rr)

			lowOrderResidue := !lowOrderComponentsAddUpToZero(lowA.Point, lowR.Point, k)
			if !found[lowOrderResidue] || jumbo {
				v := Vector{
					A: &Point{*AA}, R: &Point{*RR}, S: &Scalar{*S}, M: message,
				}
				v.SetF(LowOrderR, rZero)
				v.SetF(LowOrderA, sZero)
				v.SetF(LowOrderComponentR, lowR.Point.Equal(I) != 1)
				v.SetF(LowOrderComponentA, lowA.Point.Equal(I) != 1)
				v.SetF(LowOrderResidue, lowOrderResidue)
				vectors = append(vectors, v)
				found[lowOrderResidue] = true
			}
		}
	}

	for _, lowA := range LowOrderPoints {
		for _, lowR := range LowOrderPoints {
			addVector(lowA, lowR, true, true)
			addVector(lowA, lowR, true, false)
			addVector(lowA, lowR, false, true)
			addVector(lowA, lowR, false, false)
		}
	}
	return vectors
}

func computeK(A, R *edwards25519.Point, message string) *edwards25519.Scalar {
	kh := sha512.New()
	kh.Write(R.Bytes())
	kh.Write(A.Bytes())
	io.WriteString(kh, message)
	hramDigest := make([]byte, 0, sha512.Size)
	hramDigest = kh.Sum(hramDigest)
	return edwards25519.NewScalar().SetUniformBytes(hramDigest)
}

func lowOrderComponentsAddUpToZero(A, R *edwards25519.Point, k *edwards25519.Scalar) bool {
	p := (&edwards25519.Point{}).ScalarMult(k, A)
	return p.Add(p, R).Equal(I) == 1
}

func mustDecodePoint(s string) *edwards25519.Point {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(s + ": " + err.Error())
	}
	p := &edwards25519.Point{}
	if _, err := p.SetBytes(b); err != nil {
		panic(s + ": " + err.Error())
	}
	return p
}
