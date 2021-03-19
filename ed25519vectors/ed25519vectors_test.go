// Copyright 2021 Google LLC
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd

package main

import (
	"fmt"
	"testing"

	"filippo.io/edwards25519"
)

var I = edwards25519.NewIdentityPoint()

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
	q := edwards25519.NewIdentityPoint().MultByCofactor(p.Point)
	if q.Equal(I) != 1 {
		t.Errorf("[8]P != ∞")
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
