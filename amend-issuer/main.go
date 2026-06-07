//go:build !(js && wasm)

package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: amend-issuer <issuer.pem> <child.pem>")
		fmt.Fprintln(os.Stderr, "Writes the amended unsigned issuer certificate as PEM to stdout.")
		os.Exit(2)
	}
	issuerPEM, err := os.ReadFile(os.Args[1])
	check(err)
	childPEM, err := os.ReadFile(os.Args[2])
	check(err)
	out, err := amendIssuer(issuerPEM, childPEM)
	check(err)
	os.Stdout.Write(out)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "amend-issuer:", err)
		os.Exit(1)
	}
}
