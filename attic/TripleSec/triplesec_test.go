// The design and name of TripleSec is (C) Keybase 2013
// This Go implementation is (C) Filippo Valsorda 2014
// Use of this source code is governed by the MIT License

package triplesec

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestCycle(t *testing.T) {
	plaintext := []byte("1234567890-")
	password := []byte("42")

	c := NewCipher(password)

	orig_plaintext := append([]byte{}, plaintext...)
	ciphertext := make([]byte, len(plaintext)+Overhead)
	err := c.Encrypt(ciphertext, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	orig_ciphertext := append([]byte{}, ciphertext...)
	new_plaintext := make([]byte, len(ciphertext)-Overhead)
	err = c.Decrypt(new_plaintext, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(new_plaintext, plaintext) {
		t.Error("new_plaintext != plaintext")
	}
	if !bytes.Equal(orig_plaintext, plaintext) {
		t.Error("orig_plaintext != plaintext")
	}
	if !bytes.Equal(orig_ciphertext, ciphertext) {
		t.Error("orig_ciphertext != ciphertext")
	}
	if !bytes.Equal(password, []byte("42")) {
		t.Error("password changed")
	}
}

func BenchmarkEncrypt(b *testing.B) {
	plaintext := []byte("1234567890-")
	password := []byte("42")

	c := NewCipher(password)
	ciphertext := make([]byte, len(plaintext)+Overhead)

	for i := 0; i < b.N; i++ {
		err := c.Encrypt(ciphertext, plaintext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecrypt(b *testing.B) {
	plaintext := []byte("1234567890-")
	password := []byte("42")

	c := NewCipher(password)
	ciphertext := make([]byte, len(plaintext)+Overhead)

	err := c.Encrypt(ciphertext, plaintext)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = c.Decrypt(plaintext, ciphertext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestBiggerBufSizes(t *testing.T) {
	// TODO: should we resize the buffers when we are passed smaller ones?

	plaintext := []byte("1234567890-")
	password := []byte("42")

	c := NewCipher(password)

	orig_plaintext := append([]byte{}, plaintext...)
	ciphertext := make([]byte, len(plaintext)+Overhead+100)
	err := c.Encrypt(ciphertext, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	orig_ciphertext := append([]byte{}, ciphertext...)
	new_plaintext := make([]byte, len(ciphertext[:len(plaintext)+Overhead])-Overhead+100)
	err = c.Decrypt(new_plaintext, ciphertext[:len(plaintext)+Overhead])
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(new_plaintext[:len(plaintext)], plaintext) {
		t.Error("new_plaintext != plaintext")
	}
	if !bytes.Equal(orig_plaintext, plaintext) {
		t.Error("orig_plaintext != plaintext")
	}
	if !bytes.Equal(orig_ciphertext, ciphertext) {
		t.Error("orig_ciphertext != ciphertext")
	}
	if !bytes.Equal(password, []byte("42")) {
		t.Error("password changed")
	}
}

func TestSmallerBufSizes(t *testing.T) {
	plaintext := []byte("1234567890-")
	password := []byte("42")

	c := NewCipher(password)

	orig_plaintext := append([]byte{}, plaintext...)
	ciphertext := make([]byte, len(plaintext)+Overhead)
	err := c.Encrypt(ciphertext, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	orig_ciphertext := append([]byte{}, ciphertext...)
	new_plaintext := make([]byte, len(ciphertext)-Overhead)
	err = c.Decrypt(new_plaintext, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(new_plaintext, plaintext) {
		t.Error("new_plaintext != plaintext")
	}
	if !bytes.Equal(orig_plaintext, plaintext) {
		t.Error("orig_plaintext != plaintext")
	}
	if !bytes.Equal(orig_ciphertext, ciphertext) {
		t.Error("orig_ciphertext != ciphertext")
	}
	if !bytes.Equal(password, []byte("42")) {
		t.Error("password changed")
	}
}

func TestVector(t *testing.T) {
	ciphertext, _ := hex.DecodeString("1c94d7de0000000359a5e5d60f09ebb6bc3fdab6642725e03bc3d51e167fa60327df567476d467f8b6ce65a909b4f582443f230ff10a36f60315ebce1cf1395d7b763c768764207f4f4cc5207a21272f3a5542f35db73c94fbc7bd551d4d6b0733e0b27fdf9606b8a26d45c4b79818791b6ae1ad34c23e58de482d454895618a1528ec722c5218650f8a2f55f63a6066ccf875f46c9b68ed31bc1ddce8881d704be597e1b5006d16ebe091a02e24d569f3d09b0578d12f955543e1a1f1dd75784b8b4cba7ca0bb7044389eb6354cea628a21538d")
	buf := make([]byte, len(ciphertext)-Overhead)
	NewCipher([]byte("42")).Decrypt(buf, ciphertext)
	if !bytes.Equal(buf, []byte("ciao")) {
		t.Fail()
	}
}
