package triplesec

import (
	"encoding/hex"
	"testing"
)

func Test(t *testing.T) {
	plaintext := []byte("1234567890-")
	ciphertext := make([]byte, len(plaintext)+Overhead)
	c, err := NewCipher([]byte("42"))
	if err != nil {
		panic(err)
	}
	err = c.Encrypt(ciphertext, plaintext)
	if err != nil {
		panic(err)
	}
	println(hex.EncodeToString(ciphertext))
}
