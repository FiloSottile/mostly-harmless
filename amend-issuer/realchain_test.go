package main

import (
	"bytes"
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRealChain exercises a real-world broken chain published in golang/go#31440:
// the STM TPM ECC chain, where "STM TPM ECC Intermediate CA 02" carries an
// Issuer encoded with UTF8String while its issuer "STM TPM ECC Root CA 01"
// encodes the matching Subject attributes as PrintableString. OpenSSL accepts
// the chain; Go's byte-for-byte comparison rejects it.
//
// Certificates are published by GlobalSign; see testdata/stm-tpm-ecc/README.md.
func TestRealChain(t *testing.T) {
	dir := "testdata/stm-tpm-ecc"
	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	parse := func(name string) *x509.Certificate {
		der, err := decodeCertificate(read(name))
		if err != nil {
			t.Fatal(err)
		}
		c, err := x509.ParseCertificate(der)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}

	globalSignRoot := parse("tpmeccroot.pem")
	root01 := parse("stmtpmeccroot01.pem")
	int02 := parse("stmtpmeccint02.pem")

	// The encoding mismatch: same logical name, different DER bytes.
	if bytes.Equal(int02.RawIssuer, root01.RawSubject) {
		t.Fatal("expected int02 Issuer and root01 Subject to differ at the byte level")
	}
	if int02.Issuer.String() != root01.Subject.String() {
		t.Fatalf("expected the same logical name; got %q and %q",
			int02.Issuer.String(), root01.Subject.String())
	}

	// CurrentTime within the validity of every certificate in the chain.
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	// Before amendment: the chain int02 -> root01 -> GlobalSign root fails,
	// because Go cannot match int02's Issuer to root01's Subject.
	roots := x509.NewCertPool()
	roots.AddCert(globalSignRoot)
	intermediates := x509.NewCertPool()
	intermediates.AddCert(root01)
	if _, err := int02.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		CurrentTime:   now,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}); err == nil {
		t.Fatal("expected the unamended chain to fail verification")
	}

	// Amend the intermediate issuer (root01) for its child (int02).
	amendedPEM, err := amendIssuer(read("stmtpmeccroot01.pem"), read("stmtpmeccint02.pem"))
	if err != nil {
		t.Fatalf("amendIssuer: %v", err)
	}
	amended, err := decodeCertificate(amendedPEM)
	if err != nil {
		t.Fatal(err)
	}
	amendedCert, err := x509.ParseCertificate(amended)
	if err != nil {
		t.Fatalf("parsing amended certificate: %v", err)
	}
	if !bytes.Equal(amendedCert.RawSubject, int02.RawIssuer) {
		t.Error("amended Subject does not match int02 Issuer")
	}
	assertUnsigned(t, amended)

	// After amendment: per the intermediate workflow, the amended version is
	// added to the trusted root pool, and int02 verifies against it directly.
	amendedRoots := x509.NewCertPool()
	amendedRoots.AddCert(amendedCert)
	if _, err := int02.Verify(x509.VerifyOptions{
		Roots:       amendedRoots,
		CurrentTime: now,
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}); err != nil {
		t.Fatalf("int02 did not verify against the amended root: %v", err)
	}

	// The committed golden output must match, so the WASM and browser harnesses
	// (which compare against it) stay in sync. Regenerate with:
	//   go run . testdata/stm-tpm-ecc/stmtpmeccroot01.pem \
	//       testdata/stm-tpm-ecc/stmtpmeccint02.pem > testdata/stm-tpm-ecc/amended.pem
	if want := read("amended.pem"); !bytes.Equal(amendedPEM, want) {
		t.Error("amendIssuer output does not match testdata/stm-tpm-ecc/amended.pem")
	}
}
