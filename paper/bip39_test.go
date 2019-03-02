package main

import (
	"bytes"
	crnd "crypto/rand"
	"encoding/hex"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func TestDivideUp(t *testing.T) {
	if divideUp(10, 2) != 5 {
		t.Fail()
	}
	if divideUp(9, 2) != 5 {
		t.Fail()
	}
	if divideUp(10, 5) != 2 {
		t.Fail()
	}
	if divideUp(8, 5) != 2 {
		t.Fail()
	}
}

func TestRandomBip39(t *testing.T) {
	for i := 1; i < 1000; i++ {
		num := rand.Intn(1000)
		data := make([]byte, num)
		if _, err := crnd.Read(data); err != nil {
			t.Fatal(err)
		}
		if len(data) > 0 && data[0] == 0 {
			// The encoding can't reasonably preserve leading zeroes
			continue
		}
		mnemonic := Bip39Encode(data)
		res, c, w := Bip39Decode(mnemonic)
		if c != nil {
			t.Fatal(c)
		}
		if w != nil {
			t.Fatal(w)
		}
		if !bytes.Equal(res, data) {
			t.Log("\n" + hex.Dump(data))
			t.Log("\n" + hex.Dump(res))
			t.Fail()
		}
	}
}

func TestBip39Mistakes(t *testing.T) {
	goodData, c, w := Bip39Decode([]string{"average", "evidence", "garage"})
	if c != nil {
		t.Fatal(c)
	}
	if w != nil {
		t.Fatal(w)
	}

	data, c, w := Bip39Decode([]string{"average", "evidente", "garage"})
	if !bytes.Equal(data, goodData) {
		t.Log("\n" + hex.Dump(data))
		t.Log("\n" + hex.Dump(goodData))
		t.Fail()
	}
	if !reflect.DeepEqual(c, []string{"evidente -> evidence"}) {
		t.Fatal(c)
	}
	if w != nil {
		t.Fatal(w)
	}

	data, c, w = Bip39Decode([]string{"average", "evidente", "garale"})
	if !bytes.Equal(data, goodData) {
		t.Log("\n" + hex.Dump(data))
		t.Log("\n" + hex.Dump(goodData))
		t.Fail()
	}
	if !reflect.DeepEqual(c, []string{
		"evidente -> evidence", "garale -> garage"}) {
		t.Fatal(c)
	}
	if w != nil {
		t.Fatal(w)
	}

	data, c, w = Bip39Decode([]string{"average", "elidente", "garale"})
	if data != nil {
		t.Fatal(data)
	}
	if !reflect.DeepEqual(c, []string{"garale -> garage"}) {
		t.Fatal(c)
	}
	if !reflect.DeepEqual(w, []string{"elidente"}) {
		t.Fatal(c)
	}
}
