package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

// makeName encodes the Name "CN=Test Root", with the common name's string value
// using the given ASN.1 string tag. The same logical name encoded with two
// different tags (PrintableString vs UTF8String) reproduces the byte-for-byte
// mismatch that Go's verifier rejects.
func makeName(stringTag byte) []byte {
	cn := "Test Root"
	value := append([]byte{stringTag, byte(len(cn))}, cn...)
	commonNameOID := []byte{0x06, 0x03, 0x55, 0x04, 0x03} // 2.5.4.3
	atv := der(0x30, append(commonNameOID, value...))     // AttributeTypeAndValue
	rdn := der(0x31, atv)                                 // RelativeDistinguishedName (SET)
	return der(0x30, rdn)                                 // RDNSequence (Name)
}

// der wraps content in a short-form DER element with the given tag.
func der(tag byte, content []byte) []byte {
	if len(content) > 127 {
		panic("der: long-form length not supported")
	}
	return append([]byte{tag, byte(len(content))}, content...)
}

func certPEM(der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestAmendIssuer(t *testing.T) {
	const (
		printableString = 0x13
		utf8String      = 0x0c
	)
	notBefore := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	notAfter := time.Date(2040, 1, 1, 0, 0, 0, 0, time.UTC)

	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// The root the leaf was issued under, with a PrintableString common name.
	rootTmplA := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		RawSubject:            makeName(printableString),
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	rootDERA, err := x509.CreateCertificate(rand.Reader, rootTmplA, rootTmplA, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	rootA, err := x509.ParseCertificate(rootDERA)
	if err != nil {
		t.Fatal(err)
	}

	// The leaf, whose Issuer field copies rootA's PrintableString subject.
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "leaf.example"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		DNSNames:     []string{"leaf.example"},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, rootA, &leafKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatal(err)
	}

	// The deployed root, same key but a UTF8String common name: a byte-for-byte
	// mismatch with the leaf's Issuer field.
	rootTmplB := *rootTmplA
	rootTmplB.RawSubject = makeName(utf8String)
	rootDERB, err := x509.CreateCertificate(rand.Reader, &rootTmplB, &rootTmplB, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	rootB, err := x509.ParseCertificate(rootDERB)
	if err != nil {
		t.Fatal(err)
	}

	verify := func(root *x509.Certificate) error {
		pool := x509.NewCertPool()
		pool.AddCert(root)
		_, err := leaf.Verify(x509.VerifyOptions{
			Roots:       pool,
			CurrentTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		})
		return err
	}

	// Sanity check: the mismatched root must fail to verify the leaf, otherwise
	// the test isn't exercising the bug we work around.
	if err := verify(rootB); err == nil {
		t.Fatal("expected verification against the mismatched root to fail")
	}

	amendedPEM, err := amendIssuer(certPEM(rootDERB), certPEM(leafDER))
	if err != nil {
		t.Fatalf("amendIssuer: %v", err)
	}

	block, _ := pem.Decode(amendedPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatal("amendIssuer did not return a CERTIFICATE PEM block")
	}
	amended, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parsing amended certificate: %v", err)
	}

	// The amended Subject must be the exact bytes of the leaf's Issuer.
	if !bytes.Equal(amended.RawSubject, leaf.RawIssuer) {
		t.Errorf("amended subject = %x, want leaf issuer %x", amended.RawSubject, leaf.RawIssuer)
	}
	// Issuer is copied from the subject per RFC 9925.
	if !bytes.Equal(amended.RawIssuer, leaf.RawIssuer) {
		t.Errorf("amended issuer = %x, want %x", amended.RawIssuer, leaf.RawIssuer)
	}
	// The public key is preserved, so it can verify the leaf's signature.
	if !rootB.PublicKey.(*ecdsa.PublicKey).Equal(amended.PublicKey) {
		t.Error("amended certificate did not preserve the issuer's public key")
	}
	// It is an unsigned RFC 9925 certificate: id-alg-unsigned and empty signature.
	assertUnsigned(t, block.Bytes)

	// The leaf now verifies against the amended root.
	if err := verify(amended); err != nil {
		t.Fatalf("verification against the amended root failed: %v", err)
	}
}

// assertUnsigned checks that certDER has id-alg-unsigned in both signature
// algorithm positions and a zero-length signature BIT STRING.
func assertUnsigned(t *testing.T, certDER []byte) {
	t.Helper()
	var cert struct {
		TBS                asn1.RawValue
		SignatureAlgorithm struct{ Algorithm asn1.ObjectIdentifier }
		Signature          asn1.BitString
	}
	if _, err := asn1.Unmarshal(certDER, &cert); err != nil {
		t.Fatalf("unmarshaling amended certificate: %v", err)
	}
	if !cert.SignatureAlgorithm.Algorithm.Equal(idAlgUnsigned) {
		t.Errorf("signatureAlgorithm = %v, want %v", cert.SignatureAlgorithm.Algorithm, idAlgUnsigned)
	}
	if cert.Signature.BitLength != 0 {
		t.Errorf("signatureValue has %d bits, want 0", cert.Signature.BitLength)
	}

	var tbs struct {
		Version   int `asn1:"optional,explicit,tag:0"`
		Serial    asn1.RawValue
		Signature struct{ Algorithm asn1.ObjectIdentifier }
	}
	if _, err := asn1.Unmarshal(cert.TBS.FullBytes, &tbs); err != nil {
		t.Fatalf("unmarshaling tbsCertificate: %v", err)
	}
	if !tbs.Signature.Algorithm.Equal(idAlgUnsigned) {
		t.Errorf("tbsCertificate.signature = %v, want %v", tbs.Signature.Algorithm, idAlgUnsigned)
	}
}
