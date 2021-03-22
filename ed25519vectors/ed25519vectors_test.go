// Copyright 2021 Google LLC
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd

package main

import (
	"crypto/ed25519"
	"fmt"
	"math/rand"
	"testing"

	"filippo.io/edwards25519"
	"github.com/hdevalence/ed25519consensus"
)

func TestLowOrderPoints(t *testing.T) {
	for i, p := range LowOrderPoints {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			testLowOrderPoint(t, p)
		})
		for j := i + 1; j < len(LowOrderPoints); j++ {
			if p.Point.Equal(LowOrderPoints[j].Point) == 1 {
				t.Errorf("#%d == #%d", i, j)
			}
		}
	}
}

func knownLowOrderPoint(p *edwards25519.Point) bool {
	for _, lp := range LowOrderPoints {
		if lp.Equal(p) == 1 {
			return true
		}
	}
	return false
}

func testLowOrderPoint(t *testing.T, p *LowOrderPoint) {
	q := (&edwards25519.Point{}).MultByCofactor(p.Point)
	if q.Equal(I) != 1 {
		t.Errorf("[8]P != I")
	}

	q.Set(p.Point)
	for i := 1; i <= p.Order; i++ {
		if !knownLowOrderPoint(q) {
			t.Errorf("[%d]P not in known LowOrderPoints: %x", i, q.Bytes())
		}
		if q.Equal(p.Point) == 1 && i != 1 {
			t.Errorf("[%d]P == P, but %d <= Order", i, i)
		}
		q.Add(q, p.Point)
	}
	if q.Equal(p.Point) != 1 {
		t.Errorf("[Order + 1]P != P")
	}
}

func TestVectors(t *testing.T) {
	vectors := GenerateVectors()

	if exp := 8 * 8 * 2 * 2; exp > len(vectors) || len(vectors) > exp*2 {
		t.Errorf("expected %d to %d vectors, got %d", exp, exp*2, len(vectors))
	}

	for i, v := range vectors {
		eightA := (&edwards25519.Point{}).MultByCofactor(&v.A.Point)
		if v.F(LowOrderA) {
			if eightA.Equal(I) != 1 {
				t.Errorf("#%d: LowOrderA is true but [8]A != I", i)
			}
		} else {
			if eightA.Equal(I) == 1 {
				t.Errorf("#%d: LowOrderA is false but [8]A == I", i)
			}
		}

		eightR := (&edwards25519.Point{}).MultByCofactor(&v.R.Point)
		if v.F(LowOrderR) {
			if eightR.Equal(I) != 1 {
				t.Errorf("#%d: LowOrderR is true but [8]R != I", i)
			}
		} else {
			if eightR.Equal(I) == 1 {
				t.Errorf("#%d: LowOrderR is false but [8]R == I", i)
			}
		}

		lA := multByPrimeOrder(&v.A.Point)
		if v.F(LowOrderComponentA) {
			if lA.Equal(I) == 1 {
				t.Errorf("#%d: LowOrderComponentA is true but [l]A == I", i)
			}
		} else {
			if lA.Equal(I) != 1 {
				t.Errorf("#%d: LowOrderComponentA is false but [l]A != I", i)
			}
		}

		lR := multByPrimeOrder(&v.R.Point)
		if v.F(LowOrderComponentR) {
			if lR.Equal(I) == 1 {
				t.Errorf("#%d: LowOrderComponentR is true but [l]R == I", i)
			}
		} else {
			if lR.Equal(I) != 1 {
				t.Errorf("#%d: LowOrderComponentR is false but [l]R != I", i)
			}
		}

		if !v.F(LowOrderComponentA) && !v.F(LowOrderComponentR) && v.F(LowOrderResidue) {
			t.Errorf("#%d: there are no low order components but LowOrderResidue is true", i)
		}

		publicKey := v.A.Bytes()
		message := []byte(v.M)
		signature := append(v.R.Bytes(), v.S.Bytes()...)

		if !ed25519consensus.Verify(publicKey, message, signature) {
			t.Errorf("#%d: ZIP215 rejected signature", i)
		}

		if !v.F(LowOrderResidue) {
			if !ed25519.Verify(publicKey, message, signature) {
				t.Errorf("#%d: crypto/ed25519 rejected signature with no low order residue", i)
			}
		} else {
			if ed25519.Verify(publicKey, message, signature) {
				t.Errorf("#%d: crypto/ed25519 accepted signature with low order residue", i)
			}
		}
	}
}

func TestMultByPrimeOrder(t *testing.T) {
	b := make([]byte, 64)
	rand.Read(b)
	s := (&edwards25519.Scalar{}).SetUniformBytes(b)
	p := (&edwards25519.Point{}).ScalarBaseMult(s)
	if multByPrimeOrder(p).Equal(I) != 1 {
		t.Fail()
	}
}

var pMinusOne, _ = (&edwards25519.Scalar{}).SetCanonicalBytes([]byte{236, 211, 245, 92, 26, 99, 18, 88, 214, 156, 247, 162, 222, 249, 222, 20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 16})

func multByPrimeOrder(p *edwards25519.Point) *edwards25519.Point {
	q := &edwards25519.Point{}
	return q.ScalarMult(pMinusOne, p).Add(q, p)
}
