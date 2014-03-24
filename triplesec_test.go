// The design and name of TripleSec is (C) Keybase 2013
// This Go implementation is (C) Filippo Valsorda 2014
// Use of this source code is governed by the MIT License

package triplesec

import (
	"bytes"
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
