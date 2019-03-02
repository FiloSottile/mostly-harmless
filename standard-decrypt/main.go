// Copyright 2017 Filippo Valsorda
//
// Permission to use, copy, modify, and/or distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

// Command standard-decrypt decrypts a Standard File backup generated with
// standard-backup or through the Standard Notes application.
package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

func isSupportedVersion(version string) bool {
	switch version {
	case "002", "003":
		return true
	}
	return false
}

type Backup struct {
	Items []struct {
		UUID        string
		ContentType string    `json:"content_type"`
		CreatedAt   time.Time `json:"created_at"`
		EncItemKey  string    `json:"enc_item_key"`
		Content     string    `json:"content"`
		UpdatedAt   time.Time `json:"updated_at"`
		Deleted     bool      `json:"deleted"`
	}
	AuthParams struct {
		Salt    string `json:"pw_salt"`
		Nonce   string `json:"pw_nonce"`
		Email   string `json:"identifier"`
		Cost    int    `json:"pw_cost"`
		Version string `json:"version"`
	} `json:"auth_params"`
}

type Item struct {
	UUID        string          `json:"uuid"`
	ContentType string          `json:"content_type"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Content     json.RawMessage `json:"content"`
}

func decrypt(s, uuid string, ek, ak []byte) ([]byte, error) {
	parts := strings.Split(s, ":")
	if len(parts) < 5 {
		return nil, errors.New("wrong parts length")
	}
	if !isSupportedVersion(parts[0]) {
		return nil, errors.New("wrong version")
	}
	if parts[2] != uuid {
		return nil, errors.New("wrong uuid")
	}
	h := hmac.New(sha256.New, ak)
	h.Write([]byte(strings.Join(append([]string{parts[0]}, parts[2:5]...), ":")))
	if parts[1] != hex.EncodeToString(h.Sum(nil)) {
		return nil, errors.New("wrong hmac")
	}
	ct, err := base64.StdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, err
	}
	c, err := aes.NewCipher(ek)
	if err != nil {
		return nil, err
	}
	res := make([]byte, len(ct))
	iv, err := hex.DecodeString(parts[3])
	if err != nil {
		return nil, err
	}
	cipher.NewCBCDecrypter(c, iv).CryptBlocks(res, ct)
	for i := byte(1); i < res[len(res)-1]; i++ {
		if res[len(res)-int(i)-1] != res[len(res)-1] {
			return nil, errors.New("wrong padding")
		}
	}
	return res[:len(res)-int(res[len(res)-1])], nil
}

func derive(pw, salt string, cost int) ([]byte, []byte, []byte) {
	k := pbkdf2.Key([]byte(pw), []byte(salt), cost, 96, sha512.New)
	return k[:32], k[32:64], k[64:]
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalln("usage: standard-decrypt backup.txt")
	}
	data, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalln("Failed to open backup file:", err)
	}
	var backup *Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		log.Fatalln("Failed to parse backup file:", err)
	}
	if !isSupportedVersion(backup.AuthParams.Version) {
		log.Fatalln("Unsupported version:", backup.AuthParams.Version)
	}

	os.Stderr.WriteString("Password: ")
	s := bufio.NewScanner(os.Stdin)
	s.Scan()

	var salt string
	switch backup.AuthParams.Version {
	case "003":
		h := sha256.New()
		fmt.Fprintf(h, "%s:SF:%s:%d:%s", backup.AuthParams.Email, backup.AuthParams.Version,
			backup.AuthParams.Cost, backup.AuthParams.Nonce)
		salt = hex.EncodeToString(h.Sum(nil))
	case "002":
		salt = backup.AuthParams.Salt
	}
	spw, ek, ak := derive(s.Text(), salt, backup.AuthParams.Cost)
	log.Printf("Server password: %x", spw)

	var res []*Item
	for _, item := range backup.Items {
		if item.Deleted {
			continue
		}

		k, err := decrypt(item.EncItemKey, item.UUID, ek, ak)
		if err != nil {
			log.Fatalf("Failed to decrypt key for item %s: %v", item.UUID, err)
		}

		kk := make([]byte, hex.DecodedLen(len(k)))
		if _, err := hex.Decode(kk, k); err != nil {
			log.Fatalf("Failed to decode key for item %s: %v", item.UUID, err)
		}
		if len(kk) != 64 {
			log.Fatalf("Wrong key length for item %s: %v", item.UUID, len(kk))
		}
		content, err := decrypt(item.Content, item.UUID, kk[:32], kk[32:])
		if err != nil {
			log.Fatalf("Failed to decrypt item %s: %v", item.UUID, err)
		}
		res = append(res, &Item{
			UUID:        item.UUID,
			ContentType: item.ContentType,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
			Content:     content,
		})
	}

	json.NewEncoder(os.Stdout).Encode(res)
}
