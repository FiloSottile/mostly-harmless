# STM TPM ECC chain

A real-world broken chain from
[golang/go#31440](https://github.com/golang/go/issues/31440#issuecomment-3859462518)
(the comment by u1f35c), used by `TestRealChain` and the WebAssembly/browser
harnesses.

In `stmtpmeccint02.pem` the `Issuer` field encodes `O=STMicroelectronics NV` and
`CN=STM TPM ECC Root CA 01` as `UTF8String`, while `stmtpmeccroot01.pem` encodes
the matching `Subject` attributes as `PrintableString`. OpenSSL accepts the
chain; Go's byte-for-byte `Issuer`↔`Subject` comparison rejects it.

The certificates are published by GlobalSign:

- `tpmeccroot.pem` — GlobalSign Trusted Platform Module ECC Root CA
  <https://secure.globalsign.com/cacert/tpmeccroot.crt>
- `stmtpmeccroot01.pem` — STM TPM ECC Root CA 01 (intermediate)
  <https://secure.globalsign.com/cacert/stmtpmeccroot01.crt>
- `stmtpmeccint02.pem` — STM TPM ECC Intermediate CA 02 (the mismatched child)
  <https://secure.globalsign.com/stmtpmeccint02.crt>

`amended.pem` is the golden output of `amendIssuer(stmtpmeccroot01, stmtpmeccint02)`.
Regenerate it with:

    go run . testdata/stm-tpm-ecc/stmtpmeccroot01.pem \
        testdata/stm-tpm-ecc/stmtpmeccint02.pem > testdata/stm-tpm-ecc/amended.pem
