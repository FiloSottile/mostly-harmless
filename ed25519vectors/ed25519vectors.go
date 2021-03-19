// Copyright 2021 Google LLC
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd

package main

import (
	"encoding/hex"

	"filippo.io/edwards25519"
)

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
