# amend-issuer

This is a simple tool that given an X.509 issuer and a child, produces an
[unsigned RFC 9925][RFC 9925] version of the issuer, using for its `Subject` field
the exact byte-encoding of the child’s `Issuer` field.

This allows working around encoding mismatches that are [not supported by the Go
verifier](https://lobste.rs/s/6a5wiw/fooling_go_s_x_509_certificate#c_2gva79),
[intentionally so](https://github.com/golang/go/issues/31440#issuecomment-4660191943).
Using this tool does not require access to any private keys.

If the issuer is a **root**, the amended unsigned version can be used in place
of or in addition to the original root with equivalent semantics; certificates
that were issued with the differently-encoded `Issuer` will verify against the
amended version. If the issuer is an **intermediate**, the amended version will
not naturally verify against the original root; instead—after manually checking
the intermediate is valid and signed by a trusted root and unrestricted—the
amended version can be included in the trusted root pool, so that leaves with
the differently-encoded `Issuer` will verify against it directly.

This was designed for the Go verifier, but probably works with other stacks, too.

A similar technique can be used to amend other issuer mis-encodings that are
rejected by Go’s crypto/x509 package, or by other X.509 verifiers.

[RFC 9925]: https://www.rfc-editor.org/rfc/rfc9925.html

## Web tool

**[Run the tool in your browser](https://htmlpreview.github.io/?https://github.com/FiloSottile/mostly-harmless/blob/main/amend-issuer/index.html)**
(rendered from this repository through htmlpreview.github.io).

The page can also be served as static files from any directory (e.g.
`python3 -m http.server`).

## Command line

The same logic is available as a CLI:

    go install github.com/FiloSottile/mostly-harmless/amend-issuer@latest
    amend-issuer issuer.pem child.pem > amended.pem

## Example

A real-world case from [golang/go#31440] is the STM TPM ECC chain, used for TPM
endorsement key attestation, whose certificates are published by GlobalSign:

    GlobalSign Trusted Platform Module ECC Root CA   (trusted root)
    └─ STM TPM ECC Root CA 01                        (cross-signed root)
       └─ STM TPM ECC Intermediate CA 02             (intermediate)
          └─ TPM EK certificates                     (leaf)

“STM TPM ECC Intermediate CA 02” encodes the `O` and `CN` attributes of its
`Issuer` as `UTF8String`, while “STM TPM ECC Root CA 01” encodes the matching
`Subject` attributes as `PrintableString`. OpenSSL accepts the chain:

    $ openssl verify -CAfile tpmeccroot.pem -untrusted stmtpmeccroot01.pem stmtpmeccint02.pem
    stmtpmeccint02.pem: OK

Go rejects it, because it compares the encoded `Issuer` and `Subject` bytes:

    x509: certificate signed by unknown authority

Amend the issuer (“STM TPM ECC Root CA 01”) for its child (“STM TPM ECC
Intermediate CA 02”); the amended `Subject` reproduces the child’s `Issuer` bytes:

    $ amend-issuer stmtpmeccroot01.pem stmtpmeccint02.pem > amended-root01.pem

Add the amended root to a `CertPool` and verify a leaf against it:

```go
// amendedRoot01PEM is the output of amend-issuer above.
// It's ok to also include the original root, to verify other chains.
roots := x509.NewCertPool()
roots.AppendCertsFromPEM(amendedRoot01PEM)

// "STM TPM ECC Intermediate CA 02" now chains to the amended root.
intermediates := x509.NewCertPool()
intermediates.AppendCertsFromPEM(int02PEM)

// leaf is a TPM EK certificate issued by Intermediate CA 02.
if _, err := leaf.Verify(x509.VerifyOptions{
	Roots:         roots,
	Intermediates: intermediates,
	KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
}); err != nil {
	log.Fatal(err)
}
```

The certificates are in [`testdata/stm-tpm-ecc`](testdata/stm-tpm-ecc), and
`TestRealChain` runs this before/after verification end to end.

[golang/go#31440]: https://github.com/golang/go/issues/31440
