# MTC log-list entry generator

A client-side tool for generating entries in the
[witness-network log-list format](https://witness-network.org/log-list-format)
for issuance logs following [c2sp.org/mtc-tlog](https://c2sp.org/mtc-tlog).

The key input is an ML-DSA-44 public key, either in its raw encoding or encoded
as a PKIX SubjectPublicKeyInfo in PEM (`PUBLIC KEY`) or DER form. It can be
uploaded as a file, or pasted as PEM or base64-encoded raw/DER bytes. The verifier key
is generated in Go WebAssembly with [Torchwood](https://filippo.io/torchwood).

To rebuild:

```console
GOOS=js GOARCH=wasm go build -o main.wasm .
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" .
```
