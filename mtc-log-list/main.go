// Command mtc-log-list is a WebAssembly helper for generating witness-network
// log-list entries for MTC tiled logs.
package main

import (
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"syscall/js"

	"filippo.io/mldsa"
	mldsa_x509 "filippo.io/mldsa/x509"
	"filippo.io/torchwood"
)

func generate(_ js.Value, args []js.Value) any {
	if len(args) != 2 {
		return result("", "internal error: wrong argument count")
	}
	name := strings.TrimSpace(args[0].String())
	key := make([]byte, args[1].Get("byteLength").Int())
	if n := js.CopyBytesToGo(key, args[1]); n != len(key) {
		return result("", "internal error: failed to read key file")
	}
	vkey, err := makeVKey(name, key)
	if err != nil {
		return result("", err.Error())
	}
	return result("vkey "+vkey+"\nqpd 86400\ncontact https://github.com/your-org/your-repo/issues\n", "")
}

func result(output, err string) any {
	return map[string]any{"output": output, "error": err}
}

func makeVKey(name string, input []byte) (string, error) {
	if name == "" {
		return "", errors.New("cosigner name is required")
	}
	der := input
	if block, rest := pem.Decode(input); block != nil {
		if len(strings.TrimSpace(string(rest))) != 0 {
			return "", errors.New("PEM contains trailing data")
		}
		if block.Type != "PUBLIC KEY" {
			return "", fmt.Errorf("expected a PUBLIC KEY PEM block, got %q", block.Type)
		}
		der = block.Bytes
	} else if text := strings.Join(strings.Fields(string(input)), ""); text != "" {
		if decoded, err := base64.StdEncoding.DecodeString(text); err == nil {
			der = decoded
		}
	}
	var pk *mldsa.PublicKey
	if key, err := mldsa_x509.ParsePKIXPublicKey(der); err == nil {
		var ok bool
		pk, ok = key.(*mldsa.PublicKey)
		if !ok || pk.Parameters() != mldsa.MLDSA44() {
			return "", fmt.Errorf("expected an ML-DSA-44 public key, got %T", key)
		}
	} else {
		pk, err = mldsa.NewPublicKey(mldsa.MLDSA44(), der)
		if err != nil {
			return "", fmt.Errorf("parse ML-DSA-44 public key as SubjectPublicKeyInfo or raw key: %w", err)
		}
	}
	v, err := torchwood.NewCosignatureVerifierFromKey(name, pk)
	if err != nil {
		return "", fmt.Errorf("make verifier key: %w", err)
	}
	return v.String(), nil
}

func main() {
	js.Global().Set("generateLogListEntry", js.FuncOf(generate))
	select {}
}
